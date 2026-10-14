---
tags:
  - phase-7
  - json
  - schema
---

# Campaign JSON Schema

> [!summary]
> This note describes the executable JSON shape for Phase 7 campaigns.

This is the schema for campaigns in general.
`[[Lich's Labyrinth]]` is the first concrete campaign that should implement it.

Related notes:

- `[[Campaign Data Model]]`
- `[[Campaign Save State]]`
- `[[Lich's Labyrinth]]`
- `[[Selected Stack]]`

## Goals

- keep campaign content outside compiled code
- make authoring reviewable in git
- allow validation before runtime
- support strict authored progression with AI narration on top

## Confirmed Phase 7 Direction

- JSON should hold most authored campaign data
- Go should evaluate JSON-defined conditions and effects
- entrance/timing choices should happen in the opening scene, not the create screen
- companions should be authored inside campaign JSON
- the campaign map should be defined as a full room graph in JSON
- player campaign choices should be expressed through freeform chat and resolved into structured values

## Status

> [!success]
> The schema direction is settled at a high level.
>
> Still to finalize:
> - the exact author-facing condition syntax
> - the exact command/event bridge between Yarn and campaign runtime
> - how much variable mirroring happens between Yarn and campaign state

## Recommended File Shape

```json
{
  "id": "lichs-labyrinth",
  "version": "1",
  "title": "Lich's Labyrinth",
  "premise": "You are part of a mercenary band hired to ambush a rival adventuring party.",
  "description": "A one-shot dungeon pursuit campaign.",
  "tone": "dark-fantasy",
  "character_creation": {
    "use_standard_creator": true,
    "allowed_classes": ["fighter", "barbarian", "monk"],
    "allowed_races": ["human", "half-orc", "dwarf"],
    "allowed_subraces": ["hill-dwarf", "mountain-dwarf"],
    "locked_background_prompt": "You are a mercenary on a dirty contract.",
    "theme_hint_mode": "campaign_default",
    "preference_mode": "campaign_default"
  },
  "player_role": {},
  "party": {},
  "map": {},
  "actors": {},
  "objectives": {},
  "dialogue_assets": {},
  "story_nodes": {},
  "start_node_id": "camp_briefing",
  "intro_sequence": {}
}
```

## Required Top-Level Fields

- `id`
- `version`
- `title`
- `premise`
- `character_creation`
- `party`
- `map`
- `actors`
- `dialogue_assets`
- `story_nodes`
- `start_node_id`

## Character Creation Section

Purpose: reuse the current D&D creator while enforcing story-specific boundaries.

Suggested shape:

```json
"character_creation": {
  "use_standard_creator": true,
  "allowed_classes": ["fighter", "barbarian", "monk"],
  "allowed_races": ["human", "half-orc", "dwarf"],
  "allowed_subraces": ["hill-dwarf", "mountain-dwarf"],
  "locked_background_prompt": "You are a mercenary on a morally suspect mission.",
  "theme_hint_mode": "disabled",
  "preference_mode": "disabled"
}
```

Suggested modes:

- `disabled`
- `campaign_default`
- `player_selectable`

## Party Section

```json
"party": {
  "max_human_players": 1,
  "starting_room_id": "camp",
  "companion_ids": ["doren", "amalia", "rowan"]
}
```

Companion definitions should also live in the same campaign package, either under `party` or `actors`.

## Map Section

Phase 7 should author the full room graph in JSON.

That means JSON should explicitly define:

- every room ID
- room connections
- entrances
- floor/chokepoint tags
- important observation / ambush spaces

```json
"map": {
  "rooms": {
    "camp": {
      "name": "Mercenary Camp",
      "floor": 0,
      "type": "briefing",
      "connections": {}
    }
  },
  "entrances": [
    {
      "id": "target_entrance",
      "name": "Target Entrance",
      "start_room_id": "f1_target_entry",
      "initial_flags": {
        "entry_target": true
      }
    }
  ]
}
```

## Actors Section

Use for rival parties, companions, special NPCs, and key threats.

```json
"actors": {
  "target_party": {
    "name": "Rival Adventuring Party",
    "kind": "party",
    "goal": "Delve, defeat the lich, and return alive.",
    "importance": "primary",
    "rail": {
      "steps": [
        {
          "id": "enter_f1",
          "location_id": "f1_target_entry"
        }
      ]
    }
  }
}
```

## Dialogue Assets Section

Use this to reference Yarn-backed dialogue content.

```json
"dialogue_assets": {
  "camp_briefing": {
    "format": "yarn",
    "program": "dialogue/camp-briefing.yarnc",
    "strings": "dialogue/camp-briefing.csv"
  }
}
```

## Story Nodes Section

This is the dramatic backbone.

Transitions should be deterministic once the current turn's authored conditions have been resolved.

Story nodes should also be able to declare how they are presented.

```json
"story_nodes": {
  "camp_briefing": {
    "mode": "hybrid",
    "dialogue_asset_id": "camp_briefing",
    "dialogue_start_node": "CampBriefing",
    "title": "Camp Briefing",
    "summary": "The mercenaries prepare for the mission.",
    "objective_text": "Understand the mission and prepare to choose your approach.",
    "completion_rules": [
      {
        "conditions": [
          { "kind": "flag", "key": "briefing_complete", "op": "==", "value": "true" }
        ],
        "next_node_id": "choose_entry_plan"
      }
    ],
    "narrator_contract": {
      "scene_focus": "mission briefing and party tension",
      "must_mention": ["the rival party", "the three entrances"],
      "must_not_reveal": ["the lich chamber details"]
    }
  }
}
```

Suggested node modes:

- `ai_scene`
- `yarn_dialogue`
- `hybrid`

Current recommendation:

- use `hybrid` for the camp briefing and other cinematic setup scenes
- use `yarn_dialogue` for highly authored conversations
- use `ai_scene` for freer exploration and reactive narration

## Intro Sequence Section

This replaces procedural world-gen flavor for authored campaigns.

It should also be where the player makes early campaign-specific setup choices like entrance and timing.

Those choices should be made conversationally in play, then resolved into structured labels/flags by the condition resolver.

```json
"intro_sequence": {
  "mode": "streamed_briefing",
  "opening_node_id": "camp_briefing",
  "opening_text": "Rain whispers over the canvas of your camp as steel glints in the firelight..."
}
```

Suggested follow-up structure:

```json
"opening_choices": {
  "entry_route": ["target_entrance", "novice_entrance", "hidden_entrance"],
  "timing_strategy": ["before_target", "after_target", "wait_for_return"]
}
```

## Validation Rules

The loader should reject a campaign if:

- `start_node_id` is missing
- a node points to an unknown node
- an actor rail references an unknown room
- a companion ID has no matching actor/template
- `allowed_classes` contains unsupported class IDs
- `allowed_races` contains unsupported race IDs

## Evaluation Rule

JSON should define:

- allowed condition keys
- legal transitions
- effects to apply when a transition fires

Go should evaluate those rules.

AI may assist by resolving which condition keys were satisfied during a turn, but it should never output the branch decision directly.

In Yarn-backed scenes, authored commands and variable changes may also contribute to condition fulfillment, but the campaign transition engine should still make the branch decision.

## Authoring Principle

The JSON should define structure and truth.

The AI should supply presentation and adaptation inside that structure.
