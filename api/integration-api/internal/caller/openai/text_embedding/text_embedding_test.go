// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_text_embedding

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

func TestNew_ReturnsCaller(t *testing.T) {
	c := New(newTestLogger(), nil)
	require.NotNil(t, c)
}

func TestGetEmbeddingNewParams_MapsOptions(t *testing.T) {
	c := &caller{logger: newTestLogger()}
	params := c.getEmbeddingNewParams(&internal_callers.EmbeddingOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name":            anyString(t, "text-embedding-3-large"),
				"model.user":            anyString(t, "user-1"),
				"model.encoding_format": anyString(t, "float"),
			},
		},
	})

	assert.Equal(t, "text-embedding-3-large", params.Model)
	assert.Equal(t, "user-1", params.User.Value)
	assert.Equal(t, "float", string(params.EncodingFormat))
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
