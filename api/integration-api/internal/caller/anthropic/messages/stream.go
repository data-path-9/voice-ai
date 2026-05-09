// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_anthropic_messages

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"

	internal_anthropic_common "github.com/rapidaai/api/integration-api/internal/caller/anthropic/common"
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
	if _, err := internal_anthropic_common.ResolveAPIKey(credential); err != nil {
		logger.Errorf("Failed to create Anthropic messages stream client: %v", err)
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

	client, err := internal_anthropic_common.NewClient(s.credential)
	if err != nil {
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}

	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time

	instruction, messages := buildHistory(s.logger, allMessages)
	params := buildStreamMessageNewParams(options)
	params.Messages = messages
	params.System = instruction

	if options.PreHook != nil {
		options.PreHook(utils.ToJson(params))
	}

	stream := client.Messages.NewStreaming(ctx, params)
	message := anthropic.Message{}
	if stream.Err() != nil {
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"result": utils.ToJson(message),
				"error":  stream.Err(),
			}, metrics.Build())
		}
		if onError != nil {
			onError(requestID, stream.Err())
		}
		return stream.Err()
	}

	completeMessage := &protos.AssistantMessage{}
	var currentToolCall *protos.ToolCall
	var currentContent string
	isToolCall := false
	hasToolCalls := false

	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			if onError != nil {
				onError(requestID, err)
			}
			continue
		}

		switch event := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			switch event.ContentBlock.Type {
			case "tool_use":
				isToolCall = true
				hasToolCalls = true
				currentToolCall = &protos.ToolCall{
					Id:   event.ContentBlock.ID,
					Type: "function",
					Function: &protos.FunctionCall{
						Name: event.ContentBlock.Name,
					},
				}
			case "text":
				currentContent = ""
			}

		case anthropic.ContentBlockDeltaEvent:
			switch event.Delta.Type {
			case "text_delta":
				content := event.Delta.Text
				if content != "" && !isToolCall {
					currentContent += content
					if !hasToolCalls {
						if firstTokenTime == nil {
							now := time.Now()
							firstTokenTime = &now
						}
						tokenMsg := &protos.Message{
							Role: "assistant",
							Message: &protos.Message_Assistant{
								Assistant: &protos.AssistantMessage{
									Contents: []string{content},
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
			case "input_json_delta":
				if currentToolCall != nil {
					currentToolCall.Function.Arguments += event.Delta.PartialJSON
				}
			}

		case anthropic.ContentBlockStopEvent:
			if currentToolCall != nil {
				completeMessage.ToolCalls = append(completeMessage.ToolCalls, currentToolCall)
				currentToolCall = nil
			}
			if currentContent != "" {
				completeMessage.Contents = append(completeMessage.Contents, currentContent)
				currentContent = ""
			}
			isToolCall = false

		case anthropic.MessageStopEvent:
			metrics.OnAddMetrics(internal_anthropic_common.UsageMetrics(message.Usage)...)
			finalMsg := &protos.Message{
				Role: "assistant",
				Message: &protos.Message_Assistant{
					Assistant: completeMessage,
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
				_ = onMetrics(requestID, finalMsg, metrics.Build())
			}

			if options.PostHook != nil {
				options.PostHook(map[string]interface{}{
					"result": utils.ToJson(message),
				}, metrics.Build())
			}
			return nil
		}
	}

	if stream.Err() != nil {
		if onError != nil {
			onError(requestID, stream.Err())
		}
		return stream.Err()
	}
	return nil
}
