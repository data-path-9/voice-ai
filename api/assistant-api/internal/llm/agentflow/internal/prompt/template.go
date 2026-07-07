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

type PromptTemplate struct {
	Prompt    []PromptTemplateMessage  `json:"prompt"`
	Variables []PromptTemplateVariable `json:"variables"`
}

type PromptTemplateMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type PromptTemplateVariable struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultvalue"`
}

func BuildPromptMessages(request Request) []*protos.Message {
	templateArguments := make(utils.Option, len(request.Variables))
	for key, value := range request.Variables {
		templateArguments[key] = value
	}

	promptTemplate := parsePromptTemplate(request.Node)
	for _, variable := range promptTemplate.Variables {
		if variable.Name == "" {
			continue
		}
		if _, exists := templateArguments[variable.Name]; !exists {
			templateArguments[variable.Name] = variable.DefaultValue
		}
	}

	messages := make([]*protos.Message, 0, len(promptTemplate.Prompt)+len(request.History)+1)
	for _, item := range promptTemplate.Prompt {
		content := renderTemplate(item.Content, templateArguments)
		if strings.TrimSpace(content) == "" {
			continue
		}
		messages = append(messages, messageForRole(item.Role, content))
	}
	if len(messages) == 0 {
		messages = append(messages, messageForRole("system", renderTemplate(request.Node.StringConfig("prompt"), templateArguments)))
	}
	messages = append(messages, request.History...)
	if request.InputText != "" {
		messages = append(messages, messageForRole("user", request.InputText))
	} else if request.ContinuationText != "" {
		messages = append(messages, messageForRole("user", request.ContinuationText))
	}
	return messages
}

func parsePromptTemplate(node schema.Node) PromptTemplate {
	rawPromptTemplate := node.Config["prompt_template"]
	var promptTemplate PromptTemplate
	switch typedPromptTemplate := rawPromptTemplate.(type) {
	case string:
		_ = json.Unmarshal([]byte(typedPromptTemplate), &promptTemplate)
	default:
		bytes, err := json.Marshal(rawPromptTemplate)
		if err == nil {
			_ = json.Unmarshal(bytes, &promptTemplate)
		}
	}
	return promptTemplate
}

func messageForRole(role, content string) *protos.Message {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "user":
		return &protos.Message{Role: "user", Message: &protos.Message_User{User: &protos.UserMessage{Content: content}}}
	case "assistant":
		return &protos.Message{Role: "assistant", Message: &protos.Message_Assistant{Assistant: &protos.AssistantMessage{Contents: []string{content}}}}
	default:
		return &protos.Message{Role: "system", Message: &protos.Message_System{System: &protos.SystemMessage{Content: content}}}
	}
}

func renderTemplate(template string, arguments utils.Option) string {
	renderedTemplate := template
	for key, value := range arguments {
		renderedTemplate = strings.ReplaceAll(renderedTemplate, "{{"+key+"}}", fmt.Sprintf("%v", value))
		renderedTemplate = strings.ReplaceAll(renderedTemplate, "{{ "+key+" }}", fmt.Sprintf("%v", value))
	}
	return renderedTemplate
}
