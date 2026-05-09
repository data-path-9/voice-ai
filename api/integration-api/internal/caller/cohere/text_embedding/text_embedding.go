// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package internal_cohere_text_embedding

import (
	"context"
	"time"

	cohere "github.com/cohere-ai/cohere-go/v2"

	internal_cohere_common "github.com/rapidaai/api/integration-api/internal/caller/cohere/common"
	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
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

func (ec *caller) getEmbedRequest(opts *internal_callers.EmbeddingOptions) *cohere.V2EmbedRequest {
	options := &cohere.V2EmbedRequest{}
	for key, value := range opts.ModelParameter {
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options.Model = modelName
			}
		case "model.input_type":
			if inputType, err := utils.AnyToString(value); err == nil {
				options.InputType = cohere.EmbedInputType(inputType)
			}
		case "model.dimensions":
			if dimensions, err := utils.AnyToInt(value); err == nil {
				options.OutputDimension = cohere.Int(dimensions)
			}
		}
	}
	return options
}

func (ec *caller) GetEmbedding(
	ctx context.Context,
	content map[int32]string,
	options *internal_callers.EmbeddingOptions,
) ([]*protos.Embedding, []*protos.Metric, error) {
	metrics := internal_caller_metrics.NewMetricBuilder(options.RequestId)
	metrics.OnStart()

	client, err := internal_cohere_common.NewClient(ec.credential)
	if err != nil {
		return nil, metrics.OnFailure().Build(), err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	input := make([]string, len(content))
	for k, v := range content {
		input[k] = v
	}

	request := ec.getEmbedRequest(options)
	request.Texts = input

	resp, err := client.V2.Embed(ctx, request)
	if err != nil {
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"result": resp,
				"error":  err,
			}, metrics.OnFailure().Build())
		}
		return nil, metrics.Build(), err
	}

	output := make([]*protos.Embedding, len(input))
	for idx, embeddingData := range resp.GetEmbeddings().GetFloat() {
		output[idx] = &protos.Embedding{
			Index:     int32(idx),
			Embedding: utils.EmbeddingToFloat64(embeddingData),
			Base64:    utils.EmbeddingToBase64(embeddingData),
		}
	}

	metrics.OnSuccess()
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{
			"result": resp,
		}, metrics.Build())
	}
	return output, metrics.Build(), nil
}
