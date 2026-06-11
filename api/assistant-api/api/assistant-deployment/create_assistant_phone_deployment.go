// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_deployment_api

import (
	"context"
	"errors"

	internal_assistant_service "github.com/rapidaai/api/assistant-api/internal/services/assistant"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	assistant_api "github.com/rapidaai/protos"
)

// CreateAssistantPhoneDeployment implements assistant_api.AssistantDeploymentServiceServer.
func (deploymentApi *assistantDeploymentGrpcApi) CreateAssistantPhoneDeployment(ctx context.Context, deployment *assistant_api.CreateAssistantDeploymentRequest) (*assistant_api.GetAssistantPhoneDeploymentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || iAuth.GetCurrentProjectId() == nil {
		deploymentApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("unauthenticated request for create assistant phone deployment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	// name, role, tone, expertise, greeting, mistake, ending string,
	//
	if deployment.GetPhone() == nil {
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("illegal parameters attached to deployment"),
			"Please check and provide valid deployment request for phone.",
		)
	}
	wpDeployment, err := deploymentApi.deploymentService.CreatePhoneDeployment(ctx,
		iAuth, deployment.GetPhone().GetAssistantId(),
		deployment.GetPhone().Greeting,
		deployment.GetPhone().Mistake,
		&deployment.GetPhone().IdealTimeout,
		&deployment.GetPhone().IdealTimeoutBackoff,
		&deployment.GetPhone().IdealTimeoutMessage,
		&deployment.GetPhone().MaxSessionDuration,
		deployment.GetPhone().GetPhoneProviderName(),
		deployment.GetPhone().GetInputAudio(),
		deployment.GetPhone().GetOutputAudio(),
		deployment.GetPhone().GetPhoneOptions(),
	)

	if err != nil {
		// Surface Vobiz inbound-provisioning messages (e.g. "number already
		// attached") to the UI; keep the generic message for every other error
		// so non-vobiz flows are unchanged. The provisioning error carries its
		// underlying cause (Unwrap) which we log for debugging.
		var provErr *internal_assistant_service.InboundProvisioningError
		if errors.As(err, &provErr) {
			deploymentApi.logger.Errorf("vobiz inbound provisioning failed: %s (cause: %v)", provErr.Message, provErr.Err)
			return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](err, provErr.Error())
		}
		deploymentApi.logger.Errorf("create assistant phone deployment failed: %v", err)
		return utils.Error[assistant_api.GetAssistantPhoneDeploymentResponse](
			errors.New("illegal request for create assistant phone deployment"),
			"Please provider valid a valid request to create assistant phone deployment.",
		)
	}
	return utils.Success[assistant_api.GetAssistantPhoneDeploymentResponse](wpDeployment)
}
