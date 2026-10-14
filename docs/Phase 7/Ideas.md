---
tags:
  - phase-7
  - design
  - obsidian
---

# Phase 7 - One-Shot Campaign Architecture

> [!summary]
> Phase 7 shifts the game from `procedural sandbox with AI framing` to `authored one-shot campaign with AI-guided play`.
>
> Confirmed direction:
> - Ship **one curated campaign** first.
> - Use **structured story beats with AI flavor**.
> - Support **solo player + AI companions** first.
> - Keep a **chat-first UX** while using **soft turn progression** under the hood.
> - Use **Lich's Labyrinth** as the first campaign.
> - Define executable campaigns in **JSON**.
> - Keep the current character creator, but allow campaigns to **restrict classes, races, and setup**.
> - Replace `world-gen` with a **campaign bootstrap** flow that instantiates the starting session state.

## Note Map

- `[[Ideas]]` -> Phase 7 hub, product shape, roadmap, migration plan
- `[[Story]]` -> runtime model, orchestrator design, state model, turn flow
- `[[Lich's Labyrinth]]` -> first shipped campaign spec
- `[[Campaign Data Model]]` -> authored campaign schema
- `[[Campaign JSON Schema]]` -> executable JSON file shape
- `[[Campaign Save State]]` -> persisted runtime campaign state
- `[[Narrator Adherence]]` -> narrator/orchestrator contract
- `[[Condition Resolution]]` -> AI-assisted condition fulfillment, deterministic transitions
- `[[Selected Stack]]` -> FSM + Yarn + campaign JSON architecture
- `[[Implementation Plan]]` -> code-facing rollout plan
- `[[implementation]]` -> detailed working execution document
- `[[Vision and Design Decisions]]` -> canonical answer key for previously-open design questions (authoritative)

## Product Goal

Build a focused, replayable one-shot experience where the player enters a pre-authored campaign and the AI behaves less like a world generator and more like a dramatic game master operating inside clear story boundaries.

The campaign should feel authored, not random.

> [!important]
> **Long-term north star (see `[[Vision and Design Decisions]]`):** the eventual product is a
> **human-authored campaign platform**. Procedural generation and AI are **authoring co-pilots** that
> produce a skeleton/outline a human author then "paints" with their own style — they are **not** the
> runtime story driver. Phase 7's single curated campaign is the **gameplay-first learning vehicle**
> that will teach us what authoring tooling later needs to do.

## Progress Snapshot

> [!success]
> Architecture decisions currently settled:
> - Phase 7 targets **one curated campaign first**.
> - The first campaign is `[[Lich's Labyrinth]]`.
> - Campaigns are defined in **JSON**.
> - The current D&D creator stays, but each campaign can restrict allowed options.
> - `world-gen` should be replaced with a **campaign bootstrap** flow.
> - Story-node transitions are **deterministic**.
> - AI may resolve **which authored conditions were fulfilled**, but not choose branches.
> - The top-level runtime stack is `campaign JSON + FSM + Yarn + AI condition resolution`.
> - Runtime validation should use a **small test campaign first**, not the full `Lich's Labyrinth` content.

## Open Items

- exact Go structs for campaign loading and runtime state
- exact `campaign.json` author-facing condition syntax
- exact Yarn bridge contract for commands and variable mirroring
- exact `ws-chat` orchestration order for AI scenes vs Yarn scenes vs hybrid scenes

## Core Shift

### Before

- `client/src/routes/create.tsx` collects broad character + theme inputs.
- `server/cmd/world-gen/main.go` procedurally generates a dungeon and then asks AI for framing.
- `server/internal/game/game.go` persists a mostly world-centric session.
- The AI is responsible for too much: flavor, structure, pacing, and direction.

### After

- The player chooses a **campaign** instead of asking the system to invent one.
- Session bootstrap loads an authored campaign package.
- The AI operates inside a **story state machine / quest graph**.
- Major beats are authored ahead of time.
- AI narration fills in moment-to-moment description, reactions, banter, and local adaptation.

## Design Pillars

1. **Authored spine**
The campaign has a known dramatic arc, known key actors, and known major beats.

2. **AI as performer, not planner**
AI should embellish, react, and roleplay. It should not invent the main plot structure on the fly.

3. **Soft turns**
The system advances the world in discrete steps, but the player experiences natural conversational play.

4. **Visible intention**
The player should usually know what the party is trying to achieve right now.

5. **Controlled scope**
Phase 7 is not a general campaign authoring platform yet. It is one polished campaign with a reusable runtime pattern.

## What Becomes First-Class

Phase 7 needs a few concepts promoted from prompt text into actual game/session data.

- `campaign_id`
- `campaign_version`
- `story_node_id`
- `story_flags`
- `campaign_clock`
- `party_companions`
- `actor_rails`
- `mission_objectives`
- `allowed_scene_transitions`

These belong in runtime state, not just in prompt context.

## Static vs Dynamic

### Static / Authored

- campaign premise
- map layout and entrances
- default party cast
- major scenes
- key branching points
- important NPC motivations
- win / fail conditions
- actor rails for the rival adventuring party

### Dynamic / AI-Driven

- narration wording
- local room flavor
- party banter
- enemy reactions within authored constraints
- short-form dialogue choices
- tactical description of actions
- minor scene texture and connective tissue

## Recommended Decomposition

### 1. Campaign Package

Represent a campaign as authored content plus runtime rules.

See: `[[Story#Campaign Definition]]`

### 2. Story Runtime

Add a story graph that tracks where the campaign is and what transitions are currently legal.

See: `[[Story#Story Graph]]`

### 3. Orchestrated AI Stack

Split AI responsibilities into:

- an **Orchestrator** that decides what scene/beat is active
- a **Narrator** that writes player-facing prose and dialogue
- a lightweight **Rules / World executor** that performs deterministic state updates

See: `[[Story#AI Responsibilities]]`

### 4. Campaign Bootstrap

Replace freeform world-gen with campaign instantiation.

See: `[[Story#Bootstrap Flow]]`

### 5. First Campaign Content

Lock the first campaign tightly enough that implementation decisions stay concrete.

See: `[[Lich's Labyrinth]]`

## Proposed Migration of Current Systems

### `create` flow

Current state:

- Heavy D&D character creation
- Theme/preferences influence world generation

Phase 7 direction:

- Step 1: choose campaign
- Step 2: confirm role / party premise
- Step 3: optional character customization within campaign boundaries
- Step 4: start campaign

### `world-gen`

Current state:

- procedural dungeon layout
- encounter placement
- narrative framing

Phase 7 direction:

- load campaign definition
- instantiate authored map + actors + initial story node
- optionally stream a short "mission briefing" bootstrap sequence instead of raw generation logs

### session state

Current state:

- session stores world, rooms, combat, title/theme/quest goal

Phase 7 direction:

- session also stores campaign runtime data and story progression
- narrative history should be subordinate to story state, not the other way around

## Delivery Slices

### Slice 1 - Session model

- Add campaign runtime fields to `server/internal/game/game.go`
- Define authored campaign types
- Persist current node, flags, clocks, companion roster

### Slice 2 - Bootstrap and API

- Replace procedural world bootstrap path for the new campaign flow
- Add campaign selection to `client/src/routes/create.tsx`
- Update request/response types in `client/src/services/api.game.ts` and `client/src/types/types.ts`

### Slice 3 - Story orchestration

- Introduce node progression rules
- Introduce actor rails
- Introduce soft turn advancement

### Slice 4 - Narration constraints

- Rework prompts so the narrator receives:
  - active node
  - current objective
  - allowed revelations
  - actor positions / progress
  - scene-specific tone and constraints

### Slice 5 - First campaign content

- Define `[[Lich's Labyrinth]]` as a complete playable package
- Fill missing cast, branch, and fail-state details

### Slice 6 - UX polish

- Replace or soften world-gen terminal for authored campaign intro
- Surface current objective and campaign progress in UI
- Show companion status and party intent

## Phase 7 Non-Goals

- Fully generic campaign authoring UI
- Fully dynamic AI-generated questlines
- Full multiplayer-first balancing
- Large open world simulation
- Infinite branching narrative

## Risks

> [!warning]
> The main risk is leaving too much story control inside prompts.
> If the major beats are not represented in data, the campaign will drift back toward sandbox behavior.

Other risks:

- story nodes are too coarse, making the campaign feel rigid
- story nodes are too fine, making authoring expensive
- AI companions become noisy instead of useful
- soft turns remain implicit in code and become hard to debug

## Open Design Questions

> [!success]
> These are now **resolved** in `[[Vision and Design Decisions]]`. Summary:
>
> - **How customizable is the player character inside a fixed premise?** Character expression lives
>   inside the campaign premise; campaigns may restrict classes/races/setup. Solo-first.
> - **Do companions have authored tactical roles only, or relationship arcs?** Tactical roles + light
>   personality/banter only. **No** deep relationship arcs in Phase 7.
> - **Should the rival party be simulated continuously or advanced by checkpoints?** **Lightly
>   simulated but bounded by authored rails** — its own pace within a fixed authored path.
> - **How visible should campaign clocks be to the player?** The clock is **soft/flavor only**, not a
>   hard timer; pressure is conveyed dramatically, not via an exposed countdown.

Remaining genuinely-open items now live in `[[Vision and Design Decisions]]#Still Open After This Pass`.

## Implementation Checklist

- [ ] Define authored campaign schema
- [ ] Add campaign runtime fields to save state
- [ ] Rework create flow around campaign selection
- [ ] Replace procedural world bootstrap for Phase 7 path
- [ ] Add orchestrator node progression layer
- [ ] Add actor rail advancement
- [ ] Constrain narrator prompts by active story node
- [ ] Implement `[[Lich's Labyrinth]]`
- [ ] Surface objective + party status in the client

## Working Principle

If a piece of information is required for campaign correctness, it should live in structured state.

If a piece of information is only required for flavor, it can remain prompt material.
