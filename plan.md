# Project Plan

## Purpose

This document captures the current recommended execution plan for the Phase 7 authored-runtime work. It is intended to be edited with detailed feedback as priorities, scope, and implementation decisions change.

## Current Context

The project is in Phase 7 alpha and is centered on a campaign-first, deterministic runtime model:

- authored campaign content lives under `server/campaigns/`
- deterministic runtime state lives under `server/internal/campaigns/`
- Yarn-backed dialogue execution lives under `server/internal/dialogue/`
- WebSocket-driven turn handling currently still carries orchestration weight in `server/cmd/ws-chat/`
- the `test` campaign remains the primary systems probe for validating authored runtime correctness

The main backlog remains in `docs/TODO.md`, but the sequence below is dependency-ordered rather than document-ordered.

## Priority Order

The next three execution items should be handled in this order:

1. Interactive Yarn choices
2. Yarn command to campaign-state bridge
3. Multi-turn scene flow contract

That order matters because item 1 is the current hard blocker, item 2 makes authored progression deterministic without AI guesswork, and item 3 defines the stable boundary before any broader orchestration refactor.

## Detailed Plan

### 1. Implement Interactive Yarn Choice Support

Make authored dialogue pause on player choice instead of auto-selecting the first available option.

Why first:

- the current runtime auto-selects available Yarn options in `server/internal/dialogue/loader.go`
- the campaign runtime clears choice-waiting state in `server/internal/campaigns/runtime.go`
- until this changes, authored dialogue cannot behave like a real player-facing branching system

Primary touch points:

- `server/internal/dialogue/loader.go`
- `server/internal/game/game.go`
- `server/internal/campaigns/runtime.go`
- `client/src/types/types.ts`
- `client/src/store/gameStore.ts`
- `client/src/hooks/useGameSocket.ts`
- relevant game screen/chat UI components in `client/src/components/`

Expected implementation shape:

- extend the dialogue runtime to return available choice options instead of selecting one immediately
- persist pending choice state in campaign runtime data
- expose that pending choice state to the client view models
- render choices in the client and allow the player to explicitly select one
- preserve pending choice state across reconnect and refresh

Acceptance checks:

- a dialogue node with multiple options renders explicit choices to the player
- the session pauses until one option is chosen
- reconnect preserves the pending choice correctly
- selecting a choice advances the Yarn node deterministically


### 2. Build a Minimal Yarn Command to Campaign-State Bridge

After real choice pausing works, support a narrow set of Yarn commands that deterministically mutate campaign state.

Why second:

- the dialogue runtime already captures commands
- those commands are not yet the core mechanism for authored state mutation
- a minimal bridge reduces reliance on AI interpretation for authored progression

Primary touch points:

- `server/internal/dialogue/loader.go`
- `server/internal/campaigns/runtime.go`
- campaign runtime tests under `server/internal/campaigns/`

Recommended initial command set:

- set a campaign flag
- set a campaign label
- adjust a campaign counter
- activate or complete an objective
- start or advance a dialogue scene when explicitly authored

Acceptance checks:

- a Yarn command in a compiled asset mutates campaign state without AI interpretation
- the resulting state is visible in the returned campaign view
- command effects persist correctly after save/load and reconnect

### 3. Define the Multi-Turn Scene Flow Contract

Status: **Decision captured for Milestone C minimal PR scope.**

Formalized contract for authored scene progression across WebSocket turns and client state:

- **Awaiting choice state:** server is source-of-truth via `campaign.active_dialogue.awaiting_choice=true` with `pending_choices`. While true, freeform chat is rejected with `dialogue_choice_required` (deterministic and non-deterministic parity).
- **Choice submission transport:** client submits via WebSocket `game_action` with `sub_action: "dialogue_choice"` and payload choice id. No new frame type is introduced.
- **In-flight choice behavior:** after first click, client blocks repeated `dialogue_choice` sends until the next state or error update resolves the submission.
- **Scene complete state:** scene completion/advance is represented by updated campaign state (`awaiting_choice=false` or next dialogue/node state) delivered in normal state frames.
- **Frame order contract:** for narrative turns, server emits `narrative_chunk*` → `narrative_end` → `state_delta`/`game_state_update` containing updated campaign/dialogue state. Choice turns follow action submit → state/error response; no extra bespoke choice-ack frame.

### 4. Extract Authored Orchestration from ws-chat

After the authored turn loop is explicit, reduce campaign-specific behavior in the WebSocket handler and move progression logic into a cleaner runtime surface.

Why fourth:

- current authored flow still lives too heavily in `server/cmd/ws-chat/main.go`
- there is explicit `test`-campaign-specific handling that should shrink over time
- refactoring before the scene contract is stable would create churn

Primary touch points:

- `server/cmd/ws-chat/main.go`
- `server/internal/campaigns/`
- possibly a new authored orchestrator package if that separation proves useful

Acceptance checks:

- `ws-chat` becomes mostly transport and coordination glue
- authored campaign progression is driven by a reusable runtime/orchestrator surface
- `test`-campaign-specific logic is reduced or isolated behind authored runtime behavior

### 5. Re-Validate the Test Campaign as the Systems Probe

After orchestration changes, use the `test` campaign to verify runtime correctness again under the updated Yarn flow.

Why fifth:

- the `test` campaign is the main deterministic probe for Phase 7 runtime validation
- choice handling and command execution will change the authored runtime materially
- regression coverage should be rebuilt before expanding content work

Primary touch points:

- `server/campaigns/test/`
- authored runtime tests under `server/internal/campaigns/` and `server/internal/dialogue/`
- any environment or QA notes needed to validate production-like behavior

Acceptance checks:

- all expanded `test` campaign branches complete correctly
- state persists across refresh and reconnect
- there are no choice desyncs or unexpected auto-advances

### 6. Implement a Real Lich's Labyrinth Vertical Slice

Use the improved authored runtime to ship the first real player-facing slice of `Lich's Labyrinth`.

Why sixth:

- this is the first content slice that should prove the runtime is useful beyond the systems probe
- the authored campaign data already models briefing and route-choice progression

Primary touch points:

- `server/campaigns/lichs-labyrinth/campaign.json`
- `server/campaigns/lichs-labyrinth/dialogue/`
- any corresponding server/client state handling exposed during real content use

Target slice:

- camp briefing
- route choice
- timing strategy choice
- deterministic state update into the next authored node/objective

Acceptance checks:

- the briefing completes cleanly
- route and timing choice are explicitly player-driven
- campaign labels/objectives update deterministically
- the next node is correct and persisted

### 7. Reconcile Documentation with Runtime Reality

After the runtime flow above is stable, update the stale docs and add a concise current-status authored-runtime page.

Why seventh:

- some top-level client docs no longer fully reflect the authored-runtime architecture
- documentation should follow a stable implementation boundary rather than lead it

Primary touch points:

- `CLIENT_ARCHITECTURE.md`
- `CLIENT_DATA_FLOWS.md`
- `docs/README.md`
- a new or updated authored-runtime milestone/status document under `docs/`

Acceptance checks:

- docs describe the actual Phase 7 authored runtime
- client/server flow documentation matches the real WebSocket and campaign-backed behavior
- current status, milestones, and source-of-truth references are easy to find

## Suggested Milestones

### Milestone A

Interactive choices on the `test` campaign.

### Milestone B

Minimal Yarn command bridge plus persisted scene state.

### Milestone C

Scene flow contract finalized and enforced (guard parity + client choice-submit debounce).

### Milestone D

`Lich's Labyrinth` camp-briefing-to-route-choice vertical slice.

### Milestone E

Documentation reconciliation and authored-runtime status write-up.

## Notes for Review and Feedback

Suggested questions to answer directly in this file while reviewing:

- Is the priority order correct?
- Is choice submission better modeled as chat, action, or a dedicated authored-dialogue message?
- What is the minimum viable Yarn command set?
- Should orchestration extraction happen in one refactor or smaller slices?
- What exact test coverage should gate Milestones A through D?
- Which docs should be considered authoritative after the next round of updates?