// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_type

// OutputAudioTransformer is an interface for transforming output audio data.
// It extends the Transformers interface, specifying that it transforms
// from string (processed audio representation) to []byte (raw audio data).
type TextToSpeechTransformer interface {
	Name() string

	//
	Transformers[Packet]
}
