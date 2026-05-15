// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type textToSpeech struct {
	config *Config
	engine *dslEngine

	ctx    context.Context
	cancel context.CancelFunc

	logger   commons.Logger
	onPacket func(pkt ...internal_type.Packet) error

	mu             sync.Mutex
	connection     *websocket.Conn
	currentContext string
	connectedAt    time.Time
	turnStartedAt  time.Time
	metricEmitted  bool
}

type readErrorDisposition int

const (
	readErrorIgnore readErrorDisposition = iota
	readErrorComplete
	readErrorFail
)

func NewTextToSpeech(
	ctx context.Context,
	logger commons.Logger,
	credential *protos.VaultCredential,
	onPacket func(pkt ...internal_type.Packet) error,
	opts utils.Option,
) (internal_type.TextToSpeechTransformer, error) {
	config, err := NewConfig(credential, opts)
	if err != nil {
		return nil, err
	}
	ctx2, cancel := context.WithCancel(ctx)

	return &textToSpeech{
		config:   config,
		engine:   config.newEngine(),
		ctx:      ctx2,
		cancel:   cancel,
		logger:   logger,
		onPacket: onPacket,
	}, nil
}

func (*textToSpeech) Name() string {
	return "custom-tts-websocket-v1"
}

func (transformer *textToSpeech) Initialize() error {
	return nil
}

func (transformer *textToSpeech) Transform(ctx context.Context, in internal_type.Packet) error {
	switch input := in.(type) {
	case internal_type.TextToSpeechTextPacket:
		return transformer.handleText(input.ContextID, input.Text)
	case internal_type.LLMResponseDeltaPacket:
		return transformer.handleText(input.ContextID, input.Text)
	case internal_type.TextToSpeechDonePacket:
		return transformer.handleDone(input.ContextID, input.Text)
	case internal_type.LLMResponseDonePacket:
		return transformer.handleDone(input.ContextID, input.Text)
	case internal_type.TextToSpeechInterruptPacket:
		transformer.handleInterrupt(input.ContextID)
		return nil
	case internal_type.InterruptionDetectedPacket:
		transformer.handleInterrupt(input.ContextID)
		return nil
	default:
		return fmt.Errorf("custom-tts websocket_v1: unsupported input type %T", in)
	}
}

func (transformer *textToSpeech) Close(ctx context.Context) error {
	transformer.cancel()

	transformer.mu.Lock()
	conn := transformer.connection
	contextID := transformer.currentContext
	connectedAt := transformer.connectedAt
	transformer.connection = nil
	transformer.currentContext = ""
	transformer.connectedAt = time.Time{}
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	if !connectedAt.IsZero() {
		transformer.emitPackets(
			internal_type.ConversationEventPacket{
				ContextID: contextID,
				Name:      "tts",
				Data: map[string]string{
					"type":     "closed",
					"provider": transformer.Name(),
				},
				Time: time.Now(),
			},
			internal_type.ConversationMetricPacket{
				ContextID: 0,
				Metrics: []*protos.Metric{{
					Name:        type_enums.CONVERSATION_TTS_DURATION.String(),
					Value:       fmt.Sprintf("%d", time.Since(connectedAt).Nanoseconds()),
					Description: "Total TTS connection duration in nanoseconds",
				}},
			},
		)
	}

	return nil
}

func (transformer *textToSpeech) handleText(contextID, text string) error {
	scope := transformer.config.newScope(contextID, text)
	conn, err := transformer.getOrOpenConnection(scope)
	if err != nil {
		transformer.emitTTSError(contextID, fmt.Errorf("custom-tts websocket_v1: failed to connect: %w", err), internal_type.TTSNetworkTimeout)
		return nil
	}

	payload, err := transformer.engine.RenderTextRequest(scope)
	if err != nil {
		transformer.emitTTSError(contextID, err, internal_type.TTSInvalidInput)
		return nil
	}

	transformer.mu.Lock()
	if transformer.turnStartedAt.IsZero() {
		transformer.turnStartedAt = time.Now()
	}
	transformer.mu.Unlock()

	if err := conn.WriteJSON(payload); err != nil {
		transformer.emitTTSError(contextID, fmt.Errorf("custom-tts websocket_v1: failed to write text request: %w", err), internal_type.TTSNetworkTimeout)
		transformer.dropConnection(conn)
		return nil
	}

	transformer.emitPackets(internal_type.ConversationEventPacket{
		ContextID: contextID,
		Name:      "tts",
		Data: map[string]string{
			"type": "speaking",
			"text": text,
		},
		Time: time.Now(),
	})

	return nil
}

func (transformer *textToSpeech) handleDone(contextID, text string) error {
	if !transformer.config.HasDoneRequest {
		return nil
	}

	scope := transformer.config.newScope(contextID, text)
	payload, err := transformer.engine.RenderDoneRequest(scope)
	if err != nil {
		transformer.emitTTSError(contextID, err, internal_type.TTSInvalidInput)
		return nil
	}
	if payload == nil {
		return nil
	}

	transformer.mu.Lock()
	conn := transformer.connection
	activeContext := transformer.currentContext
	transformer.mu.Unlock()

	if conn == nil || activeContext != contextID {
		return nil
	}
	if err := conn.WriteJSON(payload); err != nil {
		transformer.emitTTSError(contextID, fmt.Errorf("custom-tts websocket_v1: failed to write done request: %w", err), internal_type.TTSNetworkTimeout)
		transformer.dropConnection(conn)
	}

	return nil
}

func (transformer *textToSpeech) handleInterrupt(contextID string) {
	transformer.mu.Lock()
	conn := transformer.connection
	transformer.connection = nil
	transformer.currentContext = ""
	transformer.connectedAt = time.Time{}
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}

	transformer.emitPackets(internal_type.ConversationEventPacket{
		ContextID: contextID,
		Name:      "tts",
		Data:      map[string]string{"type": "interrupted"},
		Time:      time.Now(),
	})
}

func (transformer *textToSpeech) getOrOpenConnection(scope requestScope) (*websocket.Conn, error) {
	transformer.mu.Lock()
	if transformer.connection != nil && transformer.currentContext == scope.MessageID {
		conn := transformer.connection
		transformer.mu.Unlock()
		return conn, nil
	}
	oldConn := transformer.connection
	transformer.connection = nil
	transformer.currentContext = scope.MessageID
	transformer.connectedAt = time.Time{}
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}

	connectionURL, err := transformer.engine.BuildConnectionURL(scope)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	for key, value := range transformer.config.Headers {
		headers.Set(key, value)
	}

	start := time.Now()
	conn, response, err := websocket.DefaultDialer.Dial(connectionURL, headers)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	transformer.mu.Lock()
	transformer.connection = conn
	transformer.currentContext = scope.MessageID
	transformer.connectedAt = time.Now()
	transformer.turnStartedAt = time.Time{}
	transformer.metricEmitted = false
	transformer.mu.Unlock()

	go transformer.readLoop(conn, scope.MessageID)

	transformer.emitPackets(internal_type.ConversationEventPacket{
		ContextID: scope.MessageID,
		Name:      "tts",
		Data: map[string]string{
			"type":     "initialized",
			"provider": transformer.Name(),
			"init_ms":  fmt.Sprintf("%d", time.Since(start).Milliseconds()),
		},
		Time: time.Now(),
	})

	return conn, nil
}

func (transformer *textToSpeech) readLoop(conn *websocket.Conn, contextID string) {
	for {
		select {
		case <-transformer.ctx.Done():
			return
		default:
		}

		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			switch transformer.classifyReadError(conn, err) {
			case readErrorIgnore:
				return
			case readErrorComplete:
				transformer.emitPackets(
					internal_type.TextToSpeechEndPacket{ContextID: contextID},
					internal_type.ConversationEventPacket{
						ContextID: contextID,
						Name:      "tts",
						Data:      map[string]string{"type": "completed"},
						Time:      time.Now(),
					},
				)
				return
			case readErrorFail:
			default:
			}
			transformer.emitTTSError(contextID, fmt.Errorf("custom-tts websocket_v1: read failed: %w", err), internal_type.TTSNetworkTimeout)
			return
		}

		frame, err := transformer.engine.ParseFrame(messageType, payload)
		if err != nil {
			transformer.emitTTSError(contextID, err, internal_type.TTSUnknownError)
			continue
		}

		outcome, err := transformer.engine.EvaluateResponse(frame, contextID)
		if err != nil {
			transformer.emitTTSError(contextID, err, internal_type.TTSUnknownError)
			continue
		}
		if !outcome.Matched {
			continue
		}

		resolvedContextID := outcome.MessageID
		if resolvedContextID == "" {
			resolvedContextID = contextID
		}

		if len(outcome.Audio) > 0 {
			transformer.emitFirstAudioMetric(resolvedContextID)
			transformer.emitPackets(internal_type.TextToSpeechAudioPacket{
				ContextID:  resolvedContextID,
				AudioChunk: outcome.Audio,
			})
		}

		if outcome.ErrorText != "" {
			transformer.emitTTSError(resolvedContextID, errors.New(outcome.ErrorText), internal_type.TTSUnknownError)
		}

		if outcome.Done {
			transformer.dropConnection(conn)
			transformer.emitPackets(
				internal_type.TextToSpeechEndPacket{ContextID: resolvedContextID},
				internal_type.ConversationEventPacket{
					ContextID: resolvedContextID,
					Name:      "tts",
					Data:      map[string]string{"type": "completed"},
					Time:      time.Now(),
				},
			)
			return
		}
	}
}

func (transformer *textToSpeech) classifyReadError(conn *websocket.Conn, err error) readErrorDisposition {
	transformer.mu.Lock()
	active := transformer.connection == conn
	turnStarted := !transformer.turnStartedAt.IsZero()
	if active {
		transformer.connection = nil
		transformer.currentContext = ""
		transformer.connectedAt = time.Time{}
		transformer.turnStartedAt = time.Time{}
		transformer.metricEmitted = false
	}
	transformer.mu.Unlock()
	if active && conn != nil {
		_ = conn.Close()
	}

	if !active {
		return readErrorIgnore
	}
	if transformer.ctx.Err() != nil {
		return readErrorIgnore
	}
	if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		if turnStarted {
			return readErrorComplete
		}
		return readErrorFail
	}
	return readErrorFail
}

func (transformer *textToSpeech) dropConnection(conn *websocket.Conn) {
	transformer.mu.Lock()
	if transformer.connection == conn {
		transformer.connection = nil
		transformer.currentContext = ""
		transformer.connectedAt = time.Time{}
		transformer.turnStartedAt = time.Time{}
		transformer.metricEmitted = false
	}
	transformer.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
}

func (transformer *textToSpeech) emitFirstAudioMetric(contextID string) {
	transformer.mu.Lock()
	startedAt := transformer.turnStartedAt
	alreadySent := transformer.metricEmitted
	if !alreadySent && !startedAt.IsZero() {
		transformer.metricEmitted = true
	}
	transformer.mu.Unlock()

	if alreadySent || startedAt.IsZero() {
		return
	}

	transformer.emitPackets(internal_type.AssistantMessageMetricPacket{
		ContextID: contextID,
		Metrics: []*protos.Metric{{
			Name:  "tts_latency_ms",
			Value: fmt.Sprintf("%d", time.Since(startedAt).Milliseconds()),
		}},
	})
}

func (transformer *textToSpeech) emitTTSError(contextID string, err error, errorType internal_type.TTSErrorType) {
	transformer.emitPackets(internal_type.TextToSpeechErrorPacket{
		ContextID: contextID,
		Error:     err,
		Type:      errorType,
	})
}

func (transformer *textToSpeech) emitPackets(packets ...internal_type.Packet) {
	if transformer.onPacket == nil || len(packets) == 0 {
		return
	}
	if err := transformer.onPacket(packets...); err != nil {
		transformer.logger.Errorf("custom-tts websocket_v1: onPacket failed: %v", err)
	}
}
