// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package node

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/transition"
)

type ConditionHandler struct{}

func (handler ConditionHandler) Type() string {
	return schema.NodeTypeCondition
}

func (handler ConditionHandler) Close(context.Context) error {
	return nil
}

func (handler ConditionHandler) Execute(_ context.Context, request Request) (Result, error) {
	variables := request.RuntimeState.VariablesSnapshot()
	nodeOutputs := request.RuntimeState.NodeOutputsSnapshot()
	for _, conditionRule := range request.Node.ConditionRules() {
		if transition.ConditionMatches(conditionRule, variables, nodeOutputs) {
			return Result{RouteHandles: []string{conditionRule.ID, conditionRule.SourceHandle}}, nil
		}
	}
	return Result{RouteHandles: []string{"else", "fallback", "default"}}, nil
}
