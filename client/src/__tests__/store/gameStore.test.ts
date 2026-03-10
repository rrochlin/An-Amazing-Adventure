import { describe, it, expect, beforeEach } from 'vitest';
import { useGameStore } from '@/store/gameStore';
import type { GameStateView, StateDelta } from '@/types/types';

// Reset store between tests
beforeEach(() => {
   useGameStore.getState().reset();
});

const makeGameState = (
   overrides: Partial<GameStateView> = {},
): GameStateView => ({
   current_room: {
      id: 'room-1',
      name: 'Tavern',
      description: 'A smoky tavern',
      connections: { north: 'room-2' },
      coordinates: { x: 0, y: 0, z: 0 },
      items: [],
      occupants: [],
   },
   player: {
      id: 'player-1',
      name: 'Hero',
      description: 'The player',
      alive: true,
      health: 100,
      friendly: true,
      inventory: [],
      equipment: {},
   },
   rooms: {
      'room-1': {
         id: 'room-1',
         name: 'Tavern',
         description: 'A smoky tavern',
         connections: { north: 'room-2' },
         coordinates: { x: 0, y: 0, z: 0 },
         items: [],
         occupants: [],
      },
   },
   chat_history: [],
   ...overrides,
});

describe('gameStore', () => {
   it('starts with null gameState and empty messages', () => {
      const { gameState, chatMessages } = useGameStore.getState();
      expect(gameState).toBeNull();
      expect(chatMessages).toHaveLength(0);
   });

   it('setGameState populates state and syncs chat history', () => {
      const state = makeGameState({
         chat_history: [{ type: 'player', content: 'Hello' }],
      });
      useGameStore.getState().setGameState(state);
      const { gameState, chatMessages } = useGameStore.getState();
      expect(gameState?.current_room.name).toBe('Tavern');
      expect(chatMessages).toHaveLength(1);
      expect(chatMessages[0].content).toBe('Hello');
   });

   it('appendStreamChunk accumulates streaming message', () => {
      useGameStore.getState().appendStreamChunk('Hello ');
      useGameStore.getState().appendStreamChunk('world');
      expect(useGameStore.getState().streamingMessage).toBe('Hello world');
   });

   it('finalizeStreamingMessage commits to chatMessages and resets streaming', () => {
      useGameStore.getState().appendStreamChunk('The goblin attacks!');
      useGameStore.getState().setStreaming(true);
      useGameStore.getState().finalizeStreamingMessage();

      const { chatMessages, streamingMessage, isStreaming } =
         useGameStore.getState();
      expect(chatMessages).toHaveLength(1);
      expect(chatMessages[0].type).toBe('narrative');
      expect(chatMessages[0].content).toBe('The goblin attacks!');
      expect(streamingMessage).toBe('');
      expect(isStreaming).toBe(false);
   });

   it('finalizeStreamingMessage does nothing if no streaming content', () => {
      useGameStore.getState().finalizeStreamingMessage();
      expect(useGameStore.getState().chatMessages).toHaveLength(0);
   });

   it('addChatMessage appends without affecting gameState', () => {
      const state = makeGameState();
      useGameStore.getState().setGameState(state);
      useGameStore
         .getState()
         .addChatMessage({ type: 'player', content: 'Go north' });
      expect(useGameStore.getState().chatMessages).toHaveLength(1);
      expect(useGameStore.getState().gameState?.current_room.name).toBe(
         'Tavern',
      );
   });

   it('applyDelta updates current_room and player', () => {
      const state = makeGameState();
      useGameStore.getState().setGameState(state);

      const delta: StateDelta = {
         current_room: {
            id: 'room-2',
            name: 'Alley',
            description: 'A dark alley',
            connections: { south: 'room-1' },
            coordinates: { x: 0, y: -100, z: 0 },
            items: [],
            occupants: [],
         },
         player: {
            id: 'player-1',
            name: 'Hero',
            description: 'The player',
            alive: true,
            health: 80,
            friendly: true,
            inventory: [],
            equipment: {},
         },
      };

      useGameStore.getState().applyDelta(delta);
      const { gameState } = useGameStore.getState();
      expect(gameState?.current_room.name).toBe('Alley');
      expect(gameState?.player.health).toBe(80);
      // room-2 should be merged into rooms map
      expect(gameState?.rooms['room-2']?.name).toBe('Alley');
   });

   it('applyDelta does NOT add to chatMessages (narrative arrives via streaming)', () => {
      // new_message was removed from StateDelta to fix duplicate messages (UI-3).
      // Narrative text reaches the client via narrative_chunk/narrative_end frames only.
      useGameStore.getState().setGameState(makeGameState());
      useGameStore.getState().applyDelta({
         player: {
            id: 'player-1',
            name: 'Hero',
            description: 'The player',
            alive: true,
            health: 75,
            friendly: true,
            inventory: [],
            equipment: {},
         },
      });
      const { chatMessages, gameState } = useGameStore.getState();
      expect(chatMessages).toHaveLength(0); // no messages added by applyDelta
      expect(gameState?.player.health).toBe(75); // state still updated
   });

   it('applyDelta does nothing if gameState is null', () => {
      // Should not throw
      useGameStore.getState().applyDelta({ player: undefined });
      expect(useGameStore.getState().chatMessages).toHaveLength(0);
   });

   it('attachEventsToLastMessage patches events onto last narrative message', () => {
      useGameStore.getState().setGameState(makeGameState());
      // Simulate streaming narrative committed by finalizeStreamingMessage
      useGameStore.getState().appendStreamChunk('The goblin strikes!');
      useGameStore.getState().finalizeStreamingMessage();
      // Now attach events
      useGameStore
         .getState()
         .attachEventsToLastMessage([
            { type: 'damage', message: 'You take 20 damage. ❤ 80' },
         ]);
      const { chatMessages } = useGameStore.getState();
      expect(chatMessages).toHaveLength(1);
      expect(chatMessages[0].type).toBe('narrative');
      expect(chatMessages[0].events).toHaveLength(1);
      expect(chatMessages[0].events![0].type).toBe('damage');
   });

   it('attachEventsToLastMessage does nothing if last message is a player message', () => {
      useGameStore.getState().setGameState(makeGameState());
      useGameStore
         .getState()
         .addChatMessage({ type: 'player', content: 'Attack!' });
      useGameStore
         .getState()
         .attachEventsToLastMessage([
            { type: 'damage', message: 'You take 5 damage.' },
         ]);
      const { chatMessages } = useGameStore.getState();
      // Events not attached to player messages
      expect(chatMessages[0].events).toBeUndefined();
   });

   it('attachEventsToLastMessage does nothing if chatMessages is empty', () => {
      // Should not throw
      useGameStore
         .getState()
         .attachEventsToLastMessage([{ type: 'damage', message: 'test' }]);
      expect(useGameStore.getState().chatMessages).toHaveLength(0);
   });

   it('reset returns to initial state', () => {
      useGameStore.getState().setGameState(makeGameState());
      useGameStore.getState().addChatMessage({ type: 'player', content: 'hi' });
      useGameStore.getState().setStreaming(true);
      useGameStore.getState().reset();

      const s = useGameStore.getState();
      expect(s.gameState).toBeNull();
      expect(s.chatMessages).toHaveLength(0);
      expect(s.isStreaming).toBe(false);
   });

   it('wsStatus and wsError update correctly', () => {
      useGameStore.getState().setWsStatus('connected');
      expect(useGameStore.getState().wsStatus).toBe('connected');
      useGameStore.getState().setWsError('timeout');
      expect(useGameStore.getState().wsError).toBe('timeout');
      useGameStore.getState().setWsError(null);
      expect(useGameStore.getState().wsError).toBeNull();
   });

   it('appendWorldGenLog accumulates lines', () => {
      useGameStore.getState().appendWorldGenLog('Loading game record...');
      useGameStore.getState().appendWorldGenLog('Summoning the Architect...');
      const { worldGenLog } = useGameStore.getState();
      expect(worldGenLog).toHaveLength(2);
      expect(worldGenLog[0]).toBe('Loading game record...');
      expect(worldGenLog[1]).toBe('Summoning the Architect...');
   });

   it('setWorldGenReady sets worldGenReady to true', () => {
      expect(useGameStore.getState().worldGenReady).toBe(false);
      useGameStore.getState().setWorldGenReady();
      expect(useGameStore.getState().worldGenReady).toBe(true);
   });

   it('reset clears worldGenLog and worldGenReady', () => {
      useGameStore.getState().appendWorldGenLog('some line');
      useGameStore.getState().setWorldGenReady();
      useGameStore.getState().reset();
      const s = useGameStore.getState();
      expect(s.worldGenLog).toHaveLength(0);
      expect(s.worldGenReady).toBe(false);
   });
});
