// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_rime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	rime_internal "github.com/rapidaai/api/assistant-api/internal/transformer/rime/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type rimeTTS struct {
	*rimeOption
	ctx       context.Context
	ctxCancel context.CancelFunc

	mu             sync.Mutex
	connection     *websocket.Conn
	contextId      string
	ttsConnectedAt time.Time
	ttsStartedAt   time.Time
	ttsMetricSent  bool

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error
}

func NewRimeTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	rimeOpts, err := NewRimeOption(logger, credential, opts)
	if err != nil {
		logger.Errorf("rime-tts: failed to initialize options: %v", err)
		return nil, err
	}
	ct, ctxCancel := context.WithCancel(ctx)
	return &rimeTTS{
		ctx:        ct,
		ctxCancel:  ctxCancel,
		onPacket:   onPacket,
		logger:     logger,
		rimeOption: rimeOpts,
	}, nil
}

func (*rimeTTS) Name() string {
	return "rime-text-to-speech"
}

// Initialize opens a fresh WebSocket connection to Rime and starts the read
// goroutine for that connection. Called at session start and after each
// interruption so the connection is warm before the first text delta arrives.
func (rt *rimeTTS) Initialize() error {
	start := time.Now()
	header := http.Header{}
	header.Set("Authorization", "Bearer "+rt.GetKey())
	conn, _, err := websocket.DefaultDialer.Dial(rt.GetTextToSpeechConnectionString(), header)
	if err != nil {
		rt.logger.Errorf("rime-tts: dial failed: %v", err)
		return err
	}

	rt.mu.Lock()
	rt.connection = conn
	if rt.ttsConnectedAt.IsZero() {
		rt.ttsConnectedAt = time.Now()
	}
	rt.mu.Unlock()

	go rt.readLoop(conn)

	rt.onPacket(internal_type.ObservabilityEventRecordPacket{
		Scope: internal_type.ObservabilityRecordScopeConversation,
		Record: observability.RecordEvent{
			Component: observability.ComponentTTS,
			Event:     observability.TTSInitialized,
			Attributes: observability.Attributes{
				"type":     "initialized",
				"provider": rt.Name(),
				"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
			},
			OccurredAt: time.Now(),
		},
	})
	return nil
}

// readLoop owns a single WebSocket connection for the duration of one TTS turn.
// It exits when the connection closes — intentionally (interrupt / flush complete)
// or unexpectedly (network drop / server error).
func (rt *rimeTTS) readLoop(conn *websocket.Conn) {
	for {
		select {
		case <-rt.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			rt.mu.Lock()
			intentional := rt.connection == nil // set to nil before conn.Close() on intentional paths
			if !intentional {
				rt.connection = nil // unintentional drop: next delta will reconnect
			}
			rt.mu.Unlock()
			if !intentional {
				rt.logger.Errorf("rime-tts: connection lost: %v", err)
			}
			return
		}

		var response rime_internal.RimeTextToSpeechResponse
		if err := json.Unmarshal(msg, &response); err != nil {
			rt.logger.Errorf("rime-tts: failed to parse message: %v", err)
			continue
		}

		switch response.Type {
		case "chunk":
			rt.handleAudio(response)
		case "done":
			// Rime signals that all audio for the current flush has been sent.
			// Emit the end packet and close this per-turn connection.
			rt.handleFlushComplete(conn)
			return
		case "error":
			rt.handleServerError(conn, response)
			return
		default:
			// rt.logger.Debugf("rime-tts: unhandled message type: %s", response.Type)
		}
	}
}

// handleAudio decodes an audio chunk and forwards it downstream.
// Chunks arriving with no active contextId are discarded (post-interrupt drain).
func (rt *rimeTTS) handleAudio(response rime_internal.RimeTextToSpeechResponse) {
	raw, err := base64.StdEncoding.DecodeString(response.Data)
	if err != nil {
		rt.logger.Errorf("rime-tts: base64 decode failed: %v", err)
		return
	}

	rt.mu.Lock()
	contextId := rt.contextId
	startedAt := rt.ttsStartedAt
	metricSent := rt.ttsMetricSent
	if !metricSent && !startedAt.IsZero() {
		rt.ttsMetricSent = true
	}
	rt.mu.Unlock()

	if contextId == "" {
		rt.logger.Debugf("rime-tts: discarding audio — no active context")
		return
	}

	if !metricSent && !startedAt.IsZero() {
		rt.onPacket(internal_type.ObservabilityMetricRecordPacket{
			ContextID:   contextId,
			Scope:       internal_type.ObservabilityRecordScopeMessage,
			MessageRole: observability.MessageRoleAssistant,
			Record: observability.RecordMetric{
				Metrics: []*protos.Metric{{
					Name:  "tts_latency_ms",
					Value: fmt.Sprintf("%d", time.Since(startedAt).Milliseconds()),
				}},
				Attributes: observability.Attributes{"provider": rt.Name()},
			},
		})
	}
	rt.onPacket(internal_type.TextToSpeechAudioPacket{ContextID: contextId, AudioChunk: raw})
}

// handleFlushComplete is called when Rime sends the "done" message confirming
// that all audio for the current flush has been delivered. It emits
// TextToSpeechEndPacket — correctly ordered after the last audio chunk — and
// closes the per-turn connection.
func (rt *rimeTTS) handleFlushComplete(conn *websocket.Conn) {
	rt.mu.Lock()
	contextId := rt.contextId
	rt.connection = nil // mark before Close so readLoop error handler sees intentional
	rt.mu.Unlock()

	rt.onPacket(
		internal_type.TextToSpeechEndPacket{ContextID: contextId},
		internal_type.ObservabilityEventRecordPacket{
			ContextID:   contextId,
			Scope:       internal_type.ObservabilityRecordScopeMessage,
			MessageRole: observability.MessageRoleAssistant,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSCompleted,
				Attributes: observability.Attributes{"type": "completed"},
				OccurredAt: time.Now(),
			},
		},
	)
	conn.Close()
}

// handleServerError logs the Rime error, surfaces it downstream, and closes
// the connection. The next delta will trigger a fresh reconnect via lazy fallback.
func (rt *rimeTTS) handleServerError(conn *websocket.Conn, response rime_internal.RimeTextToSpeechResponse) {
	rt.logger.Errorf("rime-tts: server error: %s", response.Message)
	rt.mu.Lock()
	rt.connection = nil
	rt.mu.Unlock()
	rt.onPacket(internal_type.ObservabilityEventRecordPacket{
		ContextID:   response.ContextId,
		Scope:       internal_type.ObservabilityRecordScopeMessage,
		MessageRole: observability.MessageRoleAssistant,
		Record: observability.RecordEvent{
			Component:  observability.ComponentTTS,
			Event:      observability.TTSError,
			Attributes: observability.Attributes{"type": "error", "message": response.Message},
			OccurredAt: time.Now(),
		},
	})
	conn.Close()
}

func (rt *rimeTTS) Transform(ctx context.Context, in internal_type.Packet) error {
	rt.mu.Lock()
	if in.ContextId() != rt.contextId {
		rt.contextId = in.ContextId()
		rt.ttsStartedAt = time.Time{}
		rt.ttsMetricSent = false
	}
	connection := rt.connection
	rt.mu.Unlock()

	switch input := in.(type) {
	case internal_type.TextToSpeechInterruptPacket:
		// Close the current connection so any in-flight Rime audio is discarded.
		// The old readLoop goroutine will exit. Reconnect now so the fresh
		// connection is ready before the next text delta arrives.
		rt.mu.Lock()
		rt.contextId = ""
		rt.ttsStartedAt = time.Time{}
		rt.ttsMetricSent = false
		conn := rt.connection
		rt.connection = nil
		rt.mu.Unlock()
		if conn != nil {
			conn.Close()
		}
		// Emit before Initialize so downstream sees the interrupt immediately.
		rt.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID:   input.ContextID,
			Scope:       internal_type.ObservabilityRecordScopeMessage,
			MessageRole: observability.MessageRoleAssistant,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSInterrupted,
				Attributes: observability.Attributes{"type": "interrupted"},
				OccurredAt: time.Now(),
			},
		})
		if err := rt.Initialize(); err != nil {
			rt.logger.Errorf("rime-tts: reconnect after interrupt failed: %v", err)
		}
		return nil

	case internal_type.TextToSpeechTextPacket:
		// Fallback reconnect: handles Initialize() failure during interrupt or
		// an unintentional connection drop between turns.
		if connection == nil {
			if err := rt.Initialize(); err != nil {
				rt.onPacket(internal_type.TextToSpeechErrorPacket{
					ContextID: input.ContextID,
					Error:     fmt.Errorf("rime-tts: failed to connect: %w", err),
					Type:      internal_type.TTSNetworkTimeout,
				})
				return nil
			}
			rt.mu.Lock()
			connection = rt.connection
			if rt.ttsStartedAt.IsZero() {
				rt.ttsStartedAt = time.Now()
			}
			rt.mu.Unlock()
		} else {
			rt.mu.Lock()
			if rt.ttsStartedAt.IsZero() {
				rt.ttsStartedAt = time.Now()
			}
			rt.mu.Unlock()
		}
		if err := connection.WriteJSON(map[string]interface{}{"text": input.Text}); err != nil {
			rt.logger.Errorf("rime-tts: write failed: %v", err)
			rt.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("rime-tts: failed to write text: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		rt.onPacket(internal_type.ObservabilityEventRecordPacket{
			ContextID:   input.ContextID,
			Scope:       internal_type.ObservabilityRecordScopeMessage,
			MessageRole: observability.MessageRoleAssistant,
			Record: observability.RecordEvent{
				Component:  observability.ComponentTTS,
				Event:      observability.TTSSpeaking,
				Attributes: observability.Attributes{"type": "speaking", "text": input.Text},
				OccurredAt: time.Now(),
			},
		})

	case internal_type.TextToSpeechDonePacket:
		// Interrupted before done arrived — nothing to flush.
		if connection == nil {
			return nil
		}
		if err := connection.WriteJSON(map[string]interface{}{"operation": "eos"}); err != nil {
			rt.logger.Errorf("rime-tts: flush failed: %v", err)
			rt.onPacket(internal_type.TextToSpeechErrorPacket{
				ContextID: input.ContextID,
				Error:     fmt.Errorf("rime-tts: flush failed: %w", err),
				Type:      internal_type.TTSNetworkTimeout,
			})
			return nil
		}
		// TextToSpeechEndPacket is emitted by handleFlushComplete once Rime
		// confirms all audio has been delivered via the "done" response.

	default:
		return fmt.Errorf("rime-tts: unsupported packet type %T", in)
	}
	return nil
}

func (rt *rimeTTS) Close(ctx context.Context) error {
	rt.ctxCancel()
	rt.mu.Lock()
	ctxID := rt.contextId
	connectedAt := rt.ttsConnectedAt
	rt.ttsConnectedAt = time.Time{}
	if rt.connection != nil {
		conn := rt.connection
		rt.connection = nil // mark before Close so readLoop sees intentional
		conn.Close()
	}
	rt.mu.Unlock()

	if !connectedAt.IsZero() {
		duration := time.Since(connectedAt)
		rt.onPacket(
			internal_type.ObservabilityEventRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordEvent{
					Component: observability.ComponentTTS,
					Event:     observability.TTSClosed,
					Attributes: observability.Attributes{
						"type":     "closed",
						"provider": rt.Name(),
					},
					OccurredAt: time.Now(),
				},
			},
			internal_type.ObservabilityMetricRecordPacket{
				Scope: internal_type.ObservabilityRecordScopeConversation,
				Record: observability.NewConversationMetricRecord([]*protos.Metric{{
					Name:        type_enums.CONVERSATION_TTS_DURATION.String(),
					Value:       fmt.Sprintf("%d", duration.Nanoseconds()),
					Description: "Total TTS connection duration in nanoseconds",
				}}),
			},
			internal_type.ObservabilityUsageRecordPacket{
				ContextID: ctxID,
				Scope:     internal_type.ObservabilityRecordScopeConversation,
				Record: observability.RecordUsage{
					Component: observability.ComponentTTS,
					Provider:  rt.Name(),
					Duration:  duration,
					Attributes: observability.Attributes{
						"context_id": ctxID,
						"provider":   rt.Name(),
						"metric":     type_enums.CONVERSATION_TTS_DURATION.String(),
					},
				},
			},
		)
	}
	return nil
}
