// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_azure_common

import (
	"errors"
	"fmt"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

const (
	DefaultURL         = "https://api.openai.com/v1"
	EndpointKey        = "endpoint"
	SubscriptionKeyKey = "subscription_key"

	StreamMaxConnsPerHost     = 100
	StreamMaxIdleConnsPerHost = 20
	StreamMaxIdleConns        = 100
	StreamIdleConnTimeout     = 5 * time.Minute

	ChatRoleAssistant = "assistant"
	ChatRoleFunction  = "function"
	ChatRoleSystem    = "system"
	ChatRoleTool      = "tool"
	ChatRoleUser      = "user"
)

func ResolveCredential(
	logger commons.Logger,
	credential *protos.Credential,
) (endpoint string, subscriptionKey string, err error) {
	if credential == nil || credential.GetValue() == nil {
		return "", "", errors.New("unable to resolve the credential")
	}

	credentialMap := credential.GetValue().AsMap()
	rawSubscriptionKey, ok := credentialMap[SubscriptionKeyKey]
	if !ok {
		logger.Errorf("Unable to get client for user")
		return "", "", errors.New("unable to resolve the credential")
	}
	subscriptionKey, ok = rawSubscriptionKey.(string)
	if !ok || subscriptionKey == "" {
		logger.Errorf("Unable to get client for user")
		return "", "", errors.New("unable to resolve the credential")
	}

	rawEndpoint, ok := credentialMap[EndpointKey]
	if !ok {
		logger.Debugf("Using default client connection url")
		return DefaultURL, subscriptionKey, nil
	}
	endpoint, ok = rawEndpoint.(string)
	if !ok || endpoint == "" {
		return "", "", errors.New("unable to resolve the credential")
	}
	return endpoint, subscriptionKey, nil
}

func NewClient(logger commons.Logger, credential *protos.Credential) (*openai.Client, error) {
	endpoint, subscriptionKey, err := ResolveCredential(logger, credential)
	if err != nil {
		return nil, err
	}
	client := openai.NewClient(
		option.WithBaseURL(endpoint),
		option.WithAPIKey(subscriptionKey),
	)
	return &client, nil
}

func CompletionUsageMetrics(usages openai.CompletionUsage) []*protos.Metric {
	metrics := make([]*protos.Metric, 0, 3)
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.CompletionTokens),
		Description: "LLM Output token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.PromptTokens),
		Description: "LLM Input Token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.TotalTokens),
		Description: "Total Token",
	})
	return metrics
}

func ResponseUsageMetrics(usages responses.ResponseUsage) []*protos.Metric {
	metrics := make([]*protos.Metric, 0, 4)
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.OutputTokens),
		Description: "LLM Output token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.InputTokens),
		Description: "LLM Input token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.TotalTokens),
		Description: "Total Token",
	})
	if usages.InputTokensDetails.CachedTokens > 0 {
		metrics = append(metrics, &protos.Metric{
			Name:        type_enums.CACHED_CONTENT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.InputTokensDetails.CachedTokens),
			Description: "Cached content tokens",
		})
	}
	return metrics
}
