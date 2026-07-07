// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package prompt

import (
	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type Request struct {
	ContextID        string
	Node             schema.Node
	InputText        string
	ContinuationText string
	History          []*protos.Message
	Transitions      []schema.AgentTransition
	Variables        utils.Option
}

type Result struct {
	Text           string
	TransitionID   string
	TransitionName string
	Arguments      utils.Option
	Metrics        []*protos.Metric
}
