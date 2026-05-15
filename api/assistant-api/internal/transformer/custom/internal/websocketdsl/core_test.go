// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_websocketdsl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoreParseFrame_SupportsBinaryJSONAndText(t *testing.T) {
	core := NewCore("test")

	frame, err := core.ParseFrame(2, []byte{0x01, 0x02}, func(messageType int) bool {
		return messageType == 2
	})
	require.NoError(t, err)
	assert.Equal(t, FrameBinary, frame.Kind)
	assert.Equal(t, []byte{0x01, 0x02}, frame.Binary)

	frame, err = core.ParseFrame(1, []byte(`{"type":"partial","text":"hello"}`), func(messageType int) bool {
		return messageType == 2
	})
	require.NoError(t, err)
	assert.Equal(t, FrameJSON, frame.Kind)
	assert.Equal(t, `{"type":"partial","text":"hello"}`, frame.Text)

	frame, err = core.ParseFrame(1, []byte("namaste"), func(messageType int) bool {
		return messageType == 2
	})
	require.NoError(t, err)
	assert.Equal(t, FrameText, frame.Kind)
	assert.Equal(t, "namaste", frame.Text)
}

func TestCoreMatchWhen_TextFrames(t *testing.T) {
	core := NewCore("test")
	frame := Frame{Kind: FrameText, Text: "namaste"}

	matched, err := core.MatchWhen(When{Frame: FrameText}, frame)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = core.MatchWhen(When{Frame: FrameText, Equals: "namaste"}, frame)
	require.NoError(t, err)
	assert.True(t, matched)

	matched, err = core.MatchWhen(When{Frame: FrameText, Equals: "hello"}, frame)
	require.NoError(t, err)
	assert.False(t, matched)
}

func TestCoreValidateResponseRules_TextContracts(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	contract := Contract{
		SupportedResponseFrames: []string{FrameJSON, FrameText},
		SupportedEmitKeys:       []string{"script", "interim", "error"},
		AllowedFrameSelectors:   []string{FrameText},
	}

	err := core.ValidateResponseRules([]ResponseRule{
		{
			When: When{Frame: FrameText},
			Emit: map[string]any{
				"script":  map[string]any{"$frame": FrameText},
				"interim": true,
			},
		},
	}, contract, "listen.ws.response_parser")
	require.NoError(t, err)

	err = core.ValidateResponseRules([]ResponseRule{
		{
			When: When{Frame: FrameText, Path: "type"},
			Emit: map[string]any{
				"script": map[string]any{"$frame": FrameText},
			},
		},
	}, contract, "listen.ws.response_parser")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "when.path cannot be used")
}

func TestCoreEvalResponseExpr_SupportsTextFrameSelectors(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	frame := Frame{Kind: FrameText, Text: "bonjour"}
	contract := Contract{
		AllowedFrameSelectors: []string{FrameText},
	}

	value, err := core.EvalResponseExpr(map[string]any{
		"script": map[string]any{"$frame": FrameText},
	}, frame, contract)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"script": "bonjour"}, value)
}

func TestCoreValidateRequestObject_RejectsUnknownVariable(t *testing.T) {
	core := NewCore("custom-stt websocket_v1")
	contract := Contract{
		SupportedVariables: []string{"audio", "encoding"},
	}

	err := core.ValidateRequestObject(map[string]any{
		"audio": map[string]any{"$var": "chunk"},
	}, contract, "listen.ws.audio_request")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported variable "chunk"`)
}

func TestCoreValidateQueryParams_RejectsNestedValues(t *testing.T) {
	core := NewCore("custom websocket_v1")
	contract := Contract{
		SupportedVariables: []string{"language", "sample_rate"},
	}

	err := core.ValidateQueryParams(map[string]any{
		"language": map[string]any{"$var": "language"},
		"rate":     map[string]any{"$cast": "number", "value": map[string]any{"$var": "sample_rate"}},
	}, contract, "ws.query_params")
	require.NoError(t, err)

	err = core.ValidateQueryParams(map[string]any{
		"metadata": map[string]any{"lang": "hi"},
	}, contract, "ws.query_params")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must resolve to primitive value")

	err = core.ValidateQueryParams(map[string]any{
		"rates": []any{16000},
	}, contract, "ws.query_params")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must resolve to primitive value")
}
