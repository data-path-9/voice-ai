// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

type AsteriskMediaEvent struct {
	Event            string `json:"event,omitempty"`
	Command          string `json:"command,omitempty"`
	Channel          string `json:"channel,omitempty"`
	OptimalFrameSize int    `json:"optimal_frame_size,omitempty"`
	CorrelationID    string `json:"correlation_id,omitempty"`
	RawMessage       string `json:"-"`
}

type AsteriskARIEvent struct {
	Type      string                 `json:"type"`
	Timestamp string                 `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
	Channel   *AsteriskChannel       `json:"channel,omitempty"`
	Bridge    *AsteriskBridge        `json:"bridge,omitempty"`
	Peer      *AsteriskChannel       `json:"peer,omitempty"`
	Extra     map[string]interface{} `json:"-"`
}

type AsteriskChannel struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	State       string            `json:"state"`
	Caller      *AsteriskEndpoint `json:"caller,omitempty"`
	Connected   *AsteriskEndpoint `json:"connected,omitempty"`
	Dialplan    *AsteriskDialplan `json:"dialplan,omitempty"`
	ChannelVars map[string]string `json:"channelvars,omitempty"`
}

type AsteriskEndpoint struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

type AsteriskDialplan struct {
	Context string `json:"context"`
	Exten   string `json:"exten"`
	AppData string `json:"app_data"`
}

type AsteriskBridge struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	BridgeType string   `json:"bridge_type"`
	Channels   []string `json:"channels"`
}

type AsteriskRESTRequest struct {
	Type         string              `json:"type"`
	RequestID    string              `json:"request_id"`
	Method       string              `json:"method"`
	URI          string              `json:"uri"`
	QueryStrings []map[string]string `json:"query_strings,omitempty"`
}

type AsteriskRESTResponse struct {
	Type         string `json:"type"`
	RequestID    string `json:"request_id"`
	StatusCode   int    `json:"status_code"`
	ReasonPhrase string `json:"reason_phrase"`
	MessageBody  string `json:"message_body,omitempty"`
}
