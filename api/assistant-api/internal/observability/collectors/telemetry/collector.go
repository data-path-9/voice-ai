// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
	pkgtelemetry "github.com/rapidaai/pkg/telemetry"
	"github.com/rapidaai/pkg/telemetry/providers"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

type Provider struct {
	Type    string
	Options map[string]interface{}
}

type Config struct {
	Logger             commons.Logger
	AppConfig          *config.AppConfig
	OpenSearch         connectors.OpenSearchConnector
	TelemetryConfig    *configs.TelemetryConfig
	Providers          []Provider
	AssistantProviders []*internal_telemetry_entity.AssistantTelemetryProvider
	Exporters          []pkgtelemetry.Exporter
}

type Collector struct {
	exporters []pkgtelemetry.Exporter
}

func New(ctx context.Context, cfg Config) (observability.Collector, error) {
	exporters := append([]pkgtelemetry.Exporter(nil), cfg.Exporters...)
	factoryDeps := providers.FactoryDependencies{
		Logger:     cfg.Logger,
		AppConfig:  cfg.AppConfig,
		OpenSearch: cfg.OpenSearch,
	}

	if validator.NonNil(cfg.TelemetryConfig) {
		providerType := cfg.TelemetryConfig.Type()
		if validator.NotBlank(string(providerType)) {
			exporter, err := providers.NewExporterFromOptions(ctx, string(providerType), cfg.TelemetryConfig.ToMap(), factoryDeps)
			if err != nil {
				return nil, err
			}
			if validator.NonNil(exporter) {
				exporters = append(exporters, exporter)
			}
		}
	}

	for _, provider := range cfg.Providers {
		exporter, err := providers.NewExporterFromOptions(ctx, provider.Type, cloneOptions(provider.Options), factoryDeps)
		if err != nil {
			return nil, err
		}
		if validator.NonNil(exporter) {
			exporters = append(exporters, exporter)
		}
	}

	for _, provider := range cfg.AssistantProviders {
		if !validator.NonNil(provider) || !provider.Enabled {
			continue
		}
		exporter, err := providers.NewExporterFromOptions(ctx, provider.ProviderType, cloneOptions(provider.GetOptions()), factoryDeps)
		if err != nil {
			return nil, err
		}
		if validator.NonNil(exporter) {
			exporters = append(exporters, exporter)
		}
	}

	if !validator.NotEmpty(exporters) {
		return observability.NoopCollector{}, nil
	}
	return &Collector{exporters: exporters}, nil
}

func MustNew(ctx context.Context, cfg Config) observability.Collector {
	collector, err := New(ctx, cfg)
	if err != nil {
		panic(err)
	}
	return collector
}

func (c *Collector) Collect(ctx context.Context, envelope observability.Envelope) error {
	if !validator.NonNil(c) || !validator.NotEmpty(c.exporters) {
		return nil
	}

	meta := sessionMeta(envelope.Scope)
	switch envelope.Kind {
	case observability.RecordKindEvent:
		return c.exportEvent(ctx, meta, envelope)
	case observability.RecordKindMetric:
		return c.exportMetric(ctx, meta, envelope)
	default:
		return nil
	}
}

func (c *Collector) Shutdown(ctx context.Context) error {
	var errs []error
	for _, exporter := range c.exporters {
		if !validator.NonNil(exporter) {
			continue
		}
		if err := exporter.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c *Collector) exportEvent(ctx context.Context, meta pkgtelemetry.SessionMeta, envelope observability.Envelope) error {
	rec := pkgtelemetry.EventRecord{
		ConversationID: envelope.Scope.ConversationID,
		MessageID:      messageID(envelope.Scope),
		Name:           envelope.Name.String(),
		Data:           eventData(envelope),
		Time:           occurredAt(envelope),
	}

	var errs []error
	for _, exporter := range c.exporters {
		if !validator.NonNil(exporter) {
			continue
		}
		if err := exporter.ExportEvent(ctx, meta, rec); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c *Collector) exportMetric(ctx context.Context, meta pkgtelemetry.SessionMeta, envelope observability.Envelope) error {
	record, ok := envelope.Record.(observability.MetricRecord)
	if !ok || !validator.NotEmpty(record.Metrics) {
		return nil
	}

	rec := metricRecord(envelope, record.Metrics)
	var errs []error
	for _, exporter := range c.exporters {
		if !validator.NonNil(exporter) {
			continue
		}
		if err := exporter.ExportMetric(ctx, meta, rec); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func sessionMeta(scope observability.Scope) pkgtelemetry.SessionMeta {
	return pkgtelemetry.SessionMeta{
		AssistantID:             scope.AssistantID,
		AssistantConversationID: scope.ConversationID,
		ProjectID:               scope.ProjectID,
		OrganizationID:          scope.OrganizationID,
	}
}

func metricRecord(envelope observability.Envelope, metrics []observability.Metric) pkgtelemetry.MetricRecord {
	converted := toProtoMetrics(metrics)
	conversationID := fmt.Sprintf("%d", envelope.Scope.ConversationID)
	if msgID := messageID(envelope.Scope); validator.NotBlank(msgID) {
		return pkgtelemetry.MessageMetricRecord{
			MessageID:      msgID,
			ConversationID: conversationID,
			Metrics:        converted,
			Time:           occurredAt(envelope),
		}
	}
	return pkgtelemetry.ConversationMetricRecord{
		ConversationID: conversationID,
		Metrics:        converted,
		Time:           occurredAt(envelope),
	}
}

func toProtoMetrics(metrics []observability.Metric) []*protos.Metric {
	converted := make([]*protos.Metric, 0, len(metrics))
	for _, metric := range metrics {
		converted = append(converted, &protos.Metric{
			Name:        metric.Name,
			Value:       metric.Value,
			Description: metric.Description,
		})
	}
	return converted
}

func eventData(envelope observability.Envelope) map[string]string {
	data := make(map[string]string, len(envelope.Attributes)+len(envelope.Data))
	for key, value := range envelope.Attributes {
		data[key] = value
	}
	for key, value := range envelope.Data {
		data[key] = fmt.Sprintf("%v", value)
	}
	return data
}

func messageID(scope observability.Scope) string {
	// pkg/telemetry still names this field MessageID; observability uses ContextID.
	return scope.ContextID
}

func occurredAt(envelope observability.Envelope) time.Time {
	if !envelope.OccurredAt.IsZero() {
		return envelope.OccurredAt
	}
	return envelope.ReceivedAt
}

func cloneOptions(options map[string]interface{}) map[string]interface{} {
	if len(options) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(options))
	for key, value := range options {
		cloned[key] = value
	}
	return cloned
}
