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

func (deploymentApi *AssistantDeploymentApi) GetAllAssistantWhatsappDeploymentRest(c *gin.Context) {
	auth, isAuthenticated := types.GetAuthPrinciple(c)
	if !isAuthenticated {
		c.JSON(pkg_errors.GetAllAssistantWhatsappDeploymentUnauthenticated.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentUnauthenticated.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWhatsappDeploymentUnauthenticated.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentUnauthenticated.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentUnauthenticated.ErrorMessage),
			},
		})
		return
	}
	if !auth.HasUser() || !auth.HasProject() || !auth.HasOrganization() {
		c.JSON(pkg_errors.GetAllAssistantWhatsappDeploymentMissingAuthScope.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentMissingAuthScope.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWhatsappDeploymentMissingAuthScope.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentMissingAuthScope.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentMissingAuthScope.ErrorMessage),
			},
		})
		return
	}

	assistantId, err := strconv.ParseUint(c.Param("assistantId"), 10, 64)
	if err != nil || assistantId == 0 {
		c.JSON(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidAssistantID.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidAssistantID.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidAssistantID.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidAssistantID.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidAssistantID.ErrorMessage),
			},
		})
		return
	}

	paginate := &assistant_api.Paginate{Page: 1, PageSize: 20}
	if c.Query("page") != "" {
		page, err := strconv.ParseUint(c.Query("page"), 10, 32)
		if err != nil || page == 0 {
			c.JSON(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.ErrorMessage),
				},
			})
			return
		}
		paginate.Page = uint32(page)
	}
	if c.Query("pageSize") != "" {
		pageSize, err := strconv.ParseUint(c.Query("pageSize"), 10, 32)
		if err != nil || pageSize == 0 {
			c.JSON(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.ErrorMessage),
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
			c.JSON(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCode, openapi.ErrorResponse{
				Code:    utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.HTTPStatusCodeInt32()),
				Success: utils.Ptr(false),
				Error: &openapi.Error{
					ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.CodeString())),
					ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.Error),
					HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentInvalidRequest.ErrorMessage),
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

	totalItems, deployments, err := deploymentApi.deploymentService.GetAllAssistantWhatsappDeployment(c, auth, assistantId, criterias, paginate)
	if err != nil {
		deploymentApi.logger.Errorf("unable to get all assistant whatsapp deployments: %v", err)
		c.JSON(pkg_errors.GetAllAssistantWhatsappDeploymentGetDeployment.HTTPStatusCode, openapi.ErrorResponse{
			Code:    utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentGetDeployment.HTTPStatusCodeInt32()),
			Success: utils.Ptr(false),
			Error: &openapi.Error{
				ErrorCode:    utils.Ptr(openapi.Uint64String(pkg_errors.GetAllAssistantWhatsappDeploymentGetDeployment.CodeString())),
				ErrorMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentGetDeployment.Error),
				HumanMessage: utils.Ptr(pkg_errors.GetAllAssistantWhatsappDeploymentGetDeployment.ErrorMessage),
			},
		})
		return
	}

	responseDeployments := []openapi.AssistantWhatsappDeployment{}
	for _, deployment := range deployments {
		if !validator.NonNil(deployment) {
			continue
		}
		deploymentId := openapi.Uint64String(strconv.FormatUint(deployment.Id, 10))
		deploymentAssistantId := openapi.Uint64String(strconv.FormatUint(deployment.AssistantId, 10))
		deploymentStatus := deployment.Status.String()
		responseWhatsappOptions := []openapi.Metadata{}
		for _, whatsappOption := range deployment.WhatsappOptions {
			if !validator.NonNil(whatsappOption) {
				continue
			}
			responseWhatsappOptions = append(responseWhatsappOptions, openapi.Metadata{
				Key:   utils.Ptr(whatsappOption.Key),
				Value: utils.Ptr(whatsappOption.Value),
			})
		}
		whatsappProviderName := deployment.WhatsappProvider
		responseDeployments = append(responseDeployments, openapi.AssistantWhatsappDeployment{
			Id:                    &deploymentId,
			AssistantId:           &deploymentAssistantId,
			Greeting:              deployment.Greeting,
			GreetingInterruptible: deployment.GreetingInterruptible,
			Mistake:               deployment.Mistake,
			WhatsappProviderName:  &whatsappProviderName,
			WhatsappOptions:       &responseWhatsappOptions,
			Status:                &deploymentStatus,
			MaxSessionDuration:    deployment.MaxSessionDuration,
			IdealTimeout:          deployment.IdleTimeout,
			IdealTimeoutBackoff:   deployment.IdleTimeoutBackoff,
			IdealTimeoutMessage:   deployment.IdleTimeoutMessage,
		})
	}
	totalItem := uint32(totalItems)
	currentPage := paginate.GetPage()
	c.JSON(http.StatusOK, openapi.GetAllAssistantWhatsappDeploymentResponse{
		Code:    utils.Ptr(int32(http.StatusOK)),
		Success: utils.Ptr(true),
		Data:    &responseDeployments,
		Paginated: &openapi.Paginated{
			TotalItem:   &totalItem,
			CurrentPage: &currentPage,
		},
	})
}
