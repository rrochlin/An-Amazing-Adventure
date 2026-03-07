import { describe, it, expect, beforeEach } from "vitest";
import { useGameStore } from "@/store/gameStore";
import type { GameStateView, StateDelta } from "@/types/types";

// Reset store between tests
beforeEach(() => {
  useGameStore.getState().reset();
});

const makeGameState = (overrides: Partial<GameStateView> = {}): GameStateView => ({
  current_room: {
    id: "room-1",
    name: "Tavern",
    description: "A smoky tavern",
    connections: { north: "room-2" },
    coordinates: { x: 0, y: 0, z: 0 },
    items: [],
    occupants: [],
  },
  player: {
    id: "player-1",
    name: "Hero",
    description: "The player",
    alive: true,
    health: 100,
    friendly: true,
    inventory: [],
  },
  rooms: {
    "room-1": {
      id: "room-1",
      name: "Tavern",
      description: "A smoky tavern",
      connections: { north: "room-2" },
      coordinates: { x: 0, y: 0, z: 0 },
      items: [],
      occupants: [],
    },
  },
  chat_history: [],
  ...overrides,
});

describe("gameStore", () => {
  it("starts with null gameState and empty messages", () => {
    const { gameState, chatMessages } = useGameStore.getState();
    expect(gameState).toBeNull();
    expect(chatMessages).toHaveLength(0);
  });

  it("setGameState populates state and syncs chat history", () => {
    const state = makeGameState({
      chat_history: [{ type: "player", content: "Hello" }],
    });
    useGameStore.getState().setGameState(state);
    const { gameState, chatMessages } = useGameStore.getState();
    expect(gameState?.current_room.name).toBe("Tavern");
    expect(chatMessages).toHaveLength(1);
    expect(chatMessages[0].content).toBe("Hello");
  });

  it("appendStreamChunk accumulates streaming message", () => {
    useGameStore.getState().appendStreamChunk("Hello ");
    useGameStore.getState().appendStreamChunk("world");
    expect(useGameStore.getState().streamingMessage).toBe("Hello world");
  });

  it("finalizeStreamingMessage commits to chatMessages and resets streaming", () => {
    useGameStore.getState().appendStreamChunk("The goblin attacks!");
    useGameStore.getState().setStreaming(true);
    useGameStore.getState().finalizeStreamingMessage();

    const { chatMessages, streamingMessage, isStreaming } = useGameStore.getState();
    expect(chatMessages).toHaveLength(1);
    expect(chatMessages[0].type).toBe("narrative");
    expect(chatMessages[0].content).toBe("The goblin attacks!");
    expect(streamingMessage).toBe("");
    expect(isStreaming).toBe(false);
  });

  it("finalizeStreamingMessage does nothing if no streaming content", () => {
    useGameStore.getState().finalizeStreamingMessage();
    expect(useGameStore.getState().chatMessages).toHaveLength(0);
  });

  it("addChatMessage appends without affecting gameState", () => {
    const state = makeGameState();
    useGameStore.getState().setGameState(state);
    useGameStore.getState().addChatMessage({ type: "player", content: "Go north" });
    expect(useGameStore.getState().chatMessages).toHaveLength(1);
    expect(useGameStore.getState().gameState?.current_room.name).toBe("Tavern");
  });

  it("applyDelta updates current_room and player", () => {
    const state = makeGameState();
    useGameStore.getState().setGameState(state);

    const delta: StateDelta = {
      current_room: {
        id: "room-2",
        name: "Alley",
        description: "A dark alley",
        connections: { south: "room-1" },
        coordinates: { x: 0, y: -100, z: 0 },
        items: [],
        occupants: [],
      },
      player: {
        id: "player-1",
        name: "Hero",
        description: "The player",
        alive: true,
        health: 80,
        friendly: true,
        inventory: [],
      },
    };

    useGameStore.getState().applyDelta(delta);
    const { gameState } = useGameStore.getState();
    expect(gameState?.current_room.name).toBe("Alley");
    expect(gameState?.player.health).toBe(80);
    // room-2 should be merged into rooms map
    expect(gameState?.rooms["room-2"]?.name).toBe("Alley");
  });

  it("applyDelta appends new_message to chatMessages", () => {
    useGameStore.getState().setGameState(makeGameState());
    useGameStore.getState().applyDelta({
      new_message: { type: "narrative", content: "You hear footsteps." },
    });
    const { chatMessages } = useGameStore.getState();
    expect(chatMessages).toHaveLength(1);
    expect(chatMessages[0].content).toBe("You hear footsteps.");
  });

  it("applyDelta does nothing if gameState is null", () => {
    // Should not throw
    useGameStore.getState().applyDelta({ new_message: { type: "narrative", content: "test" } });
    expect(useGameStore.getState().chatMessages).toHaveLength(0);
  });

  it("reset returns to initial state", () => {
    useGameStore.getState().setGameState(makeGameState());
    useGameStore.getState().addChatMessage({ type: "player", content: "hi" });
    useGameStore.getState().setStreaming(true);
    useGameStore.getState().reset();

    const s = useGameStore.getState();
    expect(s.gameState).toBeNull();
    expect(s.chatMessages).toHaveLength(0);
    expect(s.isStreaming).toBe(false);
  });

  it("wsStatus and wsError update correctly", () => {
    useGameStore.getState().setWsStatus("connected");
    expect(useGameStore.getState().wsStatus).toBe("connected");
    useGameStore.getState().setWsError("timeout");
    expect(useGameStore.getState().wsError).toBe("timeout");
    useGameStore.getState().setWsError(null);
    expect(useGameStore.getState().wsError).toBeNull();
  });
});
