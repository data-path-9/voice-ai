// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package prompt

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func TransitionToolDefinitions(transitions []schema.AgentTransition) []*protos.ToolDefinition {
	toolDefinitions := make([]*protos.ToolDefinition, 0, len(transitions))
	for _, transition := range transitions {
		name := strings.TrimSpace(transition.Name)
		if name == "" {
			continue
		}

		properties := make(map[string]*protos.FunctionParameterProperty)
		required := make([]string, 0)
		for _, parameter := range transition.Parameters {
			if parameter.Name == "" {
				continue
			}
			property := &protos.FunctionParameterProperty{
				Type:        normalizeParameterType(parameter.Type),
				Description: parameter.Description,
			}
			if enumValues, ok := parameter.Schema["enum"].([]interface{}); ok {
				for _, enumValue := range enumValues {
					property.Enum = append(property.Enum, fmt.Sprintf("%v", enumValue))
				}
			}
			properties[parameter.Name] = property
			if parameter.Required {
				required = append(required, parameter.Name)
			}
		}

		toolDefinitions = append(toolDefinitions, &protos.ToolDefinition{
			Type: "function",
			FunctionDefinition: &protos.FunctionDefinition{
				Name:        name,
				Description: transition.Description,
				Parameters: &protos.FunctionParameter{
					Type:       "object",
					Required:   required,
					Properties: properties,
				},
			},
		})
	}
	return toolDefinitions
}

func StreamOutputToPromptResult(output *protos.StreamChatOutput, transitions []schema.AgentTransition) (Result, bool, error) {
	if output.GetError() != nil {
		return Result{}, true, fmt.Errorf("agentflow: model error: %s", output.GetError().GetErrorMessage())
	}
	message := output.GetData()
	if message == nil || message.GetAssistant() == nil {
		return Result{}, false, nil
	}

	assistant := message.GetAssistant()
	result := Result{Text: strings.Join(assistant.GetContents(), ""), Metrics: output.GetMetrics()}
	for _, toolCall := range assistant.GetToolCalls() {
		function := toolCall.GetFunction()
		if function == nil {
			continue
		}
		for _, transition := range transitions {
			if transition.Name != function.GetName() {
				continue
			}
			result.TransitionID = transition.ID
			result.TransitionName = transition.Name
			result.Arguments = utils.Option{}
			_ = json.Unmarshal([]byte(function.GetArguments()), &result.Arguments)
			return result, true, nil
		}
	}
	return result, len(output.GetMetrics()) > 0, nil
}

func normalizeParameterType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "integer":
		return "integer"
	case "number":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "array":
		return "array"
	case "object":
		return "object"
	default:
		return "string"
	}
}
