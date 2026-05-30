// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_asterisk

import (
	"encoding/json"
	"strconv"
	"strings"
)

func ParseAsteriskEvent(data string) (*AsteriskMediaEvent, error) {
	event := &AsteriskMediaEvent{}
	if err := json.Unmarshal([]byte(data), event); err == nil && (event.Event != "" || event.Command != "") {
		return event, nil
	}
	event.RawMessage = data
	event.Event = parseEventType(data)
	params := parseKeyValuePairs(data)
	if channel, ok := params["channel"]; ok {
		event.Channel = channel
	}
	if rawFrameSize, ok := params["optimal_frame_size"]; ok {
		if frameSize, err := strconv.Atoi(rawFrameSize); err == nil {
			event.OptimalFrameSize = frameSize
		}
	}
	return event, nil
}

func parseEventType(data string) string {
	for index, char := range data {
		if char == ' ' {
			return data[:index]
		}
	}
	return data
}

func parseKeyValuePairs(data string) map[string]string {
	result := make(map[string]string)
	parts := strings.Fields(data)
	for _, part := range parts[1:] {
		for index, char := range part {
			if char == ':' {
				result[part[:index]] = part[index+1:]
				break
			}
		}
	}
	return result
}
