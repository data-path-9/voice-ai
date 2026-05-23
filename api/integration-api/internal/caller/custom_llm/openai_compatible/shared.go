// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_openai_compatible

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"

	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func newClient(logger commons.Logger, credential *protos.Credential) (*openai.Client, error) {
	config, err := internal_custom_llm_common.ParseClientConfig(logger, credential)
	if err != nil {
		return nil, err
	}
	client := internal_custom_llm_common.NewOpenAIClient(config)
	return &client, nil
}

func newStreamClient(logger commons.Logger, credential *protos.Credential) (*openai.Client, *http.Client, error) {
	config, err := internal_custom_llm_common.ParseClientConfig(logger, credential)
	if err != nil {
		return nil, nil, err
	}
	client, httpClient := internal_custom_llm_common.NewOpenAIClientWithHTTPClient(config)
	return &client, httpClient, nil
}

func buildHistory(allMessages []*protos.Message) []openai.ChatCompletionMessageParamUnion {
	msg := make([]openai.ChatCompletionMessageParamUnion, 0, len(allMessages))
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case internal_custom_llm_common.ChatRoleUser:
			if user := cntn.GetUser(); user != nil {
				msg = append(msg, openai.UserMessage(user.GetContent()))
			}
		case internal_custom_llm_common.ChatRoleAssistant:
			if assistant := cntn.GetAssistant(); assistant != nil {
				txtContent := strings.Join(assistant.GetContents(), "")
				toolCalls := assistant.GetToolCalls()
				assistantMessage := openai.ChatCompletionAssistantMessageParam{}
				if len(txtContent) > 0 {
					assistantMessage.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
						OfString: openai.String(txtContent),
					}
				}
				if len(toolCalls) > 0 {
					fctCall := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(toolCalls))
					for _, ttc := range toolCalls {
						if ttc.GetFunction() == nil {
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
				if len(txtContent) > 0 || len(toolCalls) > 0 {
					msg = append(msg, openai.ChatCompletionMessageParamUnion{
						OfAssistant: &assistantMessage,
					})
				}
			}
		case internal_custom_llm_common.ChatRoleSystem:
			if system := cntn.GetSystem(); system != nil && system.GetContent() != "" {
				msg = append(msg, openai.SystemMessage(system.GetContent()))
			}
		case internal_custom_llm_common.ChatRoleTool:
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					msg = append(msg, openai.ToolMessage(t.GetContent(), t.GetId()))
				}
			}
		}
	}
	return msg
}

func newChatCompletionParams(
	logger commons.Logger,
	opts *internal_callers.ChatCompletionOptions,
	streaming bool,
) openai.ChatCompletionNewParams {
	options := openai.ChatCompletionNewParams{}
	if streaming {
		options.StreamOptions = openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		}
	}

	if len(opts.ToolDefinitions) > 0 {
		fns := make([]openai.ChatCompletionToolUnionParam, 0, len(opts.ToolDefinitions))
		for _, tl := range opts.ToolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}
			funcDef := shared.FunctionDefinitionParam{Name: tl.Function.Name}
			if tl.Function.Description != "" {
				funcDef.Description = openai.String(tl.Function.Description)
			}
			if tl.Function.Parameters != nil {
				funcDef.Parameters = tl.Function.Parameters.ToMap()
			} else {
				funcDef.Parameters = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}
			fns = append(fns, openai.ChatCompletionFunctionTool(funcDef))
		}
		options.Tools = fns
	}

	directParams := make(map[string]interface{})
	modelParams := make(map[string]interface{})
	for key, value := range opts.ModelParameter {
		if value == nil {
			continue
		}
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options.Model = modelName
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

	extraFields := map[string]interface{}{}
	applyChatCompletionParameters(logger, &options, opts, directParams, extraFields)
	applyChatCompletionParameters(logger, &options, opts, modelParams, extraFields)
	if len(extraFields) > 0 {
		options.SetExtraFields(extraFields)
	}

	return options
}

func applyChatCompletionParameters(
	logger commons.Logger,
	options *openai.ChatCompletionNewParams,
	opts *internal_callers.ChatCompletionOptions,
	params map[string]interface{},
	extraFields map[string]interface{},
) {
	for key, value := range params {
		switch strings.ToLower(key) {
		case "user":
			if user, ok := toString(value); ok {
				options.User = openai.String(user)
			}
		case "reasoning_effort":
			if re, ok := toString(value); ok {
				options.ReasoningEffort = shared.ReasoningEffort(re)
			}
		case "seed":
			if seed, ok := toInt64(value); ok {
				options.Seed = openai.Int(seed)
			}
		case "top_logprobs":
			if topLogprobs, ok := toInt64(value); ok {
				options.TopLogprobs = openai.Int(topLogprobs)
			}
		case "metadata":
			if metadata, ok := toStringMap(value); ok {
				options.Metadata = shared.Metadata(metadata)
			}
		case "frequency_penalty":
			if fp, ok := toFloat64(value); ok {
				options.FrequencyPenalty = openai.Float(fp)
			}
		case "temperature":
			if temp, ok := toFloat64(value); ok {
				options.Temperature = openai.Float(temp)
			}
		case "top_p":
			if topP, ok := toFloat64(value); ok {
				options.TopP = openai.Float(topP)
			}
		case "presence_penalty":
			if pp, ok := toFloat64(value); ok {
				options.PresencePenalty = openai.Float(pp)
			}
		case "max_tokens":
			if maxTokens, ok := toInt64(value); ok {
				options.MaxTokens = openai.Int(maxTokens)
			}
		case "max_completion_tokens":
			if maxCompletionTokens, ok := toInt64(value); ok {
				options.MaxCompletionTokens = openai.Int(maxCompletionTokens)
			}
		case "stop":
			stops := toStringSlice(value)
			if len(stops) == 0 {
				continue
			}
			options.Stop.OfStringArray = append(options.Stop.OfStringArray, stops...)
		case "tool_choice":
			if len(opts.ToolDefinitions) == 0 {
				continue
			}
			choice, ok := toString(value)
			if !ok {
				continue
			}
			switch choice {
			case "auto", "required", "none":
				options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
					OfAuto: openai.String(choice),
				}
			default:
				logger.Warnf("custom-llm openai_compatible: unknown tool_choice %q", choice)
			}
		case "response_format":
			format, ok := toMap(value)
			if !ok {
				continue
			}
			formatType, ok := format["type"].(string)
			if !ok {
				continue
			}
			switch formatType {
			case "json_object":
				options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
					OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
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
		default:
			extraFields[key] = value
		}
	}
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
