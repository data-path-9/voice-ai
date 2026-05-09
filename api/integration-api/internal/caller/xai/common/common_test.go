// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_common

import (
	"context"
	"testing"

	internal_xai_artifacts "github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestResolveAPIKey_Succeeds(t *testing.T) {
	value, err := structpb.NewStruct(map[string]interface{}{"key": "xai-test"})
	require.NoError(t, err)

	key, err := ResolveAPIKey(&protos.Credential{Value: value})
	require.NoError(t, err)
	assert.Equal(t, "xai-test", key)
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

func TestResolveEndpoint_DefaultAndOverride(t *testing.T) {
	assert.Equal(t, DefaultGRPCEndpoint, ResolveEndpoint(nil))
	assert.Equal(t, DefaultGRPCEndpoint, ResolveEndpoint(map[string]string{}))
	assert.Equal(
		t,
		"custom.endpoint:8443",
		ResolveEndpoint(map[string]string{OptionEndpointKey: " custom.endpoint:8443 "}),
	)
}

func TestAuthContext_AddsAuthorizationMetadata(t *testing.T) {
	ctx := AuthContext(context.Background(), "abc123")
	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	values := md.Get("authorization")
	require.Len(t, values, 1)
	assert.Equal(t, "Bearer abc123", values[0])
}

func TestCompletionUsageMetrics_MapsTokenMetrics(t *testing.T) {
	metrics := CompletionUsageMetrics(&internal_xai_artifacts.SamplingUsage{
		CompletionTokens: 45,
		PromptTokens:     120,
		TotalTokens:      165,
	})

	require.Len(t, metrics, 3)
	assert.Equal(t, "45", metrics[0].GetValue())
	assert.Equal(t, "120", metrics[1].GetValue())
	assert.Equal(t, "165", metrics[2].GetValue())
}
