// Copyright (c) 2023-2025 RapidaAI
// Author: RapidaAI Team <team@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telnyx

import (
	"errors"
	"testing"
)

func TestNewStatusCallback_MissingDataUsesTypedError(t *testing.T) {
	callback, err := NewStatusCallback(map[string]interface{}{"event": "call.hangup"}, "")

	if callback != nil {
		t.Fatalf("callback=%+v want nil", callback)
	}
	if !errors.Is(err, ErrStatusCallbackDataMissing) {
		t.Fatalf("err=%v want %v", err, ErrStatusCallbackDataMissing)
	}
}

func TestNewStatusCallback_MissingEventTypeUsesTypedError(t *testing.T) {
	callback, err := NewStatusCallback(map[string]interface{}{"data": map[string]interface{}{"id": "call-id"}}, "")

	if callback != nil {
		t.Fatalf("callback=%+v want nil", callback)
	}
	if !errors.Is(err, ErrStatusCallbackEventTypeMissing) {
		t.Fatalf("err=%v want %v", err, ErrStatusCallbackEventTypeMissing)
	}
}
