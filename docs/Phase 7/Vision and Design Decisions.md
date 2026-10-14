---
tags:
  - phase-7
  - vision
  - design
  - decision-log
---

# Vision and Design Decisions

> [!summary]
> This note is the canonical answer key for previously-open Phase 7 design ambiguities.
> It supersedes the scattered "Open Questions" / "Open Design Questions" sections in the other
> Phase 7 notes. When those notes disagree with this one, this note wins.

Related notes:

- `[[Ideas]]`
- `[[Story]]`
- `[[Lich's Labyrinth]]`
- `[[Narrator Adherence]]`
- `[[Condition Resolution]]`
- `[[Selected Stack]]`
- `[[implementation]]`

## Product North Star

> [!important]
> The long-term goal is a **human-authored campaign platform**, not a procedural sandbox and not
> a purely curated set of one-shots.

- The platform exists so a **human author** can build authored campaigns.
- **Procedural generation and AI are authoring co-pilots**, not the runtime story driver. Their job
  is to produce an **outline / skeleton** (a stable underlying structure) that the human author then
  **"paints"** with their own style and preference.
- The payoff is **unique authored experiences backed by a proven deterministic structure**: authors
  get creative freedom on top of a framework that keeps campaigns stable and inspectable.
- At runtime, AI is still constrained to **narration + condition classification**, never to inventing
  the main arc. The "AI helps author" idea is about the **build-time** toolchain, not loosening the
  runtime contract.

## Authoring Approach (Now)

> [!note]
> Gameplay-first. Authoring tooling later.

- There is currently **one author** (the project owner).
- Hand-editing **segmented Yarn + JSON** is an **accepted, deliberately substandard** dev experience
  for now. The bottleneck to solve first is **"is the gameplay fun?"**, not "is authoring pleasant?".
- ~~`[[Lich's Labyrinth]]` is the **learning vehicle**.~~ Superseded by decision 12: the first
  campaign is an adaptation of the *Fable* opening; Lich's Labyrinth is deferred. Building it by hand
  is how we discover what campaign authoring actually requires. Those lessons later inform the AI/procedural
  skeleton-generator and any authoring UI.
- Rationale: nobody authors a campaign for an unfun game, and we are not yet experienced enough as
  campaign creators to design the authoring tools. So: **defer authoring UX, invest in the runtime +
  flagship campaign.**

## Resolved Decisions

### 1. Players and party

- **Solo-first.** Party feel in Phase 7 comes from **AI companions**.
- The architecture must **not preclude human co-op later**. `JoinCharacter`, characters, and party
  concepts stay as multiplayer-friendly foundations.
- Human co-op is a **plausible future**, not a Phase 7 deliverable.

### 2. Companions

- **Tactical roles + light personality and banter. No deep arcs in Phase 7.**
- Each companion (Doren / Amalia / Rowan) gets a combat/tactical role, a personality hook, and
  occasional banter.
- **No** authored loyalty/approval/betrayal/relationship arcs yet — explicitly out of scope to keep
  story control high and authoring cheap.

### 3. Movement and the map

- **Keep mechanical room-to-room movement** (`RoomMap` + `game_action move`).
- **Story nodes gate/overlay the physical room graph**; they do not replace it.
- The dungeon stays a real navigable topology (rooms, items, occupants). A node can restrict or
  permit movement, react to where the player physically is, and advance rails accordingly.
- Rooms are **not** reduced to informational-only decoration.

### 4. Combat

> [!warning]
> This is a deliberate **shift** from the current implementation.

- Target: **deterministic combat resolution** (a dice/stats engine, D&D-style, aligned with existing
  `CharacterCreationData` stats and rpg-toolkit) computes outcomes.
- The **narrator only describes** the already-resolved result. State is decided before narration.
- Current state (as of 2026-10-13): `server/internal/combat/` already resolves player weapon attacks,
  monster reply turns, and initiative via rpg-toolkit (`game_action/attack`; 5 monster types).
  Still to migrate away from: freeform chat can also inflict damage through AI tools
  (`damage_character` / `heal_character` / `set_character_alive`) via `EngineerScan` reading prose.
- Open follow-up: how much full combat engine to build for the flagship vs. abstracted encounter
  resolution.

### 5. Campaign clock

- **Soft / flavor only.** Not a hard simulated timer; no exposed countdown the player must beat.
- Time pressure is conveyed **dramatically** (the rival party feels close/ahead).
- `campaign_clock` may persist as a coarse pacing counter but must not be surfaced as a strict timer.

### 6. Rival party rail

- **Lightly simulated, bounded by authored rails.**
- The rival has its own forward agenda but cannot leave the authored path sequence
  (floor 1 → intersection → F1 boss → F2 → F2 boss → F3 → lich → retrace).
- Player actions can **disrupt, pause, or accelerate** the rival **within those bounds**. It never
  becomes a free open-ended agent.
- Advancement is driven by **player progress + authored hold/reveal/advance conditions**, not a clock.

### 7. Death, failure, and replay

- **Permadeath one-shot.** A wipe / mission failure **ends the run** with an authored aftermath.
- **No retry or reload of the same save.** Players replay by starting a **new session**.
- Failure must always route to a **satisfying authored ending/aftermath node** — game-over is
  narratively framed, never a dead-end error.
- The run should be marked complete/ended in save state.

### 8. Endings

- **Discrete authored ending nodes**: clean success / costly success / partial success / failure /
  catastrophe.
- The ending is **selected deterministically by final campaign flags** (code evaluates accumulated
  flags/labels/counters against authored conditions).
- The narrator **dramatizes** the chosen ending but does **not** pick it. Each ending is an authored
  node with its own constraints / scene package.

### 9. Condition resolver vs. the `test` campaign

- The **AI condition resolver remains the real target** for production campaigns.
- The `test` campaign's literal keyword routing (`proceed` / `hidden` / `scan` / `target` /
  `novice` / `verify`) is **only a deterministic probe placeholder** — it validates the
  FSM/dialogue/persistence plumbing without AI nondeterminism.
- Production flow (e.g. `[[Lich's Labyrinth]]`): narrator/Yarn run the scene → AI resolver classifies
  the **latest turn only** into authored flags/labels/counters (constrained vocabulary) → code
  validates → deterministic FSM transition.
- Do **not** read the `test` campaign as a pivot away from AI resolution.

### 10. Unified post-turn state resolution

> [!important]
> **Unify** the Phase 7 condition resolver and the Phase 8 Engineer / `EngineerScan` into a **single
> post-turn state-resolution step.**

- Instead of two parallel AI inference passes (one for world/combat mutations, one for story flags),
  run **one** coherent post-turn stage that reads the latest turn and emits **validated structured
  mutations** — world/combat events **and** story flags/labels/counters — against an allowed
  vocabulary.
- Code validates the output, then runs deterministic FSM transition + world apply.
- Because combat outcomes are themselves deterministic (decision 4), this unified resolver is about
  **classifying/recording what happened**, not inventing combat results.

### 11. Scope and replayability

- Target **single sitting (~30–60 min)** per run, with **high replayability via branching**.
- Story nodes should be **coarse "playable dramatic units"** (camp briefing, entry choice, floor
  shadow game, escalation, ambush setup, collision, aftermath), not fine implementation steps.
- Each campaign needs enough authored branching (entry route × timing × key choices × discrete
  endings) that replays feel meaningfully different.
- Replayability comes from **branch/ending variety + AI flavor**, not from length. This validates the
  permadeath model: a wipe costs ~30–60 min, not many hours.

### 12. First campaign: adapt the *Fable* opening (decided 2026-10-13)

- **Pivot away from `[[Lich's Labyrinth]]` as the first vertical slice.** Instead, model a limited,
  clearly scoped campaign on the opening of *Fable*: village life up to the bandit raid, the player
  hiding, and being rescued by the guild master.
- Rationale: copying a known-good story gives concrete beats to build against, so we validate the
  **software design and architecture separately from the campaign writing itself**.
- Lich's Labyrinth (original campaign) is deferred, not abandoned; revisit once the runtime is proven.
- Its open content questions (the artifact, the Lich, Ice Pits, etc.) are parked with it.
- Note: this is an internal validation campaign. Rename characters/places before any public release.

### 13. Unified post-turn resolver: single structured-output call (decided 2026-10-13)

- **One call, one JSON schema, no tool loop**: the resolver returns
  `{world_events[], story_conditions[]}` in a single pass, replacing `EngineerScan` +
  `ResolveCampaignConditions`.
- Context the tool loop used to fetch (e.g. `get_room_info`) is supplied in the prompt instead.
- **Authored nodes restrict the allowed world-event types**, alongside their existing
  `allowed_conditions` (e.g. no `create_room` during a scripted scene). Code drops anything outside
  the node's allowlist.
- **`damage_character` / `heal_character` stay in the vocabulary** even after deterministic combat:
  damage and healing can happen outside combat from events/checkpoints (traps, story beats, rests).

### 14. Combat: full deterministic 5e tactical (decided 2026-10-13)

- Build out **full turn-by-turn 5e combat** as the foundation (extends `server/internal/combat/`):
  spells/abilities, stat blocks for campaign foes, encounter hooks from story nodes.
- **Companions do not act autonomously.** They contribute bonuses and/or are player-controlled on
  their turns — no companion AI.
- **"Unwinnable" fights are long odds, not hard-coded failures.** An overwhelming foe starts at
  ~1% win odds, and authored story events/decisions shift those odds. Failure still routes to an
  authored aftermath (decision 7).

## Implications For The Runtime Contract

These decisions tighten, rather than relax, the existing Phase 7 principles:

- **State before narration** is reinforced by deterministic combat (4), deterministic ending
  selection (8), and the unified resolver (10).
- **Determinism with AI as classifier** is reaffirmed (9, 10): AI says *what happened*, code says
  *what it means and where the story goes*.
- **Authored spine, AI as performer** now extends to combat outcomes and endings, not just narration.
- The **physical room layer stays first-class** (3) underneath the story-node layer.

## Still Open After This Pass

> [!note]
> These are being decided in the shared doc
> [Phase 7 — Open Design Decisions](https://claude.ai/code/artifact/279d769d-3ca6-4ae7-8d7e-dc0fc059e954)
> (opened 2026-10-13). Each resolved item is appended above as a numbered decision (12, 13, …) and
> removed from this list.

- How permadeath end-of-run is represented in save state and surfaced in the client (from decision 7).
- Where deterministic ending-selection rules live in `campaign.json` (from decision 8).
