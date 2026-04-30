package internal_custom_llm_callers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type largeLanguageCaller struct {
	CustomLLM
}

func NewLargeLanguageCaller(logger commons.Logger, credential *protos.Credential) internal_callers.LargeLanguageCaller {
	return &largeLanguageCaller{
		CustomLLM: customLLM(logger, credential),
	}
}

func (llc *largeLanguageCaller) GetChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
) (*protos.Message, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("chat completion unable to get client for custom-llm %v", err)
		return nil, metrics.OnFailure().Build(), err
	}

	llmRequest := llc.getChatCompleteParameter(options, false)
	llmRequest.Messages = llc.buildHistory(allMessages)

	options.PreHook(utils.ToJson(llmRequest))

	resp, err := client.Chat.Completions.New(ctx, llmRequest)
	if err != nil {
		llc.logger.Errorf("chat completion failed to get response from custom-llm %v", err)
		failure := metrics.OnFailure().Build()
		payload := map[string]interface{}{"error": err}
		if resp != nil {
			payload["result"] = resp
		}
		options.PostHook(payload, failure)
		return nil, failure, err
	}

	assistantMsg := &protos.AssistantMessage{
		Contents:  make([]string, 0),
		ToolCalls: make([]*protos.ToolCall, 0),
	}

	for _, choice := range resp.Choices {
		if choice.Message.Content != "" {
			assistantMsg.Contents = append(assistantMsg.Contents, choice.Message.Content)
		}
		for _, tool := range choice.Message.ToolCalls {
			if tool.Type == "function" {
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, &protos.ToolCall{
					Id:   tool.ID,
					Type: string(tool.Type),
					Function: &protos.FunctionCall{
						Name:      tool.Function.Name,
						Arguments: tool.Function.Arguments,
					},
				})
			}
		}
	}

	message := &protos.Message{
		Role: ChatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	metrics.OnAddMetrics(llc.GetCompletionUsages(resp.Usage)...)

	options.PostHook(map[string]interface{}{
		"result": resp,
	}, metrics.OnSuccess().Build())
	return message, metrics.Build(), nil
}

func (llc *largeLanguageCaller) StreamChatCompletion(
	ctx context.Context,
	allMessages []*protos.Message,
	options *internal_callers.ChatCompletionOptions,
	onStream func(string, *protos.Message) error,
	onMetrics func(string, *protos.Message, []*protos.Metric) error,
	onError func(string, error),
) error {
	start := time.Now()
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()
	var firstTokenTime *time.Time

	client, err := llc.GetClient()
	if err != nil {
		llc.logger.Errorf("chat completion unable to get client for custom-llm: %v", err)
		onError(options.Request.GetRequestId(), err)
		options.PostHook(map[string]interface{}{
			"error": err,
		}, metrics.OnFailure().Build())
		return err
	}

	completionsOptions := llc.getChatCompleteParameter(options, true)
	completionsOptions.Messages = llc.buildHistory(allMessages)
	options.PreHook(utils.ToJson(completionsOptions))
	llc.logger.Benchmark("custom_llm.llm.GetChatCompletion.llmRequestPrepare", time.Since(start))

	resp := client.Chat.Completions.NewStreaming(ctx, completionsOptions)
	if resp.Err() != nil {
		llc.logger.Errorf("Failed to get chat completions stream: %v", resp.Err())
		options.PostHook(map[string]interface{}{
			"result": utils.ToJson(resp),
			"error":  resp.Err(),
		}, metrics.OnFailure().Build())
		onError(options.Request.GetRequestId(), resp.Err())
		return resp.Err()
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
			if content != "" {
				if len(contentBuffer) <= i {
					contentBuffer = append(contentBuffer, content)
				} else {
					contentBuffer[i] += content
				}

				if !hasToolCalls {
					if firstTokenTime == nil {
						now := time.Now()
						firstTokenTime = &now
					}
					tokenMsg := &protos.Message{
						Role: ChatRoleAssistant,
						Message: &protos.Message_Assistant{
							Assistant: &protos.AssistantMessage{
								Contents: []string{content},
							},
						},
					}
					if err := onStream(options.Request.GetRequestId(), tokenMsg); err != nil {
						llc.logger.Warnf("error streaming token: %v", err)
					}
				}
			}
		}
	}

	if resp.Err() != nil {
		llc.logger.Errorf("Failed while reading chat completions stream: %v", resp.Err())
		options.PostHook(map[string]interface{}{
			"result": utils.ToJson(resp),
			"error":  resp.Err(),
		}, metrics.OnFailure().Build())
		onError(options.Request.GetRequestId(), resp.Err())
		return resp.Err()
	}

	assistantMsg.Contents = contentBuffer
	protoMsg := &protos.Message{
		Role: ChatRoleAssistant,
		Message: &protos.Message_Assistant{
			Assistant: assistantMsg,
		},
	}

	metrics.OnAddMetrics(llc.GetCompletionUsages(accumulate.Usage)...)

	if firstTokenTime != nil {
		metrics.OnAddMetrics(&protos.Metric{
			Name:        type_enums.TIME_TO_FIRST_TOKEN.String(),
			Value:       fmt.Sprintf("%d", firstTokenTime.Sub(start)),
			Description: "Time to receive first token from LLM",
		})
	}
	metrics.OnSuccess()
	onMetrics(options.Request.GetRequestId(), protoMsg, metrics.Build())
	options.PostHook(map[string]interface{}{
		"result": utils.ToJson(accumulate),
	}, metrics.Build())

	return nil
}

func (llc *largeLanguageCaller) buildHistory(allMessages []*protos.Message) []openai.ChatCompletionMessageParamUnion {
	msg := make([]openai.ChatCompletionMessageParamUnion, 0)
	for _, cntn := range allMessages {
		switch cntn.GetRole() {
		case ChatRoleUser:
			if user := cntn.GetUser(); user != nil {
				msg = append(msg, openai.UserMessage(user.GetContent()))
			}
		case ChatRoleAssistant:
			if assistant := cntn.GetAssistant(); assistant != nil {
				txtContent := strings.Join(assistant.GetContents(), "")
				toolCalls := assistant.GetToolCalls()
				assistantMessage := openai.ChatCompletionAssistantMessageParam{}
				if len(txtContent) > 0 || len(toolCalls) > 0 {
					if len(txtContent) > 0 {
						assistantMessage.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
							OfString: openai.String(txtContent),
						}
					}
					if len(toolCalls) > 0 {
						fctCall := make([]openai.ChatCompletionMessageToolCallParam, 0)
						for _, ttc := range toolCalls {
							fctCall = append(fctCall, openai.ChatCompletionMessageToolCallParam{
								ID: ttc.GetId(),
								Function: openai.ChatCompletionMessageToolCallFunctionParam{
									Name:      ttc.GetFunction().GetName(),
									Arguments: ttc.GetFunction().GetArguments(),
								},
							})
						}
						assistantMessage.ToolCalls = fctCall
					}
					msg = append(msg, openai.ChatCompletionMessageParamUnion{
						OfAssistant: &assistantMessage,
					})
				}
			}

		case ChatRoleSystem:
			if system := cntn.GetSystem(); system != nil {
				txtContent := system.GetContent()
				if len(txtContent) > 0 {
					msg = append(msg, openai.SystemMessage(txtContent))
				}
			}

		case ChatRoleTool:
			if tool := cntn.GetTool(); tool != nil {
				for _, t := range tool.GetTools() {
					msg = append(msg, openai.ToolMessage(t.GetContent(), t.GetId()))
				}
			}
		}
	}
	return msg
}

func (llc *largeLanguageCaller) getChatCompleteParameter(
	opts *internal_callers.ChatCompletionOptions,
	streaming bool,
) openai.ChatCompletionNewParams {
	options := openai.ChatCompletionNewParams{}
	if streaming {
		options.StreamOptions = openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		}
	}
	if len(opts.ToolDefinitions) > 0 {
		fns := make([]openai.ChatCompletionToolParam, 0, len(opts.ToolDefinitions))
		for _, tl := range opts.ToolDefinitions {
			if tl.Type != "function" {
				continue
			}
			fn := tl.Function
			if fn == nil {
				continue
			}
			funcDef := openai.FunctionDefinitionParam{
				Name: fn.Name,
			}
			if fn.Description != "" {
				funcDef.Description = openai.String(fn.Description)
			}
			if fn.Parameters != nil {
				funcDef.Parameters = fn.Parameters.ToMap()
			} else {
				funcDef.Parameters = map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				}
			}
			fns = append(fns, openai.ChatCompletionToolParam{
				Function: funcDef,
			})
		}
		options.Tools = fns
	}

	if rawName, ok := opts.ModelParameter["model.name"]; ok {
		if modelName, err := utils.AnyToString(rawName); err == nil {
			options.Model = modelName
		}
	}

	if rawParams, ok := opts.ModelParameter["model.parameters"]; ok {
		llc.applyModelParameters(&options, opts, rawParams)
	}
	return options
}

func (llc *largeLanguageCaller) applyModelParameters(
	options *openai.ChatCompletionNewParams,
	opts *internal_callers.ChatCompletionOptions,
	raw interface{},
) {
	params := map[string]interface{}{}
	switch v := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return
		}
		if err := json.Unmarshal([]byte(trimmed), &params); err != nil {
			llc.logger.Warnf("custom-llm: failed to parse model.parameters: %v", err)
			return
		}
	case map[string]interface{}:
		params = v
	default:
		return
	}

	for key, value := range params {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "user":
			if user, ok := anyToString(value); ok {
				options.User = openai.String(user)
			}
		case "reasoning_effort":
			if re, ok := anyToString(value); ok {
				options.ReasoningEffort = shared.ReasoningEffort(re)
			}
		case "seed":
			if seed, ok := anyToInt64(value); ok {
				options.Seed = openai.Int(seed)
			}
		case "top_logprobs":
			if tl, ok := anyToInt64(value); ok {
				options.TopLogprobs = openai.Int(tl)
			}
		case "metadata":
			format, _ := anyToString(value)
			var mtd map[string]string
			if err := json.Unmarshal([]byte(format), &mtd); err == nil {
				options.Metadata = shared.Metadata(mtd)
			}
		case "frequency_penalty":
			if fp, ok := anyToFloat64(value); ok {
				options.FrequencyPenalty = openai.Float(fp)
			}
		case "temperature":
			if temp, ok := anyToFloat64(value); ok {
				options.Temperature = openai.Float(temp)
			}
		case "top_p":
			if topP, ok := anyToFloat64(value); ok {
				options.TopP = openai.Float(topP)
			}
		case "presence_penalty":
			if pp, ok := anyToFloat64(value); ok {
				options.PresencePenalty = openai.Float(pp)
			}
		case "max_tokens", "max_completion_tokens":
			if maxTokens, ok := anyToInt64(value); ok {
				options.MaxTokens = openai.Int(maxTokens)
			}
		case "stop":
			if stopStr, ok := anyToString(value); ok {
				for _, stopper := range strings.Split(stopStr, ",") {
					if strings.TrimSpace(stopper) != "" {
						options.Stop.OfStringArray = append(options.Stop.OfStringArray, stopper)
					}
				}
			}
		case "tool_choice":
			if choice, ok := anyToString(value); ok && len(opts.ToolDefinitions) > 0 {
				switch choice {
				case "auto":
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("auto"),
					}
				case "required":
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("required"),
					}
				case "none":
					options.ToolChoice = openai.ChatCompletionToolChoiceOptionUnionParam{
						OfAuto: openai.String("none"),
					}
				default:
					llc.logger.Warnf("custom-llm: unrecognized tool_choice value %q; leaving tool_choice unset", choice)
				}
			}
		case "response_format":
			format, ok := anyToJSON(value)
			if !ok {
				continue
			}
			_type, ok := format["type"].(string)
			if !ok {
				continue
			}
			switch _type {
			case "json_object":
				options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
					OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
				}
			case "text":
				textParam := shared.NewResponseFormatTextParam()
				options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
					OfText: &textParam,
				}
			case "json_schema":
				schemaData, ok := format["json_schema"].(map[string]interface{})
				if !ok {
					continue
				}
				jsonSchemaParam := shared.ResponseFormatJSONSchemaJSONSchemaParam{}
				jsonData, err := json.Marshal(schemaData)
				if err != nil {
					llc.logger.Warnf("custom-llm: failed to marshal json_schema: %v", err)
					continue
				}
				if err := json.Unmarshal(jsonData, &jsonSchemaParam); err != nil {
					llc.logger.Warnf("custom-llm: failed to unmarshal json_schema: %v", err)
					continue
				}
				options.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
					OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
						JSONSchema: jsonSchemaParam,
					},
				}
			}
		}
	}
}

func anyToString(v interface{}) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", t), "0"), "."), true
	case bool:
		if t {
			return "true", true
		}
		return "false", true
	case nil:
		return "", false
	default:
		return fmt.Sprintf("%v", t), true
	}
}

func anyToInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int:
		return int64(t), true
	case int64:
		return t, true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
			return int64(f), true
		}
		return 0, false
	default:
		return 0, false
	}
}

func anyToFloat64(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0, false
		}
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func anyToJSON(v interface{}) (map[string]interface{}, bool) {
	switch t := v.(type) {
	case map[string]interface{}:
		return t, true
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil, false
		}
		out := map[string]interface{}{}
		if err := json.Unmarshal([]byte(s), &out); err != nil {
			return nil, false
		}
		return out, true
	default:
		return nil, false
	}
}
