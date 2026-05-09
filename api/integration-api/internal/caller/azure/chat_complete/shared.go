// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"encoding/json"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

const (
	chatRoleAssistant = "assistant"
	chatRoleSystem    = "system"
	chatRoleTool      = "tool"
	chatRoleUser      = "user"
)

func buildResponseOptions(opts *internal_callers.ChatCompletionOptions) responses.ResponseNewParams {
	options := responses.ResponseNewParams{Store: openai.Bool(false)}

	if len(opts.ToolDefinitions) > 0 {
		fns := make([]responses.ToolUnionParam, 0, len(opts.ToolDefinitions))
		for _, tl := range opts.ToolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}
			fn := tl.Function
			funcDef := responses.FunctionToolParam{Name: fn.Name, Strict: openai.Bool(false)}
			if fn.Description != "" {
				funcDef.Description = openai.String(fn.Description)
			}
			if fn.Parameters != nil {
				funcDef.Parameters = fn.Parameters.ToMap()
			} else {
				funcDef.Parameters = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}
			fns = append(fns, responses.ToolUnionParam{OfFunction: &funcDef})
		}
		options.Tools = fns
	}

	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options.Model = shared.ResponsesModel(modelName)
			}
		case "model.user":
			if user, err := utils.AnyToString(value); err == nil {
				options.User = openai.String(user)
			}
		case "model.reasoning_effort":
			if re, err := utils.AnyToString(value); err == nil {
				options.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(re)}
			}
		case "model.service_tier":
			if st, err := utils.AnyToString(value); err == nil {
				options.ServiceTier = responses.ResponseNewParamsServiceTier(st)
			}
		case "model.top_logprobs":
			if tl, err := utils.AnyToInt64(value); err == nil {
				options.TopLogprobs = openai.Int(tl)
			}
		case "model.metadata":
			format, _ := utils.AnyToString(value)
			var mtd map[string]string
			if err := json.Unmarshal([]byte(format), &mtd); err == nil {
				options.Metadata = shared.Metadata(mtd)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = openai.Float(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.TopP = openai.Float(topP)
			}
		case "model.max_completion_tokens", "model.max_output_tokens":
			if maxTokens, err := utils.AnyToInt64(value); err == nil {
				options.MaxOutputTokens = openai.Int(maxTokens)
			}
		case "model.store":
			if store, err := utils.AnyToBool(value); err == nil {
				options.Store = openai.Bool(store)
			}
		case "model.tool_choice":
			if choice, err := utils.AnyToString(value); err == nil {
				switch choice {
				case "auto":
					options.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto)}
				case "required":
					options.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsRequired)}
				default:
					options.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone)}
				}
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["type"].(string) {
				case "json_object":
					options.Text.Format = responses.ResponseFormatTextConfigUnionParam{OfJSONObject: &shared.ResponseFormatJSONObjectParam{}}
				case "text":
					options.Text.Format = responses.ResponseFormatTextConfigUnionParam{OfText: &shared.ResponseFormatTextParam{}}
				case "json_schema":
					if schemaData, ok := format["json_schema"].(map[string]interface{}); ok {
						cfg := responses.ResponseFormatTextJSONSchemaConfigParam{Name: "response"}
						if name, ok := schemaData["name"].(string); ok && strings.TrimSpace(name) != "" {
							cfg.Name = name
						}
						if description, ok := schemaData["description"].(string); ok && description != "" {
							cfg.Description = openai.String(description)
						}
						if strict, ok := schemaData["strict"].(bool); ok {
							cfg.Strict = openai.Bool(strict)
						}
						if schema, ok := schemaData["schema"].(map[string]interface{}); ok {
							cfg.Schema = schema
						} else {
							cfg.Schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
						}
						options.Text.Format = responses.ResponseFormatTextConfigUnionParam{OfJSONSchema: &cfg}
					}
				}
			}
		}
	}

	return options
}

func buildHistory(allMessages []*protos.Message) []responses.ResponseInputItemUnionParam {
	msg := make([]responses.ResponseInputItemUnionParam, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case chatRoleUser:
			if user := cntn.GetUser(); user != nil {
				msg = append(msg, responses.ResponseInputItemParamOfMessage(user.GetContent(), responses.EasyInputMessageRoleUser))
			}
		case chatRoleAssistant:
			if assistant := cntn.GetAssistant(); assistant != nil {
				txtContent := strings.Join(assistant.GetContents(), "")
				if txtContent != "" {
					msg = append(msg, responses.ResponseInputItemParamOfMessage(txtContent, responses.EasyInputMessageRoleAssistant))
				}
				for _, ttc := range assistant.GetToolCalls() {
					if ttc.GetFunction() == nil || ttc.GetId() == "" {
						continue
					}
					msg = append(msg, responses.ResponseInputItemParamOfFunctionCall(
						ttc.GetFunction().GetArguments(),
						ttc.GetId(),
						ttc.GetFunction().GetName(),
					))
				}
			}
		case chatRoleSystem:
			if system := cntn.GetSystem(); system != nil && len(system.GetContent()) > 0 {
				msg = append(msg, responses.ResponseInputItemParamOfMessage(system.GetContent(), responses.EasyInputMessageRoleSystem))
			}
		case chatRoleTool:
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					if t.GetId() == "" {
						continue
					}
					msg = append(msg, responses.ResponseInputItemParamOfFunctionCallOutput(t.GetId(), t.GetContent()))
				}
			}
		}
	}
	return msg
}

func hasFunctionCall(items []responses.ResponseOutputItemUnion) bool {
	for _, item := range items {
		if item.Type == "function_call" {
			return true
		}
	}
	return false
}
