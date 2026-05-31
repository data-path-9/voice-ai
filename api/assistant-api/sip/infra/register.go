// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"

	"github.com/emiago/sipgo"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/commons"
)

var (
	ErrMissingDID          = internal_core.ErrMissingDID
	ErrMissingServer       = internal_core.ErrMissingServer
	ErrRegistrationFailed  = internal_core.ErrRegistrationFailed
	ErrRegistrationExpired = internal_core.ErrRegistrationExpired
	ErrDIDNotRegistered    = internal_core.ErrDIDNotRegistered
	ErrAuthFailed          = internal_core.ErrAuthFailed
	ErrPermanentFailure    = internal_core.ErrPermanentFailure
)

type Registration struct {
	DID         string
	Config      *Config
	AssistantID uint64
	ExpiresIn   int
}

func (r *Registration) Validate() error {
	return r.toCore().Validate()
}

func (r *Registration) toCore() *internal_core.Registration {
	if r == nil {
		return nil
	}
	return &internal_core.Registration{
		DID:         r.DID,
		Config:      r.Config.toCore(),
		AssistantID: r.AssistantID,
		ExpiresIn:   r.ExpiresIn,
	}
}

type RegistrationClient struct {
	inner *internal_core.RegistrationClient
}

func NewRegistrationClient(client *sipgo.Client, listenConfig *ListenConfig, logger commons.Logger) *RegistrationClient {
	return &RegistrationClient{
		inner: internal_core.NewRegistrationClient(client, listenConfig.toCore(), logger),
	}
}

func (rc *RegistrationClient) Register(ctx context.Context, reg *Registration) error {
	return rc.inner.Register(ctx, reg.toCore())
}

func (rc *RegistrationClient) Unregister(ctx context.Context, did string) error {
	return rc.inner.Unregister(ctx, did)
}

func (rc *RegistrationClient) UnregisterAll(ctx context.Context) {
	rc.inner.UnregisterAll(ctx)
}

func (rc *RegistrationClient) IsRegistered(did string) bool {
	return rc.inner.IsRegistered(did)
}

func (rc *RegistrationClient) ActiveCount() int {
	return rc.inner.ActiveCount()
}

func (rc *RegistrationClient) GetRegisteredDIDs() []string {
	return rc.inner.GetRegisteredDIDs()
}
