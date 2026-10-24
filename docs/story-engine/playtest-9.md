# Playtest 9: staging (2026-10-24)

Two Briarwell runs through the real client (`tools/localserver` + Vite + `e2e/play-driver.mjs`), the
first all the way to the ending, the second to the first scene change, checking the staging from #182:
[[game-master]] cues, marks, walking and emotes.

## What worked

- The game master staged unprompted from the first turn: Bess walked from the well to the market
  benches on the paragraph that showed it, then cried and shook. Later turns cued the player too.
- Marks survived a reload: Bess stood at the benches, the player at the garden gate after the raid.
- Dunstan walked into the long grass on his line, then walked out toward the square when the turn's
  world changes landed.

## Found and fixed

| Finding | Platform gap | Fix |
|---|---|---|
| The raid's cues never played | When a turn ends in a new scene, the authored text is a second message; the client hung the turn's events on it | The delta says when authored dialogue followed (`authored`, `scene_events`); each message keeps its own events, as the saved history already did |
| Dunstan's mark landed in the square | Staging applied after the world events had already moved him | Staging applies first, inside the room |
| "Little Bess arrives." for a girl who never left | `move_character` to the room the character is in wasn't caught | Validation drops it, like `move_player` |
| Narration seated the player, but nobody walked them | The brief asked for cues for "someone" | It names the player, and asks for the paragraph that shows the move |
| Speech came as `Bess: "…"` | Quotes around a line the name plate already attributes | The brief asks for none; the stage drops quotes around a whole spoken line |
| Bess shuffled round the well, or pushed Dunstan off his tile | Mark placement ignored where people already stood | Someone beside their mark stays; someone arriving takes a free tile |
| Ros "already gone" while she stood in the room | Nothing told the game master that who's listed is present | The brief says so: they can leave this turn, never already gone |
| Arrow keys typed into the text box | Enter focuses it, but nothing gave the keys back | Esc does, in free roam |

## Still open

- The game master still tries to hand out coins (dropped by validation every time).
