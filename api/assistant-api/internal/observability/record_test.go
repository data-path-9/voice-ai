// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"errors"
	"testing"
)

func TestAttributeValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: ""},
		{name: "string", value: "hello", want: "hello"},
		{name: "error", value: errors.New("failed"), want: "failed"},
		{name: "number", value: 42, want: "42"},
		{name: "bool", value: true, want: "true"},
		{name: "map", value: map[string]interface{}{"status": "ok"}, want: `{"status":"ok"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AttributeValue(tt.value)
			if got != tt.want {
				t.Fatalf("AttributeValue() = %q, want %q", got, tt.want)
			}
		})
	}
}
