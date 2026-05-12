// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_chat_complete

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
	caller, err := NewChat(newTestLogger(), nil)
	require.Error(t, err)
	assert.Nil(t, caller)
}

func TestNew_AcceptsValidCredential(t *testing.T) {
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
	require.NotNil(t, s.client)
	firstClient := s.client
	require.NotNil(t, s.httpClient)

	err = s.Connect(context.Background(), nil)
	require.NoError(t, err)
	assert.Same(t, firstClient, s.client)

	err = s.Close(context.Background())
	require.NoError(t, err)
	assert.Nil(t, s.client)
	assert.Nil(t, s.httpClient)
}
