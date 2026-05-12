// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_common

import "time"

type Compatibility string

const (
	CompatibilityOpenAIChatCompletions Compatibility = "openai_chat_completions"
	CompatibilityOpenAIResponses       Compatibility = "openai_responses"
	CompatibilityAnthropicMessages     Compatibility = "anthropic_messages"
	CompatibilityGeminiGenerateContent Compatibility = "gemini_generate_content"
	CompatibilityOpenAICompatible      Compatibility = "openai_compatible"
)

const DefaultCompatibility = CompatibilityOpenAIChatCompletions

const (
	CredentialKeyAPICompatibilitySnake = "api_compatibility"
	CredentialKeyAPICompatibilityCamel = "apiCompatibility"
	CredentialKeyBaseURLSnake          = "base_url"
	CredentialKeyBaseURLCamel          = "baseUrl"
	CredentialKeyHeaders               = "headers"
)

const (
	DefaultClientTimeout  = 10 * time.Minute
	DefaultRequestTimeout = 60 * time.Second
)

const (
	StreamMaxConnsPerHost     = 100
	StreamMaxIdleConnsPerHost = 20
	StreamMaxIdleConns        = 100
	StreamIdleConnTimeout     = 5 * time.Minute
)

const (
	ChatRoleAssistant = "assistant"
	ChatRoleFunction  = "function"
	ChatRoleSystem    = "system"
	ChatRoleTool      = "tool"
	ChatRoleUser      = "user"
)
