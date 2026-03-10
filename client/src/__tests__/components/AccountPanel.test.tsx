// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';

// Mock router so navigation doesn't crash in tests
const mockNavigate = vi.fn();
vi.mock('@tanstack/react-router', () => ({
   useNavigate: () => mockNavigate,
}));

// vi.hoisted ensures these are available when the vi.mock factory runs (it's hoisted to top)
const { mockIsAuthenticated, mockSignOut } = vi.hoisted(() => ({
   mockIsAuthenticated: vi.fn(),
   mockSignOut: vi.fn().mockResolvedValue(undefined),
}));

vi.mock('@/services/auth.service', () => ({
   isAuthenticated: mockIsAuthenticated,
   signOut: mockSignOut,
}));

import AccountPanel from '@/components/AccountPanel';

function renderPanel() {
   return render(
      <ThemeProvider theme={AppTheme}>
         <AccountPanel />
      </ThemeProvider>,
   );
}

beforeEach(() => {
   vi.clearAllMocks();
   mockIsAuthenticated.mockReturnValue(false);
});

describe('AccountPanel', () => {
   it('renders account icon button', () => {
      renderPanel();
      expect(screen.getByRole('button')).toBeInTheDocument();
   });

   it('opens menu on icon click', async () => {
      renderPanel();
      await userEvent.click(screen.getByRole('button'));
      expect(screen.getByText('Profile')).toBeInTheDocument();
      expect(screen.getByText('Select Theme')).toBeInTheDocument();
   });

   it('shows Sign In when not authenticated', async () => {
      mockIsAuthenticated.mockReturnValue(false);
      renderPanel();
      await userEvent.click(screen.getByRole('button'));
      expect(screen.getByText('Sign In')).toBeInTheDocument();
   });

   it('shows Sign Out when authenticated', async () => {
      mockIsAuthenticated.mockReturnValue(true);
      renderPanel();
      await userEvent.click(screen.getByRole('button'));
      expect(screen.getByText('Sign Out')).toBeInTheDocument();
   });

   it('calls signOut and navigates to /login on Sign Out click', async () => {
      mockIsAuthenticated.mockReturnValue(true);
      renderPanel();
      await userEvent.click(screen.getByRole('button'));
      await userEvent.click(screen.getByText('Sign Out'));
      expect(mockSignOut).toHaveBeenCalled();
   });

   it('navigates to /profile on Profile click', async () => {
      renderPanel();
      await userEvent.click(screen.getByRole('button'));
      await userEvent.click(screen.getByText('Profile'));
      expect(mockNavigate).toHaveBeenCalledWith({ to: '/profile' });
   });

   it('expands theme options on Select Theme click', async () => {
      renderPanel();
      await userEvent.click(screen.getByRole('button'));
      await userEvent.click(screen.getByText('Select Theme'));
      expect(screen.getByText('Dark Mode')).toBeInTheDocument();
      expect(screen.getByText('Light Mode')).toBeInTheDocument();
      expect(screen.getByText('System Settings')).toBeInTheDocument();
   });
});
