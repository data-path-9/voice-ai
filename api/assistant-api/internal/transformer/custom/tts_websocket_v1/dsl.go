// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import internal_transformer_custom_websocketdsl "github.com/rapidaai/api/assistant-api/internal/transformer/custom/internal/websocketdsl"

type requestScope struct {
	Text       string
	MessageID  string
	VoiceID    string
	Model      string
	Language   string
	Encoding   string
	SampleRate int
}

type responseFrame = internal_transformer_custom_websocketdsl.Frame

type responseOutcome struct {
	Matched   bool
	Audio     []byte
	MessageID string
	Done      bool
	ErrorText string
}

type dslEngine struct {
	config *Config
	core   *internal_transformer_custom_websocketdsl.Core
}

func (config *Config) newEngine() *dslEngine {
	return &dslEngine{
		config: config,
		core:   internal_transformer_custom_websocketdsl.NewCore("custom-tts websocket_v1"),
	}
}

func (config *Config) newScope(contextID, text string) requestScope {
	return requestScope{
		Text:       text,
		MessageID:  contextID,
		VoiceID:    config.VoiceID,
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

func (engine *dslEngine) RenderTextRequest(scope requestScope) (map[string]any, error) {
	return engine.core.RenderObject(engine.config.TextRequest, func(name string) (any, error) {
		return engine.resolveVariable(name, scope)
	})
}

func (engine *dslEngine) RenderDoneRequest(scope requestScope) (map[string]any, error) {
	if !engine.config.HasDoneRequest || engine.config.DoneRequest == nil {
		return nil, nil
	}
	return engine.core.RenderObject(engine.config.DoneRequest, func(name string) (any, error) {
		return engine.resolveVariable(name, scope)
	})
}

func (engine *dslEngine) resolveVariable(name string, scope requestScope) (any, error) {
	switch name {
	case "text":
		return scope.Text, nil
	case "message_id":
		return scope.MessageID, nil
	case "voice_id":
		return scope.VoiceID, nil
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

func (engine *dslEngine) EvaluateResponse(frame responseFrame, defaultMessageID string) (responseOutcome, error) {
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

		result, err := engine.emitOutcome(rule.Emit, frame, defaultMessageID)
		if err != nil {
			return responseOutcome{}, err
		}
		result.Matched = true
		return result, nil
	}
	return responseOutcome{}, nil
}

func (engine *dslEngine) emitOutcome(emit map[string]any, frame responseFrame, defaultMessageID string) (responseOutcome, error) {
	outcome := responseOutcome{MessageID: defaultMessageID}

	for key, expr := range emit {
		value, err := engine.core.EvalResponseExpr(expr, frame, responseContract)
		if err != nil {
			return responseOutcome{}, err
		}

		switch key {
		case "audio":
			audio, err := engine.core.ToBytes(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Audio = audio
		case "message_id":
			messageID, err := engine.core.ToString(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.MessageID = messageID
		case "done":
			done, err := engine.core.ToBool(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Done = done
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
