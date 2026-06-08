// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_sip_telephony

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_sip "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/sip/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/api/assistant-api/internal/observe"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Streamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	mu                    sync.RWMutex
	closed                atomic.Bool
	assistantOutputActive atomic.Bool

	session   *sip_infra.Session
	lifecycle sip_infra.LifecycleController
	mediaPort *internal_sip.MediaPort

	outputMu                    sync.Mutex
	pendingAssistantAudioFrames []assistantAudioFrame
	onTransferInitiated         func(targets []string, postTransferAction string)
}

type assistantAudioFrame struct {
	audio     []byte
	completed bool
}

type StreamerOptions struct {
	Context         context.Context
	Logger          commons.Logger
	Session         *sip_infra.Session
	Lifecycle       sip_infra.LifecycleController
	CallContext     *callcontext.CallContext
	VaultCredential *protos.VaultCredential
	Observer        observability.Recorder
}

type FuncOption func(*StreamerOptions)

func WithContext(ctx context.Context) FuncOption {
	return func(options *StreamerOptions) {
		options.Context = ctx
	}
}

func WithLogger(logger commons.Logger) FuncOption {
	return func(options *StreamerOptions) {
		options.Logger = logger
	}
}

func WithSession(session *sip_infra.Session) FuncOption {
	return func(options *StreamerOptions) {
		options.Session = session
	}
}

func WithLifecycle(lifecycle sip_infra.LifecycleController) FuncOption {
	return func(options *StreamerOptions) {
		options.Lifecycle = lifecycle
	}
}

func WithCallContext(callContext *callcontext.CallContext) FuncOption {
	return func(options *StreamerOptions) {
		options.CallContext = callContext
	}
}

func WithVaultCredential(vaultCredential *protos.VaultCredential) FuncOption {
	return func(options *StreamerOptions) {
		options.VaultCredential = vaultCredential
	}
}

func WithObserver(observer observability.Recorder) FuncOption {
	return func(options *StreamerOptions) {
		options.Observer = observer
	}
}

func New(opts ...FuncOption) (internal_type.SIPCallStreamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.Session == nil {
		return nil, fmt.Errorf("SIP session is required; standalone server mode is not supported")
	}
	if options.Lifecycle == nil {
		return nil, fmt.Errorf("SIP lifecycle controller is required")
	}

	s := &Streamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
	}

	// Peer BYE is reported to Talk; MediaPort owns bridge teardown safety.
	go func() {
		select {
		case <-options.Session.ByeReceived():
			s.Logger.Infow("SIP streamer: user BYE received")
			s.emitChannelEvent("disconnected", map[string]string{"reason": "bye_received"})
			if msg := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				s.Input(msg)
			}
		case <-s.Ctx.Done():
		}
	}()

	// Context cancellation is a safety net when Talk cannot drive teardown.
	go func() {
		select {
		case <-options.Session.Context().Done():
			s.Logger.Infow("SIP streamer: session context cancelled, closing")
		case <-options.Context.Done():
			s.Logger.Infow("SIP streamer: caller context cancelled, closing")
		case <-s.Ctx.Done():
			return
		}
		if msg := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
			s.Input(msg)
		}
		s.Close()
	}()

	s.session = options.Session
	s.lifecycle = options.Lifecycle
	isInbound := options.Session.GetInfo().Direction == sip_infra.CallDirectionInbound
	if !isInbound {
		s.assistantOutputActive.Store(true)
	}
	mediaPort, err := internal_sip.NewMediaPort(internal_sip.MediaPortConfig{
		Context:    s.Ctx,
		Logger:     options.Logger,
		Session:    options.Session,
		Resampler:  s.Resampler(),
		StreamSink: s.Input,
	})
	if err != nil {
		return nil, err
	}
	s.mediaPort = mediaPort
	if isInbound {
		s.mediaPort.StartInput()
	} else {
		s.mediaPort.Start()
		s.emitChannelEvent("media_started", nil)
	}
	s.Input(s.CreateConnectionRequest())
	s.emitChannelEvent("connected", nil)

	localIP, localPort := mediaPort.LocalAddr()
	options.Logger.Infow("SIP streamer created",
		"call_id", options.Session.GetCallID(),
		"codec", mediaPort.CodecName(),
		"rtp_port", localPort,
		"local_ip", localIP)

	return s, nil
}

func (s *Streamer) Context() context.Context {
	return s.Ctx
}

func (s *Streamer) Send(response internal_type.Stream) error {
	if s.closed.Load() {
		return sip_infra.ErrSessionClosed
	}
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if s.mediaPort != nil {
			s.mediaPort.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			s.markAssistantAudioReady(content.Audio)
			if !s.assistantOutputActive.Load() {
				s.queueAssistantAudio(content.Audio, data.GetCompleted())
				return nil
			}
			if s.mediaPort == nil {
				return nil
			}
			return s.mediaPort.HandleAssistantAudio(content.Audio, data.GetCompleted())
		}
	case *protos.ConversationInterruption:
		if data.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
			if s.mediaPort != nil {
				s.mediaPort.HandleInterrupt()
			}
		}
	case *protos.ConversationDisconnection:
		_ = s.Disconnect(data.GetType())
		s.emitChannelEvent("disconnected", map[string]string{"reason": data.GetType().String()})
		s.endSession()
		s.Close()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			s.SendTransferToolResult(data.GetId(), data.GetToolId(), data.GetName(), data.GetAction(), map[string]string{
				"status": "completed",
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			raw := data.GetArgs()["transfer_to"]
			if raw == "" {
				s.Logger.Warnw("Transfer tool call missing 'transfer_to' target")
				s.SendTransferToolResult(data.GetId(), data.GetToolId(), data.GetName(), data.GetAction(), map[string]string{
					"status": "failed", "reason": "missing transfer target",
				})
				return nil
			}
			targets := s.SplitTransferTargets(raw)
			postTransferAction := data.GetArgs()["post_transfer_action"]
			ringtone := data.GetArgs()["ringtone"]
			s.mu.RLock()
			if s.session != nil {
				s.session.SetMetadata(sip_infra.MetadataBridgeTransferTarget, strings.Join(targets, commons.SEPARATOR))
				s.session.SetMetadata("tool_id", data.GetToolId())
				s.session.SetMetadata("tool_context_id", data.GetId())
			}
			s.mu.RUnlock()
			s.EnterTransferMode(targets, postTransferAction, ringtone)
			return nil
		}
	}
	return nil
}

func (s *Streamer) StartAssistantOutput() {
	if !s.assistantOutputActive.CompareAndSwap(false, true) {
		return
	}
	if s.mediaPort != nil {
		s.mediaPort.StartOutput()
		s.mediaPort.StartBridgeRecorder()
		s.emitChannelEvent("media_started", nil)
	}
	s.outputMu.Lock()
	frames := s.pendingAssistantAudioFrames
	s.pendingAssistantAudioFrames = nil
	s.outputMu.Unlock()
	for _, frame := range frames {
		if s.mediaPort != nil {
			if err := s.mediaPort.HandleAssistantAudio(frame.audio, frame.completed); err != nil {
				s.Logger.Warnw("SIP queued assistant audio delivery failed", "error", err.Error())
			}
		}
	}
}

func (s *Streamer) queueAssistantAudio(audio []byte, completed bool) {
	audioCopy := make([]byte, len(audio))
	copy(audioCopy, audio)
	s.outputMu.Lock()
	s.pendingAssistantAudioFrames = append(s.pendingAssistantAudioFrames, assistantAudioFrame{
		audio:     audioCopy,
		completed: completed,
	})
	s.outputMu.Unlock()
}

func (s *Streamer) markAssistantAudioReady(audio []byte) {
	if len(audio) == 0 {
		return
	}
	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()
	if session == nil || session.GetInfo().Direction != sip_infra.CallDirectionInbound {
		return
	}
	if session.MarkInboundAssistantAudioReady() {
		s.Logger.Infow("SIP assistant audio ready", "call_id", session.GetCallID())
	}
}

func (s *Streamer) EnterTransferMode(targets []string, postTransferAction, ringtoneEnum string) {
	if s.mediaPort != nil && !s.mediaPort.EnterTransferMode(ringtoneEnum) {
		return
	}

	s.mu.RLock()
	session := s.session
	callback := s.onTransferInitiated
	s.mu.RUnlock()

	if session != nil {
		s.transitionCall(session, sip_infra.CallStateTransferring, sip_infra.LifecycleReasonTransferModeStarted)
	}

	if callback != nil {
		callback(targets, postTransferAction)
	}
}

func (s *Streamer) ResumeAssistant() {
	if s.mediaPort != nil && !s.mediaPort.ResumeAssistant() {
		return
	}

	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if session != nil {
		s.transitionCall(session, sip_infra.CallStateConnected, sip_infra.LifecycleReasonTransferModeEnded)
	}

	s.Logger.Infow("Transfer mode: exited, AI resuming")
}

func (s *Streamer) StopTransferRingback() {
	if s.mediaPort != nil {
		s.mediaPort.StopTransferRingback()
	}
}

func (s *Streamer) ConnectTransferMedia(target internal_type.SIPRTPBridgeTarget, outputCodecName string) {
	if s.mediaPort != nil {
		s.mediaPort.ConnectTransferMedia(target, outputCodecName)
	}
}

func (s *Streamer) DisconnectTransferMedia() {
	if s.mediaPort != nil {
		s.mediaPort.DisconnectTransferMedia()
	}
}

func (s *Streamer) RecordTransferOperatorAudio(audio []byte) {
	if s.mediaPort != nil {
		s.mediaPort.RecordTransferOperatorAudio(audio)
	}
}

func (s *Streamer) SendTransferToolResult(contextID, toolID, toolName string, action protos.ToolCallAction, result map[string]string) {
	s.Input(&protos.ConversationToolCallResult{
		Id:     contextID,
		ToolId: toolID,
		Name:   toolName,
		Action: action,
		Result: result,
	})
}

func (s *Streamer) SendTransferEvent(event internal_type.Stream) {
	s.Input(event)
}

func (s *Streamer) SetTransferRequestHandler(fn func(targets []string, postTransferAction string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onTransferInitiated = fn
}

func (s *Streamer) endSession() {
	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()
	if session != nil {
		s.endCall(session, sip_infra.LifecycleReasonStreamerEndSession)
	}
}

func (s *Streamer) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	s.emitChannelEvent("media_stopped", nil)

	if s.mediaPort != nil {
		if err := s.mediaPort.Close(); err != nil {
			s.Logger.Warnw("SIP media port close failed", "error", err)
		}
	}
	s.outputMu.Lock()
	s.pendingAssistantAudioFrames = nil
	s.outputMu.Unlock()
	s.BaseStreamer.Cancel()

	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if session != nil && shouldEndSessionOnClose(session.GetState()) {
		s.endCall(session, sip_infra.LifecycleReasonStreamerClosed)
	}

	s.Logger.Infow("SIP streamer closed")
	return nil
}

func shouldEndSessionOnClose(state sip_infra.CallState) bool {
	switch state {
	case sip_infra.CallStateInitializing, sip_infra.CallStateRinging, sip_infra.CallStateCancelled, sip_infra.CallStateFailed, sip_infra.CallStateEnded:
		return false
	default:
		return true
	}
}

func (s *Streamer) transitionCall(session *sip_infra.Session, next sip_infra.CallState, reason sip_infra.LifecycleReason) {
	if s.lifecycle == nil {
		s.Logger.Warnw("SIP lifecycle transition skipped: controller unavailable",
			"call_id", session.GetCallID(),
			"to", next,
			"reason", reason)
		return
	}
	s.lifecycle.TransitionCall(session, next, reason)
}

func (s *Streamer) endCall(session *sip_infra.Session, reason sip_infra.LifecycleReason) {
	if s.lifecycle == nil {
		s.Logger.Warnw("SIP lifecycle end skipped: controller unavailable",
			"call_id", session.GetCallID(),
			"reason", reason)
		return
	}
	if err := s.lifecycle.EndCallWithReason(session, reason); err != nil {
		s.Logger.Warnw("SIP lifecycle end failed",
			"call_id", session.GetCallID(),
			"reason", reason,
			"error", err)
	}
}

func (s *Streamer) emitChannelEvent(eventType string, extra map[string]string) {
	data := map[string]string{
		"type":     eventType,
		"provider": internal_sip.Provider,
	}
	for k, v := range extra {
		data[k] = v
	}
	s.Input(&protos.ConversationEvent{
		Name: observe.ComponentTelephony,
		Data: data,
		Time: timestamppb.Now(),
	})
}
