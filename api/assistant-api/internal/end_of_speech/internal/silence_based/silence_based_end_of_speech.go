// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_silence_based

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	silenceBasedEndOfSpeechName = "silenceBasedEndOfSpeech"
	optSilenceTimeout           = "microphone.eos.timeout"
	defaultSilenceTimeout       = 1000 * time.Millisecond
)

type speechSegment struct {
	ContextID string
	Text      string
	Chunks    []internal_type.SpeechToTextPacket
	Timestamp time.Time
}

type workerCommand struct {
	ctx             context.Context
	timeout         time.Duration
	segment         speechSegment
	fireImmediately bool
}

type endOfSpeechState struct {
	segment       speechSegment
	callbackFired bool
	generation    uint64
}

type silenceBasedEndOfSpeech struct {
	onPacket       func(context.Context, ...internal_type.Packet) error
	opts           utils.Option
	silenceTimeout time.Duration

	commandCh chan workerCommand
	stopCh    chan struct{}

	mu    sync.RWMutex
	state *endOfSpeechState
}

func NewSilenceBasedEndOfSpeech(
	_ commons.Logger,
	onPacket func(context.Context, ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.EndOfSpeechExecutor, error) {
	start := time.Now()
	silenceTimeout := defaultSilenceTimeout
	if value, err := opts.GetFloat64(optSilenceTimeout); err == nil {
		silenceTimeout = time.Duration(value) * time.Millisecond
	}

	endOfSpeech := &silenceBasedEndOfSpeech{
		onPacket:       onPacket,
		opts:           opts,
		silenceTimeout: silenceTimeout,
		commandCh:      make(chan workerCommand, 32),
		stopCh:         make(chan struct{}),
		state:          &endOfSpeechState{segment: speechSegment{}},
	}

	go endOfSpeech.worker()

	if onPacket != nil {
		_ = onPacket(context.Background(), internal_type.ConversationEventPacket{
			Name: "eos",
			Data: map[string]string{
				"type":     "initialized",
				"provider": endOfSpeech.Name(),
				"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
			},
			Time: time.Now(),
		})
	}

	return endOfSpeech, nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) Name() string {
	return silenceBasedEndOfSpeechName
}

func (endOfSpeech *silenceBasedEndOfSpeech) Options() utils.Option {
	return endOfSpeech.opts
}

func (endOfSpeech *silenceBasedEndOfSpeech) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) Execute(ctx context.Context, packet internal_type.Packet) error {
	switch packet := packet.(type) {
	case internal_type.UserTextReceivedPacket:
		return endOfSpeech.handleUserTextPacket(ctx, packet)
	case internal_type.EndOfSpeechInterruptionPacket:
		return endOfSpeech.handleInterruptionPacket(ctx)
	case internal_type.VadSpeechActivityPacket:
		return endOfSpeech.handleSpeechActivityPacket(ctx)
	case internal_type.SpeechToTextPacket:
		return endOfSpeech.handleSpeechToTextPacket(ctx, packet)
	}

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) handleUserTextPacket(
	ctx context.Context,
	packet internal_type.UserTextReceivedPacket,
) error {
	if packet.Text == "" {
		return nil
	}

	endOfSpeech.mu.Lock()
	segment := speechSegment{
		ContextID: packet.ContextId(),
		Text:      packet.Text,
		Timestamp: time.Now(),
	}
	endOfSpeech.state.segment = segment
	endOfSpeech.mu.Unlock()

	endOfSpeech.emitInterimSpeech(ctx, segment)
	endOfSpeech.enqueueCommand(workerCommand{
		ctx:             ctx,
		segment:         segment,
		fireImmediately: true,
	})

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) handleInterruptionPacket(ctx context.Context) error {
	return endOfSpeech.extendCurrentSegment(ctx, endOfSpeech.silenceTimeout)
}

func (endOfSpeech *silenceBasedEndOfSpeech) handleSpeechActivityPacket(ctx context.Context) error {
	return endOfSpeech.extendCurrentSegment(ctx, endOfSpeech.silenceTimeout)
}

func (endOfSpeech *silenceBasedEndOfSpeech) handleSpeechToTextPacket(
	ctx context.Context,
	packet internal_type.SpeechToTextPacket,
) error {
	endOfSpeech.mu.Lock()
	if packet.Interim {
		segment := endOfSpeech.state.segment
		endOfSpeech.mu.Unlock()
		if segment.Text == "" {
			return nil
		}

		endOfSpeech.enqueueCommand(workerCommand{
			ctx:     ctx,
			segment: segment,
			timeout: endOfSpeech.silenceTimeout,
		})

		return nil
	}

	segment := speechSegment{
		ContextID: packet.ContextId(),
		Timestamp: time.Now(),
		Text:      endOfSpeech.state.segment.Text,
		Chunks:    append([]internal_type.SpeechToTextPacket(nil), endOfSpeech.state.segment.Chunks...),
	}
	if segment.Text != "" {
		segment.Text += " " + packet.Script
	} else {
		segment.Text = packet.Script
	}
	segment.Chunks = append(segment.Chunks, packet)
	endOfSpeech.state.segment = segment
	endOfSpeech.mu.Unlock()

	endOfSpeech.emitInterimSpeech(ctx, segment)
	endOfSpeech.enqueueCommand(workerCommand{
		ctx:     ctx,
		segment: segment,
		timeout: endOfSpeech.silenceTimeout,
	})

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) extendCurrentSegment(
	ctx context.Context,
	timeout time.Duration,
) error {
	endOfSpeech.mu.RLock()
	segment := endOfSpeech.state.segment
	endOfSpeech.mu.RUnlock()

	if segment.Text == "" {
		return nil
	}

	endOfSpeech.enqueueCommand(workerCommand{
		ctx:     ctx,
		segment: segment,
		timeout: timeout,
	})

	return nil
}

func (endOfSpeech *silenceBasedEndOfSpeech) emitInterimSpeech(ctx context.Context, segment speechSegment) {
	emittedAt := time.Now()
	_ = endOfSpeech.onPacket(ctx,
		internal_type.InterimEndOfSpeechPacket{
			Speech:    segment.Text,
			ContextID: segment.ContextID,
		},
		internal_type.ConversationEventPacket{
			ContextID: segment.ContextID,
			Name:      "eos",
			Data: map[string]string{
				"type":       "interim",
				"provider":   endOfSpeech.Name(),
				"context_id": segment.ContextID,
				"speech":     segment.Text,
			},
			Time: emittedAt,
		},
	)
}

func (endOfSpeech *silenceBasedEndOfSpeech) enqueueCommand(command workerCommand) {
	select {
	case <-endOfSpeech.stopCh:
		return
	default:
	}

	select {
	case endOfSpeech.commandCh <- command:
	case <-endOfSpeech.stopCh:
	}
}

func (endOfSpeech *silenceBasedEndOfSpeech) worker() {
	var (
		timer          *time.Timer
		timerCh        <-chan time.Time
		generation     uint64
		currentCommand workerCommand
	)

	stopTimer := func() {
		if timer != nil {
			timer.Stop()
			timer = nil
			timerCh = nil
		}
	}
	resetState := func() {
		endOfSpeech.state.callbackFired = false
		endOfSpeech.state.generation++
		endOfSpeech.state.segment = speechSegment{}
	}

	for {
		select {
		case <-endOfSpeech.stopCh:
			stopTimer()
			return

		case command := <-endOfSpeech.commandCh:
			endOfSpeech.mu.Lock()

			if endOfSpeech.state.callbackFired {
				endOfSpeech.mu.Unlock()
				continue
			}

			if command.fireImmediately {
				endOfSpeech.state.callbackFired = true
				currentCommand = command
				stopTimer()
				endOfSpeech.mu.Unlock()
				endOfSpeech.emitEndOfSpeech(currentCommand.ctx, currentCommand.segment)
				endOfSpeech.mu.Lock()
				resetState()
				endOfSpeech.mu.Unlock()
				continue
			}

			generation = endOfSpeech.state.generation + 1
			endOfSpeech.state.generation = generation
			currentCommand = command
			stopTimer()
			timer = time.NewTimer(command.timeout)
			timerCh = timer.C
			endOfSpeech.mu.Unlock()

		case <-timerCh:
			endOfSpeech.mu.Lock()
			if endOfSpeech.state.callbackFired || generation != endOfSpeech.state.generation {
				endOfSpeech.mu.Unlock()
				continue
			}

			endOfSpeech.state.callbackFired = true
			command := currentCommand
			stopTimer()
			endOfSpeech.mu.Unlock()
			endOfSpeech.emitEndOfSpeech(command.ctx, command.segment)
			endOfSpeech.mu.Lock()
			resetState()
			endOfSpeech.mu.Unlock()
		}
	}
}

func (endOfSpeech *silenceBasedEndOfSpeech) emitEndOfSpeech(ctx context.Context, segment speechSegment) {
	if segment.Text == "" {
		return
	}
	if ctx != nil && ctx.Err() != nil {
		ctx = context.Background()
	}

	wordCount := len(strings.Fields(segment.Text))
	triggerAt := time.Now()
	_ = endOfSpeech.onPacket(ctx,
		internal_type.EndOfSpeechPacket{
			Speech:    segment.Text,
			ContextID: segment.ContextID,
			Speechs:   append([]internal_type.SpeechToTextPacket(nil), segment.Chunks...),
		},
		internal_type.UserMessageMetricPacket{
			ContextID: segment.ContextID,
			Metrics: []*protos.Metric{{
				Name:  "eos_latency_ms",
				Value: fmt.Sprintf("%d", triggerAt.Sub(segment.Timestamp).Milliseconds()),
			}},
		},
		internal_type.ConversationEventPacket{
			ContextID: segment.ContextID,
			Name:      "eos",
			Data: map[string]string{
				"type":               "detected",
				"provider":           endOfSpeech.Name(),
				"context_id":         segment.ContextID,
				"speech":             segment.Text,
				"confidence":         "0.0000",
				"word_count":         fmt.Sprintf("%d", wordCount),
				"char_count":         fmt.Sprintf("%d", len(segment.Text)),
				"text_to_trigger_ms": fmt.Sprintf("%d", triggerAt.Sub(segment.Timestamp).Milliseconds()),
			},
			Time: triggerAt,
		},
	)
}

func (endOfSpeech *silenceBasedEndOfSpeech) Close(_ context.Context) error {
	close(endOfSpeech.stopCh)
	return nil
}
