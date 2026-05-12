// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_entity

import "strings"

const (
	AssistantAuthenticationProviderHTTP = "http"
	AssistantWebhookProviderHTTP        = "http"
	AssistantAnalysisProviderEndpoint   = "endpoint"
)

func NormalizeAssistantAuthenticationProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case AssistantAuthenticationProviderHTTP:
		return AssistantAuthenticationProviderHTTP
	default:
		return AssistantAuthenticationProviderHTTP
	}
}

func NormalizeAssistantWebhookProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case AssistantWebhookProviderHTTP:
		return AssistantWebhookProviderHTTP
	default:
		return AssistantWebhookProviderHTTP
	}
}

func NormalizeAssistantAnalysisProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case AssistantAnalysisProviderEndpoint:
		return AssistantAnalysisProviderEndpoint
	default:
		return AssistantAnalysisProviderEndpoint
	}
}
