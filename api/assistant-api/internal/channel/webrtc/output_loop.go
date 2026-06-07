// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"fmt"

	internal_output "github.com/rapidaai/api/assistant-api/internal/channel/output"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/protos"
)

// runOutputWriter routes assistant audio to the pacer and non-audio messages to gRPC.
func (s *webrtcStreamer) runOutputWriter() {
	for {
		select {
		case <-s.Ctx.Done():
			return

		case <-s.flushAudioCh:
			clearedFrames := s.clearOutputAudio()
			if clearedFrames > 0 {
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
					Level:   observability.LevelDebug,
					Message: "WebRTC output queue cleared",
					Attributes: observability.Attributes{
						"component":                              observability.ComponentWebRTC.String(),
						webrtc_internal.DataType:                 webrtc_internal.EventOutputQueueCleared,
						webrtc_internal.DataSessionID:            s.sessionID,
						webrtc_internal.DataReason:               webrtc_internal.OutputQueueClearReasonFlush,
						webrtc_internal.DataClearedFrames:        fmt.Sprintf("%d", clearedFrames),
						webrtc_internal.DataRemainingQueueFrames: fmt.Sprintf("%d", webrtc_internal.OutputAudioQueueEmptySize),
					},
				})
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordMetric{
					Metrics: []*protos.Metric{
						{Name: "webrtc_output_cleared_frames", Value: fmt.Sprintf("%d", clearedFrames), Description: "WebRTC output queue cleared frames"},
					},
				})
			}

		case msg := <-s.OutputCh:
			if m, ok := msg.(*protos.ConversationAssistantMessage); ok {
				if audio, ok := m.Message.(*protos.ConversationAssistantMessage_Audio); ok {
					s.enqueueOutputAudio(audio.Audio)
					continue
				}
			}

			if resp := s.buildGRPCResponse(msg); resp != nil {
				if !s.dispatchOutput(resp) {
					return
				}
			}
		}
	}
}

func (s *webrtcStreamer) runAudioPacer() {
	(&internal_output.Pacer{
		Logger:        s.Logger,
		FrameDuration: webrtc_internal.OutputPaceDuration,
		Provider:      s,
		Consumer:      s,
		Health:        s.outputHealth,
	}).Run(s.Ctx)
}
