// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

type HandlerFactory func(
	ctx context.Context,
	logger commons.Logger,
	communication internal_type.Communication,
	node schema.Node,
) (Handler, error)

func NewRegistry(
	ctx context.Context,
	logger commons.Logger,
	communication internal_type.Communication,
	nodes []schema.Node,
	handlerFactoryOverrides map[string]HandlerFactory,
) (Registry, error) {
	handlerFactories := map[string]HandlerFactory{
		schema.NodeTypeChatInput: func(context.Context, commons.Logger, internal_type.Communication, schema.Node) (Handler, error) {
			return ChatInputHandler{}, nil
		},
		schema.NodeTypeAgent: func(ctx context.Context, logger commons.Logger, communication internal_type.Communication, node schema.Node) (Handler, error) {
			return NewAgentHandler(ctx, logger, communication, node)
		},
		schema.NodeTypeMessage: func(context.Context, commons.Logger, internal_type.Communication, schema.Node) (Handler, error) {
			return MessageHandler{}, nil
		},
		schema.NodeTypeCondition: func(context.Context, commons.Logger, internal_type.Communication, schema.Node) (Handler, error) {
			return ConditionHandler{}, nil
		},
		schema.NodeTypeTransfer: func(context.Context, commons.Logger, internal_type.Communication, schema.Node) (Handler, error) {
			return TransferHandler{}, nil
		},
		schema.NodeTypeEnd: func(context.Context, commons.Logger, internal_type.Communication, schema.Node) (Handler, error) {
			return EndHandler{}, nil
		},
		schema.NodeTypeStickyNote: func(context.Context, commons.Logger, internal_type.Communication, schema.Node) (Handler, error) {
			return StickyNoteHandler{}, nil
		},
	}
	for nodeType, handlerFactory := range handlerFactoryOverrides {
		handlerFactories[nodeType] = handlerFactory
	}

	registry := Registry{NodeHandlers: map[string]Handler{}, Handlers: make([]Handler, 0, len(nodes))}
	for _, currentNode := range nodes {
		handlerFactory := handlerFactories[currentNode.Type]
		if handlerFactory == nil {
			return Registry{}, fmt.Errorf("agentflow: unsupported node type %q", currentNode.Type)
		}
		handler, err := handlerFactory(ctx, logger, communication, currentNode)
		if err != nil {
			return Registry{}, err
		}
		registry.NodeHandlers[currentNode.ID] = handler
		registry.Handlers = append(registry.Handlers, handler)
	}
	return registry, nil
}

func (registry Registry) Close(ctx context.Context) error {
	var closeErrors []error
	for _, handler := range registry.Handlers {
		if err := handler.Close(ctx); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}
