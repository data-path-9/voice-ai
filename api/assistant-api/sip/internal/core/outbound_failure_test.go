// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
)

func TestClassifyOutboundFailure(t *testing.T) {
	deadlineContext, cancelDeadlineContext := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	cancelDeadlineContext()

	cases := []struct {
		name                    string
		err                     error
		answerCtx               context.Context
		expectedClass           OutboundFailureClass
		expectedFailureReason   string
		expectedLifecycleReason LifecycleReason
		expectedRetryable       bool
		expectedStatusCode      int
	}{
		{
			name:                    "nil error",
			expectedClass:           OutboundFailureUnknown,
			expectedFailureReason:   "unknown",
			expectedLifecycleReason: LifecycleReasonOutboundWaitAnswerFailed,
		},
		{
			name:                    "auth required",
			err:                     ErrAuthRequired,
			expectedClass:           OutboundFailureAuthRequired,
			expectedFailureReason:   "auth credentials missing",
			expectedLifecycleReason: LifecycleReasonOutboundAuthFailed,
		},
		{
			name:                    "wrapped auth required",
			err:                     fmt.Errorf("auth challenge failed: %w", ErrAuthRequired),
			expectedClass:           OutboundFailureAuthRequired,
			expectedFailureReason:   "auth credentials missing",
			expectedLifecycleReason: LifecycleReasonOutboundAuthFailed,
		},
		{
			name:                    "context cancelled",
			err:                     context.Canceled,
			expectedClass:           OutboundFailureCancelled,
			expectedFailureReason:   "cancelled",
			expectedLifecycleReason: LifecycleReasonOutboundCancelledBeforeAnswer,
		},
		{
			name:                    "context deadline exceeded",
			err:                     context.DeadlineExceeded,
			expectedClass:           OutboundFailureNoAnswer,
			expectedFailureReason:   "ringing timeout",
			expectedLifecycleReason: LifecycleReasonOutboundNoAnswer,
			expectedRetryable:       true,
		},
		{
			name:                    "answer context deadline exceeded",
			err:                     errors.New("answer wait failed"),
			answerCtx:               deadlineContext,
			expectedClass:           OutboundFailureNoAnswer,
			expectedFailureReason:   "ringing timeout",
			expectedLifecycleReason: LifecycleReasonOutboundNoAnswer,
			expectedRetryable:       true,
		},
		{
			name:                    "401 unauthorized",
			err:                     dialogStatusError(401, "Unauthorized"),
			expectedClass:           OutboundFailureAuthRequired,
			expectedFailureReason:   "Unauthorized",
			expectedLifecycleReason: LifecycleReasonOutboundAuthFailed,
			expectedStatusCode:      401,
		},
		{
			name:                    "407 proxy auth required",
			err:                     dialogStatusError(407, "Proxy Authentication Required"),
			expectedClass:           OutboundFailureAuthRequired,
			expectedFailureReason:   "Proxy Authentication Required",
			expectedLifecycleReason: LifecycleReasonOutboundAuthFailed,
			expectedStatusCode:      407,
		},
		{
			name:                    "403 forbidden",
			err:                     dialogStatusError(403, "Forbidden"),
			expectedClass:           OutboundFailureForbidden,
			expectedFailureReason:   "Forbidden",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      403,
		},
		{
			name:                    "404 not found",
			err:                     dialogStatusError(404, "Not Found"),
			expectedClass:           OutboundFailureNotFound,
			expectedFailureReason:   "Not Found",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      404,
		},
		{
			name:                    "408 request timeout",
			err:                     dialogStatusError(408, "Request Timeout"),
			expectedClass:           OutboundFailureNoAnswer,
			expectedFailureReason:   "Request Timeout",
			expectedLifecycleReason: LifecycleReasonOutboundNoAnswer,
			expectedRetryable:       true,
			expectedStatusCode:      408,
		},
		{
			name:                    "480 temporarily unavailable",
			err:                     dialogStatusError(480, "Temporarily Unavailable"),
			expectedClass:           OutboundFailureUnavailable,
			expectedFailureReason:   "Temporarily Unavailable",
			expectedLifecycleReason: LifecycleReasonOutboundUnavailable,
			expectedRetryable:       true,
			expectedStatusCode:      480,
		},
		{
			name:                    "486 busy",
			err:                     dialogStatusError(486, "Busy Here"),
			expectedClass:           OutboundFailureBusy,
			expectedFailureReason:   "Busy Here",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      486,
		},
		{
			name:                    "487 request terminated",
			err:                     dialogStatusError(487, "Request Terminated"),
			expectedClass:           OutboundFailureRejected,
			expectedFailureReason:   "Request Terminated",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      487,
		},
		{
			name:                    "488 media rejected",
			err:                     dialogStatusError(488, "Not Acceptable Here"),
			expectedClass:           OutboundFailureMedia,
			expectedFailureReason:   "Not Acceptable Here",
			expectedLifecycleReason: LifecycleReasonOutboundMediaRejected,
			expectedStatusCode:      488,
		},
		{
			name:                    "generic 4xx rejected",
			err:                     dialogStatusError(410, "Gone"),
			expectedClass:           OutboundFailureRejected,
			expectedFailureReason:   "Gone",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      410,
		},
		{
			name:                    "500 upstream failure",
			err:                     dialogStatusError(500, "Server Internal Error"),
			expectedClass:           OutboundFailureUpstreamFailure,
			expectedFailureReason:   "Server Internal Error",
			expectedLifecycleReason: LifecycleReasonOutboundUpstreamFailure,
			expectedRetryable:       true,
			expectedStatusCode:      500,
		},
		{
			name:                    "502 bad gateway",
			err:                     dialogStatusError(502, "Bad Gateway"),
			expectedClass:           OutboundFailureUpstreamFailure,
			expectedFailureReason:   "Bad Gateway",
			expectedLifecycleReason: LifecycleReasonOutboundUpstreamFailure,
			expectedRetryable:       true,
			expectedStatusCode:      502,
		},
		{
			name:                    "503 service unavailable",
			err:                     dialogStatusError(503, "Service Unavailable"),
			expectedClass:           OutboundFailureUpstreamFailure,
			expectedFailureReason:   "Service Unavailable",
			expectedLifecycleReason: LifecycleReasonOutboundUpstreamFailure,
			expectedRetryable:       true,
			expectedStatusCode:      503,
		},
		{
			name:                    "504 gateway timeout",
			err:                     dialogStatusError(504, "Gateway Timeout"),
			expectedClass:           OutboundFailureUpstreamFailure,
			expectedFailureReason:   "Gateway Timeout",
			expectedLifecycleReason: LifecycleReasonOutboundUpstreamFailure,
			expectedRetryable:       true,
			expectedStatusCode:      504,
		},
		{
			name:                    "generic 5xx upstream failure",
			err:                     dialogStatusError(501, "Not Implemented"),
			expectedClass:           OutboundFailureUpstreamFailure,
			expectedFailureReason:   "Not Implemented",
			expectedLifecycleReason: LifecycleReasonOutboundUpstreamFailure,
			expectedRetryable:       true,
			expectedStatusCode:      501,
		},
		{
			name:                    "600 global busy",
			err:                     dialogStatusError(600, "Busy Everywhere"),
			expectedClass:           OutboundFailureBusy,
			expectedFailureReason:   "Busy Everywhere",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      600,
		},
		{
			name:                    "603 global decline",
			err:                     dialogStatusError(603, "Decline"),
			expectedClass:           OutboundFailureRejected,
			expectedFailureReason:   "Decline",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      603,
		},
		{
			name:                    "generic 6xx rejected",
			err:                     dialogStatusError(606, "Not Acceptable"),
			expectedClass:           OutboundFailureRejected,
			expectedFailureReason:   "Not Acceptable",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      606,
		},
		{
			name:                    "wrapped SIP status",
			err:                     fmt.Errorf("INVITE failed: %w", dialogStatusError(486, "Busy Here")),
			expectedClass:           OutboundFailureBusy,
			expectedFailureReason:   "Busy Here",
			expectedLifecycleReason: LifecycleReasonOutboundRejected,
			expectedStatusCode:      486,
		},
		{
			name:                    "dns",
			err:                     &net.DNSError{Err: "no such host", Name: "trunk.example.com"},
			expectedClass:           OutboundFailureNetwork,
			expectedFailureReason:   "dns resolution failed",
			expectedLifecycleReason: LifecycleReasonOutboundNetworkFailure,
			expectedRetryable:       true,
		},
		{
			name:                    "address",
			err:                     &net.AddrError{Err: "missing port", Addr: "trunk.example.com"},
			expectedClass:           OutboundFailureNetwork,
			expectedFailureReason:   "invalid SIP address",
			expectedLifecycleReason: LifecycleReasonOutboundNetworkFailure,
		},
		{
			name:                    "transport",
			err:                     &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")},
			expectedClass:           OutboundFailureNetwork,
			expectedFailureReason:   "SIP transport failed",
			expectedLifecycleReason: LifecycleReasonOutboundNetworkFailure,
			expectedRetryable:       true,
		},
		{
			name:                    "sdp parse failed",
			err:                     ErrSDPParseFailed,
			expectedClass:           OutboundFailureMedia,
			expectedFailureReason:   ErrSDPParseFailed.Error(),
			expectedLifecycleReason: LifecycleReasonOutboundAnswerSDPFailed,
		},
		{
			name:                    "codec unsupported",
			err:                     ErrCodecNotSupported,
			expectedClass:           OutboundFailureMedia,
			expectedFailureReason:   ErrCodecNotSupported.Error(),
			expectedLifecycleReason: LifecycleReasonOutboundAnswerSDPFailed,
		},
		{
			name:                    "unknown",
			err:                     errors.New("unexpected invite failure"),
			expectedClass:           OutboundFailureUnknown,
			expectedFailureReason:   "unknown",
			expectedLifecycleReason: LifecycleReasonOutboundWaitAnswerFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answerCtx := tc.answerCtx
			if answerCtx == nil {
				answerCtx = context.Background()
			}
			failure := classifyOutboundFailure(tc.err, answerCtx)

			assert.Equal(t, tc.expectedClass, failure.Class)
			assert.Equal(t, tc.expectedFailureReason, failure.Reason)
			assert.Equal(t, tc.expectedLifecycleReason, failure.LifecycleReason)
			assert.Equal(t, tc.expectedRetryable, failure.Retryable)
			assert.Equal(t, tc.expectedStatusCode, failure.StatusCode)
		})
	}
}

func dialogStatusError(statusCode int, reason string) error {
	return &sipgo.ErrDialogResponse{Res: sip.NewResponse(statusCode, reason)}
}
