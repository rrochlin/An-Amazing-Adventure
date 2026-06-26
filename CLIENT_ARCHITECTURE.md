# Client Architecture (Phase 7 Runtime)

## Purpose

This document is the current source-of-truth for the Phase 7 client runtime shape. It focuses on the authored-campaign game experience that shipped across PRs #62-#67, not on historical client patterns.

## Current Runtime Model

The client now runs two closely related views of a session:

- **game state**: room, player/self, party, rooms map, and chat history
- **campaign state**: authored progression data returned by the server for campaign-backed sessions

A Phase 7 session is therefore not just "chat plus room updates". It is a live authored runtime where:

- the player chooses a campaign during creation
- the server persists deterministic campaign state
- the client renders that state alongside the normal room/chat UI
- authored dialogue can pause for explicit player choice before progression continues

## State Management Reality

### Zustand is the active-session source of truth

`client/src/store/gameStore.ts` is the authoritative client store for an active game session.

It owns:

- `gameState`
- `campaignState`
- `chatMessages`
- `streamingMessage`
- `isStreaming`
- `wsStatus` / `wsError`
- `worldGenLog` / `worldGenReady`
- `visitedRooms`

The important architectural boundary is:

- **WebSocket frames and game-session reads hydrate the store**
- **rendering components read from the store**
- **ephemeral UI-only concerns stay local to routes/components**

Examples of local-only state that still lives in React component state:

- command input text
- focused room / map-expanded UI
- reconnect toast
- world-gen retry UI
- choice submission lock UI state

This is a deliberate change from older docs that described the client as mostly local-state driven and explicitly "not using global state".

## Store mutation entry points

The main state entry points are:

- `setGameState(...)` — initial/full game-state hydration
- `setCampaignState(...)` — load/poll hydration for authored campaign state
- `applyDelta(...)` — incremental WebSocket updates, including campaign deltas
- `appendStreamChunk(...)` + `finalizeStreamingMessage()` — narrative streaming lifecycle
- `appendWorldGenLog(...)` + `setWorldGenReady()` — world-gen terminal lifecycle

`useGameSocket.ts` is intentionally thin: it manages connection lifecycle and dispatches incoming frame payloads into the store.

## Authored campaign runtime concepts

The client consumes a server-authored runtime model rather than deriving authored state itself.

### Campaign selection

`client/src/routes/create.tsx` loads campaign manifests from `GET api/campaigns` and submits a selected `campaign_id` when creating a new game.

Relevant types:

- `CampaignManifest`
- `CreateGameData`

### Campaign state view

`LoadGame()` returns optional `campaign` data for authored sessions. The client stores and renders that as `CampaignStateView`.

`CampaignStateView` currently exposes:

- `campaign_id`
- `campaign_version`
- `active_node_id`
- `current_objective`
- `bool_flags`
- `labels`
- `counters`
- `active_objectives`
- `active_dialogue`

This is the deterministic authored surface the client should treat as source-of-truth. The client does not infer campaign progression from prose.

### Dialogue state view

`active_dialogue` is represented by `DialogueStateView`:

- `asset_id`
- `current_node`
- `awaiting_choice`
- `pending_choices`

This is the contract for authored Yarn pauses. When `awaiting_choice` is true and `pending_choices` is non-empty, the player is expected to choose from authored options rather than send freeform chat.

## Main runtime surfaces

### Routes

- `client/src/routes/create.tsx`
  - campaign-aware character creation and game start
- `client/src/routes/game-{$sessionUUID}.tsx`
  - primary live session runtime
  - loads initial state
  - owns transient choice-submit guard state
  - wires WebSocket actions into UI
- `client/src/routes/game-{$sessionUUID}.details.tsx`
  - read-only inspection of persisted campaign metadata/state
- `client/src/routes/index.tsx`
  - session list / launcher

### Components

- `client/src/components/Chat.tsx`
  - renders player and narrative messages
  - renders pending authored choices via `ChoicePanel`
  - disables freeform input while choices are pending
- `client/src/components/GameInfo.tsx`
  - renders current campaign node, objective, dialogue metadata, and active objectives alongside normal room/player info
- `client/src/components/WorldGenTerminal.tsx`
  - renders world-generation progress before the session is ready
- `client/src/components/RoomMap.tsx`
  - renders room graph using current `gameState` and `visitedRooms`

### Hooks / services / types

- `client/src/hooks/useGameSocket.ts`
  - WebSocket connection lifecycle and frame dispatch
- `client/src/services/api.game.ts`
  - `ListCampaigns`, `CreateGame`, `LoadGame`, `RetryWorldGen`, `JoinCharacter`
- `client/src/types/types.ts`
  - source-of-truth TS contracts for game, campaign, dialogue, and WebSocket payloads

## Dialogue choice rendering and state

Authored dialogue choice flow is split intentionally between route, store, and presentation layers.

### Rendering

`Chat.tsx` shows a `ChoicePanel` above the input when:

- `pendingChoices` exists
- choices are non-empty
- a choice handler was provided

Each choice is rendered as a button using the server-provided `id` and `text`.

### Input gating

While pending choices are present:

- the freeform chat input is disabled
- the normal submit button is disabled
- the player is steered into the authored branch-selection path

### Choice submission state

`game-{$sessionUUID}.tsx` owns the transient submission guard:

- `isSubmittingChoice`
- `choiceSubmitLockRef`

This route-level state prevents repeated `dialogue_choice` submissions while the server is resolving the first click.

### Transport

Choice submission uses the existing WebSocket action channel:

- `action: "game_action"`
- `sub_action: "dialogue_choice"`
- `payload: <choice id as string>`

No separate client-side dialogue transport exists.

## WebSocket/runtime boundary

The client currently relies on two complementary sources:

- **HTTP `LoadGame()`** for initial/polled session hydration, including campaign state and persisted world-gen logs
- **WebSocket frames** for live turn progression

Important runtime behavior:

- `narrative_chunk` and `narrative_end` drive streamed narrative assembly
- `state_delta` is the authored-turn update vehicle and may include `campaign`
- `game_state_update` refreshes full game state for non-authored direct actions
- `error` updates `wsError`
- `world_gen_log` and `world_gen_ready` drive the terminal-to-game transition

## Practical directory map

```text
client/src/
├── components/
│   ├── Chat.tsx                 # streamed narrative + authored choice UI
│   ├── GameInfo.tsx             # room/player plus campaign metadata
│   ├── RoomMap.tsx              # current room graph / visited-room rendering
│   └── WorldGenTerminal.tsx     # pre-ready world-gen terminal
├── hooks/
│   └── useGameSocket.ts         # WS lifecycle + frame dispatch to Zustand
├── routes/
│   ├── create.tsx               # campaign selection + character creation
│   ├── game-{$sessionUUID}.tsx  # live session runtime
│   └── game-{$sessionUUID}.details.tsx
├── services/
│   └── api.game.ts              # campaign/session HTTP API surface
├── store/
│   └── gameStore.ts             # active-session source of truth
└── types/
    └── types.ts                 # GameStateView / CampaignStateView / WsFrame
```

## What this doc should prevent

When updating the client, do not assume:

- the app is still primarily local-state driven
- authored progression is inferred from narrative text
- freeform chat and dialogue-choice turns are the same interaction type
- campaign data is optional UI decoration

For Phase 7 authored sessions, campaign state is part of the runtime contract.
