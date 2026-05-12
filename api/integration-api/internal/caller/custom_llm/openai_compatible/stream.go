// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_openai_compatible

import (
	"context"
	"fmt"
	"net/http"
	"time"

	openai "github.com/openai/openai-go/v3"

	internal_custom_llm_common "github.com/rapidaai/api/integration-api/internal/caller/custom_llm/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type streamCaller struct {
	logger     commons.Logger
	credential *protos.Credential
	client     *openai.Client
	httpClient *http.Client
}

func NewStream(logger commons.Logger, credential *protos.Credential) (internal_callers.ChatStream, error) {
	if _, err := internal_custom_llm_common.ParseClientConfig(logger, credential); err != nil {
		logger.Errorf("custom-llm openai_compatible: failed to create stream client: %v", err)
		return nil, err
	}
	return &streamCaller{logger: logger, credential: credential}, nil
}

func (s *streamCaller) GetCredential() *protos.Credential {
	return s.credential
}

func (s *streamCaller) Connect(ctx context.Context, configuration *protos.StreamChatConfiguration) error {
	_ = ctx
	_ = configuration
	if s.client != nil {
		return nil
	}
	client, httpClient, err := newStreamClient(s.logger, s.credential)
	if err != nil {
		s.logger.Errorf("custom-llm openai_compatible: failed to create stream client: %v", err)
		return err
	}
	s.client = client
	s.httpClient = httpClient
	return nil
}

func (s *streamCaller) Close(ctx context.Context) error {
	_ = ctx
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	s.client = nil
	s.httpClient = nil
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
	if err := s.Connect(ctx, nil); err != nil {
		onError(options.Request.GetRequestId(), err)
		return err
	}
	if s.client == nil {
		err := fmt.Errorf("stream client not connected")
		onError(options.Request.GetRequestId(), err)
		return err
	}

	requestID := ""
	if options != nil && options.Request != nil {
		requestID = options.Request.GetRequestId()
	}

	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time

	completionsOptions := newChatCompletionParams(s.logger, &internal_callers.ChatCompletionOptions{
		AIOptions:       options.AIOptions,
		ToolDefinitions: options.ToolDefinitions,
	}, true)
	completionsOptions.Messages = buildHistory(allMessages)
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(completionsOptions))
	}

	resp := s.client.Chat.Completions.NewStreaming(ctx, completionsOptions)
	if resp.Err() != nil {
		err := resp.Err()
		s.logger.Errorf("custom-llm openai_compatible: stream init failed: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": utils.ToJson(resp),
			}, failure)
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}
	defer resp.Close()

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	contentBuffer := make([]string, 0)
	hasToolCalls := false
	accumulate := openai.ChatCompletionAccumulator{}

	for resp.Next() {
		chatCompletions := resp.Current()
		accumulate.AddChunk(chatCompletions)

		if tool, ok := accumulate.JustFinishedToolCall(); ok {
			hasToolCalls = true
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{
				Id:   tool.ID,
				Type: "function",
				Function: &protos.FunctionCall{
					Name:      tool.Name,
					Arguments: tool.Arguments,
				},
			})
		}

		for i, choice := range chatCompletions.Choices {
			if len(choice.Delta.ToolCalls) > 0 {
				hasToolCalls = true
			}
			content := choice.Delta.Content
			if content == "" {
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
				Role: internal_custom_llm_common.ChatRoleAssistant,
				Message: &protos.Message_Assistant{
					Assistant: &protos.AssistantMessage{Contents: []string{content}},
				},
			}
			if onStream != nil {
				if err := onStream(requestID, tokenMsg); err != nil {
					s.logger.Warnf("custom-llm openai_compatible: onStream error: %v", err)
				}
			}
		}
	}

	if resp.Err() != nil {
		err := resp.Err()
		s.logger.Errorf("custom-llm openai_compatible: stream read failed: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": utils.ToJson(resp),
			}, failure)
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}

	assistantMsg.Contents = contentBuffer
	protoMsg := &protos.Message{
		Role: internal_custom_llm_common.ChatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}
	metrics.OnAddMetrics(completionUsageMetrics(accumulate.Usage)...)
	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}
	success := metrics.OnSuccess().Build()
	if onMetrics != nil {
		onMetrics(requestID, protoMsg, success)
	}
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": utils.ToJson(accumulate)}, success)
	}
	return nil
}
