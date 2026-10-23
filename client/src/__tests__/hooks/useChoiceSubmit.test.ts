// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { useChoiceSubmit } from '@/hooks/useChoiceSubmit';

type State = { awaiting_choice: boolean; node: string };

describe('useChoiceSubmit', () => {
   it('sends once and stays locked until the server answers', () => {
      const send = vi.fn();
      const stall: State = { awaiting_choice: true, node: 'gift_stall' };
      const { result } = renderHook(({ s }) => useChoiceSubmit(s, null, send), {
         initialProps: { s: stall },
      });
      act(() => result.current.submit(1));
      act(() => result.current.submit(2)); // double click
      expect(send).toHaveBeenCalledTimes(1);
      expect(result.current.submitting).toBe(true);
   });

   it('unlocks when the next scene opens straight into its own choice', () => {
      // Playtest 4: gift stall → birthday kept awaiting_choice true throughout,
      // so the birthday's choices stayed disabled until a refresh.
      const send = vi.fn();
      const stall: State = { awaiting_choice: true, node: 'gift_stall' };
      const birthday: State = { awaiting_choice: true, node: 'birthday' };
      const { result, rerender } = renderHook(({ s }) => useChoiceSubmit(s, null, send), {
         initialProps: { s: stall },
      });
      act(() => result.current.submit(1));
      rerender({ s: birthday });
      expect(result.current.submitting).toBe(false);
      act(() => result.current.submit(0));
      expect(send).toHaveBeenLastCalledWith(0);
   });

   it('unlocks on an error', () => {
      const send = vi.fn();
      const s: State = { awaiting_choice: true, node: 'x' };
      const { result, rerender } = renderHook(({ err }) => useChoiceSubmit(s, err, send), {
         initialProps: { err: null as string | null },
      });
      act(() => result.current.submit(1));
      rerender({ err: 'boom' });
      expect(result.current.submitting).toBe(false);
   });
});
