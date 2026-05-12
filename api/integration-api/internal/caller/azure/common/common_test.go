// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_azure_common

import (
	"testing"

	"github.com/openai/openai-go/v3/responses"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func credentialWithValues(t *testing.T, values map[string]interface{}) *protos.Credential {
	t.Helper()
	pb, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.Credential{Value: pb}
}

func TestResolveCredential_RequiresSubscriptionKey(t *testing.T) {
	_, _, err := ResolveCredential(newTestLogger(), nil)
	require.Error(t, err)

	_, _, err = ResolveCredential(newTestLogger(), credentialWithValues(t, map[string]interface{}{}))
	require.Error(t, err)
}

func TestResolveCredential_DefaultsEndpoint(t *testing.T) {
	endpoint, key, err := ResolveCredential(newTestLogger(), credentialWithValues(t, map[string]interface{}{
		SubscriptionKeyKey: "test-subscription-key",
	}))
	require.NoError(t, err)
	assert.Equal(t, DefaultURL, endpoint)
	assert.Equal(t, "test-subscription-key", key)
}

func TestResolveCredential_UsesConfiguredEndpoint(t *testing.T) {
	endpoint, key, err := ResolveCredential(newTestLogger(), credentialWithValues(t, map[string]interface{}{
		SubscriptionKeyKey: "test-subscription-key",
		EndpointKey:        "https://example.azure.com/v1",
	}))
	require.NoError(t, err)
	assert.Equal(t, "https://example.azure.com/v1", endpoint)
	assert.Equal(t, "test-subscription-key", key)
}

func TestNewClient_RejectsInvalidCredential(t *testing.T) {
	client, err := NewClient(newTestLogger(), nil)
	require.Error(t, err)
	assert.Nil(t, client)
}

func TestResponseUsageMetrics_MapsCachedTokenMetric(t *testing.T) {
	metrics := ResponseUsageMetrics(responses.ResponseUsage{
		InputTokens:  120,
		OutputTokens: 45,
		TotalTokens:  165,
		InputTokensDetails: responses.ResponseUsageInputTokensDetails{
			CachedTokens: 77,
		},
	})

	require.Len(t, metrics, 4)
	assert.Equal(t, "cached_content_token", metrics[3].GetName())
	assert.Equal(t, "77", metrics[3].GetValue())
}
