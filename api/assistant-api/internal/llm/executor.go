// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_llm

import (
	"context"
	"errors"

	internal_llm_agentkit "github.com/rapidaai/api/assistant-api/internal/llm/agentkit"
	internal_llm_model "github.com/rapidaai/api/assistant-api/internal/llm/model"
	internal_llm_websocket "github.com/rapidaai/api/assistant-api/internal/llm/websocket"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

// AssistantExecutor defines LLM runtime behavior. Implementations live under
// llm/agentkit, llm/model, llm/websocket.
type AssistantExecutor interface {
	Initialize(ctx context.Context, communication internal_type.Communication, cfg *protos.ConversationInitialization) error
	Name() string
	Execute(ctx context.Context, communication internal_type.Communication, pkt internal_type.Packet) error
	Close(ctx context.Context) error
}

// NewExecutor is the factory that returns the LLM executor implementation
// matching the assistant's provider type. Construction and Initialize are
// folded together — callers receive a fully wired executor or an error.
func NewExecutor(logger commons.Logger, ctx context.Context, communication internal_type.Communication, cfg *protos.ConversationInitialization) (AssistantExecutor, error) {
	var executor AssistantExecutor
	switch communication.Assistant().AssistantProvider {
	case type_enums.AGENTKIT:
		executor = internal_llm_agentkit.NewAgentKitAssistantExecutor(logger)
	case type_enums.WEBSOCKET:
		executor = internal_llm_websocket.NewWebsocketAssistantExecutor(logger)
	case type_enums.MODEL:
		executor = internal_llm_model.NewModelAssistantExecutor(logger)
	default:
		return nil, errors.New("illegal assistant executor")
	}
	if err := executor.Initialize(ctx, communication, cfg); err != nil {
		return nil, err
	}
	return executor, nil
}
