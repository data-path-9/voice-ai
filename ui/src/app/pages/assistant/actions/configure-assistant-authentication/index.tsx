import { FC, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  ButtonSet,
  Select as CarbonSelect,
  SelectItem,
  Toggle,
} from '@carbon/react';
import { Add, ArrowRight, TrashCan } from '@carbon/icons-react';
import { TextInput, Stack } from '@/app/components/carbon/form';
import {
  PrimaryButton,
  SecondaryButton,
  TertiaryButton,
} from '@/app/components/carbon/button';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { InputGroup } from '@/app/components/input-group';
import { TypeKeySelector } from '@/app/pages/assistant/actions/configure-assistant-webhook/create-assistant-webhook';

type AuthProvider = 'api';
type HttpMethod = 'POST' | 'GET';
type FailBehavior = 'block' | 'guest';
type AuthParameterType =
  | 'event'
  | 'assistant'
  | 'client'
  | 'conversation'
  | 'argument'
  | 'metadata'
  | 'option'
  | 'analysis'
  | 'custom';

interface HeaderRow {
  key: string;
  value: string;
}

interface ParameterRow {
  type: AuthParameterType;
  key: string;
  value: string;
}

const getDefaultParameterKey = (type: AuthParameterType): string => {
  switch (type) {
    case 'event':
      return 'type';
    case 'assistant':
      return 'id';
    case 'client':
      return 'phone';
    case 'conversation':
      return 'messages';
    default:
      return '';
  }
};

export function ConfigureAssistantAuthenticationPage() {
  const { assistantId } = useParams();
  return (
    <>{assistantId && <ConfigureAssistantAuthentication assistantId={assistantId} />}</>
  );
}

const ConfigureAssistantAuthentication: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

  const [enabled, setEnabled] = useState(false);
  const [provider, setProvider] = useState<AuthProvider>('api');
  const [endpoint, setEndpoint] = useState('');
  const [method, setMethod] = useState<HttpMethod>('POST');
  const [timeout, setTimeoutValue] = useState('5000');
  const [failBehavior, setFailBehavior] = useState<FailBehavior>('block');
  const [headers, setHeaders] = useState<HeaderRow[]>([]);
  const [parameters, setParameters] = useState<ParameterRow[]>([
    { type: 'assistant', key: 'id', value: 'assistantId' },
    { type: 'conversation', key: 'id', value: 'conversationId' },
  ]);

  const canSave = useMemo(() => {
    if (!enabled) return true;
    if (!endpoint.trim()) return false;
    const headersValid = headers.every(
      header => !(header.key.trim() && !header.value.trim()),
    );
    const paramsValid = parameters.every(
      param => param.key.trim() && param.value.trim(),
    );
    const keys = parameters.map(param => `${param.type}.${param.key}`);
    const uniqueKeys = new Set(keys);
    return headersValid && paramsValid && keys.length === uniqueKeys.size;
  }, [enabled, endpoint, headers, parameters]);

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    setHeaders(prev =>
      prev.map((row, i) => (i === index ? { ...row, [field]: value } : row)),
    );
  };

  const updateParameter = (
    index: number,
    field: 'type' | 'key' | 'value',
    value: string,
  ) => {
    setParameters(prevParams =>
      prevParams.map((param, i) => {
        if (i !== index) return param;
        if (field === 'type') {
          const nextType = value as AuthParameterType;
          return {
            ...param,
            type: nextType,
            key: getDefaultParameterKey(nextType),
            value: '',
          };
        }
        return { ...param, [field]: value };
      }),
    );
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
            <h1 className="text-2xl font-light tracking-tight">Authentication</h1>
          </div>
        </div>

        <div className="flex-1 min-h-0 overflow-auto">
          <InputGroup title="Authentication Configuration">
            <Stack gap={6}>
              <Toggle
                id="assistant-auth-enabled"
                labelA="Disabled"
                labelB="Enabled"
                labelText="Enable authentication"
                toggled={enabled}
                onToggle={(value: boolean) => setEnabled(value)}
              />

              <CarbonSelect
                id="assistant-auth-provider"
                labelText="Provider"
                value={provider}
                onChange={e => setProvider(e.target.value as AuthProvider)}
                disabled={!enabled}
              >
                <SelectItem value="api" text="API" />
              </CarbonSelect>
            </Stack>
          </InputGroup>

          <InputGroup title="API Provider" initiallyExpanded={enabled}>
            <Stack gap={6}>
              <Stack gap={6} orientation="horizontal">
                <CarbonSelect
                  id="assistant-auth-method"
                  labelText="Method"
                  value={method}
                  onChange={e => setMethod(e.target.value as HttpMethod)}
                  disabled={!enabled || provider !== 'api'}
                >
                  <SelectItem value="POST" text="POST" />
                  <SelectItem value="GET" text="GET" />
                </CarbonSelect>
                <TextInput
                  id="assistant-auth-timeout"
                  labelText="Timeout (ms)"
                  value={timeout}
                  onChange={e => setTimeoutValue(e.target.value)}
                  placeholder="5000"
                  disabled={!enabled || provider !== 'api'}
                />
              </Stack>

              <TextInput
                id="assistant-auth-endpoint"
                labelText="Auth Endpoint"
                value={endpoint}
                onChange={e => setEndpoint(e.target.value)}
                placeholder="https://auth.example.com/resolve"
                disabled={!enabled || provider !== 'api'}
              />

              <CarbonSelect
                id="assistant-auth-fail-behavior"
                labelText="On auth failure"
                value={failBehavior}
                onChange={e => setFailBehavior(e.target.value as FailBehavior)}
                disabled={!enabled || provider !== 'api'}
              >
                <SelectItem value="block" text="Block session" />
                <SelectItem value="guest" text="Continue as guest" />
              </CarbonSelect>
            </Stack>
          </InputGroup>

          <InputGroup title={`Headers (${headers.length})`} childClass="space-y-4">
            <table className="w-full border-collapse border border-gray-200 dark:border-gray-700 text-sm [&_input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none [&_.cds--form-item]:!m-0">
              <thead>
                <tr className="bg-gray-50 dark:bg-gray-900">
                  <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/2">
                    Key
                  </th>
                  <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-1/2">
                    Value
                  </th>
                  <th className="border-b border-gray-200 dark:border-gray-700 w-8" />
                </tr>
              </thead>
              <tbody>
                {headers.length === 0 && (
                  <tr>
                    <td
                      colSpan={3}
                      className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400"
                    >
                      No headers yet. Click <strong>Add header</strong> below.
                    </td>
                  </tr>
                )}
                {headers.map((header, index) => (
                  <tr
                    key={index}
                    className="border-b border-gray-200 dark:border-gray-700 last:border-b-0"
                  >
                    <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                      <TextInput
                        id={`auth-header-key-${index}`}
                        labelText=""
                        hideLabel
                        value={header.key}
                        onChange={e => updateHeader(index, 'key', e.target.value)}
                        placeholder="Key"
                        size="md"
                        disabled={!enabled || provider !== 'api'}
                      />
                    </td>
                    <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                      <TextInput
                        id={`auth-header-val-${index}`}
                        labelText=""
                        hideLabel
                        value={header.value}
                        onChange={e => updateHeader(index, 'value', e.target.value)}
                        placeholder="Value"
                        size="md"
                        disabled={!enabled || provider !== 'api'}
                      />
                    </td>
                    <td className="p-0 text-center">
                      <Button
                        hasIconOnly
                        renderIcon={TrashCan}
                        iconDescription="Remove"
                        kind="danger--ghost"
                        size="sm"
                        onClick={() =>
                          setHeaders(headers.filter((_, i) => i !== index))
                        }
                        disabled={!enabled || provider !== 'api'}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <TertiaryButton
              size="md"
              renderIcon={Add}
              onClick={() => setHeaders([...headers, { key: '', value: '' }])}
              className="!w-full !max-w-none"
              disabled={!enabled || provider !== 'api'}
            >
              Add header
            </TertiaryButton>
          </InputGroup>

          <InputGroup
            title={`Request Parameters (${parameters.length})`}
            childClass="space-y-4"
          >
            <table className="w-full border-collapse border border-gray-200 dark:border-gray-700 text-sm [&_input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none [&_.cds--select-input]:!border-none [&_.cds--form-item]:!m-0">
              <thead>
                <tr className="bg-gray-50 dark:bg-gray-900">
                  <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-[140px]">
                    Type
                  </th>
                  <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700 w-[140px]">
                    Key
                  </th>
                  <th className="border-b border-r border-gray-200 dark:border-gray-700 w-8" />
                  <th className="text-left text-xs font-medium text-gray-500 dark:text-gray-400 px-3 py-2 border-b border-r border-gray-200 dark:border-gray-700">
                    Value
                  </th>
                  <th className="border-b border-gray-200 dark:border-gray-700 w-8" />
                </tr>
              </thead>
              <tbody>
                {parameters.map((param, index) => (
                  <tr
                    key={index}
                    className="border-b border-gray-200 dark:border-gray-700 last:border-b-0"
                  >
                    <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                      <CarbonSelect
                        id={`auth-param-type-${index}`}
                        labelText=""
                        hideLabel
                        value={param.type}
                        onChange={e => updateParameter(index, 'type', e.target.value)}
                        size="md"
                        disabled={!enabled || provider !== 'api'}
                      >
                        <SelectItem value="event" text="Event" />
                        <SelectItem value="assistant" text="Assistant" />
                        <SelectItem value="client" text="Client" />
                        <SelectItem value="conversation" text="Conversation" />
                        <SelectItem value="argument" text="Argument" />
                        <SelectItem value="metadata" text="Metadata" />
                        <SelectItem value="option" text="Option" />
                        <SelectItem value="analysis" text="Analysis" />
                        <SelectItem value="custom" text="Custom" />
                      </CarbonSelect>
                    </td>
                    <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                      <TypeKeySelector
                        type={param.type}
                        value={param.key}
                        onChange={newKey => updateParameter(index, 'key', newKey)}
                      />
                    </td>
                    <td className="border-r border-gray-200 dark:border-gray-700 p-0 text-center text-gray-400">
                      <ArrowRight className="w-4 h-4 mx-auto" />
                    </td>
                    <td className="border-r border-gray-200 dark:border-gray-700 p-0">
                      <TextInput
                        id={`auth-param-val-${index}`}
                        labelText=""
                        hideLabel
                        value={param.value}
                        onChange={e => updateParameter(index, 'value', e.target.value)}
                        placeholder="Value"
                        size="md"
                        disabled={!enabled || provider !== 'api'}
                      />
                    </td>
                    <td className="p-0 text-center">
                      <Button
                        hasIconOnly
                        renderIcon={TrashCan}
                        iconDescription="Remove"
                        kind="danger--ghost"
                        size="sm"
                        onClick={() =>
                          setParameters(parameters.filter((_, i) => i !== index))
                        }
                        disabled={!enabled || provider !== 'api'}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <TertiaryButton
              size="md"
              renderIcon={Add}
              onClick={() =>
                setParameters([
                  ...parameters,
                  { type: 'assistant', key: 'id', value: '' },
                ])
              }
              className="!w-full !max-w-none"
              disabled={!enabled || provider !== 'api'}
            >
              Add parameter
            </TertiaryButton>
          </InputGroup>
        </div>

        <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
          <SecondaryButton
            size="lg"
            onClick={() => showDialog(navigator.goBack)}
          >
            Cancel
          </SecondaryButton>
          <PrimaryButton size="lg" disabled={!canSave}>
            Save authentication
          </PrimaryButton>
        </ButtonSet>
      </div>
    </>
  );
};
