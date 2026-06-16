// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package vobiz is a minimal REST client for the Vobiz call API
// (https://api.vobiz.ai). It is used by the vobiz_websocket telephony provider
// to originate outbound calls; media then flows over the WebSocket stream.
package vobiz

import "fmt"

// MakeCallRequest is the body for POST /api/v1/Account/{auth_id}/Call/.
// answer_url must return XML; for the websocket integration it returns a
// <Stream> verb pointing at our WebSocket.
type MakeCallRequest struct {
	From         string `json:"from"`
	To           string `json:"to"`
	AnswerURL    string `json:"answer_url"`
	AnswerMethod string `json:"answer_method,omitempty"`
	RingURL      string `json:"ring_url,omitempty"`
	RingMethod   string `json:"ring_method,omitempty"`
	HangupURL    string `json:"hangup_url,omitempty"`
	HangupMethod string `json:"hangup_method,omitempty"`
	CallerName   string `json:"caller_name,omitempty"`
}

// CallResponse is the response of a fired outbound call. RequestUUID is the
// call identifier (equivalent to call_uuid) used to correlate callbacks.
type CallResponse struct {
	APIID       string `json:"api_id"`
	Message     string `json:"message"`
	RequestUUID string `json:"request_uuid"`
}

// VobizAPIError is returned for non-2xx Vobiz API responses. Message is a
// best-effort human-readable message extracted from the response body.
type VobizAPIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *VobizAPIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("vobiz api error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("vobiz api error (status %d): %s", e.StatusCode, e.Body)
}
