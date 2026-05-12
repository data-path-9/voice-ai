// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cloud.google.com/go/auth"
	"google.golang.org/genai"

	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

const (
	ProjectIDKey        = "project_id"
	ServiceAccountKey   = "service_account_key"
	RegionKey           = "region"
	cloudPlatformScopes = "https://www.googleapis.com/auth/cloud-platform"
)

type serviceAccountCredentials struct {
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	TokenURI    string `json:"token_uri"`
}

func ResolveCredential(credential *protos.Credential) (projectID string, serviceAccountJSON string, region string, err error) {
	if credential == nil || credential.GetValue() == nil {
		return "", "", "", errors.New("unable to resolve the credential")
	}

	credentialMap := credential.GetValue().AsMap()

	rawProjectID, ok := credentialMap[ProjectIDKey]
	if !ok {
		return "", "", "", errors.New("unable to resolve the credential")
	}
	projectID, ok = rawProjectID.(string)
	if !ok || projectID == "" {
		return "", "", "", errors.New("unable to resolve the credential")
	}

	rawServiceAccount, ok := credentialMap[ServiceAccountKey]
	if !ok {
		return "", "", "", errors.New("unable to resolve the credential")
	}
	serviceAccountJSON, ok = rawServiceAccount.(string)
	if !ok || serviceAccountJSON == "" {
		return "", "", "", errors.New("unable to resolve the credential")
	}

	rawRegion, ok := credentialMap[RegionKey]
	if !ok {
		return "", "", "", errors.New("unable to resolve the credential")
	}
	region, ok = rawRegion.(string)
	if !ok || region == "" {
		return "", "", "", errors.New("unable to resolve the credential")
	}

	return projectID, serviceAccountJSON, region, nil
}

func NewClient(credential *protos.Credential) (*genai.Client, error) {
	projectID, serviceAccountJSON, region, err := ResolveCredential(credential)
	if err != nil {
		return nil, err
	}

	serviceAccountBytes := []byte(serviceAccountJSON)
	var parsedServiceAccount serviceAccountCredentials
	if err := json.Unmarshal(serviceAccountBytes, &parsedServiceAccount); err != nil {
		return nil, fmt.Errorf("failed to parse service account JSON: %w", err)
	}
	if parsedServiceAccount.ClientEmail == "" || parsedServiceAccount.PrivateKey == "" || parsedServiceAccount.TokenURI == "" {
		return nil, errors.New("unable to resolve the credential")
	}

	tokenProvider, err := auth.New2LOTokenProvider(&auth.Options2LO{
		Email:      parsedServiceAccount.ClientEmail,
		PrivateKey: []byte(parsedServiceAccount.PrivateKey),
		TokenURL:   parsedServiceAccount.TokenURI,
		Scopes:     []string{cloudPlatformScopes},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create token provider: %w", err)
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		Backend:  genai.BackendVertexAI,
		Project:  projectID,
		Location: region,
		Credentials: auth.NewCredentials(&auth.CredentialsOptions{
			TokenProvider: tokenProvider,
			JSON:          serviceAccountBytes,
		}),
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
