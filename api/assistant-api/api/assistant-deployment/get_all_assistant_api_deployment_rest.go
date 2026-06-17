// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rapidaai/openapi"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	assistant_api "github.com/rapidaai/protos"
)

func (deploymentApi *AssistantDeploymentApi) GetAllAssistantApiDeploymentRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.GetAllAssistantApiDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantApiDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !auth.HasUser() || !auth.HasProject() || !auth.HasOrganization() {
		c.JSON(pkg_errors.GetAllAssistantApiDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantApiDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := strconv.ParseUint(c.Param("assistantId"), 10, 64)
	if err != nil || assistantId == 0 {
		c.JSON(pkg_errors.GetAllAssistantApiDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantApiDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}

	paginate := &assistant_api.Paginate{Page: 1, PageSize: 20}
	if c.Query("page") != "" {
		page, err := strconv.ParseUint(c.Query("page"), 10, 32)
		if err != nil || page == 0 {
			c.JSON(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.ErrorMessage),
				},
			})
			return
		}
		paginate.Page = uint32(page)
	}
	if c.Query("pageSize") != "" {
		pageSize, err := strconv.ParseUint(c.Query("pageSize"), 10, 32)
		if err != nil || pageSize == 0 {
			c.JSON(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.ErrorMessage),
				},
			})
			return
		}
		paginate.PageSize = uint32(pageSize)
	}

	criterias := []*assistant_api.Criteria{}
	if c.Query("criterias") != "" {
		requestCriterias := []openapi.Criteria{}
		if err := json.Unmarshal([]byte(c.Query("criterias")), &requestCriterias); err != nil {
			c.JSON(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentInvalidRequest.ErrorMessage),
				},
			})
			return
		}
		for _, criteria := range requestCriterias {
			key := ""
			if validator.NonNil(criteria.Key) {
				key = *criteria.Key
			}
			value := ""
			if validator.NonNil(criteria.Value) {
				value = *criteria.Value
			}
			logic := ""
			if validator.NonNil(criteria.Logic) {
				logic = *criteria.Logic
			}
			criterias = append(criterias, &assistant_api.Criteria{Key: key, Value: value, Logic: logic})
		}
	}

	totalItems, deployments, err := deploymentApi.deploymentService.GetAllAssistantApiDeployment(c, auth, assistantId, criterias, paginate)
	if err != nil {
		deploymentApi.logger.Errorf("unable to get all assistant api deployments: %v", err)
		c.JSON(pkg_errors.GetAllAssistantApiDeploymentGetDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentGetDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantApiDeploymentGetDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentGetDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantApiDeploymentGetDeployment.ErrorMessage),
			},
		})
		return
	}

	responseDeployments := []openapi.AssistantApiDeployment{}
	for _, deployment := range deployments {
		if !validator.NonNil(deployment) {
			continue
		}
		deploymentId := openapi.Uint64String(strconv.FormatUint(deployment.Id, 10))
		deploymentAssistantId := openapi.Uint64String(strconv.FormatUint(deployment.AssistantId, 10))
		deploymentStatus := deployment.Status.String()

		var responseInputAudio *openapi.DeploymentAudioProvider
		if validator.NonNil(deployment.InputAudio) {
			inputAudioId := openapi.Uint64String(strconv.FormatUint(deployment.InputAudio.Id, 10))
			inputAudioStatus := deployment.InputAudio.Status.String()
			inputAudioOptions := []openapi.Metadata{}
			for _, audioOption := range deployment.InputAudio.AudioOptions {
				if !validator.NonNil(audioOption) {
					continue
				}
				inputAudioOptions = append(inputAudioOptions, openapi.Metadata{
					Key:   utils.Ptr(audioOption.Key),
					Value: utils.Ptr(audioOption.Value),
				})
			}
			responseInputAudio = &openapi.DeploymentAudioProvider{
				Id:            &inputAudioId,
				AudioType:     &deployment.InputAudio.AudioType,
				AudioProvider: &deployment.InputAudio.AudioProvider,
				AudioOptions:  &inputAudioOptions,
				Status:        &inputAudioStatus,
			}
		}

		var responseOutputAudio *openapi.DeploymentAudioProvider
		if validator.NonNil(deployment.OutputAudio) {
			outputAudioId := openapi.Uint64String(strconv.FormatUint(deployment.OutputAudio.Id, 10))
			outputAudioStatus := deployment.OutputAudio.Status.String()
			outputAudioOptions := []openapi.Metadata{}
			for _, audioOption := range deployment.OutputAudio.AudioOptions {
				if !validator.NonNil(audioOption) {
					continue
				}
				outputAudioOptions = append(outputAudioOptions, openapi.Metadata{
					Key:   utils.Ptr(audioOption.Key),
					Value: utils.Ptr(audioOption.Value),
				})
			}
			responseOutputAudio = &openapi.DeploymentAudioProvider{
				Id:            &outputAudioId,
				AudioType:     &deployment.OutputAudio.AudioType,
				AudioProvider: &deployment.OutputAudio.AudioProvider,
				AudioOptions:  &outputAudioOptions,
				Status:        &outputAudioStatus,
			}
		}
		responseDeployments = append(responseDeployments, openapi.AssistantApiDeployment{
			Id:                    &deploymentId,
			AssistantId:           &deploymentAssistantId,
			Greeting:              deployment.Greeting,
			GreetingInterruptible: deployment.GreetingInterruptible,
			Mistake:               deployment.Mistake,
			InputAudio:            responseInputAudio,
			OutputAudio:           responseOutputAudio,
			Status:                &deploymentStatus,
			MaxSessionDuration:    deployment.MaxSessionDuration,
			IdealTimeout:          deployment.IdleTimeout,
			IdealTimeoutBackoff:   deployment.IdleTimeoutBackoff,
			IdealTimeoutMessage:   deployment.IdleTimeoutMessage,
		})
	}
	totalItem := uint32(totalItems)
	currentPage := paginate.GetPage()
	c.JSON(http.StatusOK, openapi.GetAllAssistantApiDeploymentResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    &responseDeployments,
		Paginated: &openapi.Paginated{
			TotalItem:   &totalItem,
			CurrentPage: &currentPage,
		},
	})
}
