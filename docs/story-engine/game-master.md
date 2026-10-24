---
tags:
  - story-engine
  - game-master
---

# Game master contract

> [!summary]
> The game master (#99) is one agent that runs a turn: it narrates, decides what changed, and settles
> the scene's beats, in a single model call. It replaces the narrator, the engineer and the resolver
> (Phase 7 decision 13). This note is its contract (#100): what it is told, what it returns, what a
> scene lets it do, and how it handles the player going off the script. The code is
> `server/internal/story/gm`; the pipeline that calls the model is #101.

## What it is told (`gm.Brief`)

One system prompt per turn, built from the script and the world:

1. **Role and rules.** Narrate what the player's action achieves this turn, where they are; stop at
   the next decision point; no option menus, no tallies; then hand in the decision.
2. **The scene's control level and its off-script policy** (below).
3. **The campaign brief** (`campaigns.NarratorBrief`): the scene, its stage directions, the cast
   (who exists and where, so no one is invented), the places, the objectives.
4. **The world right now** (`worldSlice`): the player and what they carry; their room, its exits,
   points of interest and items; who is here. Characters elsewhere are not described: need to know.
5. **How to hand in the decision:** the world changes this scene allows, and the beats it can settle,
   each with its written meaning.

Plus the narrator context (prior turns) and the player's input, as messages.

## What it returns

The prose, streamed as it is written, then one tool call, `end_turn`, carrying a `Decision`:

```json
{
  "world_events": [ { "type": "move_player", "input": { "room_name": "Meadow" } } ],
  "beats": { "deed_bully": "stopped" },
  "staging": [
    { "line": 0, "who": "player", "move_to": "old_fence_post" },
    { "line": 2, "who": "Dunstan", "move_to": "long_grass" },
    { "line": 2, "who": "Dunstan", "emote": "shake" }
  ],
  "off_script": { "kind": "improvised", "reason": "the player climbed the fence post" }
}
```

- `world_events`: what the narration made happen, in order, named by the engineer tool that applies
  it (`campaigns.WorldEventTypes`). Applied through `ai.DispatchTool` after validation.
- `beats`: the scene's beats settled this turn (label → outcome, flag → true), only when the beat's
  written meaning has clearly happened. Converted by `gm.Resolution` into the campaign's condition
  resolution, which the existing orchestration (`OrchestrateTurn`) applies: objectives, their
  `on_success` actions, scene transitions.
- `staging`: how the scene is played on the stage, in order (#182). Each cue names who (a character
  here, by name, or `player`), the narration paragraph it plays on (from 0), and a **mark** to walk to
  (one of the room's points of interest) and/or an **emote** (`hop`, `shake`, `startle`, `bow`,
  `laugh`, `cry`). `gm.Validate` keeps only cues for people in the room, marks of that room and known
  emotes. `gm.ApplyStaging` stands people at their marks (kept per room on the character, so any move
  to another room leaves the mark behind) before the world events apply, and the cues go to the client
  as `cue` events on the narration message. The client plays each when its paragraph comes up:
  characters walk tile by tile to a free tile beside the mark, and reactions play once they stop.
  Spoken lines are their own paragraphs, as `Name: words`, so they come from that character.
- `off_script`: present when the player's action wasn't scripted, and how it was handled.

The tool's JSON schema (`gm.Schema`) is built per scene: the event `type` enum is the scene's allowlist
and the `beats` object has exactly the scene's beat keys with their outcomes as enums. The model is
constrained before validation runs.

## What a scene lets it do (`gm.AllowedWorldEvents`)

| Control level | Default world changes |
|---|---|
| controlled | `move_player`, `give_item_to_player`, `take_item_from_player` |
| bounded | the above plus `move_character`, `create_item`, `place_item_in_room`, `damage_character`, `heal_character`, `update_room` |
| no scene (procedural game) | everything |

`stage_directions.allowed_world_events` adds to the defaults. **`create_character` and
`set_character_alive` are never defaults:** who exists and who dies is authored. (Playtest 5: the
engineer killed Tomas because the narrator wrote it; under this contract the raid scene would have to
allow it.) Money is handled by the story, never by the game master.

`gm.Validate` keeps what the scene permits and reports what it dropped. Dropped items go into the
turn trace, so a game master that oversteps is visible and corrected by code rather than trusted.

## Off-script policy

| Scene | Unscripted action | Decision |
|---|---|---|
| controlled | Acknowledge it in a line, then play the scripted event | `steered` |
| controlled | Can't happen here at all | `refused` (said in the fiction) |
| bounded | Inside the boundaries | `improvised` |
| bounded | Would cross a boundary | `steered` back inside |
| bounded | Can't happen here | `refused` |

The plot never changes because of an unscripted action: beats are settled only by their meanings, and
transitions remain the script's.

## How it fits

- **Pipeline (#101):** `story.GameMaster` builds the brief and calls Bedrock once with
  `ConverseStream`, **forcing** the `end_turn` tool so every turn comes with a decision. The tool input
  holds the whole turn: `narration` first (decoded out of the partial JSON as it streams, so the prose
  still reaches the player live), then `world_events` (each event type with its tool's exact argument
  schema, from `ai.ToolInputSchema`), then `beats`. Code validates the decision, applies the world
  events, converts the beats and runs `OrchestrateTurn`. A campaign opts in with
  `"story_pipeline": "game_master"`; Briarwell does. The validated decision is in the turn trace
  (`decision`) and in `sessiondump`.
- **Evaluation (#95):** replay the reference turn set (`server/testdata/turns`) through both pipelines
  and compare. The set already holds the failures this contract is designed to prevent.
- **Later:** character agents (#103) plug in as proposals the game master folds into its turn;
  dice (#139) as a `request_check` world event the engine resolves before narration continues.

### Results on the reference set (2026-10-23)

`go run ./tools/replay -from testdata/turns -pipeline gamemaster`, 7 turns:

| | current | game master |
|---|---|---|
| Latency (average per turn) | 9.3–11.8s, 4 of 7 over the 10s budget | 5.0–8.3s, all within budget |
| Input tokens | ~10–12k (three calls) | ~4–6k (one call) |
| Errors | resolver JSON parse failure on 1 turn | none |

Each targeted failure is gone: one step per turn (turn 5 searches the meadow rather than finding,
returning and getting paid), Tomas stays alive and the player stays under the bed (turn 9), the doll
isn't marked kept for learning where it is, and no coins are invented.

The first version let the model write narration as text and then call the tool. On 3 of 7 turns it
never called the tool, and free-form `input` objects got the wrong keys (`move_character` without
`room_name`). Forcing the tool and giving each event its argument schema fixed both.
