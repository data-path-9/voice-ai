// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultBaseURL is the Vobiz API base URL.
const DefaultBaseURL = "https://api.vobiz.ai"

// Client is the minimal Vobiz call surface used by the vobiz_websocket provider.
type Client interface {
	// MakeCall originates an outbound call via the Vobiz REST API.
	MakeCall(ctx context.Context, authID, authToken string, req MakeCallRequest) (*CallResponse, error)
}

type client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a Vobiz client against the production API.
func NewClient() Client {
	return NewClientWithBaseURL(DefaultBaseURL)
}

// NewClientWithBaseURL returns a Vobiz client against a custom base URL (tests).
func NewClientWithBaseURL(baseURL string) Client {
	return &client{baseURL: baseURL, http: &http.Client{Timeout: 30 * time.Second}}
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

func (c *client) MakeCall(ctx context.Context, authID, authToken string, req MakeCallRequest) (*CallResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode vobiz call request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/Account/%s/Call/", c.baseURL, authID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	// Credentials are tenant-supplied (from the integration form), not app config.
	httpReq.Header.Set("X-Auth-ID", authID)
	httpReq.Header.Set("X-Auth-Token", authToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read vobiz call response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(resp.StatusCode, raw)
	}

	var call CallResponse
	if err := json.Unmarshal(raw, &call); err != nil {
		return nil, fmt.Errorf("failed to parse vobiz call response: %w", err)
	}
	return &call, nil
}

// apiError builds a VobizAPIError, extracting a human-readable message from
// common error body shapes.
func apiError(statusCode int, raw []byte) error {
	msg := ""
	var parsed map[string]interface{}
	if json.Unmarshal(raw, &parsed) == nil {
		for _, k := range []string{"message", "error", "error_message", "detail", "errors"} {
			if v, ok := parsed[k]; ok && v != nil {
				msg = fmt.Sprintf("%v", v)
				break
			}
		}
	}
	return &VobizAPIError{StatusCode: statusCode, Body: string(raw), Message: msg}
}
