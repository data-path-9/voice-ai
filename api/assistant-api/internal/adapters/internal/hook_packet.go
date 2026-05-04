// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	"context"
	"sort"

	internal_condition "github.com/rapidaai/api/assistant-api/internal/condition"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/api/assistant-api/internal/variable"
	"github.com/rapidaai/pkg/utils"
)

func (r *genericRequestor) handleRunAnalysisPacket(ctx context.Context, packet internal_type.RunAnalysisPacket) {
	if packet.Analysis != nil {
		r.executeRunAnalysisPacket(ctx, packet)
		return
	}
	r.enqueueRunAnalysisPackets(ctx, packet)
}

func (r *genericRequestor) enqueueRunAnalysisPackets(ctx context.Context, packet internal_type.RunAnalysisPacket) {
	if packet.Assistant == nil {
		return
	}
	analyses := r.sortedAnalyses(packet.Assistant.AssistantAnalyses)
	if len(analyses) == 0 {
		r.enqueueCompletedWebhook(ctx, packet.ContextID)
		return
	}

	source := variable.NewCommunicationSource(r)
	registry := variable.NewDefaultRegistry().With("event", &variable.EventNamespace{})
	direction := ""
	if conv := r.Conversation(); conv != nil {
		direction = conv.Direction.String()
	}

	childPackets := make([]internal_type.RunAnalysisPacket, 0, len(analyses))
	for _, analysis := range analyses {
		if !r.isAnalysisAllowed(analysis, direction) {
			continue
		}
		args := registry.Apply(
			analysis.GetParameters(),
			source,
			variable.ResolveContext{Event: utils.ConversationCompleted.Get()},
		)
		childPackets = append(childPackets, internal_type.RunAnalysisPacket{
			ContextID:      packet.ContextID,
			Analysis:       analysis,
			Arguments:      args,
			TriggerWebhook: false,
			ConversationID: packet.ConversationID,
			Auth:           packet.Auth,
		})
	}
	if len(childPackets) == 0 {
		r.enqueueCompletedWebhook(ctx, packet.ContextID)
		return
	}

	childPackets[len(childPackets)-1].TriggerWebhook = true
	packets := make([]internal_type.Packet, 0, len(childPackets))
	for _, childPacket := range childPackets {
		packets = append(packets, childPacket)
	}
	if err := r.OnPacket(ctx, packets...); err != nil {
		r.logger.Warnw("failed to enqueue analysis packets", "error", err)
	}
}

func (r *genericRequestor) executeRunAnalysisPacket(ctx context.Context, packet internal_type.RunAnalysisPacket) {
	// if r.analysisExecutor != nil {
	// 	if err := r.analysisExecutor.Execute(ctx, packet); err != nil {
	// 		r.logger.Warnw("analysis execution failed", "name", packet.Analysis.GetName(), "error", err)
	// 	}
	// }

	// packets := make([]internal_type.Packet, 0, 1)
	// if packet.TriggerWebhook {
	// 	packets = append(packets, r.newRunWebhookPackets(packet.ContextID, utils.ConversationCompleted)...)
	// }
	// if len(packets) == 0 {
	// 	return
	// }
	// if err := r.OnPacket(ctx, packets...); err != nil {
	// 	r.logger.Warnw("failed to enqueue post-analysis packets", "error", err)
	// }
}

func (r *genericRequestor) enqueueCompletedWebhook(ctx context.Context, contextID string) {
	// webhookPackets := r.newRunWebhookPackets(contextID, utils.ConversationCompleted)
	// if len(webhookPackets) == 0 {
	// 	return
	// }
	// if err := r.OnPacket(ctx, webhookPackets...); err != nil {
	// 	r.logger.Warnw("failed to enqueue completed webhook packet", "error", err)
	// }
}

func (r *genericRequestor) sortedAnalyses(
	analyses []*internal_assistant_entity.AssistantAnalysis,
) []*internal_assistant_entity.AssistantAnalysis {
	filtered := make([]*internal_assistant_entity.AssistantAnalysis, 0, len(analyses))
	for _, analysis := range analyses {
		if analysis != nil {
			filtered = append(filtered, analysis)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left := filtered[i].GetExecutionPriority()
		right := filtered[j].GetExecutionPriority()
		if left == right {
			return filtered[i].Id > filtered[j].Id
		}
		return left > right
	})
	return filtered
}

func (r *genericRequestor) isAnalysisAllowed(analysis *internal_assistant_entity.AssistantAnalysis, direction string) bool {
	rawCondition, err := analysis.GetOptions().GetString("analysis.condition")
	if err != nil || rawCondition == "" {
		return true
	}
	parsed, parseErr := internal_condition.Parse(rawCondition)
	if parseErr != nil {
		r.logger.Warnf("invalid analysis.condition for analysis %s, excluding analysis: %v", analysis.GetName(), parseErr)
		return false
	}
	allowed, evalErr := parsed.Run(
		internal_condition.ConditionValue{RuleType: internal_condition.RuleTypeSource, Value: r.GetSource().Get()},
		internal_condition.ConditionValue{RuleType: internal_condition.RuleTypeMode, Value: r.GetMode().String()},
		internal_condition.ConditionValue{RuleType: internal_condition.RuleTypeDirection, Value: direction},
	)
	if evalErr != nil {
		r.logger.Warnf("invalid analysis.condition for analysis %s, excluding analysis: %v", analysis.GetName(), evalErr)
		return false
	}
	return allowed
}

func (r *genericRequestor) handleRunWebhookPacket(ctx context.Context, packet internal_type.RunWebhookPacket) {
	if r.webhookExecutor == nil || packet.Webhook == nil {
		return
	}
	if err := r.webhookExecutor.Execute(ctx, packet); err != nil {
		r.logger.Warnw("webhook execution failed", "error", err)
	}
}
