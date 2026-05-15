// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	openai "github.com/openai/openai-go/v3"
	sdkazure "github.com/openai/openai-go/v3/azure"
	"github.com/openai/openai-go/v3/option"

	internal_azure_common "github.com/rapidaai/api/integration-api/internal/caller/azure/common"
	"github.com/rapidaai/protos"
)

const (
	endpointKey        = "endpoint"
	subscriptionKeyKey = "subscription_key"
	apiVersionKey      = "api_version"
	defaultAPIVersion  = "2024-10-21"
)

type idleCloser interface {
	CloseIdleConnections()
}

func isOpenAICompatibleEndpoint(endpoint string) bool {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err == nil {
		return strings.EqualFold(strings.TrimRight(parsed.Path, "/"), "/openai/v1")
	}
	return strings.EqualFold(strings.TrimRight(strings.TrimSpace(endpoint), "/"), "/openai/v1")
}

func buildClientOptions(
	endpoint string,
	subscriptionKey string,
	apiVersion string,
	httpClient *http.Client,
) []option.RequestOption {
	opts := make([]option.RequestOption, 0, 3)
	if isOpenAICompatibleEndpoint(endpoint) {
		opts = append(opts,
			option.WithBaseURL(endpoint),
			sdkazure.WithAPIKey(subscriptionKey),
		)
	} else {
		opts = append(opts,
			sdkazure.WithEndpoint(endpoint, apiVersion),
			sdkazure.WithAPIKey(subscriptionKey),
		)
	}
	if httpClient != nil {
		opts = append(opts, option.WithHTTPClient(httpClient))
	}
	return opts
}

func resolveCredential(credential *protos.Credential) (endpoint string, subscriptionKey string, apiVersion string, err error) {
	if credential == nil || credential.GetValue() == nil {
		return "", "", "", errors.New("unable to resolve the credential")
	}

	credentialMap := credential.GetValue().AsMap()
	rawSubscriptionKey, ok := credentialMap[subscriptionKeyKey]
	if !ok {
		return "", "", "", errors.New("unable to resolve the credential")
	}
	subscriptionKey, ok = rawSubscriptionKey.(string)
	if !ok || strings.TrimSpace(subscriptionKey) == "" {
		return "", "", "", errors.New("unable to resolve the credential")
	}
	subscriptionKey = strings.TrimSpace(subscriptionKey)

	rawEndpoint, ok := credentialMap[endpointKey]
	if !ok {
		return "", "", "", errors.New("unable to resolve the credential")
	}
	endpoint, ok = rawEndpoint.(string)
	if !ok || strings.TrimSpace(endpoint) == "" {
		return "", "", "", errors.New("unable to resolve the credential")
	}
	endpoint = strings.TrimSpace(endpoint)

	apiVersion = defaultAPIVersion
	if rawAPIVersion, ok := credentialMap[apiVersionKey]; ok {
		if configuredAPIVersion, ok := rawAPIVersion.(string); ok && strings.TrimSpace(configuredAPIVersion) != "" {
			apiVersion = strings.TrimSpace(configuredAPIVersion)
		}
	}

	return endpoint, subscriptionKey, apiVersion, nil
}

func newClient(credential *protos.Credential) (*openai.Client, error) {
	endpoint, subscriptionKey, apiVersion, err := resolveCredential(credential)
	if err != nil {
		return nil, err
	}
	client := openai.NewClient(buildClientOptions(endpoint, subscriptionKey, apiVersion, nil)...)
	return &client, nil
}

func newStreamClient(credential *protos.Credential) (*openai.Client, idleCloser, error) {
	endpoint, subscriptionKey, apiVersion, err := resolveCredential(credential)
	if err != nil {
		return nil, nil, err
	}

	transport := &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxConnsPerHost:     internal_azure_common.StreamMaxConnsPerHost,
		MaxIdleConnsPerHost: internal_azure_common.StreamMaxIdleConnsPerHost,
		MaxIdleConns:        internal_azure_common.StreamMaxIdleConns,
		IdleConnTimeout:     internal_azure_common.StreamIdleConnTimeout,
	}
	httpClient := &http.Client{Transport: transport}

	client := openai.NewClient(buildClientOptions(endpoint, subscriptionKey, apiVersion, httpClient)...)
	return &client, httpClient, nil
}
