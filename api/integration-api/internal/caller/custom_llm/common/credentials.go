// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_common

import (
	"errors"
	"fmt"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func ParseClientConfig(
	logger commons.Logger,
	credential *protos.Credential,
) (ClientConfig, error) {
	if credential == nil || credential.GetValue() == nil {
		return ClientConfig{}, errors.New("custom-llm: unable to resolve credential")
	}

	raw := credential.GetValue().AsMap()
	compatibility, err := parseCompatibility(raw)
	if err != nil {
		return ClientConfig{}, err
	}
	baseURL, err := parseBaseURL(raw)
	if err != nil {
		return ClientConfig{}, err
	}
	headers, err := parseHeaders(raw)
	if err != nil {
		return ClientConfig{}, err
	}

	cfg := ClientConfig{
		Compatibility:  compatibility,
		BaseURL:        baseURL,
		Headers:        headers,
		ClientTimeout:  DefaultClientTimeout,
		RequestTimeout: DefaultRequestTimeout,
	}

	logger.Infof("custom-llm: resolved compatibility %s", cfg.Compatibility)
	return cfg, nil
}

func ResolveCompatibility(credential *protos.Credential) (Compatibility, error) {
	if credential == nil || credential.GetValue() == nil {
		return DefaultCompatibility, nil
	}
	return parseCompatibility(credential.GetValue().AsMap())
}

func parseCompatibility(credentials map[string]any) (Compatibility, error) {
	compatibility := DefaultCompatibility
	rawCompatibility, found := credentials[CredentialKeyAPICompatibilitySnake]
	if !found {
		rawCompatibility, found = credentials[CredentialKeyAPICompatibilityCamel]
	}
	if found {
		compatibilityStr, ok := rawCompatibility.(string)
		if !ok {
			return "", errors.New("custom-llm: api compatibility must be a string")
		}
		if compatibilityStr == "" {
			return "", errors.New("custom-llm: api compatibility must not be empty")
		}
		compatibility = Compatibility(compatibilityStr)
	}
	return compatibility, nil
}

func parseBaseURL(credentials map[string]any) (string, error) {
	baseURLRaw, found := credentials[CredentialKeyBaseURLSnake]
	if !found {
		baseURLRaw, found = credentials[CredentialKeyBaseURLCamel]
	}
	if !found {
		return "", errors.New("custom-llm: base url must be specified in credentials")
	}
	baseURL, ok := baseURLRaw.(string)
	if !ok {
		return "", errors.New("custom-llm: base url must be a string")
	}
	if baseURL == "" {
		return "", errors.New("custom-llm: base url must not be empty")
	}
	return baseURL, nil
}

func parseHeaders(credentials map[string]any) (map[string]string, error) {
	headersRaw, found := credentials[CredentialKeyHeaders]
	if !found {
		return map[string]string{}, nil
	}
	headers, err := utils.Option{CredentialKeyHeaders: headersRaw}.GetStringMap(CredentialKeyHeaders)
	if err != nil {
		return nil, fmt.Errorf("custom-llm: invalid headers: %w", err)
	}
	if headers == nil {
		return map[string]string{}, nil
	}
	return headers, nil
}
