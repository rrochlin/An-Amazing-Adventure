// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';

vi.mock('@tanstack/react-router', () => ({
   useNavigate: () => vi.fn(),
   createLink: (C: React.ComponentType) => C,
   Link: ({ children, ...props }: React.PropsWithChildren<object>) => (
      <a {...(props as React.AnchorHTMLAttributes<HTMLAnchorElement>)}>
         {children}
      </a>
   ),
}));

vi.mock('@/services/auth.service', () => ({
   isAuthenticated: () => false,
   signOut: vi.fn(),
}));

// Mock AccountPanel and CustomLink to avoid router dependency depth
vi.mock('@/components/AccountPanel', () => ({
   default: () => <button>account</button>,
}));

vi.mock('@/components/CustomLink', () => ({
   CustomLink: ({ children }: React.PropsWithChildren) => <a>{children}</a>,
}));

import React from 'react';
import { Header } from '@/components/Header';

function renderHeader() {
   return render(
      <ThemeProvider theme={AppTheme}>
         <Header />
      </ThemeProvider>,
   );
}

describe('Header', () => {
   it('renders the Home link', () => {
      renderHeader();
      expect(screen.getByText('Home')).toBeInTheDocument();
   });

   it('renders the account panel', () => {
      renderHeader();
      expect(screen.getByText('account')).toBeInTheDocument();
   });

   it('renders as an AppBar (nav landmark or header element)', () => {
      renderHeader();
      expect(screen.getByRole('banner')).toBeInTheDocument();
   });
});
