// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"strconv"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observability/collectors"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (d *Dispatcher) runInboundCall(ctx context.Context, v CallReceivedPipeline) *PipelineResult {
	callInfo, err := d.inboundDispatcher.ReceiveCall(v.GinContext, v.Provider)
	if err != nil {
		_ = v.Observer.Record(ctx, observability.AssistantScope{AssistantID: v.AssistantID}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "inbound call receive failed",
			Attributes: observability.Attributes{
				"provider": v.Provider,
				"error":    err.Error(),
			},
		})
		return &PipelineResult{Error: err}
	}
	if callInfo == nil {
		_ = v.Observer.Record(ctx, observability.AssistantScope{AssistantID: v.AssistantID}, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "inbound call ignored",
			Attributes: observability.Attributes{
				"provider": v.Provider,
			},
		})
		return &PipelineResult{}
	}

	_ = v.Observer.Record(ctx, observability.AssistantScope{AssistantID: v.AssistantID}, observability.RecordEvent{
		Event: observability.CallReceived,
		Attributes: observability.Attributes{
			"provider": v.Provider,
			"caller":   callInfo.CallerNumber,
		},
	})

	assistant, err := d.assistantService.Get(ctx, v.Auth, v.AssistantID, utils.GetVersionDefinition("latest"), &internal_services.GetAssistantOption{InjectPhoneDeployment: true, InjectWebhook: true})
	if err != nil {
		_ = v.Observer.Record(ctx, observability.AssistantScope{AssistantID: v.AssistantID}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "inbound assistant load failed",
			Attributes: observability.Attributes{
				"provider":     v.Provider,
				"assistant_id": strconv.FormatUint(v.AssistantID, 10),
				"caller":       callInfo.CallerNumber,
				"error":        err.Error(),
			},
		})
		return &PipelineResult{Error: err}
	}

	// added collector may be duplicate but handled at collector level
	v.Observer.AddCollectors(collectors.NewWithAssistantWebhook(d.logger, assistant.AssistantWebhooks)...)
	_ = v.Observer.Record(ctx, observability.AssistantScope{AssistantID: assistant.Id}, observability.RecordEvent{
		Event: observability.CallAssistantLoaded,
		Attributes: observability.Attributes{
			"provider": v.Provider,
			"caller":   callInfo.CallerNumber,
		},
	})

	conversation, err := d.conversationService.CreateConversation(ctx, v.Auth, callInfo.CallerNumber, assistant.Id, assistant.AssistantProviderId, type_enums.DIRECTION_INBOUND, utils.PhoneCall)
	if err != nil {
		_ = v.Observer.Record(ctx, observability.AssistantScope{AssistantID: v.AssistantID}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "inbound conversation create failed",
			Attributes: observability.Attributes{
				"provider": v.Provider,
				"caller":   callInfo.CallerNumber,
				"error":    err.Error(),
			},
		})
		return &PipelineResult{Error: err}
	}

	_ = v.Observer.Record(ctx, observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
		ConversationID: conversation.Id,
	}, observability.RecordEvent{
		Event: observability.CallConversationCreated,
		Attributes: observability.Attributes{
			"provider": v.Provider,
			"caller":   callInfo.CallerNumber,
		},
	})

	contextID, err := d.inboundDispatcher.SaveCallContext(ctx, v.Auth, assistant, conversation.Id, callInfo, v.Provider)
	if err != nil {
		_ = v.Observer.Record(ctx, observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		}, observability.RecordLog{
			Level:   observability.LevelError,
			Message: "inbound call context save failed",
			Attributes: observability.Attributes{
				"provider": v.Provider,
				"caller":   callInfo.CallerNumber,
				"error":    err.Error(),
			},
		})
		return &PipelineResult{Error: err}
	}
	_ = v.Observer.Record(ctx, observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
		ConversationID: conversation.Id,
	}, observability.RecordEvent{
		Event: observability.CallContextSaved,
		Attributes: observability.Attributes{
			"provider":   v.Provider,
			"caller":     callInfo.CallerNumber,
			"context_id": contextID,
		},
	})

	if len(callInfo.Extra) > 0 {
		metadata := make([]*protos.Metadata, 0, len(callInfo.Extra))
		for key, value := range callInfo.Extra {
			metadata = append(metadata, &protos.Metadata{Key: key, Value: value})
		}
		_ = v.Observer.Record(ctx, observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		}, observability.RecordMetadata{
			Metadata: metadata,
		})
	}
	if callInfo.StatusInfo.Event != "" {
		_ = v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			}, observability.RecordEvent{
				Event: observability.CallStatus,
				Attributes: observability.Attributes{
					"provider":     v.Provider,
					"caller":       callInfo.CallerNumber,
					"status_event": callInfo.StatusInfo.Event,
				},
			})
	}

	v.GinContext.Set("contextId", contextID)
	if err := d.inboundDispatcher.AnswerProvider(v.GinContext, v.Auth, v.Provider, v.AssistantID, callInfo.CallerNumber, conversation.Id); err != nil {
		_ = v.Observer.Record(ctx,
			observability.ConversationScope{
				AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
				ConversationID: conversation.Id,
			},
			observability.RecordLog{
				Level:   observability.LevelError,
				Message: "inbound provider answer failed",
				Attributes: observability.Attributes{
					"provider":   v.Provider,
					"caller":     callInfo.CallerNumber,
					"context_id": contextID,
					"error":      err.Error(),
				},
			})
		return &PipelineResult{Error: err}
	}
	_ = v.Observer.Record(ctx,
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: v.AssistantID},
			ConversationID: conversation.Id,
		},
		observability.RecordEvent{
			Event: observability.CallProviderAnswered,
			Attributes: observability.Attributes{
				"provider":   v.Provider,
				"caller":     callInfo.CallerNumber,
				"context_id": contextID,
			},
		})

	return &PipelineResult{ContextID: contextID, ConversationID: conversation.Id}
}
