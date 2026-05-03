import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { ConfigureAssistantAuthenticationPage } from '../index';
import {
  CreateAssistantAuthentication,
  DisableAssistantAuthentication,
  GetAssistantAuthentication,
} from '@rapidaai/react';

jest.mock('@rapidaai/react', () => {
  class CreateAssistantAuthenticationRequest {
    setAssistantid(_: string) {}
    setStatus(_: string) {}
    setFailbehavior(_: string) {}
    setTimeoutms(_: string) {}
    setOptionsList(_: unknown[]) {}
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

describe('ConfigureAssistantAuthenticationPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (GetAssistantAuthentication as jest.Mock).mockResolvedValue({
      getSuccess: () => false,
    });
    (CreateAssistantAuthentication as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({}),
    });
    (DisableAssistantAuthentication as jest.Mock).mockResolvedValue({
      getSuccess: () => true,
      getData: () => ({}),
    });
  });

  it('keeps save enabled and validates on click', async () => {
    render(<ConfigureAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());

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
    render(<ConfigureAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());

    fireEvent.click(screen.getByLabelText('Enable Session Authentication'));
    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });

    fireEvent.click(screen.getByRole('button', { name: 'Add parameter' }));
    fireEvent.change(screen.getByTestId('param-val-2'), {
      target: { value: 'assistantPrompt' },
    });

    expect(screen.getByText('Mapping (3)')).toBeInTheDocument();
    expect(screen.getByTestId('param-val-2')).toHaveValue('assistantPrompt');
  });

  it('creates authentication when enabled and valid', async () => {
    render(<ConfigureAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());

    fireEvent.click(screen.getByLabelText('Enable Session Authentication'));
    fireEvent.change(screen.getByTestId('assistant-auth-endpoint'), {
      target: { value: 'https://auth.example.com/resolve' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save authentication' }));

    await waitFor(() =>
      expect(CreateAssistantAuthentication).toHaveBeenCalledTimes(1),
    );
  });

  it('disables authentication when toggled off and saved', async () => {
    render(<ConfigureAssistantAuthenticationPage />);
    await waitFor(() => expect(GetAssistantAuthentication).toHaveBeenCalled());

    fireEvent.click(screen.getByRole('button', { name: 'Save authentication' }));

    await waitFor(() =>
      expect(DisableAssistantAuthentication).toHaveBeenCalledTimes(1),
    );
  });
});
