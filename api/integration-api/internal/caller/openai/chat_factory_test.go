// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_callers

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOpenAITestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestNewChat_RejectsMissingCredential(t *testing.T) {
	chat, err := NewChat(newOpenAITestLogger(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, chat)
}

func TestNewChat_RejectsUnsupportedTransport(t *testing.T) {
	chat, err := NewChat(newOpenAITestLogger(), nil, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, chat)
	assert.Contains(t, err.Error(), "unsupported openai transport option")
}

func TestNewChatStream_RejectsMissingCredential(t *testing.T) {
	tests := []struct {
		name   string
		option string
	}{
		{name: "websocket", option: TransportWebsocket},
		{name: "chat_complete", option: TransportChat},
		{name: "chat_response", option: TransportChatResp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := NewChatStream(newOpenAITestLogger(), nil, map[string]string{
				OptionTransportKey: tt.option,
			})
			require.Error(t, err)
			assert.Nil(t, stream)
		})
	}
}

func TestNewChat_RejectsWebsocketTransport(t *testing.T) {
	chat, err := NewChat(newOpenAITestLogger(), nil, map[string]string{
		OptionTransportKey: TransportWebsocket,
	})
	require.Error(t, err)
	assert.Nil(t, chat)
	assert.Contains(t, err.Error(), "unsupported openai transport option for chat")
}

func TestNewChatStream_RejectsInvalidTransport(t *testing.T) {
	stream, err := NewChatStream(newOpenAITestLogger(), nil, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, stream)
	assert.Contains(t, err.Error(), "unsupported openai transport option")
}
