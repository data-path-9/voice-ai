// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_anthropic_messages

import (
	"context"

	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type chatCaller struct{}

func NewChat(logger commons.Logger, credential *protos.Credential) (internal_callers.Chat, error) {
	_ = logger
	_ = credential
	return &chatCaller{}, nil
}

func (c *chatCaller) ChatComplete(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	_ = ctx
	_ = allMessages
	_ = options
	return nil, nil, internal_custom_llm_common.NotImplementedCompatibilityError{
		Compatibility: internal_custom_llm_common.CompatibilityAnthropicMessages,
		Feature:       "ChatComplete",
	}
}
