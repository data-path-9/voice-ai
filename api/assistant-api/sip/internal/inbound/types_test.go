// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package inbound

import "testing"

func TestFailureClassValues(t *testing.T) {
	expected := map[FailureClass]string{
		FailureConfig: "config",
		FailureAuth:   "auth",
		FailureMedia:  "media",
		FailureRTP:    "rtp",
		FailureDialog: "dialog",
		FailureSetup:  "setup",
	}

	for failureClass, value := range expected {
		if string(failureClass) != value {
			t.Fatalf("expected failure class %q, got %q", value, failureClass)
		}
	}
}

func TestSDPContentType(t *testing.T) {
	if SDPContentType != "application/sdp" {
		t.Fatalf("unexpected SDP content type: %q", SDPContentType)
	}
}
