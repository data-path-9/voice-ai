// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_anthropic_callers

import (
	"fmt"

	internal_anthropic_messages "github.com/rapidaai/api/integration-api/internal/caller/anthropic/messages"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

const (
	OptionTransportKey = "connection.transport"
	TransportMessages  = "messages"
)

func NewChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	transport := TransportMessages
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportMessages:
		return internal_anthropic_messages.NewChat(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported anthropic transport option: %s", transport)
	}
}

func NewChatStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	transport := TransportMessages
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportMessages:
		return internal_anthropic_messages.NewStream(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported anthropic transport option: %s", transport)
	}
}
