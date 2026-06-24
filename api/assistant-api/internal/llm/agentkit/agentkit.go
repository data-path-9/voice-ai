// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentkit

import (
	"context"
	"errors"
	"sync"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type agentkitExecutor struct {
	logger           commons.Logger
	transport        agentkitTransport
	stateMu          sync.RWMutex
	activeContextID  string
	requestStartedAt time.Time
	closing          bool
}

type options struct {
	ctx           context.Context
	logger        commons.Logger
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

func New(opts ...Option) (*agentkitExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.communication == nil {
		return nil, errors.New("agentkit: communication is required")
	}
	if options.configuration == nil {
		return nil, errors.New("agentkit: configuration is required")
	}
	if options.communication.Assistant() == nil {
		return nil, errors.New("agentkit: assistant is required")
	}
	if options.communication.Assistant().AssistantProviderAgentkit == nil {
		return nil, errors.New("agentkit: provider configuration is required")
	}
	executor := &agentkitExecutor{logger: options.logger}
	if err := executor.initialize(options.ctx, options.communication, options.configuration); err != nil {
		_ = executor.Close(options.ctx)
		return nil, err
	}
	return executor, nil
}

func (e *agentkitExecutor) Name() string { return "agentkit" }
