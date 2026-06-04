// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
)

type publisherStub struct {
	usages []Usage
	err    error
}

func (p *publisherStub) PublishUsage(_ context.Context, usage Usage) error {
	p.usages = append(p.usages, usage)
	return p.err
}

func TestCollector_PublishesUsageRecords(t *testing.T) {
	publisher := &publisherStub{}
	collector := New(publisher)
	duration := 2 * time.Second

	err := collector.Collect(context.Background(), observability.Envelope{
		ID:         "evt-1",
		Name:       observability.UsageRecorded,
		Category:   observability.CategoryUsage,
		Attributes: observability.Attributes{"k": "v"},
		Scope:      observability.Scope{AssistantID: 10, ConversationID: 20},
		Record: observability.UsageEvent{
			BaseRecord:    observability.BaseRecord{RecordName: observability.UsageRecorded},
			Component:     "stt",
			Provider:      "deepgram",
			UsageCategory: "audio",
			Duration:      duration,
		},
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(publisher.usages) != 1 {
		t.Fatalf("expected one usage, got %d", len(publisher.usages))
	}
	usage := publisher.usages[0]
	if usage.ID != "evt-1" || usage.Component != "stt" || usage.Provider != "deepgram" || usage.Duration != duration {
		t.Fatalf("unexpected usage: %+v", usage)
	}
	if usage.Scope.AssistantID != 10 || usage.Attributes["k"] != "v" {
		t.Fatalf("unexpected usage context: %+v", usage)
	}
}

func TestCollector_IgnoresNonUsageRecords(t *testing.T) {
	publisher := &publisherStub{}
	collector := New(publisher)

	err := collector.Collect(context.Background(), observability.Envelope{
		Name: observability.CallRinging,
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(publisher.usages) != 0 {
		t.Fatalf("expected no usage to be published, got %+v", publisher.usages)
	}
}

func TestCollector_ReturnsPublisherError(t *testing.T) {
	publisherErr := errors.New("publish failed")
	collector := New(&publisherStub{err: publisherErr})

	err := collector.Collect(context.Background(), observability.Envelope{
		Name: observability.UsageRecorded,
		Record: observability.UsageEvent{
			BaseRecord: observability.BaseRecord{RecordName: observability.UsageRecorded},
			Component:  "sip",
			Duration:   time.Second,
		},
	})
	if !errors.Is(err, publisherErr) {
		t.Fatalf("expected publisher error, got %v", err)
	}
}
