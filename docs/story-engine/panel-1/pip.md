# Pip's playtest: Briarwell prologue (port 9344)

Screenshots are in `t4/shots/` under the panel directory.

## 1. Verdict

**Fun score: 6/10.** The writing has a real heart. Ros's ribbon wish, Bess's doll Marta, and Dunstan saying "Sorry. For — you know. Chasing you off." got me, and a few of those lines will stick with me. But the world under the words feels like a stage set. People teleport, name tags float over empty grass, the narrator describes things I can't see, and the game takes the wheel right when I want to wander. I'd play the next chapter for the story, but I'd be reading more than playing.

## 2. Best moments

- **The honey spoon.** I asked Mother for a crumb of cake. She said no ("not a crumb, not a lick, not so much as a sniff"), then gave me a drip of honey "for luck, and not a word to Ros", and Ros "makes an outraged noise from behind you, having seen the whole thing perfectly well." That's real cozy-game family warmth, and it came from something *I* chose to try.
- **Ros's wish.** I asked what she'd wish for. She went quiet, then said "A ribbon. A proper one, not the fraying kind." When I bought the cornflower ribbon later, it felt like *my* idea.
- **The ribbon callback in the fire.** "...her cornflower ribbon still tied in her hair from yesterday." That gutted me in the best way. The story remembered my gift.
- **The well.** "The circle of sky reflected far below is bright and small, like a coin dropped in reverse." I poked at a random prop and got a lovely line back. That's the Stardew "check everything" reward loop.
- **Dunstan's turn.** I didn't fight him. I asked why he's always alone, then offered to teach him a three-skip stone throw. He slowly softened ("Skipping stones is for little kids." / "But he hasn't said no.") until he apologised. Being kind to the bully actually *worked*, and it was paced like a real scene.
- **Garrick's carved fox.** "Took me three evenings by the fire... just mind the ears." It's a small thing, but it made the market feel lived in.
- **Visuals:** the meadow full of orange flowers (`look-1793051054.png`) is gorgeous. The warm candlelit tint for the birthday evening (`look-1793051424.png`) and the grey ash palette after the raid (`look-1793051558.png`) are great mood shifts. Speech bubbles above heads ("She'll wear that till it's rags.", `look-1793051394.png`) are charming.

## 3. Worst moments and frustrations

- **I got railroaded right after the second coin.** Bess paid me, and the game instantly whisked me up to Garrick, then to evening, then to the raid. I never visited the Old Mill. A toast just said "Left unfinished: The Miller's Secret." As an explorer, that hurt. I wanted to walk back through the village, show Dunstan my stone skipping, and peek behind Hob's barn. *Expected:* after earning two coins, let me roam freely until I choose to go to Garrick's. Let the evening start when I head home.
- **No feedback for earning coins.** Dunstan's deed paid a coin silently. I only found out when Bess's coin made "Two coins clink in your pocket". No "+1 coin" pop, no jingle, no counter. *Expected:* a floating "+ Copper coin" pop over my head and a coin counter somewhere.
- **The narrator describes a world I can't see.**
  - "A couple of neighbours pass by" in an empty square. The sidebar even says "gossiping neighbours".
  - "The three small children have crept back to the meadow's edge". There were no children on the map, and I'd heard about "a smaller child's sobbing" from the start but never saw one.
  - I said "I read the notice board" while standing next to a clearly drawn signpost. The narrator replied "There's no board here." That made me stop trusting my eyes.
  - Ros and Elin are described "by the hearth" and "up to the ceiling" while we're all standing outside in the garden.
- **The raid is all told, not played.** One typed action and the whole fire, escape, separation and aftermath happened in a single wall of narration. The room looked identical, just darker: no flames, no bandits, Mother and Ros still sitting calmly at the table (`look-1793051492.png`). It's a huge tonal swing from cozy to tragedy, and I only got to *read* it.
- **My choices at the end didn't matter.** I picked "No! I have to find my family!" Holloway answered, then "He lifts you onto his horse anyway." I'd rather have one small way to act on that refusal: call out, run to the window, anything.
- **The gift moment is too short.** After all the heartfelt AI writing, handing over the ribbon gets "It's perfect! You're the best!" and then a cut. Ros never mentions the ribbon or its colour. That should be the emotional peak before the fire.
- **The game doesn't remember things I did.** Garrick says "A birthday, is it?" even though I had just chatted with him about a blue ribbon for her.

## 4. Bugs and glitches

1. **Ghost name labels.** An NPC's name tag stays floating where they used to stand after they move or vanish.
   - "ELIN" stayed in the cottage garden after Mother teleported to the corner of the house (`look-1793050718.png`). It was still there when I reached the square (`look-1793050760.png`).
   - "OLD GARRICK" floated on the path after he walked to his cart (`look-1793050994.png`).
   - "DUNSTAN" floated in the meadow (`look-1793051289.png`).
   - "LITTLE BESS" floated over the square's bench through the whole aftermath and the ending (`look-1793051558.png`, `look-1793051609.png`).
   - *Repro:* talk to any NPC who moves during the conversation.
2. **Dunstan vanished mid-conversation.** I told him to say sorry, and his sprite disappeared from the meadow while he was still talking. Only his label was left (`look-1793051289.png`). Next he turned up in the Village Square.
3. **Characters and the player end up far apart while "face to face".**
   - `talk mother` walked me to the far left, with Father between us, three tiles from her (`look-1793050677.png`). Then she teleported to the corner of the house.
   - In the meadow, the narration had me "kneeling by the fence post" and "walking right up to" Dunstan, but the sprites were on opposite sides of the meadow (`look-1793051098.png`, `look-1793051155.png`).
4. **Snap-back after talking to Bess.** I crouched in front of her, and when the conversation ended my sprite was back at the well, where I'd been before (`look-1793050890.png`).
5. **A stale objective after the story moved on.** During the raid and the ending, a green toast said "New objective: Choose a gift for Ros at Old Garrick's stall." even though I'd already given the gift (`look-1793051558.png`, `look-1793051609.png`). It stayed up through the end screen.
6. **The "Left unfinished: The Miller's Secret" toast sits on screen for a long time.** It covered the top of the map from the market through the birthday scene and into the raid.
7. **The ending map contradicts the ending text.** The text says "a thin thread of smoke is all that is left of Briarwell", but the map behind it switched back to the bright, intact daytime cottage. The sidebar says "Blackened beams and grey ash where the cottage stood" over a cottage with a perfect red roof (`look-1793051609.png`).
8. **Mother goes missing in the evening scene.** She's listed at (3,2), up in the wall, and isn't visible in the candlelit room (`look-1793051424.png`).
9. **Bess speaks about herself in the third person.** "By the old fence post, **she said**. That's where I dropped her."
10. **The game said "Reconnected to the adventure" on a brand-new game**, before I had done anything.
11. **The round palm-tree/sun avatar in the bottom-left corner overlaps the first letters of the dialogue text** ("e difference", `look-1793050677.png`).
12. **I lost 1 HP (12 to 11) during the fire with no message.** That's fine, but I only noticed it in the sidebar.
13. **Clicking the North/South exit buttons in the sidebar auto-walks you over.** That's convenient. But the very first "walk right" in the garden did nothing because Father was standing in the way, with no bump or feedback.

## 5. Feel, visuals and pacing

- **Pixel art:** cute and readable. The meadow with flowers and the tree line is the prettiest area. The village square is a big empty dirt rectangle with one well, two benches and one kid, which is very bare for "market day". Bess and Ros share a sprite.
- **Mood lighting:** the evening tint and the grey ash palette are a lovely touch. Please do more of this, and keep it on for the ending.
- **Walking:** fine, but nothing reacts to me. No NPC idles, no turning to face me, no little hop or emote. Characters stand perfectly still until the story moves them, then slide or teleport.
- **The cottage interior** renders as a small box in a sea of black, and it shrinks even further when the dialogue box grows (`look-1793050552.png`). Please zoom it in.
- **Waits:** every typed action took 11–25 seconds (usually about 20). For the payoff lines that was worth it. For "I read the notice board" it wasn't. Some kind of "the story is thinking" animation in the scene, like a thought bubble over Wren, would help.
- **Reading:** the prose is lovely but heavy. Many beats are 4–6 lines of text, so it reads more like a book than a game. Some of it could live in the scene as emotes, bubbles and movement.
- **Menus:** the SAY/LEAVE conversation bar is clear. Choice buttons are readable and look nice. Tapping an Inventory tab didn't do anything for me, so I never saw my coins or the ribbon as items.
- **Pacing:** the morning feels great while I'm roaming. After that the story gallops: gift, bed, fire and the guild in about 6 clicks.

## 6. What would make me play more (most important first)

1. **Let me keep roaming after the second coin.** Don't auto-jump to Garrick and the evening. Finish the day when I decide to go home, so I can still do the mill, revisit Dunstan, and say goodbye to the village I'm about to lose. Losing it hurts more if I've walked it.
2. **Make the map match the words.** If the narrator mentions neighbours, the three small kids, or the "smaller child sobbing", put sprites there. If there's a signpost, let me read it. If the cottage burned, show the burned version at the end.
3. **Give me "+ item" and "+ coin" pops** and a visible coin counter. Earning a coin should feel like getting an item in Pokémon.
4. **Fix the ghost labels, teleports and snap-backs** so characters actually stand next to the person they're talking to.
5. **Let me play the raid**, even just 2–3 interactive beats (smoke drifting, flame tiles, a bandit silhouette at the door), instead of one wall of text.
6. **Give the gift moment room:** Ros reacts to the *specific* gift, ties the ribbon in her hair, and maybe the family sprites gather around.
7. **Add ambient life:** idle animations, NPCs turning to face me, the odd bird or butterfly in the meadow, villagers milling in the square.
8. **Add a "thinking" indicator in the scene during the 20-second waits.**
9. **Clear old toasts when the chapter moves on**, and never show an objective I've already finished.
