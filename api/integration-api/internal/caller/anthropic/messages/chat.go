// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_anthropic_messages

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"

	internal_anthropic_common "github.com/rapidaai/api/integration-api/internal/caller/anthropic/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type chatCaller struct {
	logger commons.Logger
	client *anthropic.Client
}

func NewChat(logger commons.Logger, credential *protos.Credential) (internal_callers.Chat, error) {
	client, err := internal_anthropic_common.NewClient(credential)
	if err != nil {
		logger.Errorf("Failed to create Anthropic messages chat client: %v", err)
		return nil, err
	}
	return &chatCaller{
		logger: logger,
		client: client,
	}, nil
}

func (cc *chatCaller) ChatComplete(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	instruction, messages := buildHistory(cc.logger, allMessages)
	params := buildMessageNewParams(options)
	params.Messages = messages
	params.System = instruction

	if options.PreHook != nil {
		options.PreHook(utils.ToJson(params))
	}

	resp, err := cc.client.Messages.New(ctx, params)
	if err != nil {
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": resp,
			}, metrics.OnFailure().Build())
		}
		return nil, metrics.Build(), err
	}

	protoMessage := convertAnthropicMessageToProto(*resp)
	metrics.OnAddMetrics(internal_anthropic_common.UsageMetrics(resp.Usage)...)
	metrics.OnSuccess()

	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{
			"result": resp,
		}, metrics.Build())
	}

	return protoMessage, metrics.Build(), nil
}
