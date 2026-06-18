# Lich's Labyrinth — Campaign Design Document

> **Document purpose:** Working design reference for an AI agent generating and interpreting campaign content in Yarn. Treat everything here as design intent, not locked rules. Sections marked `TODO` need fleshing out before the relevant scene can be wired.

---

## Concept

A one-shot dungeon heist where players are a **band of criminals**, not heroes. The goal is to ambush a stronger adventuring party at their most vulnerable moment — ideally right after they kill the final boss — and steal the artifact they were hired to recover.

Players do not need to clear the dungeon. They need to outmaneuver someone else who is.

---

## The Setup

A wealthy hero party has been contracted to clear **Lich's Labyrinth** and recover a powerful artifact. The criminals (players) have obtained a copy of the dungeon map and know the hero party's mission. The heroes are significantly stronger in a direct fight, so the player strategy is built around **sabotage, timing, and positioning** rather than brute force.

Players start with:
- A full map of the dungeon (all 3 floors)
- Knowledge that the heroes are already inside
- Knowledge that the heroes need holy equipment and a priest to fight the Lich — meaning there are paths the heroes can't easily use

---

## The Dungeon

**Name:** Lich's Labyrinth
**Structure:** 3 floors + boss chamber on Floor 3
**Floors connect via:** Staircase exits labeled F1E, F2E, F3E (each floor has multiple entry/exit points)

The dungeon has multiple entrances and exits. The hero party cannot watch them all.

### Floor 1

- Two entrances: a main entrance (top-center) and a secondary **Party Entrance** (right side, where criminals start)
- **Mini-boss** in the upper-left section
- Large **Pit** divides the lower section
- **Tight Squeeze** passage in lower-left — a narrow alternate route, bypasses main corridors
- **Invisible Path** (marked on the criminal map) — a secret route in the lower section, leads directly to the F3E staircase, allows skipping to Floor 3
- Right-center contains a loop of rooms used as a **beginner training zone** — the hero party will route through here

> TODO: What guards the Tight Squeeze? What reveals the Invisible Path — is it marked on the criminal map explicitly, or something they have to find?

### Floor 2

- Large jagged hazard zone in the upper-left
- **Mini-boss** in the center (eye creature, tentatively a spectator-type)
- Second pit (**Pit L2**) in the lower section
- Multiple staircase connections: F1E enters upper-left, two F3E exits (top-right and bottom-right)

> TODO: What is the hazard zone — flooded room, trap corridor, creature nest? What specifically is the Floor 2 mini-boss?

### Floor 3 (Boss Floor)

- **Lich's chamber** in the upper-left, flanked by **Ice Pits** on both sides
- The Ice Pits are a key tactical element — the hero party's holy equipment restricts how they can navigate this area, opening routes only the criminals can use
- **Cavern Floor** occupies the lower half — rougher terrain, distinct aesthetic from the labyrinth stonework above
- **Mini-boss** in the Cavern Floor section (same eye-creature type as Floor 2 or variant)
- Connections: F2E enters top-right, F3E exits bottom-center

> TODO: What do the Ice Pits do mechanically? What paths does the holy equipment restriction specifically block for the heroes? The AI agent needs to know which routes are "criminal only."

---

## The Boss

**Type:** Lich (evil, undead spellcaster)
**Location:** Floor 3, upper-left chamber
**Weakness:** Requires holy equipment and a priest to fight effectively — this is why the hero party's composition is what it is

> TODO: Name the Lich. Design the encounter. Does the Lich have phases? What does the fight look like at the point the criminals might intervene?

---

## The Artifact

> TODO: What is the artifact? This is the MacGuffin driving everything — it needs a name, a description, and a reason both the hero party and the criminals want it. Could be a phylactery shard, a cursed weapon, a soul-bound relic. The AI agent should reference this throughout.

---

## The Hero Party

The primary targets. Well-equipped, good-aligned adventurers doing a legitimate job. They are the antagonists of this scenario.

Key facts established at the start:
- Much stronger than the player party in a direct fight
- Hired to clear the dungeon and recover the artifact
- Must bring holy equipment and a priest (Lich requirement)
- Have a known route through the dungeon — the criminal map shows where they'll go
- Cannot watch all entrances and exits simultaneously

> TODO: Name and define the hero party members. Suggested 4–5 members with distinct roles. At least one should be perceptive/suspicious (makes infiltration harder), at least one should be likable (raises moral stakes at the end). Their holy equipment requirement should have a defined mechanical expression.

---

## The Secondary Adventuring Group

A smaller, weaker group also present in the dungeon. Key to the **Kill & Impersonate** quest line.

> TODO: Who are they? 2–3 members, lower level, probably optimistic/naive. Making them sympathetic raises the moral weight of eliminating them. Do they have a connection to anyone in the hero party?

---

## Mini-Bosses

Each floor has one mini-boss. The hero party will encounter all three in their standard route. The criminals can interact with mini-bosses tactically.

| Floor | Location | Creature | Notes |
|-------|----------|----------|-------|
| 1 | Upper-left | TODO | Undead or labyrinth-themed suggested |
| 2 | Center | Eye creature (spectator-type?) | TODO |
| 3 | Cavern Floor | Eye creature (same or variant) | TODO |

Mini-boss tactical options for criminals:
- Let heroes fight them normally (heroes spend resources)
- Bait or pull a mini-boss into the heroes' path early
- Sneak past while heroes are fighting one (use the distraction)
- Lure one into the heroes' camp during rest

---

## Time / Pacing System

The hero party progresses through the dungeon on an abstract clock. This creates urgency and tactical decisions for the criminals. These numbers are approximate and should be treated as vibes for now — the intent is to give the AI agent a sense of pacing relative to criminal actions.

- Heroes have roughly **19 time units** to complete the dungeon
- Moving unimpeded through a corridor: **1 unit**
- Passing through a room: **2 units**
- There are **4 blue rooms** on the map — each can contain a fight or a puzzle
- Mini-boss encounters: cost TBD (3–4 units feels right)
- Lich encounter: cost TBD

The AI agent should track hero progress implicitly and surface it to players through environmental cues (distant sounds of combat, flickering torchlight from another corridor, etc.) rather than explicit countdowns.

> TODO: Decide what a "time unit" represents narratively — is this clock tracking a single night, or dungeon "watches"? This affects when sabotage actions (especially night sabotage) can be taken.

---

## Quest Lines

Players are not railroaded. These are available strategies — each is a valid path to winning, with different risk profiles. Players may combine them.

### Sabotage at Night
While the hero party rests, criminals sneak in to sabotage their supplies, equipment, or camp.
- Low risk if stealth succeeds
- Weakens heroes before the Lich fight
- Complication: someone may be on watch

### Lay Traps
Pre-set traps in corridors the heroes will pass through. The criminal map is key to placing them effectively.
- Low upfront risk
- Passive attrition on the hero party
- Complication: heroes may detect/disarm, or dungeon creatures may spring them first

### Pull the Boss Early
Maneuver to trigger the Lich encounter before heroes are ready.
- High risk — Lich is dangerous to everyone
- Heroes fight the Lich resource-depleted, or caught between Lich and criminals
- Complication: criminals must avoid the Lich themselves

### Kill & Impersonate
Find the secondary adventuring group, befriend them, eliminate them privately, take their gear and clothes, use the disguise to infiltrate the hero party's trust, then sabotage them during a critical battle.
- Medium risk
- Requires social deception and a clean kill
- High reward if it works — direct access to the hero party
- Complication: heroes may see through it; secondary group may resist or complicate things morally

### Attack Mid-Battle (Lich Fight)
Wait until the hero party is fully committed to fighting the Lich, then attack both sides simultaneously.
- High risk — Lich may target criminals too
- Heroes are at their most vulnerable
- Complication: surviving the Lich's AoE while also fighting the heroes

### Attack After Battle
The classic ambush. Let the heroes exhaust everything killing the Lich, then strike.
- Low-medium risk
- Best odds in direct combat
- Complication: heroes may rest before leaving; if heroes wipe to the Lich, there may be nothing left to steal

### Fail / Pass Out
If criminals are defeated, captured, or incapacitated — session ends. This is a valid outcome.
- Soft failure: escape empty-handed but alive
- Hard failure: captured by heroes or killed by dungeon creatures

---

## AI Agent Notes

These are notes specifically for the agent interpreting this campaign, not for players.

- **Player intent parsing:** Players will give freeform responses. The agent should interpret these as one of: a **game action** (do something), a **choice** (commit to a strategy), or an **information request** (ask about the dungeon, the heroes, etc.). When ambiguous, surface the ambiguity back to the player as a Yarn branch.
- **Hero party visibility:** The agent should not tell players exactly where the heroes are — instead, give indirect evidence. The criminal map shows the hero party's *expected route*, not their real-time position.
- **Moral weight:** This campaign deliberately puts players in a villain role. The agent should not editorialize about player choices but should make consequences feel real. If players kill the secondary group, that should land with weight in the narrative.
- **Tone:** Gritty, tactical, morally grey. Not comedic. Not grimdark. The criminals are professionals doing a job.
- **Branching resolution:** Most quest lines converge at the Lich encounter. The Yarn wiring for the boss chamber scene needs to account for all the ways players might arrive there (weakened/fresh, disguised/exposed, allied with secondary group or not).

---

## Open Design Questions

These need answers before the relevant Yarn nodes can be wired.

- [ ] What is the artifact? (Name, description, stakes)
- [ ] What is the Lich's name and encounter design?
- [ ] What do the Ice Pits do — mechanical effect and which hero routes they block?
- [ ] Who are the hero party members? (Names, roles, personality hooks)
- [ ] Who is the secondary adventuring group?
- [ ] What are the three mini-boss creatures?
- [ ] What is the Floor 2 hazard zone?
- [ ] What guards or blocks the Tight Squeeze?
- [ ] Is the Invisible Path on the criminal map, or discovered in play?
- [ ] What is in the Pit (Floors 1 and 2)?
- [ ] What does the Lich fight look like from the criminal perspective — phases, safe positions, opportunities to interfere?
- [ ] What narrative time unit does the pacing clock represent?
- [ ] What is the hero party's holy equipment restriction expressed as in gameplay?