// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	internal_transformer_custom_websocketdsl "github.com/rapidaai/api/assistant-api/internal/transformer/custom/internal/websocketdsl"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	defaultEncoding   = "LINEAR16"
	defaultSampleRate = 16000
)

const (
	credentialKeyBaseURLSnake = "base_url"
	credentialKeyBaseURLCamel = "baseUrl"
	credentialKeyHeaders      = "headers"
)

const (
	optionKeyModel          = "listen.model"
	optionKeyLanguage       = "listen.language"
	optionKeyEncoding       = "listen.audio.encoding"
	optionKeySampleRate     = "listen.audio.sample_rate"
	optionKeyQueryParams    = "listen.ws.query_params"
	optionKeyAudioRequest   = "listen.ws.audio_request"
	optionKeyResponseParser = "listen.ws.response_parser"
)

const (
	frameTypeJSON = "json"
	frameTypeText = "text"
)

var queryContract = internal_transformer_custom_websocketdsl.Contract{
	SupportedVariables: []string{
		"model",
		"language",
		"encoding",
		"sample_rate",
	},
}

var audioRequestContract = internal_transformer_custom_websocketdsl.Contract{
	SupportedVariables: []string{
		"audio",
		"model",
		"language",
		"encoding",
		"sample_rate",
	},
}

var responseContract = internal_transformer_custom_websocketdsl.Contract{
	SupportedResponseFrames: []string{
		frameTypeJSON,
		frameTypeText,
	},
	SupportedEmitKeys: []string{
		"script",
		"confidence",
		"language",
		"interim",
		"error",
	},
	AllowedFrameSelectors: []string{
		frameTypeText,
	},
}

type Config struct {
	BaseURL string
	Headers map[string]string

	Model      string
	Language   string
	Encoding   string
	SampleRate int

	QueryParams  map[string]any
	AudioRequest map[string]any

	HasAudioRequest bool
	ResponseParser  []ResponseRule
}

type ResponseRule = internal_transformer_custom_websocketdsl.ResponseRule
type ResponseWhen = internal_transformer_custom_websocketdsl.When

type configParser struct {
	credential *protos.VaultCredential
	opts       utils.Option
}

func NewConfig(credential *protos.VaultCredential, opts utils.Option) (*Config, error) {
	parser := &configParser{credential: credential, opts: opts}
	config := &Config{
		Headers:     map[string]string{},
		QueryParams: map[string]any{},
		Encoding:    defaultEncoding,
		SampleRate:  defaultSampleRate,
	}

	if err := parser.loadCredential(config); err != nil {
		return nil, err
	}
	if err := parser.loadOptions(config); err != nil {
		return nil, err
	}
	if err := config.validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func (parser *configParser) loadCredential(config *Config) error {
	if parser.credential == nil || parser.credential.GetValue() == nil {
		return fmt.Errorf("custom-stt websocket_v1: base url must be specified in credentials")
	}

	raw := parser.credential.GetValue().AsMap()
	baseURLRaw, found := raw[credentialKeyBaseURLSnake]
	if !found {
		baseURLRaw, found = raw[credentialKeyBaseURLCamel]
	}
	if !found {
		return fmt.Errorf("custom-stt websocket_v1: base url must be specified in credentials")
	}

	baseURL, ok := baseURLRaw.(string)
	if !ok {
		return fmt.Errorf("custom-stt websocket_v1: base url must be a string")
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return fmt.Errorf("custom-stt websocket_v1: base url must not be empty")
	}
	config.BaseURL = baseURL

	if rawHeaders, found := raw[credentialKeyHeaders]; found && rawHeaders != nil {
		headers, err := utils.Option{credentialKeyHeaders: rawHeaders}.GetStringMap(credentialKeyHeaders)
		if err != nil {
			return fmt.Errorf("custom-stt websocket_v1: invalid headers: %w", err)
		}
		if headers != nil {
			config.Headers = headers
		}
	}

	return nil
}

func (parser *configParser) loadOptions(config *Config) error {
	if model, err := parser.opts.GetString(optionKeyModel); err == nil {
		config.Model = strings.TrimSpace(model)
	}
	if language, err := parser.opts.GetString(optionKeyLanguage); err == nil {
		config.Language = strings.TrimSpace(language)
	}
	if encoding, err := parser.opts.GetString(optionKeyEncoding); err == nil && strings.TrimSpace(encoding) != "" {
		config.Encoding = strings.TrimSpace(encoding)
	}
	if rawSampleRate, found := parser.opts[optionKeySampleRate]; found && rawSampleRate != nil {
		sampleRate, err := parser.opts.GetUint32(optionKeySampleRate)
		if err != nil {
			return fmt.Errorf("custom-stt websocket_v1: invalid %s: %w", optionKeySampleRate, err)
		}
		config.SampleRate = int(sampleRate)
	}

	if found, err := parser.decodeJSONObject(optionKeyQueryParams, false, &config.QueryParams); err != nil {
		return err
	} else if found && config.QueryParams == nil {
		config.QueryParams = map[string]any{}
	}

	foundAudioRequest, err := parser.decodeJSONObject(optionKeyAudioRequest, false, &config.AudioRequest)
	if err != nil {
		return err
	}
	config.HasAudioRequest = foundAudioRequest
	if !foundAudioRequest {
		config.AudioRequest = nil
	}

	if _, err := parser.decodeJSONArray(optionKeyResponseParser, true, &config.ResponseParser); err != nil {
		return err
	}

	return nil
}

func (parser *configParser) decodeJSONObject(key string, required bool, destination *map[string]any) (bool, error) {
	raw, found := parser.opts[key]
	if !found || raw == nil {
		if required {
			return false, fmt.Errorf("custom-stt websocket_v1: %s is required", key)
		}
		return false, nil
	}

	payload, err := parser.toJSONBytes(raw, key)
	if err != nil {
		return true, err
	}
	if err := parser.decodeJSON(payload, destination, key); err != nil {
		return true, err
	}
	return true, nil
}

func (parser *configParser) decodeJSONArray(key string, required bool, destination *[]ResponseRule) (bool, error) {
	raw, found := parser.opts[key]
	if !found || raw == nil {
		if required {
			return false, fmt.Errorf("custom-stt websocket_v1: %s is required", key)
		}
		return false, nil
	}

	payload, err := parser.toJSONBytes(raw, key)
	if err != nil {
		return true, err
	}
	if err := parser.decodeJSON(payload, destination, key); err != nil {
		return true, err
	}
	return true, nil
}

func (parser *configParser) toJSONBytes(raw any, key string) ([]byte, error) {
	switch typed := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, fmt.Errorf("custom-stt websocket_v1: invalid %s: value must not be empty", key)
		}
		return []byte(trimmed), nil
	case []byte:
		trimmed := bytes.TrimSpace(typed)
		if len(trimmed) == 0 {
			return nil, fmt.Errorf("custom-stt websocket_v1: invalid %s: value must not be empty", key)
		}
		return trimmed, nil
	default:
		payload, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("custom-stt websocket_v1: invalid %s: %w", key, err)
		}
		return payload, nil
	}
}

func (parser *configParser) decodeJSON(payload []byte, destination any, key string) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("custom-stt websocket_v1: invalid %s: %w", key, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("custom-stt websocket_v1: invalid %s: trailing content after JSON value", key)
		}
		return fmt.Errorf("custom-stt websocket_v1: invalid %s: trailing content after JSON value: %w", key, err)
	}
	return nil
}

func (config *Config) validate() error {
	core := internal_transformer_custom_websocketdsl.NewCore("custom-stt websocket_v1")
	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("custom-stt websocket_v1: base url must be specified in credentials")
	}
	if strings.TrimSpace(config.Encoding) == "" {
		return fmt.Errorf("custom-stt websocket_v1: %s must not be empty", optionKeyEncoding)
	}
	if config.SampleRate <= 0 {
		return fmt.Errorf("custom-stt websocket_v1: %s must be positive", optionKeySampleRate)
	}
	if len(config.ResponseParser) == 0 {
		return fmt.Errorf("custom-stt websocket_v1: %s must contain at least one rule", optionKeyResponseParser)
	}

	if len(config.QueryParams) > 0 {
		if err := core.ValidateQueryParams(config.QueryParams, queryContract, optionKeyQueryParams); err != nil {
			return err
		}
	}
	if config.HasAudioRequest && config.AudioRequest != nil {
		if err := core.ValidateRequestObject(config.AudioRequest, audioRequestContract, optionKeyAudioRequest); err != nil {
			return err
		}
	}
	if err := core.ValidateResponseRules(config.ResponseParser, responseContract, optionKeyResponseParser); err != nil {
		return err
	}

	return nil
}
