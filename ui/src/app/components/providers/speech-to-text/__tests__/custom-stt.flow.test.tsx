import React from 'react';
import { Metadata } from '@rapidaai/react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { SpeechToTextProvider } from '..';
import {
  GetDefaultSpeechToTextIfInvalid,
  ValidateSpeechToTextIfInvalid,
} from '../provider';

jest.mock('@/app/components/json-editor', () => ({
  JsonEditor: ({ value, onChange, placeholder }: any) => (
    <textarea
      value={value ?? ''}
      placeholder={placeholder}
      onChange={e => onChange?.(e.target.value)}
    />
  ),
}));

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({ id, labelText, value, onChange, placeholder, type }: any) => (
    <div>
      {labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <input
        id={id}
        type={type || 'text'}
        value={value ?? ''}
        placeholder={placeholder}
        onChange={onChange}
      />
    </div>
  ),
  TextArea: ({ id, labelText, value, onChange, placeholder }: any) => (
    <div>
      {labelText ? <label htmlFor={id}>{labelText}</label> : null}
      <textarea
        id={id}
        value={value ?? ''}
        placeholder={placeholder}
        onChange={onChange}
      />
    </div>
  ),
}));

jest.mock('@/app/components/dropdown/credential-dropdown', () => ({
  CredentialDropdown: ({ onChangeCredential, provider }: any) => (
    <button
      type="button"
      onClick={() => onChangeCredential({ getId: () => 'cred-custom-stt-1' })}
    >
      Pick {provider} credential
    </button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  SettingsAdjust: () => null,
  Add: () => null,
  TrashCan: () => null,
}));

jest.mock('@carbon/react', () => {
  const React = require('react');
  const getValue = (item: any) =>
    item?.code ?? item?.value ?? item?.id ?? item?.name ?? '';

  return {
    Dropdown: ({
      id,
      titleText,
      label,
      items = [],
      selectedItem,
      itemToString,
      onChange,
    }: any) => (
      <div>
        {titleText ? <label htmlFor={id}>{titleText}</label> : null}
        <select
          id={id}
          aria-label={label || titleText || 'dropdown'}
          value={String(getValue(selectedItem))}
          onChange={e => {
            const selected = items.find(
              (item: any) => String(getValue(item)) === e.target.value,
            );
            onChange?.({ selectedItem: selected || null });
          }}
        >
          <option value="">Select</option>
          {items.map((item: any) => (
            <option key={String(getValue(item))} value={String(getValue(item))}>
              {itemToString
                ? itemToString(item)
                : item?.name || String(getValue(item))}
            </option>
          ))}
        </select>
      </div>
    ),
    Select: ({ id, labelText, value, onChange, children }: any) => (
      <div>
        {labelText ? <label htmlFor={id}>{labelText}</label> : null}
        <select
          id={id}
          aria-label={labelText || 'select'}
          value={value ?? ''}
          onChange={onChange}
        >
          {children}
        </select>
      </div>
    ),
    SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
    ComboBox: ({ id, titleText, placeholder }: any) => (
      <div>
        {titleText ? <label htmlFor={id}>{titleText}</label> : null}
        <input id={id} placeholder={placeholder} />
      </div>
    ),
    NumberInput: ({ id, value, onChange }: any) => (
      <input
        id={id}
        type="number"
        value={value ?? ''}
        onChange={e => onChange?.(e, { value: e.target.value })}
      />
    ),
    Slider: ({ id, labelText, value, onChange }: any) => (
      <div>
        {labelText ? <label htmlFor={id}>{labelText}</label> : null}
        <input
          id={id}
          type="range"
          value={value ?? 0}
          onChange={e => onChange?.({ value: Number(e.target.value) })}
        />
      </div>
    ),
    Button: ({ children, ...props }: any) => (
      <button {...props}>{children}</button>
    ),
    ButtonSet: ({ children }: any) => <div>{children}</div>,
    ComposedModal: ({ children, open }: any) =>
      open ? <div>{children}</div> : null,
    ModalHeader: ({ title }: any) => <div>{title}</div>,
    ModalBody: ({ children }: any) => <div>{children}</div>,
    ModalFooter: ({ children }: any) => <div>{children}</div>,
  };
});

function findMeta(source: Metadata[], key: string): string | undefined {
  return source.find(item => item.getKey() === key)?.getValue();
}

describe('custom-stt speech-to-text flow', () => {
  it('lets a user select custom-stt, pick a credential, fill required fields, and pass validation', async () => {
    let latestProvider = '';
    let latestParameters: Metadata[] = [];

    const Harness = () => {
      const [provider, setProvider] = React.useState('');
      const [parameters, setParameters] = React.useState<Metadata[]>([]);

      React.useEffect(() => {
        latestProvider = provider;
        latestParameters = parameters;
      }, [provider, parameters]);

      return (
        <SpeechToTextProvider
          provider={provider}
          parameters={parameters}
          onChangeProvider={nextProvider => {
            setProvider(nextProvider);
            setParameters(GetDefaultSpeechToTextIfInvalid(nextProvider, []));
          }}
          onChangeParameter={setParameters}
        />
      );
    };

    render(<Harness />);

    fireEvent.change(screen.getByLabelText('Select voice input provider'), {
      target: { value: 'custom-stt' },
    });

    expect(screen.getByText('Model')).toBeInTheDocument();
    expect(screen.getByText('Language')).toBeInTheDocument();
    expect(
      (screen.getByLabelText('Audio Encoding') as HTMLSelectElement).value,
    ).toBe('LINEAR16');
    expect(
      (screen.getByLabelText('Sample Rate') as HTMLSelectElement).value,
    ).toBe('16000');
    expect(screen.getByText('Query Parameters')).toBeInTheDocument();
    expect(screen.getByText('Audio Request')).toBeInTheDocument();
    expect(screen.getByText('Response Parser')).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: 'Pick custom-stt credential' }),
    );

    fireEvent.change(
      screen.getByPlaceholderText('Type [ for parser snippets'),
      {
        target: {
          value:
            '[{"when":{"frame":"json","path":"type","equals":"partial"},"emit":{"script":{"$path":"text"},"interim":true}},{"when":{"frame":"json","path":"type","equals":"final"},"emit":{"script":{"$path":"text"},"confidence":{"$cast":"number","value":{"$path":"confidence"}},"language":{"$path":"language"},"interim":false}}]',
        },
      },
    );

    await waitFor(() => {
      expect(latestProvider).toBe('custom-stt');
      expect(findMeta(latestParameters, 'rapida.credential_id')).toBe(
        'cred-custom-stt-1',
      );
      expect(findMeta(latestParameters, 'listen.audio.encoding')).toBe(
        'LINEAR16',
      );
      expect(findMeta(latestParameters, 'listen.audio.sample_rate')).toBe(
        '16000',
      );
      expect(
        ValidateSpeechToTextIfInvalid(latestProvider, latestParameters, [
          'cred-custom-stt-1',
        ]),
      ).toBeUndefined();
    });
  });
});
