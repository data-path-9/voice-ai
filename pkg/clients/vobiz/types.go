// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package vobiz is a minimal REST client for the Vobiz trunk-management API
// (https://api.vobiz.ai). It is used by web-api to auto-provision a SIP trunk
// and its username/password credential when a user adds a "vobiz" integration.
package vobiz

import "fmt"

// CreateTrunkRequest is the body for POST /api/v1/Account/{auth_id}/trunks.
type CreateTrunkRequest struct {
	Name                 string `json:"name"`
	TrunkStatus          string `json:"trunk_status,omitempty"`    // "enabled" | "disabled"
	TrunkDirection       string `json:"trunk_direction,omitempty"` // "outbound" | "inbound" | "both"
	Transport            string `json:"transport,omitempty"`       // "udp" | "tcp"
	Secure               bool   `json:"secure,omitempty"`          // TLS/SRTP
	ConcurrentCallsLimit int    `json:"concurrent_calls_limit,omitempty"`
	CpsLimit             int    `json:"cps_limit,omitempty"`
	CredentialUUID       string `json:"credential_uuid,omitempty"`
}

// Trunk is the response for a created/retrieved trunk. The TrunkDomain is the
// auto-generated SIP host ("{trunk_id}.sip.vobiz.ai") used as the SIP URI.
type Trunk struct {
	TrunkID        string `json:"trunk_id"`
	TrunkDomain    string `json:"trunk_domain"`
	TrunkStatus    string `json:"trunk_status"`
	TrunkDirection string `json:"trunk_direction"`
	Transport      string `json:"transport"`
}

// CreateCredentialRequest is the body for the standalone, account-level
// credential endpoint: POST /api/v1/Account/{auth_id}/credentials.
// The password must be >= 8 chars and is write-only on the Vobiz side
// (never returned), so the caller must persist whatever it sent. The returned
// id (credential_uuid) is attached to a trunk via the trunk's credential_uuid.
type CreateCredentialRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

// Credential is the response for a created credential (password is not
// returned). ID is the credential_uuid used to attach it to a trunk.
type Credential struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Realm    string `json:"realm"`
	Enabled  bool   `json:"enabled"`
}

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
