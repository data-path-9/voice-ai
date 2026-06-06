// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package telemetry

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	telemetry "github.com/rapidaai/pkg/telemetry"
	"github.com/rapidaai/pkg/telemetry/providers"
	"github.com/rapidaai/pkg/validator"
)

type Provider struct {
	Name    string
	Options map[string]interface{}
}

type Config struct {
	Logger    commons.Logger
	Providers Provider
	Exporters telemetry.Exporter
}

type Collector struct {
	exporter telemetry.Exporter
}

func New(ctx context.Context, cfg Config) (observability.Collector, error) {
	if validator.NonNil(cfg.Exporters) {
		return &Collector{exporter: cfg.Exporters}, nil
	}
	exporter, err := newExporter(ctx, cfg.Logger, cfg.Providers)
	if err != nil {
		return nil, err
	}
	if !validator.NonNil(exporter) {
		return observability.NoopCollector{}, nil
	}
	return &Collector{exporter: exporter}, nil
}

func (c *Collector) Collect(ctx context.Context, record observability.Record) error {
	if !validator.NonNil(c.exporter) {
		return nil
	}
	switch typed := record.(type) {
	case observability.RecordLog:
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		attributes := make(map[string]string, len(typed.Attributes))
		for key, value := range typed.Attributes {
			attributes[key] = value
		}
		global := typed.Scope.GlobalScopeValue()
		scopeAttributes := map[string]string{
			"assistantId": strconv.FormatUint(typed.Scope.AssistantScopeID(), 10),
		}
		switch typed.Scope.ScopeType() {
		case observability.ScopeConversation:
			scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.Scope.ConversationScopeID(), 10)
		case observability.ScopeMessage:
			scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.Scope.ConversationScopeID(), 10)
			scopeAttributes["messageId"] = typed.Scope.ContextID()
			scopeAttributes["messageRole"] = string(typed.Scope.MessageScopeRole())
		}
		return c.exporter.Export(ctx, telemetry.LogRecord{
			CommonRecord: telemetry.CommonRecord{
				ID:              typed.ID,
				ProjectID:       global.ProjectID,
				OrganizationID:  global.OrganizationID,
				Scope:           string(typed.Scope.ScopeType()),
				ScopeAttributes: scopeAttributes,
				Attributes:      attributes,
				OccurredAt:      occurredAt,
			},
			Level:   string(typed.Level),
			Message: typed.Message,
		})
	case observability.RecordEvent:
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		attributes := make(map[string]string, len(typed.Attributes))
		for key, value := range typed.Attributes {
			attributes[key] = value
		}
		global := typed.Scope.GlobalScopeValue()
		scopeAttributes := map[string]string{
			"assistantId": strconv.FormatUint(typed.Scope.AssistantScopeID(), 10),
		}
		switch typed.Scope.ScopeType() {
		case observability.ScopeConversation:
			scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.Scope.ConversationScopeID(), 10)
		case observability.ScopeMessage:
			scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.Scope.ConversationScopeID(), 10)
			scopeAttributes["messageId"] = typed.Scope.ContextID()
			scopeAttributes["messageRole"] = string(typed.Scope.MessageScopeRole())
		}
		return c.exporter.Export(ctx, telemetry.EventRecord{
			CommonRecord: telemetry.CommonRecord{
				ID:              typed.ID,
				ProjectID:       global.ProjectID,
				OrganizationID:  global.OrganizationID,
				Scope:           string(typed.Scope.ScopeType()),
				ScopeAttributes: scopeAttributes,
				Attributes:      attributes,
				OccurredAt:      occurredAt,
			},
			Event:     typed.Event.String(),
			Component: typed.Component.String(),
		})
	case observability.RecordMetric:
		if !validator.NotEmpty(typed.Metrics) {
			return nil
		}
		occurredAt := typed.OccurredAt
		if occurredAt.IsZero() {
			occurredAt = time.Now()
		}
		global := typed.Scope.GlobalScopeValue()
		scopeAttributes := map[string]string{
			"assistantId": strconv.FormatUint(typed.Scope.AssistantScopeID(), 10),
		}
		switch typed.Scope.ScopeType() {
		case observability.ScopeConversation:
			scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.Scope.ConversationScopeID(), 10)
		case observability.ScopeMessage:
			scopeAttributes["assistantConversationId"] = strconv.FormatUint(typed.Scope.ConversationScopeID(), 10)
			scopeAttributes["messageId"] = typed.Scope.ContextID()
			scopeAttributes["messageRole"] = string(typed.Scope.MessageScopeRole())
		}
		var errs []error
		for _, metric := range typed.Metrics {
			if metric == nil {
				continue
			}
			if err := c.exporter.Export(ctx, telemetry.MetricRecord{
				CommonRecord: telemetry.CommonRecord{
					ID:              typed.ID,
					ProjectID:       global.ProjectID,
					OrganizationID:  global.OrganizationID,
					Scope:           string(typed.Scope.ScopeType()),
					ScopeAttributes: scopeAttributes,
					OccurredAt:      occurredAt,
				},
				Name:        metric.GetName(),
				Value:       metric.GetValue(),
				Description: metric.GetDescription(),
			}); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	default:
		return nil
	}
}

func (c *Collector) Close(ctx context.Context) error {
	var errs []error
	if validator.NonNil(c.exporter) {
		if err := c.exporter.Close(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func newExporter(ctx context.Context, logger commons.Logger, provider Provider) (telemetry.Exporter, error) {
	providerName := strings.TrimSpace(provider.Name)
	if !validator.NotBlank(providerName) {
		return nil, nil
	}
	if len(provider.Options) == 0 {
		return providers.NewExporterFromOptions(logger, ctx, providerName, nil)
	}
	options := make(map[string]interface{}, len(provider.Options))
	for key, value := range provider.Options {
		options[key] = value
	}
	return providers.NewExporterFromOptions(logger, ctx, providerName, options)
}
