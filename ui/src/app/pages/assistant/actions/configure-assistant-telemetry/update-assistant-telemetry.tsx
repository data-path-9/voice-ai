import React, { FC, useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  GetAssistantConfiguration,
  GetAssistantConfigurationRequest,
  Metadata,
  UpdateAssistantConfiguration,
  UpdateAssistantConfigurationRequest,
} from '@rapidaai/react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import toast from 'react-hot-toast/headless';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ButtonSet } from '@carbon/react';
import { Stack } from '@/app/components/carbon/form';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { TelemetryProvider } from '@/app/components/providers/telemetry';
import {
  GetDefaultTelemetryIfInvalid,
  ValidateTelemetry,
} from '@/app/components/providers/telemetry/provider';
import { InputGroup } from '@/app/components/input-group';
import { Notification } from '@/app/components/carbon/notification';

const telemetryConfigurationType = 'telemetry';

export const UpdateAssistantTelemetry: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { telemetryId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const [provider, setProvider] = useState('');
  const [parameters, setParameters] = useState<Metadata[]>([]);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    if (!telemetryId) return;

    const request = new GetAssistantConfigurationRequest();
    request.setAssistantid(assistantId);
    request.setId(telemetryId);

    showLoader();
    GetAssistantConfiguration(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        hideLoader();
        if (!response?.getSuccess()) {
          toast.error('Unable to load telemetry provider');
          return;
        }

        const telemetry = response.getData();
        if (!telemetry) return;

        const loadedProvider = telemetry.getProvider();
        setProvider(loadedProvider);

        const loadedParams = telemetry.getOptionsList().map(opt => {
          const m = new Metadata();
          m.setKey(opt.getKey());
          m.setValue(opt.getValue());
          return m;
        });
        setParameters(
          GetDefaultTelemetryIfInvalid(loadedProvider, loadedParams),
        );
      })
      .catch(() => {
        hideLoader();
        toast.error('Unable to load telemetry provider');
      });
  }, [assistantId, telemetryId, authId, token, projectId]);

  const onChangeProvider = (providerCode: string) => {
    setProvider(providerCode);
    const credentialOnly = parameters.filter(
      p => p.getKey() === 'rapida.credential_id',
    );
    setParameters(GetDefaultTelemetryIfInvalid(providerCode, credentialOnly));
  };

  const onChangeParameter = (params: Metadata[]) => {
    setParameters(params);
  };

  const onSubmit = () => {
    if (!telemetryId) return;
    setErrorMessage('');

    const validationError = ValidateTelemetry(provider, parameters);
    if (validationError) {
      setErrorMessage(validationError);
      return;
    }

    const request = new UpdateAssistantConfigurationRequest();
    request.setId(telemetryId);
    request.setAssistantid(assistantId);
    request.setConfigurationtype(telemetryConfigurationType);
    request.setProvider(provider);
    request.setEnabled(true);
    request.setOptionsList(parameters);

    showLoader();
    UpdateAssistantConfiguration(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        hideLoader();
        if (response?.getSuccess()) {
          toast.success('Assistant telemetry provider updated successfully');
          navigator.goToAssistantTelemetry(assistantId);
          return;
        }

        const message = response?.getError()?.getHumanmessage();
        setErrorMessage(
          message ||
            'Unable to update assistant telemetry provider, please try again.',
        );
      })
      .catch(() => {
        hideLoader();
        setErrorMessage(
          'Unable to update assistant telemetry provider, please try again.',
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
              <header className="px-4 pt-8 pb-6 border-b border-gray-200 dark:border-gray-800">
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
                      onChangeParameter={onChangeParameter}
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
                Update telemetry
              </PrimaryButton>
            </ButtonSet>
          </div>
        </div>
      </section>
    </>
  );
};
