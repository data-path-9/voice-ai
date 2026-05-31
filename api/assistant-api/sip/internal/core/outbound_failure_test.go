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
	"testing"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	"github.com/stretchr/testify/assert"
)

func TestClassifyOutboundFailure(t *testing.T) {
	cases := []struct {
		name           string
		err            error
		expectedClass  OutboundFailureClass
		expectedReason LifecycleReason
		expectedRetry  bool
		expectedStatus int
	}{
		{name: "auth required", err: ErrAuthRequired, expectedClass: OutboundFailureAuthRequired, expectedReason: LifecycleReasonOutboundAuthFailed},
		{name: "forbidden", err: dialogStatusError(403, "Forbidden"), expectedClass: OutboundFailureForbidden, expectedReason: LifecycleReasonOutboundRejected, expectedStatus: 403},
		{name: "not found", err: dialogStatusError(404, "Not Found"), expectedClass: OutboundFailureNotFound, expectedReason: LifecycleReasonOutboundRejected, expectedStatus: 404},
		{name: "request timeout", err: dialogStatusError(408, "Request Timeout"), expectedClass: OutboundFailureNoAnswer, expectedReason: LifecycleReasonOutboundNoAnswer, expectedRetry: true, expectedStatus: 408},
		{name: "temporarily unavailable", err: dialogStatusError(480, "Temporarily Unavailable"), expectedClass: OutboundFailureUnavailable, expectedReason: LifecycleReasonOutboundUnavailable, expectedRetry: true, expectedStatus: 480},
		{name: "busy", err: dialogStatusError(486, "Busy Here"), expectedClass: OutboundFailureBusy, expectedReason: LifecycleReasonOutboundRejected, expectedStatus: 486},
		{name: "global busy", err: dialogStatusError(600, "Busy Everywhere"), expectedClass: OutboundFailureBusy, expectedReason: LifecycleReasonOutboundRejected, expectedStatus: 600},
		{name: "not acceptable", err: dialogStatusError(488, "Not Acceptable Here"), expectedClass: OutboundFailureMedia, expectedReason: LifecycleReasonOutboundMediaRejected, expectedStatus: 488},
		{name: "service unavailable", err: dialogStatusError(503, "Service Unavailable"), expectedClass: OutboundFailureUnavailable, expectedReason: LifecycleReasonOutboundUnavailable, expectedRetry: true, expectedStatus: 503},
		{name: "server failure", err: dialogStatusError(500, "Server Internal Error"), expectedClass: OutboundFailureUpstreamFailure, expectedReason: LifecycleReasonOutboundUpstreamFailure, expectedRetry: true, expectedStatus: 500},
		{name: "bad gateway", err: dialogStatusError(502, "Bad Gateway"), expectedClass: OutboundFailureUpstreamFailure, expectedReason: LifecycleReasonOutboundUpstreamFailure, expectedRetry: true, expectedStatus: 502},
		{name: "gateway timeout", err: dialogStatusError(504, "Gateway Timeout"), expectedClass: OutboundFailureUpstreamFailure, expectedReason: LifecycleReasonOutboundUpstreamFailure, expectedRetry: true, expectedStatus: 504},
		{name: "global decline", err: dialogStatusError(603, "Decline"), expectedClass: OutboundFailureRejected, expectedReason: LifecycleReasonOutboundRejected, expectedStatus: 603},
		{name: "dns", err: &net.DNSError{Err: "no such host", Name: "trunk.example.com"}, expectedClass: OutboundFailureNetwork, expectedReason: LifecycleReasonOutboundNetworkFailure, expectedRetry: true},
		{name: "address", err: &net.AddrError{Err: "missing port", Addr: "trunk.example.com"}, expectedClass: OutboundFailureNetwork, expectedReason: LifecycleReasonOutboundNetworkFailure},
		{name: "transport", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, expectedClass: OutboundFailureNetwork, expectedReason: LifecycleReasonOutboundNetworkFailure, expectedRetry: true},
		{name: "sdp", err: ErrSDPParseFailed, expectedClass: OutboundFailureMedia, expectedReason: LifecycleReasonOutboundAnswerSDPFailed},
		{name: "codec", err: ErrCodecNotSupported, expectedClass: OutboundFailureMedia, expectedReason: LifecycleReasonOutboundAnswerSDPFailed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			failure := classifyOutboundFailure(tc.err, context.Background())

			assert.Equal(t, tc.expectedClass, failure.Class)
			assert.Equal(t, tc.expectedReason, failure.LifecycleReason)
			assert.Equal(t, tc.expectedRetry, failure.Retryable)
			assert.Equal(t, tc.expectedStatus, failure.StatusCode)
		})
	}
}

func dialogStatusError(statusCode int, reason string) error {
	return &sipgo.ErrDialogResponse{Res: sip.NewResponse(statusCode, reason)}
}
