// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_pipeline

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"testing"

	"github.com/gorilla/websocket"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	observe "github.com/rapidaai/api/assistant-api/internal/observe"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lifecycleTracker tracks callback invocations during lifecycle tests.
type lifecycleTracker struct {
	Resolved        bool
	StreamerCreated bool
	TalkerCreated   bool
	TalkRan         bool
	ObserverCreated bool
	SessionComplete bool
	Provider        string
}

func newLifecycleDispatcher(t *testing.T, tracker *lifecycleTracker, talkErr error) *Dispatcher {
	t.Helper()
	return NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnResolveSession: func(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error) {
			tracker.Resolved = true
			return &callcontext.CallContext{
				ContextID:      contextID,
				Provider:       "asterisk",
				Direction:      "inbound",
				CallerNumber:   "+15551234567",
				AssistantID:    100,
				ConversationID: 200,
			}, &protos.VaultCredential{}, nil
		},
		OnCreateStreamer: func(ctx context.Context, cc *callcontext.CallContext, vc *protos.VaultCredential, ws *websocket.Conn, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) (internal_type.Streamer, error) {
			tracker.StreamerCreated = true
			tracker.Provider = cc.Provider
			return nil, nil
		},
		OnCreateTalker: func(ctx context.Context, streamer internal_type.Streamer) (internal_type.Talking, error) {
			tracker.TalkerCreated = true
			return nil, nil
		},
		OnRunTalk: func(ctx context.Context, talker internal_type.Talking, auth types.SimplePrinciple) error {
			tracker.TalkRan = true
			return talkErr
		},
		OnCreateObserver: func(ctx context.Context, callID string, auth types.SimplePrinciple, assistantID, conversationID uint64) *observe.ConversationObserver {
			tracker.ObserverCreated = true
			return nil
		},
		OnCompleteSession: func(ctx context.Context, contextID string) {
			tracker.SessionComplete = true
		},
	})
}

// --- Happy Path ---

func TestSessionLifecycle_HappyPath(t *testing.T) {
	t.Parallel()
	tracker := &lifecycleTracker{}
	d := newLifecycleDispatcher(t, tracker, nil)

	result := d.Run(context.Background(), SessionConnectedPipeline{
		ID: "happy", ContextID: "ctx-happy",
	})

	require.Nil(t, result.Error)
	assert.True(t, tracker.Resolved, "session should be resolved")
	assert.True(t, tracker.StreamerCreated, "streamer should be created")
	assert.True(t, tracker.TalkerCreated, "talker should be created")
	assert.True(t, tracker.TalkRan, "Talk should run")
	assert.True(t, tracker.ObserverCreated, "observer should be created")
	assert.True(t, tracker.SessionComplete, "session should be completed")
}

// --- Talk Error → FAILED ---

func TestSessionLifecycle_TalkError(t *testing.T) {
	t.Parallel()
	tracker := &lifecycleTracker{}
	d := newLifecycleDispatcher(t, tracker, errors.New("connection lost"))

	result := d.Run(context.Background(), SessionConnectedPipeline{
		ID: "talk-err", ContextID: "ctx-talk-err",
	})

	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "talk_error")
	assert.True(t, tracker.SessionComplete, "session should be completed on error")
}

// --- Panic Recovery ---

func TestSessionLifecycle_PanicRecovery(t *testing.T) {
	t.Parallel()
	tracker := &lifecycleTracker{}

	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnResolveSession: func(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error) {
			return &callcontext.CallContext{Provider: "twilio", AssistantID: 1, ConversationID: 2}, &protos.VaultCredential{}, nil
		},
		OnCreateStreamer: func(ctx context.Context, cc *callcontext.CallContext, vc *protos.VaultCredential, ws *websocket.Conn, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) (internal_type.Streamer, error) {
			return nil, nil
		},
		OnCreateTalker: func(ctx context.Context, streamer internal_type.Streamer) (internal_type.Talking, error) {
			return nil, nil
		},
		OnRunTalk: func(ctx context.Context, talker internal_type.Talking, auth types.SimplePrinciple) error {
			panic("nil pointer dereference")
		},
		OnCompleteSession: func(ctx context.Context, contextID string) {
			tracker.SessionComplete = true
		},
	})

	result := d.Run(context.Background(), SessionConnectedPipeline{
		ID: "panic", ContextID: "ctx-panic",
	})

	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "panic")
	assert.True(t, tracker.SessionComplete, "session should be completed after panic")
}

// --- Streamer Creation Failure ---

func TestSessionLifecycle_StreamerFailed(t *testing.T) {
	t.Parallel()

	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnResolveSession: func(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error) {
			return &callcontext.CallContext{Provider: "vonage", AssistantID: 1, ConversationID: 2}, &protos.VaultCredential{}, nil
		},
		OnCreateStreamer: func(ctx context.Context, cc *callcontext.CallContext, vc *protos.VaultCredential, ws *websocket.Conn, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) (internal_type.Streamer, error) {
			return nil, errors.New("codec not supported")
		},
	})

	result := d.Run(context.Background(), SessionConnectedPipeline{
		ID: "streamer-fail", ContextID: "ctx-sf",
	})

	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "codec not supported")
}

// --- Talker Creation Failure ---

func TestSessionLifecycle_TalkerFailed(t *testing.T) {
	t.Parallel()

	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnResolveSession: func(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error) {
			return &callcontext.CallContext{Provider: "exotel", AssistantID: 1, ConversationID: 2}, &protos.VaultCredential{}, nil
		},
		OnCreateStreamer: func(ctx context.Context, cc *callcontext.CallContext, vc *protos.VaultCredential, ws *websocket.Conn, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) (internal_type.Streamer, error) {
			return nil, nil
		},
		OnCreateTalker: func(ctx context.Context, streamer internal_type.Streamer) (internal_type.Talking, error) {
			return nil, errors.New("LLM provider unavailable")
		},
	})

	result := d.Run(context.Background(), SessionConnectedPipeline{
		ID: "talker-fail", ContextID: "ctx-tf",
	})

	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "LLM provider unavailable")
}

// --- All Providers Same Pipeline ---

func TestSessionLifecycle_AllProviders(t *testing.T) {
	t.Parallel()

	for _, provider := range []string{"asterisk", "twilio", "vonage", "exotel"} {
		provider := provider
		t.Run(provider, func(t *testing.T) {
			t.Parallel()
			var seenProvider string

			d := NewDispatcher(&DispatcherConfig{
				Logger: newTestLogger(),
				OnResolveSession: func(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error) {
					return &callcontext.CallContext{Provider: provider, AssistantID: 1, ConversationID: 2}, &protos.VaultCredential{}, nil
				},
				OnCreateStreamer: func(ctx context.Context, cc *callcontext.CallContext, vc *protos.VaultCredential, ws *websocket.Conn, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) (internal_type.Streamer, error) {
					seenProvider = cc.Provider
					return nil, nil
				},
				OnCreateTalker: func(ctx context.Context, streamer internal_type.Streamer) (internal_type.Talking, error) {
					return nil, nil
				},
				OnRunTalk: func(ctx context.Context, talker internal_type.Talking, auth types.SimplePrinciple) error {
					return nil
				},
				OnCompleteSession: func(ctx context.Context, contextID string) {},
			})

			result := d.Run(context.Background(), SessionConnectedPipeline{
				ID: fmt.Sprintf("prov-%s", provider), ContextID: fmt.Sprintf("ctx-%s", provider),
			})

			require.Nil(t, result.Error)
			assert.Equal(t, provider, seenProvider)
		})
	}
}

// --- Concurrent Sessions ---

func TestSessionLifecycle_ConcurrentSessions(t *testing.T) {
	t.Parallel()

	const sessions = 50
	var startCount, endCount atomic.Int32

	d := NewDispatcher(&DispatcherConfig{
		Logger: newTestLogger(),
		OnResolveSession: func(ctx context.Context, contextID string) (*callcontext.CallContext, *protos.VaultCredential, error) {
			return &callcontext.CallContext{Provider: "asterisk", AssistantID: 1, ConversationID: 2}, &protos.VaultCredential{}, nil
		},
		OnCreateStreamer: func(ctx context.Context, cc *callcontext.CallContext, vc *protos.VaultCredential, ws *websocket.Conn, conn net.Conn, reader *bufio.Reader, writer *bufio.Writer) (internal_type.Streamer, error) {
			return nil, nil
		},
		OnCreateTalker: func(ctx context.Context, streamer internal_type.Streamer) (internal_type.Talking, error) {
			return nil, nil
		},
		OnRunTalk: func(ctx context.Context, talker internal_type.Talking, auth types.SimplePrinciple) error {
			startCount.Add(1)
			return nil
		},
		OnCompleteSession: func(ctx context.Context, contextID string) {
			endCount.Add(1)
		},
	})

	done := make(chan struct{}, sessions)
	for i := 0; i < sessions; i++ {
		i := i
		go func() {
			d.Run(context.Background(), SessionConnectedPipeline{
				ID: fmt.Sprintf("c-%d", i), ContextID: fmt.Sprintf("ctx-%d", i),
			})
			done <- struct{}{}
		}()
	}

	for i := 0; i < sessions; i++ {
		<-done
	}

	assert.Equal(t, int32(sessions), startCount.Load())
	assert.Equal(t, int32(sessions), endCount.Load())
}
