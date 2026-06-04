// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_webhook

// Event is the normalized assistant webhook event name.
type Event string

const (
	// CallStatus is the catch-all event for every provider call status callback.
	CallStatus Event = "call.status"

	CallReceived     Event = "call.received"
	CallInitiated    Event = "call.initiated"
	CallQueued       Event = "call.queued"
	CallRinging      Event = "call.ringing"
	CallAnswered     Event = "call.answered"
	CallStarted      Event = "call.started"
	CallInProgress   Event = "call.in_progress"
	CallMediaStarted Event = "call.media_started"
	CallHangup       Event = "call.hangup"
	CallCompleted    Event = "call.completed"
	CallFailed       Event = "call.failed"
	CallBusy         Event = "call.busy"
	CallNoAnswer     Event = "call.no_answer"
	CallRejected     Event = "call.rejected"
	CallCancelled    Event = "call.cancelled"
)

const (
	ConversationBegin     Event = "conversation.begin"
	ConversationResume    Event = "conversation.resume"
	ConversationStarted   Event = "conversation.started"
	ConversationCompleted Event = "conversation.completed"
	ConversationFinalized Event = "conversation.finalized"
	ConversationFailed    Event = "conversation.failed"
	ConversationError     Event = "conversation.error"
)

var callEvents = []Event{
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
}

var conversationEvents = []Event{
	ConversationBegin,
	ConversationResume,
	ConversationStarted,
	ConversationCompleted,
	ConversationFinalized,
	ConversationFailed,
	ConversationError,
}

func (e Event) String() string {
	return string(e)
}

func (e Event) Get() string {
	return string(e)
}

func CallEvents() []Event {
	return append([]Event(nil), callEvents...)
}

func ConversationEvents() []Event {
	return append([]Event(nil), conversationEvents...)
}

func AllEvents() []Event {
	events := make([]Event, 0, len(callEvents)+len(conversationEvents))
	events = append(events, callEvents...)
	events = append(events, conversationEvents...)
	return events
}
