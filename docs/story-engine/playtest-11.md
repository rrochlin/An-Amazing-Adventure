# Playtest 11: awkward inputs and the mechanics (2026-10-24)

A Briarwell run through the real client after the mechanics landed (#202), trying what a player does that
the script doesn't expect: punching Dunstan, ignoring him to take the doll, climbing out of the window
during the raid. Follows [[playtest-10]].

## What worked

- Every pack change showed over the player as it happened: "+ Bess's rag doll", the deed's "+ copper
  coin", the stall's "− 2 × copper coin" and "+ cornflower ribbon". Dunstan's punch showed "−1 ❤" over
  him.
- The game master played out the punch (1 damage, he storms off, the deed settles as stopped) and
  steered the escape attempt into hiding during the controlled raid, as its off-script policy says.
- Validation dropped every redundant move (the player and Dunstan "moved" to the square they were in)
  and every coin the game master tried to hand over itself.

## Found and fixed

| Finding | Platform gap | Fix |
|---|---|---|
| "I walk straight up to Dunstan" was staged as a walk to the long grass | Staging could only target marks | #203: a cue can walk someone `toward` a character |
| The player hid behind the barrel by the door, and the cue sent them to the hearth | An interior had only its room's marks (hearth, door, gate) | #204: scene maps have their own marks; the cottage gets nine |

## Seen, not changed

- A click on a choice within 400 ms of it appearing is ignored (the guard from playtest 7); the driver
  hit it, a person rarely would.
- Dice (#139) need a decision first: a second model call on turns with a roll, or a die rolled before
  the turn that the game master sees.
