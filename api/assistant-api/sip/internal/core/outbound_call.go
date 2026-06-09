// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_outbound "github.com/rapidaai/api/assistant-api/sip/internal/outbound"
)

type outboundCall struct {
	server         *Server
	session        *Session
	invite         *outboundInvite
	request        OutboundInviteRequest
	answerContext  context.Context
	statusObserver internal_type.ProviderCallStatusReporter
	closeOnce      sync.Once
}

func newOutboundCall(server *Server, session *Session, invite *outboundInvite, request OutboundInviteRequest) *outboundCall {
	return &outboundCall{
		server:  server,
		session: session,
		invite:  invite,
		request: request,
	}
}

func (outboundCall *outboundCall) start() {
	go outboundCall.run()
}

func (outboundCall *outboundCall) run() {
	defer outboundCall.invite.dialogSession.Close()

	answerTime, err := outboundCall.connect()
	if err != nil {
		return
	}

	outboundCall.startInviteHandler(answerTime)
	outboundCall.waitForSessionEnd(answerTime)
}

func (outboundCall *outboundCall) connect() (time.Time, error) {
	server := outboundCall.server
	session := outboundCall.session
	dialogSession := outboundCall.invite.dialogSession
	rtpHandler := outboundCall.invite.rtpHandler
	callID := session.GetCallID()
	outboundConfig := session.config.ToOutboundConfig()
	ringingTimeout := outboundConfig.EffectiveRingingTimeout()
	assistantID := uint64(0)
	if assistant := session.GetAssistant(); assistant != nil {
		assistantID = assistant.Id
	}

	digestURI := dialogSession.InviteRequest.Recipient.Addr()
	server.logger.Debugw("Outbound call waiting for answer",
		"call_id", callID,
		"context_id", session.GetContextID(),
		"assistant_id", assistantID,
		"conversation_id", session.GetConversationID(),
		"mode", outboundCall.request.Config.Mode,
		"to_user", outboundCall.request.Identity.ToUser,
		"from_user", outboundCall.request.Identity.FromUser,
		"trunk_address", outboundCall.request.Config.Address,
		"ringing_timeout_ms", ringingTimeout.Milliseconds(),
		"auth_username", outboundConfig.Auth.Username,
		"auth_realm", outboundConfig.Auth.Realm,
		"digest_uri", digestURI,
		"request_uri", dialogSession.InviteRequest.Recipient.String())

	answerParentContext := session.Context()
	if outboundCall.answerContext != nil {
		answerParentContext = outboundCall.answerContext
	}
	answerCtx, cancelAnswerWithCause := context.WithCancelCause(context.Background())
	var answerCompleted atomic.Bool
	var preAnswerCancelOnce sync.Once
	var ringingTimeoutReached atomic.Bool
	var parentContextClosed atomic.Bool
	var ringingReported atomic.Bool
	sendPreAnswerCancel := func() {
		preAnswerCancelOnce.Do(func() {
			outboundCall.cancelPreAnswer()
			cancelAnswerWithCause(sipgo.WaitAnswerForceCancelErr)
		})
	}
	answerDone := make(chan struct{})
	go func() {
		select {
		case <-answerParentContext.Done():
			if answerCompleted.Load() {
				return
			}
			parentContextClosed.Store(true)
			sendPreAnswerCancel()
		case <-answerDone:
		}
	}()
	ringingTimer := time.AfterFunc(ringingTimeout, func() {
		if answerCompleted.Load() {
			return
		}
		ringingTimeoutReached.Store(true)
		sendPreAnswerCancel()
	})
	cancelPreAnswer := func() {
		sendPreAnswerCancel()
	}
	session.SetOnPreAnswerCancel(cancelPreAnswer)

	err := dialogSession.WaitAnswer(answerCtx, sipgo.AnswerOptions{
		Username: outboundConfig.Auth.Username,
		Password: outboundConfig.Auth.Password,
		OnResponse: func(response *sip.Response) error {
			statusCode := response.StatusCode
			server.logger.Debugw("Outbound call response",
				"call_id", callID,
				"status", statusCode)

			if outboundAuthMissingForChallenge(outboundConfig.Auth, statusCode) {
				return ErrAuthRequired
			}

			if statusCode == 180 || statusCode == 183 {
				session.SetOutboundDialogPhase(OutboundDialogPhaseProceeding)
				server.TransitionCall(session, CallStateRinging, LifecycleReasonOutboundProgressRinging)
				if ringingReported.CompareAndSwap(false, true) {
					outboundCall.reportStatus(internal_type.ProviderCallStatusUpdate{
						CallStatus:         string(OutboundCallStatusRinging),
						DisconnectReason:   LifecycleReasonOutboundProgressRinging.String(),
						ProviderStatusCode: statusCode,
					})
				}
			}

			outboundCall.logAuthChallenge(response, outboundConfig.Auth)
			return nil
		},
	})
	answerCompleted.Store(true)
	close(answerDone)
	ringingTimer.Stop()
	session.ClearOnPreAnswerCancel()
	cancelAnswerWithCause(nil)
	if err != nil {
		if ringingTimeoutReached.Load() {
			err = context.DeadlineExceeded
		} else if parentContextClosed.Load() && answerParentContext.Err() != nil {
			err = answerParentContext.Err()
		}
		outboundCall.failBeforeAnswer(err, outboundConfig.Auth, answerCtx)
		return time.Time{}, err
	}

	answerTime := time.Now()
	session.SetOutboundDialogPhase(OutboundDialogPhaseAnswered)
	server.logger.Infow("Outbound call 200 OK received; setting up RTP before ACK",
		"call_id", callID)

	answered, err := server.acceptOutboundAnswer(outboundCall.invite)
	if err != nil {
		if ackErr := outboundCall.ackAnsweredDialog(); ackErr != nil {
			server.logger.Warnw("Failed to ACK rejected outbound answer",
				"call_id", callID,
				"error", ackErr)
		} else {
			session.SetOutboundDialogPhase(OutboundDialogPhaseConfirmed)
		}
		outboundCall.failAfterAnswer(LifecycleReasonOutboundAnswerSDPFailed, err)
		return time.Time{}, err
	}
	session.SetRemoteRTP(answered.remoteIP, answered.remotePort)
	if answered.negotiatedCodec != nil {
		session.SetNegotiatedCodec(answered.negotiatedCodec.Name, int(answered.negotiatedCodec.ClockRate))
		server.logger.Infow("Outbound call codec negotiated from 200 OK",
			"call_id", callID,
			"codec", answered.negotiatedCodec.Name,
			"payload_type", answered.negotiatedCodec.PayloadType,
			"clock_rate", answered.negotiatedCodec.ClockRate)
	}

	rtpHandler.Start()

	localIP, localPort := rtpHandler.LocalAddr()
	remoteAddr := rtpHandler.GetRemoteAddr()
	server.logger.Infow("RTP started (pre-ACK)",
		"call_id", callID,
		"local_rtp", fmt.Sprintf("%s:%d", localIP, localPort),
		"remote_rtp", fmt.Sprintf("%s:%d", answered.remoteIP, answered.remotePort),
		"remote_addr_set", remoteAddr != nil,
		"elapsed_since_200ok_ms", time.Since(answerTime).Milliseconds())

	if err := outboundCall.ackAnsweredDialog(); err != nil {
		server.logger.Errorw("Failed to send ACK", "error", err, "call_id", callID)
		outboundCall.failAfterAnswer(LifecycleReasonOutboundACKFailed, err)
		return time.Time{}, err
	}
	server.logger.Infow("ACK sent (RTP already flowing)",
		"call_id", callID,
		"elapsed_since_200ok_ms", time.Since(answerTime).Milliseconds())

	session.SetOutboundDialogPhase(OutboundDialogPhaseConfirmed)
	server.TransitionCall(session, CallStateConnected, LifecycleReasonOutboundACKSent)
	outboundCall.reportStatus(internal_type.ProviderCallStatusUpdate{
		CallStatus:       string(OutboundCallStatusAnswered),
		DisconnectReason: LifecycleReasonOutboundACKSent.String(),
	})
	return answerTime, nil
}

func (outboundCall *outboundCall) reportStatus(update internal_type.ProviderCallStatusUpdate) {
	if outboundCall.statusObserver == nil {
		return
	}
	update.ChannelUUID = outboundCall.session.GetCallID()
	outboundCall.statusObserver(update)
}

func (outboundCall *outboundCall) reportFailure(failure OutboundFailure, err error) {
	status := OutboundCallStatusFailed
	if failure.Class == OutboundFailureCancelled {
		status = OutboundCallStatusCancelled
	}
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}
	outboundCall.reportStatus(internal_type.ProviderCallStatusUpdate{
		CallStatus:         string(status),
		ErrorMessage:       errorMessage,
		FailureClass:       string(failure.Class),
		FailureReason:      failure.Reason,
		DisconnectReason:   failure.LifecycleReason.String(),
		Retryable:          failure.Retryable,
		ProviderStatusCode: failure.StatusCode,
	})
}

func (outboundCall *outboundCall) logAuthChallenge(response *sip.Response, auth SIPAuthConfig) {
	server := outboundCall.server
	callID := outboundCall.session.GetCallID()
	statusCode := response.StatusCode

	if statusCode == 401 {
		if wwwAuth := response.GetHeader("WWW-Authenticate"); wwwAuth != nil {
			server.logger.Debugw("SIP 401 challenge received",
				"call_id", callID,
				"www_authenticate", wwwAuth.Value(),
				"auth_username", auth.Username)
		}
		if authHeader := outboundCall.invite.dialogSession.InviteRequest.GetHeader("Authorization"); authHeader != nil {
			server.logger.Debugw("SIP digest Authorization sent",
				"call_id", callID,
				"has_authorization", true)
		}
	}
	if statusCode == 407 {
		if proxyAuth := response.GetHeader("Proxy-Authenticate"); proxyAuth != nil {
			server.logger.Debugw("SIP 407 challenge received",
				"call_id", callID,
				"proxy_authenticate", proxyAuth.Value(),
				"auth_username", auth.Username)
		}
		if authHeader := outboundCall.invite.dialogSession.InviteRequest.GetHeader("Proxy-Authorization"); authHeader != nil {
			server.logger.Debugw("SIP digest Proxy-Authorization sent",
				"call_id", callID,
				"has_proxy_authorization", true)
		}
	}
}

func (outboundCall *outboundCall) ackAnsweredDialog() error {
	ackCtx, cancelAck := context.WithTimeout(outboundCall.session.Context(), 5*time.Second)
	defer cancelAck()
	dialogSession := outboundCall.invite.dialogSession
	internal_outbound.NormalizeDialogRouteSet(dialogSession)
	ackRequest := internal_outbound.NewAckRequest(dialogSession.InviteRequest, dialogSession.InviteResponse)
	return dialogSession.WriteAck(ackCtx, ackRequest)
}

func (outboundCall *outboundCall) cancelPreAnswer() {
	dialogSession := outboundCall.invite.dialogSession
	if dialogSession == nil || dialogSession.InviteRequest == nil {
		return
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := internal_outbound.SendCancel(cancelCtx, dialogSession, dialogSession.InviteRequest); err != nil {
		outboundCall.server.logger.Warnw("Failed to send outbound SIP CANCEL",
			"call_id", outboundCall.session.GetCallID(),
			"error", err)
	}
}

func (outboundCall *outboundCall) failBeforeAnswer(err error, auth SIPAuthConfig, answerCtx context.Context) {
	server := outboundCall.server
	session := outboundCall.session
	dialogSession := outboundCall.invite.dialogSession
	callID := session.GetCallID()
	failure := classifyOutboundFailure(err, answerCtx)
	session.SetMetadata("sip.failure_class", string(failure.Class))
	session.SetMetadata("sip.failure_reason", failure.Reason)
	outboundCall.reportFailure(failure, err)
	if session.IsEnded() {
		return
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(answerCtx.Err(), context.DeadlineExceeded):
		server.logger.Warnw("Outbound call ringing timeout reached; INVITE cancelled",
			"call_id", callID,
			"ringing_timeout_ms", outboundCall.request.Config.EffectiveRingingTimeout().Milliseconds())
	case errors.Is(err, context.Canceled):
		server.logger.Infow("Outbound call cancelled before answer",
			"call_id", callID,
			"reason", "context_cancelled")
		_ = server.CancelCall(session, LifecycleReasonOutboundCancelledBeforeAnswer)
		time.AfterFunc(2*time.Second, func() {
			dialogSession.Close()
		})
		return
	case errors.Is(err, ErrAuthRequired):
		server.logger.Errorw("Outbound call authentication required but credentials are missing",
			"call_id", callID,
			"auth_username_set", auth.Username != "",
			"auth_password_set", auth.Password != "",
			"failure_class", failure.Class)
	}
	var dialogErr *sipgo.ErrDialogResponse
	if errors.As(err, &dialogErr) {
		if dialogErr.Res.StatusCode == 401 || dialogErr.Res.StatusCode == 407 {
			server.logger.Errorw("Outbound call authentication failed; check SIP credentials in vault",
				"call_id", callID,
				"status", dialogErr.Res.StatusCode,
				"reason", dialogErr.Res.Reason,
				"auth_username", auth.Username,
				"auth_password_set", len(auth.Password) > 0,
				"digest_uri", dialogSession.InviteRequest.Recipient.Addr(),
				"failure_class", failure.Class,
				"hint", "Verify sip_username and sip_password in vault match the SIP provider's auth credentials")
		} else {
			server.logger.Warnw("Outbound call rejected by remote",
				"call_id", callID,
				"status", dialogErr.Res.StatusCode,
				"reason", dialogErr.Res.Reason,
				"failure_class", failure.Class,
				"retryable", failure.Retryable)
		}
	} else if !errors.Is(err, context.Canceled) {
		server.logger.Warnw("Outbound call failed",
			"call_id", callID,
			"error", err,
			"failure_class", failure.Class,
			"failure_reason", failure.Reason,
			"retryable", failure.Retryable)
	}
	_ = server.FailCall(session, failure.LifecycleReason, err)
	time.AfterFunc(2*time.Second, func() {
		dialogSession.Close()
	})
}

func (outboundCall *outboundCall) failAfterAnswer(reason LifecycleReason, err error) {
	server := outboundCall.server
	session := outboundCall.session

	outboundCall.closeOnce.Do(func() {
		failure := classifyOutboundFailure(err, nil)
		if failure.Class == OutboundFailureUnknown {
			if err != nil {
				failure.Reason = err.Error()
			}
			failure.LifecycleReason = reason
		}
		outboundCall.reportFailure(failure, err)
		_ = server.FailCall(session, reason, err)
	})
}

func (outboundCall *outboundCall) startInviteHandler(answerTime time.Time) {
	server := outboundCall.server
	session := outboundCall.session
	callID := session.GetCallID()

	server.mu.RLock()
	onInvite := server.onInvite
	server.mu.RUnlock()
	if onInvite == nil {
		return
	}

	info := session.GetInfo()
	server.logger.Infow("Starting onInvite handler for outbound call",
		"call_id", callID)
	if err := onInvite(session, info.LocalURI, info.RemoteURI); err != nil {
		server.logger.Errorw("Outbound INVITE handler failed", "error", err, "call_id", callID)
	} else {
		server.logger.Infow("onInvite handler completed",
			"call_id", callID,
			"total_elapsed_ms", time.Since(answerTime).Milliseconds())
	}
}

func (outboundCall *outboundCall) waitForSessionEnd(answerTime time.Time) {
	server := outboundCall.server
	session := outboundCall.session
	dialogSession := outboundCall.invite.dialogSession
	callID := session.GetCallID()
	maxCallDuration := outboundCall.request.Config.EffectiveMaxCallDuration()

	var maxDurationC <-chan time.Time
	var maxDurationTimer *time.Timer
	if maxCallDuration > 0 {
		maxDurationTimer = time.NewTimer(maxCallDuration)
		maxDurationC = maxDurationTimer.C
		defer maxDurationTimer.Stop()
	}

	server.logger.Debugw("Outbound dialog waiting for session to end", "call_id", callID)
	select {
	case <-maxDurationC:
		server.logger.Infow("Outbound call max duration reached; ending dialog",
			"call_id", callID,
			"max_call_duration_ms", maxCallDuration.Milliseconds())
		outboundCall.closeOnce.Do(func() {
			_ = server.EndCallWithReason(session, LifecycleReasonOutboundMaxDuration)
		})
	case <-session.Context().Done():
		server.logger.Infow("Outbound dialog ending; session ended",
			"call_id", callID,
			"call_duration_ms", time.Since(answerTime).Milliseconds())
	case <-dialogSession.Context().Done():
		server.logger.Infow("Outbound dialog BYE received; waiting for session teardown",
			"call_id", callID,
			"call_duration_ms", time.Since(answerTime).Milliseconds())
		select {
		case <-session.Context().Done():
			server.logger.Debugw("Outbound dialog session ended after BYE",
				"call_id", callID)
		case <-time.After(30 * time.Second):
			server.logger.Warnw("Outbound dialog session did not end within 30s after BYE; forcing teardown",
				"call_id", callID)
			if !session.IsEnded() {
				_ = server.EndCallWithReason(session, LifecycleReasonOutboundTeardownTimeout)
			}
		}
	}
}
