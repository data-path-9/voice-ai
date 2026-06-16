// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package vobiz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rapidaai/pkg/clients/rest"
)

// DefaultBaseURL is the Vobiz API base URL.
const DefaultBaseURL = "https://api.vobiz.ai"

// Client is the minimal Vobiz call surface used by the vobiz_websocket provider.
type Client interface {
	// MakeCall originates an outbound call via the Vobiz REST API.
	MakeCall(ctx context.Context, authID, authToken string, req MakeCallRequest) (*CallResponse, error)
}

type client struct {
	rest *rest.RestClient
}

// NewClient returns a Vobiz client against the production API.
func NewClient() Client {
	return NewClientWithBaseURL(DefaultBaseURL)
}

// NewClientWithBaseURL returns a Vobiz client against a custom base URL (tests).
func NewClientWithBaseURL(baseURL string) Client {
	return &client{rest: rest.NewRestClientWithConfig(baseURL, nil, 30)}
}

// authHeaders builds the per-request account auth headers. Credentials are
// tenant-supplied (from the integration form), not app config.
func authHeaders(authID, authToken string) map[string]string {
	return map[string]string{
		"X-Auth-ID":    authID,
		"X-Auth-Token": authToken,
		"Content-Type": "application/json",
	}
}

func (c *client) MakeCall(ctx context.Context, authID, authToken string, req MakeCallRequest) (*CallResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/Account/%s/Call/", authID)
	resp, err := c.rest.Post(ctx, endpoint, req, authHeaders(authID, authToken))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(resp)
	}
	var call CallResponse
	if err := json.Unmarshal(resp.Body, &call); err != nil {
		return nil, fmt.Errorf("failed to parse vobiz call response: %w", err)
	}
	return &call, nil
}

// apiError builds a VobizAPIError, extracting a human-readable message from
// common error body shapes.
func apiError(resp *rest.APIResponse) error {
	msg := ""
	var parsed map[string]interface{}
	if json.Unmarshal(resp.Body, &parsed) == nil {
		for _, k := range []string{"message", "error", "error_message", "detail", "errors"} {
			if v, ok := parsed[k]; ok && v != nil {
				msg = fmt.Sprintf("%v", v)
				break
			}
		}
	}
	return &VobizAPIError{StatusCode: resp.StatusCode, Body: resp.ToString(), Message: msg}
}
