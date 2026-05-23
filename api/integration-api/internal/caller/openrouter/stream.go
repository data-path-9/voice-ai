// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openrouter_callers

import (
	"context"
	"fmt"
	"time"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/models/operations"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_openrouter_common "github.com/rapidaai/api/integration-api/internal/caller/openrouter/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type streamCaller struct {
	logger     commons.Logger
	credential *protos.Credential
	client     *openrouter.OpenRouter
}

func NewChatStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	_ = connectionOptions

	client, err := internal_openrouter_common.NewClient(credential)
	if err != nil {
		logger.Errorf("failed to create OpenRouter stream client: %v", err)
		return nil, err
	}
	return &streamCaller{logger: logger, credential: credential, client: client}, nil
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
		err := fmt.Errorf("openrouter: stream options are required")
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
	llmRequest := newChatRequest(s.logger, requestOptions, true)
	llmRequest.Messages = buildHistory(allMessages)
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(llmRequest))
	}

	resp, err := s.client.Chat.Send(
		ctx,
		llmRequest,
		operations.WithAcceptHeaderOverride(operations.AcceptHeaderEnumTextEventStream),
	)
	if err != nil {
		s.logger.Errorf("openrouter stream init failed: %v", err)
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

	if resp == nil {
		err := fmt.Errorf("openrouter: empty stream response")
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

	if resp.ChatResult != nil {
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
		if onMetrics != nil {
			_ = onMetrics(requestID, protoMsg, success)
		}
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"result": resp.ChatResult}, success)
		}
		return nil
	}

	if resp.EventStream == nil {
		err := fmt.Errorf("openrouter: unexpected stream response type %q", resp.Type)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": resp,
			}, failure)
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}
	defer resp.EventStream.Close()

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	contentBuffer := make([]string, 0)
	hasToolCalls := false
	toolCallsAccumulator := make(map[int64]*streamToolCallAccumulator)
	var lastUsage *components.ChatUsage
	var lastChunk *components.ChatStreamChunk
	var streamErr error

	for resp.EventStream.Next() {
		event := resp.EventStream.Value()
		if event == nil {
			continue
		}
		chunk := event.GetData()
		chunkCopy := chunk
		lastChunk = &chunkCopy

		if chunkError := chunk.GetError(); chunkError != nil {
			streamErr = fmt.Errorf(
				"openrouter stream error (%d): %s",
				chunkError.GetCode(),
				chunkError.GetMessage(),
			)
			break
		}

		if chunkUsage := chunk.GetUsage(); chunkUsage != nil {
			lastUsage = chunkUsage
		}

		for i, choice := range chunk.GetChoices() {
			delta := choice.GetDelta()
			for _, toolCall := range delta.GetToolCalls() {
				hasToolCalls = true
				mergeStreamToolCall(toolCallsAccumulator, toolCall)
			}

			content, ok := delta.GetContent().GetOrZero()
			if !ok || content == "" {
				continue
			}
			if len(contentBuffer) <= i {
				contentBuffer = append(contentBuffer, content)
			} else {
				contentBuffer[i] += content
			}
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
				if err := onStream(requestID, tokenMsg); err != nil {
					s.logger.Warnf("openrouter stream onStream error: %v", err)
				}
			}
		}
	}

	if streamErr == nil {
		streamErr = resp.EventStream.Err()
	}
	if streamErr != nil {
		s.logger.Errorf("openrouter stream read failed: %v", streamErr)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  streamErr,
				"result": lastChunk,
			}, failure)
		}
		if onError != nil {
			onError(requestID, streamErr)
		}
		return streamErr
	}

	assistantMsg.Contents = contentBuffer
	assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, finalizeStreamToolCalls(toolCallsAccumulator)...)

	protoMsg := &protos.Message{
		Role: chatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	metrics.OnAddMetrics(internal_openrouter_common.CompletionUsageMetrics(lastUsage)...)
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
