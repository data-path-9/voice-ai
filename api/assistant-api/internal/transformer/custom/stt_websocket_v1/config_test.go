// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

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
		optionKeyResponseParser: `[{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"interim":false}}]`,
	}
}

func TestNewConfig_DefaultsAndOptionals(t *testing.T) {
	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
		credentialKeyHeaders:      `{"Authorization":"Bearer token"}`,
	}), baseOptions())
	require.NoError(t, err)

	assert.Equal(t, "wss://example.com/stt", config.BaseURL)
	assert.Equal(t, "Bearer token", config.Headers["Authorization"])
	assert.Equal(t, defaultEncoding, config.Encoding)
	assert.Equal(t, defaultSampleRate, config.SampleRate)
	assert.Empty(t, config.Model)
	assert.Empty(t, config.Language)
	assert.False(t, config.HasAudioRequest)
	assert.Nil(t, config.AudioRequest)
}

func TestNewConfig_WithOverrides(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyModel] = "model-a"
	opts[optionKeyLanguage] = "hi-IN"
	opts[optionKeyEncoding] = "MuLaw8"
	opts[optionKeySampleRate] = "8000"
	opts[optionKeyQueryParams] = `{"sample_rate":{"$cast":"number","value":{"$var":"sample_rate"}}}`
	opts[optionKeyAudioRequest] = `{"audio":{"$var":"audio"},"encoding":{"$var":"encoding"}}`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLSnake: "wss://example.com/stt",
	}), opts)
	require.NoError(t, err)

	assert.Equal(t, "model-a", config.Model)
	assert.Equal(t, "hi-IN", config.Language)
	assert.Equal(t, "MuLaw8", config.Encoding)
	assert.Equal(t, 8000, config.SampleRate)
	assert.True(t, config.HasAudioRequest)
	assert.NotNil(t, config.AudioRequest)
	assert.NotNil(t, config.QueryParams)
}

func TestNewConfig_ValidateRequired(t *testing.T) {
	_, err := NewConfig(testCredential(t, map[string]any{}), baseOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base url")

	opts := baseOptions()
	delete(opts, optionKeyResponseParser)
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseParser)
}

func TestNewConfig_InvalidJSON(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyAudioRequest] = `{"audio":{"$var":"audio"}`
	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyAudioRequest)

	opts = baseOptions()
	opts[optionKeyResponseParser] = `{"bad":"shape"}`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), optionKeyResponseParser)

	opts = baseOptions()
	opts[optionKeyResponseParser] = `[{"when":{"frame":"json"},"emit":{"unexpected":true}}]`
	_, err = NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "emit.unexpected")
}

func TestNewConfig_TextResponseParser(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyResponseParser] = `[{"when":{"frame":"text"},"emit":{"script":{"$frame":"text"},"interim":false}}]`

	config, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.NoError(t, err)
	require.Len(t, config.ResponseParser, 1)
	assert.Equal(t, frameTypeText, config.ResponseParser[0].When.Frame)
}

func TestNewConfig_TextResponseParserRejectsPath(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyResponseParser] = `[{"when":{"frame":"text","path":"type","equals":"partial"},"emit":{"script":{"$frame":"text"},"interim":true}}]`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "when.path cannot be used")
}

func TestNewConfig_QueryParamsRejectAudioVariable(t *testing.T) {
	opts := baseOptions()
	opts[optionKeyQueryParams] = `{"chunk":{"$var":"audio"}}`

	_, err := NewConfig(testCredential(t, map[string]any{
		credentialKeyBaseURLCamel: "wss://example.com/stt",
	}), opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported variable "audio"`)
}
