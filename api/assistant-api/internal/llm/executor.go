// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_llm

import (
	"context"
	"errors"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_llm_agentkit "github.com/rapidaai/api/assistant-api/internal/llm/agentkit"
	internal_llm_model "github.com/rapidaai/api/assistant-api/internal/llm/model"
	internal_llm_websocket "github.com/rapidaai/api/assistant-api/internal/llm/websocket"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type options struct {
	ctx           context.Context
	logger        commons.Logger
	assistant     *internal_assistant_entity.Assistant
	communication internal_type.Communication
	configuration *protos.ConversationInitialization
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithAssistant(assistant *internal_assistant_entity.Assistant) Option {
	return func(options *options) {
		options.assistant = assistant
	}
}

func WithCommunication(communication internal_type.Communication) Option {
	return func(options *options) {
		options.communication = communication
	}
}

func WithConfiguration(configuration *protos.ConversationInitialization) Option {
	return func(options *options) {
		options.configuration = configuration
	}
}

// New creates and initializes the LLM executor implementation matching the
// assistant's provider type.
func New(opts ...Option) (internal_type.LLMExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.assistant == nil {
		return nil, errors.New("llm: assistant is required")
	}
	switch options.assistant.AssistantProvider {
	case type_enums.AGENTKIT:
		if options.assistant.AssistantProviderAgentkit == nil {
			return nil, errors.New("llm: agentkit provider configuration is required")
		}
		return internal_llm_agentkit.New(
			internal_llm_agentkit.WithContext(options.ctx),
			internal_llm_agentkit.WithLogger(options.logger),
			internal_llm_agentkit.WithCommunication(options.communication),
			internal_llm_agentkit.WithConfiguration(options.configuration),
		)
	case type_enums.WEBSOCKET:
		if options.assistant.AssistantProviderWebsocket == nil {
			return nil, errors.New("llm: websocket provider configuration is required")
		}
		return internal_llm_websocket.New(
			internal_llm_websocket.WithContext(options.ctx),
			internal_llm_websocket.WithLogger(options.logger),
			internal_llm_websocket.WithCommunication(options.communication),
			internal_llm_websocket.WithConfiguration(options.configuration),
		)
	case type_enums.MODEL:
		if options.assistant.AssistantProviderModel == nil {
			return nil, errors.New("llm: model provider configuration is required")
		}
		return internal_llm_model.New(
			internal_llm_model.WithContext(options.ctx),
			internal_llm_model.WithLogger(options.logger),
			internal_llm_model.WithCommunication(options.communication),
			internal_llm_model.WithConfiguration(options.configuration),
		)
	default:
		return nil, errors.New("illegal assistant executor")
	}
}
