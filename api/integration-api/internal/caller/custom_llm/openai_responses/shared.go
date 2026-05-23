// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_openai_responses

import (
	"fmt"
	"net/http"
	"strings"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"google.golang.org/protobuf/types/known/anypb"

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

func buildResponseParams(
	opts *internal_callers.ChatCompletionOptions,
) responses.ResponseNewParams {
	request := responses.ResponseNewParams{
		Store: openai.Bool(false),
	}
	if len(opts.ToolDefinitions) > 0 {
		tools := make([]responses.ToolUnionParam, 0, len(opts.ToolDefinitions))
		for _, toolDefinition := range opts.ToolDefinitions {
			if toolDefinition.Type != "function" || toolDefinition.Function == nil {
				continue
			}
			function := toolDefinition.Function
			functionTool := responses.FunctionToolParam{
				Name:   function.Name,
				Strict: openai.Bool(false),
			}
			if function.Description != "" {
				functionTool.Description = openai.String(function.Description)
			}
			if function.Parameters != nil {
				functionTool.Parameters = function.Parameters.ToMap()
			} else {
				functionTool.Parameters = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}
			tools = append(tools, responses.ToolUnionParam{OfFunction: &functionTool})
		}
		request.Tools = tools
	}

	directModelParameters := make(map[string]*anypb.Any)
	nestedModelParameters := make(map[string]*anypb.Any)

	for key, value := range opts.ModelParameter {
		if value == nil {
			continue
		}
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				request.Model = shared.ResponsesModel(modelName)
			}
		case "model.parameters":
			if parameterMap, err := utils.AnyToJSON(value); err == nil {
				if anyMap, err := utils.InterfaceMapToAnyMap(parameterMap); err == nil {
					for parameterKey, parameterValue := range anyMap {
						nestedModelParameters[parameterKey] = parameterValue
					}
				}
			}
		default:
			if strings.HasPrefix(key, "model.") {
				directModelParameters[strings.TrimPrefix(key, "model.")] = value
			}
		}
	}

	extraFields := map[string]interface{}{}
	applyResponseParameters(&request, opts, directModelParameters, extraFields)
	applyResponseParameters(&request, opts, nestedModelParameters, extraFields)
	if len(extraFields) > 0 {
		request.SetExtraFields(extraFields)
	}

	return request
}

func applyResponseParameters(
	request *responses.ResponseNewParams,
	opts *internal_callers.ChatCompletionOptions,
	parameters map[string]*anypb.Any,
	extraFields map[string]interface{},
) {
	for rawKey, rawValue := range parameters {
		if rawValue == nil {
			continue
		}
		switch strings.ToLower(rawKey) {
		case "user":
			if user, err := utils.AnyToString(rawValue); err == nil {
				request.User = openai.String(user)
			}
		case "reasoning_effort":
			if effort, err := utils.AnyToString(rawValue); err == nil {
				request.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(effort)}
			}
		case "service_tier":
			if tier, err := utils.AnyToString(rawValue); err == nil {
				request.ServiceTier = responses.ResponseNewParamsServiceTier(tier)
			}
		case "top_logprobs":
			if topLogprobs, err := utils.AnyToInt64(rawValue); err == nil {
				request.TopLogprobs = openai.Int(topLogprobs)
			}
		case "metadata":
			metadataValue, err := utils.AnyToInterface(rawValue)
			if err != nil {
				continue
			}
			metadata, err := utils.Option{rawKey: metadataValue}.GetStringMap(rawKey)
			if err != nil {
				continue
			}
			request.Metadata = shared.Metadata(metadata)
		case "temperature":
			if temperature, err := utils.AnyToFloat64(rawValue); err == nil {
				request.Temperature = openai.Float(temperature)
			}
		case "top_p":
			if topP, err := utils.AnyToFloat64(rawValue); err == nil {
				request.TopP = openai.Float(topP)
			}
		case "max_completion_tokens", "max_output_tokens", "max_tokens":
			if maxOutputTokens, err := utils.AnyToInt64(rawValue); err == nil {
				request.MaxOutputTokens = openai.Int(maxOutputTokens)
			}
		case "store":
			if store, err := utils.AnyToBool(rawValue); err == nil {
				request.Store = openai.Bool(store)
			}
		case "tool_choice":
			if len(opts.ToolDefinitions) == 0 {
				continue
			}
			choice, err := utils.AnyToString(rawValue)
			if err != nil {
				continue
			}
			switch choice {
			case "auto":
				request.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{
					OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto),
				}
			case "required":
				request.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{
					OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsRequired),
				}
			case "none":
				request.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{
					OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone),
				}
			}
		case "response_format":
			responseFormat, err := utils.AnyToJSON(rawValue)
			if err != nil {
				continue
			}
			formatType, ok := responseFormat["type"].(string)
			if !ok {
				continue
			}
			switch formatType {
			case "json_object":
				request.Text.Format = responses.ResponseFormatTextConfigUnionParam{
					OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
				}
			case "text":
				request.Text.Format = responses.ResponseFormatTextConfigUnionParam{
					OfText: &shared.ResponseFormatTextParam{},
				}
			case "json_schema":
				schemaData, ok := responseFormat["json_schema"].(map[string]interface{})
				if !ok {
					continue
				}
				schemaConfig := responses.ResponseFormatTextJSONSchemaConfigParam{Name: "response"}
				if name, ok := schemaData["name"].(string); ok && name != "" {
					schemaConfig.Name = name
				}
				if description, ok := schemaData["description"].(string); ok && description != "" {
					schemaConfig.Description = openai.String(description)
				}
				if strict, ok := schemaData["strict"].(bool); ok {
					schemaConfig.Strict = openai.Bool(strict)
				}
				if schema, ok := schemaData["schema"].(map[string]interface{}); ok {
					schemaConfig.Schema = schema
				} else {
					schemaConfig.Schema = map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					}
				}
				request.Text.Format = responses.ResponseFormatTextConfigUnionParam{
					OfJSONSchema: &schemaConfig,
				}
			}
		default:
			if rawParameter, err := utils.AnyToInterface(rawValue); err == nil {
				extraFields[rawKey] = rawParameter
			}
		}
	}
}

func buildHistory(allMessages []*protos.Message) []responses.ResponseInputItemUnionParam {
	messageHistory := make([]responses.ResponseInputItemUnionParam, 0, len(allMessages))
	for _, message := range allMessages {
		switch message.GetRole() {
		case internal_custom_llm_common.ChatRoleUser:
			if user := message.GetUser(); user != nil {
				messageHistory = append(messageHistory, responses.ResponseInputItemParamOfMessage(
					user.GetContent(),
					responses.EasyInputMessageRoleUser,
				))
			}
		case internal_custom_llm_common.ChatRoleAssistant:
			if assistant := message.GetAssistant(); assistant != nil {
				assistantContent := strings.Join(assistant.GetContents(), "")
				if assistantContent != "" {
					messageHistory = append(messageHistory, responses.ResponseInputItemParamOfMessage(
						assistantContent,
						responses.EasyInputMessageRoleAssistant,
					))
				}
				for _, toolCall := range assistant.GetToolCalls() {
					if toolCall.GetFunction() == nil || toolCall.GetId() == "" {
						continue
					}
					messageHistory = append(messageHistory, responses.ResponseInputItemParamOfFunctionCall(
						toolCall.GetFunction().GetArguments(),
						toolCall.GetId(),
						toolCall.GetFunction().GetName(),
					))
				}
			}
		case internal_custom_llm_common.ChatRoleSystem:
			if system := message.GetSystem(); system != nil && system.GetContent() != "" {
				messageHistory = append(messageHistory, responses.ResponseInputItemParamOfMessage(
					system.GetContent(),
					responses.EasyInputMessageRoleSystem,
				))
			}
		case internal_custom_llm_common.ChatRoleTool:
			if toolMessage := message.GetTool(); toolMessage != nil {
				for _, tool := range toolMessage.GetTools() {
					if tool.GetId() == "" {
						continue
					}
					messageHistory = append(messageHistory, responses.ResponseInputItemParamOfFunctionCallOutput(
						tool.GetId(),
						tool.GetContent(),
					))
				}
			}
		}
	}
	return messageHistory
}

func containsFunctionCall(outputItems []responses.ResponseOutputItemUnion) bool {
	for _, item := range outputItems {
		if item.Type == "function_call" {
			return true
		}
	}
	return false
}

func buildResponseUsageMetrics(usages responses.ResponseUsage) []*protos.Metric {
	return []*protos.Metric{
		{
			Name:        type_enums.OUTPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.OutputTokens),
			Description: "LLM Output token",
		},
		{
			Name:        type_enums.INPUT_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.InputTokens),
			Description: "LLM Input token",
		},
		{
			Name:        type_enums.TOTAL_TOKEN.String(),
			Value:       fmt.Sprintf("%d", usages.TotalTokens),
			Description: "LLM Total token",
		},
	}
}
