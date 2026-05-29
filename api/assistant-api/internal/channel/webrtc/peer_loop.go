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

type peerEventKind int

const (
	signalEventClientMessage peerEventKind = iota + 1
	peerEventStateChanged
	peerEventICEConnectionStateChanged
)

type peerEvent struct {
	kind                  peerEventKind
	mediaSessionID        uint64
	signalClientMessage   *protos.ClientSignaling
	peerState             pionwebrtc.PeerConnectionState
	peerStateChangedAt    time.Time
	peerICEState          pionwebrtc.ICEConnectionState
	peerICEStateChangedAt time.Time
}

func (s *webrtcStreamer) runPeerEventLoop() {
	for {
		select {
		case <-s.Ctx.Done():
			return
		case event := <-s.peerEventCh:
			switch event.kind {
			case signalEventClientMessage:
				s.handleClientSignal(event.signalClientMessage)
			case peerEventStateChanged:
				s.handlePeerState(event.mediaSessionID, event.peerState, event.peerStateChangedAt)
			case peerEventICEConnectionStateChanged:
				s.handlePeerICEConnectionState(event.mediaSessionID, event.peerICEState, event.peerICEStateChangedAt)
			}
		}
	}
}

func (s *webrtcStreamer) enqueuePeerEvent(event peerEvent) {
	if s.peerEventCh == nil {
		switch event.kind {
		case signalEventClientMessage:
			s.handleClientSignal(event.signalClientMessage)
		case peerEventStateChanged:
			s.handlePeerState(event.mediaSessionID, event.peerState, event.peerStateChangedAt)
		case peerEventICEConnectionStateChanged:
			s.handlePeerICEConnectionState(event.mediaSessionID, event.peerICEState, event.peerICEStateChangedAt)
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
		s.mediaHealthState.RecordPeerConnected(peerStateChangedAt)
	case pionwebrtc.PeerConnectionStateFailed,
		pionwebrtc.PeerConnectionStateDisconnected:
		if state == pionwebrtc.PeerConnectionStateFailed {
			s.mediaHealthState.RecordPeerFailed(peerStateChangedAt)
		} else {
			s.mediaHealthState.RecordPeerDisconnected(peerStateChangedAt)
		}
	case pionwebrtc.PeerConnectionStateClosed:
		s.currentMode = protos.StreamMode_STREAM_MODE_TEXT
		s.mediaHealthState.RecordPeerClosed(peerStateChangedAt)
	}
	iceLatencyMs := peerStateChangedAt.Sub(s.mediaHealthState.ICEStartedAt).Milliseconds()
	peerConnection := s.peerConnection
	s.Mu.Unlock()

	switch state {
	case pionwebrtc.PeerConnectionStateConnected:
		s.sessionState.SetPeerConnected(true)
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
		s.Logger.Warnw("WebRTC peer failed, restarting media session", "session", s.sessionID)
		s.Input(&protos.ConversationEvent{
			Name: observe.ComponentWebRTC,
			Data: map[string]string{
				webrtc_internal.DataType:      observe.EventPeerFailed,
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataReason:    webrtc_internal.ReasonPeerFailed,
			},
			Time: timestamppb.Now(),
		})
		s.queueMediaSessionRestart(mediaSessionID, webrtc_internal.ReasonPeerFailed, peerStateChangedAt)

	case pionwebrtc.PeerConnectionStateDisconnected:
		s.Logger.Warnw("WebRTC peer disconnected, restarting media session", "session", s.sessionID)
		s.Input(&protos.ConversationEvent{
			Name: observe.ComponentWebRTC,
			Data: map[string]string{
				webrtc_internal.DataType:      observe.EventPeerDisconnected,
				webrtc_internal.DataSessionID: s.sessionID,
			},
			Time: timestamppb.Now(),
		})
		s.queueMediaSessionRestart(mediaSessionID, webrtc_internal.ReasonPeerDisconnected, peerStateChangedAt)

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

	if state == pionwebrtc.ICEConnectionStateFailed {
		s.queueMediaSessionRestart(mediaSessionID, webrtc_internal.ReasonICEFailed, iceStateChangedAt)
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
	peerConnection := s.peerConnection
	signalingSessionID := s.signalingSessionID
	s.Mu.Unlock()

	switch msg := signaling.GetMessage().(type) {
	case *protos.ClientSignaling_Sdp:
		if signalingSessionID != "" && signaling.GetSessionId() != signalingSessionID {
			s.Logger.Warnw("Received SDP for stale WebRTC signaling session, ignoring", "session", s.sessionID)
			return
		}
		if msg.Sdp.GetType() == protos.WebRTCSDP_ANSWER {
			if peerConnection == nil {
				s.Logger.Warnw("Received SDP answer but peer connection is nil, ignoring")
				return
			}
			if err := peerConnection.SetRemoteDescription(pionwebrtc.SessionDescription{
				Type: pionwebrtc.SDPTypeAnswer,
				SDP:  msg.Sdp.GetSdp(),
			}); err != nil {
				s.Logger.Errorw("Failed to set remote description", "error", err)
				return
			}

			remoteDescriptionSetAt := time.Now()
			s.Mu.Lock()
			if s.peerConnection != peerConnection {
				s.Mu.Unlock()
				return
			}
			s.mediaHealthState.RecordRemoteDescriptionSet(remoteDescriptionSetAt)
			pendingRemoteICECandidates := append([]pionwebrtc.ICECandidateInit(nil), s.signalPendingRemoteICECandidates...)
			s.signalPendingRemoteICECandidates = nil
			s.Mu.Unlock()

			for _, candidate := range pendingRemoteICECandidates {
				s.addRemoteICECandidate(peerConnection, candidate)
			}
		}

	case *protos.ClientSignaling_IceCandidate:
		if signalingSessionID != "" && signaling.GetSessionId() != signalingSessionID {
			s.Logger.Warnw("Received ICE candidate for stale WebRTC signaling session, ignoring", "session", s.sessionID)
			return
		}
		if peerConnection == nil {
			s.Logger.Warnw("Received ICE candidate but peer connection is nil, ignoring")
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

		if peerConnection.RemoteDescription() == nil {
			s.Mu.Lock()
			if s.peerConnection == peerConnection && peerConnection.RemoteDescription() == nil {
				if len(s.signalPendingRemoteICECandidates) >= webrtc_internal.PendingRemoteICECandidateLimit {
					s.Mu.Unlock()
					s.Logger.Warnw("WebRTC pending remote ICE candidate queue full, dropping candidate", "session", s.sessionID, "limit", webrtc_internal.PendingRemoteICECandidateLimit)
					return
				}
				s.signalPendingRemoteICECandidates = append(s.signalPendingRemoteICECandidates, candidate)
				s.Mu.Unlock()
				return
			}
			s.Mu.Unlock()
		}

		s.addRemoteICECandidate(peerConnection, candidate)

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
