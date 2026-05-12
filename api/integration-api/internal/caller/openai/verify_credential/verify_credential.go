// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_verify_credential

import (
	"context"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	internal_openai_common "github.com/rapidaai/api/integration-api/internal/caller/openai/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type caller struct {
	logger     commons.Logger
	credential *protos.Credential
	client     *openai.Client
}

func New(logger commons.Logger, credential *protos.Credential) internal_callers.Verifier {
	return &caller{
		logger:     logger,
		credential: credential,
	}
}

func (vc *caller) getClient() (*openai.Client, error) {
	if vc.client != nil {
		return vc.client, nil
	}
	apiKey, err := internal_openai_common.ResolveAPIKey(vc.credential)
	if err != nil {
		return nil, err
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	vc.client = &client
	return vc.client, nil
}

func (vc *caller) CredentialVerifier(
	ctx context.Context,
	options *internal_callers.CredentialVerifierOptions,
) (*string, error) {
	client, err := vc.getClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	_, err = client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Test"),
		},
		Model: openai.ChatModelGPT4o,
	})
	if err != nil {
		return nil, err
	}
	return nil, err
}
