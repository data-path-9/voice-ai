// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package telemetry

import (
	"context"
	"errors"
	"testing"
	"time"

	internal_telemetry_entity "github.com/rapidaai/api/assistant-api/internal/entity/telemetry"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/configs"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	pkgtelemetry "github.com/rapidaai/pkg/telemetry"
)

type exporterStub struct {
	events  []pkgtelemetry.EventRecord
	metrics []pkgtelemetry.MetricRecord
	closed  bool
	err     error
}

func (e *exporterStub) ExportEvent(_ context.Context, _ pkgtelemetry.SessionMeta, rec pkgtelemetry.EventRecord) error {
	e.events = append(e.events, rec)
	return e.err
}

func (e *exporterStub) ExportMetric(_ context.Context, _ pkgtelemetry.SessionMeta, rec pkgtelemetry.MetricRecord) error {
	e.metrics = append(e.metrics, rec)
	return e.err
}

func (e *exporterStub) Shutdown(context.Context) error {
	e.closed = true
	return e.err
}

func TestNew_ReturnsNoopWithoutExporters(t *testing.T) {
	collector, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected noop collector, got %T", collector)
	}
}

func TestNew_BuildsEnvAndAssistantProviderExporters(t *testing.T) {
	collector, err := New(context.Background(), Config{
		Logger: testLogger(t),
		TelemetryConfig: &configs.TelemetryConfig{
			TelemetryType: string(configs.LOGGING),
			Logging:       &configs.TelemetryLoggingConfig{},
		},
		AssistantProviders: []*internal_telemetry_entity.AssistantTelemetryProvider{
			{ProviderType: string(pkgtelemetry.LOGGING), Enabled: true},
			{ProviderType: string(pkgtelemetry.LOGGING), Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	got := collector.(*Collector)
	if len(got.exporters) != 2 {
		t.Fatalf("expected env and enabled assistant exporters, got %d", len(got.exporters))
	}
}

func TestNew_ReturnsFactoryError(t *testing.T) {
	_, err := New(context.Background(), Config{
		Providers: []Provider{{Type: string(pkgtelemetry.LOGGING)}},
	})
	if err == nil {
		t.Fatal("expected missing logger error")
	}
}

func TestCollector_ExportsEventsAndMetrics(t *testing.T) {
	first := &exporterStub{}
	second := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: []pkgtelemetry.Exporter{first, nil, second}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	now := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)

	err = collector.Collect(context.Background(), observability.Envelope{
		Kind:       observability.RecordKindEvent,
		Name:       observability.CallRinging,
		Scope:      observability.Scope{AssistantID: 10, ConversationID: 20, ContextID: "ctx-1"},
		Attributes: observability.Attributes{"status": "ringing"},
		Data:       observability.Data{"raw_status": "RINGING"},
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("event Collect returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.Envelope{
		Kind: observability.RecordKindMetric,
		Scope: observability.Scope{
			AssistantID:    10,
			ConversationID: 20,
		},
		Record: observability.MetricRecord{Metrics: []observability.Metric{{
			Name:        observability.MetricConversationDuration,
			Value:       "1000",
			Description: "duration",
		}}},
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("metric Collect returned error: %v", err)
	}

	if len(first.events) != 1 || len(second.events) != 1 {
		t.Fatalf("expected both exporters to receive event, got first=%d second=%d", len(first.events), len(second.events))
	}
	event := first.events[0]
	if event.Name != observability.CallRinging.String() || event.MessageID != "ctx-1" || !event.Time.Equal(now) {
		t.Fatalf("unexpected event record: %+v", event)
	}
	if event.Data["status"] != "ringing" || event.Data["raw_status"] != "RINGING" {
		t.Fatalf("unexpected event data: %+v", event.Data)
	}

	if len(first.metrics) != 1 || len(second.metrics) != 1 {
		t.Fatalf("expected both exporters to receive metric, got first=%d second=%d", len(first.metrics), len(second.metrics))
	}
	metric, ok := first.metrics[0].(pkgtelemetry.ConversationMetricRecord)
	if !ok {
		t.Fatalf("expected conversation metric record, got %T", first.metrics[0])
	}
	if metric.ConversationID != "20" || len(metric.Metrics) != 1 || metric.Metrics[0].Name != observability.MetricConversationDuration {
		t.Fatalf("unexpected metric record: %+v", metric)
	}
}

func TestCollector_ExportsMessageMetrics(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: []pkgtelemetry.Exporter{exporter}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.Envelope{
		Kind:  observability.RecordKindMetric,
		Scope: observability.Scope{ConversationID: 20, ContextID: "ctx-1"},
		Record: observability.MetricRecord{Metrics: []observability.Metric{{
			Name:  observability.MetricUserTurn,
			Value: "complete",
		}}},
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	metric, ok := exporter.metrics[0].(pkgtelemetry.MessageMetricRecord)
	if !ok {
		t.Fatalf("expected message metric record, got %T", exporter.metrics[0])
	}
	if metric.MessageID != "ctx-1" || metric.ConversationID != "20" {
		t.Fatalf("unexpected message metric: %+v", metric)
	}
}

func TestCollector_IgnoresMetadata(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: []pkgtelemetry.Exporter{exporter}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.Envelope{Kind: observability.RecordKindMetadata})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(exporter.events) != 0 || len(exporter.metrics) != 0 {
		t.Fatalf("expected metadata to be ignored, got events=%d metrics=%d", len(exporter.events), len(exporter.metrics))
	}
}

func TestCollector_JoinsExporterErrors(t *testing.T) {
	exportErr := errors.New("export failed")
	collector, err := New(context.Background(), Config{Exporters: []pkgtelemetry.Exporter{&exporterStub{err: exportErr}}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.Envelope{
		Kind: observability.RecordKindEvent,
		Name: observability.CallRinging,
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("expected export error, got %v", err)
	}
}

func TestCollector_ShutdownExporters(t *testing.T) {
	first := &exporterStub{}
	second := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: []pkgtelemetry.Exporter{first, second}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := collector.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	if !first.closed || !second.closed {
		t.Fatalf("expected exporters to close, got first=%t second=%t", first.closed, second.closed)
	}
}

func TestCloneOptions(t *testing.T) {
	options := map[string]interface{}{"endpoint": "localhost:4318"}
	cloned := cloneOptions(options)
	cloned["endpoint"] = "changed"
	if options["endpoint"] != "localhost:4318" {
		t.Fatalf("source options mutated: %+v", options)
	}
}

func TestAssistantProviderOptionsAreCloned(t *testing.T) {
	provider := &internal_telemetry_entity.AssistantTelemetryProvider{
		ProviderType: string(pkgtelemetry.LOGGING),
		Enabled:      true,
		Options: []*internal_telemetry_entity.AssistantTelemetryProviderOption{
			{Metadata: gorm_models.Metadata{Key: "x", Value: "y"}},
		},
	}
	options := cloneOptions(provider.GetOptions())
	options["x"] = "changed"
	if provider.GetOptions()["x"] != "y" {
		t.Fatalf("provider options mutated")
	}
}

func testLogger(t *testing.T) commons.Logger {
	t.Helper()

	logger, err := commons.NewApplicationLogger(
		commons.Name("observability-telemetry-test"),
		commons.Level("error"),
		commons.EnableFile(false),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

var _ pkgtelemetry.Exporter = (*exporterStub)(nil)
