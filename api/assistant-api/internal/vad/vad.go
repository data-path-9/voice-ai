// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vad

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_vad_firered "github.com/rapidaai/api/assistant-api/internal/vad/internal/firered_vad"
	internal_vad_silero "github.com/rapidaai/api/assistant-api/internal/vad/internal/silero_vad"
	internal_vad_ten "github.com/rapidaai/api/assistant-api/internal/vad/internal/ten_vad"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type VADIdentifier string

const (
	SILERO_VAD            VADIdentifier = "silero_vad"
	TEN_VAD               VADIdentifier = "ten_vad"
	FIRERED_VAD           VADIdentifier = "firered_vad"
	OptionsKeyVadProvider               = "microphone.vad.provider"
)

// GetVAD creates a VAD instance based on the provider option.
// Input audio is always 16 kHz LINEAR16 mono (platform internal format).
func GetVAD(ctx context.Context, logger commons.Logger, callback func(context.Context, ...internal_type.Packet) error, options utils.Option) (internal_type.VoiceActivityDetectorExecutor, error) {
	typ, _ := options.GetString(OptionsKeyVadProvider)
	switch VADIdentifier(typ) {
	case FIRERED_VAD:
		return internal_vad_firered.NewFireRedVAD(ctx, logger, callback, options)
	case TEN_VAD:
		return internal_vad_ten.NewTenVAD(ctx, logger, callback, options)
	case SILERO_VAD:
		return internal_vad_silero.NewSileroVAD(ctx, logger, callback, options)
	default:
		return internal_vad_silero.NewSileroVAD(ctx, logger, callback, options)
	}
}
