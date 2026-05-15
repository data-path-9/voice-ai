// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSLEngine_RenderRequestAndURL(t *testing.T) {
	config := &Config{
		BaseURL:    "wss://example.com/stt?tenant=test",
		Model:      "model-a",
		Language:   "en-US",
		Encoding:   defaultEncoding,
		SampleRate: 16000,
		QueryParams: map[string]any{
			"model": map[string]any{"$var": "model"},
			"sample_rate": map[string]any{
				"$cast": "number",
				"value": map[string]any{"$var": "sample_rate"},
			},
		},
		AudioRequest: map[string]any{
			"audio":    map[string]any{"$var": "audio"},
			"encoding": map[string]any{"$var": "encoding"},
		},
		HasAudioRequest: true,
	}
	engine := config.newEngine()
	scope := config.newScope("AAEC")

	url, err := engine.BuildConnectionURL(scope)
	require.NoError(t, err)
	assert.Contains(t, url, "tenant=test")
	assert.Contains(t, url, "model=model-a")
	assert.Contains(t, url, "sample_rate=16000")

	request, err := engine.RenderAudioRequest(scope)
	require.NoError(t, err)
	assert.Equal(t, "AAEC", request["audio"])
	assert.Equal(t, "LINEAR16", request["encoding"])
}

func TestDSLEngine_ParseAndEvaluateResponse(t *testing.T) {
	config := &Config{
		ResponseParser: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "partial"},
				Emit: map[string]any{
					"script":     map[string]any{"$path": "text"},
					"confidence": map[string]any{"$cast": "number", "value": map[string]any{"$path": "confidence"}},
					"language":   map[string]any{"$path": "language"},
					"interim":    true,
				},
			},
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "final"},
				Emit: map[string]any{
					"script":  map[string]any{"$path": "text"},
					"interim": false,
				},
			},
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "error"},
				Emit: map[string]any{
					"error": map[string]any{"$path": "error.message"},
				},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte(`{"type":"partial","text":"hello","confidence":"0.7","language":"en-US"}`))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "hello", outcome.Script)
	assert.InDelta(t, 0.7, outcome.Confidence, 0.0001)
	assert.Equal(t, "en-US", outcome.Language)
	assert.True(t, outcome.Interim)

	frame, err = engine.ParseFrame(1, []byte(`{"type":"final","text":"hello world"}`))
	require.NoError(t, err)
	outcome, err = engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "hello world", outcome.Script)
	assert.False(t, outcome.Interim)

	frame, err = engine.ParseFrame(1, []byte(`{"type":"error","error":{"message":"bad request"}}`))
	require.NoError(t, err)
	outcome, err = engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "bad request", outcome.ErrorText)
}

func TestDSLEngine_ParseAndEvaluateTextResponse(t *testing.T) {
	config := &Config{
		ResponseParser: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeText},
				Emit: map[string]any{
					"script":   map[string]any{"$frame": frameTypeText},
					"language": "hi",
					"interim":  false,
				},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte("namaste"))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, "namaste", outcome.Script)
	assert.Equal(t, "hi", outcome.Language)
	assert.False(t, outcome.Interim)
}

func TestDSLEngine_InvalidVariable(t *testing.T) {
	config := &Config{
		BaseURL: "wss://example.com/stt",
		QueryParams: map[string]any{
			"bad": map[string]any{"$var": "unknown"},
		},
	}
	engine := config.newEngine()
	_, err := engine.BuildConnectionURL(config.newScope("AAEC"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown variable")
}

func TestDSLEngine_NoMatch(t *testing.T) {
	config := &Config{
		ResponseParser: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "final"},
				Emit: map[string]any{"script": map[string]any{"$path": "text"}, "interim": false},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(1, []byte(`{"type":"partial","text":"hello"}`))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame)
	require.NoError(t, err)
	assert.False(t, outcome.Matched)
}
