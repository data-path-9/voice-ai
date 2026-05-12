// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openrouter_callers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"

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
)

func buildHistory(allMessages []*protos.Message) []components.ChatMessages {
	msg := make([]components.ChatMessages, 0, len(allMessages))
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case chatRoleUser:
			if user := cntn.GetUser(); user != nil {
				msg = append(msg, components.CreateChatMessagesUser(components.ChatUserMessage{
					Content: components.CreateChatUserMessageContentStr(user.GetContent()),
					Role:    components.ChatUserMessageRoleUser,
				}))
			}
		case chatRoleAssistant:
			if assistant := cntn.GetAssistant(); assistant != nil {
				txtContent := strings.Join(assistant.GetContents(), "")
				toolCalls := assistant.GetToolCalls()

				assistantMsg := components.ChatAssistantMessage{Role: components.ChatAssistantMessageRoleAssistant}
				if txtContent != "" {
					content := components.CreateChatAssistantMessageContentStr(txtContent)
					assistantMsg.Content = optionalnullable.From(&content)
				}
				if len(toolCalls) > 0 {
					assistantMsg.ToolCalls = make([]components.ChatToolCall, 0, len(toolCalls))
					for _, ttc := range toolCalls {
						if ttc.GetFunction() == nil {
							continue
						}
						toolType := components.ChatToolCallTypeFunction
						if ttc.GetType() != "" {
							toolType = components.ChatToolCallType(ttc.GetType())
						}
						assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, components.ChatToolCall{
							ID:   ttc.GetId(),
							Type: toolType,
							Function: components.ChatToolCallFunction{
								Name:      ttc.GetFunction().GetName(),
								Arguments: ttc.GetFunction().GetArguments(),
							},
						})
					}
				}
				if txtContent != "" || len(assistantMsg.ToolCalls) > 0 {
					msg = append(msg, components.CreateChatMessagesAssistant(assistantMsg))
				}
			}
		case chatRoleSystem:
			if system := cntn.GetSystem(); system != nil && system.GetContent() != "" {
				msg = append(msg, components.CreateChatMessagesSystem(components.ChatSystemMessage{
					Content: components.CreateChatSystemMessageContentStr(system.GetContent()),
					Role:    components.ChatSystemMessageRoleSystem,
				}))
			}
		case chatRoleTool:
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					msg = append(msg, components.CreateChatMessagesTool(components.ChatToolMessage{
						Content:    components.CreateChatToolMessageContentStr(t.GetContent()),
						Role:       components.ChatToolMessageRoleTool,
						ToolCallID: t.GetId(),
					}))
				}
			}
		}
	}
	return msg
}

func newChatRequest(
	logger commons.Logger,
	opts *internal_callers.ChatCompletionOptions,
	streaming bool,
) components.ChatRequest {
	request := components.ChatRequest{}

	stream := streaming
	request.Stream = &stream
	if streaming {
		includeUsage := true
		streamOptions := components.ChatStreamOptions{IncludeUsage: &includeUsage}
		request.StreamOptions = optionalnullable.From(&streamOptions)
	}

	if len(opts.ToolDefinitions) > 0 {
		fns := make([]components.ChatFunctionTool, 0, len(opts.ToolDefinitions))
		for _, tl := range opts.ToolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}
			funcDef := components.ChatFunctionToolFunctionFunction{Name: tl.Function.Name}
			if tl.Function.Description != "" {
				funcDef.Description = &tl.Function.Description
			}
			if tl.Function.Parameters != nil {
				funcDef.Parameters = tl.Function.Parameters.ToMap()
			} else {
				funcDef.Parameters = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}
			tool := components.ChatFunctionToolFunction{
				Type:     components.ChatFunctionToolTypeFunction,
				Function: funcDef,
			}
			fns = append(fns, components.CreateChatFunctionToolChatFunctionToolFunction(tool))
		}
		request.Tools = fns
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
				request.Model = &modelName
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

	applyChatParameters(logger, &request, opts, directParams)
	applyChatParameters(logger, &request, opts, modelParams)

	return request
}

func applyChatParameters(
	logger commons.Logger,
	request *components.ChatRequest,
	opts *internal_callers.ChatCompletionOptions,
	params map[string]interface{},
) {
	for key, value := range params {
		switch strings.ToLower(key) {
		case "user":
			if user, ok := toString(value); ok {
				request.User = &user
			}
		case "models":
			models := toStringSlice(value)
			if len(models) > 0 {
				request.Models = models
			}
		case "reasoning_effort":
			if re, ok := toString(value); ok {
				reasoning := ensureReasoning(request)
				effort := components.Effort(re)
				reasoning.Effort = optionalnullable.From(&effort)
			}
		case "seed":
			if seed, ok := toInt64(value); ok {
				request.Seed = optionalnullable.From(&seed)
			}
		case "top_logprobs":
			if topLogprobs, ok := toInt64(value); ok {
				request.TopLogprobs = optionalnullable.From(&topLogprobs)
			}
		case "metadata":
			if metadata, ok := toStringMap(value); ok {
				request.Metadata = metadata
			}
		case "frequency_penalty":
			if fp, ok := toFloat64(value); ok {
				request.FrequencyPenalty = optionalnullable.From(&fp)
			}
		case "temperature":
			if temp, ok := toFloat64(value); ok {
				request.Temperature = optionalnullable.From(&temp)
			}
		case "top_p":
			if topP, ok := toFloat64(value); ok {
				request.TopP = optionalnullable.From(&topP)
			}
		case "presence_penalty":
			if pp, ok := toFloat64(value); ok {
				request.PresencePenalty = optionalnullable.From(&pp)
			}
		case "max_tokens":
			if maxTokens, ok := toInt64(value); ok {
				request.MaxTokens = optionalnullable.From(&maxTokens)
			}
		case "max_completion_tokens", "max_output_tokens":
			if maxCompletionTokens, ok := toInt64(value); ok {
				request.MaxCompletionTokens = optionalnullable.From(&maxCompletionTokens)
			}
		case "logprobs":
			if logprobs, ok := toBool(value); ok {
				request.Logprobs = optionalnullable.From(&logprobs)
			}
		case "parallel_tool_calls":
			if parallelToolCalls, ok := toBool(value); ok {
				request.ParallelToolCalls = optionalnullable.From(&parallelToolCalls)
			}
		case "service_tier":
			if st, ok := toString(value); ok {
				tier := components.ChatRequestServiceTier(st)
				request.ServiceTier = optionalnullable.From(&tier)
			}
		case "stop":
			stops := toStringSlice(value)
			if len(stops) == 0 {
				continue
			}
			if len(stops) == 1 {
				stop := components.CreateStopStr(stops[0])
				request.Stop = optionalnullable.From(&stop)
				continue
			}
			stop := components.CreateStopArrayOfStr(stops)
			request.Stop = optionalnullable.From(&stop)
		case "tool_choice":
			if len(opts.ToolDefinitions) == 0 {
				continue
			}
			if choice, ok := toString(value); ok {
				switch choice {
				case "auto":
					tc := components.CreateChatToolChoiceChatToolChoiceAuto(components.ChatToolChoiceAutoAuto)
					request.ToolChoice = &tc
				case "required":
					tc := components.CreateChatToolChoiceChatToolChoiceRequired(components.ChatToolChoiceRequiredRequired)
					request.ToolChoice = &tc
				case "none":
					tc := components.CreateChatToolChoiceChatToolChoiceNone(components.ChatToolChoiceNoneNone)
					request.ToolChoice = &tc
				default:
					logger.Warnf("openrouter: unknown tool_choice %q", choice)
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

func ensureReasoning(request *components.ChatRequest) *components.Reasoning {
	if request.Reasoning == nil {
		request.Reasoning = &components.Reasoning{}
	}
	return request.Reasoning
}

func toNamedToolChoice(value interface{}) (*components.ChatToolChoice, bool) {
	asMap, ok := toMap(value)
	if !ok {
		return nil, false
	}

	choiceType, ok := toString(asMap["type"])
	if !ok || choiceType != "function" {
		return nil, false
	}

	functionValue, ok := asMap["function"]
	if !ok {
		return nil, false
	}
	functionMap, ok := toMap(functionValue)
	if !ok {
		return nil, false
	}
	name, ok := toString(functionMap["name"])
	if !ok || strings.TrimSpace(name) == "" {
		return nil, false
	}

	namedToolChoice := components.CreateChatToolChoiceChatNamedToolChoice(components.ChatNamedToolChoice{
		Type: components.ChatNamedToolChoiceTypeFunction,
		Function: components.ChatNamedToolChoiceFunction{
			Name: name,
		},
	})
	return &namedToolChoice, true
}

func toResponseFormat(value interface{}) (*components.ResponseFormat, bool) {
	format, ok := toMap(value)
	if !ok {
		return nil, false
	}
	formatType, ok := toString(format["type"])
	if !ok {
		return nil, false
	}

	switch formatType {
	case "json_object":
		responseFormat := components.CreateResponseFormatJSONObject(components.FormatJSONObjectConfig{
			Type: components.FormatJSONObjectConfigTypeJSONObject,
		})
		return &responseFormat, true
	case "text":
		responseFormat := components.CreateResponseFormatText(components.ChatFormatTextConfig{
			Type: components.ChatFormatTextConfigTypeText,
		})
		return &responseFormat, true
	case "json_schema":
		schemaData, ok := format["json_schema"].(map[string]interface{})
		if !ok {
			return nil, false
		}

		jsonSchema := components.ChatJSONSchemaConfig{
			Name: "response",
			Schema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		}
		if name, ok := toString(schemaData["name"]); ok && strings.TrimSpace(name) != "" {
			jsonSchema.Name = name
		}
		if description, ok := toString(schemaData["description"]); ok && description != "" {
			jsonSchema.Description = &description
		}
		if strict, ok := toBool(schemaData["strict"]); ok {
			jsonSchema.Strict = optionalnullable.From(&strict)
		}
		if schema, ok := schemaData["schema"].(map[string]interface{}); ok {
			jsonSchema.Schema = schema
		}

		responseFormat := components.CreateResponseFormatJSONSchema(components.ChatFormatJSONSchemaConfig{
			Type:       components.ChatFormatJSONSchemaConfigTypeJSONSchema,
			JSONSchema: jsonSchema,
		})
		return &responseFormat, true
	default:
		return nil, false
	}
}

func appendAssistantMessage(message *protos.AssistantMessage, assistant components.ChatAssistantMessage) {
	appendAssistantContent(message, assistant.GetContent())
	for _, toolCall := range assistant.GetToolCalls() {
		fn := toolCall.GetFunction()
		message.ToolCalls = append(message.ToolCalls, &protos.ToolCall{
			Id:   toolCall.GetID(),
			Type: string(toolCall.GetType()),
			Function: &protos.FunctionCall{
				Name:      fn.Name,
				Arguments: fn.Arguments,
			},
		})
	}
}

func appendAssistantContent(
	message *protos.AssistantMessage,
	content optionalnullable.OptionalNullable[components.ChatAssistantMessageContent],
) {
	contentValue, ok := content.GetOrZero()
	if !ok {
		return
	}

	switch contentValue.Type {
	case components.ChatAssistantMessageContentTypeStr:
		if contentValue.Str != nil && *contentValue.Str != "" {
			message.Contents = append(message.Contents, *contentValue.Str)
		}
	case components.ChatAssistantMessageContentTypeArrayOfChatContentItems:
		for _, item := range contentValue.ArrayOfChatContentItems {
			if text := item.ChatContentText; text != nil && text.GetText() != "" {
				message.Contents = append(message.Contents, text.GetText())
			}
		}
	case components.ChatAssistantMessageContentTypeAny:
		if text, ok := toString(contentValue.Any); ok && text != "" {
			message.Contents = append(message.Contents, text)
		}
	}
}

type streamToolCallAccumulator struct {
	id   string
	name string
	args strings.Builder
}

func mergeStreamToolCall(accumulator map[int64]*streamToolCallAccumulator, toolCall components.ChatStreamToolCall) {
	item, ok := accumulator[toolCall.GetIndex()]
	if !ok {
		item = &streamToolCallAccumulator{}
		accumulator[toolCall.GetIndex()] = item
	}

	if id := toolCall.GetID(); id != nil && *id != "" {
		item.id = mergeChunkText(item.id, *id)
	}

	function := toolCall.GetFunction()
	if function == nil {
		return
	}

	if name := function.GetName(); name != nil && *name != "" {
		item.name = mergeChunkText(item.name, *name)
	}
	if arguments := function.GetArguments(); arguments != nil && *arguments != "" {
		item.args.WriteString(*arguments)
	}
}

func finalizeStreamToolCalls(accumulator map[int64]*streamToolCallAccumulator) []*protos.ToolCall {
	if len(accumulator) == 0 {
		return nil
	}

	indexes := make([]int, 0, len(accumulator))
	for index := range accumulator {
		indexes = append(indexes, int(index))
	}
	sort.Ints(indexes)

	toolCalls := make([]*protos.ToolCall, 0, len(indexes))
	for _, index := range indexes {
		item := accumulator[int64(index)]
		if item == nil {
			continue
		}
		if item.id == "" && item.name == "" && item.args.Len() == 0 {
			continue
		}
		toolCalls = append(toolCalls, &protos.ToolCall{
			Id:   item.id,
			Type: "function",
			Function: &protos.FunctionCall{
				Name:      item.name,
				Arguments: item.args.String(),
			},
		})
	}
	return toolCalls
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
	case int:
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

func toInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case json.Number:
		i, err := v.Int64()
		return i, err == nil
	case string:
		var n json.Number = json.Number(v)
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	case string:
		var n json.Number = json.Number(v)
		f, err := n.Float64()
		return f, err == nil
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

func toStringMap(value interface{}) (map[string]string, bool) {
	switch v := value.(type) {
	case map[string]string:
		return v, true
	case map[string]interface{}:
		out := make(map[string]string, len(v))
		for key, item := range v {
			strValue, ok := toString(item)
			if !ok {
				continue
			}
			out[key] = strValue
		}
		return out, true
	case string:
		asMap := map[string]interface{}{}
		if err := json.Unmarshal([]byte(v), &asMap); err != nil {
			return nil, false
		}
		return toStringMap(asMap)
	default:
		return nil, false
	}
}
