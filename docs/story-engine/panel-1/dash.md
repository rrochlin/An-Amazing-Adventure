# Dash's run: Briarwell (any%)

Final: about 22 actions, **6 typed turns**, ending screen reached. Roughly 8 minutes of wall-clock time, and about 2 of those were spent waiting on the AI.

## 1. Verdict
**6/10.** The free-text "talk your way through it" layer is great. I double-dipped a bully and a blackmail in one sentence and the game *paid out*. Then the back half takes the controller away. The whole fire night is one typed turn and a binary choice where both answers lead to the same place. I'd do one more run to route-test, but there isn't enough game here yet to grind.

## 2. Best moments
- **Blackmailing Hob, turn 1 at the mill.** "Pay me two coins and I won't tell." Hob haggled me down to **one** ("That's what a child's silence is worth"). When I pushed for two, he stood up to his full height and called my bluff. That's a system with a spine. It rewards the play, but it won't let you print money.
- **The combo turn in the Meadow.** In one line I lied to Dunstan (said the miller was hunting a flour thief) *and* grabbed Bess's doll while he was gone. The narration handled both. His "voice goes up half a note" in front of the little kids was a good beat. The coin counter hit 2 and I was done with the errand phase in **4 typed turns**.
- **Cottage-to-mill routing.** The cottage exits east, straight into the square and then the mill, and walking is instant. Loading between areas felt like zero.
- **The coin-jar theft got blocked in character.** Father's line ("A coin you didn't earn is just a thing you took") was a fair wall, not a "you can't do that" error.

## 3. Worst moments / frustrations
- **I got yanked to Garrick's stall the instant coin #2 dropped.** I was standing in the Meadow holding Bess's doll. My plan was to return it for a third coin, or gift *the doll* to Ros. Instead the game warped me into a shop menu. The "Left unfinished: Bess's Lost Doll" toast actually fired *before* the narration showed me picking the doll up. Expected: let me keep roaming until I walk to the stall myself, or at least ask "go to Garrick now?"
- **The doll does nothing.** I'm carrying a quest item into the shop, and there's no option to give it to Ros or trade it with Garrick. In a game whose whole pitch is "type anything", a forced 3-button menu hurts most at exactly the point I've found something clever.
- **The fire night is one turn.** "Survive the night: hide, fight, or search" sounds like the actual level. I typed one action (follow Mother out the back) and was in the ashes with Holloway. No danger, no bandit, no roaming the burning village. I wanted at least 2–3 turns with real stakes, and a chance to *find* someone.
- **"No! I have to find my family!" is a fake choice.** Holloway says one line and then I'm on his horse anyway. Fine for a prologue, but don't offer a refusal you won't honor. Make it a 1-turn detour, or just cut the button.
- **The gift payoff is generic.** I picked the hand-carved bird and Ros "squeals and throws her arms around you". It would have been the same line for sweets. Three choices with one outcome means a speedrunner just mashes the top one.

## 4. Bugs, exploits and softlocks
1. **The map doesn't follow the forced warp to Market Lane.** The header said MARKET LANE / Old Garrick's Stall, but the canvas still drew the Meadow (fence, flowers, no Garrick, no stall), and the scene reported `room=meadow`. Repro: earn the 2nd coin in the Meadow and the auto-transition to the stall fires. The map caught up only after I picked a gift.
2. **A raw debug string shows in the input bar.** Whenever choices are up, the input area literally reads `dialogue_choice_required`. It showed at the stall, the bedtime choice, and Holloway.
3. **Stale objective toasts.** In the burned-cottage scene the green toast read "New objective: Give Ros her birthday gift." Earlier, "Choose a gift for Ros" lingered on the gift-giving screen. Toasts seem to show the *previous* objective one scene late.
4. **"Left unfinished: Bess's Lost Doll" fires too early.** It appeared while the narration was still mid-turn, before the doll pickup was shown. That's a spoiler and also wrong: I *had* the doll.
5. **Typing during a choice hangs.** When choices were up I tried to type ("I give Ros the rag doll instead..."). The send never resolved and my tool waited the full 120s. The game didn't crash, and the choices were still there afterwards. But it should either refuse instantly or not take keystrokes at all.
6. **Walking was blocked by an NPC with no feedback.** After the coin-jar turn, Father stood at the tile east of me and holding Right did nothing, with no bump animation or nudge. Clicking a tile further away pathed around him fine. Not a softlock, but it feels like input lag.
7. **Exploit (intended?): one sentence can clear two quests.** The Dunstan lie plus the doll grab in a single turn pays the Dunstan coin. Blackmail plus bully means 2 coins in 2 typed turns without ever finishing Bess. Fastest route I found: **Cottage → East → Mill (blackmail) → West → South Meadow (scare Dunstan)**. Talking to Bess is optional. The doll stays in your inventory forever.
8. Minor: the cottage in the "ashes" scene is just the same tiles desaturated. The roof is intact and the gate isn't hanging, while the side panel says "Blackened beams and grey ash".

## 5. Pacing and responsiveness
Typed-turn response times:

| Turn | Time |
|---|---|
| Steal coin jar | 21s |
| Blackmail Hob | 18s |
| Push for 2 coins | 13s |
| Ask Bess | 19s |
| Dunstan + doll combo | 17s |
| Flee with family | 25s |

- **Average about 19s, worst 25s.** For a speedrunner that's the killer: every typed turn costs more than walking the entire map. Stream the first sentence faster, or show *something* within 2–3s.
- **Scripted beats and choices are fast,** about 4s per pick. Enter-to-advance is fine. The first Enter completes the typewriter line and the second advances, which is standard.
- **Narration volume:** each AI turn is 4–6 text boxes. I skipped most of them. A "skip to end of this turn" key (hold Enter / Esc to dump everything) would save a lot of taps.
- **The intro is short and skippable.** Good. Starting a new game took about 14s to load into the cottage.
- **Walking feels instant and area transitions are seamless.** Click-to-move pathing works and is the better control. Arrow keys sometimes do nothing until you press Esc out of the text box.

## 6. What would make me play more (ranked)
1. **Cut AI latency, or show partial text immediately.** About 19s per turn is the single biggest drag.
2. **Don't auto-warp on objective completion.** Let me walk to Garrick, and let leftover items (the doll) matter. Gift the doll, trade it, or return it for an extra coin and a better gift.
3. **Make the fire night a real level:** 3–5 turns, actual threats, and outcomes that differ for hide, fight, and search, plus some payoff for what you did during the day (Dunstan or Hob showing up, the miller's barn secret mattering).
4. **Make choices matter or remove them:** the refusal to Holloway, which gift you buy, "Do I have to?" with Father. Each should change at least one later line.
5. **Fix the UI bugs:** the map/room desync on forced warps, the `dialogue_choice_required` text, stale objective toasts, and early "left unfinished" alerts.
6. **Speedrun hooks:** an in-game timer, a turn counter on the ending screen, and an ending summary of which deeds you did (kind or cruel). That alone would make me replay for a 5-turn route.

### Second-run plan
- Try to get 2 coins from Hob alone: blackmail, then sell him a cover story.
- Try paying Garrick with the doll, or haggling for two gifts.
- On the fire night, type "fight", and try to run to the mill or meadow to see if free roam exists at all.
- Pick "Take his hand" to check whether the ending is identical.
- Try talking Dunstan into *paying* me, to see if coins can be farmed past 2.
