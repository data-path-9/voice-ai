// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package notification

import (
	"context"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/validator"
)

type Notification struct {
	ID         string
	Event      observability.EventName
	Category   observability.Category
	Level      observability.Level
	Outcome    observability.Outcome
	Title      string
	Scope      observability.Scope
	Attributes observability.Attributes
	Data       observability.Data
	OccurredAt time.Time
}

type Notifier interface {
	Notify(ctx context.Context, notification Notification) error
}

type Selector func(envelope observability.Envelope) bool

type Config struct {
	Notifier Notifier
	Selector Selector
}

type Collector struct {
	notifier Notifier
	selector Selector
}

func New(cfg Config) observability.Collector {
	if !validator.NonNil(cfg.Notifier) {
		return observability.NoopCollector{}
	}
	selector := cfg.Selector
	if !validator.NonNil(selector) {
		selector = DefaultSelector
	}
	return &Collector{notifier: cfg.Notifier, selector: selector}
}

func (c *Collector) Collect(ctx context.Context, envelope observability.Envelope) error {
	if !c.selector(envelope) {
		return nil
	}
	return c.notifier.Notify(ctx, Notification{
		ID:         envelope.ID,
		Event:      envelope.Name,
		Category:   envelope.Category,
		Level:      envelope.Level,
		Outcome:    envelope.Outcome,
		Title:      envelope.Title,
		Scope:      envelope.Scope,
		Attributes: envelope.Attributes.Clone(),
		Data:       envelope.Data.Clone(),
		OccurredAt: envelope.OccurredAt,
	})
}

func (c *Collector) Shutdown(context.Context) error {
	return nil
}

func DefaultSelector(envelope observability.Envelope) bool {
	if envelope.Kind != observability.RecordKindEvent {
		return false
	}
	if envelope.Level == observability.LevelError || envelope.Level == observability.LevelFatal {
		return true
	}
	if envelope.Outcome == observability.OutcomeFailure {
		return true
	}
	switch envelope.Name {
	case observability.CallFailed,
		observability.ConversationFailed,
		observability.ConversationError,
		observability.WebhookFailed,
		observability.ErrorRaised:
		return true
	default:
		return false
	}
}
