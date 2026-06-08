// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telephony_base

import (
	"errors"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/stretchr/testify/require"
)

func TestReportOutboundInitiated(t *testing.T) {
	var got internal_type.ProviderCallStatusUpdate

	ReportOutboundInitiated(func(update internal_type.ProviderCallStatusUpdate) {
		got = update
	}, "provider-call-id")

	require.Equal(t, "provider-call-id", got.ChannelUUID)
	require.Equal(t, OutboundCallStatusInitiated, got.CallStatus)
}

func TestReportOutboundFailure(t *testing.T) {
	var got internal_type.ProviderCallStatusUpdate

	ReportOutboundFailure(
		func(update internal_type.ProviderCallStatusUpdate) {
			got = update
		},
		OutboundFailureClassProviderAPI,
		"provider rejected request",
		OutboundDisconnectReasonSetupFailed,
		errors.New("provider rejected request"),
		503,
	)

	require.Equal(t, OutboundCallStatusFailed, got.CallStatus)
	require.Equal(t, OutboundFailureClassProviderAPI, got.FailureClass)
	require.Equal(t, "provider rejected request", got.FailureReason)
	require.Equal(t, OutboundDisconnectReasonSetupFailed, got.DisconnectReason)
	require.Equal(t, "provider rejected request", got.ErrorMessage)
	require.Equal(t, 503, got.ProviderStatusCode)
}
