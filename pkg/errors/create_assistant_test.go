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

func TestCreateAssistantErrorConstants(t *testing.T) {
	if CreateAssistantInvalidRequest.HTTPStatusCode != http.StatusBadRequest {
		t.Fatal("expected create assistant invalid request to use HTTP bad request")
	}
	if CreateAssistantInvalidRequest.CodeString() != "1001001" {
		t.Fatal("expected create assistant invalid request to expose custom platform code")
	}
	if CreateAssistantUnauthenticated.Error == "" || CreateAssistantUnauthenticated.ErrorMessage == "" {
		t.Fatal("expected create assistant authentication messages to be defined")
	}
	if CreateAssistantMissingProvider.Error == "" || CreateAssistantInvalidProvider.ErrorMessage == "" {
		t.Fatal("expected create assistant provider validation messages to be defined")
	}
}
