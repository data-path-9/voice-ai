// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openrouter_callers

import (
	"context"
	"fmt"

	openrouter "github.com/OpenRouterTeam/go-sdk"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_openrouter_common "github.com/rapidaai/api/integration-api/internal/caller/openrouter/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type chatCaller struct {
	logger commons.Logger
	client *openrouter.OpenRouter
}

func NewChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	_ = connectionOptions

	client, err := internal_openrouter_common.NewClient(credential)
	if err != nil {
		logger.Errorf("failed to create OpenRouter chat client: %v", err)
		return nil, err
	}
	return &chatCaller{logger: logger, client: client}, nil
}

func (c *chatCaller) ChatComplete(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	requestID := uint64(0)
	if options != nil {
		requestID = options.RequestId
	}

	metrics := internal_caller_metrics.NewMetricBuilder(requestID)
	metrics.OnStart()

	if options == nil {
		err := fmt.Errorf("openrouter: chat options are required")
		failure := metrics.OnFailure().Build()
		return nil, failure, err
	}

	llmRequest := newChatRequest(c.logger, options, false)
	llmRequest.Messages = buildHistory(allMessages)
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(llmRequest))
	}

	resp, err := c.client.Chat.Send(ctx, llmRequest)
	if err != nil {
		c.logger.Errorf("openrouter chat request failed: %v", err)
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

	if resp == nil || resp.ChatResult == nil {
		responseType := ""
		if resp != nil {
			responseType = string(resp.Type)
		}
		err = fmt.Errorf("openrouter: unexpected chat response type %q", responseType)
		c.logger.Errorf("openrouter chat returned unexpected response type: %q", responseType)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": resp,
			}, failure)
		}
		return nil, failure, err
	}

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	for _, choice := range resp.ChatResult.GetChoices() {
		appendAssistantMessage(assistantMsg, choice.GetMessage())
	}

	protoMsg := &protos.Message{
		Role: chatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	metrics.OnAddMetrics(internal_openrouter_common.CompletionUsageMetrics(resp.ChatResult.GetUsage())...)
	success := metrics.OnSuccess().Build()
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp.ChatResult}, success)
	}
	return protoMsg, success, nil
}
