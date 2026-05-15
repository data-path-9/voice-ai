// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_websocket_v1

import (
	internal_transformer_custom_websocketdsl "github.com/rapidaai/api/assistant-api/internal/transformer/custom/internal/websocketdsl"
)

type requestScope struct {
	Audio      string
	Model      string
	Language   string
	Encoding   string
	SampleRate int
}

type responseFrame = internal_transformer_custom_websocketdsl.Frame

type responseOutcome struct {
	Matched bool

	Script     string
	Interim    bool
	Confidence float64
	Language   string
	ErrorText  string
}

type dslEngine struct {
	config *Config
	core   *internal_transformer_custom_websocketdsl.Core
}

func (config *Config) newEngine() *dslEngine {
	return &dslEngine{
		config: config,
		core:   internal_transformer_custom_websocketdsl.NewCore("custom-stt websocket_v1"),
	}
}

func (config *Config) newScope(audioBase64 string) requestScope {
	return requestScope{
		Audio:      audioBase64,
		Model:      config.Model,
		Language:   config.Language,
		Encoding:   config.Encoding,
		SampleRate: config.SampleRate,
	}
}

func (engine *dslEngine) BuildConnectionURL(scope requestScope) (string, error) {
	return engine.core.BuildConnectionURL(engine.config.BaseURL, engine.config.QueryParams, func(name string) (any, error) {
		return engine.resolveVariable(name, scope)
	})
}

func (engine *dslEngine) RenderAudioRequest(scope requestScope) (map[string]any, error) {
	if !engine.config.HasAudioRequest || engine.config.AudioRequest == nil {
		return nil, nil
	}
	return engine.core.RenderObject(engine.config.AudioRequest, func(name string) (any, error) {
		return engine.resolveVariable(name, scope)
	})
}

func (engine *dslEngine) resolveVariable(name string, scope requestScope) (any, error) {
	switch name {
	case "audio":
		return scope.Audio, nil
	case "model":
		return scope.Model, nil
	case "language":
		return scope.Language, nil
	case "encoding":
		return scope.Encoding, nil
	case "sample_rate":
		return scope.SampleRate, nil
	default:
		return nil, engine.core.Errorf("unknown variable %q", name)
	}
}

func (engine *dslEngine) ParseFrame(messageType int, payload []byte) (responseFrame, error) {
	return engine.core.ParseFrame(messageType, payload, func(currentType int) bool {
		return currentType == 2
	})
}

func (engine *dslEngine) EvaluateResponse(frame responseFrame) (responseOutcome, error) {
	for _, rule := range engine.config.ResponseParser {
		matched, err := engine.core.MatchWhen(
			internal_transformer_custom_websocketdsl.When{
				Frame:  rule.When.Frame,
				Path:   rule.When.Path,
				Equals: rule.When.Equals,
			},
			frame,
		)
		if err != nil {
			return responseOutcome{}, err
		}
		if !matched {
			continue
		}

		outcome, err := engine.emitOutcome(rule.Emit, frame)
		if err != nil {
			return responseOutcome{}, err
		}
		outcome.Matched = true
		return outcome, nil
	}
	return responseOutcome{}, nil
}

func (engine *dslEngine) emitOutcome(emit map[string]any, frame responseFrame) (responseOutcome, error) {
	outcome := responseOutcome{}

	for key, expr := range emit {
		value, err := engine.core.EvalResponseExpr(expr, frame, responseContract)
		if err != nil {
			return responseOutcome{}, err
		}

		switch key {
		case "script":
			script, err := engine.core.ToString(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Script = script
		case "confidence":
			confidence, err := engine.core.ToFloat64(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Confidence = confidence
		case "language":
			language, err := engine.core.ToString(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Language = language
		case "interim":
			interim, err := engine.core.ToBool(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Interim = interim
		case "error":
			errorText, err := engine.core.ToString(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.ErrorText = errorText
		}
	}

	return outcome, nil
}
