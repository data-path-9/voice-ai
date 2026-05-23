// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package aws_internal

import (
	"fmt"
	"regexp"
	"strings"

	aws_internal "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// =============================================================================
// AWS Polly Text Normalizer
// =============================================================================

// awsNormalizer handles AWS Polly TTS text preprocessing.
// AWS Polly supports full SSML with Amazon-specific extensions.
type awsNormalizer struct {
	logger             commons.Logger
	config             aws_internal.NormalizerConfig
	conjunctionPattern *regexp.Regexp
}

// NewAWSNormalizer creates an AWS Polly-specific text normalizer.
func NewAWSNormalizer(logger commons.Logger, opts utils.Option) aws_internal.TextNormalizer {
	cfg := aws_internal.DefaultNormalizerConfig()

	// Parse conjunction boundaries from options
	var conjunctionPattern *regexp.Regexp
	if conjunctionBoundaries, err := opts.GetString("speaker.conjunction.boundaries"); err == nil && conjunctionBoundaries != "" {
		cfg.Conjunctions = strings.Split(conjunctionBoundaries, commons.SEPARATOR)

		// Build conjunction pattern
		escaped := make([]string, len(cfg.Conjunctions))
		for i, c := range cfg.Conjunctions {
			escaped[i] = regexp.QuoteMeta(strings.TrimSpace(c))
		}
		pattern := `(` + strings.Join(escaped, "|") + `)`
		conjunctionPattern = regexp.MustCompile(pattern)
	}

	// Parse conjunction break duration
	if conjunctionBreak, err := opts.GetUint64("speaker.conjunction.break"); err == nil {
		cfg.PauseDurationMs = conjunctionBreak
	}

	return &awsNormalizer{
		logger:             logger,
		config:             cfg,
		conjunctionPattern: conjunctionPattern,
	}
}

// Normalize applies AWS Polly-specific text transformations.
// Markdown removal and whitespace normalization are handled upstream.
func (n *awsNormalizer) Normalize(text string) string {
	if text == "" {
		return text
	}

	// Escape XML special characters for SSML safety
	text = n.escapeXML(text)

	// Insert breaks after conjunction boundaries (only if configured)
	if n.conjunctionPattern != nil && n.config.PauseDurationMs > 0 {
		text = n.insertConjunctionBreaks(text)
	}

	return text
}

// =============================================================================
// Private Helpers
// =============================================================================

// escapeXML escapes XML special characters for SSML.
func (n *awsNormalizer) escapeXML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(text)
}

// insertConjunctionBreaks inserts SSML breaks after conjunctions.
func (n *awsNormalizer) insertConjunctionBreaks(text string) string {
	breakTag := fmt.Sprintf(`<break time="%dms"/>`, n.config.PauseDurationMs)
	return n.conjunctionPattern.ReplaceAllStringFunc(text, func(match string) string {
		return match + breakTag
	})
}

// =============================================================================
// AWS Polly SSML Helpers
// =============================================================================

// WrapWithSSML wraps text in AWS Polly SSML structure.
func (n *awsNormalizer) WrapWithSSML(text string) string {
	return fmt.Sprintf(`<speak>%s</speak>`, text)
}

// AddBreak creates an AWS Polly SSML break element.
func (n *awsNormalizer) AddBreak(durationMs int) string {
	return fmt.Sprintf(`<break time="%dms"/>`, durationMs)
}

// AddBreakStrength creates a break with strength attribute.
// strength: none, x-weak, weak, medium, strong, x-strong
func (n *awsNormalizer) AddBreakStrength(strength string) string {
	return fmt.Sprintf(`<break strength="%s"/>`, strength)
}

// AddProsody wraps text with prosody controls.
// rate: x-slow, slow, medium, fast, x-fast, or percentage
// pitch: x-low, low, medium, high, x-high, or percentage
// volume: silent, x-soft, soft, medium, loud, x-loud, or dB
func (n *awsNormalizer) AddProsody(text string, rate, pitch, volume string) string {
	attrs := ""
	if rate != "" {
		attrs += fmt.Sprintf(` rate="%s"`, rate)
	}
	if pitch != "" {
		attrs += fmt.Sprintf(` pitch="%s"`, pitch)
	}
	if volume != "" {
		attrs += fmt.Sprintf(` volume="%s"`, volume)
	}
	if attrs == "" {
		return text
	}
	return fmt.Sprintf(`<prosody%s>%s</prosody>`, attrs, text)
}

// AddEmphasis wraps text with emphasis.
// level: strong, moderate, reduced
func (n *awsNormalizer) AddEmphasis(text, level string) string {
	return fmt.Sprintf(`<emphasis level="%s">%s</emphasis>`, level, text)
}

// SayAs specifies how to interpret text.
// interpretAs: characters, spell-out, cardinal, ordinal, digits, fraction, unit, date, time, telephone, address, interjection, expletive
func (n *awsNormalizer) SayAs(text, interpretAs, format string) string {
	if format != "" {
		return fmt.Sprintf(`<say-as interpret-as="%s" format="%s">%s</say-as>`, interpretAs, format, text)
	}
	return fmt.Sprintf(`<say-as interpret-as="%s">%s</say-as>`, interpretAs, text)
}

// AddAmazonEffect adds Amazon-specific voice effects.
// effect: drc (Dynamic Range Compression), whispered
func (n *awsNormalizer) AddAmazonEffect(text, effect string) string {
	return fmt.Sprintf(`<amazon:effect name="%s">%s</amazon:effect>`, effect, text)
}

// AddWhisper creates whispering effect.
func (n *awsNormalizer) AddWhisper(text string) string {
	return n.AddAmazonEffect(text, "whispered")
}

// AddDomain specifies the speaking domain for neural voices.
// domain: conversational, news, long-form
func (n *awsNormalizer) AddDomain(text, domain string) string {
	return fmt.Sprintf(`<amazon:domain name="%s">%s</amazon:domain>`, domain, text)
}

// AddPhoneme specifies pronunciation using phonetic alphabet.
// alphabet: ipa, x-sampa
func (n *awsNormalizer) AddPhoneme(text, phoneme, alphabet string) string {
	return fmt.Sprintf(`<phoneme alphabet="%s" ph="%s">%s</phoneme>`, alphabet, phoneme, text)
}

// AddLang specifies language for a section.
func (n *awsNormalizer) AddLang(text, lang string) string {
	return fmt.Sprintf(`<lang xml:lang="%s">%s</lang>`, lang, text)
}
