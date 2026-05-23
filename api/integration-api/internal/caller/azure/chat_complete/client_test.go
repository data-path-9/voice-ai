// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	openai "github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestHTTPClient(t *testing.T, assertRequest func(*http.Request)) *http.Client {
	t.Helper()
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assertRequest(req)
			body := `{
				"id":"chatcmpl-test",
				"object":"chat.completion",
				"created":0,
				"model":"deployment-1",
				"choices":[
					{
						"index":0,
						"finish_reason":"stop",
						"message":{"role":"assistant","content":"ok"}
					}
				],
				"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}
}

func TestBuildClientOptions_UsesAzureDeploymentRouteForRootEndpoint(t *testing.T) {
	httpClient := newTestHTTPClient(t, func(req *http.Request) {
		assert.Equal(t, "/openai/deployments/deployment-1/chat/completions", req.URL.Path)
		assert.Equal(t, "api-version=2024-10-21", req.URL.RawQuery)
		assert.Equal(t, "sk-test", req.Header.Get("Api-Key"))
		assert.Empty(t, req.Header.Get("Authorization"))
	})
	client := openai.NewClient(buildClientOptions(
		"https://example.openai.azure.com",
		"sk-test",
		"2024-10-21",
		httpClient,
	)...)

	_, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModel("deployment-1"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("hello"),
		},
	})
	require.NoError(t, err)
}

func TestBuildClientOptions_UsesOpenAICompatibleRouteForV1Endpoint(t *testing.T) {
	httpClient := newTestHTTPClient(t, func(req *http.Request) {
		assert.Equal(t, "/openai/v1/chat/completions", req.URL.Path)
		assert.Empty(t, req.URL.RawQuery)
		assert.Equal(t, "sk-test", req.Header.Get("Api-Key"))
		assert.Empty(t, req.Header.Get("Authorization"))
	})
	client := openai.NewClient(buildClientOptions(
		"https://example.openai.azure.com/openai/v1",
		"sk-test",
		"2024-10-21",
		httpClient,
	)...)

	_, err := client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
		Model: openai.ChatModel("deployment-1"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("hello"),
		},
	})
	require.NoError(t, err)
}

func TestIsOpenAICompatibleEndpoint(t *testing.T) {
	assert.True(t, isOpenAICompatibleEndpoint("https://example.openai.azure.com/openai/v1"))
	assert.True(t, isOpenAICompatibleEndpoint("https://example.openai.azure.com/openai/v1/"))
	assert.False(t, isOpenAICompatibleEndpoint("https://example.openai.azure.com"))
}
