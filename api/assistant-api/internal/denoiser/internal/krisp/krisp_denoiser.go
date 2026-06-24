// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_denoiser_krisp

import (
	"context"
	"errors"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type krispDenoiser struct {
	logger   commons.Logger
	onPacket func(context.Context, ...internal_type.Packet) error
	options  utils.Option
}

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
	if options.onPacket == nil {
		return nil, errors.New("krisp-denoiser: onPacket is required")
	}
	return &krispDenoiser{logger: options.logger, onPacket: options.onPacket, options: options.options}, nil
}

func (krisp *krispDenoiser) Name() string {
	return "krisp-denoiser"
}
func (krisp *krispDenoiser) Options() utils.Option {
	krisp.logger.Warn("Krisp denoiser does not support any options yet")
	return krisp.options
}
func (krisp *krispDenoiser) Arguments() (map[string]string, error) {
	return map[string]string{}, nil
}

func (krisp *krispDenoiser) Execute(ctx context.Context, pkt internal_type.DenoiseAudioPacket) error {
	panic("not yet implimented")
}

func (krisp *krispDenoiser) Close(ctx context.Context) error {
	panic("not yet implimented")
}
