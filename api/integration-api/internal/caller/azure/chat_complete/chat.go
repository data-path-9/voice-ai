// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"context"

	openai "github.com/openai/openai-go/v3"

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
	client, err := newClient(credential)
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

	llmRequest := buildChatCompletionOptions(options)
	llmRequest.Messages = buildHistory(allMessages)

	if options.PreHook != nil {
		options.PreHook(utils.ToJson(llmRequest))
	}

	resp, err := cc.client.Chat.Completions.New(ctx, llmRequest)
	if err != nil {
		cc.logger.Errorf("chat completion failed to get chat completion from azure %v", err)
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": resp,
			}, metrics.OnFailure().Build())
		}
		return nil, metrics.OnFailure().Build(), err
	}

	assistantMsg := &protos.AssistantMessage{Contents: make([]string, 0), ToolCalls: make([]*protos.ToolCall, 0)}
	for _, choice := range resp.Choices {
		if choice.Message.Content != "" {
			assistantMsg.Contents = append(assistantMsg.Contents, choice.Message.Content)
		}
		for _, tool := range choice.Message.ToolCalls {
			if tool.Type != "function" {
				continue
			}
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{
				Id:   tool.ID,
				Type: tool.Type,
				Function: &protos.FunctionCall{
					Name:      tool.Function.Name,
					Arguments: tool.Function.Arguments,
				},
			})
		}
	}

	metrics.OnSuccess()
	metrics.OnAddMetrics(completionUsageMetrics(resp.Usage)...)
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp}, metrics.Build())
	}
	return &protos.Message{
		Role:    chatRoleAssistant,
		Message: &protos.Message_Assistant{Assistant: assistantMsg},
	}, metrics.Build(), nil
}
