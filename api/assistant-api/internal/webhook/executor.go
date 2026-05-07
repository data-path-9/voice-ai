// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_webhook

import (
	"context"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_webhook_http "github.com/rapidaai/api/assistant-api/internal/webhook/http"
	"github.com/rapidaai/pkg/commons"
)

// NewExecutor is the factory that returns a webhook executor implementation.
// Currently only HTTP is supported; switch on the webhook artifact type when
// other transports (e.g., gRPC, queue) are added.
func NewExecutor(logger commons.Logger, ctx context.Context, callback internal_type.Callback, caller internal_type.InternalCaller) (internal_type.WebhookExecutor, error) {
	return internal_webhook_http.NewExecutor(logger, ctx, callback, caller)
}
