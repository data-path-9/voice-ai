// Copyright (c) 2023-2025 RapidaAI
// Author: RapidaAI Team <team@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telnyx

type TelnyxWebSocketEvent struct {
	Event    string            `json:"event"`
	StreamID string            `json:"stream_id"`
	Start    *TelnyxStartEvent `json:"start,omitempty"`
	Media    *TelnyxMediaEvent `json:"media,omitempty"`
	Stop     *TelnyxStopEvent  `json:"stop,omitempty"`
}

type TelnyxStartEvent struct {
	CallControlID string            `json:"call_control_id"`
	MediaFormat   TelnyxMediaFormat `json:"media_format"`
}

type TelnyxMediaFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

type TelnyxMediaEvent struct {
	Track   string `json:"track"`
	Payload string `json:"payload"`
}

type TelnyxStopEvent struct {
	CallControlID string `json:"call_control_id"`
}

type TelnyxOutboundMessage struct {
	Event    string               `json:"event"`
	StreamID string               `json:"stream_id"`
	Media    *TelnyxOutboundMedia `json:"media,omitempty"`
}

type TelnyxOutboundMedia struct {
	Payload string `json:"payload"`
}
