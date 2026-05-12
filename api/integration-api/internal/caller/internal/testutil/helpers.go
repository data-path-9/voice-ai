// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package caller_testutil

import (
	"sync"
	"testing"

	internal_types "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
)

// NoopPreHook is a no-op pre-hook for test options.
var NoopPreHook = func(rst map[string]interface{}) {}

// NoopPostHook is a no-op post-hook for test options.
var NoopPostHook = func(rst map[string]interface{}, metrics []*protos.Metric) {}

// SimpleMessages returns a minimal conversation: system + user message.
func SimpleMessages() []*protos.Message {
	return []*protos.Message{
		{
			Role: "system",
			Message: &protos.Message_System{
				System: &protos.SystemMessage{Content: "You are a helpful assistant. Respond briefly."},
			},
		},
		{
			Role: "user",
			Message: &protos.Message_User{
				User: &protos.UserMessage{Content: "Say hello in one sentence."},
			},
		},
	}
}

// BuildChatOptions creates ChatCompletionOptions from a ProviderConfig.
func BuildChatOptions(pcfg ProviderConfig) *internal_types.ChatCompletionOptions {
	modelParams := BuildModelParameters(pcfg.Options)
	return &internal_types.ChatCompletionOptions{
		AIOptions: internal_types.AIOptions{
			RequestId:      1,
			PreHook:        NoopPreHook,
			PostHook:       NoopPostHook,
			ModelParameter: modelParams,
		},
		Request: &protos.ChatRequest{
			RequestId:       "integration-test-1",
			ModelParameters: modelParams,
		},
	}
}

// BuildChatStreamOptions creates ChatStreamCompletionOptions from a ProviderConfig.
func BuildChatStreamOptions(pcfg ProviderConfig) *internal_types.ChatStreamCompletionOptions {
	modelParams := BuildModelParameters(pcfg.Options)
	return &internal_types.ChatStreamCompletionOptions{
		AIOptions: internal_types.AIOptions{
			RequestId:      1,
			PreHook:        NoopPreHook,
			PostHook:       NoopPostHook,
			ModelParameter: modelParams,
		},
		Request: &protos.StreamChatInput{
			RequestId:       "integration-test-1",
			ModelParameters: modelParams,
		},
	}
}

// BuildEmbeddingOptions creates EmbeddingOptions from a ProviderConfig.
func BuildEmbeddingOptions(pcfg ProviderConfig) *internal_types.EmbeddingOptions {
	return &internal_types.EmbeddingOptions{
		AIOptions: internal_types.AIOptions{
			RequestId:      1,
			PreHook:        NoopPreHook,
			PostHook:       NoopPostHook,
			ModelParameter: BuildModelParameters(pcfg.Options),
		},
	}
}

// BuildRerankerOptions creates RerankerOptions from a ProviderConfig.
func BuildRerankerOptions(pcfg ProviderConfig) *internal_types.RerankerOptions {
	return &internal_types.RerankerOptions{
		AIOptions: internal_types.AIOptions{
			RequestId:      1,
			PreHook:        NoopPreHook,
			PostHook:       NoopPostHook,
			ModelParameter: BuildModelParameters(pcfg.Options),
		},
	}
}

// BuildVerifyOptions creates CredentialVerifierOptions from a ProviderConfig.
func BuildVerifyOptions(pcfg ProviderConfig) *internal_types.CredentialVerifierOptions {
	return &internal_types.CredentialVerifierOptions{
		AIOptions: internal_types.AIOptions{
			RequestId:      1,
			ModelParameter: BuildModelParameters(pcfg.Options),
		},
	}
}

// EmbeddingContent returns a standard single-document content map for embedding tests.
func EmbeddingContent() map[int32]string {
	return map[int32]string{
		0: "The quick brown fox jumps over the lazy dog.",
	}
}

// RerankingQuery returns a standard query for reranking tests.
func RerankingQuery() string {
	return "What is the capital of France?"
}

// RerankingContent returns a standard multi-document content map for reranking tests.
func RerankingContent() map[int32]string {
	return map[int32]string{
		0: "Berlin is the capital of Germany.",
		1: "Paris is the capital of France.",
		2: "Madrid is the capital of Spain.",
	}
}

// StreamCollector collects streaming callback results in a thread-safe manner.
type StreamCollector struct {
	mu           sync.Mutex
	StreamCount  int
	MetricsCount int
	StreamErr    error
	Metrics      []*protos.Metric
}

// OnStream is the streaming callback.
func (sc *StreamCollector) OnStream(rID string, msg *protos.Message) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.StreamCount++
	return nil
}

// OnMetrics is the metrics callback.
func (sc *StreamCollector) OnMetrics(rID string, msg *protos.Message, mtrx []*protos.Metric) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.MetricsCount++
	sc.Metrics = mtrx
	return nil
}

// OnError is the error callback.
func (sc *StreamCollector) OnError(rID string, err error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.StreamErr = err
}

// AssertStream verifies streaming expectations.
func (sc *StreamCollector) AssertStream(t *testing.T) {
	t.Helper()
	sc.mu.Lock()
	defer sc.mu.Unlock()
	assert.NoError(t, sc.StreamErr, "onError should not have been called")
	assert.Greater(t, sc.StreamCount, 0, "onStream should have been called at least once")
	assert.Equal(t, 1, sc.MetricsCount, "onMetrics should have been called exactly once")
}

// AssertHasMetric checks that a metric with the given name exists in the slice.
func AssertHasMetric(t *testing.T, metrics []*protos.Metric, name string) {
	t.Helper()
	for _, m := range metrics {
		if m.GetName() == name {
			return
		}
	}
	names := make([]string, len(metrics))
	for i, m := range metrics {
		names[i] = m.GetName()
	}
	t.Errorf("expected metric %q not found in %v", name, names)
}
