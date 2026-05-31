// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"errors"
	"net"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

type OutboundFailureClass string

const (
	OutboundFailureAuthRequired    OutboundFailureClass = "auth_required"
	OutboundFailureForbidden       OutboundFailureClass = "forbidden"
	OutboundFailureNotFound        OutboundFailureClass = "not_found"
	OutboundFailureNoAnswer        OutboundFailureClass = "no_answer"
	OutboundFailureUnavailable     OutboundFailureClass = "unavailable"
	OutboundFailureBusy            OutboundFailureClass = "busy"
	OutboundFailureRejected        OutboundFailureClass = "rejected"
	OutboundFailureUpstreamFailure OutboundFailureClass = "upstream_failure"
	OutboundFailureNetwork         OutboundFailureClass = "network"
	OutboundFailureCancelled       OutboundFailureClass = "cancelled"
	OutboundFailureMedia           OutboundFailureClass = "media"
	OutboundFailureUnknown         OutboundFailureClass = "unknown"
)

type OutboundFailure struct {
	Class           OutboundFailureClass
	StatusCode      int
	Reason          string
	Retryable       bool
	LifecycleReason LifecycleReason
}

func classifyOutboundFailure(err error, answerCtx context.Context) OutboundFailure {
	failure := OutboundFailure{
		Class:           OutboundFailureUnknown,
		Reason:          "unknown",
		LifecycleReason: LifecycleReasonOutboundWaitAnswerFailed,
	}

	if err == nil {
		return failure
	}
	if errors.Is(err, ErrAuthRequired) {
		return OutboundFailure{
			Class:           OutboundFailureAuthRequired,
			Reason:          "auth credentials missing",
			LifecycleReason: LifecycleReasonOutboundAuthFailed,
		}
	}
	if errors.Is(err, ErrSDPParseFailed) || errors.Is(err, ErrCodecNotSupported) {
		return OutboundFailure{
			Class:           OutboundFailureMedia,
			Reason:          err.Error(),
			LifecycleReason: LifecycleReasonOutboundAnswerSDPFailed,
		}
	}
	if errors.Is(err, context.Canceled) {
		return OutboundFailure{
			Class:           OutboundFailureCancelled,
			Reason:          "cancelled",
			LifecycleReason: LifecycleReasonOutboundCancelledBeforeAnswer,
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || (answerCtx != nil && errors.Is(answerCtx.Err(), context.DeadlineExceeded)) {
		return OutboundFailure{
			Class:           OutboundFailureNoAnswer,
			Reason:          "ringing timeout",
			Retryable:       true,
			LifecycleReason: LifecycleReasonOutboundNoAnswer,
		}
	}

	var dialogErr *sipgo.ErrDialogResponse
	if errors.As(err, &dialogErr) && dialogErr.Res != nil {
		return classifyOutboundSIPStatus(dialogErr.Res.StatusCode, dialogErr.Res.Reason)
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return OutboundFailure{
			Class:           OutboundFailureNetwork,
			Reason:          "dns resolution failed",
			Retryable:       true,
			LifecycleReason: LifecycleReasonOutboundNetworkFailure,
		}
	}
	var addrErr *net.AddrError
	if errors.As(err, &addrErr) {
		return OutboundFailure{
			Class:           OutboundFailureNetwork,
			Reason:          "invalid SIP address",
			LifecycleReason: LifecycleReasonOutboundNetworkFailure,
		}
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return OutboundFailure{
			Class:           OutboundFailureNetwork,
			Reason:          "SIP transport failed",
			Retryable:       true,
			LifecycleReason: LifecycleReasonOutboundNetworkFailure,
		}
	}
	return failure
}

func classifyOutboundSIPStatus(statusCode int, reason string) OutboundFailure {
	failure := OutboundFailure{
		StatusCode:      statusCode,
		Reason:          reason,
		LifecycleReason: LifecycleReasonOutboundRejected,
	}
	switch statusCode {
	case sip.StatusUnauthorized, sip.StatusProxyAuthRequired:
		failure.Class = OutboundFailureAuthRequired
		failure.LifecycleReason = LifecycleReasonOutboundAuthFailed
	case sip.StatusForbidden:
		failure.Class = OutboundFailureForbidden
	case sip.StatusNotFound:
		failure.Class = OutboundFailureNotFound
	case sip.StatusRequestTimeout:
		failure.Class = OutboundFailureNoAnswer
		failure.Retryable = true
		failure.LifecycleReason = LifecycleReasonOutboundNoAnswer
	case sip.StatusTemporarilyUnavailable, sip.StatusServiceUnavailable:
		failure.Class = OutboundFailureUnavailable
		failure.Retryable = true
		failure.LifecycleReason = LifecycleReasonOutboundUnavailable
	case sip.StatusBusyHere, sip.StatusGlobalBusyEverywhere:
		failure.Class = OutboundFailureBusy
	case sip.StatusNotAcceptableHere:
		failure.Class = OutboundFailureMedia
		failure.LifecycleReason = LifecycleReasonOutboundMediaRejected
	case sip.StatusGlobalDecline:
		failure.Class = OutboundFailureRejected
	case sip.StatusInternalServerError, sip.StatusBadGateway, sip.StatusGatewayTimeout:
		failure.Class = OutboundFailureUpstreamFailure
		failure.Retryable = true
		failure.LifecycleReason = LifecycleReasonOutboundUpstreamFailure
	default:
		switch {
		case statusCode >= 400 && statusCode < 500:
			failure.Class = OutboundFailureRejected
		case statusCode >= 500 && statusCode < 600:
			failure.Class = OutboundFailureUpstreamFailure
			failure.Retryable = true
			failure.LifecycleReason = LifecycleReasonOutboundUpstreamFailure
		case statusCode >= 600 && statusCode < 700:
			failure.Class = OutboundFailureRejected
		default:
			failure.Class = OutboundFailureUnknown
			failure.LifecycleReason = LifecycleReasonOutboundWaitAnswerFailed
		}
	}
	return failure
}
