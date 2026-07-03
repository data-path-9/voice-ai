package internal_llm_model

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

func TestModel_NewValidatesRequiredOptions(t *testing.T) {
	validCommunication := &testComm{
		assistant: &internal_assistant_entity.Assistant{
			AssistantProviderModel: &internal_assistant_entity.AssistantProviderModel{ModelProviderName: "openai"},
		},
	}
	validConfiguration := &protos.ConversationInitialization{}

	tests := []struct {
		name          string
		communication internal_type.Communication
		configuration *protos.ConversationInitialization
		wantErr       string
	}{
		{
			name:          "communication",
			configuration: validConfiguration,
			wantErr:       "model: communication is required",
		},
		{
			name:          "configuration",
			communication: validCommunication,
			wantErr:       "model: configuration is required",
		},
		{
			name:          "assistant",
			communication: &testComm{},
			configuration: validConfiguration,
			wantErr:       "model: assistant is required",
		},
		{
			name: "provider configuration",
			communication: &testComm{
				assistant: &internal_assistant_entity.Assistant{},
			},
			configuration: validConfiguration,
			wantErr:       "model: provider configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := New(
				WithCommunication(tt.communication),
				WithConfiguration(tt.configuration),
			)

			require.Nil(t, executor)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestModel_Name(t *testing.T) {
	e := &modelAssistantExecutor{}

	require.Equal(t, "model", e.Name())
}

func TestModel_CloseHandlesEmptyExecutor(t *testing.T) {
	e := &modelAssistantExecutor{}

	require.NoError(t, e.Close(context.Background()))
}

func TestModel_Close_ResetsAndClosesStream(t *testing.T) {
	e, _, stream, _ := newModelTestEnv(t)
	e.currentPacket = &internal_type.UserInputPacket{ContextID: "ctx-close"}
	e.history.AppendUser("u")

	require.NoError(t, e.Close(context.Background()))
	require.Nil(t, e.currentPacket)
	require.Nil(t, e.connection)
	require.Len(t, e.history.Snapshot(), 0)
	require.True(t, stream.closeSent)
}
