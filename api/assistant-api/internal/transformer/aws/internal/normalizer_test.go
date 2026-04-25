// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package aws_internal

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Test Setup Helpers
// =============================================================================

func newTestLogger() commons.Logger {
	l, _ := commons.NewApplicationLogger()
	return l
}

func newTestNormalizer(t *testing.T, opts utils.Option) *awsNormalizer {
	t.Helper()
	logger := newTestLogger()
	normalizer := NewAWSNormalizer(logger, opts)
	an, ok := normalizer.(*awsNormalizer)
	require.True(t, ok, "expected *awsNormalizer type")
	return an
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewAWSNormalizer(t *testing.T) {
	tests := []struct {
		name    string
		opts    utils.Option
		hasConj bool
	}{
		{
			name:    "default options",
			opts:    utils.Option{},
			hasConj: false,
		},
		{
			name: "with conjunction boundaries",
			opts: utils.Option{
				"speaker.conjunction.boundaries": "and<|||>but<|||>or",
				"speaker.conjunction.break":      uint64(300),
			},
			hasConj: true,
		},
		{
			name: "with conjunction boundaries but no break duration",
			opts: utils.Option{
				"speaker.conjunction.boundaries": "and<|||>but",
			},
			hasConj: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			an := newTestNormalizer(t, tt.opts)
			assert.NotNil(t, an.logger)
			assert.NotNil(t, an.config)
			if tt.hasConj {
				assert.NotNil(t, an.conjunctionPattern)
			} else {
				assert.Nil(t, an.conjunctionPattern)
			}
		})
	}
}

// =============================================================================
// Normalize Tests
// =============================================================================

func TestNormalize_EmptyString(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.Normalize("")
	assert.Equal(t, "", result)
}

func TestNormalize_XMLEscaping(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "double quote",
			input:    `She said "hello"`,
			expected: `She said &quot;hello&quot;`,
		},
		{
			name:     "apostrophe",
			input:    "it's fine",
			expected: "it&apos;s fine",
		},
		{
			name:     "all five entities",
			input:    `"a" < b & c > 'd'`,
			expected: `&quot;a&quot; &lt; b &amp; c &gt; &apos;d&apos;`,
		},
		{
			name:     "no escaping needed",
			input:    "Hello world",
			expected: "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := an.Normalize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalize_ConjunctionBreaks(t *testing.T) {
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but",
		"speaker.conjunction.break":      uint64(250),
	}
	an := newTestNormalizer(t, opts)

	result := an.Normalize("cats and dogs but not fish")
	assert.Contains(t, result, `and<break time="250ms"/>`)
	assert.Contains(t, result, `but<break time="250ms"/>`)
}

func TestNormalize_NoConjunctionBreaksWhenNotConfigured(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})

	result := an.Normalize("cats and dogs but not fish")
	assert.NotContains(t, result, "<break")
}

func TestNormalize_MarkdownIsNotStripped(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})

	input := "**bold** text"
	result := an.Normalize(input)
	assert.Contains(t, result, "**bold**")
}

// =============================================================================
// SSML Helper Tests
// =============================================================================

func TestWrapWithSSML(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.WrapWithSSML("Hello world")
	assert.Equal(t, "<speak>Hello world</speak>", result)
}

func TestAddBreak(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddBreak(500)
	assert.Equal(t, `<break time="500ms"/>`, result)
}

func TestAddBreakStrength(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddBreakStrength("strong")
	assert.Equal(t, `<break strength="strong"/>`, result)
}

func TestAddProsody(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})

	result := an.AddProsody("hello", "fast", "high", "loud")
	assert.Contains(t, result, `rate="fast"`)
	assert.Contains(t, result, `pitch="high"`)
	assert.Contains(t, result, `volume="loud"`)

	result = an.AddProsody("hello", "", "", "")
	assert.Equal(t, "hello", result)
}

func TestAddEmphasis(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddEmphasis("important", "strong")
	assert.Equal(t, `<emphasis level="strong">important</emphasis>`, result)
}

func TestSayAs(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})

	result := an.SayAs("123", "cardinal", "")
	assert.Equal(t, `<say-as interpret-as="cardinal">123</say-as>`, result)

	result = an.SayAs("2024-01-15", "date", "ymd")
	assert.Equal(t, `<say-as interpret-as="date" format="ymd">2024-01-15</say-as>`, result)
}

func TestAddAmazonEffect(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddAmazonEffect("secret", "whispered")
	assert.Equal(t, `<amazon:effect name="whispered">secret</amazon:effect>`, result)
}

func TestAddWhisper(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddWhisper("secret message")
	assert.Equal(t, `<amazon:effect name="whispered">secret message</amazon:effect>`, result)
}

func TestAddDomain(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddDomain("Breaking news today.", "news")
	assert.Equal(t, `<amazon:domain name="news">Breaking news today.</amazon:domain>`, result)
}

func TestAddPhoneme(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddPhoneme("pecan", "pIkAn", "ipa")
	assert.Equal(t, `<phoneme alphabet="ipa" ph="pIkAn">pecan</phoneme>`, result)
}

func TestAddLang(t *testing.T) {
	an := newTestNormalizer(t, utils.Option{})
	result := an.AddLang("Bonjour", "fr-FR")
	assert.Equal(t, `<lang xml:lang="fr-FR">Bonjour</lang>`, result)
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkNormalize_SimpleText(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewAWSNormalizer(logger, utils.Option{})
	text := "Hello, this is a simple text for TTS processing."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}

func BenchmarkNormalize_WithConjunctions(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	opts := utils.Option{
		"speaker.conjunction.boundaries": "and<|||>but<|||>or",
		"speaker.conjunction.break":      uint64(250),
	}
	normalizer := NewAWSNormalizer(logger, opts)
	text := "I like cats and dogs but not fish or snakes and birds"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}

func BenchmarkNormalize_XMLEscaping(b *testing.B) {
	logger, _ := commons.NewApplicationLogger()
	normalizer := NewAWSNormalizer(logger, utils.Option{})
	text := `Tom & Jerry said "hello" it's a < b > c`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(text)
	}
}
