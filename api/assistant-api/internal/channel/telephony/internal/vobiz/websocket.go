// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_vobiz_telephony

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_vobiz "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/vobiz/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type vobizWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer
	mediaSession *internal_telephony_media.MediaSession
	streamID     string
	connection   *websocket.Conn
	logger       commons.Logger
	writeMu      sync.Mutex
	closed       atomic.Bool
}

type StreamerOptions struct {
	Logger          commons.Logger
	Connection      *websocket.Conn
	CallContext     *callcontext.CallContext
	VaultCredential *protos.VaultCredential
	Observer        observability.Recorder
}

type FuncOption func(*StreamerOptions)

func WithLogger(logger commons.Logger) FuncOption {
	return func(o *StreamerOptions) { o.Logger = logger }
}
func WithConnection(connection *websocket.Conn) FuncOption {
	return func(o *StreamerOptions) { o.Connection = connection }
}
func WithCallContext(callContext *callcontext.CallContext) FuncOption {
	return func(o *StreamerOptions) { o.CallContext = callContext }
}
func WithVaultCredential(vaultCredential *protos.VaultCredential) FuncOption {
	return func(o *StreamerOptions) { o.VaultCredential = vaultCredential }
}
func WithObserver(observer observability.Recorder) FuncOption {
	return func(o *StreamerOptions) { o.Observer = observer }
}

func New(opts ...FuncOption) (internal_type.Streamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	audioProcessor, err := internal_vobiz.NewAudioProcessor(options.Logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_vobiz.ErrAudioProcessorInitFailed, err)
	}
	vws := &vobizWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.New(
			options.Logger, options.CallContext, options.VaultCredential, options.Observer,
		),
		connection: options.Connection,
		logger:     options.Logger,
	}
	vws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     vws.Ctx,
		Logger:      options.Logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return vws.sendControl(internal_vobiz.EventTypeClearAudio)
		},
		StreamSink: vws.Input,
		OutputSink: vws.sendOutputFrame,
		Record:     vws.Record,
	})
	go vws.runWebSocketReader()
	return vws, nil
}

func (vws *vobizWebsocketStreamer) runWebSocketReader() {
	conn := vws.connection
	if conn == nil {
		return
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			// Vobiz sends no JSON `stop` — the WebSocket close is the
			// authoritative end-of-stream signal.
			if vws.logger != nil {
				vws.logger.Debugf("vobiz: websocket reader closed: %v", err)
			}
			if msg := vws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				vws.Input(msg)
			}
			vws.Cancel()
			return
		}
		var ev internal_vobiz.VobizMediaEvent
		if err := json.Unmarshal(message, &ev); err != nil {
			if vws.logger != nil {
				vws.logger.Errorf("vobiz: failed to unmarshal ws event: %v", err)
			}
			continue
		}
		switch ev.Event {
		case internal_vobiz.EventTypeStart:
			if ev.Start != nil && ev.Start.StreamId != "" {
				vws.streamID = ev.Start.StreamId
			} else {
				vws.streamID = ev.StreamId
			}
			if vws.mediaSession != nil {
				vws.mediaSession.Start()
			}
			vws.Input(vws.CreateConnectionRequest())
		case internal_vobiz.EventTypeMedia:
			if err := vws.handleMediaEvent(ev); err != nil && vws.logger != nil {
				vws.logger.Errorf("vobiz: media frame processing failed: %v", err)
			}
		case internal_vobiz.EventTypePlayedStream, internal_vobiz.EventTypeClearedAudio:
			// playback / clear acknowledgements — no action needed.
		default:
			if vws.logger != nil {
				vws.logger.Debugf("vobiz: unhandled ws event %q", ev.Event)
			}
		}
	}
}

func (vws *vobizWebsocketStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if vws.mediaSession != nil {
			vws.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if vws.mediaSession == nil {
				return nil
			}
			return vws.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted())
		}
	case *protos.ConversationInterruption:
		if data.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
			if vws.mediaSession != nil {
				vws.mediaSession.HandleInterrupt()
			}
		}
	case *protos.ConversationDisconnection:
		_ = vws.Disconnect(data.GetType())
		_ = vws.sendControl(internal_vobiz.EventTypeStop)
		vws.stopAudioProcessing()
		vws.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			_ = vws.sendControl(internal_vobiz.EventTypeStop)
			vws.Input(&protos.ConversationToolCallResult{
				Id: data.GetId(), ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
				Result: map[string]string{"status": "completed"},
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Mid-call transfer is not supported over the vobiz websocket stream.
			vws.Input(&protos.ConversationToolCallResult{
				Id: data.GetId(), ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported on vobiz websocket", "next_action": "end_call"},
			})
		}
	}
	return nil
}

func (vws *vobizWebsocketStreamer) handleMediaEvent(ev internal_vobiz.VobizMediaEvent) error {
	if ev.Media == nil {
		return nil
	}
	payloadBytes, err := vws.Encoder().DecodeString(ev.Media.Payload)
	if err != nil {
		return err
	}
	if vws.mediaSession == nil {
		return nil
	}
	return vws.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      payloadBytes,
		ReceivedAt: time.Now(),
	})
}

func (vws *vobizWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return vws.sendPlayAudio(vws.Encoder().EncodeToString(frame.ProviderAudio))
}

func (vws *vobizWebsocketStreamer) sendPlayAudio(payload string) error {
	if vws.streamID == "" {
		return nil
	}
	msg, err := json.Marshal(internal_vobiz.VobizPlayAudioMessage{
		Event: internal_vobiz.EventTypePlayAudio,
		Media: internal_vobiz.VobizOutboundMedia{
			ContentType: internal_vobiz.OutputContentType,
			SampleRate:  internal_vobiz.OutputSampleRate,
			Payload:     payload,
		},
	})
	if err != nil {
		return err
	}
	return vws.writeMessage(msg)
}

func (vws *vobizWebsocketStreamer) sendControl(event internal_vobiz.EventType) error {
	if vws.streamID == "" {
		return nil
	}
	msg, err := json.Marshal(internal_vobiz.VobizControlMessage{Event: event, StreamID: vws.streamID})
	if err != nil {
		return err
	}
	return vws.writeMessage(msg)
}

func (vws *vobizWebsocketStreamer) writeMessage(msg []byte) error {
	vws.writeMu.Lock()
	defer vws.writeMu.Unlock()
	if vws.connection == nil {
		return nil
	}
	return vws.connection.WriteMessage(websocket.TextMessage, msg)
}

func (vws *vobizWebsocketStreamer) stopAudioProcessing() {
	if vws.mediaSession != nil {
		vws.mediaSession.Shutdown()
	}
}

func (vws *vobizWebsocketStreamer) Cancel() error {
	if !vws.closed.CompareAndSwap(false, true) {
		return nil
	}
	vws.stopAudioProcessing()
	vws.writeMu.Lock()
	conn := vws.connection
	vws.connection = nil
	vws.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	vws.BaseStreamer.Cancel()
	return nil
}
