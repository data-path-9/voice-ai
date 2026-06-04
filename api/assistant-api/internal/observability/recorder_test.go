// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type recordingCollector struct {
	envelopes []Envelope
	err       error
	closed    bool
}

func (c *recordingCollector) Collect(_ context.Context, envelope Envelope) error {
	c.envelopes = append(c.envelopes, envelope)
	return c.err
}

func (c *recordingCollector) Shutdown(context.Context) error {
	c.closed = true
	return c.err
}

type blockingCollector struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	close   sync.Once
}

func (c *blockingCollector) Collect(context.Context, Envelope) error {
	c.once.Do(func() { close(c.started) })
	<-c.release
	return nil
}

func (c *blockingCollector) Shutdown(context.Context) error {
	c.close.Do(func() { close(c.release) })
	return nil
}

func TestRecorderRecord_EnrichesAndFansOutRecord(t *testing.T) {
	now := time.Date(2026, 6, 4, 10, 30, 0, 0, time.UTC)
	first := &recordingCollector{}
	second := &recordingCollector{}
	recorder := New(Config{
		Scope: Scope{
			AssistantID:    10,
			ConversationID: 20,
			ProjectID:      30,
			OrganizationID: 40,
			ContextID:      "ctx-default",
		},
		Clock: func() time.Time { return now },
		NewID: func() string { return "evt-1" },
	}, first, second)

	err := recorder.Record(context.Background(), CallEvent{
		BaseRecord: BaseRecord{
			RecordName: CallRinging,
			RecordScope: Scope{
				OrganizationID: 999,
				ProjectID:      998,
				AssistantID:    997,
				ConversationID: 996,
				ContextID:      "ctx-call",
			},
			RecordData:    Data{"raw_status": "ringing"},
			RecordOutcome: OutcomeSuccess,
			RecordTitle:   "Call is ringing",
			Elapsed:       2 * time.Second,
		},
		Provider:  "sip",
		Direction: "outbound",
		Status:    "ringing",
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if err := recorder.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}

	if len(first.envelopes) != 1 || len(second.envelopes) != 1 {
		t.Fatalf("expected both collectors to receive one envelope, got first=%d second=%d", len(first.envelopes), len(second.envelopes))
	}
	envelope := first.envelopes[0]
	if envelope.ID != "evt-1" {
		t.Fatalf("unexpected envelope ID: %s", envelope.ID)
	}
	if envelope.Kind != RecordKindEvent || envelope.Name != CallRinging || envelope.Category != CategoryCall {
		t.Fatalf("unexpected envelope kind/name/category: %s %s %s", envelope.Kind, envelope.Name, envelope.Category)
	}
	if envelope.Level != LevelInfo || envelope.Outcome != OutcomeSuccess || envelope.Title != "Call is ringing" {
		t.Fatalf("unexpected display fields: level=%s outcome=%s title=%s", envelope.Level, envelope.Outcome, envelope.Title)
	}
	if envelope.Scope.OrganizationID != 40 || envelope.Scope.ProjectID != 30 ||
		envelope.Scope.AssistantID != 10 || envelope.Scope.ConversationID != 20 ||
		envelope.Scope.ContextID != "ctx-call" {
		t.Fatalf("unexpected scope: %+v", envelope.Scope)
	}
	if !envelope.OccurredAt.Equal(now) || !envelope.ReceivedAt.Equal(now) {
		t.Fatalf("unexpected times: occurred=%s received=%s", envelope.OccurredAt, envelope.ReceivedAt)
	}
	if envelope.Duration != 2*time.Second {
		t.Fatalf("unexpected duration: %s", envelope.Duration)
	}
	if envelope.Attributes[string(AttrProvider)] != "sip" || envelope.Attributes[string(AttrStatus)] != "ringing" {
		t.Fatalf("unexpected attributes: %+v", envelope.Attributes)
	}
	if envelope.Data["raw_status"] != "ringing" {
		t.Fatalf("unexpected data: %+v", envelope.Data)
	}
}

func TestRecorderRecord_ValidationErrorSkipsCollectors(t *testing.T) {
	collector := &recordingCollector{}
	recorder := New(Config{}, collector)

	err := recorder.Record(context.Background(), CallEvent{
		BaseRecord: BaseRecord{RecordName: ConversationBegin},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if len(collector.envelopes) != 0 {
		t.Fatalf("collector should not receive invalid record, got %d", len(collector.envelopes))
	}
	if err := recorder.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}

func TestRecorderShutdown_ReturnsCollectorErrorsAfterFanout(t *testing.T) {
	collectorErr := errors.New("collector failed")
	first := &recordingCollector{err: collectorErr}
	second := &recordingCollector{}
	recorder := New(Config{}, first, second)

	err := recorder.Record(context.Background(), ConversationEvent{
		BaseRecord: BaseRecord{RecordName: ConversationCompleted},
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	err = recorder.Shutdown(context.Background())
	if !errors.Is(err, collectorErr) {
		t.Fatalf("expected collector error on shutdown, got %v", err)
	}
	if len(first.envelopes) != 1 || len(second.envelopes) != 1 {
		t.Fatalf("expected fanout to continue after collector error, got first=%d second=%d", len(first.envelopes), len(second.envelopes))
	}
}

func TestRecordAttributes_ReturnsCopy(t *testing.T) {
	extra := Attributes{"custom": "value"}
	event := CallEvent{
		BaseRecord: BaseRecord{
			RecordName:       CallAnswered,
			RecordAttributes: extra,
		},
		Provider: "vonage",
	}

	attrs := event.Attributes()
	attrs["custom"] = "changed"

	if extra["custom"] != "value" {
		t.Fatalf("extra map was mutated: %+v", extra)
	}
}

func TestBaseRecordData_ReturnsCopy(t *testing.T) {
	data := Data{"payload": "value"}
	event := EventRecord{
		BaseRecord: BaseRecord{
			RecordName: CallStatus,
			RecordData: data,
		},
	}

	copied := event.Data()
	copied["payload"] = "changed"

	if data["payload"] != "value" {
		t.Fatalf("data map was mutated: %+v", data)
	}
}

func TestErrorEvent_DefaultsToErrorFailure(t *testing.T) {
	event := ErrorEvent{BaseRecord: BaseRecord{RecordName: ErrorRaised}}

	if event.Level() != LevelError {
		t.Fatalf("expected error level, got %s", event.Level())
	}
	if event.Outcome() != OutcomeFailure {
		t.Fatalf("expected failure outcome, got %s", event.Outcome())
	}
}

func TestUsageEvent_RecordsComponentDuration(t *testing.T) {
	event := UsageEvent{
		BaseRecord: BaseRecord{RecordName: UsageRecorded},
		Component:  "stt",
		Provider:   "deepgram",
		Duration:   1200 * time.Millisecond,
	}

	attrs := event.Attributes()
	if attrs[string(AttrComponent)] != "stt" {
		t.Fatalf("expected component attribute, got %+v", attrs)
	}
	if attrs[string(AttrProvider)] != "deepgram" {
		t.Fatalf("expected provider attribute, got %+v", attrs)
	}
	if attrs[string(AttrDuration)] != "1200" {
		t.Fatalf("expected duration attribute, got %+v", attrs)
	}
}

func TestUsageEvent_RequiresComponentAndDuration(t *testing.T) {
	recorder := New(Config{}, &recordingCollector{})
	defer recorder.Shutdown(context.Background())

	err := recorder.Record(context.Background(), UsageEvent{
		BaseRecord: BaseRecord{RecordName: UsageRecorded},
		Duration:   time.Second,
	})
	if err == nil {
		t.Fatal("expected missing component to fail validation")
	}

	err = recorder.Record(context.Background(), UsageEvent{
		BaseRecord: BaseRecord{RecordName: UsageRecorded},
		Component:  "sip",
	})
	if err == nil {
		t.Fatal("expected missing duration to fail validation")
	}
}

func TestRecorderRecord_ReturnsBufferFull(t *testing.T) {
	collector := &blockingCollector{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	recorder := New(Config{Buffer: 1}, collector)
	err := recorder.Record(context.Background(), CallEvent{BaseRecord: BaseRecord{RecordName: CallRinging}})
	if err != nil {
		t.Fatalf("first Record returned error: %v", err)
	}
	<-collector.started

	err = recorder.Record(context.Background(), CallEvent{BaseRecord: BaseRecord{RecordName: CallRinging}})
	if err != nil {
		t.Fatalf("second Record returned error: %v", err)
	}

	err = recorder.Record(context.Background(), CallEvent{BaseRecord: BaseRecord{RecordName: CallRinging}})
	if !errors.Is(err, ErrBufferFull) {
		t.Fatalf("expected buffer full error, got %v", err)
	}

	collector.close.Do(func() { close(collector.release) })
	if err := recorder.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
}
