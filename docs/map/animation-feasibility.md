# Feasibility: characters who move, react and speak on the scene

> 2026-10-24. Spike for Rob's pivot: sprites that move and shake, and dialogue that comes from the
> characters on the scene in speech bubbles, in a web game (no Unity build). Question: is our tech right
> for it, and what's missing?

## Short answer

**Feasible on the web, and the rendering library is the smallest part of it.** Any of the options below
can walk a sprite to a spot, shake it and float a bubble over its head. The larger gaps are in the
platform:

1. The server knows which room someone is in, but not where in it.
2. The game master hands in a block of prose, not a performance (who says what, who moves where, in
   what order).
3. Our art has no animation frames.
4. The client has no player for timed sequences.

Close those and the renderer choice becomes a contained decision.

**Recommendation:** keep React for the UI. Put the scene on a **PixiJS v8** canvas through
**@pixi/react v8**, with speech bubbles as HTML positioned over it. Drive it from a new **performance
script** the game master hands in, made of beats. Start with procedural motion (hop, bob, shake,
squash) on the art we have, and decide on walk-cycle art separately (the licence and hosting questions
below). A two-day prototype of one scene should come before committing.

## Where we are

| Layer | Today |
|---|---|
| Scene rendering | React DOM. The tiles are drawn once onto a `<canvas>`; characters are absolutely positioned `<div>`s with CSS transitions (`WorldMap.tsx`). No game loop. |
| Character art | Kenney Tiny Dungeon: one static 16×16 frame per character. No walk cycles, idle frames or emotes. |
| Positions | The server stores a room per character. The client spreads characters over free tiles in the room's rectangle (`placeTokens`). Points of interest have tile coordinates, but only in the client art. |
| Movement | A character's tile changes, and its sprite glides there in a straight line over 0.8s. No paths, no facing, no walking frames. |
| Dialogue | The game master's narration is one prose block with quoted speech inside it. Authored Yarn lines are `Speaker: line`, split by regex (`beats.ts`) into a docked text panel. Nothing ties a line to a sprite. |
| Timing | Story events and state deltas wait until the narration has played (`narrationPlaying` / `pendingDeltas`). Nothing is choreographed within a turn. |

## Options for the scene renderer

| | **A. React DOM + Web Animations** (what we have, extended) | **B. PixiJS v8 + @pixi/react v8** | **C. Phaser 4** |
|---|---|---|---|
| What it is | Keep the `<div>` sprites; use the Web Animations API for paths, hops and shakes. | A 2D WebGL renderer. @pixi/react v8 is built for React 19 (our version), so the scene is declared in JSX. | A full game framework: scenes, tilemaps (Tiled), tweens, frame animations, cameras, input. Phaser 4 went stable in April 2026. |
| Sprite frame animation | `background-position` with `steps()`; workable for a few frames. | `AnimatedSprite`, sheets and atlases. | Built in, including sheet and atlas loading. |
| Tweens, shake, squash | WAAPI keyframes; fine for a handful of sprites. | Animate properties per frame (ticker) or with a tween library. | Built-in tweens and camera shake and flash. |
| Lighting and mood | CSS filters (what the scene light uses now). | Filters, blend modes and per-sprite tint: real dusk and firelight, a flicker on the hearth. | Similar, plus lights in WebGL. |
| Fit with React UI | Native. | Good: same tree, React state drives the canvas. | It owns its own loop and scene graph; React talks to it through events. Two state worlds to keep in sync. |
| Speech bubbles | HTML, positioned directly. | HTML overlay placed from sprite coordinates (crisp text, selectable, accessible). | Canvas text, or the same HTML overlay with coordinate bridging. |
| Cost to adopt | Lowest. | Medium: rewrite `WorldMap` as a Pixi stage and keep the camera and placement logic. | Highest: a second app inside the app. Pays off only if the game becomes much more interactive. |
| Ceiling | Tens of sprites, simple effects. Gets hard to keep in time as choreography grows. | Hundreds of sprites, particles (ash, sparks), filters. Plenty for this design. | Highest, if we ever add direct control or combat on the map. |

**Why B:** movement is directed by the story, not by the player, so we don't need Phaser's input,
physics or scene management. We do want a real frame loop, frame animation, particles and filters, which
pushes past A. B keeps one React app and one source of state. Choose C instead only if the design moves
toward the player controlling a character on the map.

## The gaps (platform first)

### 1. Positions within a room: blocking marks (server + campaign format)
Characters can only walk somewhere the server can name. Today that's a room.
- **Proposal:** points of interest become *marks* the story addresses ("Bess walks to the stone well",
  "Dunstan stands by the fence post"). They already exist per room with tile coordinates in the art. The
  server needs each character's current mark, and the game master needs a `move_to` beat naming one.
- **The art ties marks to tiles;** the existing map test is extended so every mark has a tile.
- **Size:** M. Touches the world state (#109 area) and the campaign format.

### 2. A performance script instead of a prose block (game master contract)
To come out of a character's mouth, a line has to be handed in as that character's line.
- **Proposal:** the `end_turn` tool takes an ordered `script` of beats, replacing the single
  `narration` string. A beat is one of:
  - narration
  - `say` (an actor and a line)
  - `move_to` (an actor and a mark)
  - `emote` (an actor and one of a fixed set: `shake`, `hop`, `bow`, `startle`, `cry`, `laugh`)
  - `face`, `enter` or `exit`
- **Validation:** code checks every beat, as it does world events now: the actor is in the scene, the
  mark exists, the emote is known. World events and beats stay as they are.
- **Streaming:** the parser already decodes `narration` out of partial JSON. It can play each beat as
  soon as its JSON object closes, so the first lines still arrive within about a second.
- **Authored dialogue** (Yarn) gets the same beats through commands (`<<move bess stone_well>>`,
  `<<emote ros hop>>`), so authored and improvised scenes play the same way.
- **Size:** M. The biggest change, and the one that makes characters speak.

### 3. A timeline player (client)
- One queue per turn: beats play in order, each waiting for its animation (walk time, bubble read
  time). The player can tap to skip ahead. World changes apply when their beat plays. This generalises
  today's `pendingDeltas`.
- **Bubbles:** short `say` lines go over the speaker, longer ones go to the docked panel as now. Every
  line also lands in the log. Bubbles avoid the sign and the panel and flip below when there's no room
  above.
- **Size:** M.

### 4. Paths and facing (client)
- A walkable grid from the art's object layer (26×30 tiles is tiny: breadth-first search is enough; no
  library needed). Moves follow the path tile by tile, facing the direction of travel.
- Walking between rooms is a path across the village map. Interiors have their own grid.
- **Size:** S.

### 5. Animated character art (assets, licence, hosting)
- **What works with today's art:** procedural motion on static sprites. Hop while walking, a bob when
  idle, squash and stretch on landing, shake, a startle jump, tint flashes. Many beloved pixel games
  animate exactly like this. Size S.
- **What true walk cycles need:** a new character set with 4-direction walk frames and emotion frames.
  - **Time Fantasy** (#148's candidate) has both. The RPG Maker store edition is **licensed for RPG
    Maker only**. The itch.io edition is sold by the artist; the terms found so far are the artist's
    comments (commercial use fine, no credit needed, no redistribution), so get them confirmed in
    writing before buying.
  - No CC0 top-down set with walk cycles and emotes that matches the Kenney style turned up. Single CC0
    characters with walk cycles exist (OpenGameArt). LPC sprites have full cycles but are CC-BY-SA/GPL.
- **The repo is public,** so paid art can't be committed. It needs private hosting: a private S3
  prefix the deploy pulls from at build time, or a separate private repo. That's an infra change
  (Terraform, for the terraform session).
- **Size:** S to M for the pipeline, plus the art decision (Rob).

### 6. Pacing and latency
- A game master turn takes 5–8s today. With streamed beats, the first line plays within about a second
  and the rest follow as they're written. Walks and bubbles take real time, so a turn will often feel
  *longer* than its latency, which is what we want, but the player must be able to skip.
- **Size:** none on top of 2 and 3; a tuning pass afterwards.

### 7. Testing
- Unit tests for beat validation (server) and the timeline player (client, fake timers).
- Visual checks through the existing Playwright harness and the `playtest views` export, extended with
  a per-beat export.
- The playtest tool reports the script it played.

## Suggested order

| Step | What | Outcome |
|---|---|---|
| 0 | **Prototype (2 days):** a Pixi stage for one scene (Bess at the well), a hand-written script, a path walk, hop and shake, one bubble. | Confirms B against A on real code before we commit. |
| 1 | Marks: the server tracks each character's mark; the campaign format and art agree. | The story can say *where*. |
| 2 | Performance script in the game master contract, with validation and streaming beats. | Lines belong to characters. |
| 3 | Timeline player and bubbles, on the current DOM renderer if the prototype says that's enough, else Pixi. | Characters speak and move on screen. |
| 4 | Paths and facing. Procedural motion set. | Walking looks like walking. |
| 5 | Art decision and private asset hosting. Walk cycles if bought. | Optional polish. |

Steps 1–3 are platform work and pay off whatever the renderer. Steps 0 and 4 settle the library
question cheaply.

## Decisions for Rob

1. Renderer: PixiJS (recommended), or extend the DOM version. Phaser only if direct control is on the
   roadmap.
2. Art: procedural motion on the current CC0 art first (recommended), or buy a set with walk cycles now.
   The latter needs the artist's licence confirmed and private asset hosting.
3. Bubble rule: short lines over the speaker, long lines in the panel (recommended), or every line in a
   bubble.

## Sources

- Phaser v4 stable: [phaser.io/download/stable](https://phaser.io/download/stable), [phaser.io/download/phaser4](https://phaser.io/download/phaser4)
- PixiJS React v8 (React 19): [pixijs.com/blog/pixi-react-v8-live](https://pixijs.com/blog/pixi-react-v8-live), [README](https://cdn.jsdelivr.net/npm/@pixi/react@8.0.5/README.md)
- Time Fantasy, RPG Maker edition licence: [rpgmakerweb.com/a/graphics/time-fantasy](https://www.rpgmakerweb.com/a/graphics/time-fantasy)
- Time Fantasy, artist's terms in comments: [tf-mini-sprites comments](https://finalbossblues.itch.io/tf-mini-sprites/comments), [tf-rpg-charactersprites-1 comments](https://finalbossblues.itch.io/tf-rpg-charactersprites-1/comments?after=0)
- Kenney Tiny Dungeon (static frames): [kenney-assets.itch.io/tiny-dungeon](https://kenney-assets.itch.io/tiny-dungeon)
- CC0 walk cycle example: [opengameart.org/content/hero-character-sprite-sheet](https://opengameart.org/content/hero-character-sprite-sheet)
