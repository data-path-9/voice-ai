// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_gemini_common

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/genai"

	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

const APIKey = "key"

func ResolveAPIKey(credential *protos.Credential) (string, error) {
	if credential == nil || credential.GetValue() == nil {
		return "", errors.New("unable to resolve the credential")
	}

	credentialMap := credential.GetValue().AsMap()
	rawKey, ok := credentialMap[APIKey]
	if !ok {
		return "", errors.New("unable to resolve the credential")
	}

	key, ok := rawKey.(string)
	if !ok || key == "" {
		return "", errors.New("unable to resolve the credential")
	}
	return key, nil
}

func NewClient(credential *protos.Credential) (*genai.Client, error) {
	apiKey, err := ResolveAPIKey(credential)
	if err != nil {
		return nil, err
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func UsageMetrics(usages *genai.GenerateContentResponseUsageMetadata) []*protos.Metric {
	metrics := make([]*protos.Metric, 0)
	if usages == nil {
		return metrics
	}

	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.PromptTokenCount),
		Description: "Input tokens (including cached content)",
	})

	if usages.CachedContentTokenCount > 0 {
		metrics = append(metrics, &protos.Metric{
			Name:        "CACHED_CONTENT_TOKEN",
			Value:       fmt.Sprintf("%d", usages.CachedContentTokenCount),
			Description: "Cached content tokens",
		})
	}

	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.CandidatesTokenCount),
		Description: "Output tokens",
	})

	if usages.ToolUsePromptTokenCount > 0 {
		metrics = append(metrics, &protos.Metric{
			Name:        "TOOL_USE_PROMPT_TOKEN",
			Value:       fmt.Sprintf("%d", usages.ToolUsePromptTokenCount),
			Description: "Tool-use prompt tokens",
		})
	}

	if usages.ThoughtsTokenCount > 0 {
		metrics = append(metrics, &protos.Metric{
			Name:        "THOUGHTS_TOKEN",
			Value:       fmt.Sprintf("%d", usages.ThoughtsTokenCount),
			Description: "Thoughts tokens for thinking models",
		})
	}

	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.TotalTokenCount),
		Description: "Total tokens (prompt, response, and tool-use)",
	})

	return metrics
}
