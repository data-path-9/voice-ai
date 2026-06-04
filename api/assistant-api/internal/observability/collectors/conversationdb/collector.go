// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package conversationdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/validator"
)

var (
	ErrLoggerRequired   = errors.New("conversationdb: logger is required")
	ErrPostgresRequired = errors.New("conversationdb: postgres is required")
	ErrScopeRequired    = errors.New("conversationdb: assistant_id and conversation_id are required")
)

type Config struct {
	Logger    commons.Logger
	Postgres  connectors.PostgresConnector
	Storage   storages.Storage
	CreatedBy uint64
}

type Collector struct {
	service   internal_services.AssistantConversationService
	createdBy uint64
}

func New(cfg Config) (observability.Collector, error) {
	if !validator.NonNil(cfg.Logger) {
		return nil, ErrLoggerRequired
	}
	if !validator.NonNil(cfg.Postgres) {
		return nil, ErrPostgresRequired
	}
	return &Collector{
		service: internal_assistant_service.NewAssistantConversationService(
			cfg.Logger,
			cfg.Postgres,
			cfg.Storage,
		),
		createdBy: cfg.CreatedBy,
	}, nil
}

func MustNew(cfg Config) observability.Collector {
	collector, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return collector
}

func (c *Collector) Collect(ctx context.Context, envelope observability.Envelope) error {
	switch envelope.Kind {
	case observability.RecordKindMetric:
		return c.collectMetrics(ctx, envelope)
	case observability.RecordKindMetadata:
		return c.collectMetadata(ctx, envelope)
	default:
		return nil
	}
}

func (c *Collector) Shutdown(context.Context) error {
	return nil
}

func (c *Collector) collectMetrics(ctx context.Context, envelope observability.Envelope) error {
	record, ok := envelope.Record.(observability.MetricRecord)
	if !ok || !validator.NotEmpty(record.Metrics) {
		return nil
	}
	if err := validateCollector(c); err != nil {
		return err
	}
	if err := validateScope(envelope.Scope); err != nil {
		return err
	}
	_, err := c.service.ApplyConversationMetrics(
		ctx,
		c.auth(envelope.Scope),
		envelope.Scope.AssistantID,
		envelope.Scope.ConversationID,
		toServiceMetrics(record.Metrics),
	)
	return err
}

func (c *Collector) collectMetadata(ctx context.Context, envelope observability.Envelope) error {
	record, ok := envelope.Record.(observability.MetadataRecord)
	if !ok || !validator.NotEmpty(record.Metadata) {
		return nil
	}
	if err := validateCollector(c); err != nil {
		return err
	}
	if err := validateScope(envelope.Scope); err != nil {
		return err
	}
	_, err := c.service.ApplyConversationMetadata(
		ctx,
		c.auth(envelope.Scope),
		envelope.Scope.AssistantID,
		envelope.Scope.ConversationID,
		toServiceMetadata(record.Metadata),
	)
	return err
}

func (c *Collector) auth(scope observability.Scope) types.SimplePrinciple {
	auth := &types.ServiceScope{}
	if c.createdBy > 0 {
		auth.UserId = &c.createdBy
	}
	if scope.ProjectID > 0 {
		auth.ProjectId = &scope.ProjectID
	}
	if scope.OrganizationID > 0 {
		auth.OrganizationId = &scope.OrganizationID
	}
	return auth
}

func toServiceMetrics(metrics []observability.Metric) []*types.Metric {
	converted := make([]*types.Metric, 0, len(metrics))
	for _, metric := range metrics {
		converted = append(converted, &types.Metric{
			Name:        metric.Name,
			Value:       metric.Value,
			Description: metric.Description,
		})
	}
	return converted
}

func toServiceMetadata(metadata []observability.Metadata) []*types.Metadata {
	converted := make([]*types.Metadata, 0, len(metadata))
	for _, item := range metadata {
		converted = append(converted, types.NewMetadata(item.Key, item.Value))
	}
	return converted
}

func validateCollector(collector *Collector) error {
	if !validator.NonNil(collector) || !validator.NonNil(collector.service) {
		return ErrPostgresRequired
	}
	return nil
}

func validateScope(scope observability.Scope) error {
	if scope.AssistantID == 0 || scope.ConversationID == 0 {
		return fmt.Errorf("%w: assistant_id=%d conversation_id=%d", ErrScopeRequired, scope.AssistantID, scope.ConversationID)
	}
	return nil
}
