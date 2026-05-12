// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_callers

import (
	"fmt"

	internal_openai_chat_complete "github.com/rapidaai/api/integration-api/internal/caller/openai/chat_complete"
	internal_openai_chat_response "github.com/rapidaai/api/integration-api/internal/caller/openai/chat_response"
	internal_openai_websocket_streamer "github.com/rapidaai/api/integration-api/internal/caller/openai/websocket_streamer"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	protos "github.com/rapidaai/protos"
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
		return internal_openai_chat_complete.NewChat(logger, credential)
	case TransportChatResp:
		return internal_openai_chat_response.NewChat(logger, credential)
	case TransportWebsocket:
		return nil, fmt.Errorf("unsupported openai transport option for chat: %s", transport)
	default:
		return nil, fmt.Errorf("unsupported openai transport option: %s", transport)
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
		return internal_openai_websocket_streamer.New(logger, credential)
	case TransportChat:
		return internal_openai_chat_complete.NewStream(logger, credential)
	case TransportChatResp:
		return internal_openai_chat_response.NewStream(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported openai transport option: %s", transport)
	}
}
