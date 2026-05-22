// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	channel_pipeline "github.com/rapidaai/api/assistant-api/internal/channel/pipeline"
	"github.com/rapidaai/openapi"
	"github.com/rapidaai/pkg/preset"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

// CreatePhoneCallRest initiates an outbound phone call from the REST API.
func (cApi *ConversationApi) CreatePhoneCallRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(http.StatusForbidden, errorResponse(401, fmt.Errorf("unauthenticated request"), "Unauthenticated request, please try again with valid authentication."))
		return
	}

	var ir openapi.CreatePhoneCallRequest
	if err := c.ShouldBindJSON(&ir); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(400, err, "Invalid request."))
		return
	}

	if ir.ToNumber == nil || utils.IsEmpty(*ir.ToNumber) {
		c.JSON(http.StatusBadRequest, errorResponse(200, fmt.Errorf("missing to_phone parameter"), "Please provide the required to_phone parameter."))
		return
	}

	assistant, err := toProtoAssistantDefinition(ir.Assistant)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse(200, err, "Please provide a valid assistant."))
		return
	}
	preset.AssistantDefinition(assistant)
	if !validator.OfAssistantDefinition(assistant) {
		c.JSON(http.StatusBadRequest, errorResponse(200, fmt.Errorf("invalid assistant"), "Please provide a valid assistant."))
		return
	}

	result := cApi.channelPipeline.Run(c, channel_pipeline.OutboundRequestedPipeline{
		ID:          fmt.Sprintf("%d", assistant.GetAssistantId()),
		Auth:        auth,
		AssistantID: assistant.GetAssistantId(),
		Version:     assistant.GetVersion(),
		ToPhone:     *ir.ToNumber,
		FromPhone:   stringValue(ir.FromNumber),
		Metadata:    mapValue(ir.Metadata),
		Args:        mapValue(ir.Args),
		Options:     mapValue(ir.Options),
	})
	if result.Error != nil {
		cApi.logger.Errorf("outbound call failed: %v", result.Error)
		c.JSON(http.StatusInternalServerError, errorResponse(500, result.Error, "Failed to initiate outbound call"))
		return
	}

	cApi.logger.Infof("outbound call dispatched: contextId=%s, conversationId=%d",
		result.ContextID, result.ConversationID)

	c.JSON(http.StatusOK, openapi.CreatePhoneCallResponse{
		Code:    int32Ptr(200),
		Success: boolPtr(true),
		Data: &openapi.AssistantConversation{
			Id: uint64StringPtr(result.ConversationID),
		},
	})
}

func toProtoAssistantDefinition(assistant *openapi.AssistantDefinition) (*protos.AssistantDefinition, error) {
	if assistant == nil || assistant.AssistantId == nil {
		return nil, fmt.Errorf("invalid assistant")
	}
	assistantID, err := strconv.ParseUint(string(*assistant.AssistantId), 10, 64)
	if err != nil {
		return nil, err
	}
	return &protos.AssistantDefinition{
		AssistantId: assistantID,
		Version:     stringValue(assistant.Version),
	}, nil
}

func errorResponse(code int32, err error, humanMessage string) openapi.ErrorResponse {
	return openapi.ErrorResponse{
		Code:    &code,
		Success: boolPtr(false),
		Error: &openapi.Error{
			ErrorCode:    uint64StringPtr(uint64(code)),
			ErrorMessage: stringPtr(err.Error()),
			HumanMessage: stringPtr(humanMessage),
		},
	}
}

func mapValue(value *map[string]interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func uint64StringPtr(value uint64) *openapi.Uint64String {
	out := openapi.Uint64String(strconv.FormatUint(value, 10))
	return &out
}
