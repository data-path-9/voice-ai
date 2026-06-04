// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package timeline

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/connectors"
)

type openSearchStub struct {
	bodies []string
	err    error
}

func (o *openSearchStub) Connect(context.Context) error {
	return nil
}

func (o *openSearchStub) Name() string {
	return "opensearch-stub"
}

func (o *openSearchStub) IsConnected(context.Context) bool {
	return true
}

func (o *openSearchStub) Disconnect(context.Context) error {
	return nil
}

func (o *openSearchStub) VectorSearch(context.Context, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}

func (o *openSearchStub) HybridSearch(context.Context, string, string, []float64, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}

func (o *openSearchStub) TextSearch(context.Context, string, string, map[string]interface{}, *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	return nil, nil
}

func (o *openSearchStub) Search(context.Context, []string, string) *connectors.SearchResponse {
	return nil
}

func (o *openSearchStub) SearchWithCount(context.Context, []string, string) *connectors.SearchResponseWithCount {
	return nil
}

func (o *openSearchStub) Persist(context.Context, string, string, string) error {
	return nil
}

func (o *openSearchStub) Update(context.Context, string, string, string) error {
	return nil
}

func (o *openSearchStub) Bulk(_ context.Context, body string) error {
	o.bodies = append(o.bodies, body)
	return o.err
}

func TestNew_ReturnsNoopWithoutOpenSearch(t *testing.T) {
	collector := New(Config{})
	if _, ok := collector.(observability.NoopCollector); !ok {
		t.Fatalf("expected noop collector, got %T", collector)
	}
}

func TestCollector_PushesTimelineEnvelopeToOpenSearchBulk(t *testing.T) {
	opensearch := &openSearchStub{}
	collector := New(Config{OpenSearch: opensearch})
	now := time.Date(2026, 6, 4, 12, 30, 0, 0, time.UTC)

	err := collector.Collect(context.Background(), observability.Envelope{
		ID:       "evt-1",
		Kind:     observability.RecordKindEvent,
		Name:     observability.CallRinging,
		Category: observability.CategoryCall,
		Level:    observability.LevelInfo,
		Outcome:  observability.OutcomeSuccess,
		Title:    "Call ringing",
		Scope: observability.Scope{
			OrganizationID: 1,
			ProjectID:      2,
			AssistantID:    3,
			ConversationID: 4,
			ContextID:      "ctx-1",
		},
		Attributes: observability.Attributes{"status": "ringing"},
		Data:       observability.Data{"raw_status": "RINGING"},
		OccurredAt: now,
		ReceivedAt: now.Add(time.Second),
		Duration:   2 * time.Second,
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(opensearch.bodies) != 1 {
		t.Fatalf("expected one bulk body, got %d", len(opensearch.bodies))
	}

	lines := strings.Split(strings.TrimSpace(opensearch.bodies[0]), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected bulk metadata and document lines, got %d: %q", len(lines), opensearch.bodies[0])
	}
	if !strings.Contains(lines[0], `"rapida-timeline-20260604"`) || !strings.Contains(lines[0], `"evt-1"`) {
		t.Fatalf("unexpected bulk metadata line: %s", lines[0])
	}

	var doc document
	if err := json.Unmarshal([]byte(lines[1]), &doc); err != nil {
		t.Fatalf("failed to unmarshal document: %v", err)
	}
	if doc.ID != "evt-1" || doc.Name != observability.CallRinging.String() || doc.Kind != string(observability.RecordKindEvent) {
		t.Fatalf("unexpected document identity: %+v", doc)
	}
	if doc.OrganizationID != 1 || doc.ProjectID != 2 || doc.AssistantID != 3 || doc.AssistantConversationID != 4 {
		t.Fatalf("unexpected document scope: %+v", doc)
	}
	if doc.Attributes["status"] != "ringing" || doc.Data["raw_status"] != "RINGING" {
		t.Fatalf("unexpected document payload: %+v %+v", doc.Attributes, doc.Data)
	}
	if doc.DurationMs != 2000 {
		t.Fatalf("expected duration 2000ms, got %d", doc.DurationMs)
	}
}

func TestCollector_UsesCustomIndexPrefix(t *testing.T) {
	opensearch := &openSearchStub{}
	collector := New(Config{OpenSearch: opensearch, IndexPrefix: "custom-timeline"})
	now := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)

	if err := collector.Collect(context.Background(), observability.Envelope{ID: "evt-1", OccurredAt: now}); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if !strings.Contains(opensearch.bodies[0], `"custom-timeline-20260604"`) {
		t.Fatalf("unexpected bulk body: %s", opensearch.bodies[0])
	}
}

func TestCollector_ReturnsBulkError(t *testing.T) {
	bulkErr := errors.New("bulk failed")
	collector := New(Config{OpenSearch: &openSearchStub{err: bulkErr}})

	err := collector.Collect(context.Background(), observability.Envelope{ID: "evt-1"})
	if !errors.Is(err, bulkErr) {
		t.Fatalf("expected bulk error, got %v", err)
	}
}

func TestNewDocumentFallsBackToReceivedAt(t *testing.T) {
	receivedAt := time.Date(2026, 6, 4, 1, 2, 3, 0, time.UTC)
	doc := newDocument(observability.Envelope{ReceivedAt: receivedAt})
	if !doc.OccurredAt.Equal(receivedAt) {
		t.Fatalf("expected occurredAt fallback to receivedAt, got %s", doc.OccurredAt)
	}
}

var _ connectors.OpenSearchConnector = (*openSearchStub)(nil)
