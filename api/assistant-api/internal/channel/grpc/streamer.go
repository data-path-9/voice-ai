// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package channel_grpc

import (
	"context"

	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

type unidirectionalStreamer struct {
	ctx      context.Context
	logger   commons.Logger
	server   grpc.BidiStreamingServer[protos.AssistantTalkRequest, protos.AssistantTalkResponse]
	observer observability.Recorder
}

type StreamerOptions struct {
	Context  context.Context
	Logger   commons.Logger
	Server   protos.TalkService_AssistantTalkServer
	Observer observability.Recorder
}

type FuncOption func(*StreamerOptions)

func WithContext(ctx context.Context) FuncOption {
	return func(options *StreamerOptions) {
		options.Context = ctx
	}
}

func WithLogger(logger commons.Logger) FuncOption {
	return func(options *StreamerOptions) {
		options.Logger = logger
	}
}

func WithServer(server protos.TalkService_AssistantTalkServer) FuncOption {
	return func(options *StreamerOptions) {
		options.Server = server
	}
}

func WithObserver(observer observability.Recorder) FuncOption {
	return func(options *StreamerOptions) {
		options.Observer = observer
	}
}

func New(opts ...FuncOption) (internal_type.Streamer, error) {
	var options StreamerOptions
	for _, opt := range opts {
		opt(&options)
	}
	return &unidirectionalStreamer{
		ctx:      options.Context,
		logger:   options.Logger,
		server:   options.Server,
		observer: options.Observer,
	}, nil
}

func (uds *unidirectionalStreamer) Context() context.Context {
	return uds.ctx
}

func (uds *unidirectionalStreamer) Observer() observability.Recorder {
	return uds.observer
}

// NotifyMode is a no-op for the plain gRPC streamer (audio transport is N/A).
func (uds *unidirectionalStreamer) NotifyMode(_ protos.StreamMode) {}

func (uds *unidirectionalStreamer) Recv() (internal_type.Stream, error) {
	req, err := uds.server.Recv()
	if err != nil {
		return nil, err
	}
	switch in := req.Request.(type) {
	case *protos.AssistantTalkRequest_Initialization:
		return in.Initialization, nil
	case *protos.AssistantTalkRequest_Configuration:
		return in.Configuration, nil
	case *protos.AssistantTalkRequest_Message:
		return in.Message, nil
	case *protos.AssistantTalkRequest_Metadata:
		return in.Metadata, nil
	case *protos.AssistantTalkRequest_Metric:
		return in.Metric, nil
	}
	return nil, nil
}

// Send sends an output value to the stream.
// It returns an error if the send operation fails.

func (uds *unidirectionalStreamer) Send(out internal_type.Stream) error {
	switch out := out.(type) {
	case *protos.ConversationInitialization:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_Initialization{Initialization: out},
		})

	case *protos.ConversationConfiguration:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_Configuration{Configuration: out},
		})

	case *protos.ConversationInterruption:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_Interruption{Interruption: out},
		})

	case *protos.ConversationUserMessage:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_User{User: out},
		})

	case *protos.ConversationAssistantMessage:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_Assistant{Assistant: out},
		})

	case *protos.ConversationToolCall:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_ToolCall{ToolCall: out},
		})

	case *protos.ConversationToolCallResult:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_ToolCallResult{ToolCallResult: out},
		})

	case *protos.ConversationMetadata:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_Metadata{Metadata: out},
		})

	case *protos.ConversationMetric:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_Metric{Metric: out},
		})

	case *protos.ConversationError:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    500,
			Success: false,
			Data:    &protos.AssistantTalkResponse_Error{Error: out},
		})

	case *protos.ConversationEvent:
		return uds.server.Send(&protos.AssistantTalkResponse{
			Code:    200,
			Success: true,
			Data:    &protos.AssistantTalkResponse_Event{Event: out},
		})
	}
	return nil
}
