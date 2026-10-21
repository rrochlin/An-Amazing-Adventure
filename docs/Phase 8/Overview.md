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

Status: objective recorded 2026-10-21. Nothing is designed yet.

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
