// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package channel

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

// Envelope carries a packet together with the context it was sent from.
type Envelope struct {
	Ctx context.Context
	Pkt internal_type.Packet
}

type ChannelWriter interface {
	OnControl(Envelope)
	OnBootstrap(Envelope)
	OnIngress(Envelope)
	OnEgress(Envelope)
	OnData(Envelope)
	OnBackground(Envelope)
}

type ChannelReader interface {
	ControlChannel() chan Envelope
	BootstrapChannel() chan Envelope
	IngressChannel() chan Envelope
	EgressChannel() chan Envelope
	DataChannel() chan Envelope
	BackgroundChannel() chan Envelope
}

type ChannelFlusher interface {
	FlushControl() int
	FlushBootstrap() int
	FlushIngress() int
	FlushEgress() int
	FlushData() int
	FlushBackground() int
	FlushAll() int
}

type ChannelRunner interface {
	RunControl(context.Context, func(Envelope))
	RunBootstrap(context.Context, func(Envelope))
	RunIngress(context.Context, func(Envelope))
	RunEgress(context.Context, func(Envelope))
	RunData(context.Context, func(Envelope))
	RunBackground(context.Context, func(Envelope))
}

// RequestorChannelBus is the unified channel interface used by the requestor.
type RequestorChannelBus interface {
	ChannelWriter
	ChannelReader
	ChannelFlusher
	ChannelRunner
}

// RequestorChannels groups all dispatcher channels used by one requestor.
type RequestorChannels struct {
	// controlChannel is for urgent runtime control packets:
	// interruptions, turn-change, and other immediate control directives.
	controlChannel chan Envelope

	// bootstrapCh is reserved for session initialization/bootstrap packets.
	// Use this channel only for connect-time setup flow.
	bootstrapCh chan Envelope

	// ingressCh carries inbound user-side packets:
	// user audio/text and upstream processing packets (VAD/STT/EOS/tool result).
	ingressCh chan Envelope

	// egressCh carries outbound assistant-side packets:
	// LLM deltas/done, TTS text/audio/end, and output error/control events.
	egressCh chan Envelope

	// dataCh carries DB writes, recording, and lifecycle orchestration that does
	// not require the observer. Drained by the data dispatcher started at
	// NewGenericRequestor and runs for the entire session.
	dataCh chan Envelope

	// backgroundCh is for observer-touching telemetry (events, metrics).
	// Drained by the dispatcher started after telemetry init completes.
	backgroundCh chan Envelope
}

func NewRequestorChannels() *RequestorChannels {
	return &RequestorChannels{
		controlChannel: make(chan Envelope, 256),
		bootstrapCh:    make(chan Envelope, 512),
		ingressCh:      make(chan Envelope, 4096),
		egressCh:       make(chan Envelope, 2048),
		dataCh:         make(chan Envelope, 2048),
		backgroundCh:   make(chan Envelope, 2048),
	}
}

func (c *RequestorChannels) ControlChannel() chan Envelope    { return c.controlChannel }
func (c *RequestorChannels) BootstrapChannel() chan Envelope  { return c.bootstrapCh }
func (c *RequestorChannels) IngressChannel() chan Envelope    { return c.ingressCh }
func (c *RequestorChannels) EgressChannel() chan Envelope     { return c.egressCh }
func (c *RequestorChannels) DataChannel() chan Envelope       { return c.dataCh }
func (c *RequestorChannels) BackgroundChannel() chan Envelope { return c.backgroundCh }

// OnControl routes an envelope to the control channel.
// Keep enqueue policy in this layer (block/drop/timeout) so it can evolve
// without touching dispatch routing code.
func (c *RequestorChannels) OnControl(e Envelope) {
	c.controlChannel <- e
}

// OnBootstrap routes an envelope to the bootstrap channel.
func (c *RequestorChannels) OnBootstrap(e Envelope) {
	c.bootstrapCh <- e
}

// OnIngress routes an envelope to the ingress channel.
func (c *RequestorChannels) OnIngress(e Envelope) {
	c.ingressCh <- e
}

// OnEgress routes an envelope to the egress channel.
func (c *RequestorChannels) OnEgress(e Envelope) {
	c.egressCh <- e
}

// OnData routes an envelope to the data channel (DB writes, recording, lifecycle).
func (c *RequestorChannels) OnData(e Envelope) {
	c.dataCh <- e
}

// OnBackground routes an envelope to the background channel.
func (c *RequestorChannels) OnBackground(e Envelope) {
	c.backgroundCh <- e
}

func run(ctx context.Context, ch <-chan Envelope, onEnvelope func(Envelope)) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-ch:
			onEnvelope(e)
		}
	}
}

func (c *RequestorChannels) RunControl(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.controlChannel, onEnvelope)
}

func (c *RequestorChannels) RunBootstrap(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.bootstrapCh, onEnvelope)
}

func (c *RequestorChannels) RunIngress(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.ingressCh, onEnvelope)
}

func (c *RequestorChannels) RunEgress(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.egressCh, onEnvelope)
}

func (c *RequestorChannels) RunData(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.dataCh, onEnvelope)
}

func (c *RequestorChannels) RunBackground(ctx context.Context, onEnvelope func(Envelope)) {
	run(ctx, c.backgroundCh, onEnvelope)
}

func flushChannel(ch chan Envelope) int {
	dropped := 0
	for {
		select {
		case <-ch:
			dropped++
		default:
			return dropped
		}
	}
}

// FlushControl drains queued control packets and returns dropped count.
func (c *RequestorChannels) FlushControl() int {
	return flushChannel(c.controlChannel)
}

// FlushBootstrap drains queued bootstrap packets and returns dropped count.
func (c *RequestorChannels) FlushBootstrap() int {
	return flushChannel(c.bootstrapCh)
}

// FlushIngress drains queued ingress packets and returns dropped count.
func (c *RequestorChannels) FlushIngress() int {
	return flushChannel(c.ingressCh)
}

// FlushEgress drains queued egress packets and returns dropped count.
func (c *RequestorChannels) FlushEgress() int {
	return flushChannel(c.egressCh)
}

// FlushData drains queued data packets and returns dropped count.
func (c *RequestorChannels) FlushData() int {
	return flushChannel(c.dataCh)
}

// FlushBackground drains queued background packets and returns dropped count.
func (c *RequestorChannels) FlushBackground() int {
	return flushChannel(c.backgroundCh)
}

// FlushAll drains all channels and returns total dropped packets.
func (c *RequestorChannels) FlushAll() int {
	return c.FlushControl() +
		c.FlushBootstrap() +
		c.FlushIngress() +
		c.FlushEgress() +
		c.FlushData() +
		c.FlushBackground()
}
