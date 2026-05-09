// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_text_embedding

import (
	"context"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestLogger() commons.Logger {
	lgr, _ := commons.NewApplicationLogger()
	return lgr
}

func anyString(t *testing.T, value string) *anypb.Any {
	t.Helper()
	anyValue, err := anypb.New(structpb.NewStringValue(value))
	require.NoError(t, err)
	return anyValue
}

func anyNumber(t *testing.T, value float64) *anypb.Any {
	t.Helper()
	anyValue, err := anypb.New(structpb.NewNumberValue(value))
	require.NoError(t, err)
	return anyValue
}

func TestNew_ReturnsCaller(t *testing.T) {
	c := New(newTestLogger(), nil)
	require.NotNil(t, c)
}

func TestGetEmbedContentConfig_MapsOptions(t *testing.T) {
	c := &caller{logger: newTestLogger()}
	model, cfg := c.getEmbedContentConfig(&internal_callers.EmbeddingOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name":                  anyString(t, "text-embedding-004"),
				"model.output_dimensionality": anyNumber(t, 128),
			},
		},
	})

	assert.Equal(t, "text-embedding-004", model)
	require.NotNil(t, cfg.OutputDimensionality)
	assert.Equal(t, int32(128), *cfg.OutputDimensionality)
}

func TestGetEmbedding_ReturnsCredentialErrorForInvalidCredential(t *testing.T) {
	c := &caller{logger: newTestLogger(), credential: nil}
	options := &internal_callers.EmbeddingOptions{
		AIOptions: internal_callers.AIOptions{
			RequestId:      100,
			PreHook:        func(map[string]interface{}) {},
			PostHook:       func(map[string]interface{}, []*protos.Metric) {},
			ModelParameter: map[string]*anypb.Any{},
		},
	}

	embeddings, metrics, err := c.GetEmbedding(context.Background(), map[int32]string{0: "hello"}, options)
	require.Error(t, err)
	assert.Nil(t, embeddings)
	assert.NotEmpty(t, metrics)
}
