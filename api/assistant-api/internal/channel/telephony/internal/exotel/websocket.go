// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_exotel_telephony

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	callcontext "github.com/rapidaai/api/assistant-api/internal/callcontext"
	internal_telephony_base "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/base"
	internal_exotel "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/exotel/internal"
	internal_telephony_media "github.com/rapidaai/api/assistant-api/internal/channel/telephony/internal/media"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type exotelWebsocketStreamer struct {
	internal_telephony_base.BaseTelephonyStreamer
	mediaSession *internal_telephony_media.MediaSession
	connection   *websocket.Conn
	writeMu      sync.Mutex
	closed       atomic.Bool
	streamID     string
}

func NewExotelWebsocketStreamer(logger commons.Logger, connection *websocket.Conn, cc *callcontext.CallContext, vaultCred *protos.VaultCredential,
) (internal_type.Streamer, error) {
	audioProcessor, err := internal_exotel.NewAudioProcessor(logger)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", internal_exotel.ErrAudioProcessorInitFailed, err)
	}
	exotel := &exotelWebsocketStreamer{
		BaseTelephonyStreamer: internal_telephony_base.NewBaseTelephonyStreamer(
			logger, cc, vaultCred,
		),
		streamID:   "",
		connection: connection,
	}
	exotel.mediaSession = internal_telephony_media.NewMediaSession(internal_telephony_media.MediaSessionConfig{
		Context:     exotel.Ctx,
		Logger:      logger,
		MediaEngine: audioProcessor,
		SendProviderClear: func() error {
			return exotel.sendExotelMessage(internal_exotel.EventTypeClear, nil)
		},
		StreamSink: exotel.Input,
		OutputSink: exotel.sendOutputFrame,
		EventSink: func(event *protos.ConversationEvent) {
			if event != nil {
				if event.Data == nil {
					event.Data = map[string]string{}
				}
				event.Data["provider"] = internal_exotel.Provider
			}
			exotel.Input(event)
		},
	})
	go exotel.runWebSocketReader()
	return exotel, nil
}

func (exotel *exotelWebsocketStreamer) runWebSocketReader() {
	conn := exotel.connection
	if conn == nil {
		return
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			exotel.stopAudioProcessing()
			if msg := exotel.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				exotel.Input(msg)
			}
			exotel.BaseStreamer.Cancel()
			return
		}
		var mediaEvent internal_exotel.ExotelMediaEvent
		if err := json.Unmarshal(message, &mediaEvent); err != nil {
			exotel.Logger.Error("Failed to unmarshal Exotel media event", "error", err.Error())
			continue
		}
		switch mediaEvent.Event {
		case internal_exotel.EventTypeConnected:
			exotel.Input(exotel.CreateConnectionRequest())
			exotel.Input(&protos.ConversationEvent{
				Name: "channel",
				Data: map[string]string{"type": internal_exotel.ChannelEventConnected, "provider": internal_exotel.Provider},
				Time: timestamppb.Now(),
			})
		case internal_exotel.EventTypeStart:
			exotel.handleStartEvent(mediaEvent)
			if exotel.mediaSession != nil {
				exotel.mediaSession.Start()
			}
			exotel.Input(&protos.ConversationEvent{
				Name: "channel",
				Data: map[string]string{"type": internal_exotel.ChannelEventStreamStarted, "provider": internal_exotel.Provider, "stream_id": exotel.streamID},
				Time: timestamppb.Now(),
			})
		case internal_exotel.EventTypeMedia:
			if err := exotel.handleMediaEvent(mediaEvent); err != nil {
				exotel.Logger.Errorw("Failed to process Exotel media frame",
					"error", err,
					"stream_id", exotel.streamID,
					"conversation_uuid", exotel.ChannelUUID,
				)
			}
		case internal_exotel.EventTypeDTMF:
			exotel.Input(&protos.ConversationEvent{
				Name: "channel",
				Data: map[string]string{"type": internal_exotel.ChannelEventDTMF, "provider": internal_exotel.Provider},
				Time: timestamppb.Now(),
			})
		case internal_exotel.EventTypeStop:
			if msg := exotel.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); msg != nil {
				exotel.Input(msg)
			}
			exotel.Cancel()
			return
		default:
			exotel.Logger.Warn("Unhandled Exotel event", "event", mediaEvent.Event)
		}
	}
}

func (exotel *exotelWebsocketStreamer) Send(response internal_type.Stream) error {
	switch data := response.(type) {
	case *protos.ConversationInitialization:
		if exotel.mediaSession != nil {
			exotel.mediaSession.HandleInitialization(data)
		}
	case *protos.ConversationAssistantMessage:
		switch content := data.Message.(type) {
		case *protos.ConversationAssistantMessage_Audio:
			if exotel.mediaSession == nil {
				return nil
			}
			if err := exotel.mediaSession.HandleAssistantAudio(content.Audio, data.GetCompleted()); err != nil {
				return err
			}
			return nil
		}
	case *protos.ConversationInterruption:
		if data.Type == protos.ConversationInterruption_INTERRUPTION_TYPE_WORD {
			if exotel.mediaSession != nil {
				exotel.mediaSession.HandleInterrupt()
			}
		}
	case *protos.ConversationDisconnection:
		// Server-initiated disconnect: the talker already knows the reason
		// (it called Notify with it). No need to round-trip back through
		// CriticalCh — Exotel has no REST API to terminate a call; closing
		// the WebSocket via Cancel is the only way to release the call leg.
		_ = exotel.Disconnect(data.GetType())
		exotel.stopAudioProcessing()
		exotel.Cancel()
	case *protos.ConversationToolCall:
		switch data.GetAction() {
		case protos.ToolCallAction_TOOL_CALL_ACTION_END_CONVERSATION:
			exotel.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "completed"},
			})
		case protos.ToolCallAction_TOOL_CALL_ACTION_TRANSFER_CONVERSATION:
			// Exotel transfer is NOT supported. Exotel exposes call-flow level
			// "Connect" applets but no live mid-call transfer API on the
			// streaming WebSocket leg. A blind transfer would require building
			// an out-of-band Connect/Dial app and redirecting via the Exotel
			// HTTP API; resume_ai is not feasible without a B2BUA bridge
			// (Exotel does not provide an SDP/RTP path to bridge against).
			exotel.Logger.Warnw("Call transfer not supported for Exotel")
			exotel.Input(&protos.ConversationToolCallResult{
				Id:     data.GetId(),
				ToolId: data.GetToolId(),
				Name:   data.GetName(),
				Action: data.GetAction(),
				Result: map[string]string{"status": "failed", "reason": "transfer not supported for Exotel", "next_action": "end_call"},
			})
		}
	default:
		exotel.Logger.Warnw("Exotel Send: unknown message type, skipping", "type", fmt.Sprintf("%T", response))
	}
	return nil
}

func (exotel *exotelWebsocketStreamer) handleStartEvent(mediaEvent internal_exotel.ExotelMediaEvent) {
	exotel.streamID = mediaEvent.StreamSid
}

func (exotel *exotelWebsocketStreamer) handleMediaEvent(mediaEvent internal_exotel.ExotelMediaEvent) error {
	if mediaEvent.Media == nil {
		exotel.Logger.Warn("Exotel media event missing media payload")
		return nil
	}
	receivedAt := time.Now()
	payloadBytes, err := exotel.Encoder().DecodeString(mediaEvent.Media.Payload)
	if err != nil {
		exotel.Logger.Warn("Failed to decode media payload", "error", err.Error())
		return nil
	}

	if exotel.mediaSession == nil {
		return nil
	}
	if err := exotel.mediaSession.HandleProviderAudioFrame(internal_telephony_media.ProviderAudioFrame{
		Audio:      payloadBytes,
		ReceivedAt: receivedAt,
	}); err != nil {
		return err
	}
	return nil
}

func (exotel *exotelWebsocketStreamer) sendExotelMessage(eventType internal_exotel.EventType, mediaData *internal_exotel.ExotelOutboundMedia) error {
	if exotel.streamID == "" {
		return nil
	}
	message := internal_exotel.ExotelOutboundMessage{
		Event:    eventType,
		StreamID: exotel.streamID,
		Media:    mediaData,
	}
	exotelMessageJSON, err := json.Marshal(message)
	if err != nil {
		exotel.Logger.Error("Failed to marshal Exotel message", "error", err.Error())
		return err
	}
	exotel.writeMu.Lock()
	defer exotel.writeMu.Unlock()
	if exotel.connection == nil {
		return nil
	}
	if err := exotel.connection.WriteMessage(websocket.TextMessage, exotelMessageJSON); err != nil {
		exotel.Logger.Error("Failed to send message to Exotel", "error", err.Error())
		return err
	}
	return nil
}

func (exotel *exotelWebsocketStreamer) Cancel() error {
	if !exotel.closed.CompareAndSwap(false, true) {
		return nil
	}
	exotel.stopAudioProcessing()
	exotel.writeMu.Lock()
	conn := exotel.connection
	exotel.connection = nil
	exotel.writeMu.Unlock()
	if conn != nil {
		conn.Close()
	}
	exotel.BaseStreamer.Cancel()
	return nil
}

func (exotel *exotelWebsocketStreamer) sendOutputFrame(frame internal_telephony_media.AssistantOutputFrame) error {
	if len(frame.ProviderAudio) == 0 {
		return nil
	}
	return exotel.sendExotelMessage(internal_exotel.EventTypeMedia, &internal_exotel.ExotelOutboundMedia{
		Payload: exotel.Encoder().EncodeToString(frame.ProviderAudio),
	})
}

func (exotel *exotelWebsocketStreamer) stopAudioProcessing() {
	if exotel.mediaSession != nil {
		exotel.mediaSession.Shutdown()
	}
}
