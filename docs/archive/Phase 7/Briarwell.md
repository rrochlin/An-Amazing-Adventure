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

- **Endings:** `road_to_guild` ends the run (`ends_run`) with a single catch-all ending. Specific
  ending triggers are deferred (decision 15).
- **Combat:** the raid is narrated. A deterministic fight (decision 14, ~1% odds) comes later.
- **Morality tracking:** good and bad deeds are recorded as labels only; there is no running tally.

## Character

Briarwell is railed: `character_creation.preset` fixes a human fighter with the locked "child of
about ten" background, and the player only picks a name (default "Wren"). The server enforces this on
create and join. A rogue would suit the child better, but the character builder only supports
barbarian, fighter, and monk.

## Playtest 1 (2026-10-14)

Session `055f005e-2b41-4430-9bf3-cdf0200fd6db`. The narrator invented a village (Millhaven), an NPC,
and a missing-girl quest, and wrote the player as an adult. Cause: its system prompt carried no
campaign context, so authored narrator contracts never reached the model. Fixes: a scene brief in the
narrator prompt, `intro_text` on scenes, chosen options echoed in chat, objective updates shown
inline, `appears_in` to keep the bandits and the guild master out of early scenes, and removing the
square's "notice post", which the narrator had turned into a quest board.

## Playtest 2 (2026-10-15)

Session `1d066cbc-8917-4bf2-ae70-e21001989072`. The narrator stayed in Briarwell and kept the player a
child. Problems found: the run opened on the rescue scene's intro (intros attached to the wrong nodes);
the bully deed never ticked off (objective `success_when` was never evaluated); authored text arrived
all at once, chat-style; and the raid began the moment the gift was given. Fixes: intros repaired,
objectives evaluated each turn (#79); a dialogue stage that types one beat at a time with title cards
and quest toasts; scenes end on the player's last choice, with a "Go to bed" choice before the raid.

## Authoring notes
- Give every `ai_scene` node an `intro_text`; it is the only thing that orients the player when the
  scene starts.
- Write spoken lines as `Speaker: line` so the stage shows a name plate; keep narration separate.
- End a scene on a player choice ("Go to bed") rather than on a state change, so the player decides
  when the beat is over.
- Use `appears_in` for characters who should not exist yet from the narrator's point of view.
- Keep room descriptions free of props that invite new plot (notice boards, rumour mills).

- Dialogue is written as `.yarn` source in `server/campaigns/briarwell/dialogue/` and compiled with
  `go run ./tools/yarnc` (from `server/`), which writes the `.yarnc` and `.csv` beside it. Commit all
  three; a test fails if the compiled files are stale.
- The compiler supports a subset of Yarn Spinner 2: lines (tag them `#line:<id>` for stable IDs),
  nested `->` options, `<<jump>>`, `<<stop>>`, and game commands such as `<<set_label key value>>`.
  Variables, `<<if>>`, and `{interpolation}` are rejected. See `server/internal/yarnc`.
- Several choices in one node work: the runtime saves the choices made so far and replays them.
- `internal/campaigns/briarwell_test.go` plays the whole campaign through the runtime. Update it when
  the story graph changes.
