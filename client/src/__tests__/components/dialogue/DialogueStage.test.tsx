// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';
import {
   DialogueStage,
   SCENE_CARD_MS,
} from '@/components/dialogue/DialogueStage';
import type { ChatMessage, DialogueChoice } from '@/types/types';

type Props = Parameters<typeof DialogueStage>[0];

function setup(initial: ChatMessage[], props: Partial<Props> = {}) {
   const base: Props = {
      messages: initial,
      isLoading: false,
      command: '',
      setCommand: vi.fn(),
      handleCommand: vi.fn(),
      ...props,
   };
   const view = render(
      <ThemeProvider theme={AppTheme}>
         <DialogueStage {...base} />
      </ThemeProvider>,
   );
   const update = (next: Partial<Props>) =>
      view.rerender(
         <ThemeProvider theme={AppTheme}>
            <DialogueStage {...base} {...next} />
         </ThemeProvider>,
      );
   return { ...view, update };
}

const stage = () => screen.getByTestId('dialogue-stage');
const stageText = () => screen.getByTestId('stage-text').textContent ?? '';
// Advance in small steps so timers started by each re-render also run.
const tick = (ms: number) => {
   for (let t = 0; t < ms; t += 50) act(() => vi.advanceTimersByTime(50));
};
const opening: ChatMessage = { type: 'narrative', content: 'Sunlight spills through the shutters.' };
// A game in progress: the player has already acted once.
const inProgress: ChatMessage[] = [{ type: 'player', content: 'I wake up' }, opening];

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('DialogueStage', () => {
   it('shows what was already read instantly, with the text box ready', () => {
      setup(inProgress);
      expect(stageText()).toBe('Sunlight spills through the shutters.');
      expect(screen.getByPlaceholderText('What do you do?')).toBeInTheDocument();
   });

   it('types new text in, and shows one line at a time', () => {
      const { update } = setup(inProgress);
      update({
         messages: [
            ...inProgress,
            { type: 'narrative', content: 'Father sets down his mug.\nFather: It is your sister\'s birthday.' },
         ],
      });

      // The first new line starts typing; the second waits its turn.
      tick(100);
      expect(stageText().length).toBeGreaterThan(0);
      expect(stageText()).not.toBe('Father sets down his mug.');
      expect(screen.queryByText(/sister's birthday/)).not.toBeInTheDocument();

      // Click finishes the line, then a second click moves on.
      fireEvent.click(stage());
      expect(stageText()).toBe('Father sets down his mug.');
      fireEvent.click(stage());
      expect(screen.getByText('Father')).toBeInTheDocument(); // name plate
      tick(5000);
      expect(stageText()).toBe("It is your sister's birthday.");
   });

   it('holds choices until the text has been read', () => {
      const choices: DialogueChoice[] = [{ id: 0, text: 'Go to bed.' }];
      const { update } = setup(inProgress);
      update({
         messages: [...inProgress, { type: 'narrative', content: 'Father: Off to bed, the both of you.' }],
         pendingChoices: choices,
         onChoiceSelected: vi.fn(),
      });
      tick(100);
      expect(screen.queryByRole('button', { name: /go to bed/i })).not.toBeInTheDocument();
      tick(5000);
      expect(screen.getByRole('button', { name: /go to bed/i })).toBeInTheDocument();
   });

   it('opens a new scene with a title card, then its text, and toasts the objective', () => {
      const { update } = setup(inProgress);
      update({
         messages: [
            ...inProgress,
            {
               type: 'narrative',
               content: 'You wake to screaming.',
               events: [
                  { type: 'scene_started', message: 'Briarwell Burns' },
                  { type: 'campaign_progress', message: 'Objective updated: Survive the night.' },
               ],
            },
         ],
      });
      expect(screen.getByTestId('scene-card')).toHaveTextContent('Briarwell Burns');
      expect(screen.getByRole('alert')).toHaveTextContent('New objective: Survive the night.');

      tick(SCENE_CARD_MS + 5000);
      expect(screen.queryByTestId('scene-card')).not.toBeInTheDocument();
      expect(stageText()).toBe('You wake to screaming.');
   });

   it('waits for the player before handing back the text box', () => {
      const { update } = setup(inProgress);
      update({ messages: [...inProgress, { type: 'narrative', content: 'The square is quiet.' }] });
      tick(5000);
      expect(screen.queryByPlaceholderText('What do you do?')).not.toBeInTheDocument();
      expect(screen.getByLabelText('continue')).toBeInTheDocument();
      fireEvent.click(stage());
      expect(screen.getByPlaceholderText('What do you do?')).toBeInTheDocument();
   });

   it('hides the text box and shows progress while the story answers', () => {
      const handleCommand = vi.fn();
      setup(inProgress, { command: 'I look for Bess', handleCommand });
      fireEvent.submit(screen.getByPlaceholderText('What do you do?').closest('form')!);
      expect(handleCommand).toHaveBeenCalled();
      expect(screen.queryByPlaceholderText('What do you do?')).not.toBeInTheDocument();
      expect(screen.getByLabelText('the story continues')).toBeInTheDocument();
   });

   it('shows the ending once the last line has been read', () => {
      const { update } = setup(inProgress);
      update({
         messages: [...inProgress, { type: 'narrative', content: 'Dawn finds you on the road.' }],
         ending: { title: 'The Road to the Guild', summary: 'Briarwell is ash.' },
      });
      expect(screen.queryByText('The Road to the Guild')).not.toBeInTheDocument();
      tick(5000);
      expect(screen.getByText('The Road to the Guild')).toBeInTheDocument();
   });

   it('plays a brand-new game\'s opening from the first beat', () => {
      setup([
         {
            type: 'narrative',
            content: '',
            events: [{ type: 'scene_started', message: 'A Birthday Errand' }],
         },
         opening,
      ]);
      expect(screen.getByTestId('scene-card')).toHaveTextContent('A Birthday Errand');
      tick(SCENE_CARD_MS + 5000);
      expect(stageText()).toBe('Sunlight spills through the shutters.');
   });

   it('shows where the player is and what they are doing', () => {
      setup(inProgress, {
         location: 'Village Square',
         sceneTitle: 'Good Turns and Bad',
         objective: 'Earn two coins.',
      });
      const header = screen.getByTestId('stage-header');
      expect(header).toHaveTextContent('Village Square');
      expect(header).toHaveTextContent('Good Turns and Bad');
      expect(header).toHaveTextContent('Objective: Earn two coins.');
   });

   it('keeps the previous lines and the player\'s words on screen', () => {
      const { update } = setup(inProgress);
      update({
         messages: [
            ...inProgress,
            { type: 'narrative', content: 'Father: Off you go.\nThe door creaks shut.' },
         ],
      });
      tick(5000);
      fireEvent.click(stage()); // on to the second new line
      tick(5000);
      const transcript = screen.getByTestId('stage-transcript');
      expect(transcript).toHaveTextContent('You: I wake up');
      expect(transcript).toHaveTextContent('Sunlight spills through the shutters.');
      expect(transcript).toHaveTextContent('Father:Off you go.');
      expect(transcript).not.toHaveTextContent('The door creaks shut.'); // that's the current line
      expect(stageText()).toBe('The door creaks shut.');
   });
});
