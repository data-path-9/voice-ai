// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_artifact

import (
	"context"
	"fmt"

	internal_artifact_storage "github.com/rapidaai/api/assistant-api/internal/artifact/storage"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

const (
	providerAWS          = "aws"
	providerAzureStorage = "azure-storage"
)

func NewExecutor(
	logger commons.Logger,
	_ context.Context,
	configuration *internal_assistant_entity.AssistantConfiguration,
	caller internal_type.InternalCaller,
	auth types.SimplePrinciple,
	onPacket func(context.Context, ...internal_type.Packet) error,
) (internal_type.ArtifactPushExecutor, error) {
	switch configuration.Provider {
	case providerAWS:
		return internal_artifact_storage.NewAWSExecutor(logger, configuration, caller, auth, onPacket), nil
	case providerAzureStorage:
		return internal_artifact_storage.NewAzureStorageExecutor(logger, configuration, caller, auth, onPacket), nil
	default:
		return nil, fmt.Errorf("artifact push storage: unsupported provider %q", configuration.Provider)
	}
}
