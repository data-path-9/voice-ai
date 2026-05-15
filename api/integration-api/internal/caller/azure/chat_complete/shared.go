// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

const (
	chatRoleAssistant = "assistant"
	chatRoleSystem    = "system"
	chatRoleTool      = "tool"
	chatRoleUser      = "user"
)

func buildChatCompletionOptions(opts *internal_callers.ChatCompletionOptions) openai.ChatCompletionNewParams {
	return buildChatCompletionOptionsWithCachePolicy(opts, false)
}

func buildStreamCompletionOptions(opts *internal_callers.ChatCompletionOptions) openai.ChatCompletionNewParams {
	return buildChatCompletionOptionsWithCachePolicy(opts, true)
}

func buildChatCompletionOptionsWithCachePolicy(
	opts *internal_callers.ChatCompletionOptions,
	allowPromptCache bool,
) openai.ChatCompletionNewParams {
	options := openai.ChatCompletionNewParams{}
	if allowPromptCache {
		options.StreamOptions = openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		}
	}

	additionalData := map[string]string{}
	if opts != nil && opts.Request != nil {
		additionalData = opts.Request.GetAdditionalData()
	}
	promptCacheKeySelector := "assistant_id"

	if opts != nil && len(opts.ToolDefinitions) > 0 {
		fns := make([]openai.ChatCompletionToolUnionParam, 0, len(opts.ToolDefinitions))
		for _, tl := range opts.ToolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}
			fn := tl.Function
			funcDef := shared.FunctionDefinitionParam{Name: fn.Name, Strict: openai.Bool(false)}
			if fn.Description != "" {
				funcDef.Description = openai.String(fn.Description)
			}
			if fn.Parameters != nil {
				funcDef.Parameters = fn.Parameters.ToMap()
			} else {
				funcDef.Parameters = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}
			fns = append(fns, openai.ChatCompletionFunctionTool(funcDef))
		}
		options.Tools = fns
	}

	modelParams := make(map[string]interface{})
	if opts != nil {
		for key, value := range opts.ModelParameter {
			if value == nil {
				continue
			}
			switch key {
			case "model.name":
				if modelName, err := utils.AnyToString(value); err == nil {
					options.Model = shared.ChatModel(modelName)
				}
			case "model.parameters":
				if asJSON, err := utils.AnyToJSON(value); err == nil {
					modelParams = asJSON
				}
			}
		}
	}

	extraFields := map[string]interface{}{}
	applyChatCompletionParameters(&options, modelParams, extraFields, allowPromptCache, &promptCacheKeySelector)
	if len(extraFields) > 0 {
		options.SetExtraFields(extraFields)
	}

	if allowPromptCache {
		if cacheKey, ok := buildPromptCacheKey(promptCacheKeySelector, additionalData); ok {
			options.PromptCacheKey = openai.String(cacheKey)
		} else {
			options.PromptCacheRetention = ""
		}
	}

	return options
}

func buildPromptCacheKey(selector string, additionalData map[string]string) (string, bool) {
	assistantProviderModelID := strings.TrimSpace(additionalData["assistant_provider_model_id"])
	assistantID := strings.TrimSpace(additionalData["assistant_id"])
	if assistantProviderModelID == "" || assistantID == "" {
		return "", false
	}

	switch selector {
	case "user_identifier":
		userIdentifier := strings.TrimSpace(additionalData["user_identifier"])
		if userIdentifier == "" {
			return "", false
		}
		return userIdentifier + "__" + assistantProviderModelID + "__" + assistantID, true
	case "conversation_id":
		conversationID := strings.TrimSpace(additionalData["conversation_id"])
		if conversationID == "" {
			return "", false
		}
		return conversationID + "__" + assistantProviderModelID + "__" + assistantID, true
	case "assistant_id":
		fallthrough
	default:
		return assistantProviderModelID + "__" + assistantID, true
	}
}

func applyChatCompletionParameters(
	options *openai.ChatCompletionNewParams,
	params map[string]interface{},
	extraFields map[string]interface{},
	allowPromptCache bool,
	promptCacheKeySelector *string,
) {
	var parsedLogprobs *bool
	var parsedTopLogprobs *int64

	for key, value := range params {
		switch strings.ToLower(key) {
		case "user":
			if user, ok := toString(value); ok {
				options.User = openai.String(user)
			}
		case "reasoning_effort":
			if reasoning, ok := toString(value); ok {
				options.ReasoningEffort = shared.ReasoningEffort(reasoning)
			}
		case "service_tier":
			if serviceTier, ok := toString(value); ok {
				options.ServiceTier = openai.ChatCompletionNewParamsServiceTier(serviceTier)
			}
		case "seed":
			if seed, ok := toInt64(value); ok {
				options.Seed = openai.Int(seed)
			}
		case "top_logprobs":
			if topLogprobs, ok := toInt64(value); ok {
				parsedTopLogprobs = &topLogprobs
			}
		case "logprobs":
			if logprobs, ok := toBool(value); ok {
				parsedLogprobs = &logprobs
			}
		case "metadata":
			if metadata, ok := toStringMap(value); ok {
				options.Metadata = shared.Metadata(metadata)
			}
		case "frequency_penalty":
			if frequencyPenalty, ok := toFloat64(value); ok {
				options.FrequencyPenalty = openai.Float(frequencyPenalty)
			}
		case "temperature":
			if temperature, ok := toFloat64(value); ok {
				options.Temperature = openai.Float(temperature)
			}
		case "top_p":
			if topP, ok := toFloat64(value); ok {
				options.TopP = openai.Float(topP)
			}
		case "presence_penalty":
			if presencePenalty, ok := toFloat64(value); ok {
				options.PresencePenalty = openai.Float(presencePenalty)
			}
		case "max_tokens":
			continue
		case "max_completion_tokens", "max_output_tokens":
			if maxCompletionTokens, ok := toInt64(value); ok {
				options.MaxCompletionTokens = openai.Int(maxCompletionTokens)
			}
		case "n":
			if count, ok := toInt64(value); ok {
				options.N = openai.Int(count)
			}
		case "store":
			if store, ok := toBool(value); ok {
				options.Store = openai.Bool(store)
			}
		case "parallel_tool_calls":
			if parallelToolCalls, ok := toBool(value); ok {
				options.ParallelToolCalls = openai.Bool(parallelToolCalls)
			}
		case "prompt_cache_key":
			if allowPromptCache {
				if selector, ok := toString(value); ok && selector != "" {
					*promptCacheKeySelector = selector
				}
			}
		case "prompt_cache_retention":
			if allowPromptCache {
				if retention, ok := toString(value); ok && retention != "" {
					options.PromptCacheRetention = openai.ChatCompletionNewParamsPromptCacheRetention(retention)
				}
			}
		case "safety_identifier":
			if safetyIdentifier, ok := toString(value); ok {
				options.SafetyIdentifier = openai.String(safetyIdentifier)
			}
		case "verbosity":
			if verbosity, ok := toString(value); ok {
				options.Verbosity = openai.ChatCompletionNewParamsVerbosity(verbosity)
			}
		case "stop":
			stopSequences := toStringSlice(value)
			switch len(stopSequences) {
			case 0:
			case 1:
				options.Stop = openai.ChatCompletionNewParamsStopUnion{OfString: openai.String(stopSequences[0])}
			default:
				options.Stop = openai.ChatCompletionNewParamsStopUnion{OfStringArray: stopSequences}
			}
		case "tool_choice":
			if choice, ok := toString(value); ok {
				switch choice {
				case "auto", "required", "none":
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String(choice),
					}
				default:
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("none"),
					}
				}
			}
		case "response_format":
			format, ok := toMap(value)
			if !ok {
				continue
			}
			formatType, _ := format["type"].(string)
			switch formatType {
			case "json_object":
				options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
					OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
				}
			case "text":
				options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
					OfText: &shared.ResponseFormatTextParam{},
				}
			case "json_schema":
				schemaData, ok := format["json_schema"].(map[string]interface{})
				if !ok {
					continue
				}
				jsonSchemaParam := shared.ResponseFormatJSONSchemaJSONSchemaParam{}
				jsonData, err := json.Marshal(schemaData)
				if err != nil {
					continue
				}
				if err := json.Unmarshal(jsonData, &jsonSchemaParam); err != nil {
					continue
				}
				options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
					OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
						JSONSchema: jsonSchemaParam,
					},
				}
			}
		}
	}

	if parsedTopLogprobs != nil {
		options.TopLogprobs = openai.Int(*parsedTopLogprobs)
		options.Logprobs = openai.Bool(true)
	} else if parsedLogprobs != nil {
		options.Logprobs = openai.Bool(*parsedLogprobs)
	}
}

func buildHistory(allMessages []*protos.Message) []openai.ChatCompletionMessageParamUnion {
	msg := make([]openai.ChatCompletionMessageParamUnion, 0, len(allMessages))
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case chatRoleUser:
			if user := cntn.GetUser(); user != nil {
				msg = append(msg, openai.UserMessage(user.GetContent()))
			}
		case chatRoleAssistant:
			if assistant := cntn.GetAssistant(); assistant != nil {
				txtContent := strings.Join(assistant.GetContents(), "")
				toolCalls := assistant.GetToolCalls()
				assistantMessage := openai.ChatCompletionAssistantMessageParam{}
				if txtContent != "" {
					assistantMessage.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(txtContent),
					}
				}
				if len(toolCalls) > 0 {
					fctCall := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(toolCalls))
					for _, ttc := range toolCalls {
						if ttc.GetFunction() == nil || ttc.GetId() == "" {
							continue
						}
						fctCall = append(fctCall, openai.ChatCompletionMessageToolCallUnionParam{
							OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
								ID: ttc.GetId(),
								Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
									Name:      ttc.GetFunction().GetName(),
									Arguments: ttc.GetFunction().GetArguments(),
								},
							},
						})
					}
					assistantMessage.ToolCalls = fctCall
				}
				if txtContent != "" || len(assistantMessage.ToolCalls) > 0 {
					msg = append(msg, openai.ChatCompletionMessageParamUnion{
						OfAssistant: &assistantMessage,
					})
				}
			}
		case chatRoleSystem:
			if system := cntn.GetSystem(); system != nil && system.GetContent() != "" {
				msg = append(msg, openai.SystemMessage(system.GetContent()))
			}
		case chatRoleTool:
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					if t.GetId() == "" {
						continue
					}
					msg = append(msg, openai.ToolMessage(t.GetContent(), t.GetId()))
				}
			}
		}
	}
	return msg
}

func buildAssistantMessageFromChoices(choices []openai.ChatCompletionChoice) *protos.AssistantMessage {
	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}

	for _, choice := range choices {
		if choice.Message.Content != "" {
			assistantMsg.Contents = append(assistantMsg.Contents, choice.Message.Content)
		}
		for _, tool := range choice.Message.ToolCalls {
			if tool.Type != "function" {
				continue
			}
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{
				Id:   tool.ID,
				Type: string(tool.Type),
				Function: &protos.FunctionCall{
					Name:      tool.Function.Name,
					Arguments: tool.Function.Arguments,
				},
			})
		}
	}

	return assistantMsg
}

func finalizeStreamContentsByChoiceIndex(buffer map[int64]string) []string {
	if len(buffer) == 0 {
		return nil
	}

	indexes := make([]int64, 0, len(buffer))
	for idx := range buffer {
		indexes = append(indexes, idx)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i] < indexes[j]
	})

	content := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		if buffer[idx] == "" {
			continue
		}
		content = append(content, buffer[idx])
	}
	return content
}

func completionUsageMetrics(usages openai.CompletionUsage) []*protos.Metric {
	return []*protos.Metric{
		{
			Name:        type_enums.OUTPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.CompletionTokens),
			Description: "LLM Output token",
		},
		{
			Name:        type_enums.INPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.PromptTokens),
			Description: "LLM Input token",
		},
		{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.TotalTokens),
			Description: "LLM Total token",
		},
	}
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
		n := json.Number(v)
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
		n := json.Number(v)
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
		case "true":
			return true, true
		case "false":
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
