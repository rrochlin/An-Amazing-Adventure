# Client Data Flows (Phase 7 Authored Runtime)

## Purpose

This document captures the current live-session flow for Phase 7 as implemented in PRs #62-#67. It focuses on authored campaign turns, WebSocket frame ordering, and how deterministic campaign state reaches the client.

## Runtime Data Surfaces

The client consumes two related server views during play:

### 1. `GameStateView`

Used for the room-and-party runtime:

- `current_room`
- `player` / `self`
- `party`
- `rooms`
- `chat_history`

### 2. `CampaignStateView`

Used for deterministic authored progression:

- `campaign_id`
- `campaign_version`
- `active_node_id`
- `current_objective`
- `bool_flags`
- `labels`
- `counters`
- `active_objectives`
- `active_dialogue`

### 3. `DialogueStateView`

Nested under `campaign.active_dialogue` when a dialogue scene is active:

- `asset_id`
- `current_node`
- `awaiting_choice`
- `pending_choices`

These maps and fields are server-authored state, not client-side derivations.

## Initial Session Hydration

### Create flow

1. `create.tsx` fetches campaign manifests via `ListCampaigns()`.
2. Player selects a `campaign_id` and submits character data.
3. `CreateGame()` creates the session.
4. Client navigates to `/game-{$sessionUUID}`.

### Game route hydration

`game-{$sessionUUID}.tsx` hydrates from `LoadGame()` and polling before relying on live WebSocket turns.

`LoadGame()` can provide:

- `state`
- `campaign`
- `title/theme/quest_goal`
- `world_gen_logs`
- `ready`

High-level behavior:

1. route resets the Zustand store for the new session
2. `LoadGame()` seeds `gameState`, `campaignState`, title, and world-gen logs
3. if the world is not ready, the route keeps polling `LoadGame()`
4. `useGameSocket()` runs in parallel for live frames
5. `world_gen_ready` triggers a fresh `LoadGame()` so the route enters the game with canonical server state

## Turn Types

Phase 7 now has two distinct player-turn categories.

### A. Chat turn

A chat turn is a normal freeform player message sent through the WebSocket `chat` action.

Client behavior:

1. player enters text
2. route appends a local `player` chat message immediately
3. `sendChat(content)` sends `{ action: "chat", content }`
4. streamed narrative arrives through `narrative_chunk`
5. `narrative_end` commits the accumulated narrative message
6. `state_delta` applies the post-turn game/campaign changes

Server-side contract reflected by the client:

- prose is streamed first
- deterministic state lands after streaming completes
- the client should not expect `state_delta` to contain the narrative text itself

### B. Dialogue-choice turn

A dialogue-choice turn happens only when authored dialogue is paused with:

- `campaign.active_dialogue.awaiting_choice = true`
- `campaign.active_dialogue.pending_choices` populated

Client behavior:

1. `Chat` renders the authored choices
2. freeform input is disabled
3. player clicks a choice button
4. route sends `sendAction("dialogue_choice", String(choiceId))`
5. server resolves the authored branch and returns the resulting narrative/state flow

This is not a special chat message. It is a `game_action` sub-action.

## Frame Ordering Contract

### Narrative chat turn

The client assumes this order for normal narrated turns:

1. `narrative_chunk*`
2. `narrative_end`
3. `state_delta`

Why this matters:

- `useGameSocket.ts` finalizes the streaming message on `narrative_end`
- `applyDelta(...)` then applies room/player/campaign changes
- any `delta.events` are attached to the last committed narrative message

The store and UI are intentionally built around that ordering.

### Dialogue-choice turn

For a choice submission, the current contract is:

1. client sends `game_action/dialogue_choice`
2. server may emit authored `narrative_chunk*`
3. server emits `narrative_end`
4. server emits `state_delta` with updated `campaign` and any events

There is **no dedicated choice-ack frame**.

The route resolves the in-flight choice state by watching for either:

- `wsError`, or
- campaign state changing so the dialogue is no longer awaiting a choice

### Direct non-dialogue game actions

For direct actions like movement or inventory actions, the server uses `game_state_update` instead of the authored choice delta pattern.

Client effect:

- `game_state_update` refreshes `gameState`
- `campaignState` usually remains whatever was already loaded unless a later `LoadGame()` or `state_delta` updates it

That asymmetry is intentional in the current implementation and is worth preserving in docs until the transport contract changes.

## Deterministic campaign state maps

The authored runtime exposes three deterministic map families through `CampaignStateView`:

- `bool_flags`
- `labels`
- `counters`

These maps are the client-visible summary of authored runtime mutations made by campaign orchestration and Yarn command handling.

Use them as read-only runtime facts:

- the client displays them or uses them for UI decisions if needed
- the client does not compute or mutate them locally
- refresh/reconnect should trust server-provided values

## Dialogue state flow

`active_dialogue` is the client-facing summary of where an authored scene is paused or progressing.

Typical states:

### Dialogue active, not waiting

- `asset_id` present
- `current_node` present
- `awaiting_choice` false or absent
- `pending_choices` empty or absent

This means authored dialogue context exists, but the player is not currently blocked on a choice.

### Dialogue paused for choice

- `asset_id` present
- `current_node` present
- `awaiting_choice` true
- `pending_choices` populated

This means the next legal player action is choosing one of the surfaced options.

### Dialogue advanced/completed after choice

After resolution, the next `state_delta` may show:

- `awaiting_choice` false
- new `current_node`
- a fresh `pending_choices` set if another immediate choice follows
- updated `active_node_id`, objectives, flags, labels, or counters if the authored branch mutated campaign state

## Choice-submit debounce / guard behavior

The duplicate-submit guard is intentionally simple and route-owned.

In `game-{$sessionUUID}.tsx`:

- `choiceSubmitLockRef` blocks a second send once a choice is clicked
- `isSubmittingChoice` disables the choice buttons
- the lock is released only after:
  - a WebSocket error arrives, or
  - campaign state indicates the dialogue is no longer awaiting a choice

The chat input is also disabled while choices are visible.

High-level effect:

- one click produces one `dialogue_choice` submission
- impatient repeated clicks do not spam the server
- the UI unlocks only when the server resolves the authored state

## Reconnect / refresh behavior

The client should recover authored runtime state through `LoadGame()` rather than trying to reconstruct it locally.

Important consequences:

- pending dialogue choices survive refresh because `campaign.active_dialogue` is persisted server-side
- world-gen logs can be rehydrated from `world_gen_logs`
- the initial game route load seeds both `gameState` and `campaignState` before or alongside live frames

## Source-of-truth summary

For current Phase 7 runtime behavior:

- `types.ts` defines the client contracts
- `api.game.ts` defines the HTTP load/create surface
- `useGameSocket.ts` defines frame handling
- `gameStore.ts` defines in-memory runtime state
- `game-{$sessionUUID}.tsx` defines turn submission and choice-guard behavior
- `Chat.tsx` defines authored choice rendering and input gating

When those files and this document disagree, update this document to match the shipped runtime.
