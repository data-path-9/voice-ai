// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

func (cApi *ConversationApi) UnviersalCallback(c *gin.Context) {
	provider := c.Param("telephony")
	if !validator.NotBlank(provider) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing telephony provider"})
		return
	}
	assistantID, err := strconv.ParseUint(c.Param("assistantId"), 10, 64)
	if err != nil || !validator.AllNonZero(assistantID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assistantId"})
		return
	}

	statusInfo, err := cApi.inboundDispatcher.HandleCatchAllStatusCallback(c, provider)
	if err != nil {
		cApi.logger.Errorf("catch-all status callback failed for provider %s: %v", provider, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}
	if statusInfo == nil {
		c.Status(http.StatusCreated)
		return
	}

	cc, err := cApi.callContextStore.GetByChannelUUID(c, provider, assistantID, statusInfo.ChannelUUID)
	if err != nil {
		cApi.logger.Errorf("failed to resolve call context for provider %s assistant %d uuid %s: %v", provider, assistantID, statusInfo.ChannelUUID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}

	auth := cc.ToAuth()
	observer := cApi.Observability(c, auth)
	scope := observability.ConversationScope{
		AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
		ConversationID: cc.ConversationID,
	}
	_ = observer.Record(c, scope, observability.RecordLog{
		Level:   observability.LevelInfo,
		Message: "telephony provider callback received",
		Attributes: observability.Attributes{
			"provider":     cc.Provider,
			"status_event": statusInfo.Event,
			"context_id":   cc.ContextID,
			"direction":    cc.Direction,
			"channel_uuid": statusInfo.ChannelUUID,
			"raw_payload":  statusInfo.RawPayload,
		},
	})
	_ = observer.Record(c, scope, observability.RecordEvent{
		Event: observability.CallStatus,
		Attributes: observability.Attributes{
			"provider":     cc.Provider,
			"status_event": statusInfo.Event,
			"context_id":   cc.ContextID,
			"direction":    cc.Direction,
			"channel_uuid": statusInfo.ChannelUUID,
		},
	})
	if statusInfo.Error != nil {
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus:       callcontext.CallStatusFailed,
			CallError:        statusInfo.Error.Error,
			FailureClass:     "provider_response",
			FailureReason:    statusInfo.Error.Reason,
			DisconnectReason: statusInfo.Error.Reason,
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from failed callback: %v", cc.ContextID, err)
		}
		_ = observer.Record(c, scope, observability.RecordMetric{
			Metrics: observability.CallStatusMetric("FAILED", statusInfo.Error.Reason),
		})
		if validator.NotBlank(statusInfo.Error.Reason) {
			_ = observer.Record(c, scope, observability.RecordMetadata{
				Metadata: observability.DisconnectMetadata(statusInfo.Error.Reason, "", ""),
			})
		}
	} else if statusInfo.Completed {
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus:       callcontext.CallStatusCompleted,
			DisconnectReason: statusInfo.Event,
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from completed callback: %v", cc.ContextID, err)
		}
	} else if validator.NotBlank(statusInfo.Event) {
		if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
			CallStatus: statusInfo.Event,
		}); err != nil {
			cApi.logger.Warnf("failed to update call context %s from callback event %s: %v", cc.ContextID, statusInfo.Event, err)
		}
	}
	metrics := make([]*protos.Metric, 0, 2)
	if statusInfo.Duration != nil {
		metrics = append(metrics, &protos.Metric{Name: observability.MetricTelephonyDuration, Value: strconv.FormatInt(statusInfo.Duration.Nanoseconds(), 10)})
	}
	if validator.NotBlank(statusInfo.Price) {
		metrics = append(metrics, &protos.Metric{Name: observability.MetricTelephonyPrice, Value: statusInfo.Price})
	}
	if len(metrics) > 0 {
		_ = observer.Record(c, scope, observability.RecordMetric{Metrics: metrics})
	}
	if err := observer.Close(context.Background()); err != nil {
		cApi.logger.Warnf("failed to close callback observability recorder: %v", err)
	}

	c.Status(http.StatusCreated)
}

// CallbackByContext handles status callback webhooks using a contextId stored in Postgres.
func (cApi *ConversationApi) CallbackByContext(c *gin.Context) {
	contextID := c.Param("contextId")
	if !validator.NotBlank(contextID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing contextId"})
		return
	}

	cc, err := cApi.callContextStore.Get(c, contextID)
	if err != nil {
		cApi.logger.Errorf("failed to resolve call context %s for event callback: %v", contextID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}

	statusInfo, err := cApi.inboundDispatcher.HandleStatusCallback(c, cc.Provider, cc.ToAuth(), cc.AssistantID, cc.ConversationID)
	if err != nil {
		cApi.logger.Errorf("status callback failed for context %s: %v", contextID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event to process"})
		return
	}
	if statusInfo != nil {
		auth := cc.ToAuth()
		observer := cApi.Observability(c, auth)
		scope := observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: cc.AssistantID},
			ConversationID: cc.ConversationID,
		}
		_ = observer.Record(c, scope, observability.RecordLog{
			Level:   observability.LevelInfo,
			Message: "telephony provider callback received",
			Attributes: observability.Attributes{
				"provider":     cc.Provider,
				"status_event": statusInfo.Event,
				"context_id":   contextID,
				"direction":    cc.Direction,
				"channel_uuid": statusInfo.ChannelUUID,
				"raw_payload":  statusInfo.RawPayload,
			},
		})
		_ = observer.Record(c, scope, observability.RecordEvent{
			Event: observability.CallStatus,
			Attributes: observability.Attributes{
				"provider":     cc.Provider,
				"status_event": statusInfo.Event,
				"context_id":   contextID,
				"direction":    cc.Direction,
				"channel_uuid": statusInfo.ChannelUUID,
			},
		})
		if statusInfo.Error != nil {
			if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
				CallStatus:       callcontext.CallStatusFailed,
				CallError:        statusInfo.Error.Error,
				FailureClass:     "provider_response",
				FailureReason:    statusInfo.Error.Reason,
				DisconnectReason: statusInfo.Error.Reason,
			}); err != nil {
				cApi.logger.Warnf("failed to update call context %s from failed callback: %v", cc.ContextID, err)
			}
			_ = observer.Record(c, scope, observability.RecordMetric{
				Metrics: observability.CallStatusMetric("FAILED", statusInfo.Error.Reason),
			})
			if validator.NotBlank(statusInfo.Error.Reason) {
				_ = observer.Record(c, scope, observability.RecordMetadata{
					Metadata: observability.DisconnectMetadata(statusInfo.Error.Reason, "", ""),
				})
			}
		} else if statusInfo.Completed {
			if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
				CallStatus:       callcontext.CallStatusCompleted,
				DisconnectReason: statusInfo.Event,
			}); err != nil {
				cApi.logger.Warnf("failed to update call context %s from completed callback: %v", cc.ContextID, err)
			}
		} else if validator.NotBlank(statusInfo.Event) {
			if err := cApi.callContextStore.UpdateCallStatus(c, cc.ContextID, callcontext.CallStatusUpdate{
				CallStatus: statusInfo.Event,
			}); err != nil {
				cApi.logger.Warnf("failed to update call context %s from callback event %s: %v", cc.ContextID, statusInfo.Event, err)
			}
		}
		metrics := make([]*protos.Metric, 0, 2)
		if statusInfo.Duration != nil {
			metrics = append(metrics, &protos.Metric{Name: observability.MetricTelephonyDuration, Value: strconv.FormatInt(statusInfo.Duration.Nanoseconds(), 10), Description: "Call duration in nanoseconds"})
		}
		if validator.NotBlank(statusInfo.Price) {
			metrics = append(metrics, &protos.Metric{Name: observability.MetricTelephonyPrice, Value: statusInfo.Price, Description: "Call price"})
		}
		if len(metrics) > 0 {
			_ = observer.Record(c, scope, observability.RecordMetric{Metrics: metrics})
		}
		if err := observer.Close(context.Background()); err != nil {
			cApi.logger.Warnf("failed to close callback observability recorder: %v", err)
		}
	}

	c.Status(http.StatusCreated)
}
