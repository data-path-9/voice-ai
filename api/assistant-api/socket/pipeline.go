// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_socket

import (
	"context"

	"github.com/rapidaai/api/assistant-api/config"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
)

// newSessionPipeline creates a pipeline dispatcher wired for session handling
// (resolve context, create streamer, create talker, run Talk, observe, complete).
func newSessionPipeline(ctx context.Context, cfg *config.AssistantConfig, logger commons.Logger,
	postgres connectors.PostgresConnector,
	redis connectors.RedisConnector,
	opensearch connectors.OpenSearchConnector,
	fileStorage storages.Storage,
) (*channel_pipeline.Dispatcher, *channel_telephony.InboundDispatcher) {
	store := callcontext.NewStore(postgres, logger)
	vaultClient := web_client.NewVaultClientGRPC(&cfg.AppConfig, logger, redis)
	conversationService := internal_assistant_service.NewAssistantConversationService(logger, postgres, fileStorage)
	inbound := channel_telephony.NewInboundDispatcher(
		channel_telephony.WithConfig(cfg),
		channel_telephony.WithLogger(logger),
		channel_telephony.WithStore(store),
		channel_telephony.WithVaultClient(vaultClient),
		channel_telephony.WithAssistantService(internal_assistant_service.NewAssistantService(cfg, logger, postgres, opensearch)),
		channel_telephony.WithConversationService(conversationService),
		channel_telephony.WithTelephonyOption(channel_telephony.TelephonyOption{}),
	)

	dispatcher := channel_pipeline.NewDispatcher(
		channel_pipeline.WithLogger(logger),
		channel_pipeline.WithConversationService(conversationService),
		channel_pipeline.WithAssistantService(internal_assistant_service.NewAssistantService(cfg, logger, postgres, opensearch)),
	)

	dispatcher.Start(ctx)
	return dispatcher, inbound
}
