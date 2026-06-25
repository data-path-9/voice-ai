// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_end_of_speech

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

// MockEndOfSpeechCallback is a simple callback function for testing
var mockCallback = func(ctx context.Context, result ...internal_type.Packet) error {
	return nil
}

func pipecatOptions() utils.Option {
	return utils.Option{EndOfSpeechOptionsKeyProvider: PipecatSmartTurnEndOfSpeech}
}

func TestNewEndOfSpeech_ReturnsPipecat(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	endOfSpeech, err := newEndOfSpeechForTest(context.Background(), logger, mockCallback, pipecatOptions())

	require.NoError(t, err)
	assert.NotNil(t, endOfSpeech)
	assert.Equal(t, "pipecatSmartTurnEndOfSpeech", endOfSpeech.Name())
}

func TestNewEndOfSpeech_ReturnsErrorForUnsupportedProvider(t *testing.T) {
	endOfSpeech, err := newEndOfSpeechForTest(
		t.Context(),
		nil,
		mockCallback,
		utils.Option{EndOfSpeechOptionsKeyProvider: EndOfSpeechIdentifier("unsupported")},
	)

	require.Error(t, err)
	assert.Nil(t, endOfSpeech)
	assert.EqualError(t, err, `end_of_speech: unsupported provider "unsupported"`)
}

func TestEndOfSpeechIdentifier_Constants(t *testing.T) {
	assert.Equal(t, EndOfSpeechIdentifier("silence_based_eos"), SilenceBasedEndOfSpeech)
	assert.Equal(t, EndOfSpeechIdentifier("livekit_eos"), LiveKitEndOfSpeech)
	assert.Equal(t, EndOfSpeechIdentifier("pipecat_smart_turn_eos"), PipecatSmartTurnEndOfSpeech)
	assert.NotEqual(t, SilenceBasedEndOfSpeech, LiveKitEndOfSpeech)
}

func TestNewEndOfSpeech_WithNilLogger(t *testing.T) {
	endOfSpeech, err := newEndOfSpeechForTest(t.Context(), nil, mockCallback, pipecatOptions())

	require.NoError(t, err)
	assert.NotNil(t, endOfSpeech)
	assert.Equal(t, "pipecatSmartTurnEndOfSpeech", endOfSpeech.Name())
}

func TestNewEndOfSpeech_WithNilCallback(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	endOfSpeech, err := New(
		WithContext(t.Context()),
		WithLogger(logger),
		WithOptions(pipecatOptions()),
	)

	require.Error(t, err)
	assert.Nil(t, endOfSpeech)
}

func TestNewEndOfSpeech_WithNilOptionsReturnsError(t *testing.T) {
	logger, _ := commons.NewApplicationLogger()

	endOfSpeech, err := newEndOfSpeechForTest(t.Context(), logger, mockCallback, nil)

	require.Error(t, err)
	assert.Nil(t, endOfSpeech)
	assert.EqualError(t, err, `end_of_speech: unsupported provider ""`)
}
