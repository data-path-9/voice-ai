// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	internal_audio_resampler "github.com/rapidaai/api/assistant-api/internal/audio/resampler"
	channel_base "github.com/rapidaai/api/assistant-api/internal/channel/base"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observe"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func newTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	l, err := commons.NewApplicationLogger(commons.Level("error"), commons.Name("webrtc-test"), commons.EnableFile(false))
	require.NoError(t, err)
	return l
}

// newTestStreamer creates a WebRTC streamer with test-owned dependencies.
func newTestStreamer(t *testing.T) *webrtcStreamer {
	t.Helper()
	logger := newTestLogger(t)
	opusCodec, err := webrtc_internal.NewOpusCodec()
	require.NoError(t, err)
	resampler, err := internal_audio_resampler.GetResampler(logger)
	require.NoError(t, err)

	return &webrtcStreamer{
		BaseStreamer: channel_base.NewBaseStreamer(logger,
			channel_base.WithInputChannelSize(16),
			channel_base.WithOutputChannelSize(16),
			channel_base.WithOutputAudioConfig(internal_audio.RAPIDA_INTERNAL_AUDIO_CONFIG),
		),
		peerConfig:  webrtc_internal.DefaultConfig(),
		sessionID:   "test-session",
		resampler:   resampler,
		opusCodec:   opusCodec,
		currentMode: protos.StreamMode_STREAM_MODE_TEXT,
	}
}

type fakeAmbientMixer struct {
	cfg        internal_ambient.Config
	ambientOut []byte
}

type failingGRPCStream struct {
	sendErr error
}

func (f *failingGRPCStream) Recv() (*protos.WebTalkRequest, error) {
	return nil, io.EOF
}

func (f *failingGRPCStream) Send(*protos.WebTalkResponse) error {
	return f.sendErr
}

func (f *failingGRPCStream) SetHeader(metadata.MD) error {
	return nil
}

func (f *failingGRPCStream) SendHeader(metadata.MD) error {
	return nil
}

func (f *failingGRPCStream) SetTrailer(metadata.MD) {}

func (f *failingGRPCStream) Context() context.Context {
	return context.Background()
}

func (f *failingGRPCStream) SendMsg(any) error {
	return nil
}

func (f *failingGRPCStream) RecvMsg(any) error {
	return io.EOF
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

func TestNewWebRTCStreamer_UsesConfiguredICEServers(t *testing.T) {
	t.Setenv("TURN_USERNAME", "turn-user")
	t.Setenv("TURN_CREDENTIAL", "turn-secret")
	streamer, err := NewWebRTCStreamer(context.Background(), newTestLogger(t), &failingGRPCStream{sendErr: io.EOF}, &assistant_config.WebRTCConfig{
		ICEServers: []assistant_config.WebRTCICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
			{
				URLs: []string{
					"turn:turn.rapida.ai:3478?transport=udp",
					"turn:turn.rapida.ai:3478?transport=tcp",
					"turns:turn.rapida.ai:443?transport=tcp",
				},
				Username:   "${TURN_USERNAME}",
				Credential: "${TURN_CREDENTIAL}",
			},
		},
		ICETransportPolicy: webrtc_internal.ICETransportPolicyRelay,
	})
	require.NoError(t, err)
	s := streamer.(*webrtcStreamer)
	t.Cleanup(func() { _ = s.Close() })

	require.Len(t, s.peerConfig.ICEServers, 2)
	assert.Equal(t, []string{"stun:stun.l.google.com:19302"}, s.peerConfig.ICEServers[0].URLs)
	assert.Equal(t, []string{
		"turn:turn.rapida.ai:3478?transport=udp",
		"turn:turn.rapida.ai:3478?transport=tcp",
		"turns:turn.rapida.ai:443?transport=tcp",
	}, s.peerConfig.ICEServers[1].URLs)
	assert.Equal(t, "turn-user", s.peerConfig.ICEServers[1].Username)
	assert.Equal(t, "turn-secret", s.peerConfig.ICEServers[1].Credential)
	assert.Equal(t, webrtc_internal.ICETransportPolicyRelay, s.peerConfig.ICETransportPolicy)
}

func TestNewWebRTCStreamer_DefaultsToGoogleSTUN(t *testing.T) {
	t.Parallel()
	streamer, err := NewWebRTCStreamer(context.Background(), newTestLogger(t), &failingGRPCStream{sendErr: io.EOF}, &assistant_config.WebRTCConfig{})
	require.NoError(t, err)
	s := streamer.(*webrtcStreamer)
	t.Cleanup(func() { _ = s.Close() })

	require.Len(t, s.peerConfig.ICEServers, 2)
	assert.Equal(t, []string{"stun:stun.l.google.com:19302"}, s.peerConfig.ICEServers[0].URLs)
	assert.Equal(t, []string{"stun:stun1.l.google.com:19302"}, s.peerConfig.ICEServers[1].URLs)
	assert.Equal(t, webrtc_internal.ICETransportPolicyAll, s.peerConfig.ICETransportPolicy)
}

func TestNewWebRTCStreamer_InvalidICETransportPolicyFallsBackToAll(t *testing.T) {
	t.Parallel()
	streamer, err := NewWebRTCStreamer(context.Background(), newTestLogger(t), &failingGRPCStream{sendErr: io.EOF}, &assistant_config.WebRTCConfig{
		ICETransportPolicy: "invalid",
	})
	require.NoError(t, err)
	s := streamer.(*webrtcStreamer)
	t.Cleanup(func() { _ = s.Close() })

	assert.Equal(t, webrtc_internal.ICETransportPolicyAll, s.peerConfig.ICETransportPolicy)
}

func TestDispatchOutput_SendFailureClosesStreamer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.grpcStream = &failingGRPCStream{sendErr: errors.New("client closed")}

	ok := s.dispatchOutput(&protos.WebTalkResponse{})

	assert.False(t, ok)
	assert.True(t, s.sessionState.CloseStarted())
	select {
	case msg := <-s.CriticalCh:
		_, ok := msg.(*protos.ConversationDisconnection)
		assert.True(t, ok, "expected ConversationDisconnection, got %T", msg)
	default:
		t.Fatal("expected disconnection on gRPC send failure")
	}
}

func TestServerSignaling_UsesActiveSignalingSessionID(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"

	s.signalConfig()

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.Equal(t, "media-signaling-session", signaling.GetSessionId())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server signaling")
	}
}

func TestServerSignaling_FallsBackToStreamerSessionID(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	s.signalClear()

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.Equal(t, s.sessionID, signaling.GetSessionId())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server signaling")
	}
}

func TestServerTrickleICECandidate_UsesActiveSignalingSessionID(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	s.signalOfferSent = true
	sdpMid := "audio"
	sdpMLineIndex := uint16(0)
	usernameFragment := "ufrag"

	s.queueLocalICECandidate(pionwebrtc.ICECandidateInit{
		Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
		SDPMid:           &sdpMid,
		SDPMLineIndex:    &sdpMLineIndex,
		UsernameFragment: &usernameFragment,
	}, mediaSessionID)

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		require.NotNil(t, signaling.GetIceCandidate())
		assert.Equal(t, "media-signaling-session", signaling.GetSessionId())
		assert.NotEmpty(t, signaling.GetIceCandidate().GetCandidate())
		assert.Equal(t, sdpMid, signaling.GetIceCandidate().GetSdpMid())
		assert.Equal(t, int32(sdpMLineIndex), signaling.GetIceCandidate().GetSdpMLineIndex())
		assert.Equal(t, usernameFragment, signaling.GetIceCandidate().GetUsernameFragment())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server ICE candidate")
	}
}

func TestServerTrickleICECandidate_CachesUntilOfferSignaled(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	sdpMid := "audio"
	sdpMLineIndex := uint16(0)

	s.queueLocalICECandidate(pionwebrtc.ICECandidateInit{
		Candidate:     "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
		SDPMid:        &sdpMid,
		SDPMLineIndex: &sdpMLineIndex,
	}, mediaSessionID)

	s.Mu.Lock()
	pendingCandidateCount := len(s.signalPendingLocalICECandidates)
	signalOfferSent := s.signalOfferSent
	s.Mu.Unlock()

	assert.Equal(t, 1, pendingCandidateCount)
	assert.False(t, signalOfferSent)
	select {
	case msg := <-s.OutputCh:
		t.Fatalf("ICE candidate should not be sent before offer is signaled: %T", msg)
	default:
	}
}

func TestInitiateWebRTCHandshake_SendsOfferBeforeTrickleCandidates(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "media-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()

	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	require.NoError(t, s.sendOffer(mediaSessionID))

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		assert.NotNil(t, signaling.GetConfig())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC config")
	}

	select {
	case msg := <-s.OutputCh:
		signaling, ok := msg.(*protos.ServerSignaling)
		require.True(t, ok, "expected ServerSignaling, got %T", msg)
		require.NotNil(t, signaling.GetSdp())
		assert.Equal(t, protos.WebRTCSDP_OFFER, signaling.GetSdp().GetType())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebRTC offer")
	}
}

func TestHandleConfigurationMessage_SameModeNoop(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT

	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_TEXT)

	s.Mu.Lock()
	peerConnection := s.peerConnection
	s.Mu.Unlock()
	assert.Nil(t, peerConnection, "peer connection should not be created for same mode")
}

func TestHandleConfigurationMessage_TextToAudioFails(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.currentMode = protos.StreamMode_STREAM_MODE_TEXT

	s.handleConfigurationMessage(protos.StreamMode_STREAM_MODE_AUDIO)

	s.Mu.Lock()
	mode := s.currentMode
	s.Mu.Unlock()
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, mode, "should fall back to text on audio setup failure")
}

func TestClose_Idempotent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	err := s.Close()
	assert.NoError(t, err)

	err = s.Close()
	assert.NoError(t, err)

	assert.True(t, s.sessionState.CloseStarted())
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
	assert.True(t, s.sessionState.CloseStarted())
}

func TestResetAudioSession_ClearsState(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	previousMediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.signalingSessionID = "active-signaling"
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
	now := time.Now()
	s.mediaHealthState = webrtc_internal.MediaHealthState{
		ICEStartedAt:                           now,
		PeerConnectedAt:                        now,
		FirstUserAudioReceivedAt:               now,
		LastUserAudioReceivedAt:                now,
		UserAudioReadErrors:                    2,
		UserAudioConsecutiveReadErrors:         1,
		UserAudioEmptyRTPPayloads:              3,
		UserAudioRTPUnmarshalFailures:          4,
		UserAudioOpusDecodeFailures:            5,
		UserAudioResampleFailures:              6,
		FirstAssistantAudioQueuedAt:            now,
		LastAssistantAudioQueuedAt:             now,
		LastAssistantFrameSentAt:               now,
		AssistantFrameWriteFailures:            7,
		ConsecutiveAssistantFrameWriteFailures: 8,
		LastAssistantFrameWriteFailureAt:       now,
		ReceiverReports:                        9,
		LastReceiverReportAt:                   now,
		LastReceiverReportFractionLost:         10,
		LastReceiverReportPacketLossPercent:    9.5,
		LastReceiverReportTotalLost:            11,
		LastReceiverReportJitterMs:             11.5,
		LastReceiverReportRoundTripTimeMs:      12,
		LastReceiverReportRoundTripTimeUsable:  true,
	}

	s.stopMediaSessionAndFallbackToText()

	assert.False(t, s.sessionState.PeerConnected(), "peerConnected should be false after reset")
	assert.Greater(t, s.sessionState.ActiveMediaSessionID(), previousMediaSessionID)
	s.Mu.Lock()
	assert.Nil(t, s.peerConnection, "peer connection should be nil after reset")
	assert.Nil(t, s.assistantAudioTrack, "assistant audio track should be nil after reset")
	assert.Empty(t, s.signalingSessionID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	assert.Equal(t, webrtc_internal.MediaHealthState{}, s.mediaHealthState)
	s.Mu.Unlock()
}

func TestHandleClientSignaling_IgnoresStaleSignalingSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "current-signaling-session"

	s.queueClientSignal(&protos.ClientSignaling{
		SessionId: "stale-signaling-session",
		Message: &protos.ClientSignaling_IceCandidate{
			IceCandidate: &protos.ICECandidate{Candidate: "candidate:1 1 udp 1 127.0.0.1 9 typ host"},
		},
	})

	assert.False(t, s.sessionState.CloseStarted())
}

func TestHandleClientSignal_QueuesRemoteICEUntilAnswer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "current-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	s.handleClientSignal(&protos.ClientSignaling{
		SessionId: "current-signaling-session",
		Message: &protos.ClientSignaling_IceCandidate{
			IceCandidate: &protos.ICECandidate{
				Candidate:        "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host",
				SdpMid:           "audio",
				SdpMLineIndex:    0,
				UsernameFragment: "remote",
			},
		},
	})

	s.Mu.Lock()
	defer s.Mu.Unlock()
	require.Len(t, s.signalPendingRemoteICECandidates, 1)
	assert.Equal(t, "candidate:1 1 udp 2130706431 127.0.0.1 9 typ host", s.signalPendingRemoteICECandidates[0].Candidate)
}

func TestHandleClientSignal_CapsPendingRemoteICEBeforeAnswer(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.signalingSessionID = "current-signaling-session"
	mediaSessionID := s.sessionState.StartMediaSession()
	require.NoError(t, s.createPeer(mediaSessionID))
	t.Cleanup(func() { s.stopMediaSession() })

	for i := 0; i < webrtc_internal.PendingRemoteICECandidateLimit+1; i++ {
		s.handleClientSignal(&protos.ClientSignaling{
			SessionId: "current-signaling-session",
			Message: &protos.ClientSignaling_IceCandidate{
				IceCandidate: &protos.ICECandidate{
					Candidate:        fmt.Sprintf("candidate:%d 1 udp 2130706431 127.0.0.1 9 typ host", i+1),
					SdpMid:           "audio",
					SdpMLineIndex:    0,
					UsernameFragment: "remote",
				},
			},
		})
	}

	s.Mu.Lock()
	defer s.Mu.Unlock()
	assert.Len(t, s.signalPendingRemoteICECandidates, webrtc_internal.PendingRemoteICECandidateLimit)
}

func TestStopMediaSession_InvalidatesCurrentMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	previousMediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)

	s.stopMediaSession()

	assert.False(t, s.sessionState.PeerConnected())
	assert.Greater(t, s.sessionState.ActiveMediaSessionID(), previousMediaSessionID)
	s.Mu.Lock()
	assert.Nil(t, s.peerConnection)
	assert.Nil(t, s.assistantAudioTrack)
	assert.Nil(t, s.assistantRTPSender)
	assert.Nil(t, s.mediaCtx)
	assert.Nil(t, s.cancelMedia)
	s.Mu.Unlock()
}

func TestHandlePeerState_ClosedStopsMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.signalingSessionID = "active-signaling"
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO

	s.handlePeerState(mediaSessionID, pionwebrtc.PeerConnectionStateClosed, time.Now())

	assert.False(t, s.sessionState.PeerConnected())
	s.Mu.Lock()
	assert.Empty(t, s.signalingSessionID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	s.Mu.Unlock()
}

func TestQueueMediaSessionRestart_QueuesLifecycleEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.mediaLifecycleCh = make(chan mediaLifecycleEvent, 1)
	requestedAt := time.Now()

	s.queueMediaSessionRestart(42, webrtc_internal.ReasonPeerFailed, requestedAt)

	select {
	case event := <-s.mediaLifecycleCh:
		assert.Equal(t, mediaLifecycleEventRestart, event.kind)
		assert.Equal(t, uint64(42), event.mediaSessionID)
		assert.Equal(t, webrtc_internal.ReasonPeerFailed, event.reason)
		assert.Equal(t, requestedAt, event.requestedAt)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for media lifecycle event")
	}
}

func TestHandlePeerICEConnectionState_RecordsSeparateICEState(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	changedAt := time.Now()

	s.handlePeerICEConnectionState(mediaSessionID, pionwebrtc.ICEConnectionStateChecking, changedAt)

	s.Mu.Lock()
	assert.Equal(t, webrtc_internal.ICEStateChecking, s.mediaHealthState.ICEConnectionState)
	assert.Equal(t, changedAt, s.mediaHealthState.ICEConnectionStateChangedAt)
	assert.Equal(t, changedAt, s.mediaHealthState.ICECheckingStartedAt)
	s.Mu.Unlock()

	select {
	case msg := <-s.LowCh:
		event, ok := msg.(*protos.ConversationEvent)
		require.True(t, ok, "expected ConversationEvent, got %T", msg)
		assert.Equal(t, webrtc_internal.EventICEConnectionState, event.GetData()[webrtc_internal.DataType])
		assert.Equal(t, webrtc_internal.ICEStateChecking, event.GetData()[webrtc_internal.DataICEConnectionState])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ICE state event")
	}
}

func TestRestartMediaSession_LimitFallsBackToText(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.signalingSessionID = "active-signaling"
	s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
	_, ok := s.sessionState.TryBeginMediaRestart(webrtc_internal.MediaRestartAttemptLimit)
	require.True(t, ok)

	s.restartMediaSessionOrFallbackToText(mediaSessionID, webrtc_internal.ReasonPeerFailed, time.Now())

	assert.False(t, s.sessionState.PeerConnected())
	s.Mu.Lock()
	assert.Empty(t, s.signalingSessionID)
	assert.Equal(t, protos.StreamMode_STREAM_MODE_TEXT, s.currentMode)
	s.Mu.Unlock()
}

func TestMediaHealthState_RecordsInputMediaHealth(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}

	state.StartICE(now)
	state.RecordPeerConnected(now.Add(time.Millisecond))
	state.RecordUserAudioReadError(1)
	state.RecordUserAudioReadError(2)
	state.RecordUserAudioRTPUnmarshalFailure()
	state.RecordUserAudioEmptyRTPPayload()
	state.RecordUserAudioOpusDecodeFailure()
	state.RecordUserAudioResampleFailure()
	state.RecordUserAudioReceived(now.Add(2 * time.Millisecond))

	assert.Equal(t, now, state.ICEStartedAt)
	assert.Equal(t, now.Add(time.Millisecond), state.PeerConnectedAt)
	assert.Equal(t, uint64(2), state.UserAudioReadErrors)
	assert.Equal(t, 0, state.UserAudioConsecutiveReadErrors)
	assert.Equal(t, uint64(1), state.UserAudioRTPUnmarshalFailures)
	assert.Equal(t, uint64(1), state.UserAudioEmptyRTPPayloads)
	assert.Equal(t, uint64(1), state.UserAudioOpusDecodeFailures)
	assert.Equal(t, uint64(1), state.UserAudioResampleFailures)
	assert.Equal(t, now.Add(2*time.Millisecond), state.FirstUserAudioReceivedAt)
	assert.Equal(t, now.Add(2*time.Millisecond), state.LastUserAudioReceivedAt)
}

func TestMediaHealthState_HandshakeDeadlineExceeded(t *testing.T) {
	t.Parallel()
	now := time.Now()

	tests := []struct {
		name  string
		state webrtc_internal.MediaHealthState
		at    time.Time
		want  string
		ok    bool
	}{
		{
			name: "remote_answer_deadline",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt: now,
			},
			at:   now.Add(webrtc_internal.SignalingAnswerDeadline),
			want: webrtc_internal.ReasonRemoteAnswerDeadline,
			ok:   true,
		},
		{
			name: "ice_connected_deadline",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt:            now,
				RemoteDescriptionSetAt: now.Add(time.Millisecond),
			},
			at:   now.Add(webrtc_internal.ICEConnectedDeadline),
			want: webrtc_internal.ReasonICEConnectedDeadline,
			ok:   true,
		},
		{
			name: "peer_connected_deadline",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt:            now,
				RemoteDescriptionSetAt: now.Add(time.Millisecond),
				ICEConnectedAt:         now.Add(2 * time.Millisecond),
			},
			at:   now.Add(webrtc_internal.PeerConnectedDeadline),
			want: webrtc_internal.ReasonPeerConnectedDeadline,
			ok:   true,
		},
		{
			name: "connected",
			state: webrtc_internal.MediaHealthState{
				OfferSentAt:            now,
				RemoteDescriptionSetAt: now.Add(time.Millisecond),
				ICEConnectedAt:         now.Add(2 * time.Millisecond),
				PeerConnectedAt:        now.Add(3 * time.Millisecond),
			},
			at: now.Add(webrtc_internal.PeerConnectedDeadline),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reason, _, _, ok := tt.state.HandshakeDeadlineExceeded(tt.at)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, reason)
		})
	}
}

func TestMediaHealthState_RecordsReceiverReportQuality(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}

	state.RecordReceiverReport(
		now,
		64,
		3,
		webrtc_internal.OpusFrameSamples,
		0,
		0,
	)

	assert.Equal(t, uint64(1), state.ReceiverReports)
	assert.Equal(t, now, state.LastReceiverReportAt)
	assert.Equal(t, uint8(64), state.LastReceiverReportFractionLost)
	assert.Equal(t, 25.0, state.LastReceiverReportPacketLossPercent)
	assert.Equal(t, uint32(3), state.LastReceiverReportTotalLost)
	assert.Equal(t, 20.0, state.LastReceiverReportJitterMs)
	assert.False(t, state.LastReceiverReportRoundTripTimeUsable)
	assert.Zero(t, state.LastReceiverReportRoundTripTimeMs)
}

func TestMediaHealthState_RecordsReceiverReportRoundTripTime(t *testing.T) {
	t.Parallel()
	now := time.Now()
	delayMs := int64(20)
	roundTripMs := int64(80)
	state := webrtc_internal.MediaHealthState{}
	lastSenderReport := webrtc_internal.CompactNTP(now.Add(-time.Duration(delayMs+roundTripMs) * time.Millisecond))
	delayUnits := uint32(delayMs * webrtc_internal.RTCPCompactNTPUnitsPerSec / webrtc_internal.MillisecondsPerSecond)

	state.RecordReceiverReport(
		now,
		0,
		0,
		0,
		lastSenderReport,
		delayUnits,
	)

	assert.True(t, state.LastReceiverReportRoundTripTimeUsable)
	assert.InDelta(t, roundTripMs, state.LastReceiverReportRoundTripTimeMs, 1)
}

func TestMediaHealthState_QualityState(t *testing.T) {
	t.Parallel()
	now := time.Now()

	tests := []struct {
		name  string
		state webrtc_internal.MediaHealthState
		want  string
	}{
		{
			name: "excellent",
			state: webrtc_internal.MediaHealthState{
				LastReceiverReportPacketLossPercent: 1.0,
				LastReceiverReportJitterMs:          10.0,
			},
			want: webrtc_internal.QualityStateExcellent,
		},
		{
			name: "good",
			state: webrtc_internal.MediaHealthState{
				LastReceiverReportPacketLossPercent: webrtc_internal.QualityGoodPacketLossPercent,
			},
			want: webrtc_internal.QualityStateGood,
		},
		{
			name: "poor",
			state: webrtc_internal.MediaHealthState{
				LastReceiverReportJitterMs: webrtc_internal.QualityPoorJitterMs,
			},
			want: webrtc_internal.QualityStatePoor,
		},
		{
			name: "lost_on_write_failures",
			state: webrtc_internal.MediaHealthState{
				ConsecutiveAssistantFrameWriteFailures: webrtc_internal.RepeatedWriteFailuresThreshold,
			},
			want: webrtc_internal.QualityStateLost,
		},
		{
			name: "lost_on_missing_feedback",
			state: webrtc_internal.MediaHealthState{
				LastAssistantFrameSentAt: now.Add(-webrtc_internal.RTCPFeedbackMissingThreshold),
			},
			want: webrtc_internal.QualityStateLost,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.state.QualityState(now))
		})
	}
}

func TestMediaHealthState_RecordsAssistantWriteFailures(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}

	state.RecordAssistantFrameWriteFailure(now)
	state.RecordAssistantFrameWriteFailure(now.Add(time.Millisecond))

	assert.Equal(t, uint64(2), state.AssistantFrameWriteFailures)
	assert.Equal(t, uint64(2), state.ConsecutiveAssistantFrameWriteFailures)
	assert.Equal(t, now.Add(time.Millisecond), state.LastAssistantFrameWriteFailureAt)

	state.RecordAssistantFrameSent(now.Add(2 * time.Millisecond))

	assert.Equal(t, uint64(2), state.AssistantFrameWriteFailures)
	assert.Zero(t, state.ConsecutiveAssistantFrameWriteFailures)
	assert.Equal(t, now.Add(2*time.Millisecond), state.LastAssistantFrameSentAt)
}

func TestMediaHealthState_RecordsSelectedICECandidatePair(t *testing.T) {
	t.Parallel()
	now := time.Now()
	state := webrtc_internal.MediaHealthState{}
	pair := webrtc_internal.SelectedICECandidatePair{
		ID:                          "pair-1",
		LocalCandidateType:          "host",
		LocalProtocol:               "udp",
		RemoteCandidateType:         "srflx",
		RemoteProtocol:              "udp",
		CurrentRoundTripTimeMs:      42,
		AvailableOutgoingBitrateBps: 64000,
	}

	assert.True(t, state.RecordSelectedICECandidatePair(pair, now))
	assert.Equal(t, "pair-1", state.SelectedICECandidatePairID)
	assert.Equal(t, "host", state.SelectedICELocalCandidateType)
	assert.Equal(t, "udp", state.SelectedICELocalProtocol)
	assert.Equal(t, "srflx", state.SelectedICERemoteCandidateType)
	assert.Equal(t, int64(42), state.SelectedICECandidatePairRTTMs)
	assert.Equal(t, now, state.SelectedICECandidatePairChangedAt)

	assert.False(t, state.RecordSelectedICECandidatePair(pair, now.Add(time.Second)))
	assert.Equal(t, now, state.SelectedICECandidatePairChangedAt)
}

func TestSelectedICECandidatePairFromStats(t *testing.T) {
	t.Parallel()
	report := pionwebrtc.StatsReport{
		"local": pionwebrtc.ICECandidateStats{
			ID:            "local",
			Protocol:      "udp",
			CandidateType: pionwebrtc.ICECandidateTypeHost,
		},
		"remote": pionwebrtc.ICECandidateStats{
			ID:            "remote",
			Protocol:      "tcp",
			CandidateType: pionwebrtc.ICECandidateTypeRelay,
		},
		"pair": pionwebrtc.ICECandidatePairStats{
			ID:                       "pair",
			LocalCandidateID:         "local",
			RemoteCandidateID:        "remote",
			State:                    pionwebrtc.StatsICECandidatePairStateSucceeded,
			Nominated:                true,
			CurrentRoundTripTime:     0.125,
			AvailableOutgoingBitrate: 48000,
		},
	}

	pair, ok := selectedICECandidatePairFromStats(report)

	require.True(t, ok)
	assert.Equal(t, "pair", pair.ID)
	assert.Equal(t, "host", pair.LocalCandidateType)
	assert.Equal(t, "udp", pair.LocalProtocol)
	assert.Equal(t, "relay", pair.RemoteCandidateType)
	assert.Equal(t, "tcp", pair.RemoteProtocol)
	assert.Equal(t, int64(125), pair.CurrentRoundTripTimeMs)
	assert.Equal(t, int64(48000), pair.AvailableOutgoingBitrateBps)
}

func TestResetAudioSession_FlushesPendingOutput(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	s.WithOutputBuffer(func(buf *bytes.Buffer) {
		buf.Write([]byte{0x01, 0x02, 0x03, 0x04})
	})
	s.Output(&protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Audio{Audio: []byte{0xAA, 0xBB}},
	})

	s.stopMediaSessionAndFallbackToText()

	s.WithOutputBuffer(func(buf *bytes.Buffer) {
		assert.Equal(t, 0, buf.Len(), "output accumulation buffer should be cleared")
	})

	select {
	case <-s.OutputCh:
		t.Fatal("output channel should be drained after reset")
	default:
	}
}

func TestSend_TextMessage(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Text{Text: "hello"},
	}
	err := s.Send(msg)
	assert.NoError(t, err)
}

func TestSend_AudioBuffersWebRTCOutputPCM16kFrame(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	audio := bytes.Repeat([]byte{0x22}, webRTCOutputPCM16kFrameBytes)
	msg := &protos.ConversationAssistantMessage{
		Message: &protos.ConversationAssistantMessage_Audio{Audio: audio},
	}

	err := s.Send(msg)
	require.NoError(t, err)

	select {
	case out := <-s.OutputCh:
		assistant, ok := out.(*protos.ConversationAssistantMessage)
		require.True(t, ok, "expected ConversationAssistantMessage, got %T", out)
		got := assistant.GetAudio()
		assert.Len(t, got, webRTCOutputPCM16kFrameBytes)
		assert.Equal(t, audio, got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for assistant audio")
	}
}

func TestSend_Interruption(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.enqueueOutputAudio([]byte{0x10})
	s.enqueueOutputAudio([]byte{0x20})

	msg := &protos.ConversationInterruption{
		Type: protos.ConversationInterruption_INTERRUPTION_TYPE_WORD,
	}
	err := s.Send(msg)
	assert.NoError(t, err)

	s.outputAudioQueueMu.Lock()
	assert.Empty(t, s.outputAudioQueue)
	s.outputAudioQueueMu.Unlock()

	select {
	case msg := <-s.LowCh:
		eventMsg, ok := msg.(*protos.ConversationEvent)
		require.True(t, ok, "expected ConversationEvent, got %T", msg)
		assert.Equal(t, observe.ComponentWebRTC, eventMsg.GetName())
		assert.Equal(t, webrtc_internal.EventOutputQueueCleared, eventMsg.GetData()[webrtc_internal.DataType])
		assert.Equal(t, webrtc_internal.OutputQueueClearReasonInterruption, eventMsg.GetData()[webrtc_internal.DataReason])
		assert.Equal(t, "2", eventMsg.GetData()[webrtc_internal.DataClearedFrames])
		assert.Equal(t, fmt.Sprintf("%d", webrtc_internal.OutputAudioQueueEmptySize), eventMsg.GetData()[webrtc_internal.DataRemainingQueueFrames])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for output queue cleared event")
	}
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

func TestQueueClientSignal_QueuesPeerEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.peerEventCh = make(chan peerEvent, 1)
	signaling := &protos.ClientSignaling{
		SessionId: "signaling-session",
		Message: &protos.ClientSignaling_Disconnect{
			Disconnect: true,
		},
	}

	s.queueClientSignal(signaling)

	select {
	case event := <-s.peerEventCh:
		assert.Equal(t, signalEventClientMessage, event.kind)
		assert.Same(t, signaling, event.signalClientMessage)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for peer event")
	}
}

func TestEnqueuePeerEvent_PreservesPeerConnectionStateTransitions(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.peerEventCh = make(chan peerEvent, 2)

	s.enqueuePeerEvent(peerEvent{
		kind:               peerEventStateChanged,
		mediaSessionID:     10,
		peerState:          pionwebrtc.PeerConnectionStateDisconnected,
		peerStateChangedAt: time.Now(),
	})
	s.enqueuePeerEvent(peerEvent{
		kind:               peerEventStateChanged,
		mediaSessionID:     10,
		peerState:          pionwebrtc.PeerConnectionStateConnected,
		peerStateChangedAt: time.Now().Add(time.Millisecond),
	})

	first := <-s.peerEventCh
	second := <-s.peerEventCh
	assert.Equal(t, pionwebrtc.PeerConnectionStateDisconnected, first.peerState)
	assert.Equal(t, pionwebrtc.PeerConnectionStateConnected, second.peerState)
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
		ambientOut: make([]byte, webRTCOutputPCM16kFrameBytes),
	}
	for i := range fake.ambientOut {
		fake.ambientOut[i] = 0x11
	}
	s.ambientMixer = fake

	out := s.applyAmbientToFrame(nil)
	require.NotNil(t, out)
	assert.Len(t, out, webRTCOutputPCM16kFrameBytes)
	assert.NotEqual(t, make([]byte, len(out)), out)
}

func TestApplyAmbientToFrame_NoneLeavesPrimaryUntouched(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.ambientMixer = nil

	pcm16k := make([]byte, webRTCOutputPCM16kFrameBytes)
	for i := range pcm16k {
		pcm16k[i] = byte(i % 251)
	}
	out := s.applyAmbientToFrame(pcm16k)
	assert.Equal(t, pcm16k, out)
}

func TestEnqueueOutputAudio_BoundedDropOldest_EmitsOverflowEvent(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	limit := webrtc_internal.OutputAudioQueueMaxFrames
	for i := 0; i < limit+1; i++ {
		s.enqueueOutputAudio([]byte{byte(i % 251)})
	}

	s.outputAudioQueueMu.Lock()
	require.Len(t, s.outputAudioQueue, limit)
	// Oldest frame should be dropped on overflow; new head is former index 1.
	require.Len(t, s.outputAudioQueue[0].Audio, 1)
	assert.Equal(t, byte(1), s.outputAudioQueue[0].Audio[0])
	assert.False(t, s.outputAudioQueue[0].QueuedAt.IsZero())
	s.outputAudioQueueMu.Unlock()

	select {
	case msg := <-s.LowCh:
		eventMsg, ok := msg.(*protos.ConversationEvent)
		require.True(t, ok, "expected ConversationEvent, got %T", msg)
		assert.Equal(t, "webrtc", eventMsg.GetName())
		assert.Equal(t, webrtc_internal.EventOutputQueueOverflow, eventMsg.GetData()[webrtc_internal.DataType])
		assert.Equal(t, webrtc_internal.OutputQueuePolicyDropOldest, eventMsg.GetData()[webrtc_internal.DataPolicy])
		assert.Equal(t, "1", eventMsg.GetData()[webrtc_internal.DataDroppedFrames])
		assert.Equal(t, fmt.Sprintf("%d", limit), eventMsg.GetData()[webrtc_internal.DataLimitFrames])
		assert.Equal(t, fmt.Sprintf("%d", limit), eventMsg.GetData()[webrtc_internal.DataQueueDepthFrames])
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for overflow event")
	}
}

func TestClearOutputAudio_ReturnsClearedFrameCount(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	s.enqueueOutputAudio([]byte{0x01})
	s.enqueueOutputAudio([]byte{0x02})

	clearedFrames := s.clearOutputAudio()

	assert.Equal(t, 2, clearedFrames)
	s.outputAudioQueueMu.Lock()
	assert.Empty(t, s.outputAudioQueue)
	s.outputAudioQueueMu.Unlock()
}

func TestEnqueueOutputAudio_StoresQueuedAt(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	audio := []byte{0x11, 0x22}

	queuedBefore := time.Now()
	s.enqueueOutputAudio(audio)
	queuedAfter := time.Now()

	s.outputAudioQueueMu.Lock()
	require.Len(t, s.outputAudioQueue, 1)
	outputFrame := s.outputAudioQueue[0]
	s.outputAudioQueueMu.Unlock()

	assert.Equal(t, audio, outputFrame.Audio)
	assert.False(t, outputFrame.QueuedAt.IsZero())
	assert.False(t, outputFrame.QueuedAt.Before(queuedBefore))
	assert.False(t, outputFrame.QueuedAt.After(queuedAfter))
}

func TestEnqueueOutputAudio_TracksFirstAssistantAudioQueuedAt(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	s.enqueueOutputAudio([]byte{0x01})

	s.Mu.Lock()
	firstQueuedAt := s.mediaHealthState.FirstAssistantAudioQueuedAt
	lastQueuedAt := s.mediaHealthState.LastAssistantAudioQueuedAt
	s.Mu.Unlock()

	require.False(t, firstQueuedAt.IsZero())
	assert.Equal(t, firstQueuedAt, lastQueuedAt)

	time.Sleep(time.Millisecond)
	s.enqueueOutputAudio([]byte{0x02})

	s.Mu.Lock()
	assert.Equal(t, firstQueuedAt, s.mediaHealthState.FirstAssistantAudioQueuedAt)
	assert.True(t, s.mediaHealthState.LastAssistantAudioQueuedAt.After(firstQueuedAt))
	s.Mu.Unlock()
}

func TestNextFrame_StampsActiveMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	mediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	s.enqueueOutputAudio([]byte{0x01, 0x02})

	frame := s.NextFrame()

	assert.Equal(t, []byte{0x01, 0x02}, frame)
	assert.Equal(t, mediaSessionID, s.sessionState.PacedAssistantFrameMediaSessionID())
}

func TestConsumeFrame_TracksWriteFailureWithoutRecordingAssistantAudio(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	assistantPCM16k := bytes.Repeat([]byte{0x33}, webRTCOutputPCM16kFrameBytes)

	err := s.ConsumeFrame(assistantPCM16k)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assistant audio track is not ready")
	assert.True(t, s.mediaHealthState.LastAssistantFrameSentAt.IsZero())
	assert.Equal(t, uint64(1), s.mediaHealthState.AssistantFrameWriteFailures)
	assert.Equal(t, uint64(1), s.mediaHealthState.ConsecutiveAssistantFrameWriteFailures)
	assert.False(t, s.mediaHealthState.LastAssistantFrameWriteFailureAt.IsZero())

	select {
	case msg := <-s.InputCh:
		t.Fatalf("failed assistant frame should not be recorded, got %T", msg)
	default:
	}
}

func TestConsumeFrame_DropsStalePacedMediaSession(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	staleMediaSessionID := s.sessionState.StartMediaSession()
	s.sessionState.StampPacedAssistantFrame(staleMediaSessionID)
	s.sessionState.StartMediaSession()
	s.sessionState.SetPeerConnected(true)
	assistantPCM16k := bytes.Repeat([]byte{0x55}, webRTCOutputPCM16kFrameBytes)

	err := s.ConsumeFrame(assistantPCM16k)

	require.NoError(t, err)
	assert.True(t, s.mediaHealthState.LastAssistantFrameSentAt.IsZero())
	assert.Zero(t, s.mediaHealthState.AssistantFrameWriteFailures)

	select {
	case msg := <-s.InputCh:
		t.Fatalf("stale assistant frame should not be recorded, got %T", msg)
	default:
	}
}

func TestConsumeFrame_TracksLastAssistantFrameSentAt(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)
	assistantAudioTrack, err := pionwebrtc.NewTrackLocalStaticSample(
		pionwebrtc.RTPCodecCapability{
			MimeType:  pionwebrtc.MimeTypeOpus,
			ClockRate: webrtc_internal.OpusSampleRate,
			Channels:  webrtc_internal.OpusChannels,
		},
		"audio",
		"rapida-audio",
	)
	require.NoError(t, err)
	s.assistantAudioTrack = assistantAudioTrack

	assistantPCM16k := bytes.Repeat([]byte{0x44}, webRTCOutputPCM16kFrameBytes)

	err = s.ConsumeFrame(assistantPCM16k)
	require.NoError(t, err)

	s.Mu.Lock()
	lastSentAt := s.mediaHealthState.LastAssistantFrameSentAt
	s.Mu.Unlock()

	assert.False(t, lastSentAt.IsZero())

	select {
	case msg := <-s.InputCh:
		bridge, ok := msg.(*protos.ConversationBridgeOperatorAudio)
		require.True(t, ok, "expected ConversationBridgeOperatorAudio, got %T", msg)
		assert.Equal(t, assistantPCM16k, bridge.GetAudio())
		assert.NotNil(t, bridge.GetTime())
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for assistant bridge audio")
	}
}

func TestWriteAudioFrame_ReturnsErrorWhenAssistantTrackMissing(t *testing.T) {
	t.Parallel()
	s := newTestStreamer(t)

	err := s.writeAudioFrame([]byte{0x01, 0x02})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "assistant audio track is not ready")
}
