// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_common

import (
	"testing"

	"github.com/openai/openai-go/v3/responses"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolveAPIKey_Succeeds(t *testing.T) {
	value, err := structpb.NewStruct(map[string]interface{}{"key": "sk-test"})
	require.NoError(t, err)

	key, err := ResolveAPIKey(&protos.Credential{Value: value})
	require.NoError(t, err)
	assert.Equal(t, "sk-test", key)
}

func TestResolveAPIKey_RejectsInvalidCredential(t *testing.T) {
	tests := []struct {
		name       string
		credential *protos.Credential
	}{
		{name: "nil credential", credential: nil},
		{name: "nil credential value", credential: &protos.Credential{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ResolveAPIKey(tt.credential)
			require.Error(t, err)
			assert.Empty(t, key)
		})
	}
}

func TestResolveAPIKey_RejectsMissingOrInvalidKey(t *testing.T) {
	tests := []struct {
		name     string
		rawValue map[string]interface{}
	}{
		{name: "missing key", rawValue: map[string]interface{}{"other": "value"}},
		{name: "empty key", rawValue: map[string]interface{}{"key": ""}},
		{name: "non string key", rawValue: map[string]interface{}{"key": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := structpb.NewStruct(tt.rawValue)
			require.NoError(t, err)

			key, err := ResolveAPIKey(&protos.Credential{Value: value})
			require.Error(t, err)
			assert.Empty(t, key)
		})
	}
}

func TestResponseUsageMetrics_MapsTokenMetrics(t *testing.T) {
	metrics := ResponseUsageMetrics(responses.ResponseUsage{
		InputTokens:  120,
		OutputTokens: 45,
		TotalTokens:  165,
	})

	require.Len(t, metrics, 3)
	assert.Equal(t, "45", metrics[0].GetValue())
	assert.Equal(t, "120", metrics[1].GetValue())
	assert.Equal(t, "165", metrics[2].GetValue())
}

func TestResponseUsageMetrics_MapsCachedTokenMetric(t *testing.T) {
	metrics := ResponseUsageMetrics(responses.ResponseUsage{
		InputTokens:  120,
		OutputTokens: 45,
		TotalTokens:  165,
		InputTokensDetails: responses.ResponseUsageInputTokensDetails{
			CachedTokens: 90,
		},
	})

	require.Len(t, metrics, 4)
	assert.Equal(t, "cached_content_token", metrics[3].GetName())
	assert.Equal(t, "90", metrics[3].GetValue())
}
