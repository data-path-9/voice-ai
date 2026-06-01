// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_twilio

import (
	"errors"
	"testing"

	"github.com/rapidaai/pkg/utils"
)

func TestNewStatusCallback_MissingStatusReturnsTypedError(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{"CallSid": "CA123"})
	if err == nil {
		t.Fatal("expected missing status error")
	}
	if callback != nil {
		t.Fatalf("expected nil callback, got %#v", callback)
	}
	if !errors.Is(err, ErrStatusCallbackStatusMissing) {
		t.Fatalf("expected status callback status missing error, got %v", err)
	}
}
