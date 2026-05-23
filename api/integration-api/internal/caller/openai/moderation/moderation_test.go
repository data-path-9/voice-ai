// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_moderation

import (
	"context"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func TestNew_ReturnsCaller(t *testing.T) {
	caller := New(newTestLogger(), nil)
	require.NotNil(t, caller)
}

func TestGetModeration_ReturnsTextContentAndMetrics(t *testing.T) {
	caller := New(newTestLogger(), nil)
	content, metrics, err := caller.GetModeration(context.Background(), &types.Content{}, &internal_callers.ModerationOptions{})
	require.NoError(t, err)
	require.NotNil(t, content)
	require.NotEmpty(t, metrics)
	assert.Equal(t, "text", content.ContentType)
	assert.Equal(t, "raw", content.ContentFormat)
}
