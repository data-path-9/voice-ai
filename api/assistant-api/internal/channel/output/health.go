// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_channel_output

import "sync/atomic"

// TickHealth captures per-tick pacing health.
type TickHealth struct {
	LateTick  bool
	Active    bool
	Idle      bool
	SendError bool
}

// HealthObserver receives per-tick pacing health events.
type HealthObserver interface {
	OnTickHealth(event TickHealth)
}

// HealthStats accumulates output pacing quality counters.
type HealthStats struct {
	ticks       atomic.Uint64
	lateTicks   atomic.Uint64
	activeTicks atomic.Uint64
	idleTicks   atomic.Uint64
	sendErrors  atomic.Uint64
}

// HealthSnapshot is a point-in-time view of pacing quality counters.
type HealthSnapshot struct {
	Ticks       uint64  `json:"ticks"`
	LateTicks   uint64  `json:"late_ticks"`
	ActiveTicks uint64  `json:"active_ticks"`
	IdleTicks   uint64  `json:"idle_ticks"`
	SendErrors  uint64  `json:"send_errors"`
	IdleRatio   float64 `json:"idle_ratio"`
}

func NewHealthStats() *HealthStats {
	return &HealthStats{}
}

func (s *HealthStats) OnTickHealth(event TickHealth) {
	if s == nil {
		return
	}
	s.ticks.Add(1)
	if event.LateTick {
		s.lateTicks.Add(1)
	}
	if event.Active {
		s.activeTicks.Add(1)
	}
	if event.Idle {
		s.idleTicks.Add(1)
	}
	if event.SendError {
		s.sendErrors.Add(1)
	}
}

func (s *HealthStats) Snapshot() HealthSnapshot {
	if s == nil {
		return HealthSnapshot{}
	}
	ticks := s.ticks.Load()
	idleTicks := s.idleTicks.Load()
	idleRatio := 0.0
	if ticks > 0 {
		idleRatio = float64(idleTicks) / float64(ticks)
	}
	return HealthSnapshot{
		Ticks:       ticks,
		LateTicks:   s.lateTicks.Load(),
		ActiveTicks: s.activeTicks.Load(),
		IdleTicks:   idleTicks,
		SendErrors:  s.sendErrors.Load(),
		IdleRatio:   idleRatio,
	}
}
