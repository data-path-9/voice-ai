// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentflow

import (
	"context"
	"errors"
	"sort"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/node"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/state"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

type executor struct {
	logger       commons.Logger
	nodeRegistry node.Registry
	orchestrator *orchestrator
}

type options struct {
	ctx              context.Context
	logger           commons.Logger
	communication    internal_type.Communication
	configuration    *protos.ConversationInitialization
	handlerFactories map[string]node.HandlerFactory
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

func withHandlerFactories(handlerFactories map[string]node.HandlerFactory) Option {
	return func(options *options) {
		options.handlerFactories = handlerFactories
	}
}

func New(opts ...Option) (*executor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if !validator.NonNil(options.ctx) {
		options.ctx = context.Background()
	}
	if !validator.NonNil(options.communication) {
		return nil, errors.New("agentflow: communication is required")
	}
	if !validator.NonNil(options.communication.Assistant()) {
		return nil, errors.New("agentflow: assistant is required")
	}

	agentflowProvider := options.communication.Assistant().AssistantProviderAgentflow
	if !validator.NonNil(agentflowProvider) {
		return nil, errors.New("agentflow: provider configuration is required")
	}

	compiledGraph, err := compileDefinition(agentflowProvider.Definition)
	if err != nil {
		return nil, err
	}

	runtimeNodes := make([]Node, 0, len(compiledGraph.nodes))
	for _, currentNode := range compiledGraph.nodes {
		runtimeNodes = append(runtimeNodes, currentNode)
	}
	sort.SliceStable(runtimeNodes, func(i, j int) bool {
		return runtimeNodes[i].ID < runtimeNodes[j].ID
	})

	nodeRegistry, err := node.NewRegistry(options.ctx, options.logger, options.communication, runtimeNodes, options.handlerFactories)
	if err != nil {
		return nil, err
	}

	runtimeState := state.NewRuntimeState(compiledGraph.definition.EntryNodeID)
	return &executor{
		logger:       options.logger,
		nodeRegistry: nodeRegistry,
		orchestrator: newOrchestrator(compiledGraph, runtimeState, nodeRegistry),
	}, nil
}

func (executor *executor) Name() string {
	return "agentflow"
}

func (executor *executor) Close(ctx context.Context) error {
	return executor.nodeRegistry.Close(ctx)
}

func (executor *executor) Execute(ctx context.Context, communication internal_type.Communication, packet internal_type.Packet) error {
	switch typedPacket := packet.(type) {
	case internal_type.UserInputPacket:
		return executor.orchestrator.HandleUserInput(ctx, communication, typedPacket)
	case internal_type.LLMToolResultPacket:
		executor.orchestrator.HandleToolResult(typedPacket)
		return nil
	case internal_type.LLMToolCallPacket:
		return nil
	default:
		if executor.logger != nil {
			executor.logger.Errorf("agentflow: unsupported packet type: %T", packet)
		}
		return nil
	}
}
