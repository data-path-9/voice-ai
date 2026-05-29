// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"context"
	"fmt"
	"time"

	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observe"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type mediaLifecycleEventKind int

const (
	mediaLifecycleEventRestart mediaLifecycleEventKind = iota + 1
)

type mediaLifecycleEvent struct {
	kind           mediaLifecycleEventKind
	mediaSessionID uint64
	reason         string
	requestedAt    time.Time
}

func (s *webrtcStreamer) watchCallerContext(callerCtx context.Context) {
	select {
	case <-callerCtx.Done():
		s.Logger.Infow("Caller context cancelled, closing streamer gracefully", "session", s.sessionID)
		if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
			s.Input(disc)
		}
		s.Close()
	case <-s.Ctx.Done():
	}
}

func (s *webrtcStreamer) runMediaLifecycleLoop() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		case event := <-s.mediaLifecycleCh:
			switch event.kind {
			case mediaLifecycleEventRestart:
				s.restartMediaSessionOrFallbackToText(event.mediaSessionID, event.reason, event.requestedAt)
			}
		}
	}
}

func (s *webrtcStreamer) queueMediaSessionRestart(mediaSessionID uint64, reason string, requestedAt time.Time) {
	event := mediaLifecycleEvent{
		kind:           mediaLifecycleEventRestart,
		mediaSessionID: mediaSessionID,
		reason:         reason,
		requestedAt:    requestedAt,
	}
	if s.mediaLifecycleCh == nil {
		go s.restartMediaSessionOrFallbackToText(mediaSessionID, reason, requestedAt)
		return
	}

	select {
	case s.mediaLifecycleCh <- event:
	case <-s.Ctx.Done():
	default:
		s.Logger.Warnw("WebRTC media lifecycle queue full, dropping restart request", "session", s.sessionID, "reason", reason)
	}
}

func (s *webrtcStreamer) runMediaSessionDeadlines(mediaSessionID uint64) {
	s.Mu.Lock()
	mediaCtx := s.mediaCtx
	s.Mu.Unlock()
	if mediaCtx == nil {
		return
	}

	ticker := time.NewTicker(webrtc_internal.HandshakeDeadlineCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.Ctx.Done():
			return
		case <-mediaCtx.Done():
			return
		case deadlineCheckedAt := <-ticker.C:
			if !s.sessionState.IsActiveMediaSession(mediaSessionID) || s.sessionState.PeerConnected() {
				return
			}

			s.Mu.Lock()
			mediaHealthState := s.mediaHealthState
			s.Mu.Unlock()

			reason, deadline, elapsed, exceeded := mediaHealthState.HandshakeDeadlineExceeded(deadlineCheckedAt)
			if !exceeded {
				continue
			}

			s.Input(&protos.ConversationEvent{
				Name: observe.ComponentWebRTC,
				Data: map[string]string{
					webrtc_internal.DataType:       webrtc_internal.EventHandshakeDeadlineExceeded,
					webrtc_internal.DataSessionID:  s.sessionID,
					webrtc_internal.DataReason:     reason,
					webrtc_internal.DataDeadline:   reason,
					webrtc_internal.DataDeadlineMs: fmt.Sprintf("%d", deadline.Milliseconds()),
					webrtc_internal.DataElapsedMs:  fmt.Sprintf("%d", elapsed.Milliseconds()),
				},
				Time: timestamppb.New(deadlineCheckedAt),
			})
			s.queueMediaSessionRestart(mediaSessionID, reason, deadlineCheckedAt)
			return
		}
	}
}

func (s *webrtcStreamer) restartMediaSessionOrFallbackToText(mediaSessionID uint64, reason string, restartedAt time.Time) {
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	restartAttempt, ok := s.sessionState.TryBeginMediaRestart(webrtc_internal.MediaRestartAttemptLimit)
	if !ok {
		s.Logger.Warnw("WebRTC media restart limit reached, falling back to text mode", "session", s.sessionID, "reason", reason)
		s.stopMediaSessionAndFallbackToText()
		return
	}

	s.Input(&protos.ConversationEvent{
		Name: observe.ComponentWebRTC,
		Data: map[string]string{
			webrtc_internal.DataType:           webrtc_internal.EventMediaSessionRestarting,
			webrtc_internal.DataSessionID:      s.sessionID,
			webrtc_internal.DataReason:         reason,
			webrtc_internal.DataRestartAttempt: fmt.Sprintf("%d", restartAttempt),
			webrtc_internal.DataRestartLimit:   fmt.Sprintf("%d", webrtc_internal.MediaRestartAttemptLimit),
		},
		Time: timestamppb.New(restartedAt),
	})

	s.ClearOutputBuffer()
	s.clearOutputAudio()
	if s.ambientMixer != nil {
		s.ambientMixer.Reset()
	}

	if err := s.startMediaSession(); err != nil {
		s.Logger.Errorw("Failed to restart WebRTC media session, falling back to text mode", "error", err, "session", s.sessionID, "reason", reason)
		s.stopMediaSessionAndFallbackToText()
	}
}
