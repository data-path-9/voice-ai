// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_websocket_streamer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	internal_azure_common "github.com/rapidaai/api/integration-api/internal/caller/azure/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

const (
	chatRoleAssistant = "assistant"
	chatRoleSystem    = "system"
	chatRoleTool      = "tool"
	chatRoleUser      = "user"
)

type streamer struct {
	logger     commons.Logger
	credential *protos.Credential

	mu        sync.Mutex
	client    *openai.Client
	connected bool
	closed    bool
}

func New(logger commons.Logger, credential *protos.Credential) (internal_callers.ChatStream, error) {
	if _, err := internal_azure_common.NewClient(logger, credential); err != nil {
		logger.Errorf("Failed to create Azure websocket stream client: %v", err)
		return nil, err
	}
	return &streamer{logger: logger, credential: credential}, nil
}

func (s *streamer) GetCredential() *protos.Credential {
	return s.credential
}

func (s *streamer) Connect(ctx context.Context, configuration *protos.StreamChatConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = configuration

	if s.connected && s.client != nil {
		return nil
	}
	client, err := internal_azure_common.NewClient(s.logger, s.credential)
	if err != nil {
		s.logger.Errorf("Failed to create Azure websocket stream client: %v", err)
		return err
	}
	s.client = client
	s.connected = true
	s.closed = false
	return nil
}

func (s *streamer) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = false
	s.closed = true
	s.client = nil
	return nil
}

func (s *streamer) Chat(
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

	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		err := fmt.Errorf("stream client not connected")
		onError(options.Request.GetRequestId(), err)
		return err
	}

	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time

	completionsOptions := s.buildResponseOptions(options)
	completionsOptions.Input = responses.ResponseNewParamsInputUnion{OfInputItemList: s.buildHistory(allMessages)}
	if options.PreHook != nil {
		options.PreHook(utils.ToJson(completionsOptions))
	}
	s.logger.Benchmark("Azure.websocket_streamer.Chat.llmRequestPrepare", time.Since(start))

	resp := client.Responses.NewStreaming(ctx, completionsOptions)
	if resp.Err() != nil {
		s.markDisconnected()
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
				tokenMsg := &protos.Message{Role: chatRoleAssistant, Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{e.Delta}}}}
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
			if s.hasFunctionCall(e.Response.Output) {
				hasToolCalls = true
			}
		}
	}

	if resp.Err() != nil {
		s.markDisconnected()
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
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{Id: id, Type: "function", Function: &protos.FunctionCall{Name: fnCall.Name, Arguments: fnCall.Arguments}})
		}
		metrics.OnAddMetrics(internal_azure_common.ResponseUsageMetrics(finalResponse.Usage)...)
	} else if contentBuffer.Len() > 0 {
		assistantMsg.Contents = append(assistantMsg.Contents, contentBuffer.String())
	}

	protoMsg := &protos.Message{Role: chatRoleAssistant, Message: &protos.Message_Assistant{Assistant: assistantMsg}}
	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{Name: type_enums.TIME_TO_FIRST_TOKEN.String(), Value: fmt.Sprintf("%d", firstTokenTime.Sub(start)), Description: "Time to receive first token from LLM"})
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

func (s *streamer) markDisconnected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = false
}

func (s *streamer) hasFunctionCall(items []responses.ResponseOutputItemUnion) bool {
	for _, item := range items {
		if item.Type == "function_call" {
			return true
		}
	}
	return false
}

func (s *streamer) buildResponseOptions(opts *internal_callers.ChatStreamCompletionOptions) responses.ResponseNewParams {
	options := responses.ResponseNewParams{Store: openai.Bool(false)}
	additionalData := map[string]string{}
	if opts.Request != nil {
		additionalData = opts.Request.GetAdditionalData()
	}
	promptCacheKeySelector := "assistant_id"
	if len(opts.ToolDefinitions) > 0 {
		fns := make([]responses.ToolUnionParam, 0, len(opts.ToolDefinitions))
		for _, tl := range opts.ToolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}
			fn := tl.Function
			funcDef := responses.FunctionToolParam{Name: fn.Name, Strict: openai.Bool(false)}
			if fn.Description != "" {
				funcDef.Description = openai.String(fn.Description)
			}
			if fn.Parameters != nil {
				funcDef.Parameters = fn.Parameters.ToMap()
			} else {
				funcDef.Parameters = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}
			fns = append(fns, responses.ToolUnionParam{OfFunction: &funcDef})
		}
		options.Tools = fns
	}

	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options.Model = shared.ResponsesModel(modelName)
			}
		case "model.user":
			if user, err := utils.AnyToString(value); err == nil {
				options.User = openai.String(user)
			}
		case "model.reasoning_effort":
			if re, err := utils.AnyToString(value); err == nil {
				options.Reasoning = shared.ReasoningParam{Effort: shared.ReasoningEffort(re)}
			}
		case "model.service_tier":
			if st, err := utils.AnyToString(value); err == nil {
				options.ServiceTier = responses.ResponseNewParamsServiceTier(st)
			}
		case "model.prompt_cache_key":
			if promptCacheKey, err := utils.AnyToString(value); err == nil && promptCacheKey != "" {
				promptCacheKeySelector = promptCacheKey
			}
		case "model.prompt_cache_retention":
			if retention, err := utils.AnyToString(value); err == nil && retention != "" {
				options.PromptCacheRetention = responses.ResponseNewParamsPromptCacheRetention(retention)
			}
		case "model.top_logprobs":
			if tl, err := utils.AnyToInt64(value); err == nil {
				options.TopLogprobs = openai.Int(tl)
			}
		case "model.metadata":
			format, _ := utils.AnyToString(value)
			var mtd map[string]string
			if err := json.Unmarshal([]byte(format), &mtd); err == nil {
				options.Metadata = shared.Metadata(mtd)
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat64(value); err == nil {
				options.Temperature = openai.Float(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat64(value); err == nil {
				options.TopP = openai.Float(topP)
			}
		case "model.max_completion_tokens", "model.max_output_tokens":
			if maxTokens, err := utils.AnyToInt64(value); err == nil {
				options.MaxOutputTokens = openai.Int(maxTokens)
			}
		case "model.store":
			if store, err := utils.AnyToBool(value); err == nil {
				options.Store = openai.Bool(store)
			}
		case "model.tool_choice":
			if choice, err := utils.AnyToString(value); err == nil {
				switch choice {
				case "auto":
					options.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto)}
				case "required":
					options.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsRequired)}
				default:
					options.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone)}
				}
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				switch format["type"].(string) {
				case "json_object":
					options.Text.Format = responses.ResponseFormatTextConfigUnionParam{OfJSONObject: &shared.ResponseFormatJSONObjectParam{}}
				case "text":
					options.Text.Format = responses.ResponseFormatTextConfigUnionParam{OfText: &shared.ResponseFormatTextParam{}}
				case "json_schema":
					if schemaData, ok := format["json_schema"].(map[string]interface{}); ok {
						cfg := responses.ResponseFormatTextJSONSchemaConfigParam{Name: "response"}
						if name, ok := schemaData["name"].(string); ok && strings.TrimSpace(name) != "" {
							cfg.Name = name
						}
						if description, ok := schemaData["description"].(string); ok && description != "" {
							cfg.Description = openai.String(description)
						}
						if strict, ok := schemaData["strict"].(bool); ok {
							cfg.Strict = openai.Bool(strict)
						}
						if schema, ok := schemaData["schema"].(map[string]interface{}); ok {
							cfg.Schema = schema
						} else {
							cfg.Schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
						}
						options.Text.Format = responses.ResponseFormatTextConfigUnionParam{OfJSONSchema: &cfg}
					}
				}
			}
		}
	}

	switch promptCacheKeySelector {
	case "user_identifier":
		options.PromptCacheKey = openai.String(additionalData["user_identifier"] + additionalData["assistant_provider_model_id"] + "__" + additionalData["assistant_id"])
	case "conversation_id":
		options.PromptCacheKey = openai.String(additionalData["conversation_id"] + additionalData["assistant_provider_model_id"] + "__" + additionalData["assistant_id"])
	case "assistant_id":
		options.PromptCacheKey = openai.String(additionalData["assistant_provider_model_id"] + "__" + additionalData["assistant_id"])
	default:
		options.PromptCacheKey = openai.String(additionalData["assistant_provider_model_id"] + "__" + additionalData["assistant_id"])
	}
	return options
}

func (s *streamer) buildHistory(allMessages []*protos.Message) []responses.ResponseInputItemUnionParam {
	msg := make([]responses.ResponseInputItemUnionParam, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case chatRoleUser:
			if user := cntn.GetUser(); user != nil {
				msg = append(msg, responses.ResponseInputItemParamOfMessage(user.GetContent(), responses.EasyInputMessageRoleUser))
			}
		case chatRoleAssistant:
			if assistant := cntn.GetAssistant(); assistant != nil {
				txtContent := strings.Join(assistant.GetContents(), "")
				if txtContent != "" {
					msg = append(msg, responses.ResponseInputItemParamOfMessage(txtContent, responses.EasyInputMessageRoleAssistant))
				}
				for _, ttc := range assistant.GetToolCalls() {
					if ttc.GetFunction() == nil || ttc.GetId() == "" {
						continue
					}
					msg = append(msg, responses.ResponseInputItemParamOfFunctionCall(ttc.GetFunction().GetArguments(), ttc.GetId(), ttc.GetFunction().GetName()))
				}
			}
		case chatRoleSystem:
			if system := cntn.GetSystem(); system != nil && len(system.GetContent()) > 0 {
				msg = append(msg, responses.ResponseInputItemParamOfMessage(system.GetContent(), responses.EasyInputMessageRoleSystem))
			}
		case chatRoleTool:
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					if t.GetId() == "" {
						continue
					}
					msg = append(msg, responses.ResponseInputItemParamOfFunctionCallOutput(t.GetId(), t.GetContent()))
				}
			}
		}
	}
	return msg
}
