// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package schema

import "github.com/rapidaai/pkg/utils"

const (
	NodeTypeChatInput  = "chat-input"
	NodeTypeAgent      = "prompt"
	NodeTypeMessage    = "message"
	NodeTypeCondition  = "condition"
	NodeTypeTransfer   = "transfer"
	NodeTypeEnd        = "end"
	NodeTypeStickyNote = "sticky-note"

	DefaultSchemaVersion = "2026-07-06"
)

type Definition struct {
	SchemaVersion string   `json:"schemaVersion"`
	EntryNodeID   string   `json:"entryNodeId"`
	Name          string   `json:"name,omitempty"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Nodes         []Node   `json:"nodes"`
	Edges         []Edge   `json:"edges"`
}

type Node struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Label    string             `json:"label"`
	Position map[string]float64 `json:"position,omitempty"`
	Config   utils.Option       `json:"config,omitempty"`
}

type Edge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	Target       string `json:"target"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

type AgentTransition struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Parameters  []AgentTransitionParameter `json:"parameters,omitempty"`
	Properties  utils.Option               `json:"properties,omitempty"`
	Required    []string                   `json:"required,omitempty"`
}

type AgentTransitionParameter struct {
	ID          string       `json:"id,omitempty"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	Description string       `json:"description"`
	Required    bool         `json:"required"`
	Schema      utils.Option `json:"schema,omitempty"`
}

type ChatInputArgument struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	DefaultValue string `json:"defaultvalue"`
}

type ConditionRule struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	Field        string `json:"field"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
	Left         string `json:"left,omitempty"`
}
