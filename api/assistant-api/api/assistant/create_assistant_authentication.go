// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (assistantApi *assistantGrpcApi) CreateAssistantAuthentication(
	ctx context.Context,
	req *protos.CreateAssistantAuthenticationRequest,
) (*protos.GetAssistantAuthenticationResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAssistantAuthenticationResponse]()
	}

	authConfig, err := assistantApi.assistantAuthService.Create(
		ctx,
		iAuth,
		req.GetAssistantId(),
		req.GetProvider(),
		req.GetStatus(),
		req.GetFailBehavior(),
		req.GetTimeoutMs(),
		req.GetOptions(),
	)
	if err != nil {
		assistantApi.logger.Errorf("error while creating assistant authentication %s", err.Error())
		return exceptions.BadRequestError[protos.GetAssistantAuthenticationResponse](
			"Unable to create assistant authentication.",
		)
	}
	out := &protos.AssistantAuthentication{}
	if err = utils.Cast(authConfig, out); err != nil {
		assistantApi.logger.Errorf("unable to cast assistant authentication %v", err)
	}
	return utils.Success[protos.GetAssistantAuthenticationResponse, *protos.AssistantAuthentication](out)
}
