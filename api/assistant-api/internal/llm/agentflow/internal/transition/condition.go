// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package transition

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rapidaai/api/assistant-api/internal/llm/agentflow/internal/schema"
)

func ConditionMatches(rule schema.ConditionRule, variables map[string]interface{}, nodeOutputs map[string]map[string]interface{}) bool {
	value, exists := ResolveRuntimeValue(ConditionValuePath(rule), variables, nodeOutputs)
	if !exists {
		return strings.EqualFold(strings.TrimSpace(rule.Operator), "not_exists")
	}

	right := strings.TrimSpace(rule.Value)
	switch strings.ToLower(strings.TrimSpace(rule.Operator)) {
	case "exists":
		return true
	case "not_exists":
		return false
	case "equals", "equal", "==":
		return fmt.Sprintf("%v", value) == right
	case "not_equals", "not equals", "!=", "not equal":
		return fmt.Sprintf("%v", value) != right
	case "contains":
		return strings.Contains(strings.ToLower(fmt.Sprintf("%v", value)), strings.ToLower(right))
	case "greater_than", "greater than", ">":
		leftNumber, leftError := strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
		rightNumber, rightError := strconv.ParseFloat(right, 64)
		return leftError == nil && rightError == nil && leftNumber > rightNumber
	case "greater_than_or_equal", ">=":
		leftNumber, leftError := strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
		rightNumber, rightError := strconv.ParseFloat(right, 64)
		return leftError == nil && rightError == nil && leftNumber >= rightNumber
	case "less_than", "less than", "<":
		leftNumber, leftError := strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
		rightNumber, rightError := strconv.ParseFloat(right, 64)
		return leftError == nil && rightError == nil && leftNumber < rightNumber
	case "less_than_or_equal", "<=":
		leftNumber, leftError := strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
		rightNumber, rightError := strconv.ParseFloat(right, 64)
		return leftError == nil && rightError == nil && leftNumber <= rightNumber
	case "is true":
		booleanValue, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprintf("%v", value)))
		return err == nil && booleanValue
	case "is false":
		booleanValue, err := strconv.ParseBool(strings.TrimSpace(fmt.Sprintf("%v", value)))
		return err == nil && !booleanValue
	default:
		return false
	}
}

func ConditionValuePath(rule schema.ConditionRule) string {
	left := strings.TrimSpace(rule.Left)
	if left != "" {
		return left
	}

	field := strings.TrimSpace(rule.Field)
	if field == "" {
		return ""
	}
	if strings.HasPrefix(field, "argument.") || strings.HasPrefix(field, "arguments.") {
		return field
	}

	sourceNodeID := strings.TrimSpace(rule.SourceNodeID)
	if sourceNodeID == "" || strings.HasPrefix(field, sourceNodeID+".") {
		return field
	}
	return sourceNodeID + "." + field
}

func ResolveRuntimeValue(path string, variables map[string]interface{}, nodeOutputs map[string]map[string]interface{}) (interface{}, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}
	if strings.HasPrefix(path, "argument.") {
		return nestedValue(variables, strings.TrimPrefix(path, "argument."))
	}
	if strings.HasPrefix(path, "arguments.") {
		return nestedValue(variables, strings.TrimPrefix(path, "arguments."))
	}

	parts := strings.Split(path, ".")
	if len(parts) >= 2 {
		nodeOutput, ok := nodeOutputs[parts[0]]
		if !ok {
			return nil, false
		}
		if len(parts) == 2 {
			value, exists := nodeOutput[parts[1]]
			return value, exists
		}
		return nestedValue(nodeOutput, strings.Join(parts[1:], "."))
	}
	return nestedValue(variables, path)
}

func nestedValue(values map[string]interface{}, path string) (interface{}, bool) {
	if values == nil {
		return nil, false
	}

	current := interface{}(values)
	for _, pathSegment := range strings.Split(path, ".") {
		pathSegment = strings.TrimSpace(pathSegment)
		if pathSegment == "" {
			return nil, false
		}
		currentMap, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		nextValue, exists := currentMap[pathSegment]
		if !exists {
			return nil, false
		}
		current = nextValue
	}
	return current, true
}
