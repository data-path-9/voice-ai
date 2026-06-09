// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vonage

import (
	"strings"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
)

type StatusCallback struct {
	Status         string
	ChannelUUID    string
	Duration       *time.Duration
	Price          string
	Detail         string
	SIPCode        string
	Reason         string
	DisconnectedBy string
	RawPayload     string
	Payload        map[string]interface{}
}

func NewStatusCallback(payload map[string]interface{}, rawCallbackPayload string) (*StatusCallback, error) {
	options := utils.Option(payload)
	status, _ := options.GetString("status")
	if !validator.NotBlank(status) {
		return nil, ErrStatusCallbackStatusMissing
	}

	channelUUID, _ := options.GetString("uuid")
	duration, err := options.GetDuration("duration")
	var durationPtr *time.Duration
	if err == nil {
		durationPtr = utils.Ptr(duration)
	}
	price, _ := options.GetString("price")
	detail, _ := options.GetString("detail")
	sipCode, _ := options.GetString("sip_code")
	reason, _ := options.GetString("reason")
	disconnectedBy, _ := options.GetString("disconnected_by")

	return &StatusCallback{
		Status:         status,
		ChannelUUID:    channelUUID,
		Duration:       durationPtr,
		Price:          price,
		Detail:         detail,
		SIPCode:        sipCode,
		Reason:         reason,
		DisconnectedBy: disconnectedBy,
		RawPayload:     rawCallbackPayload,
		Payload:        payload,
	}, nil
}

func (s *StatusCallback) StatusInfo() *internal_type.StatusInfo {
	callbackFailed := s.Failed()
	statusInfo := &internal_type.StatusInfo{
		Event:       s.Status,
		ChannelUUID: s.ChannelUUID,
		Completed:   strings.EqualFold(s.Status, "completed") && !callbackFailed,
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
	statusLower := strings.ToLower(s.Status)
	return statusLower == "failed" ||
		statusLower == "busy" ||
		statusLower == "timeout" ||
		statusLower == "unanswered" ||
		statusLower == "rejected" ||
		statusLower == "cancelled" ||
		statusLower == "canceled" ||
		(statusLower == "completed" && validator.NotBlank(s.Detail) && s.Duration != nil && *s.Duration == 0)
}

func (s *StatusCallback) FailureReason() string {
	if validator.NotBlank(s.Detail) {
		return s.Detail
	}
	if validator.NotBlank(s.Reason) {
		return s.Reason
	}
	if validator.NotBlank(s.DisconnectedBy) {
		return s.DisconnectedBy
	}
	if validator.NotBlank(s.SIPCode) {
		return s.SIPCode
	}
	return s.Status
}
