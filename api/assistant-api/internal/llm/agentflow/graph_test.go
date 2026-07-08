package internal_llm_agentflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompileDefinitionRejectsUnsupportedRuntimeNode(t *testing.T) {
	_, err := compileDefinition(map[string]interface{}{
		"schemaVersion": "2026-07-06",
		"entryNodeId":   "chat-input-1",
		"nodes": []map[string]interface{}{
			{"id": "chat-input-1", "type": NodeTypeChatInput, "label": "Chat Input"},
			{"id": "tool-1", "type": "tool", "label": "Tool"},
		},
		"edges": []map[string]interface{}{
			{"id": "edge-1", "source": "chat-input-1", "target": "tool-1"},
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "unsupported runtime node type")
}

func TestCompileDefinitionBuildsHandleRouting(t *testing.T) {
	graph, err := compileDefinition(map[string]interface{}{
		"schemaVersion": "2026-07-06",
		"entryNodeId":   "chat-input-1",
		"nodes": []map[string]interface{}{
			{"id": "chat-input-1", "type": NodeTypeChatInput, "label": "Chat Input"},
			{"id": "agent-1", "type": NodeTypeAgent, "label": "Agent"},
			{"id": "end-1", "type": NodeTypeEnd, "label": "End"},
		},
		"edges": []map[string]interface{}{
			{"id": "edge-1", "source": "chat-input-1", "target": "agent-1"},
			{"id": "edge-2", "source": "agent-1", "sourceHandle": "transition-complete", "target": "end-1"},
		},
	})

	require.NoError(t, err)
	target, ok := graph.targetByHandle("agent-1", "transition-complete")
	require.True(t, ok)
	require.Equal(t, "end-1", target)
}
