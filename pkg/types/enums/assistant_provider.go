// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package type_enums

import (
	"database/sql/driver"
	"encoding/json"
)

type AssistantProvider string

const (
	AGENTKIT  AssistantProvider = "AGENTKIT"
	AGENTFLOW AssistantProvider = "AGENTFLOW"
	WEBSOCKET AssistantProvider = "WEBSOCKET"
	MODEL     AssistantProvider = "MODEL"
)

func (m AssistantProvider) String() string {
	return string(m)
}

func (c AssistantProvider) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

func (c AssistantProvider) Value() (driver.Value, error) {
	return string(c), nil
}

func ToAssistantProvider(s string) AssistantProvider {
	switch s {
	case "AGENTKIT":
		return AGENTKIT
	case "AGENTFLOW":
		return AGENTFLOW
	case "WEBSOCKET":
		return WEBSOCKET
	default:
		return MODEL // or any other default status you prefer
	}
}
