// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package assistant_sip

import (
	"context"
	"fmt"
	"testing"

	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistRemoteByeCallStatus_UpdatesCompletedDisconnectMetadata(t *testing.T) {
	store := newSIPCallStatusTestStore(&callcontext.CallContext{
		ContextID:  "ctx-bye",
		Status:     callcontext.StatusClaimed,
		CallStatus: "answered",
	})
	engine := &SIPEngine{
		logger:           newSIPTestLogger(t),
		callContextStore: store,
	}
	session := newSIPCallStatusTestSession(t, "ctx-bye")

	engine.persistRemoteByeCallStatus(session, sip_infra.DisconnectMetadata{
		Reason:             sip_infra.DisconnectReasonNormalClearing,
		ProviderStatusCode: 16,
	})

	require.NotNil(t, store.lastStatus)
	assert.Equal(t, callcontext.StatusCompleted, store.callContext.Status)
	assert.Equal(t, callcontext.StatusCompleted, store.lastStatus.CallStatus)
	assert.Equal(t, sip_infra.DisconnectReasonNormalClearing, store.lastStatus.DisconnectReason)
	assert.Equal(t, 16, store.lastStatus.ProviderStatusCode)
}

func TestPersistRemoteByeCallStatus_DoesNotDowngradeFailure(t *testing.T) {
	store := newSIPCallStatusTestStore(&callcontext.CallContext{
		ContextID:  "ctx-failed",
		Status:     callcontext.StatusFailed,
		CallStatus: callcontext.StatusFailed,
	})
	engine := &SIPEngine{
		logger:           newSIPTestLogger(t),
		callContextStore: store,
	}
	session := newSIPCallStatusTestSession(t, "ctx-failed")

	engine.persistRemoteByeCallStatus(session, sip_infra.DisconnectMetadata{
		Reason: sip_infra.DisconnectReasonRemoteHangup,
	})

	assert.Nil(t, store.lastStatus)
	assert.Equal(t, callcontext.StatusFailed, store.callContext.Status)
}

type sipCallStatusTestStore struct {
	callContext *callcontext.CallContext
	lastStatus  *callcontext.CallStatusUpdate
}

func newSIPCallStatusTestStore(callContext *callcontext.CallContext) *sipCallStatusTestStore {
	return &sipCallStatusTestStore{callContext: callContext}
}

func (s *sipCallStatusTestStore) Save(_ context.Context, cc *callcontext.CallContext) (string, error) {
	s.callContext = cc
	return cc.ContextID, nil
}

func (s *sipCallStatusTestStore) Get(_ context.Context, contextID string) (*callcontext.CallContext, error) {
	if s.callContext == nil || s.callContext.ContextID != contextID {
		return nil, fmt.Errorf("call context not found: %s", contextID)
	}
	return s.callContext, nil
}

func (s *sipCallStatusTestStore) GetByChannelUUID(_ context.Context, _ string, _ uint64, channelUUID string) (*callcontext.CallContext, error) {
	if s.callContext == nil || s.callContext.ChannelUUID != channelUUID {
		return nil, fmt.Errorf("call context not found for channel uuid: %s", channelUUID)
	}
	return s.callContext, nil
}

func (s *sipCallStatusTestStore) Claim(_ context.Context, contextID string) (*callcontext.CallContext, error) {
	return s.Get(context.Background(), contextID)
}

func (s *sipCallStatusTestStore) UpdateField(_ context.Context, contextID, field, value string) error {
	if _, err := s.Get(context.Background(), contextID); err != nil {
		return err
	}
	if field == "status" {
		s.callContext.Status = value
	}
	return nil
}

func (s *sipCallStatusTestStore) UpdateCallStatus(_ context.Context, contextID string, status callcontext.CallStatusUpdate) error {
	if _, err := s.Get(context.Background(), contextID); err != nil {
		return err
	}
	s.lastStatus = &status
	s.callContext.CallStatus = status.CallStatus
	s.callContext.DisconnectReason = status.DisconnectReason
	s.callContext.ProviderStatusCode = status.ProviderStatusCode
	if status.CallStatus == callcontext.StatusCompleted {
		s.callContext.Status = callcontext.StatusCompleted
	}
	if status.CallStatus == callcontext.StatusFailed || status.CallStatus == "cancelled" {
		s.callContext.Status = callcontext.StatusFailed
	}
	return nil
}

func newSIPCallStatusTestSession(t *testing.T, contextID string) *sip_infra.Session {
	t.Helper()
	session, err := sip_infra.NewSession(context.Background(), &sip_infra.SessionConfig{
		Config: &sip_infra.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		},
		Direction: sip_infra.CallDirectionOutbound,
		CallID:    "sip-call-id",
		ContextID: contextID,
	})
	require.NoError(t, err)
	return session
}

func newSIPTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	logger, err := commons.NewApplicationLogger(
		commons.EnableFile(false),
		commons.Level("error"),
	)
	require.NoError(t, err)
	return logger
}
