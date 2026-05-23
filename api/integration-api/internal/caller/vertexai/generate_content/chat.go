// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_generate_content

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_vertexai_common "github.com/rapidaai/api/integration-api/internal/caller/vertexai/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type chatCaller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func NewChat(logger commons.Logger, credential *protos.Credential) (internal_callers.Chat, error) {
	if _, _, _, err := internal_vertexai_common.ResolveCredential(credential); err != nil {
		logger.Errorf("Failed to create VertexAI generate_content chat client: %v", err)
		return nil, err
	}
	return &chatCaller{
		logger:     logger,
		credential: credential,
	}, nil
}

func (cc *chatCaller) ChatComplete(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	if len(allMessages) == 0 {
		err := errors.New("no messages in the input")
		metrics.OnFailure()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"error": err}, metrics.Build())
		}
		return nil, metrics.Build(), err
	}

	client, err := internal_vertexai_common.NewClient(cc.credential)
	if err != nil {
		metrics.OnFailure()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"error": err}, metrics.Build())
		}
		return nil, metrics.Build(), err
	}

	instruction, histories, current := buildHistory(cc.logger, allMessages)
	model, config := buildContentConfig(options)
	config.SystemInstruction = instruction

	chat, err := client.Chats.Create(ctx, model, config, histories)
	if err != nil {
		cc.logger.Errorf("error creating vertexai chat: %v", err)
		metrics.OnFailure()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"error": err}, metrics.Build())
		}
		return nil, metrics.Build(), err
	}

	if options.PreHook != nil {
		options.PreHook(messageJSON(model, config, histories, current))
	}

	resp, err := chat.SendMessage(ctx, current)
	if err != nil {
		cc.logger.Errorf("chat completion failed to get response from vertexai: %v", err)
		metrics.OnFailure()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"result": resp, "error": err}, metrics.Build())
		}
		return nil, metrics.Build(), err
	}

	assistant := &protos.AssistantMessage{
		Contents:  make([]string, len(resp.Candidates)),
		ToolCalls: make([]*protos.ToolCall, 0),
	}
	for _, choice := range resp.Candidates {
		if choice == nil || choice.Content == nil {
			continue
		}

		idx := int(choice.Index)
		if idx >= len(assistant.Contents) {
			diff := idx - len(assistant.Contents) + 1
			assistant.Contents = append(assistant.Contents, make([]string, diff)...)
		}

		var contentBuilder strings.Builder
		for _, part := range choice.Content.Parts {
			if part.Text != "" {
				contentBuilder.WriteString(part.Text)
			}
			if part.FunctionCall != nil {
				argsJSON, marshalErr := json.Marshal(part.FunctionCall.Args)
				if marshalErr != nil {
					cc.logger.Warnf("failed to marshal vertexai function args: %v", marshalErr)
					argsJSON = []byte("{}")
				}
				assistant.ToolCalls = append(assistant.ToolCalls, &protos.ToolCall{
					Id:   part.FunctionCall.ID,
					Type: "function",
					Function: &protos.FunctionCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		assistant.Contents[idx] = contentBuilder.String()
	}

	metrics.OnSuccess()
	if resp.UsageMetadata != nil {
		metrics.OnAddMetrics(internal_vertexai_common.UsageMetrics(resp.UsageMetadata)...)
	}

	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp}, metrics.Build())
	}

	return &protos.Message{
		Role:    chatRoleAssistant,
		Message: &protos.Message_Assistant{Assistant: assistant},
	}, metrics.Build(), nil
}
