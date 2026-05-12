// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_callers

import (
	"context"
	"fmt"
	"io"
	"time"

	internal_xai_artifacts "github.com/rapidaai/api/integration-api/internal/caller/xai/artifacts/xai/api/v1"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_xai_common "github.com/rapidaai/api/integration-api/internal/caller/xai/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type streamCaller struct {
	logger     commons.Logger
	credential *protos.Credential
	apiKey     string
	endpoint   string
}

func newGRPCStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	apiKey, err := internal_xai_common.ResolveAPIKey(credential)
	if err != nil {
		logger.Errorf("failed to create xAI stream client: %v", err)
		return nil, err
	}

	return &streamCaller{
		logger:     logger,
		credential: credential,
		apiKey:     apiKey,
		endpoint:   internal_xai_common.ResolveEndpoint(connectionOptions),
	}, nil
}

func (s *streamCaller) GetCredential() *protos.Credential {
	return s.credential
}

func (s *streamCaller) Connect(ctx context.Context, configuration *protos.StreamChatConfiguration) error {
	_ = ctx
	_ = configuration
	return nil
}

func (s *streamCaller) Close(ctx context.Context) error {
	_ = ctx
	return nil
}

func (s *streamCaller) Chat(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatStreamCompletionOptions,
	onStream func(string, *protos.Message) error,
	onMetrics func(string, *protos.Message, []*protos.Metric) error,
	onError func(string, error),
) error {
	requestID := ""
	metricRequestID := uint64(0)
	if options != nil {
		metricRequestID = options.RequestId
		if options.Request != nil {
			requestID = options.Request.GetRequestId()
		}
	}

	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(metricRequestID)
	metrics.OnStart()
	var firstTokenTime *time.Time

	if options == nil {
		err := fmt.Errorf("xai: stream options are required")
		failure := metrics.OnFailure().Build()
		if onError != nil {
			onError(requestID, err)
		}
		return errWithPostHook(options, err, failure)
	}

	requestOptions := &internal_callers.ChatCompletionOptions{
		AIOptions:       options.AIOptions,
		ToolDefinitions: options.ToolDefinitions,
	}
	llmRequest := newCompletionRequest(s.logger, allMessages, requestOptions)
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(llmRequest))
	}

	client, conn, err := internal_xai_common.NewChatClient(s.endpoint)
	if err != nil {
		s.logger.Errorf("failed to create xAI stream connection: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error": err,
			}, failure)
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}
	defer conn.Close()

	stream, err := client.GetCompletionChunk(
		internal_xai_common.AuthContext(ctx, s.apiKey),
		llmRequest,
	)
	if err != nil {
		s.logger.Errorf("xAI stream init failed: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error": err,
			}, failure)
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	contentBuffer := make(map[int32]string)
	toolCallsAccumulator := make(map[int64]*streamToolCallAccumulator)
	hasToolCalls := false
	var lastUsage *internal_xai_artifacts.SamplingUsage
	var lastChunk *internal_xai_artifacts.GetChatCompletionChunk

	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			s.logger.Errorf("xAI stream read failed: %v", recvErr)
			failure := metrics.OnFailure().Build()
			if options.PostHook != nil {
				options.PostHook(map[string]interface{}{
					"error":  recvErr,
					"result": lastChunk,
				}, failure)
			}
			if onError != nil {
				onError(requestID, recvErr)
			}
			return recvErr
		}
		if chunk == nil {
			continue
		}

		lastChunk = chunk
		if chunk.GetUsage() != nil {
			lastUsage = chunk.GetUsage()
		}

		for _, output := range chunk.GetOutputs() {
			delta := output.GetDelta()
			if delta == nil {
				continue
			}

			if delta.GetToolCalls() != nil {
				for toolCallIndex, toolCall := range delta.GetToolCalls() {
					hasToolCalls = true
					mergeStreamToolCall(
						toolCallsAccumulator,
						output.GetIndex(),
						toolCallIndex,
						toolCall,
					)
				}
			}

			content := delta.GetContent()
			if content == "" {
				continue
			}

			contentBuffer[output.GetIndex()] += content
			if hasToolCalls {
				continue
			}

			if firstTokenTime == nil {
				now := time.Now()
				firstTokenTime = &now
			}
			tokenMsg := &protos.Message{
				Role: chatRoleAssistant,
				Message: &protos.Message_Assistant{
					Assistant: &protos.AssistantMessage{Contents: []string{content}},
				},
			}
			if onStream != nil {
				if callbackErr := onStream(requestID, tokenMsg); callbackErr != nil {
					s.logger.Warnf("xAI stream onStream error: %v", callbackErr)
				}
			}
		}
	}

	assistantMsg.Contents = finalizeStreamContents(contentBuffer)
	assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, finalizeStreamToolCalls(toolCallsAccumulator)...)
	protoMsg := &protos.Message{
		Role: chatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	metrics.OnAddMetrics(internal_xai_common.CompletionUsageMetrics(lastUsage)...)
	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}

	success := metrics.OnSuccess().Build()
	if onMetrics != nil {
		_ = onMetrics(requestID, protoMsg, success)
	}
	if options.PostHook != nil {
		result := interface{}(protoMsg)
		if lastChunk != nil {
			result = lastChunk
		}
		options.PostHook(map[string]interface{}{"result": result}, success)
	}
	return nil
}

func errWithPostHook(
	options *internal_callers.ChatStreamCompletionOptions,
	err error,
	metrics []*protos.Metric,
) error {
	if options != nil && options.PostHook != nil {
		options.PostHook(map[string]interface{}{
			"error": err,
		}, metrics)
	}
	return err
}
