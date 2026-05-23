// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_openai_responses

import (
	"context"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type chatCaller struct {
	logger commons.Logger
	client *openai.Client
}

func NewChat(logger commons.Logger, credential *protos.Credential) (internal_callers.Chat, error) {
	client, err := newClient(logger, credential)
	if err != nil {
		logger.Errorf("custom-llm openai_responses: failed to create chat client: %v", err)
		return nil, err
	}
	return &chatCaller{logger: logger, client: client}, nil
}

func (c *chatCaller) ChatComplete(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	request := buildResponseParams(options)
	request.Input = responses.ResponseNewParamsInputUnion{OfInputItemList: buildHistory(allMessages)}
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(request))
	}

	resp, err := c.client.Responses.New(ctx, request)
	if err != nil {
		c.logger.Errorf("custom-llm openai_responses: request failed: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": resp,
			}, failure)
		}
		return nil, failure, err
	}

	assistantMessage := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	if outputText := resp.OutputText(); outputText != "" {
		assistantMessage.Contents = append(assistantMessage.Contents, outputText)
	}
	for _, outputItem := range resp.Output {
		if outputItem.Type != "function_call" {
			continue
		}
		functionCall := outputItem.AsFunctionCall()
		callID := functionCall.CallID
		if callID == "" {
			callID = functionCall.ID
		}
		assistantMessage.ToolCalls = append(assistantMessage.ToolCalls, &protos.ToolCall{
			Id:   callID,
			Type: "function",
			Function: &protos.FunctionCall{
				Name:      functionCall.Name,
				Arguments: functionCall.Arguments,
			},
		})
	}

	metrics.OnAddMetrics(buildResponseUsageMetrics(resp.Usage)...)
	success := metrics.OnSuccess().Build()
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp}, success)
	}
	return &protos.Message{
		Role: internal_custom_llm_common.ChatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMessage,
		},
	}, success, nil
}
