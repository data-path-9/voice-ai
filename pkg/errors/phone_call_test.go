// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package errors

import (
	"net/http"
	"testing"
)

func TestPhoneCallErrorConstants(t *testing.T) {
	if CreatePhoneCallInvalidRequest.HTTPStatusCode != http.StatusBadRequest {
		t.Fatal("expected create phone call invalid request to use HTTP bad request")
	}
	if CreatePhoneCallInvalidRequest.CodeString() != "1002001" {
		t.Fatal("expected create phone call invalid request to expose custom platform code")
	}
	if CreatePhoneCallUnauthenticated.HTTPStatusCode != http.StatusUnauthorized {
		t.Fatal("expected create phone call unauthenticated to use HTTP unauthorized")
	}
	if CreatePhoneCallInitiateOutbound.HTTPStatusCode != http.StatusInternalServerError {
		t.Fatal("expected create phone call outbound failure to use HTTP internal server error")
	}
	if CreatePhoneCallMissingToNumber.Error == "" || CreatePhoneCallMissingToNumber.ErrorMessage == "" {
		t.Fatal("expected create phone call to_number validation messages to be defined")
	}
}

func TestBulkPhoneCallErrorConstants(t *testing.T) {
	if CreateBulkPhoneCallInvalidRequest.HTTPStatusCode != http.StatusBadRequest {
		t.Fatal("expected create bulk phone call invalid request to use HTTP bad request")
	}
	if CreateBulkPhoneCallInvalidRequest.CodeString() != "1003001" {
		t.Fatal("expected create bulk phone call invalid request to expose custom platform code")
	}
	if CreateBulkPhoneCallMissingPhoneCalls.HTTPStatusCode != http.StatusBadRequest {
		t.Fatal("expected create bulk phone call missing phone calls to use HTTP bad request")
	}
	if CreateBulkPhoneCallInitiateOutbound.HTTPStatusCode != http.StatusInternalServerError {
		t.Fatal("expected create bulk phone call outbound failure to use HTTP internal server error")
	}
	if CreateBulkPhoneCallInvalidAssistant.Error == "" || CreateBulkPhoneCallInvalidAssistant.ErrorMessage == "" {
		t.Fatal("expected create bulk phone call assistant validation messages to be defined")
	}
}
