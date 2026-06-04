// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package billing

import (
	"context"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/validator"
)

type Usage struct {
	ID            string
	Scope         observability.Scope
	Component     string
	Provider      string
	UsageCategory string
	Duration      time.Duration
	Attributes    observability.Attributes
	OccurredAt    time.Time
	ReceivedAt    time.Time
}

type Publisher interface {
	PublishUsage(ctx context.Context, usage Usage) error
}

type Collector struct {
	publisher Publisher
}

func New(publisher Publisher) observability.Collector {
	if !validator.NonNil(publisher) {
		return observability.NoopCollector{}
	}
	return &Collector{publisher: publisher}
}

func (c *Collector) Collect(ctx context.Context, envelope observability.Envelope) error {
	if envelope.Name != observability.UsageRecorded {
		return nil
	}
	record, ok := envelope.Record.(observability.UsageEvent)
	if !ok {
		return nil
	}
	return c.publisher.PublishUsage(ctx, Usage{
		ID:            envelope.ID,
		Scope:         envelope.Scope,
		Component:     record.Component,
		Provider:      record.Provider,
		UsageCategory: record.UsageCategory,
		Duration:      record.Duration,
		Attributes:    envelope.Attributes.Clone(),
		OccurredAt:    envelope.OccurredAt,
		ReceivedAt:    envelope.ReceivedAt,
	})
}

func (c *Collector) Shutdown(context.Context) error {
	return nil
}
