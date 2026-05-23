// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_openai_chat_completions

import (
	"context"

	openai "github.com/openai/openai-go/v3"

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
		logger.Errorf("custom-llm openai_chat_completions: failed to create chat client: %v", err)
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

	llmRequest := newChatCompletionParams(c.logger, options, false)
	llmRequest.Messages = buildHistory(allMessages)
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(llmRequest))
	}

	resp, err := c.client.Chat.Completions.New(ctx, llmRequest)
	if err != nil {
		c.logger.Errorf("custom-llm openai_chat_completions: request failed: %v", err)
		failure := metrics.OnFailure().Build()
		payload := map[string]interface{}{"error": err}
		if resp != nil {
			payload["result"] = resp
		}
		if options.PostHook != nil {
			options.PostHook(payload, failure)
		}
		return nil, failure, err
	}

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
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
				Type: string(tool.Type),
				Function: &protos.FunctionCall{
					Name:      tool.Function.Name,
					Arguments: tool.Function.Arguments,
				},
			})
		}
	}

	protoMsg := &protos.Message{
		Role: internal_custom_llm_common.ChatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	metrics.OnAddMetrics(completionUsageMetrics(resp.Usage)...)
	success := metrics.OnSuccess().Build()
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp}, success)
	}
	return protoMsg, success, nil
}
