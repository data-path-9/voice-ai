// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import "testing"

func TestMaxUint64(t *testing.T) {
	tests := []struct {
		name     string
		a, b     uint64
		expected uint64
	}{
		{"a > b", 10, 5, 10},
		{"a < b", 5, 10, 10},
		{"equal", 5, 5, 5},
		{"zero", 0, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaxUint64(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestMinUint64(t *testing.T) {
	tests := []struct {
		name     string
		a, b     uint64
		expected uint64
	}{
		{"a > b", 10, 5, 5},
		{"a < b", 5, 10, 5},
		{"equal", 5, 5, 5},
		{"zero", 0, 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MinUint64(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestStringToUint32(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  uint32
		expectErr bool
	}{
		{name: "valid", value: "42", expected: 42},
		{name: "trimmed", value: " 120 ", expected: 120},
		{name: "max", value: "4294967295", expected: 4294967295},
		{name: "overflow", value: "4294967296", expectErr: true},
		{name: "negative", value: "-1", expectErr: true},
		{name: "invalid", value: "abc", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToUint32(tt.value)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestStringToUint64(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expected  uint64
		expectErr bool
	}{
		{name: "valid", value: "42", expected: 42},
		{name: "trimmed", value: " 120 ", expected: 120},
		{name: "max", value: "18446744073709551615", expected: 18446744073709551615},
		{name: "overflow", value: "18446744073709551616", expectErr: true},
		{name: "negative", value: "-1", expectErr: true},
		{name: "invalid", value: "abc", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToUint64(tt.value)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
