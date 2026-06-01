// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"

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

type RegistrationClient struct {
	inner *internal_core.RegistrationClient
}
