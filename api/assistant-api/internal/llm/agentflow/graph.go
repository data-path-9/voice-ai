// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_agentflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrDefinitionRequired = errors.New("agentflow definition is required")
	ErrInvalidDefinition  = errors.New("agentflow definition is invalid")
)

type graph struct {
	definition Definition
	nodes      map[string]Node
	outgoing   map[string][]Edge
}

func compileDefinition(raw interface{}) (*graph, error) {
	if raw == nil {
		return nil, ErrDefinitionRequired
	}

	var definition Definition
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal: %w", ErrInvalidDefinition, err)
	}
	if err := json.Unmarshal(bytes, &definition); err != nil {
		return nil, fmt.Errorf("%w: unmarshal: %w", ErrInvalidDefinition, err)
	}
	if strings.TrimSpace(definition.SchemaVersion) == "" {
		return nil, fmt.Errorf("%w: schemaVersion is required", ErrInvalidDefinition)
	}
	if strings.TrimSpace(definition.EntryNodeID) == "" {
		return nil, fmt.Errorf("%w: entryNodeId is required", ErrInvalidDefinition)
	}
	if len(definition.Nodes) == 0 {
		return nil, fmt.Errorf("%w: nodes are required", ErrInvalidDefinition)
	}

	nodes := make(map[string]Node, len(definition.Nodes))
	for _, node := range definition.Nodes {
		node.ID = strings.TrimSpace(node.ID)
		node.Type = strings.TrimSpace(node.Type)
		if node.ID == "" {
			return nil, fmt.Errorf("%w: node id is required", ErrInvalidDefinition)
		}
		if node.Type == "" {
			return nil, fmt.Errorf("%w: node %s type is required", ErrInvalidDefinition, node.ID)
		}
		if !isSupportedRuntimeNode(node.Type) {
			return nil, fmt.Errorf("%w: unsupported runtime node type %q", ErrInvalidDefinition, node.Type)
		}
		if _, exists := nodes[node.ID]; exists {
			return nil, fmt.Errorf("%w: duplicate node id %q", ErrInvalidDefinition, node.ID)
		}
		nodes[node.ID] = node
	}
	if _, ok := nodes[definition.EntryNodeID]; !ok {
		return nil, fmt.Errorf("%w: entry node %q does not exist", ErrInvalidDefinition, definition.EntryNodeID)
	}

	outgoing := make(map[string][]Edge)
	for _, edge := range definition.Edges {
		edge.ID = strings.TrimSpace(edge.ID)
		edge.Source = strings.TrimSpace(edge.Source)
		edge.SourceHandle = strings.TrimSpace(edge.SourceHandle)
		edge.Target = strings.TrimSpace(edge.Target)
		edge.TargetHandle = strings.TrimSpace(edge.TargetHandle)
		if edge.Source == "" || edge.Target == "" {
			return nil, fmt.Errorf("%w: edge source and target are required", ErrInvalidDefinition)
		}
		if _, ok := nodes[edge.Source]; !ok {
			return nil, fmt.Errorf("%w: edge source %q does not exist", ErrInvalidDefinition, edge.Source)
		}
		if _, ok := nodes[edge.Target]; !ok {
			return nil, fmt.Errorf("%w: edge target %q does not exist", ErrInvalidDefinition, edge.Target)
		}
		outgoing[edge.Source] = append(outgoing[edge.Source], edge)
	}
	for source := range outgoing {
		sort.SliceStable(outgoing[source], func(i, j int) bool {
			return outgoing[source][i].ID < outgoing[source][j].ID
		})
	}

	return &graph{definition: definition, nodes: nodes, outgoing: outgoing}, nil
}

func isSupportedRuntimeNode(nodeType string) bool {
	switch nodeType {
	case NodeTypeChatInput, NodeTypeAgent, NodeTypeMessage, NodeTypeCondition, NodeTypeTransfer, NodeTypeEnd, NodeTypeStickyNote:
		return true
	default:
		return false
	}
}

func (g *graph) node(id string) (Node, bool) {
	if g == nil {
		return Node{}, false
	}
	node, ok := g.nodes[id]
	return node, ok
}

func (g *graph) firstTarget(sourceID string) (string, bool) {
	if g == nil {
		return "", false
	}
	edges := g.outgoing[sourceID]
	if len(edges) == 0 {
		return "", false
	}
	return edges[0].Target, true
}

func (g *graph) targetByHandle(sourceID string, handles ...string) (string, bool) {
	if g == nil {
		return "", false
	}
	edges := g.outgoing[sourceID]
	for _, handle := range handles {
		handle = strings.TrimSpace(handle)
		if handle == "" {
			continue
		}
		for _, edge := range edges {
			if edge.SourceHandle == handle {
				return edge.Target, true
			}
		}
	}
	return "", false
}

func (g *graph) elseTarget(sourceID string) (string, bool) {
	if target, ok := g.targetByHandle(sourceID, "else", "fallback", "default"); ok {
		return target, true
	}
	return g.firstTarget(sourceID)
}
