// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/api/assistant-api/config"
	channel_telephony "github.com/rapidaai/api/assistant-api/internal/channel/telephony"
	configs "github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallReciever_ProviderAnswerHookSkipsAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/talk/vobiz/call/42?CustomField=v1%2Ftalk%2Fvobiz%2Fctx%2Fctx-1&StatusCallback=v1%2Ftalk%2Fvobiz%2Fctx%2Fctx-1%2Fevent", nil)
	c.Params = gin.Params{
		{Key: "telephony", Value: "vobiz"},
		{Key: "assistantId", Value: "42"},
	}

	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	cfg := &config.AssistantConfig{AppConfig: configs.AppConfig{Assistant: configs.ServiceHostConfig{Public: "app.rapida.ai"}}}
	cApi := &ConversationApi{
		logger: logger,
		inboundDispatcher: channel_telephony.NewInboundDispatcher(
			channel_telephony.WithConfig(cfg),
			channel_telephony.WithLogger(logger),
		),
	}

	cApi.CallReciever(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `<Response><Stream`)
	assert.Contains(t, w.Body.String(), `wss://app.rapida.ai/v1/talk/vobiz/ctx/ctx-1`)
}
