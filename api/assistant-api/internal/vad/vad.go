// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_vad

import (
	"context"
	"errors"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_vad_firered "github.com/rapidaai/api/assistant-api/internal/vad/internal/firered_vad"
	internal_vad_silero "github.com/rapidaai/api/assistant-api/internal/vad/internal/silero_vad"
	internal_vad_ten "github.com/rapidaai/api/assistant-api/internal/vad/internal/ten_vad"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type VADIdentifier string

const (
	SILERO_VAD            VADIdentifier = "silero_vad"
	TEN_VAD               VADIdentifier = "ten_vad"
	FIRERED_VAD           VADIdentifier = "firered_vad"
	OptionsKeyVadProvider               = "microphone.vad.provider"
)

type options struct {
	ctx      context.Context
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	options  utils.Option
}

type Option func(*options)

func WithContext(ctx context.Context) Option {
	return func(options *options) {
		options.ctx = ctx
	}
}

func WithLogger(logger commons.Logger) Option {
	return func(options *options) {
		options.logger = logger
	}
}

func WithOnPacket(onPacket func(context.Context, ...internal_type.Packet) error) Option {
	return func(options *options) {
		options.onPacket = onPacket
	}
}

func WithOptions(opts utils.Option) Option {
	return func(options *options) {
		options.options = opts
	}
}

// New creates a VAD executor based on the provider option.
// Input audio is always 16 kHz LINEAR16 mono (platform internal format).
func New(opts ...Option) (internal_type.VoiceActivityDetectorExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	if options.onPacket == nil {
		return nil, errors.New("vad: onPacket is required")
	}
	typ, _ := options.options.GetString(OptionsKeyVadProvider)
	switch VADIdentifier(typ) {
	case FIRERED_VAD:
		return internal_vad_firered.New(
			internal_vad_firered.WithContext(options.ctx),
			internal_vad_firered.WithLogger(options.logger),
			internal_vad_firered.WithOnPacket(options.onPacket),
			internal_vad_firered.WithOptions(options.options),
		)
	case TEN_VAD:
		return internal_vad_ten.New(
			internal_vad_ten.WithContext(options.ctx),
			internal_vad_ten.WithLogger(options.logger),
			internal_vad_ten.WithOnPacket(options.onPacket),
			internal_vad_ten.WithOptions(options.options),
		)
	case SILERO_VAD:
		return internal_vad_silero.New(
			internal_vad_silero.WithContext(options.ctx),
			internal_vad_silero.WithLogger(options.logger),
			internal_vad_silero.WithOnPacket(options.onPacket),
			internal_vad_silero.WithOptions(options.options),
		)
	default:
		return internal_vad_silero.New(
			internal_vad_silero.WithContext(options.ctx),
			internal_vad_silero.WithLogger(options.logger),
			internal_vad_silero.WithOnPacket(options.onPacket),
			internal_vad_silero.WithOptions(options.options),
		)
	}
}
