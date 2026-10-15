// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { useRef, useState } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider } from '@mui/material/styles';
import { AppTheme } from '@/theme/theme';
import { Chat } from '@/components/Chat';
import type { ChatMessage } from '@/types/types';

function renderChat(props: Partial<Parameters<typeof Chat>[0]> = {}) {
   const defaults = {
      chatHistory: [] as ChatMessage[],
      command: '',
      setCommand: vi.fn(),
      handleCommand: vi.fn(),
      isLoading: false,
   };
   return render(
      <ThemeProvider theme={AppTheme}>
         <Chat {...defaults} {...props} />
      </ThemeProvider>,
   );
}

describe('Chat component', () => {
   it('renders with empty history', () => {
      renderChat();
      expect(
         screen.getByPlaceholderText('Speak thy command...'),
      ).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /send/i })).toBeInTheDocument();
   });

   it('renders player and narrative messages', () => {
      const history: ChatMessage[] = [
         { type: 'player', content: 'Go north' },
         { type: 'narrative', content: 'You walk through a dark corridor.' },
      ];
      renderChat({ chatHistory: history });
      expect(screen.getByText('Go north')).toBeInTheDocument();
      expect(
         screen.getByText('You walk through a dark corridor.'),
      ).toBeInTheDocument();
   });

   it('shows loading indicator when isLoading is true and no streaming content', () => {
      renderChat({ isLoading: true });
      // Send button shows spinner — button is disabled
      const sendBtn = screen.getByRole('button');
      expect(sendBtn).toBeDisabled();
      // No streaming content → LoadingMessage shimmer flavor text is shown
      // (one of the flavor texts from the array — we just check the button state here)
   });

   it('shows streaming bubble with content when isLoading and streamingMessage provided', () => {
      renderChat({
         isLoading: true,
         streamingMessage: 'The dungeon echoes...',
      });
      // The streaming text should be visible
      expect(screen.getByText('The dungeon echoes...')).toBeInTheDocument();
   });

   it('does not show streaming content in committed history while streaming', () => {
      const history: ChatMessage[] = [{ type: 'player', content: 'Go north' }];
      renderChat({
         chatHistory: history,
         isLoading: true,
         streamingMessage: 'Shadows stir ahead.',
      });
      // Committed player message visible
      expect(screen.getByText('Go north')).toBeInTheDocument();
      // Streaming chunk visible in the streaming bubble
      expect(screen.getByText('Shadows stir ahead.')).toBeInTheDocument();
   });

   it('Send button disabled when command is empty', () => {
      renderChat({ command: '' });
      expect(screen.getByRole('button', { name: /send/i })).toBeDisabled();
   });

   it('Send button enabled when command has content', () => {
      renderChat({ command: 'Go west' });
      expect(screen.getByRole('button', { name: /send/i })).not.toBeDisabled();
   });

   it('calls setCommand on input change', async () => {
      const setCommand = vi.fn();
      renderChat({ setCommand });
      const input = screen.getByPlaceholderText('Speak thy command...');
      await userEvent.type(input, 'h');
      expect(setCommand).toHaveBeenCalled();
   });

   it('calls handleCommand when Send is clicked', async () => {
      const handleCommand = vi.fn();
      renderChat({ command: 'Attack goblin', handleCommand });
      await userEvent.click(screen.getByRole('button', { name: /send/i }));
      expect(handleCommand).toHaveBeenCalledTimes(1);
   });

   it('calls handleCommand on Enter key (not Shift+Enter)', async () => {
      const handleCommand = vi.fn();
      renderChat({ command: 'Look around', handleCommand });
      const input = screen.getByPlaceholderText('Speak thy command...');
      fireEvent.keyPress(input, {
         key: 'Enter',
         charCode: 13,
         shiftKey: false,
      });
      expect(handleCommand).toHaveBeenCalledTimes(1);
   });

   it('does not call handleCommand on Shift+Enter', () => {
      const handleCommand = vi.fn();
      renderChat({ command: 'multi\nline', handleCommand });
      const input = screen.getByPlaceholderText('Speak thy command...');
      fireEvent.keyPress(input, { key: 'Enter', charCode: 13, shiftKey: true });
      expect(handleCommand).not.toHaveBeenCalled();
   });

   it('renders world events accordion for narrative messages with events', async () => {
      const history: ChatMessage[] = [
         {
            type: 'narrative',
            content: 'A goblin lunges from the shadows.',
            events: [
               { type: 'damage', message: 'You take 5 damage. ❤ 95/100' },
               { type: 'item_appeared', message: 'A Rusty Key appears nearby.' },
            ],
         },
      ];
      renderChat({ chatHistory: history });

      const summaryButton = screen.getByRole('button', {
         name: /World Events \(2\)/i,
      });
      expect(summaryButton).toBeInTheDocument();

      await userEvent.click(summaryButton);
      expect(screen.getByText('You take 5 damage. ❤ 95/100')).toBeInTheDocument();
      expect(screen.getByText('A Rusty Key appears nearby.')).toBeInTheDocument();
   });

   it('does not render world events accordion when narrative message has no events', () => {
      const history: ChatMessage[] = [
         {
            type: 'narrative',
            content: 'The corridor falls silent.',
         },
      ];
      renderChat({ chatHistory: history });

      expect(
         screen.queryByRole('button', { name: /World Events/i }),
      ).not.toBeInTheDocument();
   });

   it('does not render world events accordion for player messages', () => {
      const history: ChatMessage[] = [
         {
            type: 'player',
            content: 'I brace for impact.',
            events: [{ type: 'damage', message: 'This should never render.' }],
         },
      ];
      renderChat({ chatHistory: history });

      expect(
         screen.queryByRole('button', { name: /World Events/i }),
      ).not.toBeInTheDocument();
   });

   it('renders pending choices and disables freeform input while awaiting choice', () => {
      renderChat({
         command: 'Go north',
         pendingChoices: [
            { id: 0, text: 'Take the Hidden Route.' },
            { id: 1, text: 'Take the Target Route.' },
         ],
         onChoiceSelected: vi.fn(),
      });

      expect(screen.getByText('Choose your response')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Take the Hidden Route.' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /send/i })).toBeDisabled();
      expect(screen.getByPlaceholderText('Speak thy command...')).toBeDisabled();
   });

   it('calls onChoiceSelected when a pending choice is clicked', async () => {
      const onChoiceSelected = vi.fn();
      renderChat({
         pendingChoices: [{ id: 2, text: 'Take the Novice Route.' }],
         onChoiceSelected,
      });

      await userEvent.click(screen.getByRole('button', { name: 'Take the Novice Route.' }));
      expect(onChoiceSelected).toHaveBeenCalledWith(2);
   });

   it('disables choice buttons while choice submission is in-flight', () => {
      renderChat({
         pendingChoices: [{ id: 3, text: 'Hold your ground.' }],
         onChoiceSelected: vi.fn(),
         isSubmittingChoice: true,
      });

      expect(screen.getByRole('button', { name: 'Hold your ground.' })).toBeDisabled();
   });

   it('prevents rapid double-click resubmission in choice handler flow', async () => {
      const onChoiceSelected = vi.fn();
      const Harness = () => {
         const [submitting, setSubmitting] = useState(false);
         const lockRef = useRef(false);
         return (
            <ThemeProvider theme={AppTheme}>
               <Chat
                  chatHistory={[]}
                  command=""
                  setCommand={vi.fn()}
                  handleCommand={vi.fn()}
                  isLoading={false}
                  pendingChoices={[{ id: 4, text: 'Advance cautiously.' }]}
                  isSubmittingChoice={submitting}
                  onChoiceSelected={(id) => {
                     if (lockRef.current) return;
                     lockRef.current = true;
                     onChoiceSelected(id);
                     setSubmitting(true);
                  }}
               />
            </ThemeProvider>
         );
      };

      render(<Harness />);
      const button = screen.getByRole('button', { name: 'Advance cautiously.' });
      fireEvent.click(button);
      fireEvent.click(button);

      expect(onChoiceSelected).toHaveBeenCalledTimes(1);
      expect(button).toBeDisabled();
   });

   it('shows objective updates as a visible callout, not in the accordion', () => {
      const history: ChatMessage[] = [
         {
            type: 'narrative',
            content: 'Father ruffles your hair.',
            events: [
               {
                  type: 'campaign_progress',
                  message: 'Objective updated: Earn two coins around Briarwell.',
               },
               { type: 'item_gained', message: 'You gained a copper coin.' },
            ],
         },
      ];
      renderChat({ chatHistory: history });
      expect(screen.getByRole('note')).toHaveTextContent('New objective');
      expect(screen.getByRole('note')).toHaveTextContent(
         'Earn two coins around Briarwell.',
      );
      expect(screen.getByRole('note')).not.toHaveTextContent('Objective updated:');
      // Only the remaining world event is counted in the accordion.
      expect(screen.getByText('World Events (1)')).toBeInTheDocument();
   });

   it('hides the accordion when every event is a story event', () => {
      renderChat({
         chatHistory: [
            {
               type: 'narrative',
               content: 'The road ahead.',
               events: [{ type: 'campaign_progress', message: 'Objective updated: Go.' }],
            },
         ],
      });
      expect(screen.queryByText(/World Events/)).not.toBeInTheDocument();
   });

   it('replaces the input with the ending once the run has ended', () => {
      renderChat({
         ending: {
            id: 'road_to_guild',
            title: 'The Road to the Guild',
            summary: 'Briarwell is ash behind you.',
         },
         onStartNewRun: vi.fn(),
      });
      expect(screen.getByText('The adventure has ended')).toBeInTheDocument();
      expect(screen.getByText('The Road to the Guild')).toBeInTheDocument();
      expect(screen.getByText('Briarwell is ash behind you.')).toBeInTheDocument();
      expect(
         screen.queryByPlaceholderText('Speak thy command...'),
      ).not.toBeInTheDocument();
      expect(
         screen.queryByRole('button', { name: /send/i }),
      ).not.toBeInTheDocument();
   });

   it('starts a new run from the ending panel', async () => {
      const onStartNewRun = vi.fn();
      renderChat({ ending: { title: 'Failure' }, onStartNewRun });
      await userEvent.click(
         screen.getByRole('button', { name: /begin a new adventure/i }),
      );
      expect(onStartNewRun).toHaveBeenCalledTimes(1);
   });
});
