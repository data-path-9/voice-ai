// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_generate_content

import (
	"encoding/json"
	"strings"

	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	chatRoleAssistant = "assistant"
	chatRoleModel     = "model"
	chatRoleSystem    = "system"
	chatRoleTool      = "tool"
	chatRoleUser      = "user"
)

func buildHistory(
	logger commons.Logger,
	allMessages []*protos.Message,
) (*genai.Content, []*genai.Content, genai.Part) {
	history := make([]*genai.Content, 0, len(allMessages))

	for _, msg := range allMessages {
		switch msg.GetRole() {
		case chatRoleUser:
			if user := msg.GetUser(); user != nil {
				history = append(history, &genai.Content{
					Role:  chatRoleUser,
					Parts: []*genai.Part{{Text: user.GetContent()}},
				})
			}
		case chatRoleModel, chatRoleAssistant:
			if assistant := msg.GetAssistant(); assistant != nil {
				content := &genai.Content{
					Role:  chatRoleModel,
					Parts: make([]*genai.Part, 0, len(assistant.GetContents())+len(assistant.GetToolCalls())),
				}
				for _, ct := range assistant.GetContents() {
					content.Parts = append(content.Parts, &genai.Part{Text: ct})
				}
				for _, tc := range assistant.GetToolCalls() {
					if tc.GetFunction() == nil {
						continue
					}
					argumentMap := make(map[string]any)
					if err := json.Unmarshal([]byte(tc.GetFunction().GetArguments()), &argumentMap); err != nil {
						argumentMap = make(map[string]any)
					}
					content.Parts = append(content.Parts, &genai.Part{
						FunctionCall: &genai.FunctionCall{
							ID:   tc.GetId(),
							Args: argumentMap,
							Name: tc.GetFunction().GetName(),
						},
					})
				}
				history = append(history, content)
			}
		case chatRoleSystem:
			if system := msg.GetSystem(); system != nil {
				history = append(history, &genai.Content{
					Parts: []*genai.Part{{Text: system.GetContent()}},
				})
			}
		case chatRoleTool:
			if tool := msg.GetTool(); tool != nil {
				content := &genai.Content{
					Role:  chatRoleUser,
					Parts: make([]*genai.Part, 0, len(tool.GetTools())),
				}
				for _, t := range tool.GetTools() {
					var responseMap map[string]any
					if err := json.Unmarshal([]byte(t.GetContent()), &responseMap); err != nil {
						responseMap = make(map[string]any)
					}
					content.Parts = append(content.Parts, &genai.Part{
						FunctionResponse: &genai.FunctionResponse{
							Name:     t.GetName(),
							ID:       t.GetId(),
							Response: responseMap,
						},
					})
				}
				history = append(history, content)
			}
		default:
			logger.Warnf("Unknown role: %s", msg.GetRole())
		}
	}

	var lastPart genai.Part
	if len(history) > 0 && len(history[len(history)-1].Parts) > 0 {
		lastPart = *history[len(history)-1].Parts[0]
	}

	if len(history) == 0 {
		return nil, history, lastPart
	}
	return history[0], history[1:], lastPart
}

func toGoogleSchema(fp *internal_callers.FunctionParameter) *genai.Schema {
	schema := &genai.Schema{
		Type:       genai.Type(fp.Type),
		Properties: make(map[string]*genai.Schema),
	}
	if fp.Required != nil {
		schema.Required = fp.Required
	}
	for key, prop := range fp.Properties {
		schema.Properties[key] = googleFunctionParameterPropertyToSchema(&prop)
	}
	return schema
}

func googleFunctionParameterPropertyToSchema(fpp *internal_callers.FunctionParameterProperty) *genai.Schema {
	schema := &genai.Schema{
		Type:        genai.Type(fpp.Type),
		Description: fpp.Description,
	}
	if fpp.Enum != nil {
		schema.Enum = make([]string, len(fpp.Enum))
		for i, v := range fpp.Enum {
			if v != nil {
				schema.Enum[i] = *v
			}
		}
	}
	if fpp.Items != nil {
		if itemTypeRaw, ok := fpp.Items["type"]; ok {
			if itemType, ok := itemTypeRaw.(string); ok && itemType != "" {
				schema.Items = &genai.Schema{Type: genai.Type(itemType)}
			}
		}
	}
	return schema
}

func buildContentConfig(opts *internal_callers.ChatCompletionOptions) (string, *genai.GenerateContentConfig) {
	return buildGenerateContentConfig(opts.ModelParameter, opts.ToolDefinitions)
}

func buildStreamContentConfig(opts *internal_callers.ChatStreamCompletionOptions) (string, *genai.GenerateContentConfig) {
	return buildGenerateContentConfig(opts.ModelParameter, opts.ToolDefinitions)
}

func buildGenerateContentConfig(
	modelParameter map[string]*anypb.Any,
	toolDefinitions []*internal_callers.ToolDefinition,
) (mdl string, config *genai.GenerateContentConfig) {
	config = &genai.GenerateContentConfig{}

	if len(toolDefinitions) > 0 {
		fd := make([]*genai.FunctionDeclaration, 0, len(toolDefinitions))
		for _, tl := range toolDefinitions {
			if tl.Type != "function" || tl.Function == nil {
				continue
			}
			fn := tl.Function
			funcDef := &genai.FunctionDeclaration{
				Name:        fn.Name,
				Description: fn.Description,
			}
			if fn.Parameters != nil {
				funcDef.Parameters = toGoogleSchema(fn.Parameters)
			}
			fd = append(fd, funcDef)
		}
		if len(fd) > 0 {
			config.Tools = []*genai.Tool{{
				FunctionDeclarations: fd,
			}}
		}
	}

	for key, value := range modelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				mdl = modelName
			}
		case "model.temperature":
			if temp, err := utils.AnyToFloat32(value); err == nil {
				config.Temperature = utils.Ptr(temp)
			}
		case "model.top_p":
			if topP, err := utils.AnyToFloat32(value); err == nil {
				config.TopP = utils.Ptr(topP)
			}
		case "model.top_k":
			if topK, err := utils.AnyToFloat32(value); err == nil {
				config.TopK = utils.Ptr(topK)
			}
		case "model.max_completion_tokens":
			if maxTokens, err := utils.AnyToInt64(value); err == nil {
				config.MaxOutputTokens = int32(maxTokens)
			}
		case "model.stop":
			if stopStr, err := utils.AnyToString(value); err == nil {
				if strings.TrimSpace(stopStr) != "" {
					config.StopSequences = strings.Split(stopStr, ",")
				}
			}
		case "model.frequency_penalty":
			if fp, err := utils.AnyToFloat32(value); err == nil {
				config.FrequencyPenalty = utils.Ptr(fp)
			}
		case "model.presence_penalty":
			if pp, err := utils.AnyToFloat32(value); err == nil {
				config.PresencePenalty = utils.Ptr(pp)
			}
		case "model.seed":
			if seed, err := utils.AnyToInt32(value); err == nil {
				config.Seed = utils.Ptr(seed)
			}
		case "model.thinking":
			if format, err := utils.AnyToJSON(value); err == nil {
				includeThoughts, _ := format["include_thoughts"].(bool)
				if includeThoughts {
					config.ThinkingConfig = &genai.ThinkingConfig{
						IncludeThoughts: true,
					}
					if thinkingBudgetRaw, ok := format["thinking_budget"]; ok {
						if budgetValue, err := structpb.NewValue(thinkingBudgetRaw); err == nil {
							if budgetAny, err := anypb.New(budgetValue); err == nil {
								if thinkingBudget, err := utils.AnyToInt32(budgetAny); err == nil {
									config.ThinkingConfig.ThinkingBudget = utils.Ptr(thinkingBudget)
								}
							}
						}
					}
				}
			}
		case "model.response_format":
			if format, err := utils.AnyToJSON(value); err == nil {
				responseMimeType, _ := format["response_mime_type"].(string)
				responseSchema, _ := format["response_schema"].(map[string]interface{})
				switch responseMimeType {
				case "text/x.enum":
					if responseSchema != nil {
						config.ResponseMIMEType = "text/x.enum"
						config.ResponseJsonSchema = responseSchema
					}
				case "application/json":
					if responseSchema != nil {
						config.ResponseMIMEType = "application/json"
						config.ResponseJsonSchema = responseSchema
					}
				}
			}
		}
	}
	return mdl, config
}

func messageJSON(
	model string,
	cfg *genai.GenerateContentConfig,
	history []*genai.Content,
	current genai.Part,
) map[string]interface{} {
	return utils.ToJson(struct {
		Config               *genai.GenerateContentConfig
		Current              genai.Part
		Model                string
		ComprehensiveHistory []*genai.Content
	}{
		Model:                model,
		Config:               cfg,
		Current:              current,
		ComprehensiveHistory: history,
	})
}
