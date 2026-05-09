// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_gemini_generate_content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	internal_gemini_common "github.com/rapidaai/api/integration-api/internal/caller/gemini/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/genai"
)

type chatCaller struct {
	logger commons.Logger
	client *genai.Client
}

func NewChat(logger commons.Logger, credential *protos.Credential) (internal_callers.Chat, error) {
	client, err := internal_gemini_common.NewClient(credential)
	if err != nil {
		logger.Errorf("Failed to create Gemini generate_content chat client: %v", err)
		return nil, err
	}
	return &chatCaller{
		logger: logger,
		client: client,
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

	instruction, histories, current := buildHistory(cc.logger, allMessages)
	model, config := buildContentConfig(options)
	config.SystemInstruction = instruction

	chat, err := cc.client.Chats.Create(ctx, model, config, histories)
	if err != nil {
		cc.logger.Errorf("error creating gemini chat: %v", err)
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
		cc.logger.Errorf("chat completion failed to get response from gemini: %v", err)
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
					cc.logger.Warnf("failed to marshal gemini function args: %v", marshalErr)
					argsJSON = []byte("{}")
				}
				callID := part.FunctionCall.ID
				if callID == "" {
					callID = fmt.Sprintf("toolcall-%d-%s", choice.Index, uuid.NewString())
				}
				assistant.ToolCalls = append(assistant.ToolCalls, &protos.ToolCall{
					Id:   callID,
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
		metrics.OnAddMetrics(internal_gemini_common.UsageMetrics(resp.UsageMetadata)...)
	}

	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": resp}, metrics.Build())
	}

	return &protos.Message{
		Role:    chatRoleAssistant,
		Message: &protos.Message_Assistant{Assistant: assistant},
	}, metrics.Build(), nil
}
