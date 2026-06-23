import React, { FC, useState } from 'react';
import {
  CreateAssistantConfiguration,
  CreateAssistantConfigurationRequest,
  Metadata,
} from '@rapidaai/react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import toast from 'react-hot-toast/headless';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { Stack } from '@/app/components/carbon/form';
import { Notification } from '@/app/components/carbon/notification';
import { ButtonSet } from '@carbon/react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { TelemetryProvider } from '@/app/components/providers/telemetry';
import {
  GetDefaultTelemetryIfInvalid,
  ValidateTelemetry,
} from '@/app/components/providers/telemetry/provider';
import { TELEMETRY_PROVIDER } from '@/providers';
import { InputGroup } from '@/app/components/input-group';

const telemetryConfigurationType = 'telemetry';

export const CreateAssistantTelemetry: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const defaultProvider = TELEMETRY_PROVIDER[0]?.code || 'otlp_http';
  const [provider, setProvider] = useState(defaultProvider);
  const [parameters, setParameters] = useState<Metadata[]>(() =>
    GetDefaultTelemetryIfInvalid(defaultProvider, []),
  );
  const [errorMessage, setErrorMessage] = useState('');

  const onChangeProvider = (providerCode: string) => {
    setProvider(providerCode);
    const credentialOnly = parameters.filter(
      p => p.getKey() === 'rapida.credential_id',
    );
    setParameters(GetDefaultTelemetryIfInvalid(providerCode, credentialOnly));
  };

  const onSubmit = () => {
    setErrorMessage('');
    const validationError = ValidateTelemetry(provider, parameters);
    if (validationError) {
      setErrorMessage(validationError);
      return;
    }

    const request = new CreateAssistantConfigurationRequest();
    request.setAssistantid(assistantId);
    request.setConfigurationtype(telemetryConfigurationType);
    request.setProvider(provider);
    request.setEnabled(true);
    request.setOptionsList(parameters);

    showLoader();
    CreateAssistantConfiguration(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        hideLoader();
        if (response?.getSuccess()) {
          toast.success('Telemetry provider created successfully');
          navigator.goToAssistantTelemetry(assistantId);
          return;
        }
        setErrorMessage(
          response?.getError()?.getHumanmessage() ||
            'Unable to create telemetry provider. Please try again.',
        );
      })
      .catch(() => {
        hideLoader();
        setErrorMessage(
          'Unable to create telemetry provider. Please try again.',
        );
      });
  };

  return (
    <>
      <ConfirmDialogComponent />
      <section className="flex flex-1 min-h-0 overflow-hidden">
        <div className="flex-1 min-h-0 flex flex-col bg-white dark:bg-gray-900">
          <div className="flex-1 min-h-0 overflow-y-auto flex flex-col">
            <div className="flex flex-col flex-1">
              <header className="px-8 pt-8 pb-6 border-b border-gray-200 dark:border-gray-800">
                <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 leading-tight">
                  Telemetry
                </h1>
                <p className="text-sm text-gray-500 dark:text-gray-500 mt-1.5 leading-relaxed">
                  Configure the provider and destination.
                </p>
              </header>

              <div className="pb-8 flex flex-col">
                <InputGroup title="Destination">
                  <Stack gap={6}>
                    <TelemetryProvider
                      provider={provider}
                      onChangeProvider={onChangeProvider}
                      parameters={parameters}
                      onChangeParameter={setParameters}
                    />
                  </Stack>
                </InputGroup>
              </div>
            </div>
          </div>

          <div className="shrink-0">
            {errorMessage && (
              <Notification
                kind="error"
                title="Error"
                subtitle={errorMessage}
              />
            )}
            <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
              <SecondaryButton
                size="lg"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </SecondaryButton>
              <PrimaryButton size="lg" isLoading={loading} onClick={onSubmit}>
                Configure telemetry
              </PrimaryButton>
            </ButtonSet>
          </div>
        </div>
      </section>
    </>
  );
};
