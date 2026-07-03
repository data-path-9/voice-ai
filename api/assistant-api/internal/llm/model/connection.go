// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_llm_model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
)

type ModelConnection struct {
	providerName string

	mu     sync.RWMutex
	sendMu sync.Mutex
	stream grpc.BidiStreamingClient[protos.StreamChatRequest, protos.StreamChatResponse]
}

func NewModelConnection(providerName string) *ModelConnection {
	return &ModelConnection{providerName: providerName}
}

func (c *ModelConnection) OpenStream(ctx context.Context, communication internal_type.Communication) error {
	c.mu.RLock()
	existingStream := c.stream
	c.mu.RUnlock()
	if validator.NonNil(existingStream) {
		return fmt.Errorf("%w for %s: %w", ErrModelConnectionOpenStream, c.providerName, ErrModelConnectionStreamAlreadyOpen)
	}
	if !validator.NonNil(communication) || !validator.NonNil(communication.IntegrationCaller()) {
		return fmt.Errorf("%w for %s: %w", ErrModelConnectionOpenStream, c.providerName, ErrModelConnectionNotConnected)
	}

	stream, err := communication.IntegrationCaller().StreamChat(ctx, communication.Auth(), c.providerName)
	if err != nil {
		return fmt.Errorf("%w for %s: %w", ErrModelConnectionOpenStream, c.providerName, err)
	}
	if !validator.NonNil(stream) {
		return fmt.Errorf("%w for %s: %w", ErrModelConnectionOpenStream, c.providerName, ErrModelConnectionNotConnected)
	}

	c.mu.Lock()
	if validator.NonNil(c.stream) {
		c.mu.Unlock()
		_ = stream.CloseSend()
		return fmt.Errorf("%w for %s: %w", ErrModelConnectionOpenStream, c.providerName, ErrModelConnectionStreamAlreadyOpen)
	}
	c.stream = stream
	c.mu.Unlock()
	return nil
}

func (c *ModelConnection) Send(req *protos.StreamChatRequest) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.mu.RLock()
	stream := c.stream
	c.mu.RUnlock()
	if !validator.NonNil(stream) {
		return fmt.Errorf("%w for %s: %w", ErrModelConnectionSend, c.providerName, ErrModelConnectionNotConnected)
	}
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("%w for %s: %w", ErrModelConnectionSend, c.providerName, err)
	}
	return nil
}

func (c *ModelConnection) Recv() (*protos.StreamChatResponse, error) {
	c.mu.RLock()
	stream := c.stream
	c.mu.RUnlock()
	if !validator.NonNil(stream) {
		return nil, fmt.Errorf("%w for %s: %w", ErrModelConnectionRecv, c.providerName, ErrModelConnectionNotConnected)
	}
	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("%w for %s: %w", ErrModelConnectionRecv, c.providerName, err)
	}
	return response, nil
}

func (c *ModelConnection) Close(reason string) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.mu.Lock()
	stream := c.stream
	c.stream = nil
	c.mu.Unlock()
	if !validator.NonNil(stream) {
		return nil
	}

	var closeErrors []error
	if strings.TrimSpace(reason) != "" {
		if err := stream.Send(&protos.StreamChatRequest{
			Request: &protos.StreamChatRequest_Close{
				Close: &protos.StreamChatClose{Reason: reason},
			},
		}); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("%w for %s: %w", ErrModelConnectionCloseRequest, c.providerName, err))
		}
	}
	if err := stream.CloseSend(); err != nil {
		closeErrors = append(closeErrors, fmt.Errorf("%w for %s: %w", ErrModelConnectionCloseStream, c.providerName, err))
	}
	return errors.Join(closeErrors...)
}
