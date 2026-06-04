// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability

import (
	"context"
	"errors"
)

// Collector consumes normalized observability envelopes.
type Collector interface {
	Collect(ctx context.Context, envelope Envelope) error
	Shutdown(ctx context.Context) error
}

// CollectorFunc adapts a function into a Collector.
type CollectorFunc func(ctx context.Context, envelope Envelope) error

func (f CollectorFunc) Collect(ctx context.Context, envelope Envelope) error {
	return f(ctx, envelope)
}

func (CollectorFunc) Shutdown(context.Context) error {
	return nil
}

// Collectors fans out envelopes to multiple collectors.
type Collectors struct {
	collectors []Collector
}

func NewCollectors(collectors ...Collector) Collector {
	if len(collectors) == 0 {
		return NoopCollector{}
	}
	return &Collectors{collectors: append([]Collector(nil), collectors...)}
}

func (c *Collectors) Collect(ctx context.Context, envelope Envelope) error {
	var errs []error
	for _, collector := range c.collectors {
		if collector == nil {
			continue
		}
		if err := collector.Collect(ctx, envelope); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c *Collectors) Shutdown(ctx context.Context) error {
	var errs []error
	for _, collector := range c.collectors {
		if collector == nil {
			continue
		}
		if err := collector.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type NoopCollector struct{}

func (NoopCollector) Collect(context.Context, Envelope) error {
	return nil
}

func (NoopCollector) Shutdown(context.Context) error {
	return nil
}
