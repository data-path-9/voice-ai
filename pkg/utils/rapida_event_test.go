// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package utils

import "testing"

func TestAssistantWebhookEvent_Get(t *testing.T) {
	tests := []struct {
		event    AssistantWebhookEvent
		expected string
	}{
		{ConversationBegin, "conversation.begin"},
		{ConversationResume, "conversation.resume"},
		{ConversationCompleted, "conversation.completed"},
		{ConversationFailed, "conversation.failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if result := tt.event.Get(); result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
