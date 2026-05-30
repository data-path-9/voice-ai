// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_twilio

import "time"

type TwilioMediaEvent struct {
	Event     string       `json:"event"`
	Media     *TwilioMedia `json:"media,omitempty"`
	StreamSid string       `json:"streamSid"`
}

type TwilioMedia struct {
	Track     string `json:"track"`
	Chunk     string `json:"chunk"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"`
}

type TwilioOutboundMessage struct {
	Event    string               `json:"event"`
	StreamID string               `json:"streamSid"`
	Media    *TwilioOutboundMedia `json:"media,omitempty"`
}

type TwilioOutboundMedia struct {
	Payload string `json:"payload"`
}

const (
	ChunkDuration         = 20 * time.Millisecond
	MulawBytesPerMs       = 8
	Linear16BytesPerMs    = 32
	OutputChunkSize       = MulawBytesPerMs * 20
	BridgeOutputFrameSize = Linear16BytesPerMs * 20
	InputBufferThreshold  = Linear16BytesPerMs * 60
	MulawSilence          = 0xFF
)
