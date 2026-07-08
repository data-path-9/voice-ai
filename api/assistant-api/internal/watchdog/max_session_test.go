// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package watchdog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
)

func TestMaxSessionWatchdog_StartExpiresWhenDeadlinePasses(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	maxSessionWatchdog := NewMaxSessionWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))
	<-pushedPackets

	require.True(t, maxSessionWatchdog.Start("ctx-max-session", 25*time.Millisecond))

	select {
	case packet := <-pushedPackets:
		observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-max-session", observabilityLogPacket.ContextID)
		assert.Equal(t, "max-session-watchdog: deadline expired", observabilityLogPacket.Record.Message)
		assert.Equal(t, "25", observabilityLogPacket.Record.Attributes["duration_ms"])
	case <-time.After(250 * time.Millisecond):
		t.Fatal("max-session watchdog did not expire")
	}

	select {
	case packet := <-pushedPackets:
		maxSessionExpiredPacket, ok := packet.(internal_type.MaxSessionExpiredPacket)
		require.True(t, ok)
		assert.Equal(t, "ctx-max-session", maxSessionExpiredPacket.ContextID)
	case <-time.After(250 * time.Millisecond):
		t.Fatal("max-session watchdog did not push expired packet")
	}

	assert.False(t, maxSessionWatchdog.Cancel())
}

func TestMaxSessionWatchdog_StartRejectsInvalidDuration(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	maxSessionWatchdog := NewMaxSessionWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))
	<-pushedPackets

	require.False(t, maxSessionWatchdog.Start("ctx-invalid", 0))
	require.False(t, maxSessionWatchdog.Start("ctx-invalid", -time.Millisecond))

	select {
	case packet := <-pushedPackets:
		t.Fatalf("max-session watchdog pushed packet for invalid duration: %+v", packet)
	case <-time.After(60 * time.Millisecond):
	}
}

func TestMaxSessionWatchdog_CancelStopsExpiration(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	maxSessionWatchdog := NewMaxSessionWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))
	<-pushedPackets

	require.True(t, maxSessionWatchdog.Start("ctx-cancel", 40*time.Millisecond))
	require.True(t, maxSessionWatchdog.Cancel())
	require.False(t, maxSessionWatchdog.Cancel())

	select {
	case packet := <-pushedPackets:
		t.Fatalf("max-session watchdog pushed packet after cancel: %+v", packet)
	case <-time.After(90 * time.Millisecond):
	}
}

func TestMaxSessionWatchdog_StartReplacesPreviousContext(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 4)
	maxSessionWatchdog := NewMaxSessionWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))
	<-pushedPackets

	require.True(t, maxSessionWatchdog.Start("ctx-old", 25*time.Millisecond))
	require.True(t, maxSessionWatchdog.Start("ctx-new", 120*time.Millisecond))
	defer maxSessionWatchdog.Cancel()

	select {
	case packet := <-pushedPackets:
		t.Fatalf("previous context pushed packet after replacement: %+v", packet)
	case <-time.After(70 * time.Millisecond):
	}

	require.True(t, maxSessionWatchdog.Cancel())
}

func TestMaxSessionWatchdog_ConstructorPushesInitializationInfo(t *testing.T) {
	pushedPackets := make(chan internal_type.Packet, 1)

	NewMaxSessionWatchdog(WithOnPacket(func(_ context.Context, packets ...internal_type.Packet) error {
		for _, packet := range packets {
			pushedPackets <- packet
		}
		return nil
	}))

	packet := <-pushedPackets
	observabilityLogPacket, ok := packet.(internal_type.ObservabilityLogRecordPacket)
	require.True(t, ok)
	assert.Equal(t, internal_type.ObservabilityRecordScopeConversation, observabilityLogPacket.Scope)
	assert.Equal(t, observability.LevelInfo, observabilityLogPacket.Record.Level)
	assert.Equal(t, "max-session-watchdog: initialization completed", observabilityLogPacket.Record.Message)
	assert.Equal(t, "max_session", observabilityLogPacket.Record.Attributes["watchdog"])
}
