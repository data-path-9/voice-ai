// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	internal_outbound "github.com/rapidaai/api/assistant-api/sip/internal/outbound"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOutboundCall_OwnsLifecycleDependencies(t *testing.T) {
	server := &Server{}
	session := &Session{}
	invite := &outboundInvite{callID: "call-1"}
	request, err := NewOutboundInviteRequest(testOutboundConfig(), "+15551234567", "+15557654321")
	require.NoError(t, err)

	outboundCall := newOutboundCall(server, session, invite, request)

	assert.Same(t, server, outboundCall.server)
	assert.Same(t, session, outboundCall.session)
	assert.Same(t, invite, outboundCall.invite)
	assert.Equal(t, request, outboundCall.request)
}

func TestOutboundCallStatus_Values(t *testing.T) {
	assert.Equal(t, "initiated", string(OutboundCallStatusInitiated))
	assert.Equal(t, "ringing", string(OutboundCallStatusRinging))
	assert.Equal(t, "answered", string(OutboundCallStatusAnswered))
	assert.Equal(t, "failed", string(OutboundCallStatusFailed))
	assert.Equal(t, "cancelled", string(OutboundCallStatusCancelled))
}

func TestAcceptOutboundAnswer_ConfiguresRTPFromSDP(t *testing.T) {
	server := bridgeTestServer()
	invite := &outboundInvite{
		callID:     "call-1",
		rtpHandler: newTestRTPHandler(),
		dialogSession: &sipgo.DialogClientSession{
			Dialog: sipgo.Dialog{
				InviteResponse: newOutboundAnswerResponse([]byte(validOutboundAnswerSDP())),
			},
		},
	}

	answered, err := server.acceptOutboundAnswer(invite)

	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", answered.remoteIP)
	assert.Equal(t, 19000, answered.remotePort)
	require.NotNil(t, answered.negotiatedCodec)
	assert.Equal(t, CodecPCMU.Name, answered.negotiatedCodec.Name)
	assert.Equal(t, "127.0.0.1:19000", invite.rtpHandler.GetRemoteAddr().String())
}

func TestAcceptOutboundAnswer_RejectsMissingSDP(t *testing.T) {
	server := bridgeTestServer()
	invite := &outboundInvite{
		callID:     "call-1",
		rtpHandler: newTestRTPHandler(),
		dialogSession: &sipgo.DialogClientSession{
			Dialog: sipgo.Dialog{
				InviteResponse: newOutboundAnswerResponse(nil),
			},
		},
	}

	_, err := server.acceptOutboundAnswer(invite)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSDPParseFailed))
}

func TestAcceptOutboundAnswer_RejectsUnsupportedAudioCodec(t *testing.T) {
	server := bridgeTestServer()
	invite := &outboundInvite{
		callID:     "call-1",
		rtpHandler: newTestRTPHandler(),
		dialogSession: &sipgo.DialogClientSession{
			Dialog: sipgo.Dialog{
				InviteResponse: newOutboundAnswerResponse([]byte(unsupportedOutboundAnswerSDP())),
			},
		},
	}

	_, err := server.acceptOutboundAnswer(invite)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCodecNotSupported))
}

func TestOutboundCall_PreAnswerLifecycleCancelSendsSIPCancel(t *testing.T) {
	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)
	requester := newCancelTrackingRequester()
	client.TxRequester = requester

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "rapida", Host: "127.0.0.1", Port: 5060},
	}
	dialogClientCache := sipgo.NewDialogClientCache(client, contact)
	dialogSession, err := dialogClientCache.Invite(context.Background(), sip.Uri{
		Scheme: "sip",
		User:   "+15551234567",
		Host:   "trunk.example.com",
		Port:   5060,
	}, nil)
	require.NoError(t, err)
	dialogSession.InviteRequest.AppendHeader(sip.NewHeader("Route", "<sip:initial.example.com;lr>"))

	server := &Server{
		logger:     bridgeTestLogger(),
		sessions:   make(map[string]*Session),
		lifecycles: make(map[string]*CallLifecycle),
	}
	cfg := testOutboundConfig()
	cfg.InviteTimeout = time.Minute
	session, err := NewSession(context.Background(), &SessionConfig{
		Config:    cfg,
		Direction: CallDirectionOutbound,
		CallID:    dialogSession.InviteRequest.CallID().Value(),
		Logger:    server.logger,
	})
	require.NoError(t, err)
	session.SetDialogClientSession(dialogSession)
	server.registerSession(session, session.GetCallID())

	request, err := NewOutboundInviteRequest(cfg, "+15551234567", "+15557654321")
	require.NoError(t, err)
	statusRecorder := newOutboundStatusRecorder()
	outboundCall := newOutboundCall(server, session, &outboundInvite{
		callID:        session.GetCallID(),
		dialogSession: dialogSession,
	}, request)
	outboundCall.statusObserver = statusRecorder.Record
	outboundCall.start()

	require.Eventually(t, func() bool {
		return session.GetState() == CallStateRinging
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, server.EndCallWithReason(session, LifecycleReasonEndCall))

	require.Eventually(t, func() bool {
		return requester.cancelRequests.Load() == 1
	}, time.Second, 10*time.Millisecond)
	assert.Zero(t, requester.byeRequests.Load())
	assert.Equal(t, CallStateCancelled, session.GetState())
	cancelRequest := requester.cancelRequest()
	inviteRequest := requester.inviteRequest()
	require.NotNil(t, cancelRequest)
	require.NotNil(t, inviteRequest)
	assert.Equal(t, internal_outbound.SIPUserAgent, cancelRequest.GetHeader("User-Agent").Value())
	require.NotNil(t, cancelRequest.MaxForwards())
	require.NotNil(t, cancelRequest.CSeq())
	assert.Equal(t, sip.CANCEL, cancelRequest.CSeq().MethodName)
	assert.Equal(t, inviteRequest.CSeq().SeqNo, cancelRequest.CSeq().SeqNo)
	cancelledStatus := statusRecorder.LastStatus(t, OutboundCallStatusCancelled)
	assert.Equal(t, string(OutboundFailureCancelled), cancelledStatus.FailureClass)
	assert.Equal(t, LifecycleReasonOutboundCancelledBeforeAnswer.String(), cancelledStatus.DisconnectReason)
}

func TestOutboundCall_RingingTimeoutSendsLifecycleCancel(t *testing.T) {
	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)
	requester := newCancelTrackingRequester()
	client.TxRequester = requester

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "rapida", Host: "127.0.0.1", Port: 5060},
	}
	dialogClientCache := sipgo.NewDialogClientCache(client, contact)
	dialogSession, err := dialogClientCache.Invite(context.Background(), sip.Uri{
		Scheme: "sip",
		User:   "+15551234567",
		Host:   "trunk.example.com",
		Port:   5060,
	}, nil)
	require.NoError(t, err)

	server := &Server{
		logger:     bridgeTestLogger(),
		sessions:   make(map[string]*Session),
		lifecycles: make(map[string]*CallLifecycle),
	}
	cfg := testOutboundConfig()
	cfg.InviteTimeout = 20 * time.Millisecond
	session, err := NewSession(context.Background(), &SessionConfig{
		Config:    cfg,
		Direction: CallDirectionOutbound,
		CallID:    dialogSession.InviteRequest.CallID().Value(),
		Logger:    server.logger,
	})
	require.NoError(t, err)
	session.SetDialogClientSession(dialogSession)
	server.registerSession(session, session.GetCallID())

	request, err := NewOutboundInviteRequest(cfg, "+15551234567", "+15557654321")
	require.NoError(t, err)
	statusRecorder := newOutboundStatusRecorder()
	outboundCall := newOutboundCall(server, session, &outboundInvite{
		callID:        session.GetCallID(),
		dialogSession: dialogSession,
	}, request)
	outboundCall.statusObserver = statusRecorder.Record
	outboundCall.start()

	require.Eventually(t, func() bool {
		return requester.cancelRequests.Load() == 1
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return session.GetState() == CallStateFailed
	}, time.Second, 10*time.Millisecond)
	cancelRequest := requester.cancelRequest()
	require.NotNil(t, cancelRequest)
	assert.Equal(t, internal_outbound.SIPUserAgent, cancelRequest.GetHeader("User-Agent").Value())
	failureClass, ok := session.GetMetadata("sip.failure_class")
	require.True(t, ok)
	failureReason, ok := session.GetMetadata("sip.failure_reason")
	require.True(t, ok)
	assert.Equal(t, string(OutboundFailureNoAnswer), failureClass)
	assert.Equal(t, "ringing timeout", failureReason)
	assert.Zero(t, requester.byeRequests.Load())
	failedStatus := statusRecorder.LastStatus(t, OutboundCallStatusFailed)
	assert.Equal(t, string(OutboundFailureNoAnswer), failedStatus.FailureClass)
	assert.Equal(t, LifecycleReasonOutboundNoAnswer.String(), failedStatus.DisconnectReason)
}

func TestOutboundCall_AnsweredSDPFailureSendsBYENotCancel(t *testing.T) {
	ua, err := sipgo.NewUA()
	require.NoError(t, err)
	t.Cleanup(func() { _ = ua.Close() })

	client, err := sipgo.NewClient(ua)
	require.NoError(t, err)
	requester := newAnsweredDialogTrackingRequester()
	client.TxRequester = requester

	contact := sip.ContactHeader{
		Address: sip.Uri{Scheme: "sip", User: "rapida", Host: "127.0.0.1", Port: 5060},
	}
	dialogClientCache := sipgo.NewDialogClientCache(client, contact)
	dialogSession, err := dialogClientCache.Invite(context.Background(), sip.Uri{
		Scheme: "sip",
		User:   "+15551234567",
		Host:   "trunk.example.com",
		Port:   5060,
	}, nil)
	require.NoError(t, err)

	server := &Server{
		logger:     bridgeTestLogger(),
		sessions:   make(map[string]*Session),
		lifecycles: make(map[string]*CallLifecycle),
	}
	cfg := testOutboundConfig()
	cfg.InviteTimeout = time.Minute
	session, err := NewSession(context.Background(), &SessionConfig{
		Config:    cfg,
		Direction: CallDirectionOutbound,
		CallID:    dialogSession.InviteRequest.CallID().Value(),
		Logger:    server.logger,
	})
	require.NoError(t, err)
	session.SetDialogClientSession(dialogSession)
	session.SetRTPHandler(newTestRTPHandler())
	server.registerSession(session, session.GetCallID())

	request, err := NewOutboundInviteRequest(cfg, "+15551234567", "+15557654321")
	require.NoError(t, err)
	statusRecorder := newOutboundStatusRecorder()
	outboundCall := newOutboundCall(server, session, &outboundInvite{
		callID:        session.GetCallID(),
		dialogSession: dialogSession,
		rtpHandler:    newTestRTPHandler(),
	}, request)
	outboundCall.statusObserver = statusRecorder.Record
	outboundCall.start()

	require.Eventually(t, func() bool {
		return session.GetState() == CallStateFailed
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return requester.byeRequests.Load() == 1
	}, time.Second, 10*time.Millisecond)
	assert.Equal(t, int32(1), requester.ackRequests.Load())
	assert.Zero(t, requester.cancelRequests.Load())
	assert.Equal(t, OutboundDialogPhaseTerminated, session.GetOutboundDialogPhase())
	assertOutboundRouteSet(t, requester.ackRequest(), "<sip:p1.example.com;lr>", "<sip:p2.example.com;lr>")
	assertOutboundRouteSet(t, requester.byeRequest(), "<sip:p1.example.com;lr>", "<sip:p2.example.com;lr>")
	assert.Equal(t, internal_outbound.SIPUserAgent, requester.ackRequest().GetHeader("User-Agent").Value())
	assert.Equal(t, internal_outbound.SIPUserAgent, requester.byeRequest().GetHeader("User-Agent").Value())
	require.NotNil(t, requester.inviteRequest())
	assert.Equal(t, requester.inviteRequest().CSeq().SeqNo, requester.ackRequest().CSeq().SeqNo)
	assert.Equal(t, requester.inviteRequest().CSeq().SeqNo+1, requester.byeRequest().CSeq().SeqNo)
	assert.Equal(t, sip.ACK, requester.ackRequest().CSeq().MethodName)
	assert.Equal(t, sip.BYE, requester.byeRequest().CSeq().MethodName)
	failedStatus := statusRecorder.LastStatus(t, OutboundCallStatusFailed)
	assert.Equal(t, string(OutboundFailureMedia), failedStatus.FailureClass)
	assert.Equal(t, LifecycleReasonOutboundAnswerSDPFailed.String(), failedStatus.DisconnectReason)
}

type outboundStatusRecorder struct {
	mu      sync.Mutex
	updates []internal_type.ProviderCallStatusUpdate
}

func newOutboundStatusRecorder() *outboundStatusRecorder {
	return &outboundStatusRecorder{}
}

func (r *outboundStatusRecorder) Record(update internal_type.ProviderCallStatusUpdate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updates = append(r.updates, update)
}

func (r *outboundStatusRecorder) LastStatus(t *testing.T, status OutboundCallStatus) internal_type.ProviderCallStatusUpdate {
	t.Helper()
	var update internal_type.ProviderCallStatusUpdate
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		for i := len(r.updates) - 1; i >= 0; i-- {
			if r.updates[i].CallStatus == string(status) {
				update = r.updates[i]
				return true
			}
		}
		return false
	}, time.Second, 10*time.Millisecond)
	return update
}

type cancelTrackingRequester struct {
	mu             sync.Mutex
	inviteTx       *fakeClientTransaction
	inviteReq      *sip.Request
	cancelReq      *sip.Request
	cancelRequests atomic.Int32
	byeRequests    atomic.Int32
}

func newCancelTrackingRequester() *cancelTrackingRequester {
	return &cancelTrackingRequester{}
}

func (r *cancelTrackingRequester) Request(ctx context.Context, req *sip.Request) (sip.ClientTransaction, error) {
	switch req.Method {
	case sip.INVITE:
		tx := newFakeClientTransaction()
		r.mu.Lock()
		r.inviteTx = tx
		r.inviteReq = req.Clone()
		r.mu.Unlock()
		tx.respond(sip.NewResponse(180, "Ringing"))
		return tx, nil
	case sip.CANCEL:
		r.cancelRequests.Add(1)
		r.mu.Lock()
		r.cancelReq = req.Clone()
		r.mu.Unlock()
		tx := newFakeClientTransaction()
		tx.respond(sip.NewResponse(200, "OK"))
		r.mu.Lock()
		inviteTx := r.inviteTx
		r.mu.Unlock()
		if inviteTx != nil {
			inviteTx.respond(sip.NewResponse(487, "Request Terminated"))
		}
		return tx, nil
	case sip.BYE:
		r.byeRequests.Add(1)
		tx := newFakeClientTransaction()
		tx.respond(sip.NewResponse(200, "OK"))
		return tx, nil
	default:
		return newFakeClientTransaction(), nil
	}
}

func (r *cancelTrackingRequester) cancelRequest() *sip.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancelReq
}

func (r *cancelTrackingRequester) inviteRequest() *sip.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inviteReq
}

type answeredDialogTrackingRequester struct {
	mu             sync.Mutex
	inviteReq      *sip.Request
	ackReq         *sip.Request
	byeReq         *sip.Request
	ackRequests    atomic.Int32
	byeRequests    atomic.Int32
	cancelRequests atomic.Int32
}

func newAnsweredDialogTrackingRequester() *answeredDialogTrackingRequester {
	return &answeredDialogTrackingRequester{}
}

func (r *answeredDialogTrackingRequester) Request(ctx context.Context, req *sip.Request) (sip.ClientTransaction, error) {
	tx := newFakeClientTransaction()
	switch req.Method {
	case sip.INVITE:
		r.mu.Lock()
		r.inviteReq = req.Clone()
		r.mu.Unlock()
		response := sip.NewResponseFromRequest(req, 200, "OK", []byte(unsupportedOutboundAnswerSDP()))
		response.AppendHeader(sip.NewHeader("Contact", "<sip:uas@carrier.example.com>"))
		response.AppendHeader(sip.NewHeader("Record-Route", "<sip:p2.example.com;lr>"))
		response.AppendHeader(sip.NewHeader("Record-Route", "<sip:p1.example.com;lr>"))
		tx.respond(response)
	case sip.ACK:
		r.ackRequests.Add(1)
		r.mu.Lock()
		r.ackReq = req.Clone()
		r.mu.Unlock()
	case sip.BYE:
		r.byeRequests.Add(1)
		r.mu.Lock()
		r.byeReq = req.Clone()
		r.mu.Unlock()
		tx.respond(sip.NewResponseFromRequest(req, 200, "OK", nil))
	case sip.CANCEL:
		r.cancelRequests.Add(1)
		tx.respond(sip.NewResponseFromRequest(req, 200, "OK", nil))
	}
	return tx, nil
}

func (r *answeredDialogTrackingRequester) ackRequest() *sip.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ackReq
}

func (r *answeredDialogTrackingRequester) inviteRequest() *sip.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inviteReq
}

func (r *answeredDialogTrackingRequester) byeRequest() *sip.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byeReq
}

func assertOutboundRouteSet(t *testing.T, req *sip.Request, expected ...string) {
	t.Helper()
	require.NotNil(t, req)
	routes := req.GetHeaders("Route")
	require.Len(t, routes, len(expected))
	for i, route := range routes {
		assert.Equal(t, expected[i], route.Value())
	}
}

type fakeClientTransaction struct {
	done      chan struct{}
	responses chan *sip.Response
	closeOnce sync.Once
}

func newFakeClientTransaction() *fakeClientTransaction {
	return &fakeClientTransaction{
		done:      make(chan struct{}),
		responses: make(chan *sip.Response, 8),
	}
}

func (t *fakeClientTransaction) respond(response *sip.Response) {
	t.responses <- response
}

func (t *fakeClientTransaction) Terminate() {
	t.closeOnce.Do(func() {
		close(t.done)
	})
}

func (t *fakeClientTransaction) OnTerminate(f sip.FnTxTerminate) bool {
	return true
}

func (t *fakeClientTransaction) Done() <-chan struct{} {
	return t.done
}

func (t *fakeClientTransaction) Err() error {
	return nil
}

func (t *fakeClientTransaction) Responses() <-chan *sip.Response {
	return t.responses
}

func (t *fakeClientTransaction) OnRetransmission(f sip.FnTxResponse) bool {
	return true
}

func newOutboundAnswerResponse(body []byte) *sip.Response {
	response := sip.NewResponse(200, "OK")
	response.SetBody(body)
	return response
}

func validOutboundAnswerSDP() string {
	return "v=0\r\n" +
		"o=carrier 1 1 IN IP4 127.0.0.1\r\n" +
		"s=call\r\n" +
		"c=IN IP4 127.0.0.1\r\n" +
		"t=0 0\r\n" +
		"m=audio 19000 RTP/AVP 0 101\r\n" +
		"a=rtpmap:0 PCMU/8000\r\n" +
		"a=rtpmap:101 telephone-event/8000\r\n" +
		"a=sendrecv\r\n"
}

func unsupportedOutboundAnswerSDP() string {
	return "v=0\r\n" +
		"o=carrier 1 1 IN IP4 127.0.0.1\r\n" +
		"s=call\r\n" +
		"c=IN IP4 127.0.0.1\r\n" +
		"t=0 0\r\n" +
		"m=audio 19000 RTP/AVP 18 101\r\n" +
		"a=rtpmap:18 G729/8000\r\n" +
		"a=rtpmap:101 telephone-event/8000\r\n" +
		"a=sendrecv\r\n"
}
