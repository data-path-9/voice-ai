// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package channel_webrtc

import (
	"errors"
	"fmt"
	"io"

	webrtc_internal "github.com/rapidaai/api/assistant-api/internal/channel/webrtc/internal"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// buildGRPCResponse wraps stream messages for WebTalk gRPC.
func (s *webrtcStreamer) buildGRPCResponse(msg internal_type.Stream) *protos.WebTalkResponse {
	resp := &protos.WebTalkResponse{Code: webrtc_internal.WebTalkSuccessCode, Success: true}
	switch m := msg.(type) {
	case *protos.ConversationAssistantMessage:
		resp.Data = &protos.WebTalkResponse_Assistant{Assistant: m}
	case *protos.ConversationConfiguration:
		resp.Data = &protos.WebTalkResponse_Configuration{Configuration: m}
	case *protos.ConversationInitialization:
		resp.Data = &protos.WebTalkResponse_Initialization{Initialization: m}
	case *protos.ConversationUserMessage:
		resp.Data = &protos.WebTalkResponse_User{User: m}
	case *protos.ConversationInterruption:
		resp.Data = &protos.WebTalkResponse_Interruption{Interruption: m}
	case *protos.ConversationToolCall:
		resp.Data = &protos.WebTalkResponse_ToolCall{ToolCall: m}
	case *protos.ConversationDisconnection:
		resp.Data = &protos.WebTalkResponse_Disconnection{Disconnection: m}
	case *protos.ConversationError:
		resp.Data = &protos.WebTalkResponse_Error{Error: m}
	case *protos.ConversationEvent:
		resp.Data = &protos.WebTalkResponse_Event{Event: m}
	case *protos.ConversationMetadata:
		resp.Data = &protos.WebTalkResponse_Metadata{Metadata: m}
	case *protos.ConversationMetric:
		resp.Data = &protos.WebTalkResponse_Metric{Metric: m}
	case *protos.ServerSignaling:
		resp.Data = &protos.WebTalkResponse_Signaling{Signaling: m}
	default:
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC output message skipped",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataType:      fmt.Sprintf("%T", msg),
			},
		})
		return nil
	}
	return resp
}

// dispatchOutput writes a WebTalk response to the client stream.
func (s *webrtcStreamer) dispatchOutput(resp *protos.WebTalkResponse) bool {
	if err := s.grpcStream.Send(resp); err != nil {
		if s.Ctx.Err() != nil || errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled || status.Code(err) == codes.Unavailable {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelInfo,
				Message: "WebRTC gRPC stream closed during send",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					"grpc_code":                   status.Code(err).String(),
					"error":                       err.Error(),
				},
			})
		} else {
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelError,
				Message: "Failed to send WebRTC gRPC response",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					"grpc_code":                   status.Code(err).String(),
					"error":                       err.Error(),
				},
			})
		}
		if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
			s.Input(disc)
		}
		s.Close()
		return false
	}
	return true
}

// runGrpcReader routes client gRPC messages into the conversation stream.
func (s *webrtcStreamer) runGrpcReader() {
	for {
		msg, err := s.grpcStream.Recv()
		if err != nil {
			if s.Ctx.Err() != nil || errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled || status.Code(err) == codes.Unavailable {
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
					Level:   observability.LevelInfo,
					Message: "WebRTC gRPC stream closed",
					Attributes: observability.Attributes{
						"component":                   observability.ComponentWebRTC.String(),
						webrtc_internal.DataSessionID: s.sessionID,
						"grpc_code":                   status.Code(err).String(),
						"error":                       err.Error(),
					},
				})
			} else {
				_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
					Level:   observability.LevelError,
					Message: "WebRTC gRPC receive failed",
					Attributes: observability.Attributes{
						"component":                   observability.ComponentWebRTC.String(),
						webrtc_internal.DataSessionID: s.sessionID,
						"grpc_code":                   status.Code(err).String(),
						"error":                       err.Error(),
					},
				})
			}
			if disc := s.Disconnect(protos.ConversationDisconnection_DISCONNECTION_TYPE_USER); disc != nil {
				s.Input(disc)
			}
			s.Close()
			return
		}
		_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
			Level:   observability.LevelDebug,
			Message: "WebRTC gRPC request received",
			Attributes: observability.Attributes{
				"component":                   observability.ComponentWebRTC.String(),
				webrtc_internal.DataSessionID: s.sessionID,
				webrtc_internal.DataType:      fmt.Sprintf("%T", msg.GetRequest()),
			},
		})
		switch msg.GetRequest().(type) {
		case *protos.WebTalkRequest_Initialization:
			s.Input(msg.GetInitialization())
		case *protos.WebTalkRequest_Configuration:
			s.Input(msg.GetConfiguration())
		case *protos.WebTalkRequest_Message:
			s.Input(msg.GetMessage())
		case *protos.WebTalkRequest_Metadata:
			s.Input(msg.GetMetadata())
		case *protos.WebTalkRequest_Metric:
			s.Input(msg.GetMetric())
		case *protos.WebTalkRequest_ToolCallResult:
			s.Input(msg.GetToolCallResult())
		case *protos.WebTalkRequest_Disconnection:
			if disc := s.Disconnect(msg.GetDisconnection().GetType()); disc != nil {
				s.Input(disc)
			}
		case *protos.WebTalkRequest_Signaling:
			s.queueClientSignal(msg.GetSignaling())
		default:
			_ = s.observer.Record(s.Ctx, s.sessionState.Scope, observability.RecordLog{
				Level:   observability.LevelDebug,
				Message: "Unknown WebRTC gRPC request type",
				Attributes: observability.Attributes{
					"component":                   observability.ComponentWebRTC.String(),
					webrtc_internal.DataSessionID: s.sessionID,
					webrtc_internal.DataType:      fmt.Sprintf("%T", msg.GetRequest()),
				},
			})
		}
	}
}
