// @vitest-environment jsdom
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';
import { WorldGenTerminal } from '@/components/WorldGenTerminal';

function renderTerminal(lines: string[], ready: boolean) {
   return render(
      <ThemeProvider theme={AppTheme}>
         <WorldGenTerminal lines={lines} ready={ready} />
      </ThemeProvider>,
   );
}

describe('WorldGenTerminal', () => {
   it('renders waiting message when no lines', () => {
      renderTerminal([], false);
      expect(screen.getByText(/waiting for architect/i)).toBeInTheDocument();
   });

   it('renders log lines', () => {
      renderTerminal(
         ['Summoning the Architect...', 'Blueprint ready: 8 rooms'],
         false,
      );
      expect(screen.getByText(/Summoning the Architect/)).toBeInTheDocument();
      expect(screen.getByText(/Blueprint ready/)).toBeInTheDocument();
   });

   it('shows GENERATING in title bar when not ready', () => {
      renderTerminal([], false);
      expect(screen.getByText(/GENERATING/)).toBeInTheDocument();
   });

   it('shows COMPLETE in title bar when ready', () => {
      renderTerminal(['Your adventure awaits.'], true);
      expect(screen.getByText(/COMPLETE/)).toBeInTheDocument();
   });

   it('does not show progress bar when ready', () => {
      const { container } = renderTerminal(['done'], true);
      // LinearProgress renders a role="progressbar"
      expect(container.querySelector('[role="progressbar"]')).toBeNull();
   });

   it('shows progress bar when not ready', () => {
      const { container } = renderTerminal([], false);
      expect(container.querySelector('[role="progressbar"]')).not.toBeNull();
   });
});
