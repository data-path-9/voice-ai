// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_custom_llm_openai_responses

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

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
		logger.Errorf("custom-llm openai_responses: failed to create stream client: %v", err)
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
		s.logger.Errorf("custom-llm openai_responses: failed to create stream client: %v", err)
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
	var firstTokenAt *time.Time

	request := buildResponseParams(&internal_callers.ChatCompletionOptions{
		AIOptions:       options.AIOptions,
		ToolDefinitions: options.ToolDefinitions,
	})
	request.Input = responses.ResponseNewParamsInputUnion{OfInputItemList: buildHistory(allMessages)}
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(request))
	}

	stream := s.client.Responses.NewStreaming(ctx, request)
	if stream.Err() != nil {
		err := stream.Err()
		s.logger.Errorf("custom-llm openai_responses: stream init failed: %v", err)
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": utils.ToJson(stream),
			}, failure)
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}
	defer stream.Close()
	assistantMessage := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	var contentBuilder strings.Builder
	hasToolCalls := false
	var finalResponse *responses.Response

	for stream.Next() {
		event := stream.Current()
		switch typed := event.AsAny().(type) {
		case responses.ResponseTextDeltaEvent:
			if typed.Delta == "" {
				continue
			}
			contentBuilder.WriteString(typed.Delta)
			if hasToolCalls {
				continue
			}
			if firstTokenAt == nil {
				now := time.Now()
				firstTokenAt = &now
			}
			if onStream != nil {
				tokenMessage := &protos.Message{
					Role: internal_custom_llm_common.ChatRoleAssistant,
					Message: &protos.Message_Assistant{
						Assistant: &protos.AssistantMessage{Contents: []string{typed.Delta}},
					},
				}
				if err := onStream(requestID, tokenMessage); err != nil {
					s.logger.Warnf("custom-llm openai_responses: onStream error: %v", err)
				}
			}
		case responses.ResponseFunctionCallArgumentsDeltaEvent:
			hasToolCalls = true
		case responses.ResponseFunctionCallArgumentsDoneEvent:
			hasToolCalls = true
		case responses.ResponseOutputItemAddedEvent:
			if typed.Item.Type == "function_call" {
				hasToolCalls = true
			}
		case responses.ResponseOutputItemDoneEvent:
			if typed.Item.Type == "function_call" {
				hasToolCalls = true
			}
		case responses.ResponseCompletedEvent:
			finalResponse = &typed.Response
			if containsFunctionCall(typed.Response.Output) {
				hasToolCalls = true
			}
		}
	}

	if err := stream.Err(); err != nil {
		failure := metrics.OnFailure().Build()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error":  err,
				"result": utils.ToJson(stream),
			}, failure)
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}

	if finalResponse != nil {
		if outputText := finalResponse.OutputText(); outputText != "" {
			assistantMessage.Contents = append(assistantMessage.Contents, outputText)
		}
		for _, outputItem := range finalResponse.Output {
			if outputItem.Type != "function_call" {
				continue
			}
			functionCall := outputItem.AsFunctionCall()
			callID := functionCall.CallID
			if callID == "" {
				callID = functionCall.ID
			}
			assistantMessage.ToolCalls = append(assistantMessage.ToolCalls, &protos.ToolCall{
				Id:   callID,
				Type: "function",
				Function: &protos.FunctionCall{
					Name:      functionCall.Name,
					Arguments: functionCall.Arguments,
				},
			})
		}
		metrics.OnAddMetrics(buildResponseUsageMetrics(finalResponse.Usage)...)
	} else if contentBuilder.Len() > 0 {
		assistantMessage.Contents = append(assistantMessage.Contents, contentBuilder.String())
	}

	protoMessage := &protos.Message{
		Role: internal_custom_llm_common.ChatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMessage,
		},
	}

	if firstTokenAt != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%d", firstTokenAt.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}
	success := metrics.OnSuccess().Build()
	if onMetrics != nil {
		onMetrics(requestID, protoMessage, success)
	}
	resultPayload := utils.ToJson(stream)
	if finalResponse != nil {
		resultPayload = utils.ToJson(finalResponse)
	}
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resultPayload}, success)
	}
	return nil
}
