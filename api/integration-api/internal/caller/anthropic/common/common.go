// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_anthropic_common

import (
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

const APIKey = "key"

func ResolveAPIKey(credential *protos.Credential) (string, error) {
	if credential == nil || credential.GetValue() == nil {
		return "", errors.New("unable to resolve the credential")
	}

	credentialMap := credential.GetValue().AsMap()
	rawAPIKey, ok := credentialMap[APIKey]
	if !ok {
		return "", errors.New("unable to resolve the credential")
	}

	apiKey, ok := rawAPIKey.(string)
	if !ok || apiKey == "" {
		return "", errors.New("unable to resolve the credential")
	}
	return apiKey, nil
}

func NewClient(credential *protos.Credential) (*anthropic.Client, error) {
	apiKey, err := ResolveAPIKey(credential)
	if err != nil {
		return nil, err
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &client, nil
}

func UsageMetrics(usages anthropic.Usage) []*protos.Metric {
	metrics := make([]*protos.Metric, 0, 3)
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.OutputTokens),
		Description: "Input token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.InputTokens),
		Description: "Output Token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.InputTokens+usages.OutputTokens),
		Description: "Total Token",
	})
	return metrics
}
