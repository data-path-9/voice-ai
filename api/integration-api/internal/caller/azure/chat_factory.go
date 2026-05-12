// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_azure_callers

import (
	"fmt"

	internal_azure_chat_complete "github.com/rapidaai/api/integration-api/internal/caller/azure/chat_complete"
	internal_azure_chat_response "github.com/rapidaai/api/integration-api/internal/caller/azure/chat_response"
	internal_azure_websocket_streamer "github.com/rapidaai/api/integration-api/internal/caller/azure/websocket_streamer"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

const (
	OptionTransportKey = "connection.transport"
	TransportWebsocket = "websocket"
	TransportChat      = "chat_complete"
	TransportChatResp  = "chat_response"
)

func NewChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	transport := TransportChat
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportChat:
		return internal_azure_chat_complete.NewChat(logger, credential)
	case TransportChatResp:
		return internal_azure_chat_response.NewChat(logger, credential)
	case TransportWebsocket:
		return nil, fmt.Errorf("unsupported azure transport option for chat: %s", transport)
	default:
		return nil, fmt.Errorf("unsupported azure transport option: %s", transport)
	}
}

func NewChatStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	transport := TransportChat
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportWebsocket:
		return internal_azure_websocket_streamer.New(logger, credential)
	case TransportChat:
		return internal_azure_chat_complete.NewStream(logger, credential)
	case TransportChatResp:
		return internal_azure_chat_response.NewStream(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported azure transport option: %s", transport)
	}
}
