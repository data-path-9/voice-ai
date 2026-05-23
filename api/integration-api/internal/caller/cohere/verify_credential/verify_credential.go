// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_verify_credential

import (
	"context"
	"errors"
	"time"

	internal_cohere_common "github.com/rapidaai/api/integration-api/internal/caller/cohere/common"
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
	_ = options

	client, err := internal_cohere_common.NewClient(vc.credential)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	response, err := client.CheckApiKey(ctx)
	if err != nil {
		return nil, err
	}
	if response.Valid {
		return utils.Ptr("valid"), nil
	}
	return nil, errors.New("given credential is not verified by cohere")
}
