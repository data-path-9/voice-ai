// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package state

import (
	"sync"

	"github.com/rapidaai/protos"
)

type RuntimeState struct {
	mu                sync.RWMutex
	currentNodeID     string
	variables         map[string]interface{}
	nodeOutputs       map[string]map[string]interface{}
	conversationTurns []*protos.Message
}

func NewRuntimeState(entryNodeID string) *RuntimeState {
	return &RuntimeState{
		currentNodeID:     entryNodeID,
		variables:         make(map[string]interface{}),
		nodeOutputs:       make(map[string]map[string]interface{}),
		conversationTurns: make([]*protos.Message, 0),
	}
}

func (runtimeState *RuntimeState) CurrentNodeID() string {
	runtimeState.mu.RLock()
	defer runtimeState.mu.RUnlock()
	return runtimeState.currentNodeID
}

func (runtimeState *RuntimeState) SetCurrentNodeID(nodeID string) {
	runtimeState.mu.Lock()
	runtimeState.currentNodeID = nodeID
	runtimeState.mu.Unlock()
}

func (runtimeState *RuntimeState) SetVariableDefault(name string, value interface{}) {
	if name == "" {
		return
	}
	runtimeState.mu.Lock()
	if _, exists := runtimeState.variables[name]; !exists {
		runtimeState.variables[name] = value
	}
	runtimeState.mu.Unlock()
}

func (runtimeState *RuntimeState) VariablesSnapshot() map[string]interface{} {
	runtimeState.mu.RLock()
	defer runtimeState.mu.RUnlock()

	variables := make(map[string]interface{}, len(runtimeState.variables))
	for key, value := range runtimeState.variables {
		variables[key] = value
	}
	return variables
}

func (runtimeState *RuntimeState) NodeOutputsSnapshot() map[string]map[string]interface{} {
	runtimeState.mu.RLock()
	defer runtimeState.mu.RUnlock()

	nodeOutputs := make(map[string]map[string]interface{}, len(runtimeState.nodeOutputs))
	for nodeID, values := range runtimeState.nodeOutputs {
		nodeOutputs[nodeID] = make(map[string]interface{}, len(values))
		for key, value := range values {
			nodeOutputs[nodeID][key] = value
		}
	}
	return nodeOutputs
}

func (runtimeState *RuntimeState) ConversationTurnsSnapshot() []*protos.Message {
	runtimeState.mu.RLock()
	defer runtimeState.mu.RUnlock()
	return append([]*protos.Message(nil), runtimeState.conversationTurns...)
}

func (runtimeState *RuntimeState) AppendUserTurn(text string) {
	if text == "" {
		return
	}
	runtimeState.mu.Lock()
	runtimeState.conversationTurns = append(runtimeState.conversationTurns, &protos.Message{
		Role:    "user",
		Message: &protos.Message_User{User: &protos.UserMessage{Content: text}},
	})
	runtimeState.mu.Unlock()
}

func (runtimeState *RuntimeState) AppendAssistantTurn(text string) {
	if text == "" {
		return
	}
	runtimeState.mu.Lock()
	runtimeState.conversationTurns = append(runtimeState.conversationTurns, &protos.Message{
		Role:    "assistant",
		Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{text}}},
	})
	runtimeState.mu.Unlock()
}

func (runtimeState *RuntimeState) RecordNodeOutputValue(nodeID, key string, value interface{}) {
	if nodeID == "" || key == "" {
		return
	}
	runtimeState.mu.Lock()
	output := runtimeState.nodeOutputs[nodeID]
	if output == nil {
		output = make(map[string]interface{})
	}
	output[key] = value
	runtimeState.nodeOutputs[nodeID] = output
	runtimeState.mu.Unlock()
}

func (runtimeState *RuntimeState) RecordToolResult(name string, result map[string]string) {
	if name == "" {
		return
	}
	output := make(map[string]interface{}, len(result))
	for key, value := range result {
		output[key] = value
	}
	runtimeState.mu.Lock()
	runtimeState.nodeOutputs[name] = output
	runtimeState.mu.Unlock()
}

func (runtimeState *RuntimeState) RecordTransitionOutput(nodeID, transitionID, transitionName string, result map[string]interface{}) {
	if nodeID == "" {
		return
	}

	transitionArguments := make(map[string]interface{}, len(result))
	for key, value := range result {
		transitionArguments[key] = value
	}

	runtimeState.mu.Lock()
	output := runtimeState.nodeOutputs[nodeID]
	if output == nil {
		output = make(map[string]interface{})
	}
	for key, value := range transitionArguments {
		output[key] = value
	}
	if transitionID != "" {
		output[transitionID] = copyTransitionArguments(transitionArguments)
	}
	if transitionName != "" {
		output[transitionName] = copyTransitionArguments(transitionArguments)
	}
	runtimeState.nodeOutputs[nodeID] = output
	runtimeState.mu.Unlock()
}

func copyTransitionArguments(values map[string]interface{}) map[string]interface{} {
	copiedValues := make(map[string]interface{}, len(values))
	for key, value := range values {
		copiedValues[key] = value
	}
	return copiedValues
}
