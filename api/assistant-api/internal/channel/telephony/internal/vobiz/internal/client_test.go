// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rapidaai/pkg/clients/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_UsesDefaultsAndOverrides(t *testing.T) {
	impl, ok := New(WithBaseURL("https://api.example.com"), WithTimeoutSeconds(7)).(*client)
	require.True(t, ok)

	restClient, ok := impl.http.(*rest.RestClient)
	require.True(t, ok)
	assert.Equal(t, "https://api.example.com", restClient.BaseURL)
	assert.Equal(t, 7*time.Second, restClient.HTTPClient.Timeout)
}

func TestMakeCall_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/Account/AUTH_ID/Call/", r.URL.Path)
		assert.Equal(t, "AUTH_ID", r.Header.Get("X-Auth-ID"))
		assert.Equal(t, "AUTH_TOKEN", r.Header.Get("X-Auth-Token"))

		var body MakeCallRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "+919262171438", body.From)
		assert.Equal(t, "+911234567890", body.To)
		assert.Equal(t, "https://app.rapida.ai/v1/talk/vobiz/call/42?contextId=ctx-1", body.AnswerURL)

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"api_id":       "api-1",
			"message":      "call fired",
			"request_uuid": "req-uuid-1",
		})
	}))
	defer server.Close()

	c := New(WithBaseURL(server.URL))
	resp, err := c.MakeCall(context.Background(), "AUTH_ID", "AUTH_TOKEN", MakeCallRequest{
		From:      "+919262171438",
		To:        "+911234567890",
		AnswerURL: "https://app.rapida.ai/v1/talk/vobiz/call/42?contextId=ctx-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "req-uuid-1", resp.RequestUUID)
	assert.Equal(t, "api-1", resp.APIID)
}

func TestMakeCall_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid auth token"})
	}))
	defer server.Close()

	c := New(WithBaseURL(server.URL))
	_, err := c.MakeCall(context.Background(), "AUTH_ID", "BAD", MakeCallRequest{From: "x", To: "y"})
	require.Error(t, err)
	var apiErr *VobizAPIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	assert.Equal(t, "invalid auth token", apiErr.Message)
}
