// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_channel_output

import internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"

// AudioAdapter abstracts provider codec/frame specifics for channel output.
// Input bytes are always internal downstream audio (typically LINEAR16 16kHz).
type AudioAdapter interface {
	FrameSize() int
	SilenceByte() byte
	ConvertOutput(audio []byte) ([]byte, error)
	MixAmbient(frame []byte, mixer internal_ambient.Mixer) []byte
}
