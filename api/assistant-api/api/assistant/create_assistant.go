// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"
	"errors"

	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	assistant_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/encoding/protojson"
)

// CreateAssistant implements assistant_api.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) CreateAssistant(ctx context.Context, cer *assistant_api.CreateAssistantRequest) (*assistant_api.GetAssistantResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {

		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantUnauthenticated.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantUnauthenticated.Code),
				ErrorMessage: pkg_errors.CreateAssistantUnauthenticated.Error,
				HumanMessage: pkg_errors.CreateAssistantUnauthenticated.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantUnauthenticated.Error)
	}
	if !iAuth.HasUser() {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAuthScope.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingAuthScope.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingAuthScope.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingAuthScope.Error)
	}
	if !iAuth.HasProject() {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAuthScope.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingAuthScope.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingAuthScope.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingAuthScope.Error)
	}
	if !iAuth.HasOrganization() {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingAuthScope.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAuthScope.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingAuthScope.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingAuthScope.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingAuthScope.Error)
	}
	if !validator.NotBlank(cer.GetName()) {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingName.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingName.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingName.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingName.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingName.Error)
	}
	if cer.GetAssistantProvider() == nil {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingProvider.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingProvider.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingProvider.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingProvider.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingProvider.Error)
	}
	if cer.GetAssistantProvider().GetAssistantProvider() == nil {
		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantMissingProvider.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantMissingProvider.Code),
				ErrorMessage: pkg_errors.CreateAssistantMissingProvider.Error,
				HumanMessage: pkg_errors.CreateAssistantMissingProvider.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantMissingProvider.Error)
	}
	if cer.GetAssistantProvider().GetModel() != nil {
		if !validator.NotBlank(cer.GetAssistantProvider().GetModel().GetModelProviderName()) {
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantMissingModelProviderName.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantMissingModelProviderName.Code),
					ErrorMessage: pkg_errors.CreateAssistantMissingModelProviderName.Error,
					HumanMessage: pkg_errors.CreateAssistantMissingModelProviderName.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantMissingModelProviderName.Error)
		}
	}
	if cer.GetAssistantProvider().GetAgentkit() != nil {
		if !validator.NotBlank(cer.GetAssistantProvider().GetAgentkit().GetAgentKitUrl()) {
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantMissingAgentKitURL.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantMissingAgentKitURL.Code),
					ErrorMessage: pkg_errors.CreateAssistantMissingAgentKitURL.Error,
					HumanMessage: pkg_errors.CreateAssistantMissingAgentKitURL.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantMissingAgentKitURL.Error)
		}
	}
	if cer.GetAssistantProvider().GetWebsocket() != nil {
		if !validator.NotBlank(cer.GetAssistantProvider().GetWebsocket().GetWebsocketUrl()) {
			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantMissingWebsocketURL.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantMissingWebsocketURL.Code),
					ErrorMessage: pkg_errors.CreateAssistantMissingWebsocketURL.Error,
					HumanMessage: pkg_errors.CreateAssistantMissingWebsocketURL.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantMissingWebsocketURL.Error)
		}
	}

	// creating assistant
	assistant, err := assistantApi.
		assistantService.
		CreateAssistant(
			ctx,
			iAuth,
			cer.GetName(),
			cer.GetDescription(),
			cer.GetVisibility(),
			cer.GetSource(),
			&cer.SourceIdentifier,
			cer.GetLanguage(),
		)
	if err != nil {

		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantCreateAssistant.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantCreateAssistant.Code),
				ErrorMessage: pkg_errors.CreateAssistantCreateAssistant.Error,
				HumanMessage: pkg_errors.CreateAssistantCreateAssistant.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantCreateAssistant.Error)
	}

	prd := cer.GetAssistantProvider().GetAssistantProvider()
	switch provider := prd.(type) {
	case *assistant_api.CreateAssistantProviderRequest_Model:
		providerModel, err := assistantApi.assistantService.CreateAssistantProviderModel(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			protojson.Format(provider.Model.GetTemplate()),
			provider.Model.GetModelProviderName(),
			provider.Model.GetAssistantModelOptions(),
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantCreateProviderModel.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantCreateProviderModel.Code),
					ErrorMessage: pkg_errors.CreateAssistantCreateProviderModel.Error,
					HumanMessage: pkg_errors.CreateAssistantCreateProviderModel.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantCreateProviderModel.Error)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.MODEL,
			providerModel.Id,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantAttachProviderModel.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantAttachProviderModel.Code),
					ErrorMessage: pkg_errors.CreateAssistantAttachProviderModel.Error,
					HumanMessage: pkg_errors.CreateAssistantAttachProviderModel.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantAttachProviderModel.Error)
		}

	case *assistant_api.CreateAssistantProviderRequest_Agentkit:
		agentKitProvider, err := assistantApi.assistantService.CreateAssistantProviderAgentkit(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			provider.Agentkit.GetAgentKitUrl(),
			provider.Agentkit.GetCertificate(),
			provider.Agentkit.GetMetadata(),
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantCreateProviderAgentkit.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantCreateProviderAgentkit.Code),
					ErrorMessage: pkg_errors.CreateAssistantCreateProviderAgentkit.Error,
					HumanMessage: pkg_errors.CreateAssistantCreateProviderAgentkit.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantCreateProviderAgentkit.Error)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.AGENTKIT,
			agentKitProvider.Id,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantAttachProviderAgentkit.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantAttachProviderAgentkit.Code),
					ErrorMessage: pkg_errors.CreateAssistantAttachProviderAgentkit.Error,
					HumanMessage: pkg_errors.CreateAssistantAttachProviderAgentkit.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantAttachProviderAgentkit.Error)
		}

	case *assistant_api.CreateAssistantProviderRequest_Websocket:
		websocketProvider, err := assistantApi.assistantService.CreateAssistantProviderWebsocket(
			ctx,
			iAuth,
			assistant.Id,
			cer.GetAssistantProvider().GetDescription(),
			provider.Websocket.GetWebsocketUrl(),
			provider.Websocket.GetHeaders(),
			provider.Websocket.GetConnectionParameters(),
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantCreateProviderWebsocket.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantCreateProviderWebsocket.Code),
					ErrorMessage: pkg_errors.CreateAssistantCreateProviderWebsocket.Error,
					HumanMessage: pkg_errors.CreateAssistantCreateProviderWebsocket.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantCreateProviderWebsocket.Error)
		}
		_, err = assistantApi.
			assistantService.AttachProviderModelToAssistant(
			ctx,
			iAuth,
			assistant.Id,
			type_enums.WEBSOCKET,
			websocketProvider.Id,
		)
		if err != nil {

			return &assistant_api.GetAssistantResponse{
				Code:    pkg_errors.CreateAssistantAttachProviderWebsocket.HTTPStatusCodeInt32(),
				Success: false,
				Error: &assistant_api.Error{
					ErrorCode:    uint64(pkg_errors.CreateAssistantAttachProviderWebsocket.Code),
					ErrorMessage: pkg_errors.CreateAssistantAttachProviderWebsocket.Error,
					HumanMessage: pkg_errors.CreateAssistantAttachProviderWebsocket.ErrorMessage,
				},
			}, errors.New(pkg_errors.CreateAssistantAttachProviderWebsocket.Error)
		}

	}

	for _, tl := range cer.GetAssistantTools() {
		_, err := assistantApi.createAssistantTool(
			ctx,
			iAuth,
			assistant.Id,
			tl)
		if err != nil {
			assistantApi.logger.Errorf("Unable to create assistant tools, please try again later with error %+v", err)
		}
	}

	for _, ak := range cer.GetAssistantKnowledges() {
		_, err := assistantApi.createAssistantKnowledge(
			ctx,
			iAuth,
			assistant.Id,
			ak)
		if err != nil {
			assistantApi.logger.Errorf("Unable to create assistant knowledge, please try again later with error %+v", err)
		}
	}

	_, err = assistantApi.assistantService.CreateOrUpdateAssistantTag(ctx, iAuth, assistant.Id, cer.GetTags())
	if err != nil {

		return &assistant_api.GetAssistantResponse{
			Code:    pkg_errors.CreateAssistantCreateTags.HTTPStatusCodeInt32(),
			Success: false,
			Error: &assistant_api.Error{
				ErrorCode:    uint64(pkg_errors.CreateAssistantCreateTags.Code),
				ErrorMessage: pkg_errors.CreateAssistantCreateTags.Error,
				HumanMessage: pkg_errors.CreateAssistantCreateTags.ErrorMessage,
			},
		}, errors.New(pkg_errors.CreateAssistantCreateTags.Error)
	}

	out := &assistant_api.Assistant{}
	err = utils.Cast(assistant, out)
	if err != nil {
		assistantApi.logger.Errorf("unable to cast the assistant provider model to the response object")
	}
	return utils.Success[assistant_api.GetAssistantResponse, *assistant_api.Assistant](out)
}
