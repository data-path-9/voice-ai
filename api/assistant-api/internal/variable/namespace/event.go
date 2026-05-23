// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package namespace

import (
	"github.com/rapidaai/api/assistant-api/internal/variable"
)

// EventNamespace exposes the observe webhook event. The observe consumer
// registers this namespace and passes Event in ResolveContext.
//
// Supports:
//
//	event.type  -> the event identifier
//	event.data  -> a composed payload including assistant, conversation,
//	               messages, and analysis.* metadata
type EventNamespace struct{}

func (n *EventNamespace) Get(suffix string, src variable.Source, ctx variable.ResolveContext) (any, bool) {
	switch suffix {
	case "type":
		return ctx.Event, true
	case "data":
		return n.dataPayload(src), true
	}
	return nil, false
}

func (n *EventNamespace) Enumerate(src variable.Source, ctx variable.ResolveContext) map[string]any {
	return map[string]any{
		"type": ctx.Event,
		"data": n.dataPayload(src),
	}
}

func (n *EventNamespace) dataPayload(src variable.Source) map[string]any {
	conv := (&ConversationNamespace{}).Enumerate(src, variable.ResolveContext{})
	asst := (&AssistantNamespace{}).Enumerate(src, variable.ResolveContext{})
	analysis := (&MetadataPrefixNamespace{Prefix: "analysis."}).Enumerate(src, variable.ResolveContext{})
	return map[string]any{
		"assistant":    asst,
		"conversation": conv,
		"analysis":     analysis,
	}
}
