# Vex's playtest: Briarwell prologue

Run: port 9341, one full playthrough to "Begin a New Adventure". About 11 meaningful actions: 6 free-text "say" turns, 5 choice picks, plus walking. I played the bastard on purpose: I tried to pick Father's pocket, shook down a crying six-year-old, blackmailed the miller, double-crossed the bully, kept my sister's birthday present, and then tried to make amends during the raid.

## 1. Verdict

**6/10.** Line by line, the prose the AI GM writes is better than most hand-authored CRPG barks. It is specific, it has a voice, and it reacts to the actual words I typed. The game around that prose is a Fable-shaped railroad, though. Every deed collapses into "a coin", the climax lasts one input, and the ending forgets everything I did. I'd play the next chapter to see if the reactivity grows teeth. I wouldn't pay for this prologue as it stands.

## 2. Best moments

- **Hob the Miller's blackmail scene.** It's the best thing in the build. I held out my palm and said "Silence costs a coin. Two if you want me to forget his face too." Hob haggled me down and counter-threatened me with something I'd never set up: *"One. And that buys my name out of your mouth entirely, face and all... if I hear otherwise I'll tell your father you were scrumping apples from the Widow Cray's tree again."* That's Disco-tier NPC pushback. He has leverage, a history with me, and "a thread of something that might be shame." The coin is "warm from sitting against him." Great detail.
- **Shaking down Bess.** I demanded coin up front for her doll. The GM didn't flinch and didn't moralise. It just made me feel it: *"the particular misery of a child who has already guessed the answer and is waiting to hear it anyway."* Then *"I made her a dress out of a bit of Mum's old apron."* Cruel choices with no lecture attached is exactly how it should be done.
- **Father catching me at his coat.** *"A floorboard creaks... Tomas straightens without turning around. 'Whatever you're looking for over there, you won't find it. A coin has to be earned, Wren.'"* It's in character and it shut me down without a game-over.
- **Real cross-turn memory.** I kept the wooden bird at supper. During the raid I typed that I press "the wooden bird I kept" into Ros's hands. The GM knew it was the carved bird, and it paid the moment off: *"surprise, and then a smile that breaks your heart a little." / ROS: "It's a bird."* That's the one time the game felt like it was remembering me.
- **Dunstan reads the room.** When I name-dropped his mother, *"his eyes slide sideways, just for a moment, toward the thick tangle of grass near the old fence post."* It's a nice non-verbal tell that handed me the information without a quest marker.
- **Threads get tied together.** Bess's lost doll was linked to Dunstan's bullying without my prompting ("I ran and I must have dropped her"). It made the village feel connected.

## 3. Worst moments and frustrations

- **"Kind or cruel, each deed pays a coin."** The objective text tells you up front that morality is cosmetic. So it proved: blackmail, a double-cross and a doll I never returned all landed on the same "two coins". Nobody in the village reacts, Father never asks where the money came from, and the ending never mentions any of it. In BG3, shaking down a child gets you a disapproval tick at minimum. Here the system flattens the AI's good writing back into a binary counter.
- **The game took my doll arc away from me.** I had Bess's doll stuffed in my shirt, which set up a ransom, a return, or a cruel keep. One turn later the game threw up *"Left unfinished: Bess's Lost Doll"* and teleported me to Market Lane because I'd hit two coins. I never got to resolve the thing I was most interested in. The prose even dangled it: *"somewhere across the square Little Bess is still waiting by the well."* Then the game cut away. I expected the coin threshold to *unlock* the market, not force me there.
- **The climax is a single input.** The objective was "Survive the night: hide, fight, or search for Mother and Ros." I typed one action, and the next card was *"When the shouting finally stops, the silence is worse."* I never saw a bandit, never got to choose hide or fight, and Father's fate went unmentioned. For a story whose whole premise is "things go wrong that night", the night is three text cards long.
- **Refusal choices are fake.** "Do I have to?" leads to Father saying go anyway. "No! I have to find my family!" leads to *"He lifts you onto his horse anyway, gently."* One illusory choice is a stylistic beat. Two in a ten-turn prologue is a pattern, and CRPG players notice.
- **Keeping the gift was a weak beat.** Choosing "Keep the gift for yourself" got one line, *"Ros's face falls,"* and then Father went straight to the smoke at the window. Mother and Father don't react at all. That was the most selfish thing I did, and it was right in front of the whole family.
- **Narrator location drift.** I was outside the cottage when I went for Father's coat, and the GM had him "bent over the hearthstone" and me reaching "the peg by the front door". At the mill, my "I pretend to leave" was read as leaving Bess back in the square, so it narrated me trotting across town to the place I was already standing.
- **No free text at forced choices.** At Garrick's stall I tried to trade Bess's doll. The input was ignored (the command hung about two minutes) and only the three gift buttons worked. The pitch is "an AI GM that reacts to whatever you type", so going locked-menu at the most interesting barter point undercuts it.
- **"Gossiping neighbours", "a pair of younger children" and "Red Corwin's bandits" are all prose-only.** None of them is on screen. The village square is empty except for Bess. The bullied kids in the meadow don't exist as sprites. The raid has no attackers.

## 4. Bugs and glitches

1. **Stale objective toasts.** After the gift choice, a green "New objective: Choose a gift for Ros at Old Garrick's stall." appeared during supper and stayed through the raid, covering the "Survive the night" sign. After I handed Ros the bird mid-raid, "New objective: Give Ros her birthday gift." appeared and stayed up through the Holloway ending. The toasts lag one or two objectives behind and point at tasks already finished.
   *Repro:* buy any gift, pick "Keep the gift for yourself", then read through to the raid.
2. **Floating name label left behind.** Bess's "LITTLE BESS" nameplate stayed at her old spot in the square after her sprite moved next to the well. It's also visible from the Old Mill screen (left edge) with no sprite under it.
   *Repro:* talk to Bess and say something that makes her move, then walk east to the mill.
3. **No coin feedback.** Hob "slaps a single copper coin into your hand" in prose, but nothing confirmed it. There was no toast and no objective progress, and I couldn't tell whether it counted until I was suddenly teleported to the market with "two coins".
   *Repro:* blackmail Hob at the Old Mill.
4. **Cutscene lines truncated when advancing.** Pressing Enter while a line is still typing skips to the next one. I lost "Two coins clink in your pocket as y…" and "The sun is low and gold by the time…" entirely. Enter should complete the current line first, then advance.
5. **Walking blocked silently by NPCs.** Outside the cottage, Father's sprite stood one tile east of me. Holding Right did nothing, with no bump or feedback, until I walked around him.
6. **Talk staging misses the target.** "talk bully" walked me to the fence post, several tiles from Dunstan. The conversation ran with us far apart.
7. **The burned cottage isn't burned.** After the raid the sidebar says "Blackened beams and grey ash where the cottage stood". The sprite is the same intact cottage with a grey-green tint.
8. **"Reconnected to the adventure" toast on a brand-new game** (first frame after `new`).
9. **Inconsistent dialogue formatting.** Hob's lines render inline as "Hob the Miller: You didn't see anything…" in narrator style. Every other NPC gets a proper speaker header (FATHER, BESS, DUNSTAN).

## 5. Pacing and responsiveness

- **AI turns:** 14–23 seconds each. That's tolerable because what comes back is good, but there's no "the GM is thinking" flavour to cover the wait. One free-text input sent during a choice prompt hung for about two minutes with no response.
- **Reading:** Responses arrive as four to six click-through cards. That's fine, but the typewriter combined with Enter-skips-the-line (bug 4) means fast readers lose text.
- **Walking:** Area transitions are quick and the screen layout is readable. It is mostly dead time, though. There's nothing to examine, the diamond interactables never told me anything, and the square and meadow have no ambient life.
- **Overall shape:** The morning drags slightly with long setup cards. The middle is the best part. The back third (market, supper, raid, ending) runs on rails and is over in about two minutes of clicking. The whole thing took about 11 actions. It's front-loaded in agency and back-loaded in cutscene, which is the wrong way round for a tragedy.

## 6. What would make me play more (most important first)

1. **Make the ending remember me.** Holloway, Ros, Father, Bess and Hob should reference what I actually did: the doll still in my shirt, the blackmail, the double-cross, the kept gift. One line of callback in the ash scene, e.g. Holloway finding Bess's doll on me, would sell the whole AI-GM premise. Right now the payoff is identical for a saint and a little monster.
2. **Give the raid real turns.** Let "hide / fight / search" be three to five actual turns with visible bandits, and let the outcome vary: who you saw, what you saved, a scar, a grudge. That's the emotional core, and it's currently a fade to black.
3. **Don't force-close open threads.** Reaching two coins should unlock the market, not teleport me there. Let me return or ransom the doll first. If you must cut, cut after a beat I chose.
4. **Track cruelty and kindness as separate state** that NPCs can see: a reputation, a disapproval, rumours spreading. "Each deed pays a coin" shouldn't be the whole ledger, and the objective text shouldn't announce that it is.
5. **Allow free text at forced choices**, or at least offer a "something else…" option. Bartering with Garrick was the most natural thing to try.
6. **Fix the toast and objective lag and the stray name label.** They make the UI feel untrustworthy at the exact moments when tone matters.
7. **Put the prose on the screen.** Spawn the bullied kids, a couple of neighbours, a bandit silhouette or two, and a burned cottage tile. When the text describes things the map doesn't show, it reads as two different games.
8. **Make refusals count for something.** If the answer is always "anyway", have the refusal change a later line, or don't offer it.
