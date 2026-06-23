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
import {
  CloudStorageProvider,
  GetDefaultStorageConfigIfInvalid,
  ValidateStorageOptions,
} from '@/app/components/providers/storage';
import { InputGroup } from '@/app/components/input-group';
import { Notification } from '@/app/components/carbon/notification';

const storageConfigurationType = 'storage';

export const UpdateAssistantStorage: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { storageId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const [provider, setProvider] = useState('');
  const [parameters, setParameters] = useState<Metadata[]>([]);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    if (!storageId) return;

    const request = new GetAssistantConfigurationRequest();
    request.setAssistantid(assistantId);
    request.setId(storageId);

    showLoader();
    GetAssistantConfiguration(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        hideLoader();
        if (!response?.getSuccess()) {
          toast.error('Unable to load storage');
          return;
        }

        const storage = response.getData();
        if (!storage) return;

        const loadedProvider = storage.getProvider();
        setProvider(loadedProvider);
        setParameters(
          GetDefaultStorageConfigIfInvalid(
            loadedProvider,
            storage.getOptionsList().map(option => {
              const metadata = new Metadata();
              metadata.setKey(option.getKey());
              metadata.setValue(option.getValue());
              return metadata;
            }),
          ),
        );
      })
      .catch(() => {
        hideLoader();
        toast.error('Unable to load storage');
      });
  }, [assistantId, authId, projectId, storageId, token]);

  const onChangeProvider = (providerCode: string) => {
    setProvider(providerCode);
    setParameters(
      GetDefaultStorageConfigIfInvalid(
        providerCode,
        parameters.filter(param => param.getKey() === 'rapida.credential_id'),
      ),
    );
  };

  const onSubmit = () => {
    if (!storageId) return;
    setErrorMessage('');

    if (!provider) {
      setErrorMessage('Please select a storage provider.');
      return;
    }

    if (!ValidateStorageOptions(provider, parameters)) {
      setErrorMessage('Please complete the required storage fields.');
      return;
    }

    const request = new UpdateAssistantConfigurationRequest();
    request.setId(storageId);
    request.setAssistantid(assistantId);
    request.setConfigurationtype(storageConfigurationType);
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
          toast.success('Storage updated successfully');
          navigator.goToAssistantStorage(assistantId);
          return;
        }

        const message = response?.getError()?.getHumanmessage();
        setErrorMessage(
          message || 'Unable to update assistant storage, please try again.',
        );
      })
      .catch(() => {
        hideLoader();
        setErrorMessage(
          'Unable to update assistant storage, please try again.',
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
                  Storage
                </h1>
                <p className="text-sm text-gray-500 dark:text-gray-500 mt-1.5 leading-relaxed">
                  Configure the provider and destination.
                </p>
              </header>

              <div className="pb-8 flex flex-col">
                <InputGroup title="Destination">
                  <Stack gap={6}>
                    <CloudStorageProvider
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
                Update storage
              </PrimaryButton>
            </ButtonSet>
          </div>
        </div>
      </section>
    </>
  );
};
