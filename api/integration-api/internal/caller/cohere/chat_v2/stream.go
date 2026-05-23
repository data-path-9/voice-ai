// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_chat_v2

import (
	"context"
	"fmt"
	"io"
	"time"

	internal_cohere_common "github.com/rapidaai/api/integration-api/internal/caller/cohere/common"
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
}

func NewStream(logger commons.Logger, credential *protos.Credential) (internal_callers.ChatStream, error) {
	if _, err := internal_cohere_common.ResolveAPIKey(credential); err != nil {
		logger.Errorf("Failed to create Cohere chat_v2 stream client: %v", err)
		return nil, err
	}
	return &streamCaller{
		logger:     logger,
		credential: credential,
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
	if options != nil && options.Request != nil {
		requestID = options.Request.GetRequestId()
	}

	if err := s.Connect(ctx, nil); err != nil {
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}

	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time

	client, err := internal_cohere_common.NewClient(s.credential)
	if err != nil {
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"error": err,
			}, metrics.OnFailure().Build())
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}

	chatRequest := buildStreamRequest(options)
	chatRequest.Messages = buildHistory(s.logger, allMessages)

	if options.PreHook != nil {
		options.PreHook(utils.ToJson(chatRequest))
	}
	s.logger.Benchmark("Cohere.chat_v2.Stream.llmRequestPrepare", time.Since(start))

	resp, err := client.V2.ChatStream(ctx, chatRequest)
	if err != nil {
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"result": utils.ToJson(resp),
				"error":  err,
			}, metrics.Build())
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}
	defer resp.Close()

	contents := make([]string, 0)
	toolCalls := make([]*protos.ToolCall, 0)
	var currentToolCall *protos.ToolCall
	var currentContent string
	hasToolCalls := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			rep, recvErr := resp.Recv()
			if recvErr != nil {
				if recvErr == io.EOF {
					return nil
				}
				if onError != nil {
					onError(requestID, recvErr)
				}
				return recvErr
			}

			switch {
			case rep.MessageStart != nil:
				continue
			case rep.ContentStart != nil:
				if rep.ContentStart.Delta != nil && rep.ContentStart.Delta.Message != nil && rep.ContentStart.Delta.Message.Content != nil {
					if text := rep.ContentStart.Delta.Message.Content.GetText(); text != nil {
						currentContent = *text
					}
				}
			case rep.ContentDelta != nil:
				if rep.ContentDelta.Delta != nil && rep.ContentDelta.Delta.Message != nil && rep.ContentDelta.Delta.Message.Content != nil {
					if text := rep.ContentDelta.Delta.Message.Content.GetText(); text != nil {
						currentContent += *text
						if !hasToolCalls {
							if firstTokenTime == nil {
								now := time.Now()
								firstTokenTime = &now
							}
							tokenMsg := &protos.Message{
								Role: "assistant",
								Message: &protos.Message_Assistant{
									Assistant: &protos.AssistantMessage{
										Contents: []string{*text},
									},
								},
							}
							if onStream != nil {
								if err := onStream(requestID, tokenMsg); err != nil {
									s.logger.Warnf("error streaming token: %v", err)
								}
							}
						}
					}
				}
			case rep.ContentEnd != nil:
				if currentContent != "" {
					contents = append(contents, currentContent)
					currentContent = ""
				}
			case rep.ToolCallStart != nil:
				hasToolCalls = true
				if rep.ToolCallStart.Delta != nil && rep.ToolCallStart.Delta.Message != nil && rep.ToolCallStart.Delta.Message.ToolCalls != nil {
					tc := rep.ToolCallStart.Delta.Message.ToolCalls
					var name, args string
					if tc.Function.Name != nil {
						name = *tc.Function.Name
					}
					if tc.Function.Arguments != nil {
						args = *tc.Function.Arguments
					}
					currentToolCall = &protos.ToolCall{
						Id:   tc.Id,
						Type: tc.Type(),
						Function: &protos.FunctionCall{
							Name:      name,
							Arguments: args,
						},
					}
				}
			case rep.ToolCallDelta != nil:
				if currentToolCall != nil && rep.ToolCallDelta.Delta != nil && rep.ToolCallDelta.Delta.Message != nil && rep.ToolCallDelta.Delta.Message.ToolCalls != nil {
					if rep.ToolCallDelta.Delta.Message.ToolCalls.Function.Arguments != nil {
						currentToolCall.Function.Arguments += *rep.ToolCallDelta.Delta.Message.ToolCalls.Function.Arguments
					}
				}
			case rep.ToolCallEnd != nil:
				if currentToolCall != nil {
					toolCalls = append(toolCalls, currentToolCall)
					currentToolCall = nil
				}
			case rep.MessageEnd != nil:
				metrics.OnAddMetrics(internal_cohere_common.UsageMetrics(rep.MessageEnd.Delta.Usage)...)
				protoMsg := &protos.Message{
					Role: "assistant",
					Message: &protos.Message_Assistant{
						Assistant: &protos.AssistantMessage{
							Contents:  contents,
							ToolCalls: toolCalls,
						},
					},
				}
				if firstTokenTime != nil {
					metrics.OnAddMetrics(&protos.Metric{
						Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
						Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
						Description: "Time to receive first token from LLM",
					})
				}
				metrics.OnSuccess()
				if onMetrics != nil {
					_ = onMetrics(requestID, protoMsg, metrics.Build())
				}
				if options.PostHook != nil {
					options.PostHook(map[string]interface{}{
						"result": protoMsg,
					}, metrics.Build())
				}
				return nil
			}
		}
	}
}
