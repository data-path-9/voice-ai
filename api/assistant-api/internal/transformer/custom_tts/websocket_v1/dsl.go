// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

type requestScope struct {
	Text       string
	MessageID  string
	VoiceID    string
	Model      string
	Language   string
	Encoding   string
	SampleRate int
}

type responseFrame struct {
	Kind   string
	Binary []byte
	JSON   any
}

type responseOutcome struct {
	Matched   bool
	Audio     []byte
	MessageID string
	Done      bool
	ErrorText string
}

type dslEngine struct {
	config *Config
}

func (config *Config) newEngine() *dslEngine {
	return &dslEngine{config: config}
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
	parsed, err := url.Parse(engine.config.BaseURL)
	if err != nil {
		return "", fmt.Errorf("custom-tts websocket_v1: invalid base url: %w", err)
	}
	if len(engine.config.QueryParams) == 0 {
		return parsed.String(), nil
	}

	rendered, err := engine.renderObject(engine.config.QueryParams, scope)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	for key, value := range rendered {
		if value == nil {
			continue
		}
		primitive, err := engine.toPrimitiveString(value)
		if err != nil {
			return "", fmt.Errorf("custom-tts websocket_v1: query param %q must resolve to primitive: %w", key, err)
		}
		query.Set(key, primitive)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (engine *dslEngine) RenderTextRequest(scope requestScope) (map[string]any, error) {
	return engine.renderObject(engine.config.TextRequest, scope)
}

func (engine *dslEngine) RenderDoneRequest(scope requestScope) (map[string]any, error) {
	if !engine.config.HasDoneRequest || engine.config.DoneRequest == nil {
		return nil, nil
	}
	return engine.renderObject(engine.config.DoneRequest, scope)
}

func (engine *dslEngine) renderObject(template map[string]any, scope requestScope) (map[string]any, error) {
	rendered, err := engine.renderNode(template, scope)
	if err != nil {
		return nil, err
	}
	object, ok := rendered.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("custom-tts websocket_v1: rendered request must be an object")
	}
	return object, nil
}

func (engine *dslEngine) renderNode(node any, scope requestScope) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		if rawVar, ok := typed["$var"]; ok {
			if len(typed) != 1 {
				return nil, fmt.Errorf("custom-tts websocket_v1: $var expression must only include $var")
			}
			varName, ok := rawVar.(string)
			if !ok || varName == "" {
				return nil, fmt.Errorf("custom-tts websocket_v1: $var must be non-empty string")
			}
			return engine.resolveVariable(varName, scope)
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return nil, fmt.Errorf("custom-tts websocket_v1: $cast expression must include only $cast and value")
			}
			castKind, ok := rawCast.(string)
			if !ok || castKind == "" {
				return nil, fmt.Errorf("custom-tts websocket_v1: $cast must be non-empty string")
			}
			valueExpr, found := typed["value"]
			if !found {
				return nil, fmt.Errorf("custom-tts websocket_v1: $cast requires value")
			}
			value, err := engine.renderNode(valueExpr, scope)
			if err != nil {
				return nil, err
			}
			return engine.castValue(castKind, value)
		}

		out := make(map[string]any, len(typed))
		for key, value := range typed {
			resolved, err := engine.renderNode(value, scope)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for index, item := range typed {
			resolved, err := engine.renderNode(item, scope)
			if err != nil {
				return nil, err
			}
			out[index] = resolved
		}
		return out, nil
	default:
		return node, nil
	}
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
		return nil, fmt.Errorf("custom-tts websocket_v1: unknown variable %q", name)
	}
}

func (engine *dslEngine) ParseFrame(messageType int, payload []byte) (responseFrame, error) {
	if messageType == 2 {
		return responseFrame{Kind: frameTypeBinary, Binary: append([]byte(nil), payload...)}, nil
	}

	var decoded any
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(string(payload))))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return responseFrame{}, fmt.Errorf("custom-tts websocket_v1: invalid json frame: %w", err)
	}
	return responseFrame{Kind: frameTypeJSON, JSON: decoded}, nil
}

func (engine *dslEngine) EvaluateResponse(frame responseFrame, defaultMessageID string) (responseOutcome, error) {
	for _, rule := range engine.config.ResponseParser {
		matched, err := engine.matchRule(rule, frame)
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

func (engine *dslEngine) matchRule(rule ResponseRule, frame responseFrame) (bool, error) {
	if rule.When.Frame != "" && rule.When.Frame != frame.Kind {
		return false, nil
	}
	if rule.When.Path == "" {
		return true, nil
	}
	if frame.Kind != frameTypeJSON {
		return false, nil
	}

	value, found := engine.lookupJSONPath(frame.JSON, rule.When.Path)
	if !found {
		return false, nil
	}
	if rule.When.Equals == nil {
		return true, nil
	}
	return engine.valuesEqual(value, rule.When.Equals), nil
}

func (engine *dslEngine) emitOutcome(emit map[string]any, frame responseFrame, defaultMessageID string) (responseOutcome, error) {
	outcome := responseOutcome{MessageID: defaultMessageID}

	for key, expr := range emit {
		value, err := engine.evalResponseExpr(expr, frame)
		if err != nil {
			return responseOutcome{}, err
		}

		switch key {
		case "audio":
			audio, err := engine.toBytes(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Audio = audio
		case "message_id":
			messageID, err := engine.toString(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.MessageID = messageID
		case "done":
			done, err := engine.toBool(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.Done = done
		case "error":
			errorText, err := engine.toString(value)
			if err != nil {
				return responseOutcome{}, err
			}
			outcome.ErrorText = errorText
		}
	}

	return outcome, nil
}

func (engine *dslEngine) evalResponseExpr(expr any, frame responseFrame) (any, error) {
	switch typed := expr.(type) {
	case map[string]any:
		if rawPath, ok := typed["$path"]; ok {
			if len(typed) != 1 {
				return nil, fmt.Errorf("custom-tts websocket_v1: $path expression must only include $path")
			}
			path, ok := rawPath.(string)
			if !ok || path == "" {
				return nil, fmt.Errorf("custom-tts websocket_v1: $path must be non-empty string")
			}
			value, found := engine.lookupJSONPath(frame.JSON, path)
			if !found {
				return nil, fmt.Errorf("custom-tts websocket_v1: response path %q not found", path)
			}
			return value, nil
		}
		if rawFrame, ok := typed["$frame"]; ok {
			if len(typed) != 1 {
				return nil, fmt.Errorf("custom-tts websocket_v1: $frame expression must only include $frame")
			}
			frameType, ok := rawFrame.(string)
			if !ok || frameType == "" {
				return nil, fmt.Errorf("custom-tts websocket_v1: $frame must be non-empty string")
			}
			if frameType != frameTypeBinary {
				return nil, fmt.Errorf("custom-tts websocket_v1: unsupported frame selector %q", frameType)
			}
			if frame.Kind != frameTypeBinary {
				return nil, fmt.Errorf("custom-tts websocket_v1: current frame is not binary")
			}
			return append([]byte(nil), frame.Binary...), nil
		}
		if rawDecode, ok := typed["$decode"]; ok {
			if len(typed) != 2 {
				return nil, fmt.Errorf("custom-tts websocket_v1: $decode expression must include only $decode and value")
			}
			decodeKind, ok := rawDecode.(string)
			if !ok || decodeKind == "" {
				return nil, fmt.Errorf("custom-tts websocket_v1: $decode must be non-empty string")
			}
			valueExpr, found := typed["value"]
			if !found {
				return nil, fmt.Errorf("custom-tts websocket_v1: $decode requires value")
			}
			value, err := engine.evalResponseExpr(valueExpr, frame)
			if err != nil {
				return nil, err
			}
			if decodeKind != "base64" {
				return nil, fmt.Errorf("custom-tts websocket_v1: unsupported decode %q", decodeKind)
			}
			encoded, err := engine.toString(value)
			if err != nil {
				return nil, err
			}
			decoded, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil {
				return nil, fmt.Errorf("custom-tts websocket_v1: base64 decode failed: %w", err)
			}
			return decoded, nil
		}
		if rawCast, ok := typed["$cast"]; ok {
			if len(typed) != 2 {
				return nil, fmt.Errorf("custom-tts websocket_v1: $cast expression must include only $cast and value")
			}
			castKind, ok := rawCast.(string)
			if !ok || castKind == "" {
				return nil, fmt.Errorf("custom-tts websocket_v1: $cast must be non-empty string")
			}
			valueExpr, found := typed["value"]
			if !found {
				return nil, fmt.Errorf("custom-tts websocket_v1: $cast requires value")
			}
			value, err := engine.evalResponseExpr(valueExpr, frame)
			if err != nil {
				return nil, err
			}
			return engine.castValue(castKind, value)
		}
		return expr, nil
	default:
		return expr, nil
	}
}

func (engine *dslEngine) lookupJSONPath(root any, path string) (any, bool) {
	current := root
	for _, part := range strings.Split(path, ".") {
		switch typed := current.(type) {
		case map[string]any:
			next, found := typed[part]
			if !found {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func (engine *dslEngine) castValue(kind string, value any) (any, error) {
	switch kind {
	case "string":
		return engine.toString(value)
	case "number":
		return engine.toNumber(value)
	case "boolean":
		return engine.toBool(value)
	default:
		return nil, fmt.Errorf("custom-tts websocket_v1: unsupported cast %q", kind)
	}
}

func (engine *dslEngine) toPrimitiveString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case bool:
		return strconv.FormatBool(typed), nil
	case json.Number:
		return typed.String(), nil
	case int:
		return strconv.Itoa(typed), nil
	case int8, int16, int32, int64:
		return fmt.Sprintf("%d", typed), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", typed), nil
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("value type %T is not primitive", value)
	}
}

func (engine *dslEngine) toString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	case json.Number:
		return typed.String(), nil
	case nil:
		return "", nil
	default:
		return fmt.Sprintf("%v", typed), nil
	}
}

func (engine *dslEngine) toNumber(value any) (any, error) {
	switch typed := value.(type) {
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			return intValue, nil
		}
		floatValue, err := typed.Float64()
		if err != nil {
			return nil, fmt.Errorf("custom-tts websocket_v1: invalid number %q", typed.String())
		}
		return floatValue, nil
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(typed).Int(), nil
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(typed).Uint()), nil
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if intValue, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
			return intValue, nil
		}
		floatValue, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("custom-tts websocket_v1: cannot cast %q to number", trimmed)
		}
		return floatValue, nil
	default:
		return nil, fmt.Errorf("custom-tts websocket_v1: cannot cast %T to number", value)
	}
}

func (engine *dslEngine) toBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		boolValue, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, fmt.Errorf("custom-tts websocket_v1: cannot cast %q to boolean", typed)
		}
		return boolValue, nil
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			if intValue == 0 || intValue == 1 {
				return intValue == 1, nil
			}
		}
		if floatValue, err := typed.Float64(); err == nil {
			if floatValue == 0 || floatValue == 1 {
				return floatValue == 1, nil
			}
		}
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(typed).Int() != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(typed).Uint() != 0, nil
	case float32:
		return typed != 0, nil
	case float64:
		return typed != 0, nil
	}
	return false, fmt.Errorf("custom-tts websocket_v1: cannot cast %T to boolean", value)
}

func (engine *dslEngine) toBytes(value any) ([]byte, error) {
	switch typed := value.(type) {
	case []byte:
		return append([]byte(nil), typed...), nil
	case string:
		return []byte(typed), nil
	default:
		return nil, fmt.Errorf("custom-tts websocket_v1: cannot cast %T to bytes", value)
	}
}

func (engine *dslEngine) valuesEqual(left, right any) bool {
	return reflect.DeepEqual(engine.normalize(left), engine.normalize(right))
}

func (engine *dslEngine) normalize(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if intValue, err := typed.Int64(); err == nil {
			return intValue
		}
		if floatValue, err := typed.Float64(); err == nil {
			if math.Trunc(floatValue) == floatValue {
				return int64(floatValue)
			}
			return floatValue
		}
		return typed.String()
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case uint:
		return int64(typed)
	case uint8:
		return int64(typed)
	case uint16:
		return int64(typed)
	case uint32:
		return int64(typed)
	case uint64:
		return int64(typed)
	case float32:
		floatValue := float64(typed)
		if math.Trunc(floatValue) == floatValue {
			return int64(floatValue)
		}
		return floatValue
	case float64:
		if math.Trunc(typed) == typed {
			return int64(typed)
		}
		return typed
	default:
		return value
	}
}
