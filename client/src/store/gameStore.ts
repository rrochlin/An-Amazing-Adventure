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

  // Actions
  setGameState: (state: GameStateView) => void;
  applyDelta: (delta: StateDelta) => void;
  appendStreamChunk: (chunk: string) => void;
  finalizeStreamingMessage: () => void;
  addChatMessage: (msg: ChatMessage) => void;
  setStreaming: (v: boolean) => void;
  setWsStatus: (s: WsStatus) => void;
  setWsError: (e: string | null) => void;
  reset: () => void;
}

const initialState = {
  gameState: null,
  chatMessages: [],
  streamingMessage: "",
  isStreaming: false,
  wsStatus: "idle" as WsStatus,
  wsError: null,
};

export const useGameStore = create<GameStore>((set, get) => ({
  ...initialState,

  setGameState: (state) =>
    set({
      gameState: state,
      chatMessages: state.chat_history ?? [],
    }),

  applyDelta: (delta) => {
    const { gameState, chatMessages } = get();
    if (!gameState) return;

    const updated: GameStateView = { ...gameState };

    if (delta.current_room) {
      updated.current_room = delta.current_room;
      // Also update in rooms map
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

    const newMessages = delta.new_message
      ? [...chatMessages, delta.new_message]
      : chatMessages;

    set({ gameState: updated, chatMessages: newMessages });
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

  reset: () => set(initialState),
}));
