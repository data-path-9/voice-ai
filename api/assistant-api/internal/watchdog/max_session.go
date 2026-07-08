// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type MaxSessionEvent struct {
	ContextID string
	Deadline  time.Time
	Duration  time.Duration
}

type MaxSessionOptions = WatchdogOptions
type MaxSessionOption = Option

type MaxSessionWatchdog struct {
	mu      sync.Mutex
	options MaxSessionOptions

	timer *time.Timer

	generation uint64
	active     bool
	contextID  string
	deadline   time.Time
	duration   time.Duration
}

func NewMaxSessionWatchdog(opts ...MaxSessionOption) *MaxSessionWatchdog {
	options := MaxSessionOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.applyWatchdogOptions(&options)
	}

	if options.PacketContext == nil {
		options.PacketContext = context.Background()
	}
	if options.RecordScope == "" {
		options.RecordScope = internal_type.ObservabilityRecordScopeConversation
	}

	watchdog := &MaxSessionWatchdog{
		options: options,
	}

	if options.OnPacket != nil {
		_ = options.OnPacket(
			options.PacketContext,
			internal_type.ObservabilityLogRecordPacket{
				Scope: options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "max-session-watchdog: initialization completed",
					Attributes: observability.Attributes{
						"component": observability.ComponentConversation.String(),
						"watchdog":  "max_session",
					},
					OccurredAt: time.Now(),
				},
			},
		)
	}

	return watchdog
}

func (w *MaxSessionWatchdog) Start(contextID string, duration time.Duration) bool {
	if duration <= 0 {
		return false
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.active = true
	w.contextID = contextID
	w.duration = duration
	w.deadline = time.Now().Add(duration)

	generation := w.generation
	w.timer = time.AfterFunc(duration, func() {
		w.expire(generation)
	})

	return true
}

func (w *MaxSessionWatchdog) Cancel() bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	wasActive := w.active
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	w.generation++
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}
	w.duration = 0

	return wasActive
}

func (w *MaxSessionWatchdog) expire(generation uint64) {
	w.mu.Lock()
	if !w.active || w.generation != generation {
		w.mu.Unlock()
		return
	}

	event := MaxSessionEvent{
		ContextID: w.contextID,
		Deadline:  w.deadline,
		Duration:  w.duration,
	}
	w.timer = nil
	w.generation++
	w.active = false
	w.contextID = ""
	w.deadline = time.Time{}
	w.duration = 0
	w.mu.Unlock()

	if w.options.OnPacket != nil {
		_ = w.options.OnPacket(
			w.options.PacketContext,
			internal_type.ObservabilityLogRecordPacket{
				ContextID: event.ContextID,
				Scope:     w.options.RecordScope,
				Record: observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "max-session-watchdog: deadline expired",
					Attributes: observability.Attributes{
						"component":   observability.ComponentConversation.String(),
						"watchdog":    "max_session",
						"duration_ms": observability.AttributeValue(event.Duration.Milliseconds()),
					},
					OccurredAt: time.Now(),
				},
			},
			internal_type.MaxSessionExpiredPacket{ContextID: event.ContextID},
		)
	}
}
