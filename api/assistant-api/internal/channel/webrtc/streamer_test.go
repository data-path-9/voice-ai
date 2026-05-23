// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	l, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("webrtc-test"), commons.EnableFile(false))
	require.NoError(t, err)
	return l
}

// newTestStreamer creates a minimal webrtcStreamer for unit tests.
// No gRPC stream or Pion connection — only the fields needed for each test.
func newTestStreamer(t *testing.T) *webrtcStreamer {
	t.Helper()
	logger := newTestLogger(t)
	opusCodec, err := webrtc_internal.NewOpusCodec()
	require.NoError(t, err)

	return &webrtcStreamer{
		BaseStreamer: channel_base.NewBaseStreamer(logger,
			channel_base.WithInputChannelSize(16),
			channel_base.WithOutputChannelSize(16),
		),
		config:      webrtc_internal.DefaultConfig(),
		sessionID:   "test-session",
		opusCodec:   opusCodec,
		currentMode: protos.StreamMode_STREAM_MODE_TEXT,
	}
}

type fakeAmbientMixer struct {
	cfg        internal_ambient.Config
	ambientOut []byte
}

func (f *fakeAmbientMixer) Configure(cfg internal_ambient.Config) error {
	f.cfg = cfg
	return nil
}

func (f *fakeAmbientMixer) Mix(primary []byte) ([]byte, error) {
	if primary == nil {
		return append([]byte(nil), f.ambientOut...), nil
	}
	return append([]byte(nil), primary...), nil
}

func (f *fakeAmbientMixer) Reset() {}

func (f *fakeAmbientMixer) CurrentConfig() internal_ambient.Config { return f.cfg }

// --- Test: buildGRPCResponse wraps all proto types correctly ---

func TestBuildGRPCResponse_Disconnection(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationDisconnection{}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetDisconnection())
}

func TestBuildGRPCResponse_AssistantText(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Text{Text: "hello world"},
	}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetAssistant())
}

func TestBuildGRPCResponse_ToolCall(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationToolCall{Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetToolCall())
}

func TestBuildGRPCResponse_Event(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationEvent{Name: "test", Data: map[string]string{"key": "val"}}
	resp := s.buildGRPCResponse(msg)
	require.NotNil(t, resp)
	assert.NotNil(t, resp.GetEvent())
}

// --- Test: handleConfigurationMessage deduplication ---

func TestHandleConfigurationMessage_SameModeNoop(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT

	// Calling with same mode should be a no-op (no peer connection created)
	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)

	s.Mu.Lock()
	pc := s.pc
	s.Mu.Unlock()
	assert.Nil(t, pc, "peer connection should not be created for same mode")
}

func TestHandleConfigurationMessage_TextToAudioFails(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT

	// Switching to audio without a gRPC stream will fail in createPeerConnection
	// but should not panic — it should reset to text mode
	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)

	s.Mu.Lock()
	mode := s.currentMode
	s.Mu.Unlock()
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, mode, "should fall back to text on audio setup failure")
}

// --- Test: Close idempotency ---

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	// First close should succeed
	err := s.Close()
	assert.NoError(t, err)

	// Second close should also succeed (no-op)
	err = s.Close()
	assert.NoError(t, err)

	// Verify closed flag
	assert.True(t, s.closed.Load())
}

func TestClose_ConcurrentSafe(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	var wg sync.WaitGroup
	closeCount := 20

	for i := 0; i < closeCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Close()
		}()
	}

	wg.Wait()
	assert.True(t, s.closed.Load())
}

// --- Test: resetAudioSession clears state ---

func TestResetAudioSession_ClearsState(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.peerConnected.Store(true)
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO

	s.resetAudioSession()

	assert.False(t, s.peerConnected.Load(), "peerConnected should be false after reset")
	s.Mu.Lock()
	assert.Nil(t, s.pc, "peer connection should be nil after reset")
	assert.Nil(t, s.localTrack, "local track should be nil after reset")
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	s.Mu.Unlock()
}

func TestResetAudioSession_FlushesPendingOutput(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	// Simulate buffered-but-not-yet-sent output.
	s.WithOutputBuffer(func(buf *bytes.Buffer) {
		buf.Write([]byte{0x01, 0x02, 0x03, 0x04})
	})
	s.Output(&protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0xAA, 0xBB}},
	})

	s.resetAudioSession()

	s.WithOutputBuffer(func(buf *bytes.Buffer) {
		assert.Equal(t, 0, buf.Len(), "output accumulation buffer should be cleared")
	})

	select {
	case <-s.OutputCh:
		t.Fatal("output channel should be drained after reset")
	default:
	}
}

// --- Test: Send routes correctly ---

func TestSend_TextMessage(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Text{Text: "hello"},
	}
	err := s.Send(msg)
	assert.NoError(t, err)
}

func TestSend_Interruption(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationInterruption{
		Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
	}
	err := s.Send(msg)
	assert.NoError(t, err)
}

func TestSend_EndConversation(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationToolCall{
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
	}
	err := s.Send(msg)
	assert.NoError(t, err)
}

func TestSend_TransferConversation_PushesFailedResult(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationToolCall{
		Id:     "tc-transfer",
		ToolId: "tool-transfer",
		Name:   "transfer_call",
		Action: protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION,
		Args:   map[string]string{"transfer_to": "+15551234567"},
	}

	err := s.Send(msg)
	require.NoError(t, err)

	select {
	case incoming := <-s.CriticalCh:
		result, ok := incoming.(*protos.ConversationToolCallResult)
		require.True(t, ok, "expected ConversationToolCallResult, got %T", incoming)
		assert.Equal(t, "tc-transfer", result.GetId())
		assert.Equal(t, "tool-transfer", result.GetToolId())
		assert.Equal(t, "transfer_call", result.GetName())
		assert.Equal(t, protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION, result.GetAction())
		assert.Equal(t, "failed", result.GetResult()["status"])
		assert.Contains(t, result.GetResult()["reason"], "transfer not supported for WebRTC")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ConversationToolCallResult")
	}
}

func TestApplyAmbientConfig_ReadsTypedConfig(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	fake := &fakeAmbientMixer{}
	s.ambientMixer = fake

	s.applyAmbientConfig(internal_ambient.NewConfig("cafe", 37), "test")

	assert.Equal(t, "cafe", fake.cfg.Profile)
	assert.Equal(t, 37, fake.cfg.Volume)
	assert.True(t, fake.cfg.Enabled)
}

func TestApplyAmbientConfig_InvalidAmbientFallsBackToNone(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	fake := &fakeAmbientMixer{}
	s.ambientMixer = fake

	s.applyAmbientConfig(internal_ambient.NewConfig("foobar", 24), "test")

	assert.Equal(t, "none", fake.cfg.Profile)
	assert.Equal(t, 24, fake.cfg.Volume)
	assert.False(t, fake.cfg.Enabled)
}

func TestApplyAmbientToFrame_AmbientOnlyOnSilenceTicks(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	fake := &fakeAmbientMixer{
		ambientOut: make([]byte, webrtc_internal.OpusFrameBytes),
	}
	for i := range fake.ambientOut {
		fake.ambientOut[i] = 0x11
	}
	s.ambientMixer = fake

	out := s.applyAmbientToFrame(nil)
	require.NotNil(t, out)
	assert.Len(t, out, webrtc_internal.OpusFrameBytes)
	assert.NotEqual(t, make([]byte, len(out)), out)
}

func TestApplyAmbientToFrame_NoneLeavesPrimaryUntouched(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.ambientMixer = nil

	in := make([]byte, webrtc_internal.OpusFrameBytes)
	for i := range in {
		in[i] = byte(i % 251)
	}
	out := s.applyAmbientToFrame(in)
	assert.Equal(t, in, out)
}

func TestEnqueuePendingAudio_BoundedDropOldest_EmitsOverflowEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	limit := webrtc_internal.PendingAudioMaxFrames
	for i := 0; i < limit+1; i++ {
		s.enqueuePendingAudio([]byte{byte(i % 251)})
	}

	s.pendingAudioMu.Lock()
	require.Len(t, s.pendingAudio, limit)
	// Oldest frame should be dropped on overflow; new head is former index 1.
	require.Len(t, s.pendingAudio[0], 1)
	assert.Equal(t, byte(1), s.pendingAudio[0][0])
	s.pendingAudioMu.Unlock()

	select {
	case msg := <-s.LowCh:
		eventMsg, ok := msg.(*protos.ConversationEvent)
		require.True(t, ok, "expected ConversationEvent, got %T", msg)
		assert.Equal(t, "webrtc", eventMsg.GetName())
		assert.Equal(t, "output_queue_overflow", eventMsg.GetData()["type"])
		assert.Equal(t, "drop_oldest", eventMsg.GetData()["policy"])
		assert.Equal(t, "1", eventMsg.GetData()["dropped_frames"])
		assert.Equal(t, fmt.Sprintf("%d", limit), eventMsg.GetData()["limit_frames"])
		assert.Equal(t, fmt.Sprintf("%d", limit), eventMsg.GetData()["queue_depth_frames"])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for overflow event")
	}
}
