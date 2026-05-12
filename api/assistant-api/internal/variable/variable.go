// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

// Package variable centralizes resolution of templated variables used in
// tool argument mapping, observe webhook payloads, and agent prompt building.
//
// Two consumer shapes share one data model and namespace registry:
//   - Apply(mapping, src, ctx)  : flat lookup, tool/observe callers
//   - Expand(src, ctx)          : nested map, agent prompt builder
package variable

import "time"

// Source is the read-only interface that every namespace uses to pull
// session / assistant / conversation state. VariableSource implements it.
type Source interface {
	Assistant() *AssistantInfo
	Conversation() *ConversationInfo
	Histories() []ConversationMessageInfo
	Arguments() map[string]any
	Metadata() map[string]any
	Options() map[string]any
	Mode() string
	Now() time.Time
}

// AssistantInfo is the subset of assistant fields exposed to templates.
type AssistantInfo struct {
	ID          uint64
	VersionID   uint64
	Name        string
	Language    string
	Description string
}

// ConversationInfo is the subset of conversation fields exposed to templates.
type ConversationInfo struct {
	ID          uint64
	Identifier  string
	Source      string
	Direction   string
	CreatedDate time.Time
}

// ConversationMessageInfo is a simplified message for templated payloads.
type ConversationMessageInfo struct {
	Role    string
	Content string
}

// ResolveContext carries per-call extras that some namespaces need.
// Source carries durable state; ResolveContext carries call-time bindings
// (tool name, raw tool args, observe event).
type ResolveContext struct {
	ToolName string
	ToolArgs map[string]any
	Event    string
}

// Namespace is one prefix's worth of data. Get is used by Apply (flat lookup);
// Enumerate is used by Expand (build sub-tree).
type Namespace interface {
	Get(suffix string, src Source, ctx ResolveContext) (any, bool)
	Enumerate(src Source, ctx ResolveContext) map[string]any
}
