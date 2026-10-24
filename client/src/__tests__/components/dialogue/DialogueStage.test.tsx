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

   it('ignores a click on a choice that has only just appeared', () => {
      // Playtest 7: clicking through the last line picked the first gift.
      const onChoiceSelected = vi.fn();
      const { update } = setup(inProgress);
      update({
         messages: [...inProgress, { type: 'narrative', content: 'Old Garrick: Your coin buys any one of these.' }],
         pendingChoices: [{ id: 0, text: 'Honey sweets.' }],
         onChoiceSelected,
      });
      tick(5000);
      const choice = screen.getByRole('button', { name: /honey sweets/i });
      // Shown a moment ago (fake time moves only with tick).
      act(() => vi.setSystemTime(Date.now() - 4000));
      fireEvent.click(choice);
      expect(onChoiceSelected).not.toHaveBeenCalled();
      act(() => vi.setSystemTime(Date.now() + 5000));
      fireEvent.click(choice);
      expect(onChoiceSelected).toHaveBeenCalledWith(0);
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

   it('puts what the player just said in the dialogue box, after everything before it', () => {
      const { update } = setup(inProgress);
      update({ messages: [...inProgress, { type: 'player', content: 'I open the door' }] });
      expect(screen.getByTestId('stage-player-line')).toHaveTextContent('YouI open the door');
      const transcript = screen.getByTestId('stage-transcript');
      // The line that was in the box moves up into the scrollback.
      expect(transcript).toHaveTextContent('Sunlight spills through the shutters.');
      expect(transcript).not.toHaveTextContent('I open the door');
   });

   it('keeps the whole story in the scrollback, with no separate History', () => {
      const long: ChatMessage[] = Array.from({ length: 60 }, (_, i) => ({
         type: i % 2 ? 'narrative' : 'player',
         content: `Line ${i}`,
      }));
      setup(long);
      const transcript = screen.getByTestId('stage-transcript');
      expect(transcript).toHaveTextContent('You: Line 0');
      expect(transcript).toHaveTextContent('You: Line 58');
      expect(screen.queryByRole('button', { name: 'History' })).not.toBeInTheDocument();
   });

   it('does not move on when the player is selecting text', () => {
      const { update } = setup(inProgress);
      update({
         messages: [...inProgress, { type: 'narrative', content: 'One.\nTwo.' }],
      });
      tick(5000);
      const select = vi.spyOn(window, 'getSelection').mockReturnValue({
         toString: () => 'One',
      } as Selection);
      fireEvent.click(stage());
      expect(stageText()).toBe('One.');
      select.mockRestore();
      fireEvent.click(stage());
      tick(5000);
      expect(stageText()).toBe('Two.');
   });

   it('renders the narrator\'s *emphasis* as italics', () => {
      setup([{ type: 'player', content: 'hi' }, { type: 'narrative', content: 'It is *gone*.' }]);
      const em = screen.getByTestId('stage-text').querySelector('em');
      expect(em).toHaveTextContent('gone');
      expect(stageText()).toBe('It is gone.');
   });

   it('reports catching up only once the last line has been shown', () => {
      const onCaughtUp = vi.fn();
      const { update } = setup(inProgress, { onCaughtUp });
      expect(onCaughtUp).toHaveBeenCalledTimes(1); // already-read lines: at the end at once
      update({ messages: [...inProgress, { type: 'narrative', content: 'One.\nTwo.' }], onCaughtUp });
      tick(5000); // first line typed, second not yet shown
      expect(onCaughtUp).toHaveBeenCalledTimes(1);
      fireEvent.click(stage());
      tick(5000);
      expect(onCaughtUp).toHaveBeenCalledTimes(2);
   });
});

describe('free roam and dialogue', () => {
   it('focuses the text box on Enter and gives the keys back on Esc', () => {
      setup(inProgress, { overlay: true });
      const box = screen.getByPlaceholderText('Explore, or press Enter to say what you do');
      expect(box).not.toHaveFocus();
      fireEvent.keyDown(window, { key: 'Enter' });
      expect(box).toHaveFocus();
      fireEvent.keyDown(box, { key: 'Escape' });
      expect(box).not.toHaveFocus();
   });

   it('shows an arrival at once, with nothing to dismiss and the text box ready', () => {
      const { update } = setup(inProgress, { overlay: true });
      update({
         overlay: true,
         messages: [...inProgress, { type: 'arrival', content: 'Village Square. Little Bess is here.' }],
      });
      // No typing, no "continue": the player is walking on.
      expect(stageText()).toBe('Village Square. Little Bess is here.');
      expect(screen.queryByLabelText('continue')).not.toBeInTheDocument();
      const box = screen.getByPlaceholderText('Explore, or press Enter to say what you do');
      // Free roam over the scene keeps the keys for walking: the box isn't focused.
      expect(document.activeElement).not.toBe(box);
      // Enter takes the player to the box.
      fireEvent.keyDown(window, { key: 'Enter' });
      expect(document.activeElement).toBe(box);
   });

   it('speaks to the character in a conversation, and leaves with the button or Esc', () => {
      const onLeave = vi.fn();
      setup(inProgress, { overlay: true, conversationWith: 'Tomas', onLeaveConversation: onLeave });
      const box = screen.getByPlaceholderText('Say something to Tomas…');
      expect(document.activeElement).toBe(box); // the conversation takes the keys
      fireEvent.click(screen.getByRole('button', { name: 'leave the conversation with Tomas' }));
      expect(onLeave).toHaveBeenCalledTimes(1);
      fireEvent.keyDown(window, { key: 'Escape' });
      expect(onLeave).toHaveBeenCalledTimes(2);
   });

   it('hands each staging cue to the scene once, when its line is reached', () => {
      const onCues = vi.fn();
      const staged: ChatMessage = {
         type: 'narrative',
         content: 'Bess runs to the well.\nLittle Bess: Look!',
         events: [{ type: 'cue', message: '', character: 'bess', emote: 'hop', line: 1 }],
      };
      // Already read on load: it has played; the world has her where she is.
      const { update } = setup([...inProgress, { type: 'player', content: 'hi' }, staged], { onCues });
      expect(onCues).not.toHaveBeenCalled();

      const next: ChatMessage = {
         type: 'narrative',
         content: 'Bess runs to the well.\nLittle Bess: Look!',
         events: [
            { type: 'cue', message: '', character: 'bess', mark: 'stone_well', line: 0 },
            { type: 'cue', message: '', character: 'bess', emote: 'hop', line: 1 },
         ],
      };
      const messages = [...inProgress, { type: 'player', content: 'hi' } as ChatMessage, staged, { type: 'player', content: 'again' } as ChatMessage, next];
      update({ messages, onCues });
      tick(100);
      expect(onCues).toHaveBeenCalledTimes(1);
      expect(onCues.mock.calls[0][0].map((c: { mark?: string }) => c.mark)).toEqual(['stone_well']);

      fireEvent.click(stage()); // finish the line
      fireEvent.click(stage()); // on to Bess's line
      expect(onCues).toHaveBeenCalledTimes(2);
      expect(onCues.mock.calls[1][0][0].emote).toBe('hop');
      update({ messages: [...messages], onCues });
      expect(onCues).toHaveBeenCalledTimes(2);
   });
});
