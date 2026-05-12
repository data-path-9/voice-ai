// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"context"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	internal_azure_common "github.com/rapidaai/api/integration-api/internal/caller/azure/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type chatCaller struct {
	logger commons.Logger
	client *openai.Client
}

func NewChat(logger commons.Logger, credential *protos.Credential) (internal_callers.Chat, error) {
	client, err := internal_azure_common.NewClient(logger, credential)
	if err != nil {
		logger.Errorf("Failed to create Azure chat_complete chat client: %v", err)
		return nil, err
	}
	return &chatCaller{logger: logger, client: client}, nil
}

func (cc *chatCaller) ChatComplete(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	llmRequest := buildChatResponseOptions(options)
	llmRequest.Input = responses.ResponseNewParamsInputUnion{
		OfInputItemList: buildHistory(allMessages),
	}

	if options.PreHook != nil {
		options.PreHook(utils.ToJson(llmRequest))
	}

	resp, err := cc.client.Responses.New(ctx, llmRequest)
	if err != nil {
		cc.logger.Errorf("chat completion failed to get response from azure %v", err)
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": resp,
			}, metrics.OnFailure().Build())
		}
		return nil, metrics.OnFailure().Build(), err
	}

	assistantMsg := &protos.AssistantMessage{Contents: make([]string, 0), ToolCalls: make([]*protos.ToolCall, 0)}
	if outputText := resp.OutputText(); outputText != "" {
		assistantMsg.Contents = append(assistantMsg.Contents, outputText)
	}
	for _, item := range resp.Output {
		if item.Type != "function_call" {
			continue
		}
		fnCall := item.AsFunctionCall()
		id := fnCall.CallID
		if id == "" {
			id = fnCall.ID
		}
		assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{
			Id:   id,
			Type: "function",
			Function: &protos.FunctionCall{
				Name:      fnCall.Name,
				Arguments: fnCall.Arguments,
			},
		})
	}

	metrics.OnSuccess()
	metrics.OnAddMetrics(internal_azure_common.ResponseUsageMetrics(resp.Usage)...)
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp}, metrics.Build())
	}
	return &protos.Message{
		Role:    chatRoleAssistant,
		Message: &protos.Message_Assistant{Assistant: assistantMsg},
	}, metrics.Build(), nil
}
