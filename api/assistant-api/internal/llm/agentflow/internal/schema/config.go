// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rapidaai/pkg/utils"
)

func (node Node) StringConfig(key string) string {
	if node.Config == nil {
		return ""
	}
	value, ok := node.Config[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func (node Node) IntConfig(key string, fallback int) int {
	value := node.StringConfig(key)
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (node Node) ChatInputArguments() []ChatInputArgument {
	var arguments []ChatInputArgument
	node.parseJSONArrayConfig("arguments", &arguments)
	return arguments
}

func (node Node) AgentTransitions() []AgentTransition {
	var transitions []AgentTransition
	node.parseJSONArrayConfig("transitions", &transitions)
	return transitions
}

func (node Node) ConditionRules() []ConditionRule {
	var conditions []ConditionRule
	node.parseJSONArrayConfig("conditions", &conditions)
	return conditions
}

func (node Node) ModelParameters() utils.Option {
	var rows []struct {
		Key   string      `json:"key"`
		Value interface{} `json:"value"`
	}
	node.parseJSONArrayConfig("model_parameters", &rows)

	parameters := make(utils.Option, len(rows))
	for _, row := range rows {
		key := strings.TrimSpace(row.Key)
		if key == "" {
			continue
		}
		parameters[key] = row.Value
	}
	return parameters
}

func (node Node) parseJSONArrayConfig(key string, target interface{}) {
	if node.Config == nil {
		return
	}
	value := node.Config[key]
	if value == nil {
		return
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return
		}
		_ = json.Unmarshal([]byte(typed), target)
	default:
		bytes, err := json.Marshal(value)
		if err != nil {
			return
		}
		_ = json.Unmarshal(bytes, target)
	}
}
