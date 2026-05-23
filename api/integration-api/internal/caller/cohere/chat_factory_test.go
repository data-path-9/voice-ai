// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_callers

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCohereFactoryTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestNewChat_RejectsMissingCredential(t *testing.T) {
	chat, err := NewChat(newCohereFactoryTestLogger(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, chat)
}

func TestNewChat_RejectsUnsupportedTransport(t *testing.T) {
	chat, err := NewChat(newCohereFactoryTestLogger(), nil, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, chat)
	assert.Contains(t, err.Error(), "unsupported cohere transport option")
}

func TestNewChatStream_RejectsMissingCredential(t *testing.T) {
	stream, err := NewChatStream(newCohereFactoryTestLogger(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, stream)
}

func TestNewChatStream_RejectsInvalidTransport(t *testing.T) {
	stream, err := NewChatStream(newCohereFactoryTestLogger(), nil, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, stream)
	assert.Contains(t, err.Error(), "unsupported cohere transport option")
}
