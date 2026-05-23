// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_webhook

import (
	"context"
	"fmt"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_webhook_http "github.com/rapidaai/api/assistant-api/internal/webhook/http"
	"github.com/rapidaai/pkg/commons"
)

// NewExecutor is the factory that returns a webhook executor implementation.
// Currently only HTTP is supported; switch on the webhook artifact type when
// other transports (e.g., gRPC, queue) are added.
func NewExecutor(
	logger commons.Logger,
	ctx context.Context,
	webhook *internal_assistant_entity.AssistantWebhook,
	callback internal_type.Callback,
	caller internal_type.InternalCaller,
) (internal_type.WebhookExecutor, error) {
	switch webhook.Provider {
	case internal_assistant_entity.AssistantWebhookProviderHTTP:
		return internal_webhook_http.NewExecutor(logger, ctx, webhook, callback, caller)
	default:
		return nil, fmt.Errorf("webhook: unsupported executor type %q", webhook.Provider)
	}
}
