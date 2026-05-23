// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_gemini_verify_credential

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func credentialWithValues(t *testing.T, values map[string]interface{}) *protos.Credential {
	t.Helper()
	value, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.Credential{Value: value}
}

func TestCredentialVerifier_Succeeds(t *testing.T) {
	verifier := New(newTestLogger(), credentialWithValues(t, map[string]interface{}{"key": "gemini-test-key"}))
	result, err := verifier.CredentialVerifier(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "valid", *result)
}

func TestCredentialVerifier_RejectsMissingCredential(t *testing.T) {
	verifier := New(newTestLogger(), nil)
	result, err := verifier.CredentialVerifier(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, result)
}
