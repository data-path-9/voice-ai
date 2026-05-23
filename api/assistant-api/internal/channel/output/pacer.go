// Copyright (c) 2023-2026 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_channel_output

import (
	"context"
	"time"

	"github.com/rapidaai/pkg/commons"
)

// FrameProvider provides outbound frames for paced delivery.
// NextFrame returns the next media frame, or nil when no media is buffered.
// IdleFrame returns a frame for idle ticks (ambient/silence).
type FrameProvider interface {
	NextFrame() []byte
	IdleFrame() []byte
}

// FrameConsumer writes a frame to provider transport.
type FrameConsumer interface {
	ConsumeFrame(frame []byte) error
}

// Pacer drives fixed-interval media delivery for channel transports.
type Pacer struct {
	Logger        commons.Logger
	FrameDuration time.Duration
	Provider      FrameProvider
	Consumer      FrameConsumer
	Health        HealthObserver
}

func (p *Pacer) Run(ctx context.Context) {
	if p.Provider == nil || p.Consumer == nil {
		if p.Logger != nil {
			p.Logger.Error("channel pacer requires provider and consumer")
		}
		return
	}
	frameDuration := p.FrameDuration
	if frameDuration <= 0 {
		frameDuration = 20 * time.Millisecond
	}
	lateTickTolerance := 2 * time.Millisecond

	nextSendTime := time.Now().Add(frameDuration)
	timer := time.NewTimer(frameDuration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		now := time.Now()
		tick := TickHealth{
			LateTick: now.After(nextSendTime.Add(lateTickTolerance)),
		}
		frame := p.Provider.NextFrame()
		if len(frame) == 0 {
			frame = p.Provider.IdleFrame()
			tick.Idle = true
		}
		if len(frame) > 0 {
			tick.Active = !tick.Idle
			if err := p.Consumer.ConsumeFrame(frame); err != nil {
				tick.SendError = true
				if p.Logger != nil {
					p.Logger.Debug("Failed to send audio frame", "error", err)
				}
			}
		} else {
			tick.Idle = true
		}
		if p.Health != nil {
			p.Health.OnTickHealth(tick)
		}

		nextSendTime = nextSendTime.Add(frameDuration)
		now = time.Now()
		if now.After(nextSendTime) {
			nextSendTime = now.Add(frameDuration)
		}
		nextWait := time.Until(nextSendTime)
		if nextWait < 0 {
			nextWait = 0
		}
		timer.Reset(nextWait)
	}
}
