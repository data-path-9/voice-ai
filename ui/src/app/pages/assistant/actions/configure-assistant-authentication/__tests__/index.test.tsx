import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  CreateAssistantAuthenticationPage,
  UpdateAssistantAuthenticationPage,
} from '../index';
import {
  CreateAssistantAuthentication,
  DisableAssistantAuthentication,
  GetAssistantAuthentication,
} from '@rapidaai/react';

jest.mock('@rapidaai/react', () => {
  class CreateAssistantAuthenticationRequest {
    assistantId = '';
    provider = '';
    status = '';
    failBehavior = '';
    timeoutMs = '';
    optionsList: unknown[] = [];
    setAssistantid(v: string) {
      this.assistantId = v;
    }
    setProvider(v: string) {
      this.provider = v;
    }
    setStatus(v: string) {
      this.status = v;
    }
    setFailbehavior(v: string) {
      this.failBehavior = v;
    }
    setTimeoutms(v: string) {
      this.timeoutMs = v;
    }
    setOptionsList(v: unknown[]) {
      this.optionsList = v;
    }
  }
  class GetAssistantAuthenticationRequest {
    setAssistantid(_: string) {}
  }
  class DisableAssistantAuthenticationRequest {
    setAssistantid(_: string) {}
  }
  class ConnectionConfig {
    constructor(_: unknown) {}
  }
  class Metadata {
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
  }
  return {
    ConnectionConfig,
    CreateAssistantAuthenticationRequest,
    GetAssistantAuthenticationRequest,
    DisableAssistantAuthenticationRequest,
    Metadata,
    CreateAssistantAuthentication: jest.fn(),
    DisableAssistantAuthentication: jest.fn(),
    GetAssistantAuthentication: jest.fn(),
  };
});

jest.mock('react-hot-toast/headless', () => ({
  __esModule: true,
  default: {
    success: jest.fn(),
    error: jest.fn(),
  },
}));

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ assistantId: 'assistant-1' }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'auth-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goBack: jest.fn(),
  }),
}));

jest.mock('@/app/pages/assistant/actions/hooks/use-confirmation', () => ({
  useConfirmDialog: () => ({
    showDialog: (cb: () => void) => cb(),
    ConfirmDialogComponent: () => null,
  }),
}));

jest.mock('@/app/components/input-group', () => ({
  InputGroup: ({ title, children }: any) => (
    <section>
      {title ? <div>{title}</div> : null}
      {children}
    </section>
  ),
}));

jest.mock('@/app/components/conditions/source-condition-rule', () => ({
  SourceConditionRule: () => <div>conditions</div>,
}));

jest.mock('@/app/components/external-api/api-header', () => ({
  APiStringHeader: () => <div>headers</div>,
}));

jest.mock('@/app/components/carbon/notification', () => ({
  Notification: ({ subtitle }: any) => <div>{subtitle}</div>,
}));

jest.mock('@/app/components/carbon/status-indicator', () => ({
  CarbonStatusIndicator: ({ state }: any) => <span>{state}</span>,
}));

jest.mock('@/app/components/carbon/overflow-menu', () => ({
  OverflowMenu: ({ children }: any) => <div>{children}</div>,
  OverflowMenuItem: ({ itemText, onClick, disabled }: any) => (
    <button disabled={disabled} onClick={onClick}>
      {itemText}
    </button>
  ),
}));

jest.mock('@/app/components/carbon/empty-state', () => ({
  EmptyState: ({
    title,
    subtitle,
    actionButtonText,
    onActionButtonClick,
  }: any) => (
    <div>
      <div>{title}</div>
      <div>{subtitle}</div>
      {actionButtonText ? (
        <button onClick={onActionButtonClick}>{actionButtonText}</button>
      ) : null}
    </div>
  ),
}));

jest.mock('@/app/components/loader/section-loader', () => ({
  SectionLoader: () => <div>loading</div>,
}));

jest.mock('@/app/components/sections/table-section', () => ({
  TableSection: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/carbon/form/input-checkbox', () => ({
  InputCheckbox: ({ id, checked, onChange, children }: any) => (
    <label htmlFor={id}>
      <input id={id} type="checkbox" checked={checked} onChange={onChange} />
      {children}
    </label>
  ),
}));

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({ id, labelText, value, onChange, hideLabel }: any) => (
    <div>
      {!hideLabel && labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <input id={id} data-testid={id} value={value ?? ''} onChange={onChange} />
    </div>
  ),
  TextArea: ({ id, value, onChange }: any) => (
    <textarea id={id} value={value ?? ''} onChange={onChange} />
  ),
}));

jest.mock('@/app/components/carbon/button', () => ({
  PrimaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  SecondaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
  TertiaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@carbon/react', () => ({
  Breadcrumb: ({ children }: any) => <div>{children}</div>,
  BreadcrumbItem: ({ children }: any) => <span>{children}</span>,
  ButtonSet: ({ children }: any) => <div>{children}</div>,
  CheckboxGroup: ({ children }: any) => <div>{children}</div>,
  Slider: ({ id, value, onChange }: any) => (
    <input
      id={id}
      type="range"
      value={value}
      onChange={e => onChange?.({ value: Number(e.target.value) })}
    />
  ),
  Select: ({ id, labelText, value, onChange, children, hideLabel }: any) => (
    <div>
      {!hideLabel && labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <select id={id} data-testid={id} value={value} onChange={onChange}>
        {children}
      </select>
    </div>
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  Button: ({ children, iconDescription, ...props }: any) => (
    <button aria-label={iconDescription || children || 'button'} {...props}>
      {children || 'button'}
    </button>
  ),
  Tooltip: ({ children }: any) => <span>{children}</span>,
}));

describe('CreateAssistantAuthenticationPage', () => {
  const getSuccessLoadResponse = (
    status = 'inactive',
    failBehavior = 'BLOCK',
  ) => ({
    getSuccess: () => true,
    getData: () => ({
      getStatus: () => status,
      getProvider: () => 'http',
      getFailbehavior: () => failBehavior,
      getTimeoutms: () => '5000',
      getOptionsList: () => [],
    }),
  });

  beforeEach(() => {
    jest.clearAllMocks();
    (GetAssistantAuthentication as jest.Mock).mockResolvedValue(
      getSuccessLoadResponse(),
    );
    (CreateAssistantAuthentication as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({}),
    });
    (DisableAssistantAuthentication as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({}),
    });
  });

  const waitUntilReady = async () => {
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Save authentication' }),
      ).not.toBeDisabled(),
    );
  };

  it('keeps save enabled and validates on click', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAssistantAuthentication).not.toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText('Enable Session Authentication'));

    const saveButton = screen.getByRole('button', {
      name: 'Save authentication',
    });
    expect(saveButton).not.toBeDisabled();

    fireEvent.click(saveButton);

    expect(
      screen.getByText('Please provide a server URL for authentication.'),
    ).toBeInTheDocument();
    expect(CreateAssistantAuthentication).not.toHaveBeenCalled();
  });

  it('supports add and edit for authentication parameter mapping', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAssistantAuthentication).not.toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText('Enable Session Authentication'));
    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });

    const beforeCount = screen.getAllByTestId(/param-val-/).length;
    fireEvent.click(screen.getByRole('button', { name: 'Add parameter' }));
    await waitFor(() =>
      expect(screen.getAllByTestId(/param-val-/).length).toBe(beforeCount + 1),
    );
    const fields = screen.getAllByTestId(/param-val-/);
    const lastField = fields[fields.length - 1];
    fireEvent.change(lastField, {
      target: { value: 'assistantPrompt' },
    });

    expect(screen.getByText(/Mapping \(\d+\)/)).toBeInTheDocument();
    expect(lastField).toHaveValue('assistantPrompt');
  });

  it('creates authentication when enabled and valid', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAssistantAuthentication).not.toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText('Enable Session Authentication'));
    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });
    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );

    await waitFor(() =>
      expect(CreateAssistantAuthentication).toHaveBeenCalledTimes(1),
    );
    const createRequest = (CreateAssistantAuthentication as jest.Mock).mock
      .calls[0][1] as {
      provider: string;
      status: string;
      failBehavior: string;
      optionsList: Array<{
        getKey: () => string;
        getValue: () => string;
      }>;
    };
    expect(createRequest.provider).toBe('http');
    expect(createRequest.status).toBe('ACTIVE');
    expect(createRequest.failBehavior).toBe('BLOCK');
    const optionMap = new Map(
      createRequest.optionsList.map(option => [
        option.getKey(),
        option.getValue(),
      ]),
    );
    expect(optionMap.get('http_method')).toBe('POST');
    expect(optionMap.get('http_url')).toBe('https://auth.example.com/resolve');
    expect(optionMap.get('http_headers')).toBe('{}');
    expect(optionMap.get('http_body')).toBe(
      '{"assistant.id":"assistantId","client.phone":"clientPhone"}',
    );
    expect(optionMap.get('authentication.condition')).toBeDefined();
    expect(optionMap.get('auth.provider')).toBeUndefined();
    expect([...optionMap.keys()].sort()).toEqual(
      [
        'http_method',
        'http_url',
        'http_headers',
        'http_body',
        'authentication.condition',
      ].sort(),
    );
  });

  it('sends DO_NOTHING when on error is set to do nothing', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAssistantAuthentication).not.toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText('Enable Session Authentication'));
    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });
    fireEvent.change(screen.getByTestId('assistant-auth-fail-behavior'), {
      target: { value: 'do_nothing' },
    });
    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );

    await waitFor(() =>
      expect(CreateAssistantAuthentication).toHaveBeenCalledTimes(1),
    );
    const createRequest = (CreateAssistantAuthentication as jest.Mock).mock
      .calls[0][1] as {
      failBehavior: string;
    };
    expect(createRequest.failBehavior).toBe('DO_NOTHING');
  });

  it('disables authentication when toggled off and saved', async () => {
    render(<CreateAssistantAuthenticationPage />);
    await waitUntilReady();
    expect(GetAssistantAuthentication).not.toHaveBeenCalled();

    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );

    await waitFor(() =>
      expect(DisableAssistantAuthentication).toHaveBeenCalledTimes(1),
    );
  });

  it('keeps toggle unchecked when status is inactive', async () => {
    (GetAssistantAuthentication as jest.Mock).mockResolvedValueOnce(
      getSuccessLoadResponse('inactive'),
    );

    render(<UpdateAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());
    await waitUntilReady();

    expect(
      screen.getByRole('checkbox', { name: 'Enable Session Authentication' }),
    ).not.toBeChecked();
    expect(
      screen.getByRole('button', { name: 'Save authentication' }),
    ).not.toBeDisabled();
  });

  it('maps DO_NOTHING from API to do nothing option in UI', async () => {
    (GetAssistantAuthentication as jest.Mock).mockResolvedValueOnce(
      getSuccessLoadResponse('active', 'DO_NOTHING'),
    );

    render(<UpdateAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());
    await waitUntilReady();

    expect(screen.getByTestId('assistant-auth-fail-behavior')).toHaveValue(
      'do_nothing',
    );
  });

  it('maps legacy none from API to do nothing option in UI', async () => {
    (GetAssistantAuthentication as jest.Mock).mockResolvedValueOnce(
      getSuccessLoadResponse('active', 'none'),
    );

    render(<UpdateAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());
    await waitUntilReady();

    expect(screen.getByTestId('assistant-auth-fail-behavior')).toHaveValue(
      'do_nothing',
    );
  });

  it('saves DO_NOTHING when loaded legacy none without changing selection', async () => {
    (GetAssistantAuthentication as jest.Mock).mockResolvedValueOnce(
      getSuccessLoadResponse('active', 'none'),
    );

    render(<UpdateAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());
    await waitUntilReady();

    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });
    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );

    await waitFor(() =>
      expect(CreateAssistantAuthentication).toHaveBeenCalledTimes(1),
    );
    const createRequest = (CreateAssistantAuthentication as jest.Mock).mock
      .calls[0][1] as {
      failBehavior: string;
    };
    expect(createRequest.failBehavior).toBe('DO_NOTHING');
  });

  it('falls back to add flow when initial load does not return authentication data', async () => {
    (GetAssistantAuthentication as jest.Mock).mockResolvedValueOnce({
      getSuccess: () => false,
      getError: () => ({
        getHumanmessage: () => 'Failed to load auth',
      }),
    });

    render(<UpdateAssistantAuthenticationPage />);

    await waitFor(() =>
      expect(screen.getByText('Add Authentication')).toBeInTheDocument(),
    );
    expect(screen.queryByText('Failed to load auth')).not.toBeInTheDocument();
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Save authentication' }),
      ).not.toBeDisabled(),
    );

    fireEvent.click(
      screen.getByRole('button', { name: 'Save authentication' }),
    );
    await waitFor(() =>
      expect(DisableAssistantAuthentication).toHaveBeenCalledTimes(1),
    );
    expect(CreateAssistantAuthentication).not.toHaveBeenCalled();
  });
});
