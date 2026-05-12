// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

type AssistantAuthenticationService interface {
	Get(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
	) (*internal_assistant_entity.AssistantAuthentication, error)

	Create(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		provider string,
		status string,
		failBehavior string,
		timeoutMs uint64,
		options []*protos.Metadata,
	) (*internal_assistant_entity.AssistantAuthentication, error)

	Disable(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
	) (*internal_assistant_entity.AssistantAuthentication, error)
}
