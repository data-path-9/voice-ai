package internal_custom_llm_openai_chat_completions

import (
	"encoding/json"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
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

func TestNewChatCompletionParams_AppliesKnownAndExtraFields(t *testing.T) {
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

	params := newChatCompletionParams(testLogger(), options, false)
	body, err := json.Marshal(params)
	require.NoError(t, err)

	payload := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, "gpt-4o-mini", payload["model"])
	assert.Equal(t, float64(0.2), payload["temperature"])
	assert.Equal(t, float64(10), payload["top_k"])

	chatTemplateKwargs, ok := payload["chat_template_kwargs"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, false, chatTemplateKwargs["enable_thinking"])
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
}
