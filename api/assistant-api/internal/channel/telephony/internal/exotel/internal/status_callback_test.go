// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_exotel

import (
	"errors"
	"testing"

	"github.com/rapidaai/pkg/utils"
)

func TestNewStatusCallback_MissingStatusUsesTypedError(t *testing.T) {
	callback, err := NewStatusCallback(utils.Option{"CallSid": "call-id"})

	if callback != nil {
		t.Fatalf("callback=%+v want nil", callback)
	}
	if !errors.Is(err, ErrStatusCallbackStatusMissing) {
		t.Fatalf("err=%v want %v", err, ErrStatusCallbackStatusMissing)
	}
}
