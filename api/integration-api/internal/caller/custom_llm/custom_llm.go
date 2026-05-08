// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_callers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	integration_api "github.com/rapidaai/protos"
)

type CustomLLM struct {
	logger     commons.Logger
	credential internal_callers.CredentialResolver
}

const (
	CRED_API_COMPATIBILITY = "apiCompatibility"
	CRED_BASE_URL          = "baseUrl"
	CRED_HEADERS           = "headers"
)

const (
	CompatOpenAI    = "openai"
	CompatAnthropic = "anthropic"
	CompatCustom    = "custom"
)

const (
	ChatRoleAssistant string = "assistant"
	ChatRoleFunction  string = "function"
	ChatRoleSystem    string = "system"
	ChatRoleTool      string = "tool"
	ChatRoleUser      string = "user"
)

func customLLM(logger commons.Logger, credential *integration_api.Credential) CustomLLM {
	_credential := credential.GetValue().AsMap()
	return CustomLLM{
		logger: logger,
		credential: func() map[string]interface{} {
			return _credential
		},
	}
}

func (cl *CustomLLM) GetClient() (*openai.Client, error) {
	credentials := cl.credential()

	compat := strings.ToLower(strings.TrimSpace(stringValue(credentials, CRED_API_COMPATIBILITY)))
	if compat == "" {
		compat = CompatOpenAI
	}
	if compat != CompatOpenAI {
		return nil, fmt.Errorf("custom-llm: api compatibility %q is not implemented", compat)
	}

	baseURL := strings.TrimSpace(stringValue(credentials, CRED_BASE_URL))
	if baseURL == "" {
		cl.logger.Errorf("custom-llm: missing base url")
		return nil, errors.New("custom-llm: base url must be a non-empty string")
	}

	opts := []option.RequestOption{
		option.WithBaseURL(baseURL),
	}
	for k, v := range parseHeaders(credentials[CRED_HEADERS]) {
		opts = append(opts, option.WithHeader(k, v))
	}

	client := openai.NewClient(opts...)
	return &client, nil
}

func (cl *CustomLLM) GetCompletionUsages(usages openai.CompletionUsage) []*integration_api.Metric {
	metrics := make([]*integration_api.Metric, 0)
	metrics = append(metrics, &integration_api.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.CompletionTokens),
		Description: "LLM Output token",
	})

	metrics = append(metrics, &integration_api.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.PromptTokens),
		Description: "LLM Input Token",
	})

	metrics = append(metrics, &integration_api.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.TotalTokens),
		Description: "Total Token",
	})
	return metrics
}

func stringValue(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func parseHeaders(raw interface{}) map[string]string {
	out := map[string]string{}
	if raw == nil {
		return out
	}
	switch v := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return out
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
			return out
		}
		for k, val := range decoded {
			if s, ok := val.(string); ok {
				out[k] = s
			} else {
				out[k] = fmt.Sprintf("%v", val)
			}
		}
	case map[string]interface{}:
		for k, val := range v {
			if s, ok := val.(string); ok {
				out[k] = s
			} else {
				out[k] = fmt.Sprintf("%v", val)
			}
		}
	}
	return out
}
