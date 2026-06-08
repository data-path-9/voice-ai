// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	sip_infra "github.com/rapidaai/api/assistant-api/sip/infra"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/require"
)

func newPipelineTestLogger(t *testing.T) commons.Logger {
	t.Helper()
	l, err := commons.NewApplicationLogger(
		commons.Level("error"),
		commons.Name("sip-pipeline-test"),
		commons.EnableFile(false),
	)
	require.NoError(t, err)
	return l
}

func newPipelineTestSession(t *testing.T) *sip_infra.Session {
	t.Helper()
	s, err := sip_infra.NewSession(context.Background(), &sip_infra.SessionConfig{
		Config: &sip_infra.Config{
			Server:            "127.0.0.1",
			Port:              5060,
			RTPPortRangeStart: 10000,
			RTPPortRangeEnd:   10020,
		},
		Direction: sip_infra.CallDirectionInbound,
	})
	require.NoError(t, err)
	return s
}

func TestHandleSessionEstablished_ConversationErrorEndsSession(t *testing.T) {
	t.Parallel()

	transferServer := &fakeTransferServer{}
	d := New(
		WithLogger(newPipelineTestLogger(t)),
		WithTransferServer(transferServer),
	)

	s := newPipelineTestSession(t)
	d.handleSessionEstablished(context.Background(), sip_infra.SessionEstablishedPipeline{
		ID:          "call-setup-fail",
		Session:     s,
		Direction:   sip_infra.CallDirectionInbound,
		AssistantID: 1,
	})

	require.Eventually(t, s.IsEnded, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, []sip_infra.LifecycleReason{sip_infra.LifecycleReasonPipelineConversationFailed}, transferServer.lifecycleEndReasons())
}

func TestHandleSessionEstablished_RuntimeSetupFailureEndsSession(t *testing.T) {
	t.Parallel()

	transferServer := &fakeTransferServer{}
	d := New(
		WithLogger(newPipelineTestLogger(t)),
		WithTransferServer(transferServer),
	)

	s := newPipelineTestSession(t)
	d.handleSessionEstablished(context.Background(), sip_infra.SessionEstablishedPipeline{
		ID:             "call-callbacks-missing",
		Session:        s,
		Direction:      sip_infra.CallDirectionInbound,
		AssistantID:    1,
		ConversationID: 42,
	})

	require.Eventually(t, s.IsEnded, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, []sip_infra.LifecycleReason{sip_infra.LifecycleReasonPipelineSetupFailed}, transferServer.lifecycleEndReasons())
}

func TestPrepareSessionDefersOutboundRuntimeUntilExplicitStart(t *testing.T) {
	t.Parallel()

	d := New(
		WithLogger(newPipelineTestLogger(t)),
	)

	stage := sip_infra.SessionEstablishedPipeline{
		ID:             "call-prepared",
		Session:        newPipelineTestSession(t),
		Direction:      sip_infra.CallDirectionOutbound,
		AssistantID:    1,
		ConversationID: 42,
	}

	require.NoError(t, d.PrepareSession(context.Background(), stage))
	require.NotNil(t, d.popPreparedSession(stage.ID))
}

func TestDiscardPreparedSessionPreventsLateStart(t *testing.T) {
	t.Parallel()

	d := New(
		WithLogger(newPipelineTestLogger(t)),
	)
	stage := sip_infra.SessionEstablishedPipeline{
		ID:             "call-runtime-discarded",
		Session:        newPipelineTestSession(t),
		Direction:      sip_infra.CallDirectionOutbound,
		AssistantID:    1,
		ConversationID: 42,
	}

	require.NoError(t, d.PrepareSession(context.Background(), stage))
	d.DiscardPreparedSession(context.Background(), stage.ID)

	require.Error(t, d.StartPreparedSession(context.Background(), stage))
}

func TestDispatcherBackpressureAndTeardownStress(t *testing.T) {
	logger := newPipelineTestLogger(t)

	const calls = 400

	transferServer := &fakeTransferServer{}

	d := New(
		WithLogger(logger),
		WithTransferServer(transferServer),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.Start(ctx)

	for i := 0; i < calls; i++ {
		s := newPipelineTestSession(t)
		d.OnPipeline(ctx, sip_infra.SessionEstablishedPipeline{
			ID:          fmt.Sprintf("call-%d", i),
			Session:     s,
			Direction:   sip_infra.CallDirectionInbound,
			AssistantID: 1,
		})
	}

	require.Eventually(t, func() bool {
		return len(transferServer.lifecycleEndReasons()) == calls
	}, 10*time.Second, 10*time.Millisecond)
}
