// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_moderation

import (
	"context"
	"fmt"
	"time"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type Caller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func New(logger commons.Logger, credential *protos.Credential) *Caller {
	return &Caller{
		logger:     logger,
		credential: credential,
	}
}

func (mc *Caller) GetModeration(
	ctx context.Context,
	content *types.Content,
	options *internal_callers.ModerationOptions,
) (*types.Content, []*protos.Metric, error) {
	start := time.Now()
	timeMetric := &protos.Metric{
		Name:        type_enums.TIME_TAKEN.String(),
		Value:       fmt.Sprintf("%d", int64(time.Since(start))),
		Description: "Time taken to serve the llm request",
	}

	return &types.Content{
		ContentType:   "text",
		ContentFormat: "raw",
	}, []*protos.Metric{timeMetric}, nil
}
