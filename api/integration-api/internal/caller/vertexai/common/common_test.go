// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_common

import (
	"testing"

	"google.golang.org/genai"

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

func TestResolveCredential_Succeeds(t *testing.T) {
	projectID, serviceAccountJSON, region, err := ResolveCredential(credentialWithValues(t, map[string]interface{}{
		ProjectIDKey:      "test-project",
		ServiceAccountKey: `{"client_email":"test@example.com","private_key":"test","token_uri":"https://oauth2.googleapis.com/token"}`,
		RegionKey:         "us-central1",
	}))
	require.NoError(t, err)
	assert.Equal(t, "test-project", projectID)
	assert.Equal(t, "us-central1", region)
	assert.NotEmpty(t, serviceAccountJSON)
}

func TestResolveCredential_RejectsInvalidCredential(t *testing.T) {
	tests := []struct {
		name       string
		credential *protos.Credential
	}{
		{name: "nil credential", credential: nil},
		{name: "nil credential value", credential: &protos.Credential{}},
		{name: "missing project id", credential: credentialWithValues(t, map[string]interface{}{
			ServiceAccountKey: `{"client_email":"test@example.com","private_key":"test","token_uri":"https://oauth2.googleapis.com/token"}`,
			RegionKey:         "us-central1",
		})},
		{name: "missing service account", credential: credentialWithValues(t, map[string]interface{}{
			ProjectIDKey: "test-project",
			RegionKey:    "us-central1",
		})},
		{name: "missing region", credential: credentialWithValues(t, map[string]interface{}{
			ProjectIDKey:      "test-project",
			ServiceAccountKey: `{"client_email":"test@example.com","private_key":"test","token_uri":"https://oauth2.googleapis.com/token"}`,
		})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectID, serviceAccountJSON, region, err := ResolveCredential(tt.credential)
			require.Error(t, err)
			assert.Empty(t, projectID)
			assert.Empty(t, serviceAccountJSON)
			assert.Empty(t, region)
		})
	}
}

func TestNewClient_RejectsInvalidCredential(t *testing.T) {
	client, err := NewClient(nil)
	require.Error(t, err)
	assert.Nil(t, client)
}

func TestUsageMetrics_MapsTokenMetrics(t *testing.T) {
	metrics := UsageMetrics(&genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     12,
		CandidatesTokenCount: 7,
		TotalTokenCount:      19,
	})

	require.GreaterOrEqual(t, len(metrics), 3)
	assert.Equal(t, "12", metrics[0].GetValue())
	assert.Equal(t, "7", metrics[1].GetValue())
	assert.Equal(t, "19", metrics[len(metrics)-1].GetValue())
}
