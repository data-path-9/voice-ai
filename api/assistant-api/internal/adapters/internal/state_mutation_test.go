package adapter_internal

import (
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_conversation_entity "github.com/rapidaai/api/assistant-api/internal/entity/conversations"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/stretchr/testify/require"
)

func TestSetMetadataRecordsStringValuesWithoutPrintfCorruption(t *testing.T) {
	recorder := &recordingObservabilityRecorder{}
	requestor := &genericRequestor{
		assistant: &internal_assistant_entity.Assistant{
			Audited: gorm_models.Audited{Id: 10},
		},
		assistantConversation: &internal_conversation_entity.AssistantConversation{
			Audited: gorm_models.Audited{Id: 20},
		},
		metadata:              map[string]interface{}{},
		observabilityRecorder: recorder,
	}

	requestor.setMetadata(map[string]interface{}{
		"client.assistant_phone": "+agent-2324502337780973568",
		"client.context_id":      "c292369b-8c95-4d0d-bb89-58b4f2874c9b",
		"client.phone":           "+08730906095",
	})

	require.Len(t, recorder.scopes, 1)
	scope, ok := recorder.scopes[0].(observability.ConversationScope)
	require.True(t, ok)
	require.Equal(t, uint64(10), scope.AssistantScopeID())
	require.Equal(t, uint64(20), scope.ConversationScopeID())

	require.Len(t, recorder.records, 1)
	record, ok := recorder.records[0].(observability.RecordMetadata)
	require.True(t, ok)
	require.Equal(t, "+agent-2324502337780973568", metadataValueByKey(t, record, "client.assistant_phone"))
	require.Equal(t, "c292369b-8c95-4d0d-bb89-58b4f2874c9b", metadataValueByKey(t, record, "client.context_id"))
	require.Equal(t, "+08730906095", metadataValueByKey(t, record, "client.phone"))
}

func metadataValueByKey(t *testing.T, record observability.RecordMetadata, key string) string {
	t.Helper()
	for _, metadata := range record.Metadata {
		if metadata.GetKey() == key {
			return metadata.GetValue()
		}
	}
	t.Fatalf("metadata key %q not found", key)
	return ""
}
