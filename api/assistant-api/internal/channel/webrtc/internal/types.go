// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package webrtc_internal

// Opus audio constants (WebRTC standard: 48kHz)
const (
	OpusSampleRate    = 48000
	OpusFrameDuration = 20   // milliseconds
	OpusFrameBytes    = 1920 // 960 samples * 2 bytes (20ms at 48kHz)
	OpusChannels      = 2    // Opus RTP always signals 2 encoding channels (opus/48000/2) per RFC 7587, even for mono voice
	OpusPayloadType   = 111  // Standard dynamic payload type for Opus
	OpusSDPFmtpLine   = "minptime=10;useinbandfec=1;stereo=0;sprop-stereo=0;ptime=20"
)

// Channel and buffer sizes
const (
	InputChannelSize     = 500  // Buffered channel for incoming messages (~10s of 20ms audio frames)
	OutputChannelSize    = 1500 // Buffered channel for outgoing messages (~30s of 20ms audio frames)
	RTPBufferSize        = 1500 // Max RTP packet size (MTU)
	MaxConsecutiveErrors = 50   // Max read errors before stopping
	InputBufferThreshold = 1920 // 60ms at 16kHz (32 bytes/ms * 60ms) — consistent with telephony streamers

	// OutputBufferThreshold triggers flushing accumulated 48kHz PCM into
	// Opus-encoded frames. Must be >= OpusFrameBytes so at least one full
	// 20ms frame can be encoded per flush.
	OutputBufferThreshold = OpusFrameBytes // flush every complete 20ms frame

	// OutputPaceInterval is the real-time interval between sending consecutive
	// audio frames to the WebRTC track. Matches OpusFrameDuration so that
	// TTS bursts are smoothed to playback rate rather than flooding the client.
	OutputPaceInterval    = OpusFrameDuration // milliseconds (20ms per frame)
	PendingAudioMaxFrames = 4000
)

// Config holds WebRTC configuration
type Config struct {
	ICEServers         []ICEServer
	ICETransportPolicy string // "all" or "relay"
}

// ICEServer represents a STUN/TURN server
type ICEServer struct {
	URLs       []string
	Username   string
	Credential string
}

// DefaultConfig returns default WebRTC configuration
func DefaultConfig() *Config {
	return &Config{
		ICEServers: []ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun1.l.google.com:19302"}},
		},
		ICETransportPolicy: "all",
	}
}

// ICECandidate represents an ICE candidate for signaling
type ICECandidate struct {
	Candidate        string
	SDPMid           string
	SDPMLineIndex    int
	UsernameFragment string
}
