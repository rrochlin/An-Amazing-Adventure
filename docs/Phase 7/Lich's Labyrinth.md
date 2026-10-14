---
aliases:
  - LichsLabyrinth
tags:
  - phase-7
  - campaign
  - oneshot
---

# Lich's Labyrinth

> [!summary]
> First shipped one-shot campaign.
>
> Premise: the player is part of a mercenary band hired to ambush a rival adventuring party exploring the Lich's Labyrinth.

> [!note]
> `Lich's Labyrinth` remains the target first real campaign, but runtime validation should happen first against the much smaller `server/campaigns/test/campaign.json` POC.

Related notes:

- `[[Ideas]]`
- `[[Story]]`

Executable draft artifact:

- `server/campaigns/lichs-labyrinth/campaign.json`

## Campaign Promise

This campaign should deliver:

- a strong opening premise
- immediate tactical intent
- a rival party the player can track, outmaneuver, or collide with
- authored progression through a memorable dungeon
- room for AI-driven tension, banter, and improvisational detail

## Player Role

The player is a member of a mercenary crew with a morally gray to evil-aligned brief: intercept, sabotage, or eliminate the target adventuring party before they complete their descent and return.

This is a **fixed-premise campaign**. Character expression exists inside the premise rather than replacing it.

## Party Model

Phase 7 assumes:

- **one human player**
- **AI-controlled companions**
- a fixed party identity

This keeps story control high while still giving the party-based feel your notes are aiming for.

## Default Companion Cast

Current seed names:

- Doren
- Amalia
- Rowan

These should become authored companion templates with:

- combat role
- personality hook
- relationship to the mission
- likely reactions under pressure

For implementation, these companion definitions should live directly in the campaign JSON.

> [!todo]
> Flesh out each companion as a reusable authored template.

## Core Opposing Forces

### Target Adventuring Party

- primary target of the mission
- enters on a known path
- clears deeper sections as they descend
- defeats floor bosses on the way down
- becomes vulnerable on the way back out

### Novice Adventurers

- secondary group
- briefly intersect with the main target path
- can become witnesses, collateral, or unexpected leverage

### Dungeon Monsters

- mostly remain in authored areas
- act as environmental pressure more than free-roaming agents

### The Lich

- final apex threat
- may be encountered directly or indirectly depending on timing and player choices

## Map / Entrance Model

The campaign begins with a choice around infiltration route and timing.

For Phase 7, these choices should happen inside the opening story flow, not on the create screen.

### Entrances

1. **Target Entrance**
Follow the rival party's likely route. Safest for tracking, riskiest for direct collision.

2. **Novice Entrance**
Move through the less dangerous route used by beginners. Better for flanking, delay, or manipulation.

3. **Hidden Entrance**
Highest stealth value. Best option for getting ahead of the target party or setting traps.

### Timing Choices

- enter before the target party
- enter shortly after them
- potentially wait for their return path

These choices should set early campaign flags and strongly shape the middle of the run.

## Campaign Structure

### Node 1 - Camp Briefing

The mercenary band prepares at camp, discusses the contract, and establishes tone, party dynamics, and intent.

Goals:

- establish the mission
- introduce companions
- let the player ask basic questions
- foreshadow the labyrinth and rival party

### Node 2 - Choose Entry Plan

The player chooses entrance and timing.

This is the first major branch and should set flags such as:

- `entry_route`
- `entered_before_target`
- `entered_after_target`
- `entered_via_hidden_entrance`

### Node 3 - Floor 1 Shadow Game

The party enters the labyrinth and begins tracking, avoiding, or getting ahead of the target party.

Possible beats:

- moving through cleared rooms
- hearing evidence of the target party nearby
- deciding whether to engage monsters or avoid them
- observing the novice group at the intersection

### Node 4 - First Escalation

Pressure rises once the player meaningfully alters the expected rhythm.

Possible escalation triggers:

- target party becomes alerted
- novices are threatened or warned
- mercenary party splits up
- player burns resources too early

### Node 5 - Floor 2 Pressure / Setup

This is the setup phase for the ambush proper.

The player should feel that the rival party is becoming both more dangerous and more reachable.

Possible beats:

- deciding whether to let the target party fight the floor boss first
- choosing where to set an ambush
- deciding whether to strike before or after the target's major fight

### Node 6 - Endgame Collision

The campaign converges around one of the main climaxes:

- ambush the target party on the way down
- ambush the target party after they defeat the lich
- collide with both the target party and lich-side threats in unstable conditions

### Node 7 - Escape / Aftermath

The surviving party attempts to leave with the consequences of the mission.

Possible outcomes:

- clean success
- costly success
- betrayal or internal fracture
- mission failure with survival
- catastrophic collapse

## Authored Rails

### Rival Party Rail

> [!note]
> See `[[Vision and Design Decisions]]` §6. The rival is **lightly simulated but bounded by these
> rails**: it has its own forward pace, and player actions can disrupt/pause/accelerate it, but it
> never leaves this authored path sequence or becomes a free open-ended agent. Advancement is driven
> by player progress + authored hold/reveal/advance conditions, **not** a literal clock (the campaign
> clock is soft/flavor only, §5).

Baseline authored progression:

1. enter floor 1
2. reach the novice intersection
3. defeat floor 1 mini-boss
4. descend to floor 2
5. defeat floor 2 mini-boss
6. descend to floor 3
7. confront lich
8. retrace route toward exit

This rail should pause, accelerate, or mutate when disrupted by player actions, but it should never become fully unbounded.

### Novice Rail

Baseline authored progression:

1. enter beginner route
2. briefly intersect with important traffic
3. retreat to safer areas
4. potentially become endangered witnesses or survivors

### Monster Rail

- monsters mostly remain in assigned zones
- some can be activated by noise, bloodshed, or magical disturbance
- monsters are a pacing and pressure tool, not a second story engine

## What Must Be Authored

- exact floor structure and key choke points
- where intersections happen
- when the rival party can be observed
- where a believable ambush can occur
- how floor bosses affect timing
- what the lich confrontation changes in the dungeon state

## What AI May Freely Elaborate

- room-level description
- party banter
- mercenary camp dialogue
- taunts and social pressure from the rival party
- sensory detail
- dramatic wording of success/failure

## Campaign State Suggestions

Useful flags / variables:

- `entry_route`
- `timing_strategy`
- `target_party_alert_level`
- `target_party_progress`
- `novice_party_status`
- `mercenary_party_cohesion`
- `ambush_site_prepared`
- `floor_1_boss_state`
- `floor_2_boss_state`
- `lich_state`

## Success and Failure

> [!note]
> See `[[Vision and Design Decisions]]` §7–§8. Endings are **discrete authored nodes** (clean success
> / costly success / partial success / failure / catastrophe) selected **deterministically by final
> campaign flags**; the narrator dramatizes but does not choose them. Failure is **permadeath**: the
> run ends with an authored aftermath and there is **no retry/reload of the same save** — the player
> replays by starting a new session.

### Success

- the target party is stopped, crippled, robbed, or eliminated
- the player party survives long enough to secure the outcome

### Failure

- the target party escapes with its objective intact
- the mercenary band is destroyed or routed
- the player loses the initiative beyond recovery

### Partial Success

- the contract goal is technically achieved, but at major cost
- the player survives but loses companions, leverage, loot, or reputation

## Authoring Backlog

- [x] draft first executable `campaign.json`
- [ ] define the target adventuring party roster
- [ ] define companion roles and personalities
- [ ] define the floor-by-floor map and chokepoints
- [ ] define the exact novice intersection scene
- [ ] define endgame variants
- [ ] define what happens if the player tries to ally with or betray either side

## Guiding Rule For This Campaign

The player should feel like they are hunting a party through a dangerous machine with just enough freedom to be clever, but not so much freedom that the story loses shape.
