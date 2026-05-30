// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"fmt"
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observe"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *webrtcStreamer) runPeerEventLoop() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		case event := <-s.peerEventCh:
			switch event.Kind {
			case webrtc_internal.SignalEventClientMessage:
				s.handleClientSignal(event.SignalClientMessage)
			case webrtc_internal.PeerEventStateChanged:
				s.handlePeerState(event.MediaSessionID, event.PeerState, event.PeerStateChangedAt)
			case webrtc_internal.PeerEventICEConnectionStateChanged:
				s.handlePeerICEConnectionState(event.MediaSessionID, event.PeerICEState, event.PeerICEStateChangedAt)
			}
		}
	}
}

func (s *webrtcStreamer) enqueuePeerEvent(event webrtc_internal.PeerEvent) {
	if s.peerEventCh == nil {
		switch event.Kind {
		case webrtc_internal.SignalEventClientMessage:
			s.handleClientSignal(event.SignalClientMessage)
		case webrtc_internal.PeerEventStateChanged:
			s.handlePeerState(event.MediaSessionID, event.PeerState, event.PeerStateChangedAt)
		case webrtc_internal.PeerEventICEConnectionStateChanged:
			s.handlePeerICEConnectionState(event.MediaSessionID, event.PeerICEState, event.PeerICEStateChangedAt)
		}
		return
	}

	select {
	case s.peerEventCh <- event:
	case <-s.Ctx.Done():
	}
}

func (s *webrtcStreamer) handlePeerState(mediaSessionID uint64, state pionwebrtc.PeerConnectionState, peerStateChangedAt time.Time) {
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	s.Logger.Infow("WebRTC connection state changed", "state", state, "session", s.sessionID)

	s.Mu.Lock()
	switch state {
	case pionwebrtc.PeerConnectionStateConnected:
		s.currentMode = protos.StreamMode_STREAM_MODE_AUDIO
		s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioConnected)
		s.mediaHealthState.RecordPeerConnected(peerStateChangedAt)
	case pionwebrtc.PeerConnectionStateFailed,
		pionwebrtc.PeerConnectionStateDisconnected:
		s.sessionState.SetMediaState(webrtc_internal.MediaStateAudioNegotiating)
		if state == pionwebrtc.PeerConnectionStateFailed {
			s.mediaHealthState.RecordPeerFailed(peerStateChangedAt)
		} else {
			s.mediaHealthState.RecordPeerDisconnected(peerStateChangedAt)
		}
	case pionwebrtc.PeerConnectionStateClosed:
		s.currentMode = protos.StreamMode_STREAM_MODE_TEXT
		s.sessionState.SetMediaState(webrtc_internal.MediaStateText)
		s.mediaHealthState.RecordPeerClosed(peerStateChangedAt)
	}
	iceLatencyMs := peerStateChangedAt.Sub(s.mediaHealthState.ICEStartedAt).Milliseconds()
	peerConnection := s.peerConnection
	s.Mu.Unlock()

	switch state {
	case pionwebrtc.PeerConnectionStateConnected:
		s.sessionState.SetPeerConnected(true)
		s.sessionState.ResetICERestartAttempts()
		s.Input(&protos.ConversationEvent{
			Name: observe.ComponentWebRTC,
			Data: map[string]string{
				webrtc_internal.DataType:         observe.EventPeerConnected,
				webrtc_internal.DataSessionID:    s.sessionID,
				webrtc_internal.DataICELatencyMs: fmt.Sprintf("%d", iceLatencyMs),
			},
			Time: timestamppb.Now(),
		})
		s.reportSelectedICECandidatePair(peerConnection, peerStateChangedAt)
		s.signalReady()

	case pionwebrtc.PeerConnectionStateFailed:
		s.sessionState.SetPeerConnected(false)
		s.Logger.Warnw("WebRTC peer failed, restarting ICE", "session", s.sessionID)
		s.Input(&protos.ConversationEvent{
			Name: observe.ComponentWebRTC,
			Data: map[string]string{
				webrtc_internal.DataType:      observe.EventPeerFailed,
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataReason:    webrtc_internal.ReasonPeerFailed,
			},
			Time: timestamppb.Now(),
		})
		s.queueMediaSessionRecovery(mediaSessionID, webrtc_internal.ReasonPeerFailed, peerStateChangedAt)

	case pionwebrtc.PeerConnectionStateDisconnected:
		s.sessionState.SetPeerConnected(false)
		s.Logger.Warnw("WebRTC peer disconnected, restarting ICE", "session", s.sessionID)
		s.Input(&protos.ConversationEvent{
			Name: observe.ComponentWebRTC,
			Data: map[string]string{
				webrtc_internal.DataType:      observe.EventPeerDisconnected,
				webrtc_internal.DataSessionID: s.sessionID,
			},
			Time: timestamppb.Now(),
		})
		s.queueMediaSessionRecovery(mediaSessionID, webrtc_internal.ReasonPeerDisconnected, peerStateChangedAt)

	case pionwebrtc.PeerConnectionStateClosed:
		s.Logger.Infow("WebRTC peer closed, resetting audio", "session", s.sessionID)
		s.stopMediaSessionAndFallbackToText()
	}
}

func (s *webrtcStreamer) handlePeerICEConnectionState(mediaSessionID uint64, state pionwebrtc.ICEConnectionState, iceStateChangedAt time.Time) {
	if !s.sessionState.IsActiveMediaSession(mediaSessionID) {
		return
	}

	s.Logger.Infow("WebRTC ICE connection state changed", "state", state, "session", s.sessionID)

	stateName := state.String()
	s.Mu.Lock()
	s.mediaHealthState.RecordICEConnectionState(stateName, iceStateChangedAt)
	s.Mu.Unlock()

	eventType := webrtc_internal.EventICEConnectionState
	if state == pionwebrtc.ICEConnectionStateConnected || state == pionwebrtc.ICEConnectionStateCompleted {
		eventType = observe.EventICEConnected
	}
	if state == pionwebrtc.ICEConnectionStateFailed {
		eventType = observe.EventICEFailed
	}

	s.Input(&protos.ConversationEvent{
		Name: observe.ComponentWebRTC,
		Data: map[string]string{
			webrtc_internal.DataType:               eventType,
			webrtc_internal.DataSessionID:          s.sessionID,
			webrtc_internal.DataICEConnectionState: stateName,
		},
		Time: timestamppb.New(iceStateChangedAt),
	})

	if state == pionwebrtc.ICEConnectionStateConnected || state == pionwebrtc.ICEConnectionStateCompleted {
		s.sessionState.ResetICERestartAttempts()
	}
	if state == pionwebrtc.ICEConnectionStateFailed {
		s.sessionState.SetPeerConnected(false)
		s.queueMediaSessionRecovery(mediaSessionID, webrtc_internal.ReasonICEFailed, iceStateChangedAt)
	}
}

func (s *webrtcStreamer) reportSelectedICECandidatePair(peerConnection *pionwebrtc.PeerConnection, selectedAt time.Time) {
	if peerConnection == nil {
		return
	}
	pair, ok := selectedICECandidatePairFromStats(peerConnection.GetStats())
	if !ok {
		return
	}

	s.Mu.Lock()
	changed := s.mediaHealthState.RecordSelectedICECandidatePair(pair, selectedAt)
	qualityState := s.mediaHealthState.QualityState(selectedAt)
	s.Mu.Unlock()
	if !changed {
		return
	}

	s.Input(&protos.ConversationEvent{
		Name: observe.ComponentWebRTC,
		Data: map[string]string{
			webrtc_internal.DataType:                        webrtc_internal.EventSelectedICECandidatePair,
			webrtc_internal.DataSessionID:                   s.sessionID,
			webrtc_internal.DataCandidatePairID:             pair.ID,
			webrtc_internal.DataLocalCandidateType:          pair.LocalCandidateType,
			webrtc_internal.DataLocalProtocol:               pair.LocalProtocol,
			webrtc_internal.DataRemoteCandidateType:         pair.RemoteCandidateType,
			webrtc_internal.DataRemoteProtocol:              pair.RemoteProtocol,
			webrtc_internal.DataCandidatePairRTTMs:          fmt.Sprintf("%d", pair.CurrentRoundTripTimeMs),
			webrtc_internal.DataAvailableOutgoingBitrateBps: fmt.Sprintf("%d", pair.AvailableOutgoingBitrateBps),
			webrtc_internal.DataQualityState:                qualityState,
		},
		Time: timestamppb.New(selectedAt),
	})
}

func selectedICECandidatePairFromStats(report pionwebrtc.StatsReport) (webrtc_internal.SelectedICECandidatePair, bool) {
	candidates := make(map[string]pionwebrtc.ICECandidateStats)
	var selectedPair pionwebrtc.ICECandidatePairStats
	selected := false

	for _, stat := range report {
		switch typed := stat.(type) {
		case pionwebrtc.ICECandidateStats:
			candidates[typed.ID] = typed
		case pionwebrtc.ICECandidatePairStats:
			if typed.Nominated && typed.State == pionwebrtc.StatsICECandidatePairStateSucceeded {
				selectedPair = typed
				selected = true
			}
		}
	}
	if !selected {
		return webrtc_internal.SelectedICECandidatePair{}, false
	}

	localCandidate := candidates[selectedPair.LocalCandidateID]
	remoteCandidate := candidates[selectedPair.RemoteCandidateID]
	return webrtc_internal.SelectedICECandidatePair{
		ID:                          selectedPair.ID,
		LocalCandidateType:          localCandidate.CandidateType.String(),
		LocalProtocol:               localCandidate.Protocol,
		RemoteCandidateType:         remoteCandidate.CandidateType.String(),
		RemoteProtocol:              remoteCandidate.Protocol,
		CurrentRoundTripTimeMs:      int64(selectedPair.CurrentRoundTripTime * float64(webrtc_internal.MillisecondsPerSecond)),
		AvailableOutgoingBitrateBps: int64(selectedPair.AvailableOutgoingBitrate),
	}, true
}

// handleClientSignal applies already-ordered SDP and ICE updates from the browser.
func (s *webrtcStreamer) handleClientSignal(signaling *protos.ClientSignaling) {
	if signaling == nil {
		return
	}

	s.Mu.Lock()
	signalingSessionID := s.signalingSessionID
	mediaSessionID := s.sessionState.ActiveMediaSessionID()
	s.Mu.Unlock()

	switch msg := signaling.GetMessage().(type) {
	case *protos.ClientSignaling_Sdp:
		if signalingSessionID != "" && signaling.GetSessionId() != signalingSessionID {
			s.Logger.Warnw("Received SDP for stale WebRTC signaling session, ignoring", "session", s.sessionID)
			return
		}
		if msg.Sdp.GetType() == protos.WebRTCSDP_ANSWER {
			s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
				Kind:            webrtc_internal.WebRTCOperationApplyRemoteAnswer,
				MediaSessionID:  mediaSessionID,
				RemoteAnswerSDP: msg.Sdp.GetSdp(),
			})
		}

	case *protos.ClientSignaling_IceCandidate:
		if signalingSessionID != "" && signaling.GetSessionId() != signalingSessionID {
			s.Logger.Warnw("Received ICE candidate for stale WebRTC signaling session, ignoring", "session", s.sessionID)
			return
		}
		ice := msg.IceCandidate
		if ice == nil || ice.GetCandidate() == "" {
			return
		}
		idx := uint16(ice.GetSdpMLineIndex())
		sdpMid := ice.GetSdpMid()
		usernameFragment := ice.GetUsernameFragment()
		candidate := pionwebrtc.ICECandidateInit{
			Candidate:        ice.GetCandidate(),
			SDPMid:           &sdpMid,
			SDPMLineIndex:    &idx,
			UsernameFragment: &usernameFragment,
		}
		s.enqueueWebRTCOperation(webrtc_internal.WebRTCOperation{
			Kind:               webrtc_internal.WebRTCOperationAddRemoteICECandidate,
			MediaSessionID:     mediaSessionID,
			RemoteICECandidate: candidate,
		})

	case *protos.ClientSignaling_Disconnect:
		if msg.Disconnect {
			if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
				s.Input(disc)
			}
		}
	}
}

func (s *webrtcStreamer) addRemoteICECandidate(peerConnection *pionwebrtc.PeerConnection, candidate pionwebrtc.ICECandidateInit) {
	if err := peerConnection.AddICECandidate(candidate); err != nil {
		s.Logger.Warnw("Failed to add ICE candidate (non-fatal)", "error", err)
	}
}
