// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openrouter_callers

import (
	"testing"

	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOpenRouterTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestNewChat_RejectsMissingCredential(t *testing.T) {
	chat, err := NewChat(newOpenRouterTestLogger(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, chat)
}

func TestNewChatStream_RejectsMissingCredential(t *testing.T) {
	stream, err := NewChatStream(newOpenRouterTestLogger(), nil, nil)
	require.Error(t, err)
	assert.Nil(t, stream)
}
