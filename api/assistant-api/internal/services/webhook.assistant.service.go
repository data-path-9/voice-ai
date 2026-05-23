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
	"github.com/rapidaai/protos"
)

type AssistantWebhookService interface {
	Get(ctx context.Context, auth types.SimplePrinciple, webhookId uint64, assistantId uint64) (*internal_assistant_entity.AssistantWebhook, error)
	Delete(ctx context.Context, auth types.SimplePrinciple, webhookId uint64, assistantId uint64) (*internal_assistant_entity.AssistantWebhook, error)
	Create(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		provider string,
		assistantEvents []string,
		options []*protos.Metadata,
		executionPriority uint32,
		description *string,
	) (*internal_assistant_entity.AssistantWebhook, error)
	Update(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		webhookId uint64,
		provider string,
		assistantEvents []string,
		options []*protos.Metadata,
		executionPriority uint32,
		description *string,
	) (*internal_assistant_entity.AssistantWebhook, error)
	GetAll(
		ctx context.Context,
		auth types.SimplePrinciple,
		assistantId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate,
	) (int64, []*internal_assistant_entity.AssistantWebhook, error)
}
