// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_exotel

import (
	"errors"
	"time"
)

const (
	Provider                  = "exotel"
	WebhookEvent              = "webhook"
	ChannelEventConnected     = "connected"
	ChannelEventStreamStarted = "stream_started"
	ChannelEventDTMF          = "dtmf"

	ChunkDuration         = 20 * time.Millisecond
	Linear8kHzBytesPerMs  = 16
	Linear16kHzBytesPerMs = 32
	OutputChunkSize       = Linear8kHzBytesPerMs * 20
	BridgeOutputFrameSize = Linear16kHzBytesPerMs * 20
	InputBufferThreshold  = Linear16kHzBytesPerMs * 60
)

type EventType string

const (
	EventTypeConnected EventType = "connected"
	EventTypeStart     EventType = "start"
	EventTypeMedia     EventType = "media"
	EventTypeDTMF      EventType = "dtmf"
	EventTypeStop      EventType = "stop"
	EventTypeClear     EventType = "clear"
)

type ExotelMediaEvent struct {
	Event     EventType    `json:"event"`
	StreamSid string       `json:"stream_sid"`
	Media     *ExotelMedia `json:"media,omitempty"`
}

type ExotelMedia struct {
	Payload string `json:"payload"`
}

type ExotelOutboundMessage struct {
	Event    EventType            `json:"event"`
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

var (
	ErrAudioProcessorInitFailed       = errors.New("failed to initialize Exotel audio processor")
	ErrResamplerCreateFailed          = errors.New("failed to create resampler")
	ErrProviderAudioConversionFailed  = errors.New("audio conversion to 16kHz failed")
	ErrAssistantAudioConversionFailed = errors.New("audio conversion to 8kHz failed")

	ErrCallbackFormParseFailed     = errors.New("failed to parse callback form-data")
	ErrStatusCallbackStatusMissing = errors.New("status not found in payload")
	ErrCatchAllCallSIDMissing      = errors.New("call sid not found in callback")
	ErrVaultCredentialValueMissing = errors.New("vault credential value is nil")
	ErrVaultAccountSIDMissing      = errors.New("illegal vault config accountSid is not found")
	ErrVaultClientIDMissing        = errors.New("illegal vault config client_id not found")
	ErrVaultClientSecretMissing    = errors.New("illegal vault config client_secret not found")
	ErrVaultCredentialInvalid      = errors.New("illegal vault config: credentials must be non-empty strings")
	ErrAppIDMissing                = errors.New("illegal app_id option is not found")
	ErrVaultAccountSIDInvalid      = errors.New("illegal vault config account_sid must be a non-empty string")
	ErrInboundFromMissing          = errors.New("missing or empty 'from' query parameter")
)
