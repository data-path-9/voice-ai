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

type StickyNoteHandler struct{}

func (handler StickyNoteHandler) Type() string {
	return schema.NodeTypeStickyNote
}

func (handler StickyNoteHandler) Close(context.Context) error {
	return nil
}

func (handler StickyNoteHandler) Execute(_ context.Context, _ Request) (Result, error) {
	return Result{}, nil
}
