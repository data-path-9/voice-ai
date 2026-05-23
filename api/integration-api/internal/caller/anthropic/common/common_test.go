// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_anthropic_common

import (
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func credentialWithValues(t *testing.T, values map[string]interface{}) *protos.Credential {
	t.Helper()
	value, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.Credential{Value: value}
}

func TestResolveAPIKey_Succeeds(t *testing.T) {
	key, err := ResolveAPIKey(credentialWithValues(t, map[string]interface{}{"key": "anthropic-test-key"}))
	require.NoError(t, err)
	assert.Equal(t, "anthropic-test-key", key)
}

func TestResolveAPIKey_RejectsInvalidCredential(t *testing.T) {
	tests := []struct {
		name       string
		credential *protos.Credential
	}{
		{name: "nil credential", credential: nil},
		{name: "nil credential value", credential: &protos.Credential{}},
		{name: "missing key", credential: credentialWithValues(t, map[string]interface{}{"other": "value"})},
		{name: "empty key", credential: credentialWithValues(t, map[string]interface{}{"key": ""})},
		{name: "non-string key", credential: credentialWithValues(t, map[string]interface{}{"key": 123})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := ResolveAPIKey(tt.credential)
			require.Error(t, err)
			assert.Empty(t, key)
		})
	}
}

func TestNewClient_RejectsInvalidCredential(t *testing.T) {
	client, err := NewClient(nil)
	require.Error(t, err)
	assert.Nil(t, client)
}

func TestUsageMetrics_MapsTokenMetrics(t *testing.T) {
	metrics := UsageMetrics(anthropic.Usage{
		InputTokens:  10,
		OutputTokens: 4,
	})

	require.Len(t, metrics, 3)
	assert.Equal(t, "4", metrics[0].GetValue())
	assert.Equal(t, "10", metrics[1].GetValue())
	assert.Equal(t, "14", metrics[2].GetValue())
}
