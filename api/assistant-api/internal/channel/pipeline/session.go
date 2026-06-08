// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"context"
	"fmt"
	"time"

	obs "github.com/rapidaai/api/assistant-api/internal/observe"
)

// runSession handles telephony media setup and keeps the Talk lifecycle synchronous.
func (d *Dispatcher) runSession(ctx context.Context, v SessionConnectedPipeline) *PipelineResult {
	startTime := time.Now()
	d.logger.Infow("Pipeline: SessionConnected", "call_id", v.ID)

	contextID := v.ContextID
	if contextID == "" {
		contextID = v.ID
	}

	auth := v.CallContext.ToAuth()

	var projectID, organizationID uint64
	if currentProjectID := auth.GetCurrentProjectId(); currentProjectID != nil {
		projectID = *currentProjectID
	}
	if currentOrganizationID := auth.GetCurrentOrganizationId(); currentOrganizationID != nil {
		organizationID = *currentOrganizationID
	}
	observer := obs.NewConversationObserver(&obs.ConversationObserverConfig{
		Logger:         d.logger,
		Auth:           auth,
		AssistantID:    v.CallContext.AssistantID,
		ConversationID: v.CallContext.ConversationID,
		ProjectID:      projectID,
		OrganizationID: organizationID,
	})
	observer.EmitMetadata(ctx, obs.ClientMetadata(
		v.CallContext.CallerNumber, v.CallContext.FromNumber, v.CallContext.Direction, v.CallContext.Provider,
		v.CallContext.ChannelUUID, contextID, "", "", // codec/sampleRate set by streamer
	))
	observer.EmitEvent(ctx, obs.ComponentTelephony, map[string]string{
		obs.DataContextID: contextID,
		obs.DataType:      obs.EventCallStarted,
		obs.DataProvider:  v.CallContext.Provider,
		obs.DataDirection: v.CallContext.Direction,
	})

	reason := "talk_completed"
	status := "COMPLETED"

	func() {
		defer func() {
			if r := recover(); r != nil {
				reason = fmt.Sprintf("panic: %v", r)
				status = "FAILED"
				d.logger.Errorw("Pipeline: Talk panicked", "call_id", v.ID, "panic", r)
			}
		}()

		err := v.Talker.Talk(ctx, auth)
		if err != nil {
			reason = fmt.Sprintf("talk_error: %v", err)
			status = "FAILED"
		}
	}()

	observer.EmitEvent(ctx, obs.ComponentTelephony, map[string]string{
		obs.DataType:      obs.EventCallEnded,
		obs.DataProvider:  v.CallContext.Provider,
		obs.DataDirection: v.CallContext.Direction,
		obs.DataReason:    reason,
	})
	observer.EmitMetric(ctx, obs.CallStatusMetric(status, reason))
	observer.Shutdown(ctx)

	d.logger.Debugf("session completed: contextId=%s", contextID)

	d.logger.Infow("Pipeline: CallEnded",
		"call_id", v.ID,
		"duration", fmt.Sprintf("%dms", time.Since(startTime).Milliseconds()),
		"reason", reason,
		"status", status)

	if status == "FAILED" {
		return &PipelineResult{Error: fmt.Errorf("%s", reason)}
	}
	return &PipelineResult{}
}

func (d *Dispatcher) handleModeSwitch(ctx context.Context, v ModeSwitchPipeline) {
	d.logger.Infow("Pipeline: ModeSwitch", "call_id", v.ID, "from", v.From, "to", v.To)
}
