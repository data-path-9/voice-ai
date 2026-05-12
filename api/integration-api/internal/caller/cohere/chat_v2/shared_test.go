// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_chat_v2

import (
	"testing"

	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildHistory_UserMessage(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role:    "user",
			Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hello"}},
		},
	}

	history := buildHistory(newTestLogger(), msgs)
	require.Len(t, history, 1)
	assert.Equal(t, "user", history[0].Role)
	require.NotNil(t, history[0].User)
	assert.Equal(t, "Hello", history[0].User.Content.String)
}

func TestBuildHistory_SystemMessage(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role:    "system",
			Message: &protos.Message_System{System: &protos.SystemMessage{Content: "Be helpful"}},
		},
	}

	history := buildHistory(newTestLogger(), msgs)
	require.Len(t, history, 1)
	assert.Equal(t, "system", history[0].Role)
	require.NotNil(t, history[0].System)
	assert.Equal(t, "Be helpful", history[0].System.Content.String)
}

func TestBuildHistory_AssistantWithContent(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Hello!", "How can I help?"},
				},
			},
		},
	}

	history := buildHistory(newTestLogger(), msgs)
	require.Len(t, history, 1)
	assert.Equal(t, "assistant", history[0].Role)
	require.NotNil(t, history[0].Assistant)
	assert.Equal(t, "Hello!How can I help?", history[0].Assistant.Content.String)
}

func TestBuildHistory_ToolMessage(t *testing.T) {
	msgs := []*protos.Message{
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{
						{Id: "call_1", Name: "fn", Content: `{"result":"ok"}`},
						{Id: "call_2", Name: "fn2", Content: `{"result":"done"}`},
					},
				},
			},
		},
	}

	history := buildHistory(newTestLogger(), msgs)
	require.Len(t, history, 2)
	assert.Equal(t, "tool", history[0].Role)
	assert.Equal(t, "call_1", history[0].Tool.ToolCallId)
	assert.Equal(t, "tool", history[1].Role)
	assert.Equal(t, "call_2", history[1].Tool.ToolCallId)
}

func TestBuildHistory_MixedMessages(t *testing.T) {
	msgs := []*protos.Message{
		{Role: "system", Message: &protos.Message_System{System: &protos.SystemMessage{Content: "Be brief"}}},
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hi"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"Hello"}}}},
	}

	history := buildHistory(newTestLogger(), msgs)
	require.Len(t, history, 3)
	assert.Equal(t, "system", history[0].Role)
	assert.Equal(t, "user", history[1].Role)
	assert.Equal(t, "assistant", history[2].Role)
}

func TestBuildHistory_EmptyMessages(t *testing.T) {
	history := buildHistory(newTestLogger(), []*protos.Message{})
	assert.Empty(t, history)
}
