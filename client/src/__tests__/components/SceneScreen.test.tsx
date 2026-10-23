// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';
import { SceneScreen } from '@/components/SceneScreen';
import { BRIARWELL_ART } from '@/components/map/art/briarwell';
import type { ObjectiveStateView, RoomView } from '@/types/types';

vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue(null);

const rooms = {
   village_square: { id: 'village_square', name: 'Village Square', discovered: true, occupants: [], points_of_interest: [] },
} as unknown as Record<string, RoomView>;
const quests: ObjectiveStateView[] = [
   { id: 'hear_father', status: 'completed', visible_text: 'Listen to Father' },
   { id: 'earn_coins', status: 'active', visible_text: 'Earn Some Coin' },
   { id: 'return_doll', status: 'failed', visible_text: "Bess's Lost Doll" },
];

function setup(objectives = quests) {
   const renderStage = vi.fn((logOpen: boolean) => <div data-testid="stage">{logOpen ? 'log open' : 'log closed'}</div>);
   const view = render(
      <ThemeProvider theme={AppTheme}>
         <SceneScreen
            art={BRIARWELL_ART}
            rooms={rooms}
            currentRoomId="village_square"
            location="Village Square"
            sceneTitle="Good Turns and Bad"
            objective="Earn two coins"
            journalTitle="The Burning of Briarwell"
            objectives={objectives}
            renderStage={renderStage}
         />
      </ThemeProvider>,
   );
   return { ...view, renderStage };
}

describe('SceneScreen', () => {
   it('shows the scene, where you are, and the story over it', () => {
      setup();
      expect(screen.getByTestId('scene-chip')).toHaveTextContent('Village Square· Good Turns and BadEarn two coins');
      expect(screen.getByTestId('stage')).toHaveTextContent('log closed');
      expect(screen.getByTestId('scene-window')).toBeInTheDocument();
   });

   it('opens the log over the scene', () => {
      setup();
      fireEvent.click(screen.getByRole('button', { name: 'Log' }));
      expect(screen.getByTestId('stage')).toHaveTextContent('log open');
   });

   it('keeps the quests in the journal, active first, without internal ids', () => {
      setup();
      fireEvent.click(screen.getByRole('button', { name: 'Journal' }));
      const journal = screen.getByRole('dialog', { name: 'Journal' });
      expect(journal).toHaveTextContent('The Burning of Briarwell');
      const quests = screen.getAllByRole('listitem').map((li) => li.getAttribute('aria-label'));
      expect(quests).toEqual(['Earn Some Coin (active)', 'Listen to Father (completed)', "Bess's Lost Doll (failed)"]);
      expect(journal).not.toHaveTextContent('village_deeds');
      fireEvent.keyDown(window, { key: 'Escape' });
      expect(screen.queryByRole('dialog', { name: 'Journal' })).not.toBeInTheDocument();
      // The button toggles it too.
      fireEvent.click(screen.getByRole('button', { name: 'Journal' }));
      expect(screen.getByRole('dialog', { name: 'Journal' })).toBeInTheDocument();
      fireEvent.click(screen.getByRole('button', { name: 'Journal' }));
      expect(screen.queryByRole('dialog', { name: 'Journal' })).not.toBeInTheDocument();
   });

   it('opens the adventure details from the button bar', () => {
      const onDetails = vi.fn();
      render(
         <ThemeProvider theme={AppTheme}>
            <SceneScreen art={BRIARWELL_ART} rooms={rooms} currentRoomId="village_square" objectives={quests}
               renderStage={() => null} onDetails={onDetails} />
         </ThemeProvider>,
      );
      fireEvent.click(screen.getByRole('button', { name: 'Adventure details' }));
      expect(onDetails).toHaveBeenCalledTimes(1);
   });

   it('marks the journal when quests change while it is closed', () => {
      const { rerender, renderStage } = setup();
      const dot = () => screen.getByRole('button', { name: 'Journal' }).querySelector('.MuiBadge-dot');
      expect(dot()).toHaveClass('MuiBadge-invisible');
      rerender(
         <ThemeProvider theme={AppTheme}>
            <SceneScreen art={BRIARWELL_ART} rooms={rooms} currentRoomId="village_square"
               objectives={[...quests.slice(0, 1), { ...quests[1], status: 'completed' }, quests[2]]} renderStage={renderStage} />
         </ThemeProvider>,
      );
      expect(dot()).not.toHaveClass('MuiBadge-invisible');
   });
});
