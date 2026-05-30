// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"bytes"
	"time"

	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newWebRTCAudioBufferState() webrtc_internal.WebRTCAudioBufferState {
	return webrtc_internal.WebRTCAudioBufferState{
		InputAudioBuffer:  bytes.NewBuffer(make([]byte, 0, webrtc_internal.InputBufferThreshold*2)),
		OutputAudioBuffer: bytes.NewBuffer(make([]byte, 0, webrtc_internal.WebRTCOutputPCM16kFrameBytes*2)),
	}
}

func (s *webrtcStreamer) bufferAndSendInput(audio []byte, inputAudioReceivedAt time.Time) {
	if inputAudioReceivedAt.IsZero() {
		inputAudioReceivedAt = time.Now()
	}
	s.Input(&protos.ConversationBridgeUserAudio{
		Audio: audio,
		Time:  timestamppb.New(inputAudioReceivedAt),
	})

	s.audioBufferState.InputAudioBufferMu.Lock()
	s.audioBufferState.InputAudioBuffer.Write(audio)
	if s.audioBufferState.InputAudioBuffer.Len() < webrtc_internal.InputBufferThreshold {
		s.audioBufferState.InputAudioBufferMu.Unlock()
		return
	}

	audioData := s.audioBufferState.InputAudioBuffer.Bytes()
	s.audioBufferState.InputAudioBuffer = bytes.NewBuffer(make([]byte, 0, webrtc_internal.InputBufferThreshold*2))
	s.audioBufferState.InputAudioBufferMu.Unlock()

	s.Input(&protos.ConversationUserMessage{
		Message: &protos.ConversationUserMessage_Audio{Audio: audioData},
		Time:    timestamppb.New(inputAudioReceivedAt),
	})
}

func (s *webrtcStreamer) bufferAndSendOutput(audio []byte) {
	s.audioBufferState.OutputAudioBufferMu.Lock()
	s.audioBufferState.OutputAudioBuffer.Write(audio)
	if s.audioBufferState.OutputAudioBuffer.Len() < webrtc_internal.WebRTCOutputPCM16kFrameBytes {
		s.audioBufferState.OutputAudioBufferMu.Unlock()
		return
	}

	var frames [][]byte
	for s.audioBufferState.OutputAudioBuffer.Len() >= webrtc_internal.WebRTCOutputPCM16kFrameBytes {
		frame := make([]byte, webrtc_internal.WebRTCOutputPCM16kFrameBytes)
		s.audioBufferState.OutputAudioBuffer.Read(frame)
		frames = append(frames, frame)
	}
	s.audioBufferState.OutputAudioBufferMu.Unlock()

	frameTimestamp := timestamppb.Now()
	for _, frame := range frames {
		s.Output(&protos.ConversationAssistantMessage{
			Message: &protos.ConversationAssistantMessage_Audio{Audio: frame},
			Time:    frameTimestamp,
		})
	}
}

func (s *webrtcStreamer) clearBufferedOutputAudio() {
	s.audioBufferState.OutputAudioBufferMu.Lock()
	s.audioBufferState.OutputAudioBuffer.Reset()
	s.audioBufferState.OutputAudioBufferMu.Unlock()

	select {
	case s.flushAudioCh <- struct{}{}:
	default:
	}

	for {
		select {
		case <-s.OutputCh:
		default:
			return
		}
	}
}

func (s *webrtcStreamer) withOutputAudioBuffer(fn func(buf *bytes.Buffer)) {
	s.audioBufferState.OutputAudioBufferMu.Lock()
	defer s.audioBufferState.OutputAudioBufferMu.Unlock()
	fn(s.audioBufferState.OutputAudioBuffer)
}
