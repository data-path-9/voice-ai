// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_exotel

import (
	"time"
)

const (
	ChunkDuration         = 20 * time.Millisecond
	Linear8kHzBytesPerMs  = 16
	Linear16kHzBytesPerMs = 32
	OutputChunkSize       = Linear8kHzBytesPerMs * 20
	BridgeOutputFrameSize = Linear16kHzBytesPerMs * 20
	InputBufferThreshold  = Linear16kHzBytesPerMs * 60
)

type ExotelMediaEvent struct {
	Event     string       `json:"event"`
	StreamSid string       `json:"stream_sid"`
	Media     *ExotelMedia `json:"media,omitempty"`
}

type ExotelMedia struct {
	Payload string `json:"payload"`
}

type ExotelOutboundMessage struct {
	Event    string               `json:"event"`
	StreamID string               `json:"streamSid"`
	Media    *ExotelOutboundMedia `json:"media,omitempty"`
}

type ExotelOutboundMedia struct {
	Payload string `json:"payload"`
}

type MakeCallResponse struct {
	Call struct {
		Sid              string  `json:"Sid"`
		Status           string  `json:"Status"`
		RecordingUrl     string  `json:"RecordingUrl"`
		ConversationUuid *string `json:"ParentCallSid"`
	} `json:"Call"`
}
