// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telephony_media

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	"github.com/rapidaai/api/assistant-api/internal/observe"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MediaEngine defines shared telephony media semantics independent of transport.
type MediaEngine interface {
	SetInputAudioCallback(callback func(audio []byte))
	ProcessInputAudio(audio []byte) error
	ProcessOutputAudio(audio []byte) error
	Complete()
	ClearOutputBuffer()
	RunOutputSender(ctx context.Context)
	ConfigureAmbient(cfg internal_ambient.Config) error
}

// MediaSession owns telephony media lifecycle for a channel transport.
// Transport implementations only need to feed provider audio in and send clear commands.
type MediaSession struct {
	logger commons.Logger
	engine MediaEngine

	sendClear func() error

	inputSinkMu sync.RWMutex
	inputSink   func([]byte)
	eventSink   func(*protos.ConversationEvent)

	started atomic.Bool
	closed  atomic.Bool

	startMu sync.Mutex
	cancel  context.CancelFunc
	ctx     context.Context
}

type outputHealthSnapshotter interface {
	OutputHealthSnapshot() internal_output.HealthSnapshot
}

func NewMediaSession(parent context.Context, logger commons.Logger, engine MediaEngine, sendClear func() error) *MediaSession {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	s := &MediaSession{
		logger:    logger,
		engine:    engine,
		sendClear: sendClear,
		ctx:       ctx,
		cancel:    cancel,
	}
	if engine != nil {
		engine.SetInputAudioCallback(s.onInputAudio)
	}
	return s
}

func (s *MediaSession) SetInputSink(fn func(audio []byte)) {
	s.inputSinkMu.Lock()
	s.inputSink = fn
	s.inputSinkMu.Unlock()
}

func (s *MediaSession) SetEventSink(fn func(event *protos.ConversationEvent)) {
	s.inputSinkMu.Lock()
	s.eventSink = fn
	s.inputSinkMu.Unlock()
}

func (s *MediaSession) Start() {
	if s == nil || s.engine == nil || s.closed.Load() {
		return
	}
	s.startMu.Lock()
	defer s.startMu.Unlock()
	if !s.started.CompareAndSwap(false, true) {
		return
	}
	go s.engine.RunOutputSender(s.ctx)
	if hs, ok := s.engine.(outputHealthSnapshotter); ok {
		go s.runOutputHealthReporter(hs)
	}
}

func (s *MediaSession) HandleInitialization(init *protos.ConversationInitialization) {
	if s == nil || s.engine == nil || init == nil {
		return
	}
	cfg, ok := internal_ambient.ParseFromInitialization(init)
	if !ok {
		return
	}
	if err := s.engine.ConfigureAmbient(cfg); err != nil && s.logger != nil {
		s.logger.Warnw("Failed to configure ambient audio", "error", err.Error(), "profile", cfg.Profile)
	}
}

func (s *MediaSession) HandleAssistantAudio(audio []byte, completed bool) error {
	if s == nil || s.engine == nil {
		return nil
	}
	if err := s.engine.ProcessOutputAudio(audio); err != nil {
		return err
	}
	if completed {
		s.engine.Complete()
	}
	return nil
}

func (s *MediaSession) HandleProviderAudio(audio []byte) error {
	if s == nil || s.engine == nil {
		return nil
	}
	return s.engine.ProcessInputAudio(audio)
}

func (s *MediaSession) HandleInterrupt() {
	if s == nil || s.engine == nil {
		return
	}
	s.engine.ClearOutputBuffer()
	if s.sendClear != nil {
		if err := s.sendClear(); err != nil && s.logger != nil {
			s.logger.Warn("Failed to send telephony clear command", "error", err.Error())
		}
	}
}

func (s *MediaSession) Shutdown() {
	if s == nil {
		return
	}
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *MediaSession) onInputAudio(audio []byte) {
	s.inputSinkMu.RLock()
	fn := s.inputSink
	s.inputSinkMu.RUnlock()
	if fn == nil {
		return
	}
	fn(audio)
}

func (s *MediaSession) emitEvent(data map[string]string) {
	s.inputSinkMu.RLock()
	fn := s.eventSink
	s.inputSinkMu.RUnlock()
	if fn == nil {
		return
	}
	fn(&protos.ConversationEvent{
		Name: observe.ComponentTelephony,
		Data: data,
		Time: timestamppb.Now(),
	})
}

func (s *MediaSession) runOutputHealthReporter(hs outputHealthSnapshotter) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var prev internal_output.HealthSnapshot
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
		}

		snap := hs.OutputHealthSnapshot()
		if snap.Ticks == prev.Ticks {
			continue
		}

		s.emitEvent(map[string]string{
			"type":         "output_pacer_health",
			"ticks":        fmt.Sprintf("%d", snap.Ticks),
			"late_ticks":   fmt.Sprintf("%d", snap.LateTicks),
			"active_ticks": fmt.Sprintf("%d", snap.ActiveTicks),
			"idle_ticks":   fmt.Sprintf("%d", snap.IdleTicks),
			"send_errors":  fmt.Sprintf("%d", snap.SendErrors),
			"idle_ratio":   fmt.Sprintf("%.4f", snap.IdleRatio),
		})

		if snap.SendErrors > prev.SendErrors {
			s.emitEvent(map[string]string{
				"type":              "output_send_error",
				"send_errors_delta": fmt.Sprintf("%d", snap.SendErrors-prev.SendErrors),
				"total_send_errors": fmt.Sprintf("%d", snap.SendErrors),
				"ticks":             fmt.Sprintf("%d", snap.Ticks),
				"late_ticks":        fmt.Sprintf("%d", snap.LateTicks),
				"active_ticks":      fmt.Sprintf("%d", snap.ActiveTicks),
				"idle_ticks":        fmt.Sprintf("%d", snap.IdleTicks),
				"idle_ratio":        fmt.Sprintf("%.4f", snap.IdleRatio),
			})
		}
		prev = snap
	}
}
