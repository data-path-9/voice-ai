// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/rapidaai/api/assistant-api/internal/observability"
)

type notifierStub struct {
	notifications []Notification
	err           error
}

func (n *notifierStub) Notify(_ context.Context, notification Notification) error {
	n.notifications = append(n.notifications, notification)
	return n.err
}

func TestCollector_NotifiesFailures(t *testing.T) {
	notifier := &notifierStub{}
	collector := New(Config{Notifier: notifier})

	err := collector.Collect(context.Background(), observability.Envelope{
		ID:       "evt-1",
		Kind:     observability.RecordKindEvent,
		Name:     observability.CallFailed,
		Category: observability.CategoryCall,
		Outcome:  observability.OutcomeFailure,
		Title:    "Call failed",
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(notifier.notifications) != 1 {
		t.Fatalf("expected one notification, got %d", len(notifier.notifications))
	}
	got := notifier.notifications[0]
	if got.ID != "evt-1" || got.Event != observability.CallFailed || got.Title != "Call failed" {
		t.Fatalf("unexpected notification: %+v", got)
	}
}

func TestCollector_DefaultSelectorSkipsSuccessfulEvents(t *testing.T) {
	notifier := &notifierStub{}
	collector := New(Config{Notifier: notifier})

	err := collector.Collect(context.Background(), observability.Envelope{
		Kind:     observability.RecordKindEvent,
		Name:     observability.CallRinging,
		Category: observability.CategoryCall,
		Outcome:  observability.OutcomeSuccess,
	})
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(notifier.notifications) != 0 {
		t.Fatalf("expected no notifications, got %+v", notifier.notifications)
	}
}

func TestCollector_ReturnsNotifierError(t *testing.T) {
	notifyErr := errors.New("notify failed")
	collector := New(Config{Notifier: &notifierStub{err: notifyErr}})

	err := collector.Collect(context.Background(), observability.Envelope{
		Kind:    observability.RecordKindEvent,
		Name:    observability.ErrorRaised,
		Level:   observability.LevelError,
		Outcome: observability.OutcomeFailure,
	})
	if !errors.Is(err, notifyErr) {
		t.Fatalf("expected notifier error, got %v", err)
	}
}
