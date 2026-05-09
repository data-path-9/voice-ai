// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_common

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	cohereV2 "github.com/cohere-ai/cohere-go/v2"
	cohereclient "github.com/cohere-ai/cohere-go/v2/client"

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

func NewClient(credential *protos.Credential) (*cohereclient.Client, error) {
	apiKey, err := ResolveAPIKey(credential)
	if err != nil {
		return nil, err
	}
	client := cohereclient.NewClient(
		cohereclient.WithToken(apiKey),
		cohereclient.WithHTTPClient(
			&http.Client{
				Timeout: time.Minute,
			},
		),
	)
	return client, nil
}

func UsageMetrics(usages *cohereV2.Usage) []*protos.Metric {
	metrics := make([]*protos.Metric, 0)
	if usages == nil {
		return metrics
	}

	if usages.Tokens.InputTokens != nil {
		metrics = append(metrics, &protos.Metric{
			Name:        type_enums.OUTPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%f", *usages.Tokens.InputTokens),
			Description: "Input token",
		})
	}
	if usages.Tokens.OutputTokens != nil {
		metrics = append(metrics, &protos.Metric{
			Name:        type_enums.INPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%f", *usages.Tokens.OutputTokens),
			Description: "Output Token",
		})
	}
	if usages.Tokens.OutputTokens != nil && usages.Tokens.InputTokens != nil {
		metrics = append(metrics, &protos.Metric{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%f", *usages.Tokens.InputTokens+*usages.Tokens.OutputTokens),
			Description: "Total Token",
		})
	}
	return metrics
}
