// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_inbound "github.com/rapidaai/api/assistant-api/sip/internal/inbound"
)

type inboundAnswerController struct {
	server      *Server
	session     *Session
	request     *sip.Request
	transaction sip.ServerTransaction
	dialog      *sipgo.DialogServerSession
	policy      InboundAnswerPolicy
	callID      string

	mu                   sync.Mutex
	finalResponseStarted bool
	ringingCancel        context.CancelFunc
	ringingStopped       chan struct{}
}

func newInboundAnswerController(
	server *Server,
	session *Session,
	request *sip.Request,
	transaction sip.ServerTransaction,
	dialog *sipgo.DialogServerSession,
	policy InboundAnswerPolicy,
	callID string,
) *inboundAnswerController {
	return &inboundAnswerController{
		server:      server,
		session:     session,
		request:     request,
		transaction: transaction,
		dialog:      dialog,
		policy:      policy,
		callID:      callID,
	}
}

func (controller *inboundAnswerController) SendTrying() error {
	if controller.dialog == nil {
		return fmt.Errorf("inbound dialog session is required before trying")
	}
	if err := controller.dialog.Respond(100, "Trying", nil); err != nil {
		return fmt.Errorf("failed to send 100 Trying: %w", err)
	}
	controller.recordPhase(InboundSetupPhaseTryingSent, LifecycleReasonInboundInviteReceived)
	return nil
}

func (controller *inboundAnswerController) StartRinging(ctx context.Context) error {
	if controller.dialog == nil {
		return fmt.Errorf("inbound dialog session is required before ringing")
	}
	controller.mu.Lock()
	if controller.ringingCancel != nil {
		controller.mu.Unlock()
		return nil
	}
	controller.mu.Unlock()
	if err := controller.sendRingingResponse(true); err != nil {
		return err
	}

	ringingContext, ringingCancel := context.WithCancel(ctx)
	ringingStopped := make(chan struct{})
	controller.mu.Lock()
	controller.ringingCancel = ringingCancel
	controller.ringingStopped = ringingStopped
	controller.mu.Unlock()
	go controller.runRingingLoop(ringingContext, ringingStopped)
	return nil
}

func (controller *inboundAnswerController) StopRinging() {
	controller.mu.Lock()
	cancel := controller.ringingCancel
	stopped := controller.ringingStopped
	controller.ringingCancel = nil
	controller.ringingStopped = nil
	controller.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-stopped
}

func (controller *inboundAnswerController) sendRingingResponse(recordPhase bool) error {
	if controller.dialog == nil {
		return fmt.Errorf("inbound dialog session is required before ringing")
	}
	if err := controller.dialog.Respond(180, "Ringing", nil); err != nil {
		return fmt.Errorf("failed to send 180 Ringing: %w", err)
	}
	if recordPhase {
		controller.recordPhase(InboundSetupPhaseRingingSent, LifecycleReasonInboundInviteRinging)
	}
	return nil
}

func (controller *inboundAnswerController) runRingingLoop(ctx context.Context, stopped chan<- struct{}) {
	defer close(stopped)
	ticker := time.NewTicker(controller.server.effectiveInboundRingingInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-controller.server.ctx.Done():
			return
		case <-controller.session.Context().Done():
			return
		case <-ticker.C:
		}
		controller.mu.Lock()
		finalResponseStarted := controller.finalResponseStarted
		controller.mu.Unlock()
		if finalResponseStarted {
			return
		}
		if err := controller.sendRingingResponse(false); err != nil {
			controller.server.logger.Warnw("Inbound SIP ringing retransmit failed",
				"error", err,
				"call_id", controller.callID)
			return
		}
	}
}

func (controller *inboundAnswerController) WaitUntilAnswerReady(ctx context.Context) error {
	policy := controller.policy
	if policy.Mode == "" {
		policy = DefaultInboundAnswerPolicy()
	}
	if !policy.Mode.IsValid() {
		return fmt.Errorf("%w: invalid inbound answer mode %q", ErrInvalidConfig, policy.Mode)
	}

	switch policy.Mode {
	case InboundAnswerModeImmediate:
	case InboundAnswerModeAfterMinRingDuration:
		if policy.MinRingDuration <= 0 {
			return fmt.Errorf("%w: min_ring_duration is required for answer_after_min_ring_ms", ErrInvalidConfig)
		}
		if err := controller.waitForMinimumRing(ctx, policy.MinRingDuration); err != nil {
			return err
		}
	}
	controller.recordPhase(InboundSetupPhaseAnswerReady, LifecycleReasonInboundAnswerPolicyReady)
	return nil
}

func (controller *inboundAnswerController) AnswerAndWaitACK(ctx context.Context, sdpBody string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	controller.StopRinging()
	controller.mu.Lock()
	controller.finalResponseStarted = true
	controller.mu.Unlock()
	if !controller.server.beginPendingInviteFinalResponse(controller.callID) {
		return ErrInboundInviteCancelled
	}
	controller.recordPhase(InboundSetupPhaseAnswered, LifecycleReasonInboundInviteAnswered)
	if err := controller.server.sendSDPResponseAndWaitACK(
		controller.transaction,
		controller.request,
		controller.session,
		sdpBody,
		LifecycleReasonInboundInviteACKReceived,
		controller.policy.ACKTimeout,
	); err != nil {
		if err == ErrInboundACKTimeout {
			return fmt.Errorf("%w: initial INVITE ACK not received", ErrInboundACKTimeout)
		}
		return fmt.Errorf("failed to send inbound 200 OK: %w", err)
	}
	return nil
}

func (controller *inboundAnswerController) CancelBeforeAnswer(reason LifecycleReason) bool {
	if controller == nil {
		return false
	}
	controller.StopRinging()
	terminated := controller.server.terminatePendingInvite(controller.callID, 487)
	if terminated {
		controller.mu.Lock()
		controller.finalResponseStarted = true
		controller.mu.Unlock()
	}
	return terminated
}

func (controller *inboundAnswerController) FailBeforeAnswer(statusCode int, failureClass internal_inbound.FailureClass, reason LifecycleReason, err error) {
	if controller == nil {
		return
	}
	controller.mu.Lock()
	if controller.finalResponseStarted {
		controller.mu.Unlock()
		return
	}
	controller.mu.Unlock()
	controller.StopRinging()
	if controller.session == nil {
		controller.server.RejectInboundInvite(controller.request, controller.transaction, controller.callID, statusCode, failureClass, reason, err)
		controller.mu.Lock()
		controller.finalResponseStarted = true
		controller.mu.Unlock()
		return
	}
	controller.sendSessionFinalResponse(statusCode)
}

func (controller *inboundAnswerController) FinalResponseStarted() bool {
	if controller == nil {
		return false
	}
	controller.mu.Lock()
	defer controller.mu.Unlock()
	return controller.finalResponseStarted
}

func (controller *inboundAnswerController) waitForMinimumRing(ctx context.Context, minRingDuration time.Duration) error {
	timings := controller.session.GetInboundSetupTimings()
	if minRingDuration <= 0 || timings.RingingSentAt.IsZero() {
		return nil
	}
	if remaining := minRingDuration - time.Since(timings.RingingSentAt); remaining > 0 {
		timer := time.NewTimer(remaining)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-controller.server.ctx.Done():
			return controller.server.ctx.Err()
		case <-controller.session.Context().Done():
			return controller.session.Context().Err()
		case <-timer.C:
		}
	}
	return nil
}

func (controller *inboundAnswerController) sendSessionFinalResponse(statusCode int) {
	controller.mu.Lock()
	if controller.finalResponseStarted {
		controller.mu.Unlock()
		return
	}
	controller.mu.Unlock()
	controller.StopRinging()
	if controller.dialog == nil || controller.dialog.InviteRequest == nil {
		controller.server.logger.Errorw("Inbound session final response skipped without dialog ownership",
			"call_id", controller.callID,
			"status_code", statusCode)
		controller.mu.Lock()
		controller.finalResponseStarted = true
		controller.mu.Unlock()
		return
	}

	response := sip.NewResponseFromRequest(controller.dialog.InviteRequest, statusCode, "", nil)
	if response.Contact() == nil {
		contactHeader := buildSIPContactHeader(controller.server.listenConfig)
		response.AppendHeader(&contactHeader)
	}
	controller.dialog.InviteResponse = response
	if err := controller.transaction.Respond(response); err != nil {
		controller.server.logger.Errorw("Failed to send inbound dialog final response",
			"error", err,
			"call_id", controller.callID,
			"status_code", statusCode)
	}
	controller.mu.Lock()
	controller.finalResponseStarted = true
	controller.mu.Unlock()
}

func (controller *inboundAnswerController) recordPhase(phase InboundSetupPhase, reason LifecycleReason) {
	timestamp := time.Now()
	if controller.session != nil {
		controller.session.SetInboundSetupPhase(phase)
		controller.session.MarkInboundSetupTimestamp(phase, timestamp)
	}
	controller.server.logger.Infow("Inbound SIP setup phase",
		"call_id", controller.callID,
		"phase", phase,
		"reason", reason)
}
