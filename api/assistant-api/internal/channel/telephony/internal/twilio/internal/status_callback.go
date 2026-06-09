// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_twilio

import (
	"strings"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

type StatusCallback struct {
	Event        string
	ChannelUUID  string
	Duration     *time.Duration
	Price        string
	ErrorCode    string
	ErrorMessage string
	StreamError  string
	RawPayload   string
	Payload      utils.Option
}

func NewStatusCallback(eventDetails utils.Option, rawCallbackPayload string) (*StatusCallback, error) {
	event, _ := eventDetails.GetString("CallStatus")
	streamEvent, _ := eventDetails.GetString("StreamEvent")
	if validator.NotBlank(streamEvent) {
		event = streamEvent
	}
	if !validator.NotBlank(event) {
		return nil, ErrStatusCallbackStatusMissing
	}

	channelUUID, _ := eventDetails.GetString("CallSid")
	duration, err := eventDetails.GetDuration("CallDuration")
	if err != nil {
		duration, err = eventDetails.GetDuration("Duration")
	}
	var durationPtr *time.Duration
	if err == nil {
		durationPtr = utils.Ptr(duration)
	}

	price, _ := eventDetails.GetString("Price")
	errorCode, _ := eventDetails.GetString("ErrorCode")
	errorMessage, _ := eventDetails.GetString("ErrorMessage")
	streamError, _ := eventDetails.GetString("StreamError")

	return &StatusCallback{
		Event:        event,
		ChannelUUID:  channelUUID,
		Duration:     durationPtr,
		Price:        price,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		StreamError:  streamError,
		RawPayload:   rawCallbackPayload,
		Payload:      eventDetails,
	}, nil
}

func (s *StatusCallback) StatusInfo() *internal_type.StatusInfo {
	callbackFailed := s.Failed()
	statusInfo := &internal_type.StatusInfo{
		Event:       s.Event,
		ChannelUUID: s.ChannelUUID,
		Completed:   strings.EqualFold(s.Event, "completed") && !callbackFailed,
		Duration:    s.Duration,
		Price:       s.Price,
		RawPayload:  s.RawPayload,
		Payload:     s.Payload,
	}
	if callbackFailed {
		statusInfo.Error = &internal_type.StatusError{Error: "failed", Reason: s.FailureReason()}
	}
	return statusInfo
}

func (s *StatusCallback) Failed() bool {
	eventLower := strings.ToLower(s.Event)
	return eventLower == "failed" ||
		eventLower == "busy" ||
		eventLower == "no-answer" ||
		eventLower == "canceled" ||
		eventLower == "cancelled" ||
		validator.NotBlank(s.ErrorCode) ||
		validator.NotBlank(s.ErrorMessage) ||
		validator.NotBlank(s.StreamError)
}

func (s *StatusCallback) FailureReason() string {
	if validator.NotBlank(s.ErrorMessage) {
		return s.ErrorMessage
	}
	if validator.NotBlank(s.StreamError) {
		return s.StreamError
	}
	if validator.NotBlank(s.ErrorCode) {
		return s.ErrorCode
	}
	return s.Event
}
