// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_gemini_verify_credential

import (
	"context"

	internal_gemini_common "github.com/rapidaai/api/integration-api/internal/caller/gemini/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type caller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func New(logger commons.Logger, credential *protos.Credential) internal_callers.Verifier {
	return &caller{
		logger:     logger,
		credential: credential,
	}
}

func (vc *caller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions,
) (*string, error) {
	_ = ctx
	_ = options

	if _, err := internal_gemini_common.ResolveAPIKey(vc.credential); err != nil {
		return nil, err
	}
	return utils.Ptr("valid"), nil
}
