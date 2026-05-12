// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

type AssistantWebhookEvent string

const (
	//
	ConversationBegin     AssistantWebhookEvent = "conversation.begin"
	ConversationResume    AssistantWebhookEvent = "conversation.resume"
	ConversationCompleted AssistantWebhookEvent = "conversation.completed"
	// Triggered when a conversation ends successfully.

	ConversationFailed AssistantWebhookEvent = "conversation.failed"
	// Triggered when a conversation encounters an error.

)

func (r AssistantWebhookEvent) Get() string {
	return string(r)
}
