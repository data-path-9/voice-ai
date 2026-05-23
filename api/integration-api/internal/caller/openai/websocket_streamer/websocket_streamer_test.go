// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_websocket_streamer

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
	value, err := structpb.NewStruct(map[string]interface{}{"key": "sk-test"})
	require.NoError(t, err)
	return &protos.Credential{Value: value}
}

func TestNew_RejectsMissingCredential(t *testing.T) {
	caller, err := New(newTestLogger(), nil)
	require.Error(t, err)
	assert.Nil(t, caller)
}

func TestStreamer_ConnectAndCloseLifecycle(t *testing.T) {
	caller, err := New(newTestLogger(), credentialWithKey(t))
	require.NoError(t, err)

	s, ok := caller.(*streamer)
	require.True(t, ok)
	require.False(t, s.connected)
	require.False(t, s.closed)
	require.Nil(t, s.client)

	err = s.Connect(context.Background(), nil)
	require.NoError(t, err)
	require.True(t, s.connected)
	require.False(t, s.closed)
	require.NotNil(t, s.client)

	firstClient := s.client
	err = s.Connect(context.Background(), nil)
	require.NoError(t, err)
	assert.Same(t, firstClient, s.client)

	s.markDisconnected()
	assert.False(t, s.connected)
	assert.NotNil(t, s.client)

	err = s.Close(context.Background())
	require.NoError(t, err)
	assert.False(t, s.connected)
	assert.True(t, s.closed)
	assert.Nil(t, s.client)
}
