import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { ConfigureAPIRequest } from '../index';

const createMetadata = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

const toMap = (items: Metadata[]) =>
  Object.fromEntries(items.map(item => [item.getKey(), item.getValue()]));

jest.mock('@/app/components/input-group', () => ({
  InputGroup: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/carbon/form', () => ({
  Stack: ({ children }: any) => <div>{children}</div>,
  TextInput: ({ id, value, onChange, labelText, hideLabel }: any) => (
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
  TertiaryButton: ({ children, ...props }: any) => (
    <button {...props}>{children}</button>
  ),
}));

jest.mock('@carbon/react', () => ({
  Select: ({ id, value, onChange, children, hideLabel, labelText }: any) => (
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

jest.mock('@/app/components/external-api/api-header', () => ({
  APiStringHeader: () => <div>headers</div>,
}));

jest.mock(
  '@/app/components/container/message/notice-block/doc-notice-block',
  () => ({
    DocNoticeBlock: ({ children }: any) => <div>{children}</div>,
  }),
);

jest.mock('@/app/components/form/editor/code-editor', () => ({
  CodeEditor: ({ value, onChange }: any) => (
    <textarea value={value ?? ''} onChange={e => onChange(e.target.value)} />
  ),
}));

describe('ConfigureAPIRequest', () => {
  it('supports add and edit mapping parameters', () => {
    const onParameterChange = jest.fn();

    render(
      <ConfigureAPIRequest
        parameters={[
          createMetadata('tool.method', 'POST'),
          createMetadata('tool.endpoint', 'https://api.example.com'),
          createMetadata('tool.headers', '{}'),
          createMetadata('tool.parameters', '{}'),
        ]}
        onParameterChange={onParameterChange}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Add parameter' }));
    fireEvent.change(screen.getByTestId('param-type-0'), {
      target: { value: 'tool' },
    });
    fireEvent.change(screen.getByTestId('param-key-0'), {
      target: { value: 'name' },
    });
    fireEvent.change(screen.getByTestId('param-val-0'), {
      target: { value: 'customer_name' },
    });

    const lastCallArgs = onParameterChange.mock.calls.at(-1)?.[0] as Metadata[];
    const byKey = toMap(lastCallArgs);
    expect(byKey['tool.parameters']).toBe('{"tool.name":"customer_name"}');
  });
});
