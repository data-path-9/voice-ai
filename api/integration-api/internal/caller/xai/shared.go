// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_callers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	internal_xai_artifacts "github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	chatRoleAssistant = "assistant"
	chatRoleSystem    = "system"
	chatRoleTool      = "tool"
	chatRoleUser      = "user"
	defaultJSONSchema = `{"type":"object","properties":{}}`
)

func buildHistory(allMessages []*protos.Message) []*internal_xai_artifacts.Message {
	msg := make([]*internal_xai_artifacts.Message, 0, len(allMessages))
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case chatRoleUser:
			if user := cntn.GetUser(); user != nil && user.GetContent() != "" {
				msg = append(msg, &internal_xai_artifacts.Message{
					Role: internal_xai_artifacts.MessageRole_ROLE_USER,
					Content: []*internal_xai_artifacts.Content{
						newTextContent(user.GetContent()),
					},
				})
			}
		case chatRoleAssistant:
			if assistant := cntn.GetAssistant(); assistant != nil {
				assistantMessage := &internal_xai_artifacts.Message{
					Role:      internal_xai_artifacts.MessageRole_ROLE_ASSISTANT,
					Content:   make([]*internal_xai_artifacts.Content, 0),
					ToolCalls: make([]*internal_xai_artifacts.ToolCall, 0),
				}

				txtContent := strings.Join(assistant.GetContents(), "")
				if txtContent != "" {
					assistantMessage.Content = append(assistantMessage.Content, newTextContent(txtContent))
				}

				for _, ttc := range assistant.GetToolCalls() {
					if ttc.GetFunction() == nil {
						continue
					}
					assistantMessage.ToolCalls = append(assistantMessage.ToolCalls, &internal_xai_artifacts.ToolCall{
						Id:   ttc.GetId(),
						Type: internal_xai_artifacts.ToolCallType_TOOL_CALL_TYPE_CLIENT_SIDE_TOOL,
						Tool: &internal_xai_artifacts.ToolCall_Function{
							Function: &internal_xai_artifacts.FunctionCall{
								Name:      ttc.GetFunction().GetName(),
								Arguments: ttc.GetFunction().GetArguments(),
							},
						},
					})
				}

				if len(assistantMessage.Content) > 0 || len(assistantMessage.ToolCalls) > 0 {
					msg = append(msg, assistantMessage)
				}
			}
		case chatRoleSystem:
			if system := cntn.GetSystem(); system != nil && system.GetContent() != "" {
				msg = append(msg, &internal_xai_artifacts.Message{
					Role: internal_xai_artifacts.MessageRole_ROLE_SYSTEM,
					Content: []*internal_xai_artifacts.Content{
						newTextContent(system.GetContent()),
					},
				})
			}
		case chatRoleTool:
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					toolMessage := &internal_xai_artifacts.Message{
						Role: internal_xai_artifacts.MessageRole_ROLE_TOOL,
					}
					if t.GetContent() != "" {
						toolMessage.Content = []*internal_xai_artifacts.Content{
							newTextContent(t.GetContent()),
						}
					}
					if t.GetId() != "" {
						id := t.GetId()
						toolMessage.ToolCallId = &id
					}
					if len(toolMessage.Content) > 0 || toolMessage.GetToolCallId() != "" {
						msg = append(msg, toolMessage)
					}
				}
			}
		}
	}
	return msg
}

func newTextContent(text string) *internal_xai_artifacts.Content {
	return &internal_xai_artifacts.Content{
		Content: &internal_xai_artifacts.Content_Text{
			Text: text,
		},
	}
}

func newCompletionRequest(
	logger commons.Logger,
	allMessages []*protos.Message,
	opts *internal_callers.ChatCompletionOptions,
) *internal_xai_artifacts.GetCompletionsRequest {
	request := &internal_xai_artifacts.GetCompletionsRequest{
		Messages: buildHistory(allMessages),
	}

	if len(opts.ToolDefinitions) > 0 {
		request.Tools = make([]*internal_xai_artifacts.Tool, 0, len(opts.ToolDefinitions))
		for _, tl := range opts.ToolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}

			parameters := map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
			if tl.Function.Parameters != nil {
				parameters = tl.Function.Parameters.ToMap()
			}

			parametersJSON, err := json.Marshal(parameters)
			if err != nil {
				parametersJSON = []byte(defaultJSONSchema)
			}

			request.Tools = append(request.Tools, &internal_xai_artifacts.Tool{
				Tool: &internal_xai_artifacts.Tool_Function{
					Function: &internal_xai_artifacts.Function{
						Name:        tl.Function.Name,
						Description: tl.Function.Description,
						Parameters:  string(parametersJSON),
					},
				},
			})
		}
	}

	directParams := make(map[string]interface{})
	modelParams := make(map[string]interface{})
	for key, value := range opts.ModelParameter {
		if value == nil {
			continue
		}
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil && strings.TrimSpace(modelName) != "" {
				request.Model = strings.TrimSpace(modelName)
			}
		case "model.parameters":
			if asJSON, err := utils.AnyToJSON(value); err == nil {
				modelParams = asJSON
			}
		default:
			if !strings.HasPrefix(key, "model.") {
				continue
			}
			rawValue, err := utils.AnyToInterface(value)
			if err != nil {
				continue
			}
			directParams[strings.TrimPrefix(key, "model.")] = rawValue
		}
	}

	applyCompletionParameters(logger, request, opts, directParams)
	applyCompletionParameters(logger, request, opts, modelParams)

	return request
}

func applyCompletionParameters(
	logger commons.Logger,
	request *internal_xai_artifacts.GetCompletionsRequest,
	opts *internal_callers.ChatCompletionOptions,
	params map[string]interface{},
) {
	for key, value := range params {
		switch strings.ToLower(key) {
		case "user":
			if user, ok := toString(value); ok {
				request.User = user
			}
		case "n":
			if n, ok := toInt32(value); ok {
				request.N = &n
			}
		case "seed":
			if seed, ok := toInt32(value); ok {
				request.Seed = &seed
			}
		case "top_logprobs":
			if topLogprobs, ok := toInt32(value); ok {
				request.TopLogprobs = &topLogprobs
			}
		case "frequency_penalty":
			if fp, ok := toFloat32(value); ok {
				request.FrequencyPenalty = &fp
			}
		case "temperature":
			if temp, ok := toFloat32(value); ok {
				request.Temperature = &temp
			}
		case "top_p":
			if topP, ok := toFloat32(value); ok {
				request.TopP = &topP
			}
		case "presence_penalty":
			if pp, ok := toFloat32(value); ok {
				request.PresencePenalty = &pp
			}
		case "max_tokens", "max_completion_tokens", "max_output_tokens":
			if maxTokens, ok := toInt32(value); ok {
				request.MaxTokens = &maxTokens
			}
		case "logprobs":
			if logprobs, ok := toBool(value); ok {
				request.Logprobs = logprobs
			}
		case "parallel_tool_calls":
			if parallelToolCalls, ok := toBool(value); ok {
				request.ParallelToolCalls = &parallelToolCalls
			}
		case "stop":
			stops := toStringSlice(value)
			if len(stops) > 0 {
				request.Stop = stops
			}
		case "store", "store_messages":
			if storeMessages, ok := toBool(value); ok {
				request.StoreMessages = storeMessages
			}
		case "use_encrypted_content":
			if encryptedContent, ok := toBool(value); ok {
				request.UseEncryptedContent = encryptedContent
			}
		case "previous_response_id":
			if previousResponseID, ok := toString(value); ok {
				request.PreviousResponseId = &previousResponseID
			}
		case "max_turns":
			if maxTurns, ok := toInt32(value); ok {
				request.MaxTurns = &maxTurns
			}
		case "reasoning_effort":
			if reasoningEffort, ok := toReasoningEffort(value); ok {
				request.ReasoningEffort = &reasoningEffort
			}
		case "tool_choice":
			if len(opts.ToolDefinitions) == 0 {
				continue
			}
			if choice, ok := toString(value); ok {
				switch strings.ToLower(strings.TrimSpace(choice)) {
				case "auto":
					request.ToolChoice = &internal_xai_artifacts.ToolChoice{
						ToolChoice: &internal_xai_artifacts.ToolChoice_Mode{
							Mode: internal_xai_artifacts.ToolMode_TOOL_MODE_AUTO,
						},
					}
				case "required":
					request.ToolChoice = &internal_xai_artifacts.ToolChoice{
						ToolChoice: &internal_xai_artifacts.ToolChoice_Mode{
							Mode: internal_xai_artifacts.ToolMode_TOOL_MODE_REQUIRED,
						},
					}
				case "none":
					request.ToolChoice = &internal_xai_artifacts.ToolChoice{
						ToolChoice: &internal_xai_artifacts.ToolChoice_Mode{
							Mode: internal_xai_artifacts.ToolMode_TOOL_MODE_NONE,
						},
					}
				default:
					logger.Warnf("xai: unknown tool_choice %q", choice)
				}
				continue
			}
			if namedToolChoice, ok := toNamedToolChoice(value); ok {
				request.ToolChoice = namedToolChoice
			}
		case "response_format":
			if responseFormat, ok := toResponseFormat(value); ok {
				request.ResponseFormat = responseFormat
			}
		}
	}
}

func toReasoningEffort(value interface{}) (internal_xai_artifacts.ReasoningEffort, bool) {
	asString, ok := toString(value)
	if !ok {
		return internal_xai_artifacts.ReasoningEffort_INVALID_EFFORT, false
	}

	switch strings.ToLower(strings.TrimSpace(asString)) {
	case "low", "effort_low":
		return internal_xai_artifacts.ReasoningEffort_EFFORT_LOW, true
	case "medium", "effort_medium":
		return internal_xai_artifacts.ReasoningEffort_EFFORT_MEDIUM, true
	case "high", "effort_high":
		return internal_xai_artifacts.ReasoningEffort_EFFORT_HIGH, true
	case "none", "effort_none":
		return internal_xai_artifacts.ReasoningEffort_EFFORT_NONE, true
	default:
		return internal_xai_artifacts.ReasoningEffort_INVALID_EFFORT, false
	}
}

func toNamedToolChoice(value interface{}) (*internal_xai_artifacts.ToolChoice, bool) {
	asMap, ok := toMap(value)
	if !ok {
		return nil, false
	}

	choiceType, ok := toString(asMap["type"])
	if !ok || strings.ToLower(strings.TrimSpace(choiceType)) != "function" {
		return nil, false
	}

	if functionValue, ok := asMap["function"]; ok {
		if functionMap, ok := toMap(functionValue); ok {
			if name, ok := toString(functionMap["name"]); ok && strings.TrimSpace(name) != "" {
				return &internal_xai_artifacts.ToolChoice{
					ToolChoice: &internal_xai_artifacts.ToolChoice_FunctionName{
						FunctionName: strings.TrimSpace(name),
					},
				}, true
			}
		}
	}

	if name, ok := toString(asMap["name"]); ok && strings.TrimSpace(name) != "" {
		return &internal_xai_artifacts.ToolChoice{
			ToolChoice: &internal_xai_artifacts.ToolChoice_FunctionName{
				FunctionName: strings.TrimSpace(name),
			},
		}, true
	}

	return nil, false
}

func toResponseFormat(value interface{}) (*internal_xai_artifacts.ResponseFormat, bool) {
	format, ok := toMap(value)
	if !ok {
		return nil, false
	}

	formatType, ok := toString(format["type"])
	if !ok {
		return nil, false
	}

	switch strings.ToLower(strings.TrimSpace(formatType)) {
	case "json_object":
		return &internal_xai_artifacts.ResponseFormat{
			FormatType: internal_xai_artifacts.FormatType_FORMAT_TYPE_JSON_OBJECT,
		}, true
	case "text":
		return &internal_xai_artifacts.ResponseFormat{
			FormatType: internal_xai_artifacts.FormatType_FORMAT_TYPE_TEXT,
		}, true
	case "json_schema":
		schema := defaultJSONSchema
		if schemaValue, ok := format["json_schema"]; ok {
			switch v := schemaValue.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					schema = strings.TrimSpace(v)
				}
			case map[string]interface{}:
				if nestedSchema, ok := v["schema"]; ok {
					switch nested := nestedSchema.(type) {
					case string:
						if strings.TrimSpace(nested) != "" {
							schema = strings.TrimSpace(nested)
						}
					case map[string]interface{}:
						if schemaJSON, err := json.Marshal(nested); err == nil {
							schema = string(schemaJSON)
						}
					}
				} else if looksLikeJSONSchema(v) {
					if schemaJSON, err := json.Marshal(v); err == nil {
						schema = string(schemaJSON)
					}
				}
			}
		}

		return &internal_xai_artifacts.ResponseFormat{
			FormatType: internal_xai_artifacts.FormatType_FORMAT_TYPE_JSON_SCHEMA,
			Schema:     &schema,
		}, true
	default:
		return nil, false
	}
}

func looksLikeJSONSchema(schema map[string]interface{}) bool {
	if schema == nil {
		return false
	}
	if _, ok := schema["type"]; ok {
		return true
	}
	if _, ok := schema["properties"]; ok {
		return true
	}
	if _, ok := schema["required"]; ok {
		return true
	}
	return false
}

func buildAssistantMessage(outputs []*internal_xai_artifacts.CompletionOutput) *protos.AssistantMessage {
	message := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	if len(outputs) == 0 {
		return message
	}

	sortedOutputs := make([]*internal_xai_artifacts.CompletionOutput, 0, len(outputs))
	sortedOutputs = append(sortedOutputs, outputs...)
	sort.SliceStable(sortedOutputs, func(i, j int) bool {
		return sortedOutputs[i].GetIndex() < sortedOutputs[j].GetIndex()
	})

	for _, output := range sortedOutputs {
		completionMessage := output.GetMessage()
		if completionMessage == nil {
			continue
		}
		if content := completionMessage.GetContent(); content != "" {
			message.Contents = append(message.Contents, content)
		}
		appendAssistantToolCalls(message, completionMessage.GetToolCalls())
	}
	return message
}

func appendAssistantToolCalls(
	message *protos.AssistantMessage,
	toolCalls []*internal_xai_artifacts.ToolCall,
) {
	for _, toolCall := range toolCalls {
		fn := toolCall.GetFunction()
		if fn == nil {
			continue
		}
		message.ToolCalls = append(message.ToolCalls, &protos.ToolCall{
			Id:   toolCall.GetId(),
			Type: "function",
			Function: &protos.FunctionCall{
				Name:      fn.GetName(),
				Arguments: fn.GetArguments(),
			},
		})
	}
}

type streamToolCallAccumulator struct {
	id   string
	name string
	args string
}

func mergeStreamToolCall(
	accumulator map[int64]*streamToolCallAccumulator,
	outputIndex int32,
	toolCallIndex int,
	toolCall *internal_xai_artifacts.ToolCall,
) {
	if toolCall == nil {
		return
	}

	key := streamToolCallKey(outputIndex, toolCallIndex)
	item, ok := accumulator[key]
	if !ok {
		item = &streamToolCallAccumulator{}
		accumulator[key] = item
	}

	if id := toolCall.GetId(); id != "" {
		item.id = mergeChunkText(item.id, id)
	}

	function := toolCall.GetFunction()
	if function == nil {
		return
	}

	if name := function.GetName(); name != "" {
		item.name = mergeChunkText(item.name, name)
	}
	if arguments := function.GetArguments(); arguments != "" {
		item.args = mergeChunkText(item.args, arguments)
	}
}

func streamToolCallKey(outputIndex int32, toolCallIndex int) int64 {
	return (int64(uint32(outputIndex)) << 32) | int64(uint32(toolCallIndex))
}

func finalizeStreamToolCalls(accumulator map[int64]*streamToolCallAccumulator) []*protos.ToolCall {
	if len(accumulator) == 0 {
		return nil
	}

	keys := make([]int64, 0, len(accumulator))
	for key := range accumulator {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	toolCalls := make([]*protos.ToolCall, 0, len(keys))
	for _, key := range keys {
		item := accumulator[key]
		if item == nil {
			continue
		}
		if item.id == "" && item.name == "" && item.args == "" {
			continue
		}
		toolCalls = append(toolCalls, &protos.ToolCall{
			Id:   item.id,
			Type: "function",
			Function: &protos.FunctionCall{
				Name:      item.name,
				Arguments: item.args,
			},
		})
	}
	return toolCalls
}

func finalizeStreamContents(buffer map[int32]string) []string {
	if len(buffer) == 0 {
		return nil
	}

	indexes := make([]int, 0, len(buffer))
	for idx := range buffer {
		indexes = append(indexes, int(idx))
	}
	sort.Ints(indexes)

	content := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		segment := buffer[int32(idx)]
		if segment == "" {
			continue
		}
		content = append(content, segment)
	}
	return content
}

func mergeChunkText(existing string, next string) string {
	if existing == "" {
		return next
	}
	if next == "" {
		return existing
	}
	if strings.HasPrefix(next, existing) {
		return next
	}
	if strings.HasPrefix(existing, next) {
		return existing
	}
	return existing + next
}

func toString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	case float64:
		return fmt.Sprintf("%v", v), true
	case float32:
		return fmt.Sprintf("%v", v), true
	case int:
		return fmt.Sprintf("%d", v), true
	case int32:
		return fmt.Sprintf("%d", v), true
	case int64:
		return fmt.Sprintf("%d", v), true
	case bool:
		if v {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func toInt32(value interface{}) (int32, bool) {
	switch v := value.(type) {
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	case float64:
		return int32(v), true
	case float32:
		return int32(v), true
	case json.Number:
		i, err := v.Int64()
		return int32(i), err == nil
	case string:
		var n json.Number = json.Number(v)
		i, err := n.Int64()
		return int32(i), err == nil
	default:
		return 0, false
	}
}

func toFloat32(value interface{}) (float32, bool) {
	switch v := value.(type) {
	case float32:
		return v, true
	case float64:
		return float32(v), true
	case int:
		return float32(v), true
	case int32:
		return float32(v), true
	case int64:
		return float32(v), true
	case json.Number:
		f, err := v.Float64()
		return float32(f), err == nil
	case string:
		var n json.Number = json.Number(v)
		f, err := n.Float64()
		return float32(f), err == nil
	default:
		return 0, false
	}
}

func toBool(value interface{}) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "y":
			return true, true
		case "false", "0", "no", "n":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func toStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case string:
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, item := range parts {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			out = append(out, item)
		}
		return out
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := toString(item); ok && str != "" {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}

func toMap(value interface{}) (map[string]interface{}, bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		return v, true
	case string:
		rst := map[string]interface{}{}
		if err := json.Unmarshal([]byte(v), &rst); err != nil {
			return nil, false
		}
		return rst, true
	default:
		return nil, false
	}
}
