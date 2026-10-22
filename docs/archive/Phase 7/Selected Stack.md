---
tags:
  - phase-7
  - architecture
  - dependencies
---

# Selected Stack

> [!summary]
> Phase 7 should use a mixed runtime:
> - **custom campaign JSON** as the top-level authored campaign format
> - **FSM** for deterministic questline / story-node progression
> - **Yarn** for authored dialogue flows

## Status

> [!success]
> Selected and accepted for Phase 7:
> - `github.com/enetx/fsm` for quest/story progression
> - `github.com/DrJosh9000/yarn` for authored dialogue scenes
> - custom campaign JSON remains the overall campaign format

## Rejected Alternatives

- using Yarn as the entire campaign runtime
- using a generic FSM as the entire campaign definition format
- relying on AI narration alone for campaign progression
- keeping procedural `world-gen` as the main bootstrap path

Related notes:

- `[[Ideas]]`
- `[[Story]]`
- `[[Campaign JSON Schema]]`
- `[[Condition Resolution]]`

## Selected Direction

### Campaign Schema

Keep a custom campaign JSON schema as the top-level source of truth.

Why:

- it needs to describe more than dialogue
- it must own rooms, actors, entrances, rails, objectives, and story nodes
- neither Yarn nor a generic FSM library is a good full replacement for that

### Questline Runtime

Use an FSM for story-node / questline progression.

Recommended library:

- `github.com/enetx/fsm`

Why it fits better than a minimal FSM package:

- guarded transitions
- hooks and context
- JSON serialization support
- state visualization
- better fit for persistent runtime state and debugging

### Dialogue Runtime

Use Yarn for authored dialogue.

Recommended library:

- `github.com/DrJosh9000/yarn`

Why:

- established external dialogue format/style
- purpose-built for authored branching dialogue
- good fit for camp briefings, companion conversations, interrogations, and other structured scenes

## Division Of Responsibilities

### Campaign JSON owns

- campaign metadata
- room graph
- actor definitions
- companion roster
- story nodes
- allowed transitions
- condition vocabulary
- references to dialogue assets / entry nodes

### FSM owns

- current story node / quest state
- deterministic transitions between story nodes
- guard evaluation using current runtime data
- transition hooks for state changes

### Yarn owns

- scripted dialogue text
- dialogue options
- local dialogue variables
- authored commands emitted during dialogue

### AI owns

- narration outside structured dialogue scenes
- condition fulfillment classification for freeform player input
- interpretation of ambiguous natural-language choices into authored condition values

## Important Boundary

Yarn should not become the whole campaign engine.

It should be used for **dialogue scenes**, not for:

- full dungeon topology
- actor rail simulation
- campaign persistence model
- global questline truth

Likewise, the FSM should not own all prose. It should own progression, not writing.

## Recommended Story Node Modes

Each story node should declare how it is presented.

Suggested values:

- `ai_scene`
- `yarn_dialogue`
- `hybrid`

### `ai_scene`

Use the narrator + condition resolver flow.

### `yarn_dialogue`

Run a Yarn node as the primary interaction surface.

### `hybrid`

Use AI narration around a Yarn-backed authored conversation or choice moment.

This is likely the best fit for scenes like the camp briefing.

## Phase 7 Convention

Use a readable author-facing condition format in campaign JSON.

That corresponds to the earlier `2/3` direction:

- authors write named condition types
- runtime may normalize them into a more generic internal form

Example author-facing style:

```json
{
  "type": "flag_true",
  "key": "briefing_complete"
}
```

Potential normalized internal form:

```json
{
  "kind": "flag",
  "key": "briefing_complete",
  "op": "==",
  "value": true
}
```

## Practical Result

Phase 7 does not need one giant narrative engine dependency.

It should compose three layers:

1. campaign JSON for authored structure
2. FSM for deterministic progression
3. Yarn for authored dialogue scenes

That keeps the architecture conventional and easier to debug.

## Current Integration Decision

Recommended runtime order:

1. load campaign JSON
2. restore campaign/session runtime state
3. use FSM to determine active story node and legal transitions
4. if the node is Yarn-backed, run Yarn for local dialogue flow
5. if the node is AI or hybrid, run narrator flow
6. resolve authored conditions from the latest turn
7. apply deterministic transition evaluation
