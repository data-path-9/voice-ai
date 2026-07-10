// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package watchdog

import (
	"context"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

type PacketOptions struct {
	OnPacket      func(context.Context, ...internal_type.Packet) error
	PacketContext context.Context
	RecordScope   internal_type.ObservabilityRecordScope
}

type WatchdogOptions struct {
	PacketOptions

	WordsPerMinute int
	MinimumTimeout time.Duration
	GracePeriod    time.Duration
}

type Option interface {
	applyWatchdogOptions(*WatchdogOptions)
}

type onPacketOption struct {
	onPacket func(context.Context, ...internal_type.Packet) error
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return onPacketOption{onPacket: onPacket}
}

func (option onPacketOption) applyWatchdogOptions(options *WatchdogOptions) {
	options.OnPacket = option.onPacket
}

type packetContextOption struct {
	ctx context.Context
}

func WithPacketContext(ctx context.Context) Option {
	return packetContextOption{ctx: ctx}
}

func (option packetContextOption) applyWatchdogOptions(options *WatchdogOptions) {
	options.PacketContext = option.ctx
}

type recordScopeOption struct {
	scope internal_type.ObservabilityRecordScope
}

func WithRecordScope(scope internal_type.ObservabilityRecordScope) Option {
	return recordScopeOption{scope: scope}
}

func (option recordScopeOption) applyWatchdogOptions(options *WatchdogOptions) {
	options.RecordScope = option.scope
}
