// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_output_normalizers

import (
	"strings"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// =============================================================================
// Benchmark Mock Logger (minimal implementation for benchmarks)
// =============================================================================

// =============================================================================
// Sample Input Data for Benchmarks
// =============================================================================

var (
	shortSentence     = "Hello world"
	mediumSentence    = "The CEO meeting at 14:30 on 2024-01-15 at 123 Main St costs $500.50 with 25% discount"
	longSentence      = strings.Repeat("The CEO meeting at 14:30 on 2024-01-15 at 123 Main St costs $500.50 with API and ML. ", 10)
	currencyInput     = "Total: $1,234.56 plus $99.99 equals $1,334.55"
	dateInput         = "Events on 2024-01-15, 2024-06-30, and 2024-12-25"
	timeInput         = "Meetings at 09:00, 14:30, and 17:45"
	numberInput       = "We have 5 apples, 12 oranges, and 42 bananas"
	addressInput      = "Visit 123 Main St, 456 Park Ave, and 789 Oak Rd"
	urlInput          = "Check https://example.com, www.google.com, and api.test.org"
	symbolInput       = "Growth is 25% with ±5% variance, temperature 25℃"
	techInput         = "Using AI, ML, API, DevOps, and CI/CD for automation"
	roleInput         = "CEO, CFO, CTO, and VP discussed R&D plans"
	generalInput      = "Dr. Smith aka Johnny Jr. said etc. i.e. examples"
	mixedComplexInput = "The CEO Dr. Smith announced at 14:30 on 2024-01-15 that our API costs $500.50 with 25% growth at https://rapida.ai using AI & ML"
	unicodeHeavyInput = "℃ ℉ £ € ¥ ₩ ₿ ™ © ® ° ± × ÷ ≈ ≠ ≤ ≥ ∞ π √ ∑ ∫"
	noMatchInput      = "This is a plain sentence without any special patterns to match"
)

// =============================================================================
// Individual Normalizer Benchmarks
// =============================================================================

func benchLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func BenchmarkCurrencyNormalizer(b *testing.B) {
	normalizer := NewCurrencyNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("currency_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(currencyInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkDateNormalizer(b *testing.B) {
	normalizer := NewDateNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("date_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(dateInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkTimeNormalizer(b *testing.B) {
	normalizer := NewTimeNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("time_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(timeInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkNumberToWordNormalizer(b *testing.B) {
	normalizer := NewNumberToWordNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("number_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(numberInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkAddressNormalizer(b *testing.B) {
	normalizer := NewAddressNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("address_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(addressInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkUrlNormalizer(b *testing.B) {
	normalizer := NewUrlNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("url_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(urlInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkSymbolNormalizer(b *testing.B) {
	normalizer := NewSymbolNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("symbol_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(symbolInput)
		}
	})

	b.Run("unicode_heavy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(unicodeHeavyInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkTechAbbreviationNormalizer(b *testing.B) {
	normalizer := NewTechAbbreviationNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("tech_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(techInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkRoleAbbreviationNormalizer(b *testing.B) {
	normalizer := NewRoleAbbreviationNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("role_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(roleInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

func BenchmarkGeneralAbbreviationNormalizer(b *testing.B) {
	normalizer := NewGeneralAbbreviationNormalizer(benchLogger())

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(shortSentence)
		}
	})

	b.Run("general_input", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(generalInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(noMatchInput)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			normalizer.Normalize(longSentence)
		}
	})
}

// =============================================================================
// Normalizer Chain Benchmarks (simulating real-world pipeline)
// =============================================================================

func BenchmarkNormalizerChain(b *testing.B) {
	normalizers := []internal_type.TextNormalizer{
		NewCurrencyNormalizer(benchLogger()),
		NewDateNormalizer(benchLogger()),
		NewTimeNormalizer(benchLogger()),
		NewNumberToWordNormalizer(benchLogger()),
		NewAddressNormalizer(benchLogger()),
		NewUrlNormalizer(benchLogger()),
		NewTechAbbreviationNormalizer(benchLogger()),
		NewRoleAbbreviationNormalizer(benchLogger()),
		NewGeneralAbbreviationNormalizer(benchLogger()),
		NewSymbolNormalizer(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range normalizers {
			result = n.Normalize(result)
		}
		return result
	}

	b.Run("short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(shortSentence)
		}
	})

	b.Run("medium", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(mediumSentence)
		}
	})

	b.Run("long", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(longSentence)
		}
	})

	b.Run("mixed_complex", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(mixedComplexInput)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			applyAll(noMatchInput)
		}
	})
}

// =============================================================================
// Memory Allocation Benchmarks
// =============================================================================

func BenchmarkNormalizerAllocations(b *testing.B) {
	normalizers := map[string]internal_type.TextNormalizer{
		"currency": NewCurrencyNormalizer(benchLogger()),
		"date":     NewDateNormalizer(benchLogger()),
		"time":     NewTimeNormalizer(benchLogger()),
		"number":   NewNumberToWordNormalizer(benchLogger()),
		"address":  NewAddressNormalizer(benchLogger()),
		"url":      NewUrlNormalizer(benchLogger()),
		"symbol":   NewSymbolNormalizer(benchLogger()),
		"tech":     NewTechAbbreviationNormalizer(benchLogger()),
		"role":     NewRoleAbbreviationNormalizer(benchLogger()),
		"general":  NewGeneralAbbreviationNormalizer(benchLogger()),
	}

	for name, normalizer := range normalizers {
		b.Run(name+"_allocs", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				normalizer.Normalize(mediumSentence)
			}
		})
	}
}

// =============================================================================
// Scaling Benchmarks (input size scaling)
// =============================================================================

func BenchmarkInputSizeScaling(b *testing.B) {
	normalizer := NewSymbolNormalizer(benchLogger())

	sizes := []int{10, 100, 1000, 10000}
	baseText := "Hello 25% world & test + more = result @ place # tag "

	for _, size := range sizes {
		input := strings.Repeat(baseText, size/len(baseText)+1)[:size]
		b.Run(string(rune('0'+size/1000))+"k_chars", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				normalizer.Normalize(input)
			}
		})
	}
}

func BenchmarkChainInputSizeScaling(b *testing.B) {
	normalizers := []internal_type.TextNormalizer{
		NewCurrencyNormalizer(benchLogger()),
		NewDateNormalizer(benchLogger()),
		NewTimeNormalizer(benchLogger()),
		NewSymbolNormalizer(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range normalizers {
			result = n.Normalize(result)
		}
		return result
	}

	sizes := []int{50, 200, 500, 1000}
	baseText := "Meeting at 14:30 costs $50.00 with 25% off "

	for _, size := range sizes {
		input := strings.Repeat(baseText, size/len(baseText)+1)[:size]
		b.Run(string(rune('0'+size/100))+"00_chars", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				applyAll(input)
			}
		})
	}
}

// =============================================================================
// Concurrent Access Benchmarks
// =============================================================================

func BenchmarkConcurrentNormalization(b *testing.B) {
	normalizer := NewSymbolNormalizer(benchLogger())

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			normalizer.Normalize(symbolInput)
		}
	})
}

func BenchmarkConcurrentChain(b *testing.B) {
	normalizers := []internal_type.TextNormalizer{
		NewCurrencyNormalizer(benchLogger()),
		NewDateNormalizer(benchLogger()),
		NewTimeNormalizer(benchLogger()),
		NewSymbolNormalizer(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range normalizers {
			result = n.Normalize(result)
		}
		return result
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			applyAll(mediumSentence)
		}
	})
}

// =============================================================================
// Worst Case Benchmarks (many matches)
// =============================================================================

func BenchmarkWorstCaseCurrency(b *testing.B) {
	normalizer := NewCurrencyNormalizer(benchLogger())
	// Many currency values in one string
	input := strings.Repeat("$1.00 ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(input)
	}
}

func BenchmarkWorstCaseSymbol(b *testing.B) {
	normalizer := NewSymbolNormalizer(benchLogger())
	// Many symbols in one string
	input := strings.Repeat("% & + = @ # ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(input)
	}
}

func BenchmarkWorstCaseAddress(b *testing.B) {
	normalizer := NewAddressNormalizer(benchLogger())
	// Many address abbreviations
	input := strings.Repeat("123 Main St 456 Park Ave 789 Oak Rd ", 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		normalizer.Normalize(input)
	}
}

// =============================================================================
// Normalizer Creation Benchmarks
// =============================================================================

func BenchmarkNormalizerCreation(b *testing.B) {
	b.Run("currency", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewCurrencyNormalizer(benchLogger())
		}
	})

	b.Run("date", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewDateNormalizer(benchLogger())
		}
	})

	b.Run("time", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewTimeNormalizer(benchLogger())
		}
	})

	b.Run("number", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewNumberToWordNormalizer(benchLogger())
		}
	})

	b.Run("address", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewAddressNormalizer(benchLogger())
		}
	})

	b.Run("url", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewUrlNormalizer(benchLogger())
		}
	})

	b.Run("symbol", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewSymbolNormalizer(benchLogger())
		}
	})

	b.Run("tech", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewTechAbbreviationNormalizer(benchLogger())
		}
	})

	b.Run("role", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewRoleAbbreviationNormalizer(benchLogger())
		}
	})

	b.Run("general", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewGeneralAbbreviationNormalizer(benchLogger())
		}
	})

	b.Run("all_normalizers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewCurrencyNormalizer(benchLogger())
			NewDateNormalizer(benchLogger())
			NewTimeNormalizer(benchLogger())
			NewNumberToWordNormalizer(benchLogger())
			NewAddressNormalizer(benchLogger())
			NewUrlNormalizer(benchLogger())
			NewTechAbbreviationNormalizer(benchLogger())
			NewRoleAbbreviationNormalizer(benchLogger())
			NewGeneralAbbreviationNormalizer(benchLogger())
			NewSymbolNormalizer(benchLogger())
		}
	})
}

// =============================================================================
// Real-World TTS Input Benchmarks
// =============================================================================

func BenchmarkRealWorldTTSInputs(b *testing.B) {
	normalizers := []internal_type.TextNormalizer{
		NewCurrencyNormalizer(benchLogger()),
		NewDateNormalizer(benchLogger()),
		NewTimeNormalizer(benchLogger()),
		NewNumberToWordNormalizer(benchLogger()),
		NewAddressNormalizer(benchLogger()),
		NewUrlNormalizer(benchLogger()),
		NewTechAbbreviationNormalizer(benchLogger()),
		NewRoleAbbreviationNormalizer(benchLogger()),
		NewGeneralAbbreviationNormalizer(benchLogger()),
		NewSymbolNormalizer(benchLogger()),
	}

	applyAll := func(input string) string {
		result := input
		for _, n := range normalizers {
			result = n.Normalize(result)
		}
		return result
	}

	realWorldInputs := map[string]string{
		"customer_service":  "Hello! Your order #12345 for $99.99 will arrive on 2024-01-20 between 14:00 and 17:00. Visit https://track.example.com for updates.",
		"appointment":       "Dr. Smith will see you at 10:30 a.m. on 2024-03-15. The consultation costs $150.00. Our address is 123 Main St.",
		"tech_announcement": "The new AI & ML features in our API are launching on 2024-06-01. CEO John Smith said this represents 25% improvement.",
		"financial_report":  "Q4 revenue: $1,234,567.89 with 15% YoY growth. CFO meeting at 09:00 on 2024-01-30.",
		"simple_greeting":   "Hello, how can I help you today?",
		"numbers_heavy":     "You have 5 items, 12 messages, and 42 notifications. Total: 59 updates.",
	}

	for name, input := range realWorldInputs {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				applyAll(input)
			}
		})
	}
}
