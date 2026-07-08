// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentflow

import "github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"

const (
	NodeTypeChatInput  = schema.NodeTypeChatInput
	NodeTypeAgent      = schema.NodeTypeAgent
	NodeTypeMessage    = schema.NodeTypeMessage
	NodeTypeCondition  = schema.NodeTypeCondition
	NodeTypeTransfer   = schema.NodeTypeTransfer
	NodeTypeEnd        = schema.NodeTypeEnd
	NodeTypeStickyNote = schema.NodeTypeStickyNote

	DefaultSchemaVersion = schema.DefaultSchemaVersion
)

type Definition = schema.Definition
type Node = schema.Node
type Edge = schema.Edge
type AgentTransition = schema.AgentTransition
type AgentTransitionParameter = schema.AgentTransitionParameter
type ChatInputArgument = schema.ChatInputArgument
type ConditionRule = schema.ConditionRule
