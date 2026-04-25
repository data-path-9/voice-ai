// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package deepgram_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Deepgram Text Normalizer
// =============================================================================

// deepgramNormalizer handles Deepgram TTS text preprocessing.
// Deepgram does NOT support SSML - only plain text is accepted.
type deepgramNormalizer struct {
	logger   commons.Logger
	language string
}

// NewDeepgramNormalizer creates a Deepgram-specific text normalizer.
func NewDeepgramNormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &deepgramNormalizer{
		logger:   logger,
		language: language,
	}
}

// Normalize returns text unchanged. Deepgram does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *deepgramNormalizer) Normalize(text string) string {
	return text
}
