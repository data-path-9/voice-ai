// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_openai_verify_credential

import (
	"context"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestNew_ReturnsVerifier(t *testing.T) {
	verifier := New(newTestLogger(), nil)
	require.NotNil(t, verifier)
}

func TestCredentialVerifier_ReturnsCredentialErrorForInvalidCredential(t *testing.T) {
	verifier := &caller{
		logger:     newTestLogger(),
		credential: nil,
	}

	result, err := verifier.CredentialVerifier(context.Background(), &internal_callers.CredentialVerifierOptions{})
	require.Error(t, err)
	assert.Nil(t, result)
}
