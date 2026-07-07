// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
)

type ChatInputHandler struct{}

func (handler ChatInputHandler) Type() string {
	return schema.NodeTypeChatInput
}

func (handler ChatInputHandler) Close(context.Context) error {
	return nil
}

func (handler ChatInputHandler) Execute(_ context.Context, request Request) (Result, error) {
	for _, argument := range request.Node.ChatInputArguments() {
		request.RuntimeState.SetVariableDefault(argument.Name, argument.DefaultValue)
	}
	return Result{}, nil
}
