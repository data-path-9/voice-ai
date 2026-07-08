// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
)

type EndHandler struct{}

func (handler EndHandler) Type() string {
	return schema.NodeTypeEnd
}

func (handler EndHandler) Close(context.Context) error {
	return nil
}

func (handler EndHandler) Execute(ctx context.Context, request Request) (Result, error) {
	reason := request.Node.StringConfig("reason")
	if reason == "" {
		reason = "agentflow end node reached"
	}

	err := request.Communication.OnPacket(ctx, internal_type.LLMToolCallPacket{
		ContextID: request.ContextID,
		Name:      "end_conversation",
		Action:    protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION,
		Arguments: map[string]string{"reason": reason},
	})
	return Result{Terminal: true}, err
}
