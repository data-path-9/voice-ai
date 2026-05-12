package internal_custom_llm_callers

import (
	"fmt"
	"testing"

	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	logger, _ := commons.NewApplicationLogger()
	return logger
}

func newCredential(
	t *testing.T,
	values map[string]interface{},
) *protos.Credential {
	t.Helper()
	pb, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.Credential{Value: pb}
}

func TestNewChat_DefaultTransportIsCredentialCompatibility(t *testing.T) {
	chat, err := NewChat(newTestLogger(), newCredential(t, map[string]interface{}{
		"base_url": "http://localhost:8000/v1",
	}), nil)
	require.NoError(t, err)
	assert.Equal(t, "*internal_custom_llm_openai_chat_completions.chatCaller", fmt.Sprintf("%T", chat))

	stream, err := NewChatStream(newTestLogger(), newCredential(t, map[string]interface{}{
		"base_url": "http://localhost:8000/v1",
	}), nil)
	require.NoError(t, err)
	assert.Equal(t, "*internal_custom_llm_openai_chat_completions.streamCaller", fmt.Sprintf("%T", stream))
}

func TestNewChat_RoutesByCompatibility(t *testing.T) {
	tests := []struct {
		name          string
		compatibility string
		wantChatType  string
		wantStrmType  string
	}{
		{
			name:          "openai chat completions",
			compatibility: string(internal_custom_llm_common.CompatibilityOpenAIChatCompletions),
			wantChatType:  "*internal_custom_llm_openai_chat_completions.chatCaller",
			wantStrmType:  "*internal_custom_llm_openai_chat_completions.streamCaller",
		},
		{
			name:          "openai responses",
			compatibility: string(internal_custom_llm_common.CompatibilityOpenAIResponses),
			wantChatType:  "*internal_custom_llm_openai_responses.chatCaller",
			wantStrmType:  "*internal_custom_llm_openai_responses.streamCaller",
		},
		{
			name:          "openai compatible",
			compatibility: string(internal_custom_llm_common.CompatibilityOpenAICompatible),
			wantChatType:  "*internal_custom_llm_openai_compatible.chatCaller",
			wantStrmType:  "*internal_custom_llm_openai_compatible.streamCaller",
		},
		{
			name:          "anthropic messages",
			compatibility: string(internal_custom_llm_common.CompatibilityAnthropicMessages),
			wantChatType:  "*internal_custom_llm_anthropic_messages.chatCaller",
			wantStrmType:  "*internal_custom_llm_anthropic_messages.streamCaller",
		},
		{
			name:          "gemini generate content",
			compatibility: string(internal_custom_llm_common.CompatibilityGeminiGenerateContent),
			wantChatType:  "*internal_custom_llm_gemini_generate_content.chatCaller",
			wantStrmType:  "*internal_custom_llm_gemini_generate_content.streamCaller",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			credential := newCredential(t, map[string]interface{}{
				"api_compatibility": tc.compatibility,
				"base_url":          "http://localhost:8000/v1",
			})

			chat, err := NewChat(newTestLogger(), credential, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.wantChatType, fmt.Sprintf("%T", chat))

			stream, err := NewChatStream(newTestLogger(), credential, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.wantStrmType, fmt.Sprintf("%T", stream))
		})
	}
}

func TestNewChat_ConnectionTransportOverridesCompatibility(t *testing.T) {
	credential := newCredential(t, map[string]interface{}{
		"api_compatibility": string(internal_custom_llm_common.CompatibilityAnthropicMessages),
		"base_url":          "http://localhost:8000/v1",
	})

	chat, err := NewChat(newTestLogger(), credential, map[string]string{
		OptionTransportKey: string(internal_custom_llm_common.CompatibilityOpenAIResponses),
	})
	require.NoError(t, err)
	assert.Equal(t, "*internal_custom_llm_openai_responses.chatCaller", fmt.Sprintf("%T", chat))

	stream, err := NewChatStream(newTestLogger(), credential, map[string]string{
		OptionTransportKey: string(internal_custom_llm_common.CompatibilityOpenAIResponses),
	})
	require.NoError(t, err)
	assert.Equal(t, "*internal_custom_llm_openai_responses.streamCaller", fmt.Sprintf("%T", stream))
}

func TestNewChat_RejectsUnsupportedTransport(t *testing.T) {
	credential := newCredential(t, map[string]interface{}{
		"base_url": "http://localhost:8000/v1",
	})

	chat, err := NewChat(newTestLogger(), credential, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, chat)
	assert.Contains(t, err.Error(), "unsupported custom-llm transport option")

	stream, err := NewChatStream(newTestLogger(), credential, map[string]string{
		OptionTransportKey: "invalid",
	})
	require.Error(t, err)
	assert.Nil(t, stream)
	assert.Contains(t, err.Error(), "unsupported custom-llm transport option")
}

func TestNewChat_RejectsUnsupportedCompatibility(t *testing.T) {
	credential := newCredential(t, map[string]interface{}{
		"api_compatibility": "unsupported_compatibility",
		"base_url":          "http://localhost:8000/v1",
	})

	chat, err := NewChat(newTestLogger(), credential, nil)
	require.Error(t, err)
	assert.Nil(t, chat)
	_, ok := err.(internal_custom_llm_common.UnsupportedCompatibilityError)
	assert.True(t, ok)

	stream, err := NewChatStream(newTestLogger(), credential, nil)
	require.Error(t, err)
	assert.Nil(t, stream)
	_, ok = err.(internal_custom_llm_common.UnsupportedCompatibilityError)
	assert.True(t, ok)
}
