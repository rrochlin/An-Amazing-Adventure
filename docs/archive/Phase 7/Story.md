---
tags:
  - phase-7
  - system-design
  - story-runtime
---

# Story Runtime Design

> [!info]
> The campaign is the source of truth for progression.
> The AI is responsible for presentation and local adaptation, not for inventing the main arc.

Related notes:

- `[[Ideas]]`
- `[[Lich's Labyrinth]]`
- `[[Campaign Data Model]]`
- `[[Campaign Save State]]`
- `[[Narrator Adherence]]`
- `[[Selected Stack]]`

## Runtime Goal

Create a runtime model where authored story beats, world state, and AI narration reinforce each other instead of competing for control.

Phase 7 should do this with three layers:

- campaign JSON for authored structure
- FSM for deterministic questline progression
- Yarn for authored dialogue scenes

## Campaign Definition

A campaign definition should be a versioned JSON package loaded at session start.

Suggested shape:

```ts
type CampaignDefinition = {
  id: string
  version: string
  title: string
  premise: string
  playerRole: string
  tone: string
  map: CampaignMap
  companions: CompanionTemplate[]
  actors: ActorTemplate[]
  storyGraph: StoryNode[]
  objectives: ObjectiveTemplate[]
  intro: IntroSequence
}
```

The important part is not the exact type. The important part is that the campaign owns the structure.

In implementation terms, the server should load campaign JSON and validate it into runtime structs.

Campaign JSON should also reference any dialogue assets needed by the node runtime, such as Yarn files and entry nodes.

## Campaign Session State

Each running game should add a story-runtime layer on top of the current game state.

Suggested fields:

```ts
type CampaignSessionState = {
  campaignId: string
  campaignVersion: string
  storyNodeId: string
  storyFlags: Record<string, boolean | string | number>
  campaignClock: number
  objectiveIds: string[]
  companionStates: CompanionState[]
  actorStates: ActorState[]
  sceneState: SceneState
}
```

This should live in persisted state, not be reconstructed from chat history.

## Story Graph

The questline idea from the original notes should become a proper directed graph.

Each node should contain:

- `id`
- `title`
- `purpose`
- `entry_conditions`
- `completion_conditions`
- `failure_conditions`
- `allowed_transitions`
- `world_setup`
- `ai_constraints`
- `player_objective_text`

### Node Granularity

Nodes should represent **playable dramatic units**, not tiny implementation steps.

Good examples:

- `camp_briefing`
- `choose_entry_and_timing`
- `shadow_target_party_f1`
- `intercept_after_f2_boss`
- `final_confrontation`
- `escape_or_aftermath`

Bad examples:

- `player_clicked_entrance_button`
- `hero_party_moved_one_room`
- `narrator_described_corridor`

## Story Flags

Story flags carry important consequences across nodes.

Examples for Phase 7:

- `entered_via_hidden_entrance`
- `arrived_before_target_party`
- `novices_warned`
- `target_party_alerted`
- `mini_boss_f1_alive`
- `mini_boss_f2_alive`
- `lich_alive`
- `ambush_failed`
- `party_split`

Flags should be small, explicit, and inspectable.

## Actor Rails

Not every actor needs full simulation.

For Phase 7, most important non-player groups should run on authored rails.

Examples:

- target adventuring party progresses through the labyrinth on a known path
- novice party enters, reaches the intersection, then retreats to beginner areas
- monsters mostly remain in designated zones unless triggered

Actor rails should define:

- path / location sequence
- advancement rules
- hold conditions
- reveal conditions
- fallback behavior if disrupted by player action

This gives determinism without making the world feel frozen.

## Soft Turn Model

The player should still type natural language, but internally the game should advance in steps.

### Turn Loop

1. Player submits input.
2. Orchestrator interprets intent in the context of the active story node.
3. Deterministic game/world updates run.
4. Narrator receives the current scene package and allowed turn outcomes.
5. Narrator streams the response.
6. A condition resolver agent classifies which structured conditions were fulfilled during the turn.
7. Actor rails advance if allowed.
8. Story node progression is evaluated deterministically.
9. Updated runtime state is persisted.

This is turn-based in state management, but not necessarily in presentation.

## AI Responsibilities

### Orchestrator

Responsibilities:

- decide what scene the game is in
- validate which transitions are legal
- advance story nodes
- decide what information the narrator is allowed to reveal
- decide whether actor rails advance this step
- evaluate transitions after condition fulfillment has been resolved

The orchestrator should be mostly deterministic and heavily state-driven.

It should coordinate both:

- the questline FSM
- the active scene mode (`ai_scene`, `yarn_dialogue`, or `hybrid`)

### Dialogue Runtime

Use Yarn for structured authored dialogue scenes.

Best uses:

- camp briefing
- companion banter trees
- negotiations
- interrogations
- high-stakes confrontations with explicit authored beats

Yarn should handle local dialogue flow, but not replace the campaign runtime.

### Condition Resolver

The condition resolver is an AI-assisted classification layer.

Responsibilities:

- inspect the completed turn
- map what happened onto a fixed set of campaign conditions / flags
- return structured condition results only

The condition resolver should not:

- choose the next node
- invent new flags or effects outside the campaign vocabulary
- mutate campaign state directly

This is the right place for AI judgment about ambiguous narrative outcomes, while keeping progression deterministic.

### Narrator

Responsibilities:

- describe the scene
- roleplay characters in-scene
- present consequences of actions
- maintain tone, pacing, and dramatic texture

The narrator should not:

- invent new major objectives
- skip required campaign beats
- move important actors off-rail without permission
- reveal unreached campaign knowledge

### Rules / World Executor

Responsibilities:

- movement
- item state
- combat state
- room occupancy
- explicit world mutations

This should remain deterministic wherever possible.

> [!important]
> **Combat target (see `[[Vision and Design Decisions]]` §4):** combat resolution should become a
> **deterministic dice/stats engine** (D&D-style, aligned with `CharacterCreationData` + rpg-toolkit).
> The engine computes outcomes; the narrator only **describes** the already-resolved result. This is a
> shift away from the current AI-tool-inferred combat (`damage_character` / `heal_character` via
> `EngineerScan`).
>
> **Post-turn resolution target (§10):** the condition resolver and the Phase 8 `EngineerScan` should
> be **unified into one post-turn state-resolution step** that emits validated structured mutations
> (world/combat events **and** story flags/labels/counters), rather than two parallel AI passes.

## Scene Package

Before each narrator response, construct a compact scene package.

Suggested contents:

- active story node
- current objective
- player party location
- visible actors
- recent world events
- actor rail summary
- forbidden revelations
- allowed condition vocabulary for this turn
- allowed outcome envelope for this turn

For nodes that use Yarn, the scene package should also include the active Yarn entry node and any campaign variables that need to be mirrored into dialogue state.
- allowed scene outcomes

This is the boundary object between state and prose.

## Bootstrap Flow

Phase 7 bootstrap should look more like campaign instantiation than world generation.

### Proposed flow

1. Client selects campaign.
2. Server creates a new session with `campaign_id`.
3. Bootstrap loads the campaign JSON definition.
4. Bootstrap spawns authored map, companions, and actor rails.
5. Bootstrap sets the initial node.
6. Bootstrap emits a short intro / briefing stream.
7. Session becomes playable.

This can still reuse the current async infrastructure if helpful, but the work being done is different.

The conceptual shift is:

- old flow: generate a world
- new flow: instantiate a pre-authored campaign session

There is no real need for `world-gen` once campaigns are prebuilt.

## What Should Stay From Current Architecture

- WebSocket streaming for narrative output
- persisted save state
- chat-first UX
- combat subsystem where already reliable
- room/node map representation as the physical layer of the dungeon

## What Needs Refactoring

### Current mismatch

Today the system stores:

- rooms
- players
- monsters
- title/theme/quest goal
- narrative history

But it does not store:

- active dramatic beat
- authored actor progress
- campaign consequences
- scene constraints

That is the main architectural gap for Phase 7.

## Authoring Principle

Campaign authors should define:

- where the story can go
- what must happen before key reveals
- which actors are where and why
- what outcomes count as success, failure, or escalation

The AI should define:

- how it feels
- how it sounds
- how characters speak
- how the moment is dramatized

## Suggested First Data Additions

For `server/internal/game/game.go` / save state:

- `CampaignID`
- `CampaignVersion`
- `StoryNodeID`
- `StoryFlags`
- `CampaignClock`
- `ObjectiveText`
- `ActorStates`
- `CompanionStates`

For `client/src/types/types.ts` / load responses:

- campaign summary metadata
- current objective
- current story node title
- companion roster/status

For `client/src/routes/create.tsx`:

- campaign selection step
- campaign premise preview
- role / party preview

## Debuggability Requirement

> [!important]
> If the orchestrator changes the active node or actor rails, that change should be inspectable in persisted state and easy to log.

Phase 7 will be much easier to debug if node transitions are explicit and observable.

## Open Questions

> [!success]
> Resolved in `[[Vision and Design Decisions]]`:
>
> - **Should node transitions be fully deterministic, or AI-recommended then validated?** Transitions
>   stay **deterministic in code**. AI only **classifies** which authored conditions a turn fulfilled
>   (latest-turn-only, constrained vocabulary); code decides the branch.
> - **Should companion banter be generated every turn or only on marked opportunities?** Companions
>   are tactical roles + **light** banter — banter is occasional/marked, not every turn, and there are
>   no deep arcs in Phase 7.
> - **How many story flags are too many?** Keep nodes coarse ("playable dramatic units") and flags
>   small/explicit; the ~30–60 min single-sitting scope bounds how much flag complexity is warranted.

## Build Order

- [ ] Define campaign and runtime schema
- [ ] Add save-state support
- [ ] Add bootstrap flow for authored campaigns
- [ ] Add orchestrator node evaluation
- [ ] Add actor rail advancement
- [ ] Constrain narrator prompt by scene package
- [ ] Expose objective and campaign metadata in UI
