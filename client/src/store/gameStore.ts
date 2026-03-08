/**
 * gameStore.ts
 * Zustand store — single source of truth for active game state.
 * All components read from here; all mutations go through these actions.
 */
import { create } from "zustand";
import type {
  GameStateView,
  ChatMessage,
  StateDelta,
  WorldEvent,
} from "../types/types";

type WsStatus = "idle" | "connecting" | "connected" | "disconnected" | "error";

interface GameStore {
  // State
  gameState: GameStateView | null;
  chatMessages: ChatMessage[];
  streamingMessage: string; // narrative being streamed, not yet committed
  isStreaming: boolean;
  wsStatus: WsStatus;
  wsError: string | null;
  // World-gen terminal
  worldGenLog: string[];
  worldGenReady: boolean;

  // Actions
  setGameState: (state: GameStateView) => void;
  applyDelta: (delta: StateDelta) => void;
  attachEventsToLastMessage: (events: WorldEvent[]) => void;
  appendStreamChunk: (chunk: string) => void;
  finalizeStreamingMessage: () => void;
  addChatMessage: (msg: ChatMessage) => void;
  setStreaming: (v: boolean) => void;
  setWsStatus: (s: WsStatus) => void;
  setWsError: (e: string | null) => void;
  appendWorldGenLog: (line: string) => void;
  setWorldGenReady: () => void;
  reset: () => void;
}

const initialState = {
  gameState: null,
  chatMessages: [],
  streamingMessage: "",
  isStreaming: false,
  wsStatus: "idle" as WsStatus,
  wsError: null,
  worldGenLog: [] as string[],
  worldGenReady: false,
};

export const useGameStore = create<GameStore>((set, get) => ({
  ...initialState,

  setGameState: (state) =>
    set({
      gameState: state,
      chatMessages: state.chat_history ?? [],
    }),

  applyDelta: (delta) => {
    const { gameState } = get();
    if (!gameState) return;

    const updated: GameStateView = { ...gameState };

    if (delta.current_room) {
      updated.current_room = delta.current_room;
      // Also update in rooms map so it stays in sync
      updated.rooms = {
        ...updated.rooms,
        [delta.current_room.id]: delta.current_room,
      };
    }
    if (delta.player) {
      updated.player = delta.player;
    }
    if (delta.updated_rooms) {
      updated.rooms = { ...updated.rooms, ...delta.updated_rooms };
    }

    // new_message intentionally not handled here — narrative text arrives via
    // narrative_chunk / narrative_end streaming frames. Adding it here caused
    // duplicate chat messages (UI-3 fix).
    set({ gameState: updated });
  },

  attachEventsToLastMessage: (events) => {
    if (!events.length) return;
    const { chatMessages } = get();
    if (!chatMessages.length) return;
    const last = chatMessages[chatMessages.length - 1];
    // Only attach to narrative messages (not player messages)
    if (last.type !== "narrative") return;
    const updated = [...chatMessages];
    updated[updated.length - 1] = { ...last, events };
    set({ chatMessages: updated });
  },

  appendStreamChunk: (chunk) =>
    set((s) => ({ streamingMessage: s.streamingMessage + chunk })),

  finalizeStreamingMessage: () => {
    const { streamingMessage, chatMessages } = get();
    if (!streamingMessage) return;
    set({
      chatMessages: [
        ...chatMessages,
        { type: "narrative", content: streamingMessage },
      ],
      streamingMessage: "",
      isStreaming: false,
    });
  },

  addChatMessage: (msg) =>
    set((s) => ({ chatMessages: [...s.chatMessages, msg] })),

  setStreaming: (v) => set({ isStreaming: v }),

  setWsStatus: (s) => set({ wsStatus: s }),

  setWsError: (e) => set({ wsError: e }),

  appendWorldGenLog: (line) =>
    set((s) => ({ worldGenLog: [...s.worldGenLog, line] })),

  setWorldGenReady: () => set({ worldGenReady: true }),

  reset: () => set(initialState),
}));
