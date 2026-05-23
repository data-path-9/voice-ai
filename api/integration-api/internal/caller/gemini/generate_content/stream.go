// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_gemini_generate_content

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	internal_gemini_common "github.com/rapidaai/api/integration-api/internal/caller/gemini/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type streamCaller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func NewStream(logger commons.Logger, credential *protos.Credential) (internal_callers.ChatStream, error) {
	if _, err := internal_gemini_common.ResolveAPIKey(credential); err != nil {
		logger.Errorf("Failed to create Gemini generate_content stream client: %v", err)
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

	client, err := internal_gemini_common.NewClient(s.credential)
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

	instruction, history, current := buildHistory(s.logger, allMessages)
	model, config := buildStreamContentConfig(options)
	config.SystemInstruction = instruction

	chat, err := client.Chats.Create(ctx, model, config, history)
	if err != nil {
		metrics.OnFailure()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"error": err}, metrics.Build())
		}
		if onError != nil {
			onError(requestID, err)
		}
		return err
	}

	if options.PreHook != nil {
		options.PreHook(messageJSON(model, config, history, current))
	}

	contents := make([]string, 0)
	toolCalls := make([]*protos.ToolCall, 0)
	contentBuilders := make([]strings.Builder, 0)
	hasToolCalls := false
	accumulator := &googleChatCompletionAccumulator{}

	for resp, streamErr := range chat.SendMessageStream(ctx, current) {
		if streamErr != nil {
			metrics.OnFailure()
			if options.PostHook != nil {
				options.PostHook(map[string]interface{}{
					"result": resp,
					"error":  streamErr,
				}, metrics.Build())
			}
			if onError != nil {
				onError(requestID, streamErr)
			}
			return streamErr
		}

		accumulator.AddChunk(resp)

		for _, choice := range resp.Candidates {
			if choice == nil || choice.Content == nil {
				continue
			}
			for _, part := range choice.Content.Parts {
				if part.FunctionCall != nil {
					hasToolCalls = true
					for len(toolCalls) <= int(choice.Index) {
						toolCalls = append(toolCalls, nil)
					}

					argsJSON, marshalErr := json.Marshal(part.FunctionCall.Args)
					if marshalErr != nil {
						s.logger.Warnf("failed to marshal gemini function args: %v", marshalErr)
						argsJSON = []byte("{}")
					}

					if part.FunctionCall.ID == "" {
						part.FunctionCall.ID = fmt.Sprintf("toolcall-%d-%s", choice.Index, uuid.NewString())
					}

					toolCalls[int(choice.Index)] = &protos.ToolCall{
						Id:   part.FunctionCall.ID,
						Type: "function",
						Function: &protos.FunctionCall{
							Name:      part.FunctionCall.Name,
							Arguments: string(argsJSON),
						},
					}
				}

				if part.Text != "" {
					for len(contentBuilders) <= int(choice.Index) {
						contentBuilders = append(contentBuilders, strings.Builder{})
					}
					contentBuilders[int(choice.Index)].WriteString(part.Text)

					if !hasToolCalls {
						if firstTokenTime == nil {
							now := time.Now()
							firstTokenTime = &now
						}
						tokenMsg := &protos.Message{
							Role: chatRoleAssistant,
							Message: &protos.Message_Assistant{
								Assistant: &protos.AssistantMessage{
									Contents: []string{part.Text},
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
		}
	}

	for _, builder := range contentBuilders {
		contents = append(contents, builder.String())
	}

	filteredToolCalls := make([]*protos.ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		if tc != nil {
			filteredToolCalls = append(filteredToolCalls, tc)
		}
	}

	metrics.OnAddMetrics(internal_gemini_common.UsageMetrics(accumulator.UsageMetadata)...)
	protoMsg := &protos.Message{
		Role: chatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: &protos.AssistantMessage{
				Contents:  contents,
				ToolCalls: filteredToolCalls,
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
			"result": accumulator,
		}, metrics.Build())
	}

	return nil
}
