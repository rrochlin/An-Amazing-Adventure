---
tags:
  - story-engine
  - spike
---

# Framing the story agents: film or tabletop

> [!summary]
> Spike #92. Compares two ways to conceive of the story agents, **film** (director, actors, screenplay)
> and **tabletop** (game master, players, campaign module), plus a blend, using one real Briarwell turn.
> The goal is to narrow the choice down; the decision is Rob's. The choice leaks into the product: what a
> writer writes, what characters know and do when the player isn't looking, how outcomes are decided,
> and whether more than one human can play.

## Already decided (both framings must respect these)

- The **script stays in control** of plot, world and key decisions. The story agent is a game master
  between a film director and a tabletop DM: it keeps to the script but handles the player going off
  the rails.
- **Scenes decide who acts.** Only characters in the current scene, plus threads deliberately run in the
  background, take part in a turn.
- Characters get **only what they need to know**.
- **Code decides outcomes** (Phase 7 decision 4); agents describe them.

## The two framings

### Film: director, actors, screenplay

The campaign is a **screenplay**. Scenes carry stage directions (what must happen, the mood, who is on
set) and beats. Each character is an **actor** playing a part. For a scene, an actor gets their *sides*:
the character, what they want in this scene, how they react, sometimes lines. The **director** runs the
scene, decides how each beat plays out, asks actors for performances, and cuts the result together.

- Characters have no life off camera. Between scenes they don't exist.
- Only the protagonist (the human) improvises. Actors interpret; they don't decide where the plot goes.
- Writers write scenes, beats, stage directions and dialogue.

### Tabletop: game master, players, campaign module

The campaign is a **module**: locations, characters with motivations and secrets, and milestones that
must happen. Each non-player character is run like a **player**: it has goals, knows only what its
character has perceived, and declares what it *intends* to do. The **game master** adjudicates (rolls
dice, applies the rules), keeps the milestones on track, and narrates.

- Characters have goals of their own and can act off screen when the game master runs their thread.
- Everyone at the table, human or AI, has the same shape: perceive, intend, get adjudicated.
- Writers write characters, places, secrets and milestones, not lines.

## One turn, both ways

**The turn** (session b31d479d, village deeds scene, Meadow): *"Let's help out the little ones and take
Dunstan's coin. I want to sneak up on him and ambush him."*

### Film

1. The **director** reads the screenplay's Meadow beat: *Dunstan is bullying smaller children. Stopping
   him or joining him each pays a coin. On set: Dunstan, three small children.* (Tomas is not on set.)
2. The director decides how the beat plays: the ambush works, which matches the script's "stopped"
   outcome. It sets `deed_bully = stopped`, and the coin changes hands in state.
3. It asks the **Dunstan actor** for a reaction, with sides: *"Bully, two years older. Humiliated when
   outwitted. Pays up, sulks off."* The actor returns a shriek, a line, an exit.
4. The director cuts it together into the turn's prose.

Calls: director, plus one actor. The outcome was decided by the director, not by a roll.

### Tabletop

1. The **game master** reads the module: Meadow, Dunstan (*wants to look tough; carries a coin; will
   not admit defeat in front of the little ones*), the children, and the milestone *"one deed in
   Briarwell earns a coin"*.
2. It adjudicates the sneak: a **Stealth check** against Dunstan's passive Perception, rolled by code. It
   succeeds, and the dice result is shown to the player.
3. It tells the **Dunstan player** only what Dunstan perceives: *"A child bursts out of the grass and
   splatters you with mud; you're on your back."* Dunstan declares an intent, for example: *"Scramble
   up and threaten to tell the miller's wife what this brat did."*
4. The game master adjudicates that too (an intimidation attempt the player can answer next turn),
   applies state (the dropped coin), checks the milestone (`deed_bully = stopped`), and narrates.

Calls: game master, plus one call per character asked to act (here one). The outcome came from dice plus
a character's own choice, so the scene can go somewhere the script didn't spell out.

## What each one costs and buys

| | Film | Tabletop |
|---|---|---|
| **Script control** | Strong: actors can't steer the plot | Weaker: characters pursue goals, so the GM must rein them in toward milestones |
| **Player off the rails** | Director has to improvise new beats | Natural: characters react from their motivations |
| **Character consistency** | Good within a scene (sides); no memory between scenes unless written in | Good everywhere: one sheet, perception-based knowledge |
| **World feels alive** | Only on camera | Characters can act off screen (background threads) |
| **Mechanics and dice** | Optional; the director picks outcomes | Built in (fits the D&D 5e and rpg-toolkit stack, and "show the mechanics", #119) |
| **Multiple humans** | Awkward: a film has one protagonist (an ensemble is possible, but it's the director's story) | Natural: humans and AI players have the same shape |
| **What a writer writes** | Scenes, beats, stage directions, dialogue | Characters, places, secrets, milestones |
| **Presentation** | Matches the dialogue stage: scenes, title cards, paced lines | Needs table elements in the UI: rolls, intents, turn order |
| **Cost per turn** | Director + actors with lines (often 1–2 calls) | GM + each character asked to act (1 + n), plus adjudication in code |
| **Testability** | Outputs judged against the script | Outputs judged against motivations and rules; more variance |

## Blends

1. **Film-led:** director and screenplay; characters are actors with sides; dice only where the script
   asks for a check.
2. **Tabletop-led:** GM and module; characters are players with goals; milestones are the only script.
3. **A game master holding a screenplay:** the *authoring format* and *presentation* are film
   (scenes, beats, stage directions, title cards). The *runtime* is tabletop: characters are players with
   motivations and perception-limited knowledge, and the GM adjudicates with dice while steering every
   scene to its scripted beats. Characters are asked to act only when the scene gives them permission or
   a background thread needs them.

**My lean is blend 3.** It is closest to the middle ground Rob described (control of the script, a GM
who handles the rails), and it keeps the cinematic dialogue stage. It also lines up with decisions
already made: scenes deciding who acts, need-to-know knowledge, code-decided outcomes, and a path to
multiple humans. The cost is more calls per turn than film-led, mitigated by asking characters to act
only when needed. This is a lean, not a decision.

## Questions that decide it

1. **When the player isn't looking, should characters do things?** Yes points to players (tabletop).
2. **Should outcomes go through visible dice and rules?** Yes points to a tabletop runtime.
3. **What do you want to write: lines, or motivations?** Lines point to film; motivations to tabletop.
   A screenplay that sets goals and beats for characters who improvise their lines is blend 3.
4. **Should playing feel like being in a film or sitting at a table?** This sets presentation, and
   can differ from the runtime (blend 3).
5. **How soon do multiple humans matter?** Soon pushes hard toward players.
6. **What's the latency budget per turn?** The evaluation harness (#95) can measure each option.

## Decision (2026-10-22)

**Blend 3**, with Rob's answers to the questions:

1. **Off-screen life:** sometimes, for some characters. Leaving an area pauses its storylines; returning
   triggers a catch-up where the game master and the relevant characters work out what happened during the
   time skip.
2. **Dice:** yes, at the game master's discretion. Some moments call for a roll, others for a ruling.
3. **Lines or motivations:** mostly improvised, but planned scenes must unfold as planned. Scenes integral
   to the main story are controlled; lesser scenes are bounded by parameters. Characters come in two tiers:
   low-autonomy **actors** for enacting scenes, and **players** with autonomy within reason who present
   real obstacles.
4. **Feel:** a game, not a film.
5. **Latency:** about 10 seconds per turn to start, refined later.

Recorded in `docs/decisions.md`. Next: the script format (#93) defines control levels and agent tiers.
