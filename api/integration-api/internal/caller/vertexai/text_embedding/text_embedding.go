// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_vertexai_text_embedding

import (
	"context"
	"time"

	"google.golang.org/genai"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_vertexai_common "github.com/rapidaai/api/integration-api/internal/caller/vertexai/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type caller struct {
	logger     commons.Logger
	credential *protos.Credential
}

func New(logger commons.Logger, credential *protos.Credential) internal_callers.EmbeddingCaller {
	return &caller{
		logger:     logger,
		credential: credential,
	}
}

func (ec *caller) getEmbedContentConfig(opts *internal_callers.EmbeddingOptions) (model string, cfg *genai.EmbedContentConfig) {
	cfg = &genai.EmbedContentConfig{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				model = modelName
			}
		case "model.output_dimensionality":
			if dimensions, err := utils.AnyToInt32(value); err == nil {
				cfg.OutputDimensionality = utils.Ptr(dimensions)
			}
		}
	}
	return model, cfg
}

func (ec *caller) GetEmbedding(
	ctx context.Context,
	content map[int32]string,
	options *internal_callers.EmbeddingOptions,
) ([]*protos.Embedding, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := internal_vertexai_common.NewClient(ec.credential)
	if err != nil {
		metrics.OnFailure()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{"error": err}, metrics.Build())
		}
		return nil, metrics.Build(), err
	}

	if options.PreHook != nil {
		options.PreHook(map[string]interface{}{"request": content})
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	output := make([]*protos.Embedding, len(content))
	model, cfg := ec.getEmbedContentConfig(options)
	contents := make([]*genai.Content, len(content))
	for idx, st := range content {
		contents[idx] = genai.NewContentFromText(st, "user")
	}

	resp, err := client.Models.EmbedContent(ctx, model, contents, cfg)
	if err != nil {
		metrics.OnFailure()
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"result": resp,
				"error":  err,
			}, metrics.Build())
		}
		return nil, metrics.Build(), err
	}

	for idx, v := range resp.Embeddings {
		output[idx] = &protos.Embedding{
			Index:     int32(idx),
			Embedding: utils.EmbeddingToFloat64(v.Values),
			Base64:    utils.EmbeddingToBase64(utils.EmbeddingToFloat64(v.Values)),
		}
	}

	metrics.OnSuccess()
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{"result": output}, metrics.Build())
	}
	return output, metrics.Build(), nil
}
