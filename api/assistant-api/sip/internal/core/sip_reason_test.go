// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSIPReasonHeader_Q850NormalClearing(t *testing.T) {
	metadata := parseSIPReasonHeader(`Q.850;cause=16;text="Normal call clearing"`)

	assert.Equal(t, DisconnectReasonNormalClearing, metadata.Reason)
	assert.Equal(t, 16, metadata.ProviderStatusCode)
	assert.Equal(t, "Normal call clearing", metadata.Text)
}

func TestParseSIPReasonHeader_SIPBusy(t *testing.T) {
	metadata := parseSIPReasonHeader(`SIP;cause=486;text="Busy Here"`)

	assert.Equal(t, DisconnectReasonBusy, metadata.Reason)
	assert.Equal(t, 486, metadata.ProviderStatusCode)
	assert.Equal(t, "Busy Here", metadata.Text)
}

func TestParseSIPReasonHeader_QuotedTextWithSemicolon(t *testing.T) {
	metadata := parseSIPReasonHeader(`Q.850;cause=41;text="Temporary failure; upstream"`)

	assert.Equal(t, DisconnectReasonNetworkFailure, metadata.Reason)
	assert.Equal(t, "Temporary failure; upstream", metadata.Text)
}

func TestParseSIPDisconnectMetadata_DefaultsToRemoteHangup(t *testing.T) {
	metadata := parseSIPDisconnectMetadata(nil)

	assert.Equal(t, DisconnectReasonRemoteHangup, metadata.Reason)
	assert.Zero(t, metadata.ProviderStatusCode)
}
