// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_callers

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newXAITestLogger() commons.Logger {
	logger, _ := commons.NewApplicationLogger()
	return logger
}

func TestNewChat_RejectsMissingCredential(t *testing.T) {
	chat, err := NewChat(newXAITestLogger(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, chat)
}

func TestNewChat_RejectsUnsupportedTransport(t *testing.T) {
	chat, err := NewChat(newXAITestLogger(), nil, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, chat)
	assert.Contains(t, err.Error(), "unsupported xai transport option")
}

func TestNewChatStream_RejectsMissingCredential(t *testing.T) {
	stream, err := NewChatStream(newXAITestLogger(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, stream)
}

func TestNewChatStream_RejectsUnsupportedTransport(t *testing.T) {
	stream, err := NewChatStream(newXAITestLogger(), nil, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, stream)
	assert.Contains(t, err.Error(), "unsupported xai transport option")
}
