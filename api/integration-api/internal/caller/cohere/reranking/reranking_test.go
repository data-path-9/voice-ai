// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_reranking

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

func TestGetRerankRequest_MapsOptions(t *testing.T) {
	c := &caller{logger: newTestLogger()}
	req := c.getRerankRequest(&internal_callers.RerankerOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.name":               anyString(t, "rerank-v3.5"),
				"model.top_n":              anyNumber(t, 3),
				"model.max_chunks_per_doc": anyNumber(t, 2),
				"model.rank_fields":        anyString(t, "title,body"),
			},
		},
	})

	require.NotNil(t, req.Model)
	assert.Equal(t, "rerank-v3.5", *req.Model)
	require.NotNil(t, req.TopN)
	assert.Equal(t, 3, *req.TopN)
	require.NotNil(t, req.MaxChunksPerDoc)
	assert.Equal(t, 2, *req.MaxChunksPerDoc)
	assert.Equal(t, []string{"title", "body"}, req.RankFields)
}

func TestGetReranking_ReturnsCredentialErrorForInvalidCredential(t *testing.T) {
	c := &caller{logger: newTestLogger(), credential: nil}
	options := &internal_callers.RerankerOptions{
		AIOptions: internal_callers.AIOptions{
			RequestId:      100,
			PreHook:        func(map[string]interface{}) {},
			PostHook:       func(map[string]interface{}, []*protos.Metric) {},
			ModelParameter: map[string]*anypb.Any{},
		},
	}

	results, metrics, err := c.GetReranking(
		context.Background(),
		"query",
		map[int32]string{0: "doc"},
		options,
	)
	require.Error(t, err)
	assert.Nil(t, results)
	assert.NotEmpty(t, metrics)
}
