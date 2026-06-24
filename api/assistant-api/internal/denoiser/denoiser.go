// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_denoiser

import (
	"context"

	internal_denoiser_krisp "github.com/rapidaai/api/assistant-api/internal/denoiser/internal/krisp"
	internal_denoiser_rnnoise "github.com/rapidaai/api/assistant-api/internal/denoiser/internal/rn_noise"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type DenoiserIdentifier string

const (
	RN_NOISE                   DenoiserIdentifier = "rn_noise"
	KRISP                      DenoiserIdentifier = "krisp"
	DenoiserOptionsKeyProvider                    = "microphone.denoising.provider"
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

func New(opts ...Option) (internal_type.VoiceDenoiserExecutor, error) {
	options := &options{ctx: context.Background()}
	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}

	provider, _ := options.options.GetString(DenoiserOptionsKeyProvider)
	switch DenoiserIdentifier(provider) {
	case KRISP:
		return internal_denoiser_krisp.New(
			internal_denoiser_krisp.WithContext(options.ctx),
			internal_denoiser_krisp.WithLogger(options.logger),
			internal_denoiser_krisp.WithOnPacket(options.onPacket),
			internal_denoiser_krisp.WithOptions(options.options),
		)
	default:
		return internal_denoiser_rnnoise.New(
			internal_denoiser_rnnoise.WithContext(options.ctx),
			internal_denoiser_rnnoise.WithLogger(options.logger),
			internal_denoiser_rnnoise.WithOnPacket(options.onPacket),
			internal_denoiser_rnnoise.WithOptions(options.options),
		)
	}
}
