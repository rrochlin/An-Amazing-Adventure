# Phase 7 Status

## What Phase 7 is

Phase 7 is the shift from a purely freeform text-adventure loop to a **campaign-backed authored runtime**. Sessions can now carry deterministic campaign progression, active objectives, and Yarn-driven dialogue state in addition to normal room/player game state.

## What shipped

Core authored-runtime milestones are now complete across PRs #62-#67:

- authored campaigns can be selected at game creation time
- campaign runtime state is exposed to the client as `CampaignStateView`
- Yarn dialogue can pause on explicit player choices
- choice submission is enforced through `game_action/dialogue_choice`
- freeform chat is blocked while authored dialogue is awaiting a choice
- narrative/state frame ordering is codified for authored turns
- a minimal deterministic command/state bridge exists through campaign runtime orchestration
- the `test` campaign remains the systems probe
- `ws-chat` orchestration has been reduced in favor of clearer authored runtime surfaces

## Key runtime surfaces

### Client

- `client/src/store/gameStore.ts`
  - active-session source of truth
- `client/src/hooks/useGameSocket.ts`
  - WebSocket frame lifecycle
- `client/src/routes/create.tsx`
  - campaign selection
- `client/src/routes/game-{$sessionUUID}.tsx`
  - live authored/runtime session flow
- `client/src/components/Chat.tsx`
  - dialogue choice rendering and input gating
- `client/src/components/GameInfo.tsx`
  - campaign state visibility in the main play UI

### Shared client contracts

- `CampaignStateView`
- `DialogueStateView`
- `DialogueChoice`
- `StateDelta`
- `WsFrame`

These are defined in `client/src/types/types.ts`.

### Server runtime surfaces

- `server/internal/campaigns/`
  - authored orchestration and dialogue-command application
- `server/internal/dialogue/`
  - Yarn execution
- `server/cmd/ws-chat/`
  - narrated chat turns
- `server/cmd/ws-game-action/`
  - direct actions plus `dialogue_choice`

## Current limitations

This phase is shipped, but a few boundaries are still intentionally narrow:

- campaign state is most directly refreshed on authored `state_delta` turns and initial `LoadGame()` hydration
- non-dialogue direct actions still use `game_state_update`, not a richer authored full-state envelope
- the `test` campaign is still the main runtime probe; broader content authoring is later work
- the planned `Lich's Labyrinth` vertical slice is still future work; this phase wrapped the runtime foundation rather than shipping full authored content breadth
- these docs describe runtime behavior, not the full campaign-authoring workflow

## Practical source of truth

For current behavior, start with:

1. `client/src/types/types.ts`
2. `client/src/store/gameStore.ts`
3. `client/src/hooks/useGameSocket.ts`
4. `client/src/routes/game-{$sessionUUID}.tsx`
5. `server/internal/campaigns/`
6. `server/cmd/ws-chat/` and `server/cmd/ws-game-action/`

This document is intentionally short: it should stay a status page, not become a full design spec.
