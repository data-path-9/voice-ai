// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_callers

import (
	"context"
	"time"

	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type verifyCredentialCaller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func NewVerifyCredentialCaller(
	logger commons.Logger,
	credential *protos.Credential,
) internal_callers.Verifier {
	return &verifyCredentialCaller{
		logger:     logger,
		credential: credential,
	}
}

func (vc *verifyCredentialCaller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions,
) (*string, error) {
	_ = options

	compatibility, err := internal_custom_llm_common.ResolveCompatibility(vc.credential)
	if err != nil {
		return nil, err
	}

	switch compatibility {
	case internal_custom_llm_common.CompatibilityOpenAIChatCompletions,
		internal_custom_llm_common.CompatibilityOpenAIResponses,
		internal_custom_llm_common.CompatibilityOpenAICompatible:
		config, err := internal_custom_llm_common.ParseClientConfig(vc.logger, vc.credential)
		if err != nil {
			return nil, err
		}
		client := internal_custom_llm_common.NewOpenAIClient(config)

		timedCtx, cancel := context.WithTimeout(ctx, time.Minute)
		defer cancel()
		_, err = client.Models.List(timedCtx)
		if err != nil {
			return nil, err
		}
		return nil, nil
	case internal_custom_llm_common.CompatibilityAnthropicMessages:
		return nil, internal_custom_llm_common.NotImplementedCompatibilityError{
			Compatibility: compatibility,
			Feature:       "CredentialVerifier",
		}
	case internal_custom_llm_common.CompatibilityGeminiGenerateContent:
		return nil, internal_custom_llm_common.NotImplementedCompatibilityError{
			Compatibility: compatibility,
			Feature:       "CredentialVerifier",
		}
	default:
		return nil, internal_custom_llm_common.UnsupportedCompatibilityError{Compatibility: compatibility}
	}
}
