# Playtest 6: two Briarwell runs, played by Claude

> 2026-10-24. #165. Played through the real ws-chat and ws-game-action handlers with `tools/playtest`
> (in-memory session, real Bedrock, the game master pipeline). Two complete runs on opposite branches;
> each problem was fixed and the turn replayed before moving on.

## How

```bash
cd server
export AWS_PROFILE=aa-debug AWS_REGION=us-west-2
go run ./tools/playtest -dir DIR new
go run ./tools/playtest -dir DIR say "I head east to the Old Mill"
go run ./tools/playtest -dir DIR choose 1
go run ./tools/playtest -dir DIR rewind 4      # replay a turn after a fix
go run ./tools/playtest -dir DIR views         # client state per turn, for a UI harness
```

The report (`DIR/report.md`) shows, per turn, the frames sent, what streamed to the player, the log
entries, the game master's decision, dropped events, world changes and what the player can now see.
`DIR/traces/` holds the full turn traces.

| Run | Path |
|---|---|
| 1 | eager; miller's bribe, then tried to tell the wife too; found the doll; stopped Dunstan; wooden bird, given; fought in the raid; refused Holloway |
| 2 | reluctant; asked Mother for bread; returned the doll; joined Dunstan; honey sweets, kept; hid in the raid; took Holloway's hand |

Latency: 25 game master turns, mean 6.6s, max 10.1s (one over the 10s budget). Authored turns (choices)
are instant.

## Found and fixed

| Turn | What happened | Fix |
|---|---|---|
| 1·5 | After taking Hob's bribe, telling his wife rewrote `deed_miller` to `told_wife`, and the prose paid a second coin the state never gave. The game master couldn't see which beats were settled. | The brief lists settled beats as final and says what each open one pays. Validation drops re-settling. The schema offers only open beats. |
| 1·5 | On the retry, it refused by taking over the player character ("your hand hovers… you let it drop"). | Agency rule: play out what the player attempts; refusals come from the world. |
| 1·2, 1·4 | Narration ended on a menu ("You could…, or you could…") or a recap of open errands. | The ending rule is in the brief and in the `narration` field description. |
| 1·8 | `beats` came back as `""`; the decision half-parsed (world events applied, error logged). | `gm.ParseDecision` reads empty or quoted values and reports them. An unreadable decision applies nothing. |
| 1·15, 2·11 | Story rail moves only changed actor state: Red Corwin and Holloway never appeared anywhere, and the family stayed in the burned cottage. | Rail moves place the character (created on first arrival, off-stage for a step without a room). `@player` means wherever the player is. Briarwell's raid takes the family and bandits off-stage and brings Holloway to the player. |
| 2·5 | The doll lying in the meadow was created again when picked up; the player carried two. | Validation drops `create_item` for a thing already in the room or the pack. |
| 2·11 | Hiding in the raid: the game master damaged Tomas and wrote "Tomas lies still near the door" into the room. | The raid gets stage directions: nobody dies on screen, the family's fate stays unknown, a child never wins a fight. It also gets beats with meanings. |
| 2·11 | The rescue said ash was falling over a sunny, intact village. | Scene light: `mood` (dawn, dusk, night, fire, ashes) tints the map. Birthday dusk, raid fire, rescue ashes, road dawn. |
| — | No-op `move_player` to the room the player is in. | Dropped in validation. |

## Working as intended

- One beat, one payout: bribe, bully (stopped or joined), doll returned: each paid exactly one coin.
  When the game master tried to create coins itself, the code refused it.
- One step per turn: run 2 found the doll lying in the meadow and picked it up the next turn.
- Interiors switch at the right scenes; free text is refused while a choice is pending, and the client
  hides the input then anyway.

## Still open

- #166: the doll quest fails silently while the player holds the doll (story decision).
- #167: the single ending ignores the player's choices.
- #168: ws-game-action tests make real AWS calls.
- #169: campaign sessions drop the wizard's D&D character.
- The game master still gives a few unscripted characters names ("Tommy Weaver"); a rule now asks it not to.
- When asked to search for a doll it had already found, it narrated finding it again.
- Narration sometimes runs to 5–6 paragraphs, which pushed one turn past 10s.
- A thing handed over in passing (a heel of bread) isn't always recorded; the model tends to treat it as
  eaten, which is consistent enough.
- The ending plays while the map still shows the cottage.
