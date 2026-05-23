// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_callers

import (
	"fmt"

	internal_anthropic_callers "github.com/rapidaai/api/integration-api/internal/caller/anthropic"
	internal_anthropic_verify_credential "github.com/rapidaai/api/integration-api/internal/caller/anthropic/verify_credential"
	internal_azure_callers "github.com/rapidaai/api/integration-api/internal/caller/azure"
	internal_azure_text_embedding "github.com/rapidaai/api/integration-api/internal/caller/azure/text_embedding"
	internal_azure_verify_credential "github.com/rapidaai/api/integration-api/internal/caller/azure/verify_credential"
	internal_cohere_callers "github.com/rapidaai/api/integration-api/internal/caller/cohere"
	internal_cohere_reranking "github.com/rapidaai/api/integration-api/internal/caller/cohere/reranking"
	internal_cohere_text_embedding "github.com/rapidaai/api/integration-api/internal/caller/cohere/text_embedding"
	internal_cohere_verify_credential "github.com/rapidaai/api/integration-api/internal/caller/cohere/verify_credential"
	internal_custom_llm_callers "github.com/rapidaai/api/integration-api/internal/caller/custom_llm"
	internal_gemini_callers "github.com/rapidaai/api/integration-api/internal/caller/gemini"
	internal_gemini_text_embedding "github.com/rapidaai/api/integration-api/internal/caller/gemini/text_embedding"
	internal_gemini_verify_credential "github.com/rapidaai/api/integration-api/internal/caller/gemini/verify_credential"
	internal_openai_callers "github.com/rapidaai/api/integration-api/internal/caller/openai"
	internal_openai_text_embedding "github.com/rapidaai/api/integration-api/internal/caller/openai/text_embedding"
	internal_openai_verify_credential "github.com/rapidaai/api/integration-api/internal/caller/openai/verify_credential"
	internal_openrouter_callers "github.com/rapidaai/api/integration-api/internal/caller/openrouter"
	internal_vertexai_callers "github.com/rapidaai/api/integration-api/internal/caller/vertexai"
	internal_vertexai_text_embedding "github.com/rapidaai/api/integration-api/internal/caller/vertexai/text_embedding"
	internal_vertexai_verify_credential "github.com/rapidaai/api/integration-api/internal/caller/vertexai/verify_credential"
	internal_voyageai_callers "github.com/rapidaai/api/integration-api/internal/caller/voyageai"
	internal_xai_callers "github.com/rapidaai/api/integration-api/internal/caller/xai"
	internal_types "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type IntegrationProvider string

const (
	OPENAI      IntegrationProvider = "openai"
	CUSTOM_LLM  IntegrationProvider = "custom-llm"
	ANTHROPIC   IntegrationProvider = "anthropic"
	GEMINI      IntegrationProvider = "gemini"
	VERTEXAI    IntegrationProvider = "vertexai"
	AZURE       IntegrationProvider = "azure-foundry"
	COHERE      IntegrationProvider = "cohere"
	MISTRAL     IntegrationProvider = "mistral"
	REPLICATE   IntegrationProvider = "replicate"
	HUGGINGFACE IntegrationProvider = "huggingface"
	VOYAGEAI    IntegrationProvider = "voyageai"
	OPENROUTER  IntegrationProvider = "openrouter"
	XAI         IntegrationProvider = "xai"
)

func GetLargeLanguageCaller(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.LargeLanguageCaller, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return nil, fmt.Errorf("openai large language caller is removed; use chat/chat_stream openai factory")
	case CUSTOM_LLM:
		return nil, fmt.Errorf("custom-llm large language caller is removed; use chat/chat_stream custom-llm factory")
	case ANTHROPIC:
		return nil, fmt.Errorf("anthropic large language caller is removed; use chat/chat_stream anthropic factory")
	case GEMINI:
		return nil, fmt.Errorf("gemini large language caller is removed; use chat/chat_stream gemini factory")
	case VERTEXAI:
		return nil, fmt.Errorf("vertexai large language caller is removed; use chat/chat_stream vertexai factory")
	case COHERE:
		return nil, fmt.Errorf("cohere large language caller is removed; use chat/chat_stream cohere factory")
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}

func GetChat(
	logger commons.Logger,
	provider string,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_types.Chat, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return internal_openai_callers.NewChat(logger, credential, connectionOptions)
	case AZURE:
		return internal_azure_callers.NewChat(logger, credential, connectionOptions)
	case GEMINI:
		return internal_gemini_callers.NewChat(logger, credential, connectionOptions)
	case VERTEXAI:
		return internal_vertexai_callers.NewChat(logger, credential, connectionOptions)
	case ANTHROPIC:
		return internal_anthropic_callers.NewChat(logger, credential, connectionOptions)
	case COHERE:
		return internal_cohere_callers.NewChat(logger, credential, connectionOptions)
	case CUSTOM_LLM:
		return internal_custom_llm_callers.NewChat(logger, credential, connectionOptions)
	case OPENROUTER:
		return internal_openrouter_callers.NewChat(logger, credential, connectionOptions)
	case XAI:
		return internal_xai_callers.NewChat(logger, credential, connectionOptions)
	default:
		return nil, fmt.Errorf("unsupported chat provider: %s", provider)
	}
}

func GetChatStream(
	logger commons.Logger,
	provider string,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_types.ChatStream, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return internal_openai_callers.NewChatStream(logger, credential, connectionOptions)
	case AZURE:
		return internal_azure_callers.NewChatStream(logger, credential, connectionOptions)
	case GEMINI:
		return internal_gemini_callers.NewChatStream(logger, credential, connectionOptions)
	case VERTEXAI:
		return internal_vertexai_callers.NewChatStream(logger, credential, connectionOptions)
	case ANTHROPIC:
		return internal_anthropic_callers.NewChatStream(logger, credential, connectionOptions)
	case COHERE:
		return internal_cohere_callers.NewChatStream(logger, credential, connectionOptions)
	case CUSTOM_LLM:
		return internal_custom_llm_callers.NewChatStream(logger, credential, connectionOptions)
	case OPENROUTER:
		return internal_openrouter_callers.NewChatStream(logger, credential, connectionOptions)
	case XAI:
		return internal_xai_callers.NewChatStream(logger, credential, connectionOptions)
	default:
		return nil, fmt.Errorf("unsupported stream provider: %s", provider)
	}
}

func GetEmbeddingCaller(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.EmbeddingCaller, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return internal_openai_text_embedding.New(logger, credential), nil
	case GEMINI:
		return internal_gemini_text_embedding.New(logger, credential), nil
	case VERTEXAI:
		return internal_vertexai_text_embedding.New(logger, credential), nil
	case AZURE:
		return internal_azure_text_embedding.New(logger, credential), nil
	case COHERE:
		return internal_cohere_text_embedding.New(logger, credential), nil
	case VOYAGEAI:
		return internal_voyageai_callers.NewEmbeddingCaller(logger, credential), nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
	}
}

func GetRerankingCaller(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.RerankingCaller, error) {
	switch IntegrationProvider(provider) {
	case COHERE:
		return internal_cohere_reranking.New(logger, credential), nil
	case VOYAGEAI:
		return internal_voyageai_callers.NewRerankingCaller(logger, credential), nil
	default:
		return nil, fmt.Errorf("unsupported reranking provider: %s", provider)
	}
}

func GetVerifier(logger commons.Logger, provider string, credential *protos.Credential) (internal_types.Verifier, error) {
	switch IntegrationProvider(provider) {
	case OPENAI:
		return internal_openai_verify_credential.New(logger, credential), nil
	case CUSTOM_LLM:
		return internal_custom_llm_callers.NewVerifyCredentialCaller(logger, credential), nil
	case ANTHROPIC:
		return internal_anthropic_verify_credential.New(logger, credential), nil
	case GEMINI:
		return internal_gemini_verify_credential.New(logger, credential), nil
	case VERTEXAI:
		return internal_vertexai_verify_credential.New(logger, credential), nil
	case AZURE:
		return internal_azure_verify_credential.New(logger, credential), nil
	case COHERE:
		return internal_cohere_verify_credential.New(logger, credential), nil
	case VOYAGEAI:
		return internal_voyageai_callers.NewVerifyCredentialCaller(logger, credential), nil
	default:
		return nil, fmt.Errorf("unsupported provider for credential verification: %s", provider)
	}
}
