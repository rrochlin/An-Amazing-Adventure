// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';
import { EndingPanel } from '@/components/dialogue/EndingPanel';

describe('EndingPanel', () => {
   it('says the adventure has ended, with its title and summary', () => {
      render(
         <ThemeProvider theme={AppTheme}>
            <EndingPanel ending={{ id: 'road_to_guild', title: 'The Road to the Guild', summary: 'Briarwell is ash behind you.' }} />
         </ThemeProvider>,
      );
      expect(screen.getByText('The adventure has ended')).toBeInTheDocument();
      expect(screen.getByText('The Road to the Guild')).toBeInTheDocument();
      expect(screen.getByText('Briarwell is ash behind you.')).toBeInTheDocument();
   });

   it('starts a new run', async () => {
      const onStartNewRun = vi.fn();
      render(
         <ThemeProvider theme={AppTheme}>
            <EndingPanel ending={{ title: 'Failure' }} onStartNewRun={onStartNewRun} />
         </ThemeProvider>,
      );
      await userEvent.click(screen.getByRole('button', { name: /begin a new adventure/i }));
      expect(onStartNewRun).toHaveBeenCalledTimes(1);
   });
});
