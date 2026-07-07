import {
  AgentflowBuilder,
  AgentflowDefinition,
} from '@/app/pages/assistant/actions/agentflow';
import { connectionConfig } from '@/configs';
import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  CreateAssistant,
  CreateAssistantProviderRequest,
  CreateAssistantRequest,
} from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import type { JavaScriptValue } from 'google-protobuf/google/protobuf/struct_pb';

const toAgentflowStruct = (definition: AgentflowDefinition) =>
  Struct.fromJavaScript(
    JSON.parse(JSON.stringify(definition)) as Record<string, JavaScriptValue>,
  );

export function CreateAgentflow() {
  const { authId, token, projectId } = useCurrentCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const navigator = useGlobalNavigation();

  const saveAgentflow = async (definition: AgentflowDefinition) => {
    showLoader('overlay');
    try {
      const assistantProvider = new CreateAssistantProviderRequest();
      const agentflow =
        new CreateAssistantProviderRequest.CreateAssistantProviderAgentflow();
      agentflow.setSchemaversion(definition.schemaVersion);
      agentflow.setDefinition(toAgentflowStruct(definition));
      assistantProvider.setAgentflow(agentflow);
      assistantProvider.setDescription(definition.description ?? '');

      const request = new CreateAssistantRequest();
      request.setAssistantprovider(assistantProvider);
      request.setName(definition.name ?? 'Agentflow');
      request.setDescription(definition.description ?? '');
      request.setTagsList(definition.tags ?? []);

      const response = await CreateAssistant(connectionConfig, request, {
        authorization: token,
        'x-auth-id': authId,
        'x-project-id': projectId,
      });

      if (!response?.getSuccess()) {
        throw new Error(
          response?.getError()?.getHumanmessage() ??
            'Unable to create agentflow assistant.',
        );
      }

      const assistant = response.getData();
      if (assistant?.getId()) {
        navigator.goToAssistantVersions(assistant.getId());
      }
    } finally {
      hideLoader();
    }
  };

  return <AgentflowBuilder title="Create Agentflow" onSave={saveAgentflow} />;
}
