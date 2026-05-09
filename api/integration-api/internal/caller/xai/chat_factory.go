// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_xai_callers

import (
	"fmt"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	protos "github.com/rapidaai/protos"
)

const (
	OptionTransportKey = "connection.transport"
	TransportGRPC      = "grpc"
)

func NewChat(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.Chat, error) {
	transport := TransportGRPC
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportGRPC:
		return newGRPCChat(logger, credential, connectionOptions)
	default:
		return nil, fmt.Errorf("unsupported xai transport option: %s", transport)
	}
}

func NewChatStream(
	logger commons.Logger,
	credential *protos.Credential,
	connectionOptions map[string]string,
) (internal_callers.ChatStream, error) {
	transport := TransportGRPC
	if connectionOptions != nil {
		if option, ok := connectionOptions[OptionTransportKey]; ok && option != "" {
			transport = option
		}
	}

	switch transport {
	case TransportGRPC:
		return newGRPCStream(logger, credential, connectionOptions)
	default:
		return nil, fmt.Errorf("unsupported xai transport option: %s", transport)
	}
}
