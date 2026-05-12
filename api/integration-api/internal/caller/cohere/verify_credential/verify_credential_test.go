// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_verify_credential

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestCredentialVerifier_RejectsMissingCredential(t *testing.T) {
	verifier := New(newTestLogger(), nil)
	result, err := verifier.CredentialVerifier(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, result)
}
