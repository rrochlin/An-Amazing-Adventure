---
tags:
  - phase-7
  - roadmap
  - implementation
---

# Implementation Plan

Related notes:

- `[[Ideas]]`
- `[[Campaign Data Model]]`
- `[[Campaign Save State]]`
- `[[Narrator Adherence]]`
- `[[Lich's Labyrinth]]`
- `[[Selected Stack]]`

## Progress Snapshot

> [!success]
> Completed design decisions:
> - campaign schema is JSON-first
> - procedural `world-gen` is being retired conceptually
> - questline progression should use an FSM
> - authored dialogue should use Yarn
> - node transitions remain deterministic
> - AI is limited to narration and condition-resolution support

> [!success]
> Implemented in code and verified in production:
> - campaign registry + manifest API
> - persisted `CampaignRuntimeState` in schema v5
> - authored bootstrap path for campaign sessions
> - deterministic JSON-driven transition runtime
> - campaign-first create flow with restrictions support
> - campaign state surfaced to the client and updated through websocket `state_delta`
> - real Yarn VM-backed execution of compiled `.yarnc` assets for authored opening dialogue
> - expanded deterministic `test` campaign probe with branching, counters, actor gates, objective-status gates, and failure branches
> - interactive Yarn choices in the UI, enforced via `game_action/dialogue_choice` (#56, #63)
> - Yarn commands bridged into campaign-state mutations (e201cd8)
> - authored turn orchestration extracted from `ws-chat` into `internal/campaigns` (#65, #66)

> [!todo]
> Current major gaps (as of 2026-10-13; open calls tracked in `[[Vision and Design Decisions]]`):
> - narrator contract / scene package enforcement is still partial
> - `[[Briarwell]]` (Fable-opening adaptation, decision 12) has its story graph and dialogue; it needs playtesting, endings, and the raid fight. `Lich's Labyrinth` deferred
> - unified post-turn resolver (EngineerScan + condition resolution are still two passes)
> - run-end state (read-only finished runs) and ordered `endings[]` selection not yet implemented (decision 15)

## Phase 7 Workstreams

## 1. Campaign Registry

Goal: make campaigns selectable and loadable.

Likely code touchpoints:

- `server/internal/campaigns/` (new)
- `server/campaigns/` (new JSON content)
- `server/cmd/http-games/main.go`
- `client/src/routes/create.tsx`
- `client/src/services/api.game.ts`
- `client/src/types/types.ts`

Deliverables:

- campaign manifest listing
- campaign-aware create request
- `Lich's Labyrinth` registered as first campaign
- JSON loader + schema validation

Implementation note:

- keep the create screen focused on campaign selection + D&D character creation
- defer campaign-specific tactical choices like entrance/timing to the opening scene

## 2. Save-State Upgrade

Goal: persist campaign progression as first-class session state.

Likely code touchpoints:

- `server/internal/game/game.go`
- `server/internal/db/db.go`
- `server/internal/game/*_test.go`

Deliverables:

- schema v5
- `CampaignRuntimeState`
- migration behavior for legacy sessions

## 3. Bootstrap Refactor

Goal: replace world generation with campaign instantiation for Phase 7 sessions.

Likely code touchpoints:

- `server/cmd/world-gen/main.go` removal or retirement path
- new campaign bootstrap path
- `server/cmd/http-games/main.go`

Deliverables:

- campaign intro bootstrap
- authored map + actors seeded into session
- initial node + flags set

Recommendation:

- do not reuse `world-gen` conceptually
- introduce a `campaign bootstrap` flow whose only job is session instantiation

Bootstrap should stop at:

- loading campaign JSON
- creating the session
- placing actors and companions
- setting the opening node
- streaming the intro

The player should make entrance/timing decisions during that intro flow, not before session creation.

## 4. Orchestrator Layer

Goal: evaluate player turns against campaign state.

Likely code touchpoints:

- `server/internal/ai/`
- `server/cmd/ws-chat/main.go`
- new package, likely `server/internal/story/` or `server/internal/campaignruntime/`

Deliverables:

- intent evaluation
- condition resolution layer
- node transition logic
- actor rail advancement
- scene package generation

Phase 7 constraints:

- resolver reads latest turn only
- resolver emits only flags / labels / counters
- transition engine remains deterministic

## 5. Dialogue Runtime

Goal: integrate Yarn for authored dialogue scenes.

Likely code touchpoints:

- `server/internal/dialogue/` (new)
- `server/campaigns/**/dialogue/`
- `server/cmd/ws-chat/main.go`

Deliverables:

- [x] Yarn asset loader
- [x] campaign node -> Yarn asset mapping
- [x] minimal VM-backed node execution for authored opening dialogue
- [ ] runtime bridge between Yarn variables/commands and campaign state
- [ ] interactive player-facing Yarn choice handling

## 6. Narrator Contract

Goal: make the narrator adhere to authored progression outside or around Yarn scenes.

Likely code touchpoints:

- `server/internal/ai/bedrock.go`

Deliverables:

- campaign-aware narrator prompt
- scene package injection
- reduced dependence on post-hoc engineer inference
- hybrid scene support around Yarn-backed dialogue

## 7. Client UX

Goal: surface campaign identity and current objective cleanly.

Likely code touchpoints:

- `client/src/routes/create.tsx`
- `client/src/routes/game-{$sessionUUID}.tsx`
- `client/src/components/GameInfo.tsx`
- `client/src/types/types.ts`

Deliverables:

- campaign selection
- current objective display
- companion status
- cleaner authored intro flow

## Suggested Build Order

1. campaign registry + manifest
2. save-state schema
3. authored bootstrap
4. FSM-backed quest runtime
5. Yarn dialogue runtime
6. condition resolver + orchestrator runtime
7. narrator contract
8. UI objective / companion surfacing
9. campaign content tuning

## Suggested First Milestone

> [!milestone]
> A player can start the deterministic `test` campaign, progress through a branching authored probe, and have the session persist node/objective/dialogue state correctly through refresh and reconnect.

That vertical slice is now proven. The next milestone should shift to a true `Lich's Labyrinth` authored scene slice with real Yarn-backed dialogue and choice handling.

## Risks To Watch During Implementation

- trying to generalize authoring too early
- leaving campaign state half in prompts and half in data
- overusing AI where deterministic code should decide canon
- mixing legacy sandbox assumptions into the new flow

## Decision Log Candidates

- [x] campaign definitions in JSON
- [x] replace `world-gen` with campaign bootstrap path
- [x] use FSM for deterministic questline progression
- [x] use Yarn for authored dialogue scenes
- [x] keep node transitions deterministic after AI-assisted condition resolution
- [x] resolve campaign-specific choices through freeform chat, not pre-game menus
- [x] use latest-turn-only context for condition resolution
- [x] limit condition resolver output to flags / labels / counters
- [ ] keep `EngineerScan` for incidental events vs retire it in Phase 7
- [x] keep D&D character creation but allow campaign-specific restrictions

## Immediate Next Work

1. Surface interactive Yarn choices in the runtime/client instead of auto-resolving to a single path.
2. Define and implement the Yarn command -> campaign state bridge.
3. Strengthen the narrator contract / scene package for authored nodes.
4. Implement a real `Lich's Labyrinth` camp-briefing-to-route-choice vertical slice.
