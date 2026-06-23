import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

import { UpdateAssistantAnalysis } from '@/app/pages/assistant/actions/configure-assistant-analysis/update-assistant-analysis';
import {
  GetAssistantConfiguration,
  UpdateAssistantConfiguration,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';

const mockGoBack = jest.fn();
const mockGoToConfigureAssistantAnalysis = jest.fn();
const mockToastError = jest.fn();

let mockParams: Record<string, string | undefined> = {
  analysisId: 'analysis-1',
};

jest.mock('@rapidaai/react', () => ({
  ConnectionConfig: class ConnectionConfig {},
  Metadata: class Metadata {
    key = '';
    value = '';
    setKey(v: string) {
      this.key = v;
    }
    setValue(v: string) {
      this.value = v;
    }
    getKey() {
      return this.key;
    }
    getValue() {
      return this.value;
    }
  },
  GetAssistantConfigurationRequest: class GetAssistantConfigurationRequest {
    id = '';
    assistantId = '';
    setId(v: string) {
      this.id = v;
    }
    getId() {
      return this.id;
    }
    setAssistantid(v: string) {
      this.assistantId = v;
    }
    getAssistantid() {
      return this.assistantId;
    }
  },
  UpdateAssistantConfigurationRequest: class UpdateAssistantConfigurationRequest {
    id = '';
    assistantId = '';
    provider = '';
    configurationType = '';
    enabled = false;
    optionsList: any[] = [];
    setId(v: string) {
      this.id = v;
    }
    getId() {
      return this.id;
    }
    setAssistantid(v: string) {
      this.assistantId = v;
    }
    getAssistantid() {
      return this.assistantId;
    }
    setProvider(v: string) {
      this.provider = v;
    }
    getProvider() {
      return this.provider;
    }
    setConfigurationtype(v: string) {
      this.configurationType = v;
    }
    getConfigurationtype() {
      return this.configurationType;
    }
    setEnabled(v: boolean) {
      this.enabled = v;
    }
    getEnabled() {
      return this.enabled;
    }
    setOptionsList(v: any[]) {
      this.optionsList = v;
    }
    getOptionsList() {
      return this.optionsList;
    }
  },
  GetAssistantConfiguration: jest.fn(),
  UpdateAssistantConfiguration: jest.fn(),
}));

jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: {
    success: jest.fn(),
    error: (...args: any[]) => mockToastError(...args),
  },
}));

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => mockParams,
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: mockGoBack,
    goToConfigureAssistantAnalysis: mockGoToConfigureAssistantAnalysis,
  }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'auth-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/utils', () => ({
  cn: (...inputs: any[]) => inputs.filter(Boolean).join(' '),
  randomMeaningfullName: () => 'analysis-default',
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (cb: () => void) => cb(),
    ConfirmDialogComponent: () => null,
  }),
}));

jest.mock('@/app/components/dropdown/endpoint-dropdown', () => ({
  EndpointDropdown: ({ onChangeEndpoint }: any) => (
    <button
      type="button"
      onClick={() =>
        onChangeEndpoint({
          getId: () => 'endpoint-2',
        })
      }
    >
      Pick endpoint
    </button>
  ),
}));

jest.mock('@/app/components/form/tab-form', () => ({
  TabForm: ({ form, activeTab, errorMessage, formHeading }: any) => {
    const React = require('react');
    const active = form.find((f: any) => f.code === activeTab) || form[0];
    return (
      <div>
        <h1>{formHeading}</h1>
        {errorMessage ? <div>{errorMessage}</div> : null}
        <div>{active.body}</div>
        <div>
          {Array.isArray(active.actions)
            ? active.actions.map((action: React.ReactElement, idx: number) => (
                <div key={idx}>{action}</div>
              ))
            : active.actions}
        </div>
      </div>
    );
  },
}));

jest.mock('@/app/components/carbon/button', () => ({
  TertiaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  DangerTertiaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({ children, renderIcon: _renderIcon, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, renderIcon: _renderIcon, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  TertiaryButton: ({ children, renderIcon: _renderIcon, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({
    id,
    labelText,
    value,
    onChange,
    type = 'text',
    helperText: _h,
    hideLabel: _hl,
    warn: _w,
    warnText: _wt,
    invalid: _inv,
    invalidText: _it,
    ...rest
  }: any) => (
    <div>
      {labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <input
        id={id}
        value={value ?? ''}
        onChange={onChange}
        type={type}
        {...rest}
      />
    </div>
  ),
  TextArea: ({
    id,
    labelText,
    value,
    onChange,
    helperText: _h,
    hideLabel: _hl,
    warn: _w,
    warnText: _wt,
    invalid: _inv,
    invalidText: _it,
    ...rest
  }: any) => (
    <div>
      {labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <textarea id={id} value={value ?? ''} onChange={onChange} {...rest} />
    </div>
  ),
}));

jest.mock('@carbon/react', () => ({
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  Select: ({ id, value, onChange, children, labelText, hideLabel }: any) => (
    <div>
      {!hideLabel && labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <select id={id} value={value} onChange={onChange}>
        {children}
      </select>
    </div>
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  Tooltip: ({ children }: any) => <span>{children}</span>,
  Button: ({
    children,
    iconDescription,
    hasIconOnly: _,
    renderIcon: _r,
    ...props
  }: any) => (
    <button aria-label={iconDescription} {...props}>
      {children}
    </button>
  ),
  NumberInput: ({ id, value, onChange, label, hideLabel }: any) => (
    <div>
      {!hideLabel && label ? <label htmlFor={id}>{label}</label> : null}
      <input
        id={id}
        type="number"
        value={value ?? ''}
        onChange={e => onChange?.(e, { value: e.target.value })}
      />
    </div>
  ),
}));

jest.mock('@/app/components/form/fieldset', () => ({
  FieldSet: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/form-label', () => ({
  FormLabel: ({ children }: any) => <label>{children}</label>,
}));

jest.mock('@/app/components/form/input', () => ({
  Input: ({ ...props }: any) => <input {...props} />,
}));

jest.mock('@/app/components/form/select', () => ({
  Select: ({ options = [], value, onChange }: any) => (
    <select value={value} onChange={onChange}>
      {options.map((o: any) => (
        <option key={o.value} value={o.value}>
          {o.name}
        </option>
      ))}
    </select>
  ),
}));

jest.mock('@/app/components/form/textarea', () => ({
  Textarea: ({ ...props }: any) => <textarea {...props} />,
}));

jest.mock('@/app/components/input-helper', () => ({
  InputHelper: ({ children }: any) => <span>{children}</span>,
}));

jest.mock('@/app/components/blocks/section-divider', () => ({
  SectionDivider: ({ label }: any) => <h3>{label}</h3>,
}));

jest.mock('lucide-react', () => ({
  ArrowRight: () => <span>arrow-right</span>,
  Plus: () => <span>plus</span>,
  Trash2: () => <span>trash</span>,
}));

describe('UpdateAssistantAnalysis', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockParams = { analysisId: 'analysis-1' };
    (GetAssistantConfiguration as jest.Mock).mockImplementation(() =>
      Promise.resolve({
        getData: () => ({
          getOptionsList: () => [
            {
              getKey: () => 'name',
              getValue: () => 'loaded-analysis',
            },
            {
              getKey: () => 'description',
              getValue: () => 'loaded-description',
            },
            {
              getKey: () => 'execution_priority',
              getValue: () => '3',
            },
            {
              getKey: () => 'endpoint_id',
              getValue: () => 'endpoint-1',
            },
            {
              getKey: () => 'endpoint_version',
              getValue: () => 'latest',
            },
            {
              getKey: () => 'endpoint_parameters',
              getValue: () =>
                JSON.stringify({
                  'conversation.messages': 'messages',
                }),
            },
            {
              getKey: () => 'analysis.condition',
              getValue: () =>
                JSON.stringify([
                  { key: 'source', condition: '=', value: 'all' },
                ]),
            },
          ],
        }),
      }),
    );
  });

  it('loads analysis on mount and pre-fills fields', async () => {
    render(<UpdateAssistantAnalysis assistantId="assistant-1" />);

    await waitFor(() => {
      expect(GetAssistantConfiguration).toHaveBeenCalled();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    expect(screen.getByDisplayValue('loaded-analysis')).toBeInTheDocument();
  });

  it('shows validation error when configure tab has no parameters', async () => {
    render(<UpdateAssistantAnalysis assistantId="assistant-1" />);

    await waitFor(() => {
      expect(GetAssistantConfiguration).toHaveBeenCalled();
    });

    const removeButtons = screen.getAllByRole('button', { name: 'Remove' });
    fireEvent.click(removeButtons[removeButtons.length - 1]);
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      screen.getByText('Please provide one or more parameters.'),
    ).toBeInTheDocument();
  });

  it('updates analysis successfully and navigates back to analysis listing', async () => {
    (UpdateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });

    render(<UpdateAssistantAnalysis assistantId="assistant-1" />);

    await waitFor(() => {
      expect(GetAssistantConfiguration).toHaveBeenCalled();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update analysis' }));

    await waitFor(() => {
      expect(UpdateAssistantConfiguration).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(toast.success).toHaveBeenCalledWith(
        `Assistant's analysis updated successfully`,
      );
    });
    expect(mockGoToConfigureAssistantAnalysis).toHaveBeenCalledWith(
      'assistant-1',
    );
  });

  it('shows human error message when update response is unsuccessful', async () => {
    (UpdateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => false,
      getError: () => ({ getHumanmessage: () => 'Invalid analysis name' }),
    });

    render(<UpdateAssistantAnalysis assistantId="assistant-1" />);

    await waitFor(() => {
      expect(GetAssistantConfiguration).toHaveBeenCalled();
    });

    fireEvent.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update analysis' }));

    expect(
      await screen.findByText('Invalid analysis name'),
    ).toBeInTheDocument();
  });

  it('supports add and edit for parameter mapping before update', async () => {
    (UpdateAssistantConfiguration as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
    });

    render(<UpdateAssistantAnalysis assistantId="assistant-1" />);

    await waitFor(() => expect(GetAssistantConfiguration).toHaveBeenCalled());

    fireEvent.change(
      document.getElementById('tool-condition-key') as HTMLElement,
      {
        target: { value: 'conversation_mode' },
      },
    );
    fireEvent.change(
      document.getElementById('tool-condition-source-value') as HTMLElement,
      {
        target: { value: 'text' },
      },
    );
    fireEvent.change(document.getElementById('param-key-0') as HTMLElement, {
      target: { value: 'id' },
    });
    fireEvent.change(document.getElementById('param-val-0') as HTMLElement, {
      target: { value: 'conversationId' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Add parameter' }));
    fireEvent.change(document.getElementById('param-type-1') as HTMLElement, {
      target: { value: 'assistant' },
    });
    fireEvent.change(document.getElementById('param-key-1') as HTMLElement, {
      target: { value: 'name' },
    });
    fireEvent.change(document.getElementById('param-val-1') as HTMLElement, {
      target: { value: 'assistantName' },
    });

    fireEvent.click(screen.getByRole('button', { name: 'Pick endpoint' }));
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));
    fireEvent.click(screen.getByRole('button', { name: 'Update analysis' }));

    await waitFor(() =>
      expect(UpdateAssistantConfiguration).toHaveBeenCalled(),
    );
    const request = (UpdateAssistantConfiguration as jest.Mock).mock
      .calls[0][1];
    expect(request.getAssistantid()).toBe('assistant-1');
    expect(request.getId()).toBe('analysis-1');
    expect(request.getConfigurationtype()).toBe('analysis');
    expect(request.getProvider()).toBe('endpoint');
    expect(request.getEnabled()).toBe(true);
    const mappedParams = request.getOptionsList().map((option: any) => ({
      key: option.getKey(),
      value: option.getValue(),
    }));
    expect(mappedParams).toEqual(
      expect.arrayContaining([
        { key: 'name', value: 'loaded-analysis' },
        { key: 'description', value: 'loaded-description' },
        { key: 'execution_priority', value: '3' },
        { key: 'endpoint_id', value: 'endpoint-2' },
        { key: 'endpoint_version', value: 'latest' },
        {
          key: 'endpoint_parameters',
          value: JSON.stringify({
            'conversation.id': 'conversationId',
            'assistant.name': 'assistantName',
          }),
        },
        {
          key: 'analysis.condition',
          value: JSON.stringify([
            {
              key: 'conversation_mode',
              condition: '=',
              value: 'text',
            },
          ]),
        },
      ]),
    );
  });

  it('blocks reserved analysis option mapping key', async () => {
    render(<UpdateAssistantAnalysis assistantId="assistant-1" />);

    await waitFor(() => expect(GetAssistantConfiguration).toHaveBeenCalled());

    fireEvent.change(document.getElementById('param-type-0') as HTMLElement, {
      target: { value: 'option' },
    });
    fireEvent.change(document.getElementById('param-key-0') as HTMLElement, {
      target: { value: 'endpoint_parameters' },
    });
    fireEvent.change(document.getElementById('param-val-0') as HTMLElement, {
      target: { value: 'shouldFail' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Continue' }));

    expect(
      screen.getByText(
        'option.endpoint_parameters is reserved and managed by analysis options.',
      ),
    ).toBeInTheDocument();
    expect(UpdateAssistantConfiguration).not.toHaveBeenCalled();
  });
});
