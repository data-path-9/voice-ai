// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_openai_text_embedding

import (
	"context"
	"fmt"
	"time"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	internal_caller_metrics "github.com/rapidaai/api/integration-api/internal/caller/metrics"
	internal_openai_common "github.com/rapidaai/api/integration-api/internal/caller/openai/common"
	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

type caller struct {
	logger     commons.Logger
	credential *protos.Credential
	client     *openai.Client
}

func New(logger commons.Logger, credential *protos.Credential) internal_callers.EmbeddingCaller {
	return &caller{
		logger:     logger,
		credential: credential,
	}
}

func (ec *caller) getClient() (*openai.Client, error) {
	if ec.client != nil {
		return ec.client, nil
	}
	apiKey, err := internal_openai_common.ResolveAPIKey(ec.credential)
	if err != nil {
		return nil, err
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	ec.client = &client
	return ec.client, nil
}

func (ec *caller) getEmbeddingNewParams(opts *internal_callers.EmbeddingOptions) openai.EmbeddingNewParams {
	options := openai.EmbeddingNewParams{}
	for key, value := range opts.ModelParameter {
		ec.logger.Debugf("goting %+v. %+v", key, value)
		switch key {
		case "model.name":
			if modelName, err := utils.AnyToString(value); err == nil {
				options.Model = modelName
			}
		case "model.user":
			if user, err := utils.AnyToString(value); err == nil {
				options.User = openai.String(user)
			}
		case "model.encoding_format":
			if re, err := utils.AnyToString(value); err == nil {
				options.EncodingFormat = openai.EmbeddingNewParamsEncodingFormat(re)
			}
		case "model.dimensions":
			if dimensions, err := utils.AnyToInt64(value); err == nil {
				options.Dimensions = openai.Int(dimensions)
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

	client, err := ec.getClient()
	if err != nil {
		return nil, metrics.OnFailure().Build(), err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	input := make([]string, len(content))
	for k, v := range content {
		input[k] = v
	}

	opts := ec.getEmbeddingNewParams(options)
	opts.Input = openai.EmbeddingNewParamsInputUnion{
		OfArrayOfStrings: input,
	}

	if options.PreHook != nil {
		options.PreHook(map[string]interface{}{"input": opts})
	}
	resp, err := client.Embeddings.New(
		ctx,
		opts,
	)

	if err != nil {
		if options.PostHook != nil {
			options.PostHook(map[string]interface{}{
				"result": resp,
				"error":  err,
			}, metrics.OnFailure().Build())
		}
		return nil, metrics.Build(), err
	}
	metrics.OnSuccess()
	output := make([]*protos.Embedding, len(resp.Data))

	metrics.OnAddMetrics(&protos.Metric{
		Name:        type_enums.OUTPUT_TOKEN.String(),
		Value:       fmt.Sprintf("%d", resp.Usage.PromptTokens),
		Description: "Input token",
	}, &protos.Metric{
		Name:        type_enums.TOTAL_TOKEN.String(),
		Value:       fmt.Sprintf("%d", resp.Usage.TotalTokens),
		Description: "Total Token",
	})

	for _, embeddingData := range resp.Data {
		output[embeddingData.Index] = &protos.Embedding{
			Index:     int32(embeddingData.Index),
			Embedding: embeddingData.Embedding,
		}
	}
	if options.PostHook != nil {
		options.PostHook(map[string]interface{}{
			"result": resp,
		}, metrics.Build())
	}
	return output, metrics.Build(), nil
}
