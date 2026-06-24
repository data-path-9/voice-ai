// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vad

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

func newVADForTest(
	ctx context.Context,
	logger commons.Logger,
	onPacket func(context.Context, ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.VoiceActivityDetectorExecutor, error) {
	if onPacket == nil {
		onPacket = func(context.Context, ...internal_type.Packet) error {
			return nil
		}
	}
	return New(
		WithContext(ctx),
		WithLogger(logger),
		WithOnPacket(onPacket),
		WithOptions(opts),
	)
}
