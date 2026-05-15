// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts

import (
	"context"
	"fmt"

	internal_transformer_custom_tts_websocket_v1 "github.com/rapidaai/api/assistant-api/internal/transformer/custom_tts/websocket_v1"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type Compatibility string

const (
	CompatibilityWebSocketV1 Compatibility = "websocket_v1"

	DefaultCompatibility = CompatibilityWebSocketV1
)

const (
	CredentialKeyAPICompatibilitySnake = "api_compatibility"
	CredentialKeyAPICompatibilityCamel = "apiCompatibility"
)

func ResolveCompatibility(credential *protos.VaultCredential) (Compatibility, error) {
	if credential == nil || credential.GetValue() == nil {
		return DefaultCompatibility, nil
	}
	return parseCompatibility(credential.GetValue().AsMap())
}

func NewTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	compatibility, err := ResolveCompatibility(credential)
	if err != nil {
		return nil, err
	}

	switch compatibility {
	case CompatibilityWebSocketV1:
		return internal_transformer_custom_tts_websocket_v1.NewTextToSpeech(ctx, logger, credential, onPacket, opts)
	default:
		return nil, fmt.Errorf("custom-tts: unsupported api compatibility %q", compatibility)
	}
}

func parseCompatibility(credentials map[string]any) (Compatibility, error) {
	compatibility := DefaultCompatibility
	rawCompatibility, found := credentials[CredentialKeyAPICompatibilitySnake]
	if !found {
		rawCompatibility, found = credentials[CredentialKeyAPICompatibilityCamel]
	}
	if !found || rawCompatibility == nil {
		return compatibility, nil
	}

	compatibilityStr, ok := rawCompatibility.(string)
	if !ok {
		return "", fmt.Errorf("custom-tts: api compatibility must be a string")
	}
	if compatibilityStr == "" {
		return "", fmt.Errorf("custom-tts: api compatibility must not be empty")
	}

	return Compatibility(compatibilityStr), nil
}
