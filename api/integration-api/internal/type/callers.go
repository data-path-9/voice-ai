// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_callers

import (
	"context"
	"io"

	"google.golang.org/protobuf/types/known/anypb"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
)

type AICaller struct {
	logger commons.Logger
}

// CredentialResolver defines a function type for resolving credentials.
// It returns a map representing the credentials.
type CredentialResolver = func() map[string]interface{}

// AIOptions holds configuration and hooks for handling an AI request.
// RequestId is a unique identifier for the request.
// PreHook is a function executed before the main AI logic, allowing manipulation of the result map `rst`.
// PostHook is a function executed after the AI processing, passing the result map `rst` and performance metrics `metrics`.
// ModelParameter holds a slice of model parameters used for AI processing, defined by the `protos.ModelParameter` type.
// This struct enables customizable pre- and post-processing of requests and results while carrying relevant AI configuration data.
type AIOptions struct {
	RequestId      uint64
	PreHook        func(rst map[string]interface{})
	PostHook       func(rst map[string]interface{}, metrics []*protos.Metric)
	ModelParameter map[string]*anypb.Any
}

type ModerationOptions struct {
	AIOptions
	Language    *string
	Filename    *string
	Temperature *float32
}

type CompletionOptions struct {
	AIOptions

	Version *string
}

type CredentialVerifierOptions struct {
	AIOptions
}

// Caller is an interface for making HTTP calls.
// - Call: Makes a call with headers and payload represented as a map.
// - CallWithPayload: Allows the payload to be passed as an io.Reader for flexibility in handling larger data.
type Caller interface {
	Call(ctx context.Context, endpoint, method string, headers map[string]string, payload map[string]interface{}) (*string, error)
	CallWithPayload(ctx context.Context, endpoint, method string, headers map[string]string, payload io.Reader) (*string, error)
}

// Verifier is an interface for verifying credentials.
// - CredentialVerifier: Processes credential verification based on given options.
type Verifier interface {
	CredentialVerifier(
		ctx context.Context,
		options *CredentialVerifierOptions) (*string, error)
}

// LargeLanguageCaller handles operations related to large language model interactions.
// - GetChatCompletion: Processes chat completion using allMessages and given options, returning a response message and metrics.
// - StreamChatCompletion: Streams responses for chat completion, allowing user-defined handlers for streaming, metrics, and error monitoring.
type LargeLanguageCaller interface {
	GetChatCompletion(
		ctx context.Context,
		allMessages []*protos.Message,
		options *ChatCompletionOptions,
	) (*protos.Message, []*protos.Metric, error)

	StreamChatCompletion(
		ctx context.Context,
		allMessages []*protos.Message,
		options *ChatCompletionOptions,
		onStream func(rID string, msg *protos.Message) error,
		onMetrics func(rID string, msg *protos.Message, mtrx []*protos.Metric) error,
		onError func(rID string, err error),
	) error
}

// Chat handles unary chat completion.
type Chat interface {
	ChatComplete(
		ctx context.Context,
		allMessages []*protos.Message,
		options *ChatCompletionOptions,
	) (*protos.Message, []*protos.Metric, error)
}

// ChatStream handles stream lifecycle and turn execution.
type ChatStream interface {
	GetCredential() *protos.Credential
	Connect(ctx context.Context, configuration *protos.StreamChatConfiguration) error
	Chat(
		ctx context.Context,
		allMessages []*protos.Message,
		options *ChatStreamCompletionOptions,
		onStream func(rID string, msg *protos.Message) error,
		onMetrics func(rID string, msg *protos.Message, mtrx []*protos.Metric) error,
		onError func(rID string, err error),
	) error
	Close(ctx context.Context) error
}

// EmbeddingCaller is an interface for working with embeddings.
// - GetEmbedding: Generates embeddings for the supplied content and options.
type EmbeddingCaller interface {
	GetEmbedding(ctx context.Context,
		content map[int32]string, options *EmbeddingOptions) ([]*protos.Embedding, []*protos.Metric, error)
}

// RerankingCaller is an interface for reranking models.
// - GetReranking: Uses the query and content map to return a reranked result set along with metrics.
type RerankingCaller interface {
	GetReranking(ctx context.Context,
		query string,
		content map[int32]string,
		options *RerankerOptions,
	) ([]*protos.Reranking, []*protos.Metric, error)
}
