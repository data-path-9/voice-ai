// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package vobiz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rapidaai/pkg/clients/rest"
)

// DefaultBaseURL is the Vobiz API base URL.
const DefaultBaseURL = "https://api.vobiz.ai"

// Client is the minimal Vobiz trunk-management surface used for provisioning.
type Client interface {
	// CreateTrunk provisions a new SIP trunk and returns its id + SIP domain.
	CreateTrunk(ctx context.Context, authID, authToken string, req CreateTrunkRequest) (*Trunk, error)
	// CreateCredential creates a standalone account-level SIP credential and
	// returns its id (credential_uuid), which is then attached to a trunk via
	// the trunk's credential_uuid.
	CreateCredential(ctx context.Context, authID, authToken string, req CreateCredentialRequest) (*Credential, error)
	// DeleteTrunk removes a trunk (best-effort rollback on partial provisioning).
	DeleteTrunk(ctx context.Context, authID, authToken, trunkID string) error
	// DeleteCredential removes a standalone credential (best-effort rollback).
	DeleteCredential(ctx context.Context, authID, authToken, credentialID string) error
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

func (c *client) CreateTrunk(ctx context.Context, authID, authToken string, req CreateTrunkRequest) (*Trunk, error) {
	endpoint := fmt.Sprintf("/api/v1/Account/%s/trunks", authID)
	resp, err := c.rest.Post(ctx, endpoint, req, authHeaders(authID, authToken))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(resp)
	}
	var trunk Trunk
	if err := json.Unmarshal(resp.Body, &trunk); err != nil {
		return nil, fmt.Errorf("failed to parse vobiz trunk response: %w", err)
	}
	if trunk.TrunkID == "" || trunk.TrunkDomain == "" {
		return nil, fmt.Errorf("vobiz trunk response missing trunk_id/trunk_domain: %s", resp.ToString())
	}
	return &trunk, nil
}

func (c *client) CreateCredential(ctx context.Context, authID, authToken string, req CreateCredentialRequest) (*Credential, error) {
	endpoint := fmt.Sprintf("/api/v1/Account/%s/credentials", authID)
	resp, err := c.rest.Post(ctx, endpoint, req, authHeaders(authID, authToken))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, apiError(resp)
	}
	cred := Credential{Username: req.Username}
	if len(bytes.TrimSpace(resp.Body)) > 0 {
		if err := json.Unmarshal(resp.Body, &cred); err != nil {
			return nil, fmt.Errorf("failed to parse vobiz credential response: %w", err)
		}
	}
	// The credential id (credential_uuid) is required to attach the credential
	// to a trunk — without it the trunk would have no usable SIP auth.
	if cred.ID == "" {
		return nil, fmt.Errorf("vobiz credential response missing id: %s", resp.ToString())
	}
	return &cred, nil
}

func (c *client) DeleteTrunk(ctx context.Context, authID, authToken, trunkID string) error {
	endpoint := fmt.Sprintf("/api/v1/Account/%s/trunks/%s", authID, trunkID)
	resp, err := c.rest.Delete(ctx, endpoint, authHeaders(authID, authToken))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError(resp)
	}
	return nil
}

func (c *client) DeleteCredential(ctx context.Context, authID, authToken, credentialID string) error {
	endpoint := fmt.Sprintf("/api/v1/Account/%s/credentials/%s", authID, credentialID)
	resp, err := c.rest.Delete(ctx, endpoint, authHeaders(authID, authToken))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError(resp)
	}
	return nil
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
