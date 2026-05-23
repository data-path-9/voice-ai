import { FC, useEffect, useState } from 'react';
import {
  CreateAssistantAuthentication,
  CreateAssistantAuthenticationRequest,
  DisableAssistantAuthentication,
  DisableAssistantAuthenticationRequest,
  GetAssistantAuthentication,
  GetAssistantAuthenticationRequest,
  Metadata,
} from '@rapidaai/react';
import {
  Breadcrumb,
  BreadcrumbItem,
  ButtonSet,
  CheckboxGroup,
  Select as CarbonSelect,
  SelectItem,
  Slider,
} from '@carbon/react';
import toast from 'react-hot-toast/headless';

import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { connectionConfig } from '@/configs';
import { Notification } from '@/app/components/carbon/notification';
import { InputCheckbox } from '@/app/components/carbon/form/input-checkbox';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { InputGroup } from '@/app/components/input-group';
import { APiStringHeader } from '@/app/components/external-api/api-header';
import {
  ASSISTANT_CONDITION_KEY_OPTIONS,
  ASSISTANT_CONDITION_OPERATOR_OPTIONS,
  ASSISTANT_CONDITION_SOURCE_OPTIONS,
  ASSISTANT_CONDITION_VALUE_OPTIONS_BY_KEY,
  ParameterEditor,
  normalizeAssistantConditionEntries,
} from '@/app/components/tools/common';
import { SourceConditionRule } from '@/app/components/conditions/source-condition-rule';
import { Stack, TextInput } from '@/app/components/carbon/form';

import {
  AUTH_KEY_OPTIONS_BY_TYPE,
  AUTH_OPTION_BODY,
  AUTH_OPTION_CONDITION,
  AUTH_OPTION_ENDPOINT,
  AUTH_OPTION_HEADERS,
  AUTH_OPTION_METHOD,
  AUTH_PARAMETER_TYPE_OPTIONS,
  DEFAULT_BODY,
  DEFAULT_HEADERS,
  DEFAULT_SOURCE_CONDITIONS,
  DEFAULT_TIMEOUT_MS,
  FailBehavior,
  fromApiFailBehavior,
  HttpMethod,
  toApiFailBehavior,
  toOptionMap,
} from './shared';

interface SharedAuthenticationFormProps {
  assistantId: string;
  loadExisting: boolean;
}

const getDefaultConditions = () =>
  DEFAULT_SOURCE_CONDITIONS.map(c => ({ ...c }));

const AuthenticationFormBase: FC<SharedAuthenticationFormProps> = ({
  assistantId,
  loadExisting,
}) => {
  const navigator = useGlobalNavigation();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const { authId, token, projectId } = useCurrentCredential();

  const [enabled, setEnabled] = useState(false);
  const [endpoint, setEndpoint] = useState('');
  const [method, setMethod] = useState<HttpMethod>('POST');
  const [timeout, setTimeoutValue] = useState(DEFAULT_TIMEOUT_MS);
  const [failBehavior, setFailBehavior] = useState<FailBehavior>('block');
  const [headers, setHeaders] = useState(DEFAULT_HEADERS);
  const [body, setBody] = useState(DEFAULT_BODY);
  const [sourceConditions, setSourceConditions] =
    useState(getDefaultConditions);

  const [errorMessage, setErrorMessage] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [isInitializing, setIsInitializing] = useState(loadExisting);
  const [hasAuthentication, setHasAuthentication] = useState(false);

  const resetForm = () => {
    setErrorMessage('');
    setHasAuthentication(false);
    setEnabled(false);
    setEndpoint('');
    setMethod('POST');
    setTimeoutValue(DEFAULT_TIMEOUT_MS);
    setFailBehavior('block');
    setHeaders(DEFAULT_HEADERS);
    setBody(DEFAULT_BODY);
    setSourceConditions(getDefaultConditions());
  };

  const loadAuthentication = () => {
    setIsInitializing(true);
    resetForm();

    const request = new GetAssistantAuthenticationRequest();
    request.setAssistantid(assistantId);

    GetAssistantAuthentication(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        if (!response?.getSuccess()) {
          setIsInitializing(false);
          return;
        }

        const data = response.getData();
        if (!data) {
          setIsInitializing(false);
          return;
        }

        setHasAuthentication(true);
        setEnabled((data.getStatus() || '').toLowerCase() === 'active');
        setFailBehavior(fromApiFailBehavior(data.getFailbehavior()));

        const persistedTimeout = Number(data.getTimeoutms());
        setTimeoutValue(
          Number.isFinite(persistedTimeout) && persistedTimeout > 0
            ? persistedTimeout
            : DEFAULT_TIMEOUT_MS,
        );

        const optionMap = toOptionMap(data.getOptionsList() || []);
        const persistedMethod = optionMap[AUTH_OPTION_METHOD];
        if (persistedMethod === 'POST' || persistedMethod === 'GET') {
          setMethod(persistedMethod);
        }

        if (optionMap[AUTH_OPTION_ENDPOINT]) {
          setEndpoint(optionMap[AUTH_OPTION_ENDPOINT]);
        }
        if (optionMap[AUTH_OPTION_HEADERS]) {
          setHeaders(optionMap[AUTH_OPTION_HEADERS]);
        }
        if (optionMap[AUTH_OPTION_BODY]) {
          setBody(optionMap[AUTH_OPTION_BODY]);
        }

        if (optionMap[AUTH_OPTION_CONDITION]) {
          try {
            setSourceConditions(
              normalizeAssistantConditionEntries(
                JSON.parse(optionMap[AUTH_OPTION_CONDITION]),
              ),
            );
          } catch {
            setSourceConditions(getDefaultConditions());
          }
        }

        setIsInitializing(false);
      })
      .catch(() => {
        setErrorMessage(
          'Unable to load assistant authentication. Please try again.',
        );
        setIsInitializing(false);
      });
  };

  useEffect(() => {
    if (!loadExisting) {
      resetForm();
      setIsInitializing(false);
      return;
    }
    loadAuthentication();
  }, [assistantId, authId, token, projectId, loadExisting]);

  const validateEnabledConfiguration = (): boolean => {
    if (!endpoint.trim()) {
      setErrorMessage('Please provide a server URL for authentication.');
      return false;
    }

    if (!/^https?:\/\/.+/.test(endpoint.trim())) {
      setErrorMessage('Please provide a valid server URL for authentication.');
      return false;
    }

    let parsedHeaders: unknown;
    try {
      parsedHeaders = JSON.parse(headers || '{}');
    } catch {
      setErrorMessage('Please provide valid values for headers key and value.');
      return false;
    }

    if (
      typeof parsedHeaders !== 'object' ||
      parsedHeaders === null ||
      Array.isArray(parsedHeaders)
    ) {
      setErrorMessage('Please provide valid values for headers key and value.');
      return false;
    }

    const hasInvalidHeader = Object.entries(
      parsedHeaders as Record<string, unknown>,
    ).some(
      ([key, value]) =>
        !key.trim() || typeof value !== 'string' || !value.trim(),
    );
    if (hasInvalidHeader) {
      setErrorMessage('Please provide valid values for headers key and value.');
      return false;
    }

    let parsedBody: unknown;
    try {
      parsedBody = JSON.parse(body || '{}');
    } catch {
      setErrorMessage(
        'Please provide valid values for parameters key and value.',
      );
      return false;
    }

    if (
      typeof parsedBody !== 'object' ||
      parsedBody === null ||
      Array.isArray(parsedBody)
    ) {
      setErrorMessage(
        'Please provide valid values for parameters key and value.',
      );
      return false;
    }

    const bodyEntries = Object.entries(parsedBody as Record<string, unknown>);
    if (bodyEntries.length === 0) {
      setErrorMessage(
        'Please provide one or more parameters for authentication.',
      );
      return false;
    }

    const hasInvalidBodyEntry = bodyEntries.some(
      ([key, value]) =>
        !key.trim() || typeof value !== 'string' || !value.trim(),
    );
    if (hasInvalidBodyEntry) {
      setErrorMessage(
        'Please provide valid values for parameters key and value.',
      );
      return false;
    }

    return true;
  };

  const validateBeforeSave = (): boolean => {
    setErrorMessage('');
    if (!enabled) return true;
    return validateEnabledConfiguration();
  };

  const saveDisabledAuthentication = async () => {
    const request = new DisableAssistantAuthenticationRequest();
    request.setAssistantid(assistantId);
    const response = await DisableAssistantAuthentication(
      connectionConfig,
      request,
      {
        'x-auth-id': authId,
        authorization: token,
        'x-project-id': projectId,
      },
    );

    if (response?.getSuccess()) {
      toast.success('Assistant authentication disabled successfully.');
      navigator.goTo(
        `/deployment/assistant/${assistantId}/configure-authentication`,
      );
      return;
    }

    setErrorMessage(
      response?.getError()?.getHumanmessage() ||
        'Unable to disable assistant authentication.',
    );
  };

  const saveEnabledAuthentication = async () => {
    const request = new CreateAssistantAuthenticationRequest();
    request.setAssistantid(assistantId);
    request.setProvider('http');
    request.setStatus('ACTIVE');
    request.setFailbehavior(toApiFailBehavior(failBehavior));
    request.setTimeoutms(String(timeout));

    const options: Metadata[] = [];
    const addOption = (key: string, value: string) => {
      const metadata = new Metadata();
      metadata.setKey(key);
      metadata.setValue(value);
      options.push(metadata);
    };

    addOption(AUTH_OPTION_METHOD, method);
    addOption(AUTH_OPTION_ENDPOINT, endpoint.trim());
    addOption(AUTH_OPTION_HEADERS, headers || DEFAULT_HEADERS);
    addOption(AUTH_OPTION_BODY, body || DEFAULT_BODY);
    addOption(AUTH_OPTION_CONDITION, JSON.stringify(sourceConditions));
    request.setOptionsList(options);

    const response = await CreateAssistantAuthentication(
      connectionConfig,
      request,
      {
        'x-auth-id': authId,
        authorization: token,
        'x-project-id': projectId,
      },
    );

    if (response?.getSuccess()) {
      toast.success('Assistant authentication saved successfully.');
      navigator.goTo(
        `/deployment/assistant/${assistantId}/configure-authentication`,
      );
      return;
    }

    setErrorMessage(
      response?.getError()?.getHumanmessage() ||
        'Unable to save assistant authentication.',
    );
  };

  const onSubmit = async () => {
    if (isInitializing || isSaving) return;
    if (!validateBeforeSave()) return;

    setIsSaving(true);
    try {
      if (!enabled) {
        await saveDisabledAuthentication();
        return;
      }
      await saveEnabledAuthentication();
    } catch (err: any) {
      setErrorMessage(
        err?.message || 'Unable to save assistant authentication.',
      );
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <>
      <ConfirmDialogComponent />
      <div className="flex flex-col flex-1 min-h-0 bg-white dark:bg-gray-900">
        <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
          <div>
            <Breadcrumb noTrailingSlash className="mb-2">
              <BreadcrumbItem
                href={`/deployment/assistant/${assistantId}/overview`}
              >
                Assistant
              </BreadcrumbItem>
            </Breadcrumb>
            <h1 className="text-2xl font-light tracking-tight">
              {hasAuthentication ? 'Edit Authentication' : 'Add Authentication'}
            </h1>
          </div>
        </div>

        <div className="flex-1 min-h-0 overflow-auto">
          <div className="p-6">
            <CheckboxGroup
              legendText=""
              warn
              warnText={
                enabled
                  ? 'All sessions must be verified before initialization.'
                  : 'Authentication is disabled. Sessions will continue without verification.'
              }
            >
              <InputCheckbox
                id="assistant-auth-enabled"
                checked={enabled}
                disabled={isSaving || isInitializing}
                onChange={e => {
                  setEnabled(e.target.checked);
                  setErrorMessage('');
                }}
              >
                Enable Session Authentication
              </InputCheckbox>
            </CheckboxGroup>
          </div>

          {enabled && (
            <>
              <InputGroup title="Condition">
                <SourceConditionRule
                  conditions={sourceConditions}
                  onChangeConditions={setSourceConditions}
                  conditionOptions={ASSISTANT_CONDITION_OPERATOR_OPTIONS}
                  sourceOptions={ASSISTANT_CONDITION_SOURCE_OPTIONS}
                  keyOptions={ASSISTANT_CONDITION_KEY_OPTIONS}
                  valueOptionsByKey={ASSISTANT_CONDITION_VALUE_OPTIONS_BY_KEY}
                  keyTooltipText="The variable to evaluate for this condition."
                />
              </InputGroup>

              <InputGroup title="Definition">
                <Stack gap={7}>
                  <div className="flex space-x-2">
                    <div className="relative w-40">
                      <CarbonSelect
                        id="assistant-auth-method"
                        labelText="Method"
                        value={method}
                        onChange={e => {
                          setMethod(e.target.value as HttpMethod);
                          setErrorMessage('');
                        }}
                      >
                        <SelectItem value="POST" text="POST" />
                        <SelectItem value="GET" text="GET" />
                      </CarbonSelect>
                    </div>
                    <div className="relative w-full">
                      <TextInput
                        id="assistant-auth-endpoint"
                        labelText="Server URL"
                        value={endpoint}
                        onChange={e => {
                          setEndpoint(e.target.value);
                          setErrorMessage('');
                        }}
                        placeholder="https://auth.example.com/resolve"
                      />
                    </div>
                  </div>

                  <div className="flex space-x-2">
                    <div className="relative w-40">
                      <CarbonSelect
                        id="assistant-auth-fail-behavior"
                        labelText="On Error"
                        value={failBehavior}
                        onChange={e => {
                          setFailBehavior(e.target.value as FailBehavior);
                          setErrorMessage('');
                        }}
                      >
                        <SelectItem value="block" text="Block" />
                        <SelectItem value="do_nothing" text="Do nothing" />
                      </CarbonSelect>
                    </div>
                    <div className="relative w-full">
                      <Slider
                        id="assistant-auth-timeout"
                        labelText="Timeout (ms)"
                        value={timeout}
                        min={500}
                        max={10000}
                        step={100}
                        onChange={(data: { value: number | number[] }) => {
                          setTimeoutValue(
                            Array.isArray(data.value)
                              ? data.value[0]
                              : data.value,
                          );
                          setErrorMessage('');
                        }}
                      />
                    </div>
                  </div>

                  <div>
                    <p className="text-xs font-medium mb-2">Headers</p>
                    <APiStringHeader
                      headerValue={headers}
                      setHeaderValue={value => {
                        setHeaders(value);
                        setErrorMessage('');
                      }}
                    />
                  </div>

                  <ParameterEditor
                    value={body}
                    onChange={value => {
                      setBody(value);
                      setErrorMessage('');
                    }}
                    typeOptions={AUTH_PARAMETER_TYPE_OPTIONS}
                    keyOptionsByType={AUTH_KEY_OPTIONS_BY_TYPE}
                    includeEmptyKeyOption
                  />
                </Stack>
              </InputGroup>
            </>
          )}
        </div>

        <div className="shrink-0 w-full">
          {errorMessage && (
            <Notification kind="error" title="Error" subtitle={errorMessage} />
          )}
          <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
            <SecondaryButton
              size="lg"
              onClick={() =>
                showDialog(() =>
                  navigator.goTo(
                    `/deployment/assistant/${assistantId}/configure-authentication`,
                  ),
                )
              }
            >
              Cancel
            </SecondaryButton>
            <PrimaryButton
              size="lg"
              onClick={onSubmit}
              isLoading={isSaving}
              disabled={isSaving || isInitializing}
            >
              Save authentication
            </PrimaryButton>
          </ButtonSet>
        </div>
      </div>
    </>
  );
};

interface AuthenticationFormProps {
  assistantId: string;
}

export const CreateAuthenticationForm: FC<AuthenticationFormProps> = ({
  assistantId,
}) => <AuthenticationFormBase assistantId={assistantId} loadExisting={false} />;

export const EditAuthenticationForm: FC<AuthenticationFormProps> = ({
  assistantId,
}) => <AuthenticationFormBase assistantId={assistantId} loadExisting />;
