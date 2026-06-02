// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import internal_transformer_custom_dsl "github.com/rapidaai/api/assistant-api/internal/transformer/custom/internal/dsl"

type queryScope struct {
	Text       string
	MessageID  string
	VoiceID    string
	Model      string
	Language   string
	Encoding   string
	SampleRate int
}

type outboundRequest struct {
	Frame string
	Body  any
}

type responseFrame = internal_transformer_custom_dsl.Frame

type responseOutcome struct {
	Matched   bool
	Audio     []byte
	MessageID string
	Done      bool
	ErrorText string
}

type dslEngine struct {
	config *Config
	core   *internal_transformer_custom_dsl.Core
}

func (config *Config) newEngine() *dslEngine {
	return &dslEngine{
		config: config,
		core:   internal_transformer_custom_dsl.NewCore("custom-tts websocket_v1"),
	}
}

func (config *Config) newQueryScope(messageID, text string) queryScope {
	return queryScope{
		Text:       text,
		MessageID:  messageID,
		VoiceID:    config.VoiceID,
		Model:      config.Model,
		Language:   config.Language,
		Encoding:   config.Encoding,
		SampleRate: config.SampleRate,
	}
}

func (config *Config) newRequestScope(packet, messageID, text string) map[string]any {
	return map[string]any{
		"config": map[string]any{
			"voice": map[string]any{
				"id": config.VoiceID,
			},
			"model":    config.Model,
			"language": config.Language,
			"audio": map[string]any{
				"encoding":    config.Encoding,
				"sample_rate": config.SampleRate,
			},
		},
		"packet": map[string]any{
			"kind":       packet,
			"message_id": messageID,
			"text":       text,
		},
	}
}

func (engine *dslEngine) BuildConnectionURL(scope queryScope) (string, error) {
	return engine.core.BuildConnectionURL(engine.config.BaseURL, engine.config.QueryParams, func(name string) (any, error) {
		return engine.resolveQueryVariable(name, scope)
	})
}

func (engine *dslEngine) EvaluateRequestRules(packet string, scope map[string]any) ([]outboundRequest, error) {
	requests := make([]outboundRequest, 0, len(engine.config.RequestRules))
	for _, rule := range engine.config.RequestRules {
		if !engine.core.MatchRequestWhen(rule.When, packet) {
			continue
		}

		body, err := engine.core.EvalRequestRuleBody(rule.Send.Body, scope)
		if err != nil {
			return nil, err
		}

		switch rule.Send.Frame {
		case frameTypeBinary:
			payload, err := engine.core.ToBytes(body)
			if err != nil {
				return nil, err
			}
			requests = append(requests, outboundRequest{
				Frame: frameTypeBinary,
				Body:  payload,
			})
		case frameTypeText:
			payload, err := engine.core.ToString(body)
			if err != nil {
				return nil, err
			}
			requests = append(requests, outboundRequest{
				Frame: frameTypeText,
				Body:  payload,
			})
		case frameTypeJSON:
			requests = append(requests, outboundRequest{
				Frame: frameTypeJSON,
				Body:  body,
			})
		default:
			return nil, engine.core.Errorf("unsupported request frame %q", rule.Send.Frame)
		}
	}

	return requests, nil
}

func (engine *dslEngine) HasRequestRules(packet string) bool {
	for _, rule := range engine.config.RequestRules {
		if engine.core.MatchRequestWhen(rule.When, packet) {
			return true
		}
	}
	return false
}

func (engine *dslEngine) resolveQueryVariable(name string, scope queryScope) (any, error) {
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
	for _, rule := range engine.config.ResponseRules {
		matched, err := engine.core.MatchWhen(
			internal_transformer_custom_dsl.When{
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
