// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_callers

import (
	"context"
	"fmt"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_xai_common "github.com/rapidaai/api/integration-api/internal/caller/xai/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type chatCaller struct {
	logger   commons.Logger
	apiKey   string
	endpoint string
}

func newGRPCChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	apiKey, err := internal_xai_common.ResolveAPIKey(credential)
	if err != nil {
		logger.Errorf("failed to create xAI chat client: %v", err)
		return nil, err
	}

	return &chatCaller{
		logger:   logger,
		apiKey:   apiKey,
		endpoint: internal_xai_common.ResolveEndpoint(connectionOptions),
	}, nil
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
		err := fmt.Errorf("xai: chat options are required")
		failure := metrics.OnFailure().Build()
		return nil, failure, err
	}

	llmRequest := newCompletionRequest(c.logger, allMessages, options)
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(llmRequest))
	}

	client, conn, err := internal_xai_common.NewChatClient(c.endpoint)
	if err != nil {
		c.logger.Errorf("failed to create xAI chat connection: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error": err,
			}, failure)
		}
		return nil, failure, err
	}
	defer conn.Close()

	resp, err := client.GetCompletion(internal_xai_common.AuthContext(ctx, c.apiKey), llmRequest)
	if err != nil {
		c.logger.Errorf("xAI chat request failed: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error": err,
			}, failure)
		}
		return nil, failure, err
	}

	assistantMsg := buildAssistantMessage(resp.GetOutputs())
	protoMsg := &protos.Message{
		Role: chatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	metrics.OnAddMetrics(internal_xai_common.CompletionUsageMetrics(resp.GetUsage())...)
	success := metrics.OnSuccess().Build()
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp}, success)
	}
	return protoMsg, success, nil
}
