// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_chat_complete

import (
	"context"
	"fmt"
	"net/http"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_openai_common "github.com/rapidaai/api/integration-api/internal/caller/openai/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type streamCaller struct {
	logger     commons.Logger
	credential *protos.Credential
	client     *openai.Client
	httpClient *http.Client
}

func (s *streamCaller) handleStreamFailure(
	requestID string,
	options *internal_callers.ChatStreamCompletionOptions,
	metrics *internal_caller_metrics.MetricBuilder,
	err error,
	result interface{},
	onError func(string, error),
) error {
	failure := metrics.OnFailure().Build()
	if options != nil && options.PostHook != nil {
		payload := map[string]interface{}{"error": err}
		if result != nil {
			payload["result"] = result
		}
		options.PostHook(payload, failure)
	}
	if onError != nil {
		onError(requestID, err)
	}
	return err
}

func NewStream(logger commons.Logger, credential *protos.Credential) (internal_callers.ChatStream, error) {
	if _, err := internal_openai_common.ResolveAPIKey(credential); err != nil {
		logger.Errorf("Failed to create OpenAI chat_complete stream client: %v", err)
		return nil, err
	}
	return &streamCaller{logger: logger, credential: credential}, nil
}

func (s *streamCaller) Connect(ctx context.Context, configuration *protos.StreamChatConfiguration) error {
	_ = ctx
	_ = configuration
	if s.client != nil {
		return nil
	}
	apiKey, err := internal_openai_common.ResolveAPIKey(s.credential)
	if err != nil {
		s.logger.Errorf("Failed to create OpenAI chat_complete stream client: %v", err)
		return err
	}

	transport := &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxConnsPerHost:     internal_openai_common.StreamMaxConnsPerHost,
		MaxIdleConnsPerHost: internal_openai_common.StreamMaxIdleConnsPerHost,
		MaxIdleConns:        internal_openai_common.StreamMaxIdleConns,
		IdleConnTimeout:     internal_openai_common.StreamIdleConnTimeout,
	}
	s.httpClient = &http.Client{Transport: transport}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(s.httpClient),
	)
	s.client = &client
	return nil
}

func (s *streamCaller) GetCredential() *protos.Credential {
	return s.credential
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
	requestID := ""
	if options != nil && options.Request != nil {
		requestID = options.Request.GetRequestId()
	}

	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	if err := s.Connect(ctx, nil); err != nil {
		return s.handleStreamFailure(requestID, options, metrics, err, nil, onError)
	}

	client := s.client
	if client == nil {
		err := fmt.Errorf("stream client not connected")
		return s.handleStreamFailure(requestID, options, metrics, err, nil, onError)
	}

	start := time.Now()
	var firstTokenTime *time.Time

	request := &protos.ChatRequest{}
	if options.Request != nil {
		request.AdditionalData = options.Request.GetAdditionalData()
	}
	streamOptions := buildStreamCompletionOptions(&internal_callers.ChatCompletionOptions{
		AIOptions:       options.AIOptions,
		Request:         request,
		ToolDefinitions: options.ToolDefinitions,
	})
	streamOptions.Messages = buildHistory(allMessages)
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(streamOptions))
	}
	s.logger.Benchmark("Openai.chat_complete.Stream.llmRequestPrepare", time.Since(start))

	resp := client.Chat.Completions.NewStreaming(ctx, streamOptions)
	if resp.Err() != nil {
		s.logger.Errorf("Failed to get responses stream: %v", resp.Err())
		return s.handleStreamFailure(requestID, options, metrics, resp.Err(), utils.ToJson(resp), onError)
	}
	defer resp.Close()

	contentBuffer := make(map[int64]string)
	toolCallChoices := make(map[int64]struct{})
	accumulate := openai.ChatCompletionAccumulator{}

	for resp.Next() {
		chunk := resp.Current()
		accumulate.AddChunk(chunk)

		for _, choice := range chunk.Choices {
			if len(choice.Delta.ToolCalls) > 0 {
				toolCallChoices[choice.Index] = struct{}{}
			}
			content := choice.Delta.Content
			if content == "" {
				continue
			}
			contentBuffer[choice.Index] += content
			if _, hasToolCalls := toolCallChoices[choice.Index]; hasToolCalls {
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
					s.logger.Warnf("error streaming token: %v", err)
				}
			}
		}
	}

	if resp.Err() != nil {
		s.logger.Errorf("Failed while reading responses stream: %v", resp.Err())
		return s.handleStreamFailure(requestID, options, metrics, resp.Err(), utils.ToJson(resp), onError)
	}

	assistantMsg := buildAssistantMessageFromChoices(accumulate.Choices)
	if len(assistantMsg.Contents) == 0 {
		assistantMsg.Contents = finalizeStreamContentsByChoiceIndex(contentBuffer)
	}
	protoMsg := &protos.Message{Role: chatRoleAssistant, Message: &protos.Message_Assistant{Assistant: assistantMsg}}
	metrics.OnAddMetrics(completionUsageMetrics(accumulate.Usage)...)
	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}
	metrics.OnSuccess()
	if onMetrics != nil {
		onMetrics(requestID, protoMsg, metrics.Build())
	}
	result := utils.ToJson(accumulate)
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": result}, metrics.Build())
	}
	return nil
}
