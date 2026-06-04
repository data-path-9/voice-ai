// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package collectors

import (
	"context"
	"errors"

	assistant_config "github.com/rapidaai/api/assistant-api/config"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/telemetry"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/timeline"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors/webhook"
	app_config "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/validator"
)

type DefaultConfig struct {
	Logger                     commons.Logger
	AssistantConfig            *assistant_config.AssistantConfig
	OpenSearch                 connectors.OpenSearchConnector
	AssistantTelemetryProvider []*internal_telemetry_entity.AssistantTelemetryProvider
	AssistantWebhooks          []*internal_assistant_entity.AssistantWebhook
}

func AppendDefault(ctx context.Context, out []observability.Collector, cfg DefaultConfig) ([]observability.Collector, error) {
	if err := ValidateDefaultConfig(cfg); err != nil {
		return nil, err
	}

	if DefaultTelemetryEnabled(cfg.AssistantConfig) || hasEnabledAssistantTelemetryProvider(cfg.AssistantTelemetryProvider) {
		collector, err := telemetry.New(ctx, telemetry.Config{
			Logger:             cfg.Logger,
			AppConfig:          appConfig(cfg.AssistantConfig),
			OpenSearch:         cfg.OpenSearch,
			TelemetryConfig:    telemetryConfig(cfg.AssistantConfig),
			AssistantProviders: cfg.AssistantTelemetryProvider,
		})
		if err != nil {
			return nil, err
		}
		out = appendActive(out, collector)
	}

	if OpenSearchTimelineEnabled(cfg) {
		out = appendActive(out, timeline.New(timeline.Config{
			Logger:      cfg.Logger,
			OpenSearch:  cfg.OpenSearch,
			IndexPrefix: timelineIndexPrefix(cfg.AssistantConfig),
		}))
	}
	out = appendActive(out, webhook.New(webhook.Config{
		Logger:   cfg.Logger,
		Webhooks: cfg.AssistantWebhooks,
	}))
	return out, nil
}

func NewDefault(ctx context.Context, cfg DefaultConfig) ([]observability.Collector, error) {
	return AppendDefault(ctx, nil, cfg)
}

func ValidateDefaultConfig(cfg DefaultConfig) error {
	for _, provider := range cfg.AssistantTelemetryProvider {
		if !validator.NonNil(provider) || !provider.Enabled {
			continue
		}
		if !validator.NotBlank(provider.ProviderType) {
			return errors.New("observability collectors: assistant telemetry provider type is required")
		}
	}

	if requiresLogger(cfg) && !validator.NonNil(cfg.Logger) {
		return errors.New("observability collectors: logger is required")
	}
	if requiresOpenSearch(cfg) && !validator.NonNil(cfg.OpenSearch) {
		return errors.New("observability collectors: opensearch connector is required")
	}
	return nil
}

func DefaultTelemetryEnabled(cfg *assistant_config.AssistantConfig) bool {
	tel := telemetryConfig(cfg)
	return validator.NonNil(tel) &&
		validator.NotBlank(tel.TelemetryType) &&
		validator.NotBlank(string(tel.Type()))
}

func DefaultOpenSearchTelemetryEnabled(cfg *assistant_config.AssistantConfig) bool {
	tel := telemetryConfig(cfg)
	return validator.NonNil(tel) &&
		validator.NotBlank(tel.TelemetryType) &&
		tel.Type() == configs.OPENSEARCH
}

func OpenSearchTimelineEnabled(cfg DefaultConfig) bool {
	return validator.NonNil(cfg.OpenSearch) &&
		(DefaultOpenSearchTelemetryEnabled(cfg.AssistantConfig) ||
			hasEnabledProviderType(cfg.AssistantTelemetryProvider, string(configs.OPENSEARCH)))
}

func appConfig(cfg *assistant_config.AssistantConfig) *app_config.AppConfig {
	if !validator.NonNil(cfg) {
		return nil
	}
	return &cfg.AppConfig
}

func telemetryConfig(cfg *assistant_config.AssistantConfig) *configs.TelemetryConfig {
	if !validator.NonNil(cfg) {
		return nil
	}
	return cfg.TelemetryConfig
}

func timelineIndexPrefix(cfg *assistant_config.AssistantConfig) string {
	tel := telemetryConfig(cfg)
	if !DefaultOpenSearchTelemetryEnabled(cfg) || !validator.NonNil(tel.OpenSearch) {
		return ""
	}
	return tel.OpenSearch.IndexPrefix
}

func appendActive(out []observability.Collector, collector observability.Collector) []observability.Collector {
	if !validator.NonNil(collector) {
		return out
	}
	if _, ok := collector.(observability.NoopCollector); ok {
		return out
	}
	return append(out, collector)
}

func hasEnabledAssistantTelemetryProvider(providers []*internal_telemetry_entity.AssistantTelemetryProvider) bool {
	for _, provider := range providers {
		if validator.NonNil(provider) && provider.Enabled && validator.NotBlank(provider.ProviderType) {
			return true
		}
	}
	return false
}

func hasEnabledProviderType(providers []*internal_telemetry_entity.AssistantTelemetryProvider, providerType string) bool {
	for _, provider := range providers {
		if validator.NonNil(provider) &&
			provider.Enabled &&
			validator.NotBlank(provider.ProviderType) &&
			provider.ProviderType == providerType {
			return true
		}
	}
	return false
}

func requiresLogger(cfg DefaultConfig) bool {
	return defaultProviderType(cfg.AssistantConfig, string(configs.LOGGING)) ||
		defaultProviderType(cfg.AssistantConfig, string(configs.OPENSEARCH)) ||
		hasEnabledProviderType(cfg.AssistantTelemetryProvider, string(configs.LOGGING)) ||
		hasEnabledProviderType(cfg.AssistantTelemetryProvider, string(configs.OPENSEARCH))
}

func requiresOpenSearch(cfg DefaultConfig) bool {
	return defaultProviderType(cfg.AssistantConfig, string(configs.OPENSEARCH)) ||
		hasEnabledProviderType(cfg.AssistantTelemetryProvider, string(configs.OPENSEARCH))
}

func defaultProviderType(cfg *assistant_config.AssistantConfig, providerType string) bool {
	tel := telemetryConfig(cfg)
	return validator.NonNil(tel) &&
		validator.NotBlank(tel.TelemetryType) &&
		tel.Type() == configs.TelemetryType(providerType)
}
