---
tags:
  - phase-7
  - campaign
  - briarwell
---

# Briarwell

> [!summary]
> `server/campaigns/briarwell/` is the first real campaign vertical slice (decision 12 in
> `[[Vision and Design Decisions]]`). It follows the beats of the *Fable* opening with original
> names, so we can validate the runtime against a story whose shape we already know.

## Story graph

| # | Node | Mode | Player decides | Recorded as |
| --- | --- | --- | --- | --- |
| 1 | `morning_errand` | yarn_dialogue | Agree to Father's errand eagerly or reluctantly | label `attitude` |
| 2 | `village_deeds` | ai_scene | Any two of three deeds, each good or bad | labels `deed_doll`, `deed_miller`, `deed_bully` |
| 3 | `gift_stall` | hybrid | Sweets, ribbon, or carved bird | label `gift` |
| 4 | `birthday` | hybrid | Give Ros the gift or keep it | flag `gift_given` / `gift_kept` |
| 5 | `bandit_raid` | ai_scene | Hide, fight, or search for family | label `raid_response` → flag `stayed_hidden` / `wounded` / `saw_ros_taken` |
| 6 | `rescue` | yarn_dialogue | Go with Master Holloway willingly or refuse | label `guild_answer` |
| 7 | `road_to_guild` | ai_scene | (terminal) | |

Every path reaches the rescue. Choices change the flags and labels a run ends with, and those are
what future ending selection (decision 15) will read.

## Cast

Tomas (father), Elin (mother), Ros (sister), Old Garrick (trader), Little Bess, Hob the Miller,
Dunstan (bully), Red Corwin (bandit leader), Master Holloway (guild master).

## Deliberately not in this slice

- **Endings:** the runtime has no `endings[]` support yet (decision 15), so `road_to_guild` is a
  plain terminal node.
- **Combat:** the raid is narrated. A deterministic fight (decision 14, ~1% odds) comes later.
- **Morality tracking:** good and bad deeds are recorded as labels only; there is no running tally.

## Playtest 1 (2026-10-14)

Session `055f005e-2b41-4430-9bf3-cdf0200fd6db`. The narrator invented a village (Millhaven), an NPC,
and a missing-girl quest, and wrote the player as an adult. Cause: its system prompt carried no
campaign context, so authored narrator contracts never reached the model. Fixes: a scene brief in the
narrator prompt, `intro_text` on scenes, chosen options echoed in chat, objective updates shown
inline, `appears_in` to keep the bandits and the guild master out of early scenes, and removing the
square's "notice post", which the narrator had turned into a quest board.

## Authoring notes
- Give every `ai_scene` node an `intro_text`; it is the only thing that orients the player when the
  scene starts.
- Use `appears_in` for characters who should not exist yet from the narrator's point of view.
- Keep room descriptions free of props that invite new plot (notice boards, rumour mills).

- `.yarnc` files are Yarn bytecode in protobuf text format, written by hand (there is no compiler in
  the toolchain). Option labels point at 0-based instruction indices.
- String tables are two-column `id,text` CSV; quoted text uses standard CSV `""` escaping.
- `internal/campaigns/briarwell_test.go` plays the whole campaign through the runtime. Update it when
  the story graph changes.
