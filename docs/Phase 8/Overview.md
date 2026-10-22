---
tags:
  - phase-8
  - world
  - map
---

# Phase 8: A Living World

> [!summary]
> **Main objective:** make the world concrete and inhabited. Today it is ephemeral. Rooms, people
> and places mostly exist as text the narrator writes, and nothing keeps the world going between
> turns. Phase 8 adds a central world engine that owns the world's state, and a real map that shows
> it. Phase 7 work on the story loop (Briarwell) continues alongside this.

Status: objective recorded 2026-10-21; agent-engine direction added 2026-10-22. Nothing is designed yet.

## The problem

- **Ephemeral world.** Places and people are described by the narrator and the authored campaign,
  but they have little lasting, simulated state. Of the world's state, only room-to-room movement is
  mechanical. NPCs don't do anything unless the narrator moves them (`move_npc` / `move_player`
  engineer tools, #81).
- **No central engine.** Turn handling is split across `ws-chat`, `ws-game-action` and the
  campaign orchestrator. Each one reacts to a player turn. Nothing advances the world on its own:
  no schedules, routines, time of day, or consequences that carry forward.
- **Weak map.** `RoomMap.tsx` (react-konva) draws a node graph. Campaign maps were only just made to
  lay out at all (BFS layout, #81). In play the map adds little sense of place.

## Direction (to be designed)

1. **World engine.** One deterministic owner of world state (locations, inhabitants, time, routines,
   consequences) that the narrator reads from and that the post-turn resolver writes to (decision
   13). AI narrates and classifies. Code decides what is true, consistent with Phase 7's "state before
   narration".
2. **Concrete places.** Locations with lasting contents and inhabitants who have their own
   whereabouts and routines, so returning to a place shows what changed.
3. **The map as a primary view.** A map that shows that world: where you are, who is where, what
   you have discovered. It is a main part of the screen, not a debug panel.

### Agents as components of the engine (Rob, 2026-10-22)

The narrator, engineer and resolver are separate prompts glued together by handlers. Rob wants a game
engine with the agents as **components**:

- **Purpose:** generate a procedural campaign, or steer a chat session in a chosen direction, while keeping
  an AI that can decide things on the fly.
- **Screenplay form:** scenes are written like a screenplay, with **stage directions** (tone notes,
  blocking, what must happen) rather than loose prose hints.
- **Actor agents:** each character gets a **backstory** and is played by an actor agent that contributes
  dialogue and actions true to its role.
- **Director agent:** a central director coordinates the actors and **edits** their contributions into the
  finished piece the player reads.

#### Evidence: session b31d479d (Briarwell, 2026-10-21)

Each narration failure in that run maps to a missing piece of this engine:

| Failure | Missing piece |
|---|---|
| Tomas (the player's father) narrated as a stranger, the father of other children | Characters with backstories and relationships, i.e. actors who know who they are |
| One deed paid two coins; "that's already enough" contradicted the objective | Engine-owned state (coins, deeds done) instead of prose |
| The narrator played Garrick's stall before its scene, so the gift was bought twice | A director that owns scene boundaries: what is on stage now and what is not yet |
| Bess's doll marked returned though it was never found | Beats with defined meaning, checked against what actually happened |
| An owl in the AI's prose, a carved bird in the authored choice; the raid used the owl | One record of established facts that every agent reads |
| The player never left the cottage | Engine-owned movement (the tool-input bug, fixed in #89) |

#### Open questions

- **Turn pipeline:** does the director plan a beat, actors respond, the engine validate and apply, and the
  director edit the result? How many model calls is that per turn, and when does streaming start?
- **Authored vs generated:** is there one screenplay format that both authored campaigns and a procedural
  campaign generator produce? Where do Yarn dialogue and its choices fit?
- **Actor knowledge:** does each actor only know what its character has seen, or is it omniscient?
- **Phase 7 overlap:** how does this relate to decision 13 (the unified post-turn resolver) and to the current
  narrator, engineer and resolver?
- **Evaluation:** turn traces (#88) make sessions replayable. Could a set of recorded turns serve as a
  regression suite for prompt and agent changes?

Constraint carried over from Phase 7 decision 3: the physical room graph stays first-class, and story
nodes sit on top of it rather than replacing it.

## Earlier notes to draw from

- `[[Loose Thoughts]]`: three.js maze, point-and-click movement to points of interest, a map you fill
  in as you explore, characters on the map, zone maps, "more focus on the map element".
- `Phase 2/Design.md`: show the player's immediate surroundings (roughly a 20×20 grid).
- `Phase 5/UI.md` and `Phase 5/Migration Plan.md` (Q4): keep Konva and fix it; "revisit if the
  pop-out/3D feature is added later".
- `Phase 5/TODO.md` UI-FUT-4/5: room panel tied to the selected map node, NPC tokens on the map.
  These appear unshipped.
- `Phase 7/Campaign JSON Schema.md`: authored room graph (rooms, connections, floors, entrances,
  tags).

## Related issues

- #84: the player-facing log (`ChatHistory`) grows without limit inside the 400 KB session item. A
  world engine adds more lasting state, so session storage needs reworking either way.
- #83: stop persisting authored Yarn text and store line references instead.
- #82: two history views on the dialogue stage.

## Open questions

- What does the engine simulate: time of day, NPC schedules, economy, reputation? What is the
  smallest version that makes Briarwell feel lived in?
- When does the world advance: per player turn, per in-game clock tick, or both?
- Map form: improve the Konva node graph, a tile/grid map, or something richer (3D)? Should
  authored campaign maps carry coordinates or art?
- Where does world state live in DynamoDB once it outgrows one session item (see #84)?
