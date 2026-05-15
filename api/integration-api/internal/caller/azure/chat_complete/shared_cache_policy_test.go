// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_azure_chat_complete

import (
	"encoding/json"
	"testing"

	internal_callers "github.com/rapidaai/api/integration-api/internal/type"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func cacheTestAnyString(t *testing.T, value string) *anypb.Any {
	t.Helper()
	anyValue, err := anypb.New(structpb.NewStringValue(value))
	require.NoError(t, err)
	return anyValue
}

func cacheTestToMap(t *testing.T, params interface{ MarshalJSON() ([]byte, error) }) map[string]interface{} {
	t.Helper()
	payload, err := params.MarshalJSON()
	require.NoError(t, err)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &data))
	return data
}

func cachePolicyOptions(t *testing.T) *internal_callers.ChatCompletionOptions {
	t.Helper()
	return &internal_callers.ChatCompletionOptions{
		AIOptions: internal_callers.AIOptions{
			ModelParameter: map[string]*anypb.Any{
				"model.parameters": testAny(t, map[string]interface{}{
					"prompt_cache_key":       "conversation_id",
					"prompt_cache_retention": "24h",
				}),
			},
		},
		Request: &protos.ChatRequest{
			AdditionalData: map[string]string{
				"conversation_id":             "conv-1",
				"assistant_provider_model_id": "model-1",
				"assistant_id":                "assistant-1",
			},
		},
	}
}

func TestBuildChatCompletionOptions_DisablesPromptCache(t *testing.T) {
	options := cachePolicyOptions(t)
	data := cacheTestToMap(t, buildChatCompletionOptions(options))

	_, hasPromptCacheKey := data["prompt_cache_key"]
	_, hasPromptCacheRetention := data["prompt_cache_retention"]
	assert.False(t, hasPromptCacheKey)
	assert.False(t, hasPromptCacheRetention)
}

func TestBuildStreamCompletionOptions_EnablesPromptCache(t *testing.T) {
	options := cachePolicyOptions(t)
	data := cacheTestToMap(t, buildStreamCompletionOptions(options))

	assert.Equal(t, "conv-1__model-1__assistant-1", data["prompt_cache_key"])
	assert.Equal(t, "24h", data["prompt_cache_retention"])
}

func TestBuildStreamCompletionOptions_OmitsPromptCacheWhenBaseKeyMetadataMissing(t *testing.T) {
	options := cachePolicyOptions(t)
	delete(options.Request.AdditionalData, "assistant_provider_model_id")
	data := cacheTestToMap(t, buildStreamCompletionOptions(options))

	_, hasPromptCacheKey := data["prompt_cache_key"]
	_, hasPromptCacheRetention := data["prompt_cache_retention"]
	assert.False(t, hasPromptCacheKey)
	assert.False(t, hasPromptCacheRetention)
}

func TestBuildStreamCompletionOptions_OmitsPromptCacheWhenSelectorMetadataMissing(t *testing.T) {
	options := cachePolicyOptions(t)
	delete(options.Request.AdditionalData, "conversation_id")
	data := cacheTestToMap(t, buildStreamCompletionOptions(options))

	_, hasPromptCacheKey := data["prompt_cache_key"]
	_, hasPromptCacheRetention := data["prompt_cache_retention"]
	assert.False(t, hasPromptCacheKey)
	assert.False(t, hasPromptCacheRetention)
}
