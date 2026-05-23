import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { HomePage } from '@/app/pages/main/web-dashboard';

const mockGoToCreateAssistant = jest.fn();

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({ user: { id: 'user-1', name: 'Prashant' } }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateAssistant: mockGoToCreateAssistant,
  }),
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    children,
    href,
    onClick,
    className,
    renderIcon,
    ...props
  }: any) =>
    href ? (
      <a href={href} className={className} {...props}>
        {children}
      </a>
    ) : (
      <button onClick={onClick} className={className} {...props}>
        {children}
      </button>
    ),
  Dropdown: ({ label }: any) => <button>{label}</button>,
  Link: ({ children, href, className }: any) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
  Tile: ({ children, className }: any) => (
    <section className={className}>{children}</section>
  ),
}));

describe('Dashboard HomePage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders the welcome dashboard structure from the reference design', () => {
    render(<HomePage />);

    expect(
      screen.getByRole('heading', { name: 'Welcome, Prashant!' }),
    ).toBeInTheDocument();
    expect(
      screen.getByText('Product Highlight: Voice AI Assistants'),
    ).toBeInTheDocument();
    expect(screen.getByText('Explore Features')).toBeInTheDocument();
    expect(screen.getByText('AI Assistants')).toBeInTheDocument();
    expect(screen.getByText('Deployments')).toBeInTheDocument();
    expect(screen.getByText('Connect your provider')).toBeInTheDocument();
    expect(screen.getByText('Help and Resources')).toBeInTheDocument();
    expect(screen.getByText('Documentation')).toBeInTheDocument();
    expect(screen.getByText('GitHub repository')).toBeInTheDocument();
    expect(screen.getByText('Pricing')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Talk to us' })).toHaveAttribute(
      'href',
      'https://cal.com/prashant-srivastav-u8duzh/30min',
    );
    expect(screen.getByText("What's new")).toBeInTheDocument();
    expect(screen.getAllByText(/Read more/i)).toHaveLength(3);
  });

  it('keeps the banner create action wired to assistant creation', () => {
    render(<HomePage />);

    fireEvent.click(
      screen.getByRole('button', { name: 'Create first voice agent' }),
    );

    expect(mockGoToCreateAssistant).toHaveBeenCalledTimes(1);
  });
});
