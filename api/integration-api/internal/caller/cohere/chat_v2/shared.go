// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_chat_v2

import (
	"strings"

	cohere "github.com/cohere-ai/cohere-go/v2"
	"google.golang.org/protobuf/types/known/anypb"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

func buildHistory(logger commons.Logger, allMessages []*protos.Message) cohere.ChatMessages {
	msgHistory := make([]*cohere.ChatMessageV2, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case "assistant":
			if assistant := cntn.GetAssistant(); assistant != nil {
				msg := &cohere.ChatMessageV2{
					Role:      "assistant",
					Assistant: &cohere.AssistantMessage{},
				}

				if len(assistant.GetContents()) > 0 {
					msg.Assistant.Content = &cohere.AssistantMessageV2Content{
						String: strings.Join(assistant.GetContents(), ""),
					}
				}
				if len(assistant.GetToolCalls()) > 0 {
					fctCall := make([]*cohere.ToolCallV2, 0)
					if err := utils.Cast(assistant.GetToolCalls(), &fctCall); err != nil {
						logger.Errorf("unable to cast to function tool call %v", err)
					}
					msg.Assistant.ToolCalls = fctCall
				}
				msgHistory = append(msgHistory, msg)
			}
		case "system":
			if system := cntn.GetSystem(); system != nil {
				msgHistory = append(msgHistory, &cohere.ChatMessageV2{
					Role: "system",
					System: &cohere.SystemMessageV2{
						Content: &cohere.SystemMessageV2Content{
							String: system.GetContent(),
						},
					},
				})
			}
		case "user":
			if user := cntn.GetUser(); user != nil {
				msgHistory = append(msgHistory, &cohere.ChatMessageV2{
					Role: "user",
					User: &cohere.UserMessageV2{
						Content: &cohere.UserMessageV2Content{
							String: user.GetContent(),
						},
					},
				})
			}
		case "tool":
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					msgHistory = append(msgHistory, &cohere.ChatMessageV2{
						Role: "tool",
						Tool: &cohere.ToolMessageV2{
							ToolCallId: t.GetId(),
							Content: &cohere.ToolMessageV2Content{
								String: t.GetContent(),
							},
						},
					})
				}
			}
		default:
			logger.Warnf("Unknown role: %s and everytihgn", cntn.String())
		}
	}
	return msgHistory
}

func buildChatRequest(opts *internal_callers.ChatCompletionOptions) *cohere.V2ChatRequest {
	options := &cohere.V2ChatRequest{}
	buildChatParams(options, opts.ModelParameter)
	return options
}

func buildStreamRequest(opts *internal_callers.ChatStreamCompletionOptions) *cohere.V2ChatStreamRequest {
	options := &cohere.V2ChatStreamRequest{}
	if len(opts.ToolDefinitions) > 0 {
		options.Tools = make([]*cohere.ToolV2, len(opts.ToolDefinitions))
		for idx, tl := range opts.ToolDefinitions {
			if tl.Function == nil {
				continue
			}
			fn := &cohere.ToolV2Function{
				Name: tl.Function.Name,
			}
			if tl.Function.Parameters != nil {
				fn.Parameters = tl.Function.Parameters.ToMap()
			}
			if tl.Function.Description != "" {
				fn.Description = &tl.Function.Description
			}
			options.Tools[idx] = &cohere.ToolV2{
				Function: fn,
			}
		}
	}
	buildStreamParams(options, opts.ModelParameter)
	return options
}

func buildChatParams(options *cohere.V2ChatRequest, modelParameter map[string]*anypb.Any) {
	for key, value := range modelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = mn
			}
		case "model.max_tokens":
			if mt, err := utils.AnyToInt(value); err == nil {
				options.MaxTokens = utils.Ptr(mt)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.P = utils.Ptr(topP)
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat64(value); err == nil {
				options.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat64(value); err == nil {
				options.PresencePenalty = utils.Ptr(pp)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.StopSequences = strings.Split(stopStr, ",")
			}
		}
	}
}

func buildStreamParams(options *cohere.V2ChatStreamRequest, modelParameter map[string]*anypb.Any) {
	for key, value := range modelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = mn
			}
		case "model.max_tokens":
			if mt, err := utils.AnyToInt(value); err == nil {
				options.MaxTokens = utils.Ptr(mt)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.P = utils.Ptr(topP)
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat64(value); err == nil {
				options.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat64(value); err == nil {
				options.PresencePenalty = utils.Ptr(pp)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.StopSequences = strings.Split(stopStr, ",")
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["type"].(string) {
				case "text":
					options.ResponseFormat = &cohere.ResponseFormatV2{
						Type: "text",
					}
				case "json_object":
					if schemaData, ok := format["json_schema"].(map[string]interface{}); ok {
						options.ResponseFormat = &cohere.ResponseFormatV2{
							Type: "json_object",
							JsonObject: &cohere.JsonResponseFormatV2{
								JsonSchema: schemaData,
							},
						}
					}
				}
			}
		}
	}
}

func buildProtoMessage(resp *cohere.V2ChatResponse) *protos.Message {
	contents := make([]string, 0)
	toolCalls := make([]*protos.ToolCall, 0)

	for _, msg := range resp.GetMessage().GetContent() {
		contents = append(contents, msg.Text.Text)
	}

	for _, tl := range resp.GetMessage().GetToolCalls() {
		var name, args string
		if n := tl.GetFunction().GetName(); n != nil {
			name = *n
		}
		if a := tl.GetFunction().GetArguments(); a != nil {
			args = *a
		}
		toolCalls = append(toolCalls, &protos.ToolCall{
			Id:   tl.GetId(),
			Type: tl.Type(),
			Function: &protos.FunctionCall{
				Name:      name,
				Arguments: args,
			},
		})
	}

	return &protos.Message{
		Role: "assistant",
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents:  contents,
				ToolCalls: toolCalls,
			},
		},
	}
}
