// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDSLEngine_BuildConnectionURLAndEvaluateRequestRules(t *testing.T) {
	config := &Config{
		BaseURL:    "wss://example.com/tts?tenant=test",
		VoiceID:    "voice-a",
		Model:      "model-a",
		Language:   "hi-IN",
		Encoding:   defaultEncoding,
		SampleRate: 16000,
		QueryParams: map[string]any{
			"message_id": map[string]any{"$var": "message_id"},
			"sample_rate": map[string]any{
				"$cast": "number",
				"value": map[string]any{"$var": "sample_rate"},
			},
		},
		RequestRules: []RequestRule{
			{
				When: RequestWhen{Packet: requestPacketText},
				Send: RequestSend{
					Frame: frameTypeJSON,
					Body: map[string]any{
						"text":       map[string]any{"$path": "packet.text"},
						"voice_id":   map[string]any{"$path": "config.voice.id"},
						"request_id": map[string]any{"$path": "packet.message_id"},
					},
				},
			},
			{
				When: RequestWhen{Packet: requestPacketDone},
				Send: RequestSend{
					Frame: frameTypeJSON,
					Body: map[string]any{
						"type":       "done",
						"request_id": map[string]any{"$path": "packet.message_id"},
					},
				},
			},
			{
				When: RequestWhen{Packet: requestPacketInterrupt},
				Send: RequestSend{
					Frame: frameTypeText,
					Body:  "interrupt",
				},
			},
		},
	}
	engine := config.newEngine()
	queryScope := config.newQueryScope("ctx-1", "hello")

	url, err := engine.BuildConnectionURL(queryScope)
	require.NoError(t, err)
	assert.Contains(t, url, "tenant=test")
	assert.Contains(t, url, "message_id=ctx-1")
	assert.Contains(t, url, "sample_rate=16000")

	textRequests, err := engine.EvaluateRequestRules(
		requestPacketText,
		config.newRequestScope(requestPacketText, "ctx-1", "hello"),
	)
	require.NoError(t, err)
	require.Len(t, textRequests, 1)
	assert.Equal(t, frameTypeJSON, textRequests[0].Frame)
	assert.Equal(t, map[string]any{
		"text":       "hello",
		"voice_id":   "voice-a",
		"request_id": "ctx-1",
	}, textRequests[0].Body)

	doneRequests, err := engine.EvaluateRequestRules(
		requestPacketDone,
		config.newRequestScope(requestPacketDone, "ctx-1", ""),
	)
	require.NoError(t, err)
	require.Len(t, doneRequests, 1)
	assert.Equal(t, map[string]any{
		"type":       "done",
		"request_id": "ctx-1",
	}, doneRequests[0].Body)

	interruptRequests, err := engine.EvaluateRequestRules(
		requestPacketInterrupt,
		config.newRequestScope(requestPacketInterrupt, "ctx-1", ""),
	)
	require.NoError(t, err)
	require.Len(t, interruptRequests, 1)
	assert.Equal(t, frameTypeText, interruptRequests[0].Frame)
	assert.Equal(t, "interrupt", interruptRequests[0].Body)
}

func TestDSLEngine_ParseAndEvaluateResponse(t *testing.T) {
	audioPayload := []byte("pcm")
	encoded := base64.StdEncoding.EncodeToString(audioPayload)

	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeBinary},
				Emit: map[string]any{"audio": map[string]any{"$frame": "binary"}},
			},
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "chunk"},
				Emit: map[string]any{
					"audio":      map[string]any{"$decode": "base64", "value": map[string]any{"$path": "audio"}},
					"message_id": map[string]any{"$path": "request_id"},
				},
			},
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "is_final", Equals: true},
				Emit: map[string]any{
					"message_id": map[string]any{"$path": "request_id"},
					"done":       true,
				},
			},
		},
	}
	engine := config.newEngine()

	frame, err := engine.ParseFrame(2, []byte("pcm"))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame, "ctx-default")
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, []byte("pcm"), outcome.Audio)
	assert.Equal(t, "ctx-default", outcome.MessageID)

	frame, err = engine.ParseFrame(1, []byte(`{"type":"chunk","audio":"`+encoded+`","request_id":"ctx-2"}`))
	require.NoError(t, err)
	outcome, err = engine.EvaluateResponse(frame, "ctx-default")
	require.NoError(t, err)
	assert.True(t, outcome.Matched)
	assert.Equal(t, audioPayload, outcome.Audio)
	assert.Equal(t, "ctx-2", outcome.MessageID)

	frame, err = engine.ParseFrame(1, []byte(`{"is_final":true,"request_id":"ctx-2"}`))
	require.NoError(t, err)
	outcome, err = engine.EvaluateResponse(frame, "ctx-default")
	require.NoError(t, err)
	assert.True(t, outcome.Done)
	assert.Equal(t, "ctx-2", outcome.MessageID)
}

func TestDSLEngine_InvalidVariable(t *testing.T) {
	config := &Config{
		BaseURL: "wss://example.com/ws",
		QueryParams: map[string]any{
			"bad": map[string]any{"$var": "unknown"},
		},
	}
	engine := config.newEngine()
	_, err := engine.BuildConnectionURL(config.newQueryScope("ctx-1", "hello"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown variable")
}

func TestDSLEngine_CastSampleRateFromString(t *testing.T) {
	config := &Config{
		BaseURL: "wss://example.com/ws",
		QueryParams: map[string]any{
			"sample_rate": map[string]any{
				"$cast": "number",
				"value": "16000",
			},
		},
	}
	engine := config.newEngine()
	url, err := engine.BuildConnectionURL(config.newQueryScope("ctx-1", "hello"))
	require.NoError(t, err)
	assert.Contains(t, url, "sample_rate=16000")
}

func TestDSLEngine_UsesNumbersFromJSON(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "event", Equals: "done"},
				Emit: map[string]any{
					"done": map[string]any{"$cast": "boolean", "value": true},
				},
			},
		},
	}
	engine := config.newEngine()
	frame, err := engine.ParseFrame(1, []byte(`{"event":"done","count":1}`))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame, "ctx")
	require.NoError(t, err)
	assert.True(t, outcome.Done)
}

func TestDSLEngine_NoMatch(t *testing.T) {
	config := &Config{
		ResponseRules: []ResponseRule{
			{
				When: ResponseWhen{Frame: frameTypeJSON, Path: "type", Equals: "done"},
				Emit: map[string]any{"done": true},
			},
		},
	}
	engine := config.newEngine()
	frame, err := engine.ParseFrame(1, []byte(`{"type":"chunk"}`))
	require.NoError(t, err)
	outcome, err := engine.EvaluateResponse(frame, "ctx")
	require.NoError(t, err)
	assert.False(t, outcome.Matched)
}
