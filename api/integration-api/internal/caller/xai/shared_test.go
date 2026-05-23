// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_callers

import (
	"testing"

	internal_xai_artifacts "github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func testXAILogger() commons.Logger {
	logger, _ := commons.NewApplicationLogger()
	return logger
}

func testAny(t *testing.T, value interface{}) *anypb.Any {
	t.Helper()
	pbValue, err := structpb.NewValue(value)
	require.NoError(t, err)
	anyValue, err := anypb.New(pbValue)
	require.NoError(t, err)
	return anyValue
}

func TestNewCompletionRequest_MapsModelParameters(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name": testAny(t, "grok-4"),
				"model.parameters": testAny(t, map[string]interface{}{
					"temperature":         0.6,
					"top_p":               0.8,
					"stop":                []interface{}{"END"},
					"parallel_tool_calls": false,
					"response_format": map[string]interface{}{
						"type": "json_object",
					},
				}),
			},
		},
	}

	request := newCompletionRequest(testXAILogger(), nil, options)

	assert.Equal(t, "grok-4", request.GetModel())
	require.NotNil(t, request.Temperature)
	assert.Equal(t, float32(0.6), request.GetTemperature())
	require.NotNil(t, request.TopP)
	assert.Equal(t, float32(0.8), request.GetTopP())
	assert.Equal(t, []string{"END"}, request.GetStop())
	require.NotNil(t, request.ParallelToolCalls)
	assert.False(t, request.GetParallelToolCalls())
	require.NotNil(t, request.GetResponseFormat())
	assert.Equal(t, internal_xai_artifacts.FormatType_FORMAT_TYPE_JSON_OBJECT, request.GetResponseFormat().GetFormatType())
}

func TestNewCompletionRequest_DoesNotPanicOnInvalidResponseFormat(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"response_format": map[string]interface{}{
						"json_schema": map[string]interface{}{
							"name": "test",
						},
					},
				}),
			},
		},
	}

	require.NotPanics(t, func() {
		request := newCompletionRequest(testXAILogger(), nil, options)
		assert.Nil(t, request.GetResponseFormat())
	})
}

func TestBuildHistory_MapsMessages(t *testing.T) {
	messages := []*protos.Message{
		{
			Role: chatRoleUser,
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "hello"},
			},
		},
		{
			Role: chatRoleAssistant,
			Message: &protos.Message_Assistant{
				Assistant: &protos.AssistantMessage{
					Contents: []string{"hi"},
					ToolCalls: []*protos.ToolCall{
						{
							Id:   "tool-1",
							Type: "function",
							Function: &protos.FunctionCall{
								Name:      "lookup_weather",
								Arguments: `{"city":"singapore"}`,
							},
						},
					},
				},
			},
		},
		{
			Role: chatRoleTool,
			Message: &protos.Message_Tool{
				Tool: &protos.ToolMessage{
					Tools: []*protos.ToolMessage_Tool{
						{
							Id:      "tool-1",
							Content: `{"temp":"30"}`,
						},
					},
				},
			},
		},
	}

	history := buildHistory(messages)
	require.Len(t, history, 3)

	assert.Equal(t, internal_xai_artifacts.MessageRole_ROLE_USER, history[0].GetRole())
	assert.Equal(t, "hello", history[0].GetContent()[0].GetText())

	assert.Equal(t, internal_xai_artifacts.MessageRole_ROLE_ASSISTANT, history[1].GetRole())
	assert.Equal(t, "hi", history[1].GetContent()[0].GetText())
	require.Len(t, history[1].GetToolCalls(), 1)
	assert.Equal(t, "lookup_weather", history[1].GetToolCalls()[0].GetFunction().GetName())

	assert.Equal(t, internal_xai_artifacts.MessageRole_ROLE_TOOL, history[2].GetRole())
	assert.Equal(t, "tool-1", history[2].GetToolCallId())
	assert.Equal(t, `{"temp":"30"}`, history[2].GetContent()[0].GetText())
}

func TestBuildAssistantMessage_CollectsContentAndToolCalls(t *testing.T) {
	outputs := []*internal_xai_artifacts.CompletionOutput{
		{
			Index: 1,
			Message: &internal_xai_artifacts.CompletionMessage{
				Content: "second",
			},
		},
		{
			Index: 0,
			Message: &internal_xai_artifacts.CompletionMessage{
				Content: "first",
				ToolCalls: []*internal_xai_artifacts.ToolCall{
					{
						Id: "call-1",
						Tool: &internal_xai_artifacts.ToolCall_Function{
							Function: &internal_xai_artifacts.FunctionCall{
								Name:      "f1",
								Arguments: `{"x":1}`,
							},
						},
					},
				},
			},
		},
	}

	msg := buildAssistantMessage(outputs)
	assert.Equal(t, []string{"first", "second"}, msg.GetContents())
	require.Len(t, msg.GetToolCalls(), 1)
	assert.Equal(t, "call-1", msg.GetToolCalls()[0].GetId())
	assert.Equal(t, "f1", msg.GetToolCalls()[0].GetFunction().GetName())
}

func TestFinalizeStreamToolCalls_MergesChunks(t *testing.T) {
	acc := map[int64]*streamToolCallAccumulator{}
	mergeStreamToolCall(acc, 0, 0, &internal_xai_artifacts.ToolCall{
		Id: "call-",
		Tool: &internal_xai_artifacts.ToolCall_Function{
			Function: &internal_xai_artifacts.FunctionCall{
				Name:      "fn",
				Arguments: `{"a":`,
			},
		},
	})
	mergeStreamToolCall(acc, 0, 0, &internal_xai_artifacts.ToolCall{
		Id: "call-1",
		Tool: &internal_xai_artifacts.ToolCall_Function{
			Function: &internal_xai_artifacts.FunctionCall{
				Name:      "fn",
				Arguments: `1}`,
			},
		},
	})

	toolCalls := finalizeStreamToolCalls(acc)
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "call-1", toolCalls[0].GetId())
	assert.Equal(t, "fn", toolCalls[0].GetFunction().GetName())
	assert.Equal(t, `{"a":1}`, toolCalls[0].GetFunction().GetArguments())
}
