// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_chat_complete

import (
	"encoding/json"
	"testing"

	openai "github.com/openai/openai-go/v3"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func testAny(t *testing.T, value interface{}) *anypb.Any {
	t.Helper()
	pbValue, err := structpb.NewValue(value)
	require.NoError(t, err)
	anyValue, err := anypb.New(pbValue)
	require.NoError(t, err)
	return anyValue
}

func TestBuildChatCompletionOptions_AppliesKnownModelParametersAndIgnoresUnknownKeys(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name": testAny(t, "gpt-4o-mini"),
				"model.parameters": testAny(t, map[string]interface{}{
					"temperature": 0.2,
					"top_k":       10,
					"chat_template_kwargs": map[string]interface{}{
						"enable_thinking": false,
					},
				}),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "gpt-4o-mini", payload["model"])
	assert.Equal(t, float64(0.2), payload["temperature"])
	_, hasTopK := payload["top_k"]
	assert.False(t, hasTopK)
	_, hasChatTemplateKwargs := payload["chat_template_kwargs"]
	assert.False(t, hasChatTemplateKwargs)
}

func TestBuildChatCompletionOptions_MapsMaxCompletionTokensToMaxCompletionTokens(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"max_completion_tokens": 321,
				}),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, float64(321), payload["max_completion_tokens"])
	_, hasMaxTokens := payload["max_tokens"]
	assert.False(t, hasMaxTokens)
}

func TestBuildChatCompletionOptions_IgnoresDeprecatedDirectMaxTokens(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.max_tokens": testAny(t, float64(123)),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	_, hasMaxTokens := payload["max_tokens"]
	assert.False(t, hasMaxTokens)
	_, hasMaxCompletionTokens := payload["max_completion_tokens"]
	assert.False(t, hasMaxCompletionTokens)
}

func TestBuildChatCompletionOptions_IgnoresDeprecatedModelParametersMaxTokens(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"max_tokens": 123,
				}),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	_, hasMaxTokens := payload["max_tokens"]
	assert.False(t, hasMaxTokens)
	_, hasMaxCompletionTokens := payload["max_completion_tokens"]
	assert.False(t, hasMaxCompletionTokens)
}

func TestBuildChatCompletionOptions_TopLogprobsForcesLogprobsTrue(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"top_logprobs": 5,
					"logprobs":     false,
				}),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, float64(5), payload["top_logprobs"])
	assert.Equal(t, true, payload["logprobs"])
}

func TestBuildChatCompletionOptions_ExplicitLogprobsWithoutTopLogprobs(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"logprobs": false,
				}),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, false, payload["logprobs"])
	_, hasTopLogprobs := payload["top_logprobs"]
	assert.False(t, hasTopLogprobs)
}

func TestBuildChatCompletionOptions_AcceptsLegacyToolType(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		ToolDefinitions: []*internal_callers.ToolDefinition{
			{
				Type: "tool",
				Function: &internal_callers.FunctionDefinition{
					Name:        "weather",
					Description: "Get weather",
					Parameters: &internal_callers.FunctionParameter{
						Type:       "object",
						Properties: map[string]internal_callers.FunctionParameterProperty{},
					},
				},
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	tools, ok := payload["tools"].([]interface{})
	require.True(t, ok)
	require.Len(t, tools, 1)
}

func TestBuildChatCompletionOptions_IgnoresDirectTopLevelModelParameter(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.temperature": testAny(t, 0.2),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	_, hasTemperature := payload["temperature"]
	assert.False(t, hasTemperature)
}

func TestBuildChatCompletionOptions_OmitsToolChoiceWithoutTools(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"tool_choice": "required",
				}),
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	_, hasToolChoice := payload["tool_choice"]
	assert.False(t, hasToolChoice)
}

func TestBuildChatCompletionOptions_PreservesToolChoiceWithTools(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"tool_choice": "required",
				}),
			},
		},
		ToolDefinitions: []*internal_callers.ToolDefinition{
			{
				Type: "function",
				Function: &internal_callers.FunctionDefinition{
					Name: "weather",
					Parameters: &internal_callers.FunctionParameter{
						Type:       "object",
						Properties: map[string]internal_callers.FunctionParameterProperty{},
					},
				},
			},
		},
	}

	params := buildChatCompletionOptions(options)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "required", payload["tool_choice"])
}

func TestBuildHistory_MixedMessages(t *testing.T) {
	msgs := []*protos.Message{
		{Role: "system", Message: &protos.Message_System{System: &protos.SystemMessage{Content: "Be brief"}}},
		{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: "Hi"}}},
		{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{"Hello"}}}},
		{Role: "tool", Message: &protos.Message_Tool{Tool: &protos.ToolMessage{Tools: []*protos.ToolMessage_Tool{{Id: "call_1", Name: "weather", Content: `{"temp":72}`}}}}},
	}

	history := buildHistory(msgs)
	require.Len(t, history, 4)
	assert.NotNil(t, history[0].OfSystem)
	assert.NotNil(t, history[1].OfUser)
	assert.NotNil(t, history[2].OfAssistant)
	assert.NotNil(t, history[3].OfTool)
	require.NotNil(t, history[1].OfUser)
	assert.NotEmpty(t, history[1].OfUser.Content.OfArrayOfContentParts)
}

func TestBuildAssistantMessageFromChoices_SkipsSparseChoicesAndPreservesToolCalls(t *testing.T) {
	assistantMsg := buildAssistantMessageFromChoices([]openai.ChatCompletionChoice{
		{},
		{
			Index: 1,
			Message: openai.ChatCompletionMessage{
				Content: "second-choice",
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
					{
						ID:   "call_1",
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      "weather",
							Arguments: `{"city":"sg"}`,
						},
					},
				},
			},
		},
	})

	require.NotNil(t, assistantMsg)
	assert.Equal(t, []string{"second-choice"}, assistantMsg.GetContents())
	require.Len(t, assistantMsg.GetToolCalls(), 1)
	assert.Equal(t, "call_1", assistantMsg.GetToolCalls()[0].GetId())
	assert.Equal(t, "weather", assistantMsg.GetToolCalls()[0].GetFunction().GetName())
	assert.Equal(t, `{"city":"sg"}`, assistantMsg.GetToolCalls()[0].GetFunction().GetArguments())
}

func TestBuildUnaryAssistantMessageFromChoices_UsesFinishReasonSemantics(t *testing.T) {
	assistantMsg := buildUnaryAssistantMessageFromChoices([]openai.ChatCompletionChoice{
		{
			FinishReason: "stop",
			Message: openai.ChatCompletionMessage{
				Content: "final-answer",
			},
		},
		{
			FinishReason: "tool_calls",
			Message: openai.ChatCompletionMessage{
				Content: "ignored",
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{
					{
						ID:   "call_1",
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      "weather",
							Arguments: `{"city":"sg"}`,
						},
					},
				},
			},
		},
		{
			FinishReason: "length",
			Message: openai.ChatCompletionMessage{
				Content: "truncated",
			},
		},
	})

	require.NotNil(t, assistantMsg)
	assert.Equal(t, []string{"final-answer"}, assistantMsg.GetContents())
	require.Len(t, assistantMsg.GetToolCalls(), 1)
	assert.Equal(t, "call_1", assistantMsg.GetToolCalls()[0].GetId())
}

func TestFinalizeStreamContentsByChoiceIndex_SortsChoiceIndexes(t *testing.T) {
	content := finalizeStreamContentsByChoiceIndex(map[int64]string{
		3: "third-choice",
		1: "first-choice",
		2: "",
	})

	assert.Equal(t, []string{"first-choice", "third-choice"}, content)
}
