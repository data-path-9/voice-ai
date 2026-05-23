package internal_custom_llm_anthropic_messages

import (
	"context"
	"testing"

	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() commons.Logger {
	logger, _ := commons.NewApplicationLogger()
	return logger
}

func TestCallers_ReturnNotImplementedErrors(t *testing.T) {
	credential := &protos.Credential{}

	chat, err := NewChat(testLogger(), credential)
	require.NoError(t, err)
	_, _, err = chat.ChatComplete(context.Background(), nil, &internal_callers.ChatCompletionOptions{})
	require.Error(t, err)
	_, ok := err.(internal_custom_llm_common.NotImplementedCompatibilityError)
	assert.True(t, ok)

	stream, err := NewStream(testLogger(), credential)
	require.NoError(t, err)
	assert.Equal(t, credential, stream.GetCredential())

	err = stream.Chat(
		context.Background(),
		nil,
		&internal_callers.ChatStreamCompletionOptions{Request: &protos.StreamChatInput{RequestId: "req-id"}},
		nil,
		nil,
		nil,
	)
	require.Error(t, err)
	_, ok = err.(internal_custom_llm_common.NotImplementedCompatibilityError)
	assert.True(t, ok)
}
