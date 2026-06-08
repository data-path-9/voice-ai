// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"fmt"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (d *Dispatcher) runOutbound(ctx context.Context, v OutboundRequestedPipeline) *PipelineResult {
	d.logger.Infow("Pipeline: OutboundRequested",
		"to", v.ToPhone,
		"from", v.FromPhone,
		"assistant_id", v.AssistantID)

	assistantScope := observability.AssistantScope{AssistantID: v.AssistantID}
	assistant, err := d.assistantService.Get(ctx, v.Auth, v.AssistantID, utils.GetVersionDefinition("latest"), &internal_services.GetAssistantOption{InjectPhoneDeployment: true, InjectWebhook: true})
	if err != nil {
		_ = v.Observer.Record(ctx, assistantScope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "outbound assistant load failed",
			Attributes: observability.Attributes{
				"assistant_id": fmt.Sprintf("%d", v.AssistantID),
				"to":           v.ToPhone,
				"from":         v.FromPhone,
				"error":        err.Error(),
			},
		})
		return &PipelineResult{Error: fmt.Errorf("invalid assistant: %w", err)}
	}
	assistantScope.AssistantID = assistant.Id
	if assistant.AssistantPhoneDeployment == nil {
		_ = v.Observer.Record(ctx, assistantScope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "outbound phone deployment not enabled",
			Attributes: observability.Attributes{
				"assistant_id": fmt.Sprintf("%d", assistant.Id),
				"to":           v.ToPhone,
				"from":         v.FromPhone,
			},
		})
		return &PipelineResult{Error: fmt.Errorf("phone deployment not enabled")}
	}

	fromPhone := v.FromPhone
	if fromPhone == "" {
		fn, err := assistant.AssistantPhoneDeployment.GetOptions().GetString("phone")
		if err != nil {
			_ = v.Observer.Record(ctx, assistantScope, observability.RecordLog{
				Level:   observability.LevelError,
				Message: "outbound from phone resolution failed",
				Attributes: observability.Attributes{
					"assistant_id": fmt.Sprintf("%d", assistant.Id),
					"to":           v.ToPhone,
					"error":        err.Error(),
				},
			})
			return &PipelineResult{Error: fmt.Errorf("no phone number configured: %w", err)}
		}
		fromPhone = fn
	}
	provider := assistant.AssistantPhoneDeployment.TelephonyProvider
	conversation, err := d.conversationService.CreateConversation(ctx, v.Auth, v.ToPhone, assistant.Id, assistant.AssistantProviderId, type_enums.DIRECTION_OUTBOUND, utils.PhoneCall)
	if err != nil {
		_ = v.Observer.Record(ctx, assistantScope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "outbound conversation create failed",
			Attributes: observability.Attributes{
				"assistant_id": fmt.Sprintf("%d", assistant.Id),
				"provider":     provider,
				"to":           v.ToPhone,
				"from":         fromPhone,
				"error":        err.Error(),
			},
		})
		return &PipelineResult{Error: fmt.Errorf("failed to create conversation: %w", err)}
	}

	conversationScope := observability.ConversationScope{
		AssistantScope: assistantScope,
		ConversationID: conversation.Id,
	}
	_ = v.Observer.Record(ctx, conversationScope, observability.RecordEvent{
		Event: observability.CallConversationCreated,
		Attributes: observability.Attributes{
			"provider": provider,
			"to":       v.ToPhone,
			"from":     fromPhone,
		},
	})

	if len(v.Options) > 0 {
		if _, err := d.conversationService.CreateOrUpdateConversationOption(ctx, v.Auth, assistant.Id, conversation.Id, v.Options); err != nil {
			d.logger.Warnw("Failed to CreateOrUpdate conversation extras", "error", err)
		}
	}
	if len(v.Args) > 0 {
		if _, err := d.conversationService.CreateOrUpdateConversationArgument(ctx, v.Auth, assistant.Id, conversation.Id, v.Args); err != nil {
			d.logger.Warnw("Failed to CreateOrUpdate conversation extras", "error", err)
		}
	}
	if len(v.Metadata) > 0 {
		conversationMetadata := make([]*protos.Metadata, 0, len(v.Metadata))
		for key, value := range v.Metadata {
			conversationMetadata = append(conversationMetadata, &protos.Metadata{Key: key, Value: fmt.Sprintf("%v", value)})
		}
		_ = v.Observer.Record(ctx, conversationScope, observability.RecordMetadata{
			Metadata: conversationMetadata,
		})
	}
	callInfo := &internal_type.CallInfo{CallerNumber: v.ToPhone, FromNumber: fromPhone, Direction: "outbound", Provider: provider, Status: "queued"}
	contextID, err := d.inboundDispatcher.SaveCallContext(ctx, v.Auth, assistant, conversation.Id, callInfo, provider)
	if err != nil {
		_ = v.Observer.Record(ctx, conversationScope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "outbound call context save failed",
			Attributes: observability.Attributes{
				"provider": provider,
				"to":       v.ToPhone,
				"from":     fromPhone,
				"error":    err.Error(),
			},
		})
		return &PipelineResult{Error: fmt.Errorf("failed to save call context: %w", err)}
	}
	_ = v.Observer.Record(ctx, conversationScope, observability.RecordEvent{
		Event: observability.CallContextSaved,
		Attributes: observability.Attributes{
			"provider":   provider,
			"to":         v.ToPhone,
			"from":       fromPhone,
			"context_id": contextID,
		},
	})
	_ = v.Observer.Record(ctx, conversationScope, observability.RecordMetadata{
		Metadata: observability.ClientMetadata(
			v.ToPhone, fromPhone, "outbound", provider,
			"", contextID, "", "",
		),
	})
	_ = v.Observer.Record(ctx, conversationScope, observability.RecordEvent{
		Event: observability.CallOutboundRequested,
		Attributes: observability.Attributes{
			"provider":   provider,
			"to":         v.ToPhone,
			"from":       fromPhone,
			"context_id": contextID,
		},
	})

	if err := d.outboundDispatcher.Dispatch(ctx, contextID); err != nil {
		d.logger.Error("Pipeline: outbound dispatch failed", "error", err)
		_ = v.Observer.Record(ctx, conversationScope, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "outbound dispatch failed",
			Attributes: observability.Attributes{
				"provider":   provider,
				"to":         v.ToPhone,
				"from":       fromPhone,
				"context_id": contextID,
				"error":      err.Error(),
			},
		})
		_ = v.Observer.Record(ctx, conversationScope, observability.RecordEvent{
			Event: observability.CallOutboundDispatchFailed,
			Attributes: observability.Attributes{
				"provider":   provider,
				"context_id": contextID,
				"error":      err.Error(),
			},
		})
		_ = v.Observer.Record(ctx, conversationScope, observability.RecordMetric{
			Metrics: observability.CallStatusMetric("FAILED", err.Error()),
		})
		return &PipelineResult{ContextID: contextID, ConversationID: conversation.Id, Error: err}
	}

	_ = v.Observer.Record(ctx, conversationScope, observability.RecordEvent{
		Event: observability.CallOutboundDispatched,
		Attributes: observability.Attributes{
			"provider":   provider,
			"context_id": contextID,
		},
	})

	return &PipelineResult{ContextID: contextID, ConversationID: conversation.Id}
}
