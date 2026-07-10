// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/pkg/utils"
)

func (r *genericRequestor) deploymentBehavior() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
	assistant, err := r.Assistant()
	if err != nil {
		return nil, err
	}

	switch r.source {
	case utils.PhoneCall:
		if assistant.AssistantPhoneDeployment != nil {
			return &assistant.AssistantPhoneDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.Whatsapp:
		if assistant.AssistantWhatsappDeployment != nil {
			return &assistant.AssistantWhatsappDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.SDK:
		if assistant.AssistantApiDeployment != nil {
			return &assistant.AssistantApiDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.WebPlugin:
		if assistant.AssistantWebPluginDeployment != nil {
			return &assistant.AssistantWebPluginDeployment.AssistantDeploymentBehavior, nil
		}
	case utils.Debugger:
		if assistant.AssistantDebuggerDeployment != nil {
			return &assistant.AssistantDebuggerDeployment.AssistantDeploymentBehavior, nil
		}
	}

	return nil, errDeploymentNotEnabled
}
