// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useGameSocket } from '@/hooks/useGameSocket';
import { useGameStore } from '@/store/gameStore';

// Mock auth so getStoredTokens works without real localStorage entries
vi.mock('@/services/auth.service', () => ({
   getStoredTokens: vi.fn(() => ({ accessToken: 'test-access-token' })),
}));

// Stub VITE_WS_ENDPOINT
vi.stubEnv('VITE_WS_ENDPOINT', 'wss://test.example.com/ws');

// ── WebSocket mock ────────────────────────────────────────────────────────────
// We capture the last created WS instance and the handlers attached to it
// so tests can simulate server-sent messages.

interface MockWS {
   onopen: (() => void) | null;
   onmessage: ((e: { data: string }) => void) | null;
   onerror: (() => void) | null;
   onclose: (() => void) | null;
   readyState: number;
   send: ReturnType<typeof vi.fn>;
   close: ReturnType<typeof vi.fn>;
   url: string;
}

let lastWs: MockWS | null = null;

class FakeWebSocket implements MockWS {
   static OPEN = 1;
   onopen: (() => void) | null = null;
   onmessage: ((e: { data: string }) => void) | null = null;
   onerror: (() => void) | null = null;
   onclose: (() => void) | null = null;
   readyState = FakeWebSocket.OPEN;
   send = vi.fn();
   close = vi.fn();
   url: string;

   constructor(url: string) {
      this.url = url;
      lastWs = this;
      // Use a microtask (Promise) instead of setTimeout so flushPromises() catches it
      Promise.resolve().then(() => this.onopen?.());
   }
}

vi.stubGlobal('WebSocket', FakeWebSocket);

// Helper: flush all pending microtasks and promises
const flushPromises = () => new Promise((r) => setTimeout(r, 0));

beforeEach(() => {
   lastWs = null;
   useGameStore.getState().reset();
});

afterEach(() => {
   vi.unstubAllEnvs();
});

describe('useGameSocket', () => {
   it('opens a WebSocket URL containing token and gameId', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-123' }));
      await flushPromises();
      expect(lastWs?.url).toMatch(/^wss?:\/\//);
      expect(lastWs?.url).toContain('token=test-access-token');
      expect(lastWs?.url).toContain('gameId=sess-123');
   });

   it('sets wsStatus to connected on open', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      expect(useGameStore.getState().wsStatus).toBe('connected');
   });

   it('sets wsStatus to error on onerror', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      act(() => {
         lastWs?.onerror?.();
      });
      expect(useGameStore.getState().wsStatus).toBe('error');
   });

   it('dispatches narrative_chunk frame to store', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      act(() => {
         lastWs?.onmessage?.({
            data: JSON.stringify({
               type: 'narrative_chunk',
               payload: { content: 'The goblin attacks!' },
            }),
         });
      });
      expect(useGameStore.getState().streamingMessage).toBe(
         'The goblin attacks!',
      );
      expect(useGameStore.getState().isStreaming).toBe(true);
   });

   it('finalizes streaming on narrative_end', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      act(() => {
         lastWs?.onmessage?.({
            data: JSON.stringify({
               type: 'narrative_chunk',
               payload: { content: 'Hello' },
            }),
         });
         lastWs?.onmessage?.({
            data: JSON.stringify({ type: 'narrative_end' }),
         });
      });
      const state = useGameStore.getState();
      expect(state.streamingMessage).toBe('');
      expect(state.isStreaming).toBe(false);
      expect(state.chatMessages[0].content).toBe('Hello');
   });

   it('sets wsError on error frame', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      act(() => {
         lastWs?.onmessage?.({
            data: JSON.stringify({
               type: 'error',
               payload: { message: 'Something went wrong' },
            }),
         });
      });
      expect(useGameStore.getState().wsError).toBe('Something went wrong');
   });

   it('sendChat sends JSON over WS when open', async () => {
      const { result } = renderHook(() =>
         useGameSocket({ sessionId: 'sess-1' }),
      );
      await flushPromises();
      act(() => {
         result.current.sendChat('Go north');
      });
      expect(lastWs?.send).toHaveBeenCalledWith(
         JSON.stringify({ action: 'chat', content: 'Go north' }),
      );
   });

   it('sendAction sends game_action JSON over WS', async () => {
      const { result } = renderHook(() =>
         useGameSocket({ sessionId: 'sess-1' }),
      );
      await flushPromises();
      act(() => {
         result.current.sendAction('move', 'north');
      });
      expect(lastWs?.send).toHaveBeenCalledWith(
         JSON.stringify({
            action: 'game_action',
            sub_action: 'move',
            payload: 'north',
         }),
      );
   });

   it('silently ignores malformed JSON frames', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      expect(() => {
         act(() => {
            lastWs?.onmessage?.({ data: 'not-json' });
         });
      }).not.toThrow();
   });

   it('always opens WebSocket without enabled gate', async () => {
      // The hook now connects unconditionally — no enabled prop
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      expect(lastWs).not.toBeNull();
      expect(lastWs?.url).toContain('sess-1');
   });

   it('calls onWorldReady callback when world_gen_ready frame arrives', async () => {
      const onWorldReady = vi.fn();
      renderHook(() => useGameSocket({ sessionId: 'sess-1', onWorldReady }));
      await flushPromises();
      act(() => {
         lastWs?.onmessage?.({
            data: JSON.stringify({ type: 'world_gen_ready' }),
         });
      });
      expect(useGameStore.getState().worldGenReady).toBe(true);
      expect(onWorldReady).toHaveBeenCalledOnce();
   });

   it('appends world_gen_log lines to store', async () => {
      renderHook(() => useGameSocket({ sessionId: 'sess-1' }));
      await flushPromises();
      act(() => {
         lastWs?.onmessage?.({
            data: JSON.stringify({
               type: 'world_gen_log',
               payload: { line: 'Building rooms...' },
            }),
         });
         lastWs?.onmessage?.({
            data: JSON.stringify({
               type: 'world_gen_log',
               payload: { line: 'Placing items...' },
            }),
         });
      });
      expect(useGameStore.getState().worldGenLog).toEqual([
         'Building rooms...',
         'Placing items...',
      ]);
   });

   it('closes WebSocket on unmount', async () => {
      const { unmount } = renderHook(() =>
         useGameSocket({ sessionId: 'sess-1' }),
      );
      await flushPromises();
      unmount();
      expect(lastWs?.close).toHaveBeenCalled();
   });
});
