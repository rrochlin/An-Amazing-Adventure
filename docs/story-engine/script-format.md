---
tags:
  - story-engine
  - campaign-format
---

# Campaign script format

> [!summary]
> How a campaign is written for the story engine (#93): **character sheets** for the cast and a
> **scene script** on every story node. It follows the framing decision (blend 3, `framing.md`):
> authored like a screenplay, run like a table. Scenes are **controlled** or **bounded**, and each
> character acts as an **actor** or a **player**. The format extends `campaign.json`; everything
> here is optional, so older campaigns still load.

## Character sheets (`actors.<id>`)

| Field | Meaning |
|---|---|
| `name`, `kind`, `goal` | As before. `goal` is the character's standing want. |
| `description` | Who they are, including what they are to the player ("The player's father."). |
| `backstory` | Their history, for whoever plays them. |
| `motivations` | What drives them. |
| `relationships` | Actor ID, or `"player"`, → how they see that character. |
| `knows` | Things they know, secrets included. Agents only get what they need (need-to-know). |
| `default_tier` | `actor` (default) or `player`: how much freedom the agent playing them has. |
| `start_location_id` | The room they start in, as a real character in the world. |
| `appears_in` | Scenes they can appear in. A scene can't cast them otherwise. |

## Scene scripts (`story_nodes.<id>`)

| Field | Meaning |
|---|---|
| `control` | `controlled`: the scene unfolds as written. `bounded`: it improvises inside its boundaries. Defaults to `controlled` for `yarn_dialogue` scenes and `bounded` otherwise. |
| `stage_directions` | `focus`, `setting`, `must_mention`, `must_happen` (in order), `must_not_reveal`, `boundaries`, `tone`. Replaces `narrator_contract`; the old key is still read. |
| `cast` | Who may act in this scene: `actor_id`, `tier` (overrides the default), `part` (what they're doing here), `wants` (what they want from this scene), `knows` (scene-specific knowledge). Characters not in the cast don't act. |
| `beats` | The story conditions the scene can settle, with **meanings written out**. Label beats list `outcomes` (`value` and `meaning`); flag beats have a `meaning`. |

### Rules the loader enforces

- A controlled scene needs exact lines (a Yarn dialogue) or `stage_directions.must_happen`.
- Cast members must exist, appear only once, have a valid tier, and be allowed by their `appears_in`.
- Every beat has a description, and every outcome a value and a meaning.
- Once a scene has beats, every label or flag its completion or failure rules read must be a beat,
  and every label value they test must be one of its outcomes. So every outcome the story depends on
  has a meaning.
- Relationships name real actors or `"player"`; tiers are `actor` or `player`.

### What reads it today

- **The resolver** gets each beat's description and meanings in its allowed conditions. That stops
  guesses like marking Bess's doll "returned" when the player only learned where it was (session
  b31d479d).
- **The narrator brief** states the control level, setting, must-happen events, boundaries, and each
  cast member's part and wants.
- The game master (#99) and character agents (#103) will be built on the same fields.

## Example: a bounded scene

The village deeds: the player roams, and anything goes inside the boundaries.

```json
"village_deeds": {
  "mode": "ai_scene",
  "control": "bounded",
  "title": "Good Turns and Bad",
  "stage_directions": {
    "focus": "small-village moral choices, each worth a coin",
    "boundaries": ["Stay in Briarwell", "No one is seriously hurt", "Each deed pays exactly one coin"],
    "must_not_reveal": ["the bandits", "the coming raid"],
    "tone": ["villagers react honestly to kindness and cruelty"]
  },
  "cast": [
    { "actor_id": "bully", "tier": "player", "part": "Bullying the smaller children in the meadow",
      "wants": "To look tough, and to keep his coin" },
    { "actor_id": "bess", "part": "Crying by the well over her lost doll" },
    { "actor_id": "miller", "tier": "player", "part": "Slipping behind the barn",
      "wants": "To keep his secret, by paying if he must" }
  ],
  "beats": [
    { "key": "deed_doll", "kind": "label", "description": "Bess's lost doll",
      "outcomes": [
        { "value": "returned", "meaning": "The player gives Bess her doll back. Only learning where it is doesn't count." },
        { "value": "kept", "meaning": "The player finds the doll and keeps or sells it." }
      ] }
  ]
}
```

## Example: a controlled scene

The bandit raid: the game master performs it, but these events happen as written. (The birthday is
controlled through its Yarn dialogue instead, which gives exact lines.)

```json
"bandit_raid": {
  "mode": "ai_scene",
  "control": "controlled",
  "title": "Briarwell Burns",
  "stage_directions": {
    "setting": "Night. The cottage fills with smoke; orange light through the shutters.",
    "must_happen": [
      "Father holds the door with his woodaxe and tells Mother to take Ros out the back",
      "The door splinters: Red Corwin's bandits are in Briarwell",
      "The player chooses: hide, fight, or search for Mother and Ros"
    ],
    "tone": ["terror through a child's eyes", "nothing heroic yet"]
  },
  "cast": [
    { "actor_id": "father", "part": "Holding the door" },
    { "actor_id": "bandit_leader", "part": "Leading the raid" }
  ]
}
```
