// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"testing"

	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func testCredential(t *testing.T, values map[string]any) *protos.VaultCredential {
	t.Helper()
	value, err := structpb.NewStruct(values)
	require.NoError(t, err)
	return &protos.VaultCredential{Value: value}
}

func baseOptions() utils.Option {
	return utils.Option{
		optionKeyVoiceID:        "voice-1",
		optionKeyTextRequest:    `{"text":{"$var":"text"},"request_id":{"$var":"message_id"}}`,
		optionKeyResponseParser: `[{"when":{"frame":"binary"},"emit":{"audio":{"$frame":"binary"}}}]`,
	}
}

func TestNewConfig_DefaultsAndOptionals(t *testing.T) {
	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
		credentialKeyHeaders:      `{"Authorization":"Bearer token"}`,
	}), baseOptions())
	require.NoError(t, err)

	assert.Equal(t, "wss://example.com/ws", config.BaseURL)
	assert.Equal(t, "Bearer token", config.Headers["Authorization"])
	assert.Equal(t, "voice-1", config.VoiceID)
	assert.Equal(t, defaultEncoding, config.Encoding)
	assert.Equal(t, defaultSampleRate, config.SampleRate)
	assert.Empty(t, config.Model)
	assert.Empty(t, config.Language)
	assert.False(t, config.HasDoneRequest)
}

func TestNewConfig_WithOverrides(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyEncoding] = "MuLaw8"
	opts[optionKeySampleRate] = "48000"
	opts[optionKeyModel] = "my-model"
	opts[optionKeyLanguage] = "hi-IN"
	opts[optionKeyQueryParams] = `{"lang":{"$var":"language"}}`
	opts[optionKeyDoneRequest] = `{"continue":false}`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "wss://example.com/ws",
	}), opts)
	require.NoError(t, err)

	assert.Equal(t, "MuLaw8", config.Encoding)
	assert.Equal(t, 48000, config.SampleRate)
	assert.Equal(t, "my-model", config.Model)
	assert.Equal(t, "hi-IN", config.Language)
	assert.True(t, config.HasDoneRequest)
	assert.NotNil(t, config.QueryParams)
}

func TestNewConfig_ValidateRequired(t *testing.T) {
	_, err := NewConfig(testCredential(t, map[string]any{}), baseOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base url")

	opts := baseOptions()
	delete(opts, optionKeyVoiceID)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyVoiceID)

	opts = baseOptions()
	delete(opts, optionKeyTextRequest)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyTextRequest)
}

func TestNewConfig_InvalidJSON(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyTextRequest] = `{"text":{"$var":"text"}`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyTextRequest)

	opts = baseOptions()
	opts[optionKeyResponseParser] = `{"bad":"shape"}`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseParser)

	opts = baseOptions()
	opts[optionKeyTextRequest] = `{"text":{"$var":"text"}} trailing`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/ws",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trailing content")
}
