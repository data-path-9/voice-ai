// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package cartesia_internal

import (
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// Cartesia Text Normalizer
// =============================================================================

// cartesiaNormalizer handles Cartesia TTS text preprocessing.
// Cartesia does NOT support SSML - only plain text is accepted.
type cartesiaNormalizer struct {
	logger   commons.Logger
	language string
}

// NewCartesiaNormalizer creates a Cartesia-specific text normalizer.
func NewCartesiaNormalizer(logger commons.Logger, opts utils.Option) internal_type.TextNormalizer {
	language, _ := opts.GetString("speaker.language")
	if language == "" {
		language = "en"
	}

	return &cartesiaNormalizer{
		logger:   logger,
		language: language,
	}
}

// Normalize returns text unchanged. Cartesia does NOT support SSML.
// Markdown removal and whitespace normalization are handled upstream.
func (n *cartesiaNormalizer) Normalize(text string) string {
	return text
}
