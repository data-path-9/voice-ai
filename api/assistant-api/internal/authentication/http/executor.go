// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_authentication_http

import (
	"context"
	"fmt"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/clients/rest"
	"github.com/rapidaai/pkg/commons"
)

const (
	OptionHTTPURLKey     = "http_url"
	OptionHTTPMethodKey  = "http_method"
	OptionHTTPHeadersKey = "http_headers"

	FailBehaviorBlock = "block"
	FailBehaviorAllow = "allow"
)

// Result carries the outcome of an authentication attempt.
type Result struct {
	Authenticated bool
	Args          map[string]interface{}
	Metadata      map[string]interface{}
	Options       map[string]interface{}
}

type runtimeExecutor struct {
	logger   commons.Logger
	callback internal_type.Callback
}

// NewExecutor creates a fully wired HTTP authentication executor.
func NewExecutor(logger commons.Logger, _ context.Context, callback internal_type.Callback, _ internal_type.InternalCaller) (internal_type.AuthenticationExecutor, error) {
	return &runtimeExecutor{
		logger:   logger,
		callback: callback,
	}, nil
}

// Execute runs authentication against the configured endpoint and emits packetized outcome.
func (e *runtimeExecutor) Execute(ctx context.Context, packet internal_type.ExecuteSessionAuthenticationPacket) error {
	auth := packet.Authentication

	url, err := auth.GetOptions().GetString(OptionHTTPURLKey)
	if err != nil || url == "" {
		e.callback.OnPacket(ctx, internal_type.SessionAuthenticationFailedPacket{
			ContextID:      packet.ContextID,
			Error:          fmt.Errorf("authentication: missing %s", OptionHTTPURLKey),
			Initialization: packet.Initialization,
		})
		return nil
	}

	method := "POST"
	if m, err := auth.GetOptions().GetString(OptionHTTPMethodKey); err == nil && m != "" {
		method = m
	}

	headers := map[string]string{}
	if h, err := auth.GetOptions().GetStringMap(OptionHTTPHeadersKey); err == nil {
		headers = h
	}

	timeout := auth.TimeoutMs
	if timeout == 0 {
		timeout = 5000
	}

	client := rest.NewRestClientWithConfig(url, headers, uint32(timeout/1000))

	callCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	response, err := e.send(callCtx, client, method, packet.Arguments, headers)
	if err != nil {
		if auth.FailBehavior == FailBehaviorAllow {
			e.logger.Warnw("authentication failed, allowing due to fail_behavior=allow", "url", url, "error", err)
			e.callback.OnPacket(ctx, internal_type.SessionAuthenticationSucceededPacket{
				ContextID:      packet.ContextID,
				Authenticated:  false,
				Initialization: packet.Initialization,
			})
			return nil
		}
		e.callback.OnPacket(ctx, internal_type.SessionAuthenticationFailedPacket{
			ContextID:      packet.ContextID,
			Error:          fmt.Errorf("authentication: request failed: %w", err),
			Initialization: packet.Initialization,
		})
		return nil
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if auth.FailBehavior == FailBehaviorAllow {
			e.logger.Warnw("authentication returned non-2xx, allowing due to fail_behavior=allow",
				"url", url, "status", response.StatusCode)
			e.callback.OnPacket(ctx, internal_type.SessionAuthenticationSucceededPacket{
				ContextID:      packet.ContextID,
				Authenticated:  false,
				Initialization: packet.Initialization,
			})
			return nil
		}
		e.callback.OnPacket(ctx, internal_type.SessionAuthenticationFailedPacket{
			ContextID:      packet.ContextID,
			Error:          fmt.Errorf("authentication: endpoint returned status %d", response.StatusCode),
			Initialization: packet.Initialization,
		})
		return nil
	}

	result := &Result{Authenticated: true}
	if parsed, err := response.ToMap(); err == nil {
		if args, ok := parsed["args"].(map[string]interface{}); ok {
			result.Args = args
		}
		if metadata, ok := parsed["metadata"].(map[string]interface{}); ok {
			result.Metadata = metadata
		}
		if options, ok := parsed["options"].(map[string]interface{}); ok {
			result.Options = options
		}
	}

	e.callback.OnPacket(ctx, internal_type.SessionAuthenticationSucceededPacket{
		ContextID:      packet.ContextID,
		Authenticated:  result.Authenticated,
		Arguments:      result.Args,
		Metadata:       result.Metadata,
		Options:        result.Options,
		Initialization: packet.Initialization,
	})
	return nil
}

// Close releases executor dependencies.
func (e *runtimeExecutor) Close(_ context.Context) error {
	e.callback = nil
	return nil
}

func (e *runtimeExecutor) send(ctx context.Context, client *rest.RestClient, method string, body map[string]interface{}, headers map[string]string) (*rest.APIResponse, error) {
	switch method {
	case "POST":
		return client.Post(ctx, "", body, headers)
	case "PUT":
		return client.Put(ctx, "", body, headers)
	case "PATCH":
		return client.Patch(ctx, "", body, headers)
	default:
		return client.Get(ctx, "", body, headers)
	}
}
