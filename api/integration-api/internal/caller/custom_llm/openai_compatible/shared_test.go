package internal_custom_llm_openai_compatible

import (
	"encoding/json"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func testLogger() commons.Logger {
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

func TestNewChatCompletionParams_UsesSetExtraFieldsForCompatibilityOptions(t *testing.T) {
	options := &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name": testAny(t, "llama3.1"),
				"model.parameters": testAny(t, map[string]interface{}{
					"temperature": 0.6,
					"top_k":       20,
					"chat_template_kwargs": map[string]interface{}{
						"enable_thinking": false,
					},
				}),
			},
		},
	}

	params := newChatCompletionParams(testLogger(), options, false)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "llama3.1", payload["model"])
	assert.Equal(t, float64(0.6), payload["temperature"])
	assert.Equal(t, float64(20), payload["top_k"])
	chatTemplateKwargs, ok := payload["chat_template_kwargs"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, chatTemplateKwargs["enable_thinking"])
}
