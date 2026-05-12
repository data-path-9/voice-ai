// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_generate_content

import (
	"context"
	"testing"

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

func credentialWithKey(t *testing.T) *protos.Credential {
	t.Helper()
	value, err := structpb.NewStruct(map[string]interface{}{
		"project_id":          "test-project",
		"region":              "us-central1",
		"service_account_key": `{"client_email":"test@example.com","private_key":"test","token_uri":"https://oauth2.googleapis.com/token"}`,
	})
	require.NoError(t, err)
	return &protos.Credential{Value: value}
}

func TestNewChat_RejectsMissingCredential(t *testing.T) {
	caller, err := NewChat(newTestLogger(), nil)
	require.Error(t, err)
	assert.Nil(t, caller)
}

func TestNewChat_AcceptsValidCredential(t *testing.T) {
	caller, err := NewChat(newTestLogger(), credentialWithKey(t))
	require.NoError(t, err)
	assert.NotNil(t, caller)
}

func TestNewStream_RejectsMissingCredential(t *testing.T) {
	caller, err := NewStream(newTestLogger(), nil)
	require.Error(t, err)
	assert.Nil(t, caller)
}

func TestStream_ConnectAndCloseLifecycle(t *testing.T) {
	caller, err := NewStream(newTestLogger(), credentialWithKey(t))
	require.NoError(t, err)

	s, ok := caller.(*streamCaller)
	require.True(t, ok)

	err = s.Connect(context.Background(), nil)
	require.NoError(t, err)
	err = s.Connect(context.Background(), nil)
	require.NoError(t, err)

	err = s.Close(context.Background())
	require.NoError(t, err)
}
