// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package namespace

import (
	"github.com/rapidaai/api/assistant-api/internal/variable"
)

// SessionNamespace exposes session-level fields: mode, source.
type SessionNamespace struct{}

func (n *SessionNamespace) Get(suffix string, src variable.Source, _ variable.ResolveContext) (any, bool) {
	v, ok := n.fields(src)[suffix]
	return v, ok
}

func (n *SessionNamespace) Enumerate(src variable.Source, _ variable.ResolveContext) map[string]any {
	return n.fields(src)
}

func (n *SessionNamespace) fields(src variable.Source) map[string]any {
	out := map[string]any{}
	if mode := src.Mode(); mode != "" {
		out["mode"] = mode
	}
	if c := src.Conversation(); c != nil && c.Source != "" {
		out["source"] = c.Source
	}
	return out
}
