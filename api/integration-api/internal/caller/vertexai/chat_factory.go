// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_callers

import (
	"fmt"

	internal_vertexai_generate_content "github.com/rapidaai/api/integration-api/internal/caller/vertexai/generate_content"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

const (
	OptionTransportKey       = "connection.transport"
	TransportGenerateContent = "generate_content"
)

func NewChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	transport := TransportGenerateContent
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportGenerateContent:
		return internal_vertexai_generate_content.NewChat(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported vertexai transport option: %s", transport)
	}
}

func NewChatStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	transport := TransportGenerateContent
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportGenerateContent:
		return internal_vertexai_generate_content.NewStream(logger, credential)
	default:
		return nil, fmt.Errorf("unsupported vertexai transport option: %s", transport)
	}
}
