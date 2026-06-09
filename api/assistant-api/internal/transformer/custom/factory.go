// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom

import (
	"context"
	"fmt"

	internal_transformer_custom_stt_http_v1 "github.com/rapidaai/api/assistant-api/internal/transformer/custom/stt_http_v1"
	internal_transformer_custom_stt_websocket_v1 "github.com/rapidaai/api/assistant-api/internal/transformer/custom/stt_websocket_v1"
	internal_transformer_custom_tts_websocket_v1 "github.com/rapidaai/api/assistant-api/internal/transformer/custom/tts_websocket_v1"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type Compatibility string

const (
	CompatibilityWebSocketV1 Compatibility = "websocket_v1"
	CompatibilityHTTPV1      Compatibility = "http_v1"
	DefaultCompatibility                   = CompatibilityWebSocketV1
)

const (
	CredentialKeyAPICompatibilitySnake = "api_compatibility"
	CredentialKeyAPICompatibilityCamel = "apiCompatibility"
)

func ResolveCompatibility(credential *protos.VaultCredential) (Compatibility, error) {
	return resolveCompatibility("custom transformer", credential)
}

func resolveCompatibility(providerLabel string, credential *protos.VaultCredential) (Compatibility, error) {
	if credential == nil || credential.GetValue() == nil {
		return DefaultCompatibility, nil
	}
	return compatibility(providerLabel, credential.GetValue().AsMap())
}

func NewTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	compatibility, err := resolveCompatibility("custom-tts", credential)
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

func NewSpeechToText(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.SpeechToTextTransformer, error) {
	compatibility, err := resolveCompatibility("custom-stt", credential)
	if err != nil {
		return nil, err
	}

	switch compatibility {
	case CompatibilityWebSocketV1:
		return internal_transformer_custom_stt_websocket_v1.NewSpeechToText(ctx, logger, credential, onPacket, opts)
	case CompatibilityHTTPV1:
		return internal_transformer_custom_stt_http_v1.NewSpeechToText(ctx, logger, credential, onPacket, opts)
	default:
		return nil, fmt.Errorf("custom-stt: unsupported api compatibility %q", compatibility)
	}
}

func compatibility(providerLabel string, credentials map[string]any) (Compatibility, error) {
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
		return "", fmt.Errorf("%s: api compatibility must be a string", providerLabel)
	}
	if compatibilityStr == "" {
		return "", fmt.Errorf("%s: api compatibility must not be empty", providerLabel)
	}

	return Compatibility(compatibilityStr), nil
}
