// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_chat_v2

import (
	"context"
	"time"

	internal_cohere_common "github.com/rapidaai/api/integration-api/internal/caller/cohere/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type chatCaller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func NewChat(logger commons.Logger, credential *protos.Credential) (internal_callers.Chat, error) {
	if _, err := internal_cohere_common.ResolveAPIKey(credential); err != nil {
		logger.Errorf("Failed to create Cohere chat_v2 chat client: %v", err)
		return nil, err
	}
	return &chatCaller{
		logger:     logger,
		credential: credential,
	}, nil
}

func (cc *chatCaller) ChatComplete(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := internal_cohere_common.NewClient(cc.credential)
	if err != nil {
		return nil, metrics.OnFailure().Build(), err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	chatRequest := buildChatRequest(options)
	chatRequest.Messages = buildHistory(cc.logger, allMessages)

	if options.PreHook != nil {
		options.PreHook(utils.ToJson(*chatRequest))
	}

	resp, err := client.V2.Chat(ctx, chatRequest)
	if err != nil {
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": resp,
			}, metrics.OnFailure().Build())
		}
		return nil, metrics.Build(), err
	}

	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{
			"result": resp,
		}, metrics.OnSuccess().Build())
	}

	if resp.Usage != nil {
		metrics.OnAddMetrics(internal_cohere_common.UsageMetrics(resp.Usage)...)
	}

	return buildProtoMessage(resp), metrics.Build(), nil
}
