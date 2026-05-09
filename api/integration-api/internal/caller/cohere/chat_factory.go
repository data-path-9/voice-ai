// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_callers

import (
	"fmt"

	internal_cohere_chat_v2 "github.com/rapidaai/api/integration-api/internal/caller/cohere/chat_v2"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

const (
	OptionTransportKey = "connection.transport"
	TransportChatV2    = "chat_v2"
)

func NewChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	transport := TransportChatV2
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportChatV2:
		return internal_cohere_chat_v2.NewChat(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported cohere transport option: %s", transport)
	}
}

func NewChatStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	transport := TransportChatV2
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportChatV2:
		return internal_cohere_chat_v2.NewStream(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported cohere transport option: %s", transport)
	}
}
