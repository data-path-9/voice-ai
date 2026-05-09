// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_generate_content

import (
	"encoding/json"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestBuildHistory_UserMessage(t *testing.T) {
	allMessages := []*protos.Message{
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{
					Content: "Hello, how are you?",
				},
			},
		},
	}

	instruction, history, lastPart := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, "user", instruction.Role)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, "Hello, how are you?", lastPart.Text)
}

func TestBuildHistory_AssistantMessage_WithContent(t *testing.T) {
	allMessages := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"I'm doing well", "How can I help?"},
				},
			},
		},
	}

	instruction, history, _ := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, "model", instruction.Role)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, 2, len(instruction.Parts))
	assert.Equal(t, "I'm doing well", instruction.Parts[0].Text)
	assert.Equal(t, "How can I help?", instruction.Parts[1].Text)
}

func TestBuildHistory_AssistantMessage_WithToolCall(t *testing.T) {
	toolArgs := map[string]interface{}{
		"query": "weather in NYC",
	}
	argsJSON, _ := json.Marshal(toolArgs)

	allMessages := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Let me check the weather for you"},
					ToolCalls: []*protos.ToolCall{
						{
							Id: "call_123",
							Function: &protos.FunctionCall{
								Name:      "get_weather",
								Arguments: string(argsJSON),
							},
						},
					},
				},
			},
		},
	}

	instruction, history, _ := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, "model", instruction.Role)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, 2, len(instruction.Parts))
	assert.Equal(t, "Let me check the weather for you", instruction.Parts[0].Text)
	assert.NotNil(t, instruction.Parts[1].FunctionCall)
	assert.Equal(t, "call_123", instruction.Parts[1].FunctionCall.ID)
	assert.Equal(t, "get_weather", instruction.Parts[1].FunctionCall.Name)
	assert.Equal(t, "weather in NYC", instruction.Parts[1].FunctionCall.Args["query"])
}

func TestBuildHistory_SystemMessage(t *testing.T) {
	allMessages := []*protos.Message{
		{
			Role: "system",
			Message: &protos.Message_System{
				System: &protos.SystemMessage{
					Content: "You are a helpful assistant",
				},
			},
		},
	}

	instruction, history, _ := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, "", instruction.Role)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, "You are a helpful assistant", instruction.Parts[0].Text)
}

func TestBuildHistory_ToolMessage(t *testing.T) {
	toolResult := map[string]interface{}{
		"temperature": 72,
		"condition":   "sunny",
	}
	resultJSON, _ := json.Marshal(toolResult)

	allMessages := []*protos.Message{
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{
						{
							Name:    "get_weather",
							Id:      "call_123",
							Content: string(resultJSON),
						},
					},
				},
			},
		},
	}

	instruction, history, _ := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, "user", instruction.Role)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, 1, len(instruction.Parts))
	assert.NotNil(t, instruction.Parts[0].FunctionResponse)
	assert.Equal(t, "get_weather", instruction.Parts[0].FunctionResponse.Name)
	assert.Equal(t, "call_123", instruction.Parts[0].FunctionResponse.ID)
	assert.Equal(t, float64(72), instruction.Parts[0].FunctionResponse.Response["temperature"])
	assert.Equal(t, "sunny", instruction.Parts[0].FunctionResponse.Response["condition"])
}

func TestBuildHistory_MixedMessages(t *testing.T) {
	allMessages := []*protos.Message{
		{
			Role: "system",
			Message: &protos.Message_System{
				System: &protos.SystemMessage{
					Content: "You are a helpful assistant",
				},
			},
		},
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{
					Content: "What's the weather?",
				},
			},
		},
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"Let me check"},
				},
			},
		},
	}

	instruction, history, lastPart := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, "You are a helpful assistant", instruction.Parts[0].Text)
	assert.Equal(t, 2, len(history))
	assert.Equal(t, "user", history[0].Role)
	assert.Equal(t, "model", history[1].Role)
	assert.Equal(t, "Let me check", lastPart.Text)
}

func TestBuildHistory_EmptyMessages(t *testing.T) {
	instruction, history, lastPart := buildHistory(newTestLogger(), []*protos.Message{})
	assert.Nil(t, instruction)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, "", lastPart.Text)
}

func TestBuildHistory_ModelRole(t *testing.T) {
	allMessages := []*protos.Message{
		{
			Role: "model",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"This is a model response"},
				},
			},
		},
	}

	instruction, history, _ := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, "model", instruction.Role)
	assert.Equal(t, "This is a model response", instruction.Parts[0].Text)
}

func TestBuildHistory_InvalidToolJSONHandling(t *testing.T) {
	allMessages := []*protos.Message{
		{
			Role: "tool",
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{
						{
							Name:    "operation1",
							Id:      "call_1",
							Content: "invalid json {{{",
						},
					},
				},
			},
		},
	}

	instruction, history, _ := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, 0, len(history))
	assert.Equal(t, "operation1", instruction.Parts[0].FunctionResponse.Name)
	assert.Equal(t, 0, len(instruction.Parts[0].FunctionResponse.Response))
}

func TestBuildHistory_InvalidToolCallJSONHandling(t *testing.T) {
	allMessages := []*protos.Message{
		{
			Role: "assistant",
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					ToolCalls: []*protos.ToolCall{
						{
							Id: "call_123",
							Function: &protos.FunctionCall{
								Name:      "get_weather",
								Arguments: "invalid json {{{",
							},
						},
					},
				},
			},
		},
	}

	instruction, history, _ := buildHistory(newTestLogger(), allMessages)
	assert.NotNil(t, instruction)
	assert.Equal(t, 0, len(history))
	assert.NotNil(t, instruction.Parts[0].FunctionCall)
	assert.Equal(t, 0, len(instruction.Parts[0].FunctionCall.Args))
}

func mustAnyValue(t *testing.T, input interface{}) *anypb.Any {
	t.Helper()
	v, err := structpb.NewValue(input)
	require.NoError(t, err)
	a, err := anypb.New(v)
	require.NoError(t, err)
	return a
}

func TestBuildContentConfig_AcceptsVertexKeys(t *testing.T) {
	opts := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name":                  mustAnyValue(t, "gemini-2.0-flash-001"),
				"model.max_completion_tokens": mustAnyValue(t, float64(1234)),
				"model.stop":                  mustAnyValue(t, "END,STOP"),
			},
		},
	}

	model, config := buildContentConfig(opts)
	assert.Equal(t, "gemini-2.0-flash-001", model)
	assert.Equal(t, int32(1234), config.MaxOutputTokens)
	assert.Equal(t, []string{"END", "STOP"}, config.StopSequences)
}

func TestBuildContentConfig_MapsThinking(t *testing.T) {
	opts := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name": mustAnyValue(t, "gemini-2.0-flash-001"),
				"model.thinking": mustAnyValue(t, map[string]interface{}{
					"include_thoughts": true,
					"thinking_budget":  float64(1200),
				}),
			},
		},
	}

	model, config := buildContentConfig(opts)
	assert.Equal(t, "gemini-2.0-flash-001", model)
	require.NotNil(t, config.ThinkingConfig)
	assert.True(t, config.ThinkingConfig.IncludeThoughts)
	require.NotNil(t, config.ThinkingConfig.ThinkingBudget)
	assert.Equal(t, int32(1200), *config.ThinkingConfig.ThinkingBudget)
}
