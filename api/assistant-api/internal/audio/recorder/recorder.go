// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_audio_recorder

import (
	"context"

	internal_recorder "github.com/rapidaai/api/assistant-api/internal/audio/recorder/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func GetConversationRecordingExecutor(
	contextID string,
	emitPacket func(context.Context, ...internal_type.Packet) error,
) (internal_type.ConversationRecordingExecutor, error) {
	return internal_recorder.NewConversationRecordingExecutor(contextID, emitPacket)
}
