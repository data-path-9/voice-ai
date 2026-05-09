// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_reranking

import (
	"context"
	"strings"
	"time"

	cohere "github.com/cohere-ai/cohere-go/v2"

	internal_cohere_common "github.com/rapidaai/api/integration-api/internal/caller/cohere/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	protos "github.com/rapidaai/protos"
)

type caller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func New(logger commons.Logger, credential *protos.Credential) internal_callers.RerankingCaller {
	return &caller{
		logger:     logger,
		credential: credential,
	}
}

func (rr *caller) getRerankRequest(opts *internal_callers.RerankerOptions) *cohere.RerankRequest {
	options := cohere.RerankRequest{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if mn, err := utils.AnyToString(value); err == nil {
				options.Model = utils.Ptr(mn)
			}
		case "model.top_n":
			if topN, err := utils.AnyToInt(value); err == nil {
				options.TopN = utils.Ptr(topN)
			}
		case "model.max_chunks_per_doc":
			if mxChunk, err := utils.AnyToInt(value); err == nil {
				options.MaxChunksPerDoc = utils.Ptr(mxChunk)
			}
		case "model.rank_fields":
			if stopStr, err := utils.AnyToString(value); err == nil {
				options.RankFields = strings.Split(stopStr, ",")
			}
		}
	}
	return &options
}

func (rr *caller) GetReranking(
	ctx context.Context,
	query string,
	content map[int32]string,
	options *internal_callers.RerankerOptions,
) ([]*protos.Reranking, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := internal_cohere_common.NewClient(rr.credential)
	if err != nil {
		return nil, metrics.OnFailure().Build(), err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	input := make([]*cohere.RerankRequestDocumentsItem, len(content))
	for k, v := range content {
		input[k] = &cohere.RerankRequestDocumentsItem{
			String: v,
			RerankDocument: map[string]string{
				"Description": v,
			},
		}
	}

	rerankRequest := rr.getRerankRequest(options)
	rerankRequest.Query = query
	rerankRequest.Documents = input

	if options.PreHook != nil {
		options.PreHook(utils.ToJson(rerankRequest))
	}

	resp, err := client.Rerank(ctx, rerankRequest)
	if err != nil {
		if options.PostHook != nil {
			options.PostHook(nil, metrics.OnFailure().Build())
		}
		return nil, metrics.Build(), err
	}

	metrics.OnSuccess()
	output := make([]*protos.Reranking, len(resp.Results))
	for _, rerankedData := range resp.Results {
		output[rerankedData.Index] = &protos.Reranking{
			Index:          int32(rerankedData.Index),
			Content:        content[int32(rerankedData.Index)],
			RelevanceScore: rerankedData.RelevanceScore,
		}
	}

	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{
			"result": resp,
		}, metrics.Build())
	}
	return output, metrics.Build(), nil
}
