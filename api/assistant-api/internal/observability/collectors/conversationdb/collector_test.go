// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package conversationdb

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/commons"
	"gorm.io/gorm"
)

type postgresStub struct{}

func (postgresStub) Connect(context.Context) error {
	return nil
}

func (postgresStub) Name() string {
	return "postgres-stub"
}

func (postgresStub) IsConnected(context.Context) bool {
	return true
}

func (postgresStub) Disconnect(context.Context) error {
	return nil
}

func (postgresStub) Query(context.Context, string, interface{}) error {
	return nil
}

func (postgresStub) DB(context.Context) *gorm.DB {
	return nil
}

func TestNew_RequiresConcreteDeps(t *testing.T) {
	_, err := New(Config{Postgres: postgresStub{}})
	if !errors.Is(err, ErrLoggerRequired) {
		t.Fatalf("expected logger error, got %v", err)
	}

	_, err = New(Config{Logger: testLogger(t)})
	if !errors.Is(err, ErrPostgresRequired) {
		t.Fatalf("expected postgres error, got %v", err)
	}

	collector, err := New(Config{
		Logger:    testLogger(t),
		Postgres:  postgresStub{},
		CreatedBy: 99,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if collector == nil {
		t.Fatal("expected collector")
	}
}

func TestCollect_IgnoresNonDBRecords(t *testing.T) {
	collector := &Collector{}
	err := collector.Collect(context.Background(), observability.Envelope{
		Kind: observability.RecordKindEvent,
		Name: observability.CallRinging,
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
}

func TestCollect_RequiresScopeForNonEmptyDBRecords(t *testing.T) {
	collector, err := New(Config{Logger: testLogger(t), Postgres: postgresStub{}})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	err = collector.Collect(context.Background(), observability.Envelope{
		Kind:  observability.RecordKindMetric,
		Scope: observability.Scope{AssistantID: 10},
		Record: observability.MetricRecord{Metrics: []observability.Metric{
			{Name: observability.MetricConversationDuration, Value: "1000"},
		}},
	})
	if !errors.Is(err, ErrScopeRequired) {
		t.Fatalf("expected scope error, got %v", err)
	}

	err = collector.Collect(context.Background(), observability.Envelope{
		Kind:  observability.RecordKindMetadata,
		Scope: observability.Scope{ConversationID: 20},
		Record: observability.MetadataRecord{Metadata: []observability.Metadata{
			{Key: observability.MetadataDisconnectReason, Value: "normal_clearing"},
		}},
	})
	if !errors.Is(err, ErrScopeRequired) {
		t.Fatalf("expected scope error, got %v", err)
	}
}

func TestCollect_EmptyDBRecordsAreNoop(t *testing.T) {
	collector := &Collector{}
	if err := collector.Collect(context.Background(), observability.Envelope{
		Kind:   observability.RecordKindMetric,
		Record: observability.MetricRecord{},
	}); err != nil {
		t.Fatalf("empty metrics should be no-op, got %v", err)
	}
	if err := collector.Collect(context.Background(), observability.Envelope{
		Kind:   observability.RecordKindMetadata,
		Record: observability.MetadataRecord{},
	}); err != nil {
		t.Fatalf("empty metadata should be no-op, got %v", err)
	}
}

func TestAuthUsesScopeAndCreatedBy(t *testing.T) {
	collector, err := New(Config{
		Logger:    testLogger(t),
		Postgres:  postgresStub{},
		CreatedBy: 99,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	auth := collector.(*Collector).auth(observability.Scope{OrganizationID: 1, ProjectID: 2})
	if auth.GetUserId() == nil || *auth.GetUserId() != 99 {
		t.Fatalf("unexpected auth user: %v", auth.GetUserId())
	}
	if auth.GetCurrentOrganizationId() == nil || *auth.GetCurrentOrganizationId() != 1 {
		t.Fatalf("unexpected auth organization: %v", auth.GetCurrentOrganizationId())
	}
	if auth.GetCurrentProjectId() == nil || *auth.GetCurrentProjectId() != 2 {
		t.Fatalf("unexpected auth project: %v", auth.GetCurrentProjectId())
	}
}

func TestConversionToServiceTypes(t *testing.T) {
	metrics := toServiceMetrics([]observability.Metric{{
		Name:        observability.MetricConversationDuration,
		Value:       "1000",
		Description: "duration",
	}})
	if len(metrics) != 1 || metrics[0].Name != observability.MetricConversationDuration || metrics[0].Description != "duration" {
		t.Fatalf("unexpected service metrics: %+v", metrics)
	}

	metadata := toServiceMetadata([]observability.Metadata{{
		Key:   observability.MetadataDisconnectReason,
		Value: "normal_clearing",
	}})
	if len(metadata) != 1 || metadata[0].Key != observability.MetadataDisconnectReason || metadata[0].Value != "normal_clearing" {
		t.Fatalf("unexpected service metadata: %+v", metadata)
	}
}

func testLogger(t *testing.T) commons.Logger {
	t.Helper()

	logger, err := commons.NewApplicationLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}
