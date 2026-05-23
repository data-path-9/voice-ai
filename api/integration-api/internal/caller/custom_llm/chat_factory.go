// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_callers

import (
	"fmt"

	internal_custom_llm_anthropic_messages "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/anthropic_messages"
	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	internal_custom_llm_gemini_generate_content "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/gemini_generate_content"
	internal_custom_llm_openai_chat_completions "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/openai_chat_completions"
	internal_custom_llm_openai_compatible "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/openai_compatible"
	internal_custom_llm_openai_responses "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/openai_responses"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

const OptionTransportKey = "connection.transport"

func NewChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	transport, err := resolveTransport(credential, connectionOptions)
	if err != nil {
		return nil, err
	}

	switch transport {
	case string(internal_custom_llm_common.CompatibilityOpenAIChatCompletions):
		return internal_custom_llm_openai_chat_completions.NewChat(logger, credential)
	case string(internal_custom_llm_common.CompatibilityOpenAICompatible):
		return internal_custom_llm_openai_compatible.NewChat(logger, credential)
	case string(internal_custom_llm_common.CompatibilityOpenAIResponses):
		return internal_custom_llm_openai_responses.NewChat(logger, credential)
	case string(internal_custom_llm_common.CompatibilityAnthropicMessages):
		return internal_custom_llm_anthropic_messages.NewChat(logger, credential)
	case string(internal_custom_llm_common.CompatibilityGeminiGenerateContent):
		return internal_custom_llm_gemini_generate_content.NewChat(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported custom-llm transport option: %s", transport)
	}
}

func NewChatStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	transport, err := resolveTransport(credential, connectionOptions)
	if err != nil {
		return nil, err
	}

	switch transport {
	case string(internal_custom_llm_common.CompatibilityOpenAIChatCompletions):
		return internal_custom_llm_openai_chat_completions.NewStream(logger, credential)
	case string(internal_custom_llm_common.CompatibilityOpenAICompatible):
		return internal_custom_llm_openai_compatible.NewStream(logger, credential)
	case string(internal_custom_llm_common.CompatibilityOpenAIResponses):
		return internal_custom_llm_openai_responses.NewStream(logger, credential)
	case string(internal_custom_llm_common.CompatibilityAnthropicMessages):
		return internal_custom_llm_anthropic_messages.NewStream(logger, credential)
	case string(internal_custom_llm_common.CompatibilityGeminiGenerateContent):
		return internal_custom_llm_gemini_generate_content.NewStream(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported custom-llm transport option: %s", transport)
	}
}

func resolveTransport(
	credential *protos.Credential,
	connectionOptions map[string]string,
) (string, error) {
	if connectionOptions != nil {
		if transport, ok := connectionOptions[OptionTransportKey]; ok && transport != "" {
			return transport, nil
		}
	}

	compatibility, err := internal_custom_llm_common.ResolveCompatibility(credential)
	if err != nil {
		return "", err
	}

	switch compatibility {
	case internal_custom_llm_common.CompatibilityOpenAIResponses:
		return string(internal_custom_llm_common.CompatibilityOpenAIResponses), nil
	case internal_custom_llm_common.CompatibilityOpenAICompatible:
		return string(internal_custom_llm_common.CompatibilityOpenAICompatible), nil
	case internal_custom_llm_common.CompatibilityAnthropicMessages:
		return string(internal_custom_llm_common.CompatibilityAnthropicMessages), nil
	case internal_custom_llm_common.CompatibilityGeminiGenerateContent:
		return string(internal_custom_llm_common.CompatibilityGeminiGenerateContent), nil
	case internal_custom_llm_common.CompatibilityOpenAIChatCompletions:
		return string(internal_custom_llm_common.CompatibilityOpenAIChatCompletions), nil
	default:
		return "", internal_custom_llm_common.UnsupportedCompatibilityError{Compatibility: compatibility}
	}
}
