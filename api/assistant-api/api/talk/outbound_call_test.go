package assistant_talk_api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePhoneCall_Success(t *testing.T) {
	conversationApi := &ConversationGrpcApi{
		ConversationApi: ConversationApi{
			logger:          outboundCallTestLogger(t),
			channelPipeline: outboundCallTestDispatcher(t, nil),
		},
	}
	ctx := context.WithValue(context.Background(), types.CTX_, outboundCallTestAuth())

	response, err := conversationApi.CreatePhoneCall(ctx, &protos.CreatePhoneCallRequest{
		Assistant: &protos.AssistantDefinition{AssistantId: 42},
		ToNumber:  "+19999999999",
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Data)
	assert.True(t, response.Success)
	assert.Equal(t, int32(http.StatusOK), response.Code)
	assert.Equal(t, uint64(200), response.Data.Id)
}

func TestCreatePhoneCall_PipelineErrorUsesStandardError(t *testing.T) {
	conversationApi := &ConversationGrpcApi{
		ConversationApi: ConversationApi{
			logger:          outboundCallTestLogger(t),
			channelPipeline: outboundCallTestDispatcher(t, errors.New("provider credential leaked")),
		},
	}
	ctx := context.WithValue(context.Background(), types.CTX_, outboundCallTestAuth())

	response, err := conversationApi.CreatePhoneCall(ctx, &protos.CreatePhoneCallRequest{
		Assistant: &protos.AssistantDefinition{AssistantId: 42},
		ToNumber:  "+19999999999",
	})

	require.Error(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Error)
	assert.Equal(t, int32(pkg_errors.CreatePhoneCallInitiateOutbound.HTTPStatusCode), response.Code)
	assert.Equal(t, uint64(pkg_errors.CreatePhoneCallInitiateOutbound.Code), response.Error.ErrorCode)
	assert.Equal(t, pkg_errors.CreatePhoneCallInitiateOutbound.Error, response.Error.ErrorMessage)
	assert.Equal(t, pkg_errors.CreatePhoneCallInitiateOutbound.ErrorMessage, response.Error.HumanMessage)
	assert.NotContains(t, response.Error.ErrorMessage, "provider credential leaked")
	assert.NotContains(t, err.Error(), "provider credential leaked")
}

func TestCreateBulkPhoneCall_MissingPhoneCallsUsesStandardError(t *testing.T) {
	conversationApi := &ConversationGrpcApi{}
	ctx := context.WithValue(context.Background(), types.CTX_, outboundCallTestAuth())

	response, err := conversationApi.CreateBulkPhoneCall(ctx, &protos.CreateBulkPhoneCallRequest{})

	require.Error(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Error)
	assert.Equal(t, int32(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.HTTPStatusCode), response.Code)
	assert.Equal(t, uint64(pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.Code), response.Error.ErrorCode)
	assert.Equal(t, pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.Error, response.Error.ErrorMessage)
	assert.Equal(t, pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.ErrorMessage, response.Error.HumanMessage)
}

func TestCreatePhoneCallRest_PipelineErrorUsesStandardError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conversationApi := &ConversationApi{
		logger:          outboundCallTestLogger(t),
		channelPipeline: outboundCallTestDispatcher(t, errors.New("provider credential leaked")),
	}
	requestBody := []byte(`{"assistant":{"assistantId":"42"},"toNumber":"+19999999999"}`)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/talk/create-phone-call", bytes.NewReader(requestBody))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set(string(types.CTX_), outboundCallTestAuth())

	conversationApi.CreatePhoneCallRest(context)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	response := map[string]interface{}{}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	errorBody, ok := response["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, pkg_errors.CreatePhoneCallInitiateOutbound.CodeString(), errorBody["errorCode"])
	assert.Equal(t, pkg_errors.CreatePhoneCallInitiateOutbound.Error, errorBody["errorMessage"])
	assert.Equal(t, pkg_errors.CreatePhoneCallInitiateOutbound.ErrorMessage, errorBody["humanMessage"])
	assert.NotContains(t, recorder.Body.String(), "provider credential leaked")
}

func TestCreateBulkPhoneCallRest_MissingPhoneCallsUsesStandardError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conversationApi := &ConversationApi{}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/talk/create-bulk-phone-call", bytes.NewReader([]byte(`{"phoneCalls":[]}`)))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Set(string(types.CTX_), outboundCallTestAuth())

	conversationApi.CreateBulkPhoneCallRest(context)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.CodeString())
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.Error)
	assert.Contains(t, recorder.Body.String(), pkg_errors.CreateBulkPhoneCallMissingPhoneCalls.ErrorMessage)
}

func outboundCallTestDispatcher(t *testing.T, dispatchError error) *channel_pipeline.Dispatcher {
	t.Helper()

	return channel_pipeline.NewDispatcher(&channel_pipeline.DispatcherConfig{
		Logger: outboundCallTestLogger(t),
		OnLoadAssistant: func(ctx context.Context, auth types.SimplePrinciple, assistantID uint64) (*internal_assistant_entity.Assistant, error) {
			assistant := &internal_assistant_entity.Assistant{
				AssistantProviderId: 1,
				AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
					AssistantDeploymentTelephony: internal_assistant_entity.AssistantDeploymentTelephony{
						TelephonyProvider: "twilio",
						TelephonyOption: []*internal_assistant_entity.AssistantDeploymentTelephonyOption{
							{Metadata: gorm_model.Metadata{Key: "phone", Value: "+10000000000"}},
						},
					},
				},
			}
			assistant.Id = assistantID
			return assistant, nil
		},
		OnCreateConversation: func(ctx context.Context, auth types.SimplePrinciple, callerNumber string, assistantID, assistantProviderID uint64, direction string) (uint64, error) {
			return 200, nil
		},
		OnSaveCallContext: func(ctx context.Context, auth types.SimplePrinciple, assistant *internal_assistant_entity.Assistant, conversationID uint64, callInfo *internal_type.CallInfo, provider string) (string, error) {
			return "ctx-outbound-test", nil
		},
		OnDispatchOutbound: func(ctx context.Context, contextID string) error {
			return dispatchError
		},
	})
}

func outboundCallTestAuth() *types.PlainAuthPrinciple {
	return &types.PlainAuthPrinciple{
		User: types.UserInfo{Id: 11},
		OrganizationRole: &types.OrganizaitonRole{
			OrganizationId: 22,
		},
		CurrentProjectRole: &types.ProjectRole{
			ProjectId: 33,
		},
	}
}

func outboundCallTestLogger(t *testing.T) commons.Logger {
	t.Helper()

	logger, err := commons.NewApplicationLogger(
		commons.EnableConsole(true),
		commons.EnableFile(false),
		commons.Level("error"),
	)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}
