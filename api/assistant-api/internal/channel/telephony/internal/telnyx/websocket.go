// Copyright (c) 2023-2025 RapidaAI
// Author: RapidaAI Team <team@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_telnyx_telephony

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
	internal_telnyx "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/telnyx/internal"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type telnyxWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer

	mediaSession *internal_telephony_media.MediaSession

	streamID      string
	callControlID string
	connection    *websocket.Conn
	writeMu       sync.Mutex
	closed        atomic.Bool
	telephony     *telnyxTelephony
}

// NewTelnyxWebsocketStreamer creates a Telnyx WebSocket streamer.
// Telnyx sends PCMU 8kHz, matching Twilio's provider audio format.
func NewTelnyxWebsocketStreamer(logger commons.Logger, connection *websocket.Conn, cc *callcontext.CallContext, vaultCred *protos.VaultCredential) (internal_type.Streamer, error) {
	audioProcessor, err := internal_telnyx.NewAudioProcessor(logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_telnyx.ErrAudioProcessorInitFailed, err)
	}

	tws := &telnyxWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
			logger, cc, vaultCred,
		),
		streamID:   "",
		connection: connection,
		telephony: &telnyxTelephony{
			logger: logger,
		},
	}

	tws.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     tws.Ctx,
		Logger:      logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return tws.sendTelnyxMessage(internal_telnyx.EventTypeClear, nil)
		},
		StreamSink: tws.Input,
		OutputSink: tws.sendOutputFrame,
		EventSink: func(event *protos.ConversationEvent) {
			if event != nil {
				if event.Data == nil {
					event.Data = map[string]string{}
				}
				event.Data["provider"] = internal_telnyx.Provider
			}
			tws.Input(event)
		},
	})

	go tws.runWebSocketReader()
	return tws, nil
}

func (tws *telnyxWebsocketStreamer) runWebSocketReader() {
	conn := tws.connection
	if conn == nil {
		return
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			tws.stopAudioProcessing()
			if msg := tws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				tws.Input(msg)
			}
			tws.BaseStreamer.Cancel()
			return
		}

		if messageType != websocket.TextMessage {
			tws.Logger.Warn("Unhandled message type", "type", messageType)
			continue
		}

		var mediaEvent internal_telnyx.TelnyxWebSocketEvent
		if err := json.Unmarshal(message, &mediaEvent); err != nil {
			tws.Logger.Error("Failed to unmarshal Telnyx media event", "error", err.Error())
			continue
		}

		switch mediaEvent.Event {
		case internal_telnyx.EventTypeConnected:
			tws.Input(&protos.ConversationEvent{
				Name: "channel",
				Data: map[string]string{"type": internal_telnyx.ChannelEventConnected, "provider": internal_telnyx.Provider},
				Time: timestamppb.Now(),
			})
		case internal_telnyx.EventTypeStart:
			tws.handleStartEvent(mediaEvent)
			if tws.mediaSession != nil {
				tws.mediaSession.Start()
			}
			tws.Input(tws.CreateConnectionRequest())
			tws.Input(&protos.ConversationEvent{
				Name: "channel",
				Data: map[string]string{
					"type":            internal_telnyx.ChannelEventStreamStarted,
					"provider":        internal_telnyx.Provider,
					"stream_id":       tws.streamID,
					"call_control_id": tws.callControlID,
				},
				Time: timestamppb.Now(),
			})
		case internal_telnyx.EventTypeMedia:
			if err := tws.handleMediaEvent(mediaEvent); err != nil {
				tws.Logger.Errorw("Failed to process Telnyx media frame",
					"error", err,
					"stream_id", tws.streamID,
					"call_control_id", tws.callControlID,
					"conversation_uuid", tws.GetConversationUuid(),
				)
			}
		case internal_telnyx.EventTypeDTMF:
			tws.Input(&protos.ConversationEvent{
				Name: "channel",
				Data: map[string]string{"type": internal_telnyx.ChannelEventDTMF, "provider": internal_telnyx.Provider},
				Time: timestamppb.Now(),
			})
		case internal_telnyx.EventTypeStop:
			tws.Logger.Info("Telnyx stream stopped")
			if msg := tws.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				tws.Input(msg)
			}
			tws.Cancel()
			return
		default:
			tws.Logger.Warn("Unhandled Telnyx event", "event", mediaEvent.Event)
		}
	}
}

func (tws *telnyxWebsocketStreamer) Send(response internal_type.Stream) error {
	if tws.connection == nil {
		return nil
	}
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if tws.mediaSession != nil {
			tws.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if tws.mediaSession == nil {
				return nil
			}
			if err := tws.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				return err
			}
			return nil
		}
	case *protos.ConversationInterruption:
		if data.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
			if tws.mediaSession != nil {
				tws.mediaSession.HandleInterrupt()
			}
		}
	case *protos.ConversationDisconnection:
		_ = tws.Disconnect(data.GetType())
		if tws.GetConversationUuid() != "" {
			if err := tws.telephony.HangupCall(tws.GetConversationUuid(), tws.VaultCredential()); err != nil {
				tws.Logger.Errorf("Error ending Telnyx call: %v", err)
			}
		}
		tws.stopAudioProcessing()
		tws.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			result := map[string]string{"status": "completed"}
			if tws.GetConversationUuid() != "" {
				if err := tws.telephony.HangupCall(tws.GetConversationUuid(), tws.VaultCredential()); err != nil {
					tws.Logger.Errorf("Error ending Telnyx call: %v", err)
					result = map[string]string{"status": "failed", "reason": fmt.Sprintf("hangup failed: %v", err)}
				}
			}
			tws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: result,
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			tws.Logger.Warnw("Telnyx call transfer not yet implemented", "transfer_to", data.GetArgs()["transfer_to"])
			tws.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(), Name: data.GetName(), Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for Telnyx", "next_action": "end_call"},
			})
		}
	default:
		tws.Logger.Warnw("Telnyx Send: unknown message type, skipping", "type", fmt.Sprintf("%T", response))
	}
	return nil
}

func (tws *telnyxWebsocketStreamer) handleStartEvent(mediaEvent internal_telnyx.TelnyxWebSocketEvent) {
	tws.streamID = mediaEvent.StreamID
	if mediaEvent.Start == nil {
		return
	}
	tws.callControlID = mediaEvent.Start.CallControlID
	tws.ChannelUUID = mediaEvent.Start.CallControlID
}

func (tws *telnyxWebsocketStreamer) handleMediaEvent(mediaEvent internal_telnyx.TelnyxWebSocketEvent) error {
	if mediaEvent.Media == nil {
		return nil
	}
	receivedAt := time.Now()
	payloadBytes, err := tws.Encoder().DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		tws.Logger.Warn("Failed to decode media payload", "error", err.Error())
		return nil
	}

	if tws.mediaSession == nil {
		return nil
	}
	if err := tws.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      payloadBytes,
		ReceivedAt: receivedAt,
	}); err != nil {
		return err
	}
	return nil
}

func (tws *telnyxWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return tws.sendTelnyxMessage(internal_telnyx.EventTypeMedia, &internal_telnyx.TelnyxOutboundMedia{
		Payload: tws.Encoder().EncodeToString(frame.ProviderAudio),
	})
}

func (tws *telnyxWebsocketStreamer) sendTelnyxMessage(eventType internal_telnyx.EventType, mediaData *internal_telnyx.TelnyxOutboundMedia) error {
	if tws.connection == nil || tws.streamID == "" {
		return nil
	}
	message := internal_telnyx.TelnyxOutboundMessage{
		Event:    eventType,
		StreamID: tws.streamID,
		Media:    mediaData,
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		tws.Logger.Error("Failed to marshal Telnyx message", "error", err.Error())
		return err
	}

	tws.writeMu.Lock()
	defer tws.writeMu.Unlock()
	if tws.connection == nil {
		return nil
	}
	if err := tws.connection.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
		tws.Logger.Error("Failed to send message to Telnyx", "error", err.Error())
		return err
	}
	return nil
}

func (tws *telnyxWebsocketStreamer) stopAudioProcessing() {
	if tws.mediaSession != nil {
		tws.mediaSession.Shutdown()
	}
}

func (tws *telnyxWebsocketStreamer) GetConversationUuid() string {
	return tws.ChannelUUID
}

func (tws *telnyxWebsocketStreamer) Cancel() error {
	if !tws.closed.CompareAndSwap(false, true) {
		return nil
	}
	tws.stopAudioProcessing()
	tws.writeMu.Lock()
	conn := tws.connection
	tws.connection = nil
	tws.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	tws.BaseStreamer.Cancel()
	return nil
}
