// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package observability_api

import (
	"github.com/rapidaai/api/assistant-api/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/protos"
)

type observabilityApi struct {
	cfg        *config.AssistantConfig
	logger     commons.Logger
	opensearch connectors.OpenSearchConnector
}

type observabilityGrpcApi struct {
	observabilityApi
	protos.UnimplementedObservabilityServiceServer
}

func NewObservabilityGRPCApi(
	config *config.AssistantConfig,
	logger commons.Logger,
	opensearch connectors.OpenSearchConnector,
) protos.ObservabilityServiceServer {
	return &observabilityGrpcApi{
		observabilityApi: observabilityApi{
			cfg:        config,
			logger:     logger,
			opensearch: opensearch,
		},
	}
}
