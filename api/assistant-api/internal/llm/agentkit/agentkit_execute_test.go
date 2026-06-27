package internal_llm_agentkit

import (
	"context"
	"fmt"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute_UserInputPacket(t *testing.T) {
	talker := newMockTalker()
	e := newTestExecutor(talker)
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "hello world",
	})

	require.NoError(t, err)

	evs := findPackets[internal_type.ObservabilityEventRecordPacket](collector.all())
	require.Len(t, evs, 1)
	assert.Equal(t, observability.LLMStarted, evs[0].Record.Event)
	assert.Equal(t, "11", evs[0].Record.Attributes["input_char_count"])

	talker.mu.Lock()
	defer talker.mu.Unlock()
	require.Len(t, talker.sendCalls, 1)
	msg := talker.sendCalls[0].GetMessage()
	require.NotNil(t, msg)
	assert.Equal(t, "hello world", msg.GetText())
}

func TestExecute_InjectMessagePacket(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.InjectMessagePacket{
		ContextID: "ctx-1",
		Text:      "static text",
	})

	require.NoError(t, err)
	assert.Empty(t, collector.all(), "InjectMessagePacket should emit no packets")
}

func TestExecute_ToolPacketsAreNoop(t *testing.T) {
	e := newTestExecutor()
	comm, collector := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMToolCallPacket{
		ContextID: "ctx-1",
		ToolID:    "tool-1",
		Name:      "lookup",
		Arguments: map[string]string{"q": "test"},
	})
	require.NoError(t, err)

	err = e.Execute(context.Background(), comm, internal_type.LLMToolResultPacket{
		ContextID: "ctx-1",
		ToolID:    "tool-1",
		Name:      "lookup",
		Result:    map[string]string{"ok": "true"},
	})
	require.NoError(t, err)
	assert.Empty(t, collector.all(), "tool packets should not emit packets")
}

func TestExecute_UnsupportedPacket(t *testing.T) {
	e := newTestExecutor()
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.EndOfSpeechPacket{ContextID: "x"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAgentkitExecuteUnsupportedPacket)
}

func TestExecute_UserInputPacket_SendError(t *testing.T) {
	talker := newMockTalker()
	talker.sendErr = fmt.Errorf("connection lost")
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{
		ContextID: "ctx-1",
		Text:      "hello",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
}

func TestExecute_LLMInterruptPacket_ClearsCurrentContext(t *testing.T) {
	e := newTestExecutor()
	e.activeContextID = "ctx-1"
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.LLMInterruptPacket{ContextID: "ctx-1"})
	require.NoError(t, err)
	assert.Equal(t, "", e.activeContextID)
}

func TestExecute_PropagatesSendError(t *testing.T) {
	talker := newMockTalker()
	talker.sendErr = fmt.Errorf("write failed")
	e := newTestExecutor(talker)
	comm, _ := newTestComm()

	err := e.Execute(context.Background(), comm, internal_type.UserInputPacket{ContextID: "ctx-1", Text: "hello"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAgentkitConnectionSend)
	assert.Contains(t, err.Error(), "write failed")
}
