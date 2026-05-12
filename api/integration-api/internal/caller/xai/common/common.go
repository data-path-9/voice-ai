// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_common

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"

	internal_xai_artifacts "github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

const (
	APIKey              = "key"
	OptionEndpointKey   = "connection.endpoint"
	DefaultGRPCEndpoint = "api.x.ai:443"
)

func ResolveAPIKey(credential *protos.Credential) (string, error) {
	if credential == nil || credential.GetValue() == nil {
		return "", errors.New("unable to resolve the credential")
	}

	credentialValue := credential.GetValue().AsMap()
	raw, ok := credentialValue[APIKey]
	if !ok {
		return "", errors.New("unable to resolve the credential")
	}

	apiKey, ok := raw.(string)
	if !ok || strings.TrimSpace(apiKey) == "" {
		return "", errors.New("unable to resolve the credential")
	}
	return strings.TrimSpace(apiKey), nil
}

func ResolveEndpoint(connectionOptions map[string]string) string {
	if connectionOptions == nil {
		return DefaultGRPCEndpoint
	}
	if endpoint, ok := connectionOptions[OptionEndpointKey]; ok && strings.TrimSpace(endpoint) != "" {
		return strings.TrimSpace(endpoint)
	}
	return DefaultGRPCEndpoint
}

func AuthContext(ctx context.Context, apiKey string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return metadata.AppendToOutgoingContext(
		ctx,
		"authorization",
		fmt.Sprintf("Bearer %s", apiKey),
	)
}

func NewChatClient(endpoint string) (internal_xai_artifacts.ChatClient, *grpc.ClientConn, error) {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultGRPCEndpoint
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})),
	)
	if err != nil {
		return nil, nil, err
	}
	return internal_xai_artifacts.NewChatClient(conn), conn, nil
}

func CompletionUsageMetrics(usages *internal_xai_artifacts.SamplingUsage) []*protos.Metric {
	if usages == nil {
		return nil
	}

	metrics := make([]*protos.Metric, 0, 3)
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.GetCompletionTokens()),
		Description: "LLM Output token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.INPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.GetPromptTokens()),
		Description: "LLM Input token",
	})
	metrics = append(metrics, &protos.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", usages.GetTotalTokens()),
		Description: "Total Token",
	})
	return metrics
}
