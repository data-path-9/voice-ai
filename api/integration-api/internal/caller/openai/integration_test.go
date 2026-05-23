// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_callers

import (
	"context"
	"testing"
	"time"

	testutil "github.com/rapidaai/api/integration-api/internal/caller/internal/testutil"
	internal_openai_text_embedding "github.com/rapidaai/api/integration-api/internal/caller/openai/text_embedding"
	internal_openai_verify_credential "github.com/rapidaai/api/integration-api/internal/caller/openai/verify_credential"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const providerName = "openai"

// TestIntegration_ChatCompletion verifies non-streaming chat completion: send a
// simple prompt and assert the assistant responds with content and metrics.
func TestIntegration_ChatCompletion(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.ChatProvider(t, providerName)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	chat, err := NewChat(testutil.NewTestLogger(), cred, nil)
	require.NoError(t, err)
	opts := testutil.BuildChatOptions(pcfg)

	msg, metrics, err := chat.ChatComplete(ctx, testutil.SimpleMessages(), opts)
	require.NoError(t, err, "GetChatCompletion should succeed")
	require.NotNil(t, msg, "response message should not be nil")

	contents := msg.GetAssistant().GetContents()
	assert.NotEmpty(t, contents, "assistant should return content")
	assert.NotEmpty(t, metrics, "metrics should be returned")
	testutil.AssertHasMetric(t, metrics, "TIME_TAKEN")
	t.Logf("provider=%s response=%q", providerName, contents)
}

// TestIntegration_StreamChatCompletion verifies streaming chat completion: tokens
// should be streamed via onStream, and metrics delivered once via onMetrics.
func TestIntegration_StreamChatCompletion(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.ChatProvider(t, providerName)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	stream, err := NewChatStream(testutil.NewTestLogger(), cred, map[string]string{
		OptionTransportKey: TransportChatResp,
	})
	require.NoError(t, err)
	opts := testutil.BuildChatStreamOptions(pcfg)
	sc := &testutil.StreamCollector{}

	err = stream.Connect(ctx, nil)
	require.NoError(t, err, "Connect should succeed")
	defer func() { _ = stream.Close(ctx) }()

	err = stream.Chat(ctx, testutil.SimpleMessages(), opts, sc.OnStream, sc.OnMetrics, sc.OnError)
	require.NoError(t, err, "StreamChatCompletion should succeed")
	sc.AssertStream(t)
	t.Logf("provider=%s stream_tokens=%d", providerName, sc.StreamCount)
}

// TestIntegration_Embedding verifies embedding generation: a single document
// should produce a non-empty vector with TIME_TAKEN metric.
func TestIntegration_Embedding(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.EmbeddingProvider(t, providerName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	caller := internal_openai_text_embedding.New(testutil.NewTestLogger(), cred)
	opts := testutil.BuildEmbeddingOptions(pcfg)

	embeddings, metrics, err := caller.GetEmbedding(ctx, testutil.EmbeddingContent(), opts)
	require.NoError(t, err, "GetEmbedding should succeed")
	require.NotEmpty(t, embeddings, "should return at least one embedding")
	for i, emb := range embeddings {
		assert.NotEmpty(t, emb.GetEmbedding(), "embedding[%d] vector should not be empty", i)
	}
	testutil.AssertHasMetric(t, metrics, "TIME_TAKEN")
	t.Logf("provider=%s embeddings=%d dimensions=%d", providerName, len(embeddings), len(embeddings[0].GetEmbedding()))
}

// TestIntegration_VerifyCredential verifies that valid credentials pass
// the provider's credential verification endpoint without error.
func TestIntegration_VerifyCredential(t *testing.T) {
	cfg := testutil.LoadConfig(t)
	pcfg := cfg.VerifyProvider(t, providerName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cred := testutil.BuildCredential(pcfg.Credential)
	verifier := internal_openai_verify_credential.New(testutil.NewTestLogger(), cred)
	_, err := verifier.CredentialVerifier(ctx, testutil.BuildVerifyOptions(pcfg))
	require.NoError(t, err, "CredentialVerifier should succeed with valid credentials")
	t.Logf("provider=%s credential_verification=ok", providerName)
}
