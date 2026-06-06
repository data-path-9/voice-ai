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

	"github.com/rapidaai/api/assistant-api/internal/observability"
	telemetry "github.com/rapidaai/pkg/telemetry"
	"github.com/rapidaai/protos"
)

type exporterStub struct {
	events  []telemetry.EventRecord
	metrics []telemetry.MetricRecord
	logs    []telemetry.LogRecord
	closed  bool
	err     error
}

func (e *exporterStub) Export(_ context.Context, rec telemetry.Record) error {
	switch typed := rec.(type) {
	case telemetry.LogRecord:
		e.logs = append(e.logs, typed)
	case telemetry.EventRecord:
		e.events = append(e.events, typed)
	case telemetry.MetricRecord:
		e.metrics = append(e.metrics, typed)
	}
	return e.err
}

func (e *exporterStub) Close(context.Context) error {
	e.closed = true
	return e.err
}

func TestNew_ReturnsNoopWithoutExporter(t *testing.T) {
	collector, err := New(context.Background(), Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected noop collector, got %T", collector)
	}
}

func TestCollector_ExportsEventsAndMetrics(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: exporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)

	err = collector.Collect(context.Background(), observability.RecordEvent{
		CommonRecord: observability.CommonRecord{
			Scope: observability.ConversationScope{
				AssistantScope: observability.AssistantScope{
					GlobalScope: observability.GlobalScope{
						ProjectID:      30,
						OrganizationID: 40,
					},
					AssistantID: 10,
				},
				ConversationID: 20,
			},
			OccurredAt: now,
		},
		Component:  observability.ComponentCall,
		Event:      observability.CallRinging,
		Attributes: observability.Attributes{"status": "ringing"},
	})
	if err != nil {
		t.Fatalf("CollectEvent returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.RecordMetric{
		CommonRecord: observability.CommonRecord{
			Scope: observability.ConversationScope{
				AssistantScope: observability.AssistantScope{
					GlobalScope: observability.GlobalScope{
						ProjectID:      30,
						OrganizationID: 40,
					},
					AssistantID: 10,
				},
				ConversationID: 20,
			},
			OccurredAt: now,
		},
		Metrics: []*protos.Metric{{
			Name:        observability.MetricConversationDuration,
			Value:       "1000",
			Description: "duration",
		}},
	})
	if err != nil {
		t.Fatalf("CollectMetric returned error: %v", err)
	}

	if len(exporter.events) != 1 {
		t.Fatalf("expected one event, got %d", len(exporter.events))
	}
	event := exporter.events[0]
	if event.Event != observability.CallRinging.String() || !event.OccurredAt.Equal(now) {
		t.Fatalf("unexpected event record: %+v", event)
	}
	if event.Component != observability.ComponentCall.String() {
		t.Fatalf("unexpected event component: %+v", event)
	}
	if event.ProjectID != 30 || event.OrganizationID != 40 ||
		event.ScopeAttributes["assistantId"] != "10" ||
		event.ScopeAttributes["assistantConversationId"] != "20" {
		t.Fatalf("unexpected event scope: %+v", event)
	}
	if event.Attributes["status"] != "ringing" {
		t.Fatalf("unexpected event attributes: %+v", event.Attributes)
	}

	if len(exporter.metrics) != 1 {
		t.Fatalf("expected one metric, got %d", len(exporter.metrics))
	}
	metric := exporter.metrics[0]
	if metric.ScopeAttributes["assistantConversationId"] != "20" ||
		metric.Name != observability.MetricConversationDuration ||
		metric.Value != "1000" {
		t.Fatalf("unexpected metric record: %+v", metric)
	}
}

func TestCollector_ExportsMessageMetrics(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: exporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.RecordMetric{
		CommonRecord: observability.CommonRecord{
			Scope: observability.MessageScope{
				ConversationScope: observability.ConversationScope{
					AssistantScope: observability.AssistantScope{AssistantID: 10},
					ConversationID: 20,
				},
				MessageID: "user-ctx-1",
				Role:      observability.MessageRoleUser,
			},
		},
		Metrics: []*protos.Metric{{
			Name:  observability.MetricUserTurn,
			Value: "complete",
		}},
	})
	if err != nil {
		t.Fatalf("CollectMetric returned error: %v", err)
	}
	metric := exporter.metrics[0]
	if metric.ScopeAttributes["messageId"] != "user-ctx-1" ||
		metric.ScopeAttributes["messageRole"] != "user" ||
		metric.ScopeAttributes["assistantConversationId"] != "20" {
		t.Fatalf("unexpected message metric: %+v", metric)
	}
}

func TestCollector_ReturnsExporterErrors(t *testing.T) {
	exportErr := errors.New("export failed")
	collector, err := New(context.Background(), Config{Exporters: &exporterStub{err: exportErr}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.RecordEvent{
		CommonRecord: observability.CommonRecord{
			Scope: observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: 10},
				ConversationID: 20,
			},
		},
		Event: observability.CallRinging,
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("expected export error, got %v", err)
	}
}

func TestCollector_CloseExporter(t *testing.T) {
	exporter := &exporterStub{}
	collector, err := New(context.Background(), Config{Exporters: exporter})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if err := collector.Close(context.Background()); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !exporter.closed {
		t.Fatal("expected exporter to close")
	}
}

var _ telemetry.Exporter = (*exporterStub)(nil)
