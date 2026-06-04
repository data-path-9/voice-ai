// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"context"
	"errors"
	"testing"
)

func TestCollectors_CollectFansOutAndJoinsErrors(t *testing.T) {
	collectorErr := errors.New("collector failed")
	first := &recordingCollector{}
	second := &recordingCollector{err: collectorErr}
	fanout := NewCollectors(first, nil, second)

	err := fanout.Collect(context.Background(), Envelope{ID: "evt-1", Name: CallRinging})
	if !errors.Is(err, collectorErr) {
		t.Fatalf("expected collector error, got %v", err)
	}
	if len(first.envelopes) != 1 || len(second.envelopes) != 1 {
		t.Fatalf("expected both collectors to receive envelope, got first=%d second=%d", len(first.envelopes), len(second.envelopes))
	}
}

func TestCollectors_ShutdownFansOut(t *testing.T) {
	first := &recordingCollector{}
	second := &recordingCollector{}
	fanout := NewCollectors(first, second)

	if err := fanout.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	if !first.closed || !second.closed {
		t.Fatalf("expected both collectors to close, got first=%t second=%t", first.closed, second.closed)
	}
}

func TestNoopCollector(t *testing.T) {
	collector := NewCollectors()
	if err := collector.Collect(context.Background(), Envelope{}); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if err := collector.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestCollectorFunc(t *testing.T) {
	var got Envelope
	collector := CollectorFunc(func(_ context.Context, envelope Envelope) error {
		got = envelope
		return nil
	})

	if err := collector.Collect(context.Background(), Envelope{ID: "evt-1"}); err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if got.ID != "evt-1" {
		t.Fatalf("expected function collector to receive envelope, got %+v", got)
	}
	if err := collector.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}
