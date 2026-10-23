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
  "off_script": { "kind": "improvised", "reason": "the player climbed the fence post" }
}
```

- `world_events`: what the narration made happen, in order, named by the engineer tool that applies
  it (`campaigns.WorldEventTypes`). Applied through `ai.DispatchTool` after validation.
- `beats`: the scene's beats settled this turn (label → outcome, flag → true), only when the beat's
  written meaning has clearly happened. Converted by `gm.Resolution` into the campaign's condition
  resolution, which the existing orchestration (`OrchestrateTurn`) applies: objectives, their
  `on_success` actions, scene transitions.
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

- **Pipeline (#101):** a `story.Pipeline` that builds the brief, calls Bedrock once with
  `ConverseStream` and the `end_turn` tool, streams the text deltas as narration, validates the
  decision, applies the world events, converts the beats and runs `OrchestrateTurn`. Enabled per
  campaign, with the current narrator-engineer-resolver pipeline as the default until it wins.
- **Evaluation (#95):** replay the reference turn set (`server/testdata/turns`) through both pipelines
  and compare. The set already holds the failures this contract is designed to prevent.
- **Later:** character agents (#103) plug in as proposals the game master folds into its turn;
  dice (#139) as a `request_check` world event the engine resolves before narration continues.
