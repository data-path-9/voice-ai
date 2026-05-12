// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_authentication

import (
	"context"
	"fmt"

	internal_authentication_http "github.com/rapidaai/api/assistant-api/internal/authentication/http"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

// NewExecutor is the factory that returns an authentication executor implementation.
// Currently only HTTP is supported; switch on the assistant authentication type
// when other modes (e.g., JWT, static) are added.
func NewExecutor(
	logger commons.Logger,
	ctx context.Context,
	authenticator *internal_assistant_entity.AssistantAuthentication,
	callback internal_type.Callback,
	caller internal_type.InternalCaller,
) (internal_type.AuthenticationExecutor, error) {
	switch authenticator.Provider {
	case internal_assistant_entity.AssistantAuthenticationProviderHTTP:
		return internal_authentication_http.NewExecutor(logger, ctx, authenticator, callback, caller)
	default:
		return nil, fmt.Errorf("authentication: unsupported executor type %q", authenticator.Provider)
	}
}
