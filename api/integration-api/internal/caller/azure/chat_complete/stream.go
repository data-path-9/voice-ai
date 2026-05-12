// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"

	internal_azure_common "github.com/rapidaai/api/integration-api/internal/caller/azure/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
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

func NewStream(logger commons.Logger, credential *protos.Credential) (internal_callers.ChatStream, error) {
	if _, err := internal_azure_common.NewClient(logger, credential); err != nil {
		logger.Errorf("Failed to create Azure chat_complete stream client: %v", err)
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
	endpoint, subscriptionKey, err := internal_azure_common.ResolveCredential(s.logger, s.credential)
	if err != nil {
		s.logger.Errorf("Failed to create Azure chat_complete stream client: %v", err)
		return err
	}

	transport := &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxConnsPerHost:     internal_azure_common.StreamMaxConnsPerHost,
		MaxIdleConnsPerHost: internal_azure_common.StreamMaxIdleConnsPerHost,
		MaxIdleConns:        internal_azure_common.StreamMaxIdleConns,
		IdleConnTimeout:     internal_azure_common.StreamIdleConnTimeout,
	}
	s.httpClient = &http.Client{Transport: transport}

	client := openai.NewClient(
		option.WithBaseURL(endpoint),
		option.WithAPIKey(subscriptionKey),
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
	if err := s.Connect(ctx, nil); err != nil {
		onError(options.Request.GetRequestId(), err)
		return err
	}

	client := s.client
	if client == nil {
		err := fmt.Errorf("stream client not connected")
		onError(options.Request.GetRequestId(), err)
		return err
	}

	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time

	request := &protos.ChatRequest{}
	if options.Request != nil {
		request.AdditionalData = options.Request.GetAdditionalData()
	}
	streamOptions := buildStreamResponseOptions(&internal_callers.ChatCompletionOptions{
		AIOptions:       options.AIOptions,
		Request:         request,
		ToolDefinitions: options.ToolDefinitions,
	})
	streamOptions.Input = responses.ResponseNewParamsInputUnion{OfInputItemList: buildHistory(allMessages)}
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(streamOptions))
	}
	s.logger.Benchmark("Azure.chat_complete.Stream.llmRequestPrepare", time.Since(start))

	resp := client.Responses.NewStreaming(ctx, streamOptions)
	if resp.Err() != nil {
		s.logger.Errorf("Failed to get responses stream: %v", resp.Err())
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"result": utils.ToJson(resp), "error": resp.Err()}, metrics.Build())
		}
		onError(options.Request.GetRequestId(), resp.Err())
		return resp.Err()
	}
	defer resp.Close()

	assistantMsg := &protos.AssistantMessage{Contents: make([]string, 0), ToolCalls: make([]*protos.ToolCall, 0)}
	var contentBuffer strings.Builder
	hasToolCalls := false
	var finalResponse *responses.Response

	for resp.Next() {
		event := resp.Current()
		switch e := event.AsAny().(type) {
		case responses.ResponseTextDeltaEvent:
			if e.Delta == "" {
				continue
			}
			contentBuffer.WriteString(e.Delta)
			if !hasToolCalls {
				if firstTokenTime == nil {
					now := time.Now()
					firstTokenTime = &now
				}
				tokenMsg := &protos.Message{
					Role: chatRoleAssistant,
					Message: &protos.Message_Assistant{
						Assistant: &protos.AssistantMessage{Contents: []string{e.Delta}},
					},
				}
				if err := onStream(options.Request.GetRequestId(), tokenMsg); err != nil {
					s.logger.Warnf("error streaming token: %v", err)
				}
			}
		case responses.ResponseFunctionCallArgumentsDeltaEvent, responses.ResponseFunctionCallArgumentsDoneEvent:
			hasToolCalls = true
		case responses.ResponseOutputItemAddedEvent:
			if e.Item.Type == "function_call" {
				hasToolCalls = true
			}
		case responses.ResponseOutputItemDoneEvent:
			if e.Item.Type == "function_call" {
				hasToolCalls = true
			}
		case responses.ResponseCompletedEvent:
			finalResponse = &e.Response
			if hasFunctionCall(e.Response.Output) {
				hasToolCalls = true
			}
		}
	}

	if resp.Err() != nil {
		s.logger.Errorf("Failed while reading responses stream: %v", resp.Err())
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"result": utils.ToJson(resp), "error": resp.Err()}, metrics.OnFailure().Build())
		}
		onError(options.Request.GetRequestId(), resp.Err())
		return resp.Err()
	}

	if finalResponse != nil {
		if outputText := finalResponse.OutputText(); outputText != "" {
			assistantMsg.Contents = append(assistantMsg.Contents, outputText)
		}
		for _, item := range finalResponse.Output {
			if item.Type != "function_call" {
				continue
			}
			fnCall := item.AsFunctionCall()
			id := fnCall.CallID
			if id == "" {
				id = fnCall.ID
			}
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{
				Id:   id,
				Type: "function",
				Function: &protos.FunctionCall{
					Name:      fnCall.Name,
					Arguments: fnCall.Arguments,
				},
			})
		}
		metrics.OnAddMetrics(internal_azure_common.ResponseUsageMetrics(finalResponse.Usage)...)
	} else if contentBuffer.Len() > 0 {
		assistantMsg.Contents = append(assistantMsg.Contents, contentBuffer.String())
	}

	protoMsg := &protos.Message{Role: chatRoleAssistant, Message: &protos.Message_Assistant{Assistant: assistantMsg}}
	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}
	metrics.OnSuccess()
	onMetrics(options.Request.GetRequestId(), protoMsg, metrics.Build())
	result := utils.ToJson(resp)
	if finalResponse != nil {
		result = utils.ToJson(finalResponse)
	}
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": result}, metrics.Build())
	}
	return nil
}
