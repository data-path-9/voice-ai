// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package vobiz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTrunk_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/Account/AUTH_ID/trunks", r.URL.Path)
		assert.Equal(t, "AUTH_ID", r.Header.Get("X-Auth-ID"))
		assert.Equal(t, "AUTH_TOKEN", r.Header.Get("X-Auth-Token"))

		var body CreateTrunkRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "outbound", body.TrunkDirection)
		assert.Equal(t, "udp", body.Transport)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"trunk_id":        "abc123",
			"trunk_domain":    "abc123.sip.vobiz.ai",
			"trunk_status":    "enabled",
			"trunk_direction": "outbound",
			"transport":       "udp",
		})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL)
	trunk, err := c.CreateTrunk(context.Background(), "AUTH_ID", "AUTH_TOKEN", CreateTrunkRequest{
		Name:           "My Trunk",
		TrunkStatus:    "enabled",
		TrunkDirection: "outbound",
		Transport:      "udp",
	})
	require.NoError(t, err)
	assert.Equal(t, "abc123", trunk.TrunkID)
	assert.Equal(t, "abc123.sip.vobiz.ai", trunk.TrunkDomain)
}

func TestCreateTrunk_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid auth token"})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL)
	_, err := c.CreateTrunk(context.Background(), "AUTH_ID", "BAD", CreateTrunkRequest{Name: "x"})
	require.Error(t, err)
	var apiErr *VobizAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "invalid auth token", apiErr.Message)
}

func TestCreateCredential_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/Account/AUTH_ID/credentials", r.URL.Path)
		assert.Equal(t, "AUTH_ID", r.Header.Get("X-Auth-ID"))
		assert.Equal(t, "AUTH_TOKEN", r.Header.Get("X-Auth-Token"))

		var body CreateCredentialRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "trunkuser", body.Username)
		assert.GreaterOrEqual(t, len(body.Password), 8) // Vobiz requires password >= 8 chars
		assert.True(t, body.Enabled)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":       "cred-1",
			"username": "trunkuser",
			"realm":    "MA_X.sip.vobiz.ai",
			"enabled":  true,
		})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL)
	cred, err := c.CreateCredential(context.Background(), "AUTH_ID", "AUTH_TOKEN", CreateCredentialRequest{
		Username: "trunkuser",
		Password: "supersecret123",
		Enabled:  true,
	})
	require.NoError(t, err)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Equal(t, "trunkuser", cred.Username)
}

func TestCreateOriginationURI_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/Account/AUTH_ID/origination-uris", r.URL.Path)
		assert.Equal(t, "AUTH_ID", r.Header.Get("X-Auth-ID"))
		assert.Equal(t, "AUTH_TOKEN", r.Header.Get("X-Auth-Token"))

		var body CreateOriginationURIRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "1.2.3.4:5090", body.URI) // bare host:port — vobiz adds the sip: scheme
		assert.Equal(t, "udp", body.Transport)
		assert.True(t, body.Enabled)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        "uri-1",
			"uri":       "1.2.3.4:5090",
			"transport": "udp",
			"enabled":   true,
		})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL)
	uri, err := c.CreateOriginationURI(context.Background(), "AUTH_ID", "AUTH_TOKEN", CreateOriginationURIRequest{
		URI: "1.2.3.4:5090", Transport: "udp", Priority: 1, Weight: 10, Enabled: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "uri-1", uri.ID)
}

func TestAssignNumber_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		// DID is URL-encoded in the path ("+" -> "%2B").
		assert.Equal(t, "/api/v1/Account/AUTH_ID/numbers/+919262171438/assign", r.URL.Path)

		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "trunk-1", body["trunk_group_id"])

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL)
	err := c.AssignNumber(context.Background(), "AUTH_ID", "AUTH_TOKEN", "+919262171438", "trunk-1")
	require.NoError(t, err)
}

func TestAssignNumber_AlreadyAttached(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "Number already assigned to another trunk"})
	}))
	defer server.Close()

	c := NewClientWithBaseURL(server.URL)
	err := c.AssignNumber(context.Background(), "AUTH_ID", "AUTH_TOKEN", "+919262171438", "trunk-1")
	require.Error(t, err)
	var apiErr *VobizAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}
