// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_callers

import (
	"fmt"

	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type ChatCompletionOptions struct {
	AIOptions
	// request identifier
	Request *protos.ChatRequest
	// The available tool definitions that the chat completions request can use, including caller-defined functions.
	ToolDefinitions []*ToolDefinition `json:"tool_definitions"`
}

type ChatStreamCompletionOptions struct {
	AIOptions
	// request identifier
	Request *protos.StreamChatInput
	// The available tool definitions that the chat completions request can use, including caller-defined functions.
	ToolDefinitions []*ToolDefinition `json:"tool_definitions"`
}

type ToolDefinition struct {
	// type of tool
	Type string `json:"type"`

	// if type is function then definitions
	Function *FunctionDefinition `json:"functionDefinition"`
}

type FunctionDefinition struct {
	// REQUIRED; The name of the function to be called.
	Name string `json:"name"`
	// A description of what the function does. The model will use this
	// description when selecting the function and interpreting its parameters.
	Description string `json:"description"`

	// REQUIRED; The function definition details for the function tool.
	Parameters *FunctionParameter `json:"parameters"`
}

type FunctionParameter struct {
	// Required is a list of required parameter names
	Required []string `json:"required"`

	// Type specifies the data type of the parameter (e.g., "object", "array", etc.)
	Type string `json:"type"`

	// Properties is a map of parameter properties, where the key is the property name
	// and the value is a FunctionParameterProperty struct
	Properties map[string]FunctionParameterProperty `json:"properties"`
}

func (fp *FunctionParameter) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	// Type is required for valid JSON schema
	if fp.Type != "" {
		result["type"] = fp.Type
	} else {
		result["type"] = "object"
	}

	// Always include properties for valid JSON schema format
	// Output: {"type": "object", "properties": {}} instead of {"type": "object"}
	properties := make(map[string]interface{})
	if fp.Properties != nil {
		for key, prop := range fp.Properties {
			properties[key] = prop.ToMap()
		}
	}
	result["properties"] = properties

	// Only include required if there are required fields
	if len(fp.Required) > 0 {
		result["required"] = fp.Required
	}

	return result
}

type FunctionParameterProperty struct {
	// Type specifies the data type of the property (e.g., "string", "number", etc.)
	Type string `json:"type"`

	// Description provides a human-readable explanation of the property
	Description string `json:"description"`

	// Enum is an optional list of allowed values for the property
	Enum []*string `json:"enum,omitempty"`

	// Items is used for array types to describe the structure of array elements
	Items map[string]interface{} `json:"items,omitempty"`
}

func (fpp *FunctionParameterProperty) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	if fpp.Type != "" {
		result["type"] = fpp.Type
	}

	if fpp.Description != "" {
		result["description"] = fpp.Description
	}

	if fpp.Enum != nil {
		result["enum"] = fpp.Enum
	}

	if fpp.Items != nil {
		result["items"] = fpp.Items
	}

	return result
}

func NewChatOptions(
	uuID uint64,
	irRequest *protos.ChatRequest,
	preHook func(rst map[string]interface{}),
	postHook func(rst map[string]interface{}, metrics []*protos.Metric),
) *ChatCompletionOptions {
	cc := &ChatCompletionOptions{
		Request: irRequest,
		AIOptions: AIOptions{
			RequestId:      uuID,
			PreHook:        preHook,
			PostHook:       postHook,
			ModelParameter: irRequest.GetModelParameters(),
		},
	}
	cc.ToolDefinitions = castToolDefinitions(irRequest.GetToolDefinitions())
	return cc
}

func NewChatStreamOptions(
	uuID uint64,
	irRequest *protos.StreamChatInput,
	preHook func(rst map[string]interface{}),
	postHook func(rst map[string]interface{}, metrics []*protos.Metric),
) *ChatStreamCompletionOptions {
	cc := &ChatStreamCompletionOptions{
		Request: irRequest,
		AIOptions: AIOptions{
			RequestId:      uuID,
			PreHook:        preHook,
			PostHook:       postHook,
			ModelParameter: irRequest.GetModelParameters(),
		},
	}
	cc.ToolDefinitions = castToolDefinitions(irRequest.GetToolDefinitions())
	return cc
}

func castToolDefinitions(toolDefinitions []*protos.ToolDefinition) []*ToolDefinition {
	var casted []*ToolDefinition
	err := utils.Cast(toolDefinitions, &casted)
	if err != nil {
		fmt.Printf("initializing function definition with err %+v", err)
	}
	return casted
}
