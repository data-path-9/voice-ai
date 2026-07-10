// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vobiz

import (
	"strings"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

// StatusCallback parses a vobiz call/stream webhook (ring_url/hangup_url and
// Stream statusCallbackUrl events: Ring, StartApp, Hangup, StartStream,
// StopStream).
type StatusCallback struct {
	Event       string
	CallStatus  string
	ChannelUUID string
	Duration    *time.Duration
	RawPayload  string
	Payload     utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	event, _ := eventDetails.GetString("Event")
	callStatus, _ := eventDetails.GetString("CallStatus")
	channelUUID, _ := eventDetails.GetString("CallUUID")
	if !validator.NotBlank(channelUUID) {
		channelUUID, _ = eventDetails.GetString("RequestUUID")
	}
	var durationPtr *time.Duration
	if d, err := eventDetails.GetDuration("Duration"); err == nil {
		durationPtr = utils.Ptr(d)
	}
	return &StatusCallback{
		Event:       event,
		CallStatus:  callStatus,
		ChannelUUID: channelUUID,
		Duration:    durationPtr,
		RawPayload:  rawCallbackPayload,
		Payload:     eventDetails,
	}, nil
}

func (s *StatusCallback) StatusInfo() *internal_type.StatusInfo {
	failed := s.Failed()
	completed := strings.EqualFold(s.CallStatus, "completed") || strings.EqualFold(s.Event, "Hangup")
	statusInfo := &internal_type.StatusInfo{
		Event:       s.eventName(),
		ChannelUUID: s.ChannelUUID,
		Completed:   completed && !failed,
		Duration:    s.Duration,
		RawPayload:  s.RawPayload,
		Payload:     s.Payload,
	}
	if failed {
		statusInfo.Error = &internal_type.StatusError{Error: "failed", Reason: s.FailureReason()}
	}
	return statusInfo
}

func (s *StatusCallback) eventName() string {
	if validator.NotBlank(s.Event) {
		return s.Event
	}
	if validator.NotBlank(s.CallStatus) {
		return s.CallStatus
	}
	return "webhook"
}

func (s *StatusCallback) Failed() bool {
	st := strings.ToLower(s.CallStatus)
	return st == "failed" || st == "busy" || st == "no-answer" || st == "no_answer" || st == "canceled" || st == "cancelled"
}

func (s *StatusCallback) FailureReason() string {
	if validator.NotBlank(s.CallStatus) {
		return s.CallStatus
	}
	return s.Event
}
