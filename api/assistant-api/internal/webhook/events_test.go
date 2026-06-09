// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_webhook

import "testing"

func TestEventValues(t *testing.T) {
	tests := []struct {
		event Event
		want  string
	}{
		{CallStatus, "call.status"},
		{CallReceived, "call.received"},
		{CallInitiated, "call.initiated"},
		{CallQueued, "call.queued"},
		{CallRinging, "call.ringing"},
		{CallAnswered, "call.answered"},
		{CallStarted, "call.started"},
		{CallInProgress, "call.in_progress"},
		{CallMediaStarted, "call.media_started"},
		{CallHangup, "call.hangup"},
		{CallCompleted, "call.completed"},
		{CallFailed, "call.failed"},
		{CallBusy, "call.busy"},
		{CallNoAnswer, "call.no_answer"},
		{CallRejected, "call.rejected"},
		{CallCancelled, "call.cancelled"},
		{ConversationBegin, "conversation.begin"},
		{ConversationResume, "conversation.resume"},
		{ConversationStarted, "conversation.started"},
		{ConversationCompleted, "conversation.completed"},
		{ConversationFinalized, "conversation.finalized"},
		{ConversationFailed, "conversation.failed"},
		{ConversationError, "conversation.error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.event.Get(); got != tt.want {
				t.Fatalf("Get() = %q, want %q", got, tt.want)
			}
			if got := tt.event.String(); got != tt.want {
				t.Fatalf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventGroups(t *testing.T) {
	assertEvents(t, CallEvents(), []Event{
		CallStatus,
		CallReceived,
		CallInitiated,
		CallQueued,
		CallRinging,
		CallAnswered,
		CallStarted,
		CallInProgress,
		CallMediaStarted,
		CallHangup,
		CallCompleted,
		CallFailed,
		CallBusy,
		CallNoAnswer,
		CallRejected,
		CallCancelled,
	})

	assertEvents(t, ConversationEvents(), []Event{
		ConversationBegin,
		ConversationResume,
		ConversationStarted,
		ConversationCompleted,
		ConversationFinalized,
		ConversationFailed,
		ConversationError,
	})

	all := AllEvents()
	if len(all) != len(CallEvents())+len(ConversationEvents()) {
		t.Fatalf("AllEvents length = %d, want %d", len(all), len(CallEvents())+len(ConversationEvents()))
	}
	assertNoDuplicateEvents(t, all)
}

func assertEvents(t *testing.T, got, want []Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func assertNoDuplicateEvents(t *testing.T, events []Event) {
	t.Helper()
	seen := map[Event]bool{}
	for _, event := range events {
		if seen[event] {
			t.Fatalf("duplicate event %q", event)
		}
		seen[event] = true
	}
}
