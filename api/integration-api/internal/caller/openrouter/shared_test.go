// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openrouter_callers

import (
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func testOpenRouterLogger() commons.Logger {
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

func TestNewChatRequest_MapsModelParameters(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name": testAny(t, "openai/gpt-4o-mini"),
				"model.parameters": testAny(t, map[string]interface{}{
					"temperature": 0.6,
					"top_p":       0.8,
					"metadata": map[string]interface{}{
						"source": "test",
					},
					"stop": []interface{}{"END"},
				}),
			},
		},
	}

	request := newChatRequest(testOpenRouterLogger(), options, false)

	require.NotNil(t, request.GetModel())
	assert.Equal(t, "openai/gpt-4o-mini", *request.GetModel())

	temp, ok := request.GetTemperature().GetOrZero()
	require.True(t, ok)
	assert.Equal(t, 0.6, temp)

	topP, ok := request.GetTopP().GetOrZero()
	require.True(t, ok)
	assert.Equal(t, 0.8, topP)

	assert.Equal(t, map[string]string{"source": "test"}, request.GetMetadata())

	stop, ok := request.GetStop().GetOrZero()
	require.True(t, ok)
	require.NotNil(t, stop.Str)
	assert.Equal(t, "END", *stop.Str)
}

func TestNewChatRequest_DoesNotPanicOnInvalidResponseFormat(t *testing.T) {
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
		request := newChatRequest(testOpenRouterLogger(), options, false)
		assert.Nil(t, request.GetResponseFormat())
	})
}
