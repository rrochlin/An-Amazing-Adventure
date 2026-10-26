# Playtest panel 1: five gamers play Briarwell (2026-10-26)

Five simulated players, all gamers, each with their own browser and game against the real handlers and
game master (current main). They were asked to judge enjoyment, not accessibility. Their reports, in
their own voices, are in [panel-1/](panel-1/).

| Tester | Kind of player | Score |
|---|---|---|
| Vex | CRPG veteran (Baldur's Gate 3, Disco Elysium) | 6 |
| Dash | Speedrunner, min-maxer | 6 |
| Morgan | Tabletop D&D player and DM | 6 |
| Pip | Cozy indie (Stardew, Pokémon, Undertale) | 6 |
| Rook | Chaos sandbox (Skyrim, GTA) | 6.5 |

## What everyone liked

The writing. The game master says "yes, and", reacts to exactly what was typed, gives people voices and
calls back to earlier moments (Ros's ribbon in her hair during the fire, Hob bargaining a blackmail down
to one coin, "A bee drones past. Somewhere a chicken clucks." for a cast of Wish). The mood tints
(candlelight, ash) landed.

## Findings, by how many hit them

| # | Finding | Testers | Status |
|---|---|---|---|
| 1 | Choices don't matter: refusals send you anyway, Ros reacts the same to every gift, the ending ignores your deeds, cruelty has no consequence | 5 | Design: proposed below |
| 2 | The second coin whisks you to the stall and closes quests mid-way (the doll in your pocket) | 5 | Fixed: free roam until you walk to the stall |
| 3 | The raid is one typed turn | 5 | Design: proposed below |
| 4 | Name labels left floating where people had stood | 5 | Fixed |
| 5 | Objective toasts lag a scene behind and never go away | 5 | Fixed (the Snackbar never re-armed its timer) |
| 6 | The burned cottage looks intact | 5 | Open: needs ruined-cottage art |
| 7 | The narration puts people in the place who aren't on the map | 3 | Fixed in the brief |
| 8 | The raw code `dialogue_choice_required` shows | 2 | Fixed: codes read as sentences |
| 9 | No feedback for coins, no coin count | 2 | Partly: item pops exist (#202); a coin count is open |
| 10 | "Reconnected" on a new game; FATHER/TOMAS; Hob's lines not shown as speech; blocked by Father with no feedback | 1–2 | Fixed, except the bump feedback |

Two complaints came from the test harness, not the game: "Enter skips lines" (the harness pressed Enter
twice per step) and "~19 s turns" (the model answered in a median 5.9 s, 90th percentile 8.1 s over 54
turns; the rest was the harness polling while the narration streamed).

## Proposed next (design decisions)

- **Consequences (1).** Facts from the player's choices ("kept the gift", "joined Dunstan", "refused
  Holloway") recorded as state (#110), told to the game master in later scenes, and read by the ending
  (#167). Authored refusal choices should lead somewhere different, even if briefly.
- **The raid (3).** A sequence of beats the player lives through over several turns (wake, the door
  breaks, a choice to hide, fight or run, the family lost) instead of one turn and a cut.
- **The world on the map (6, 7).** Scene art that changes with the story (a ruined cottage after the
  raid), and unnamed extras the stage can show (the bullied children).
