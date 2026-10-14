---
tags:
  - phase-7
  - ai
  - narrator
  - orchestration
---

# Narrator Adherence

> [!summary]
> For Phase 7, the narrator must describe campaign state, not invent it.

Related notes:

- `[[Story]]`
- `[[Campaign Data Model]]`
- `[[Campaign Save State]]`

## Current Architecture Limitation

Today the flow in `server/cmd/ws-chat/main.go` is roughly:

1. narrator streams prose
2. engineer infers mutations from prose
3. game state is saved

This works for loose sandbox play, but it is risky for authored campaigns.

### Why it is risky

- the narrator can imply facts that were never committed to state
- the engineer is forced to reverse-engineer canon from dramatic prose
- story progression becomes downstream of style rather than upstream of truth
- major campaign beats can drift

## Phase 7 Rule

Story-critical state must be decided **before** narration.

That means:

- legal transitions are defined before narration
- actor rail advancement happens in the orchestrator
- objective changes happen in the orchestrator
- the narrator receives committed scene facts and renders them dramatically

Important nuance:

- the AI may help determine **which authored conditions were fulfilled by the turn**
- the AI should not choose branches directly
- node transitions remain deterministic after that condition-resolution step

## Recommended Turn Pipeline

```text
player input
  -> intent classification
  -> orchestrator evaluation
  -> deterministic state updates
  -> scene package construction
  -> narrator streaming
  -> condition resolver extracts fulfilled authored conditions
  -> deterministic node transition / actor rail updates
  -> persist state + narrative
```

## Prebuilt Campaign Implication

Because Phase 7 uses prebuilt campaigns, the narrator should never be responsible for inventing:

- map structure
- quest structure
- actor goals
- initial story state

Those come from loaded campaign JSON plus saved runtime state.

## Role Split

### Orchestrator

The orchestrator owns canon.

Responsibilities:

- interpret the player action at the story level
- check active node constraints
- determine whether time advances
- advance rails when conditions are met
- change flags/counters/objectives
- decide node transitions
- build the narrator scene package

The orchestrator consumes structured condition results, but it owns the actual transition decision.

### Condition Resolver

The condition resolver is a constrained AI agent that answers questions like:

- Did the player commit to an entry route?
- Was the briefing meaningfully concluded?
- Did the target party become alerted?
- Was a named objective advanced?

Its output should be structured, whitelisted, and easy to validate.

Example shape:

```json
{
  "fulfilled_conditions": [
    { "kind": "flag", "key": "briefing_complete", "value": true },
    { "kind": "label", "key": "entry_route", "value": "hidden_entrance" }
  ],
  "notes": "The player clearly chose the hidden entrance and ended the planning discussion."
}
```

### Narrator

The narrator owns prose.

Responsibilities:

- describe the committed scene
- voice characters
- convey stakes and consequences
- maintain immersion

The narrator must **not**:

- create new mission goals
- reveal future beats early
- move important actors unless the scene package says they moved
- kill or save major actors unless state already says so
- skip forward in time beyond the allowed scene horizon

## Scene Package

The narrator should receive a compact, explicit package every turn.

Suggested shape:

```go
type NarratorScenePackage struct {
    CampaignTitle      string
    Premise            string
    ActiveNodeID       string
    ActiveNodeTitle    string
    ObjectiveText      string
    SceneFocus         string
    CurrentRoomName    string
    VisibleActors      []string
    RecentEvents       []string
    MustMention        []string
    MustNotReveal      []string
    AllowedOutcomes    []string
    ToneNotes          []string
    CombatLog          string
}
```

The narrator should be unable to see hidden campaign data that would tempt leakage.

## Prompt Contract

Instead of only giving the narrator broad DM instructions, give it a structured contract.

Suggested prompt sections:

```text
[ROLE]
You are the narrator of an authored one-shot campaign.

[CANONICAL FACTS]
- current node
- objective
- visible actors
- committed world events

[SCENE CONSTRAINTS]
- must mention
- must not reveal
- allowed scene horizon
- allowed outcomes

[STYLE]
- immersive prose only
- 2-4 paragraphs
- sensory, dramatic, in-world
```

## Hard vs Soft Constraints

### Hard constraints

These should come from deterministic state and should never be violated:

- active node
- room and actor positions
- success/failure outcomes already committed
- objective status
- combat results
- forbidden revelations

### Soft constraints

These influence tone and focus:

- pacing
- tension level
- companion banter opportunity
- environmental emphasis

## Condition Resolution Guardrail

> [!important]
> AI may resolve whether an authored condition was satisfied.
> AI may not invent new campaign state categories or choose the branch outcome directly.

## Recommended Change To Current Prompting

`server/internal/ai/bedrock.go:narratorSystemPrompt` is currently too generic for Phase 7.

It should be upgraded from:

- general DM persona
- current room + character context
- optional combat log

To include:

- campaign premise
- active node title and purpose
- explicit objective text
- committed actor progress
- forbidden revelations
- allowed outcomes for this turn

It should also include campaign-defined narration constraints from JSON, not only code-defined defaults.

## What To Do With `EngineerScan`

Recommendation for Phase 7:

- do **not** let `EngineerScan` decide story-critical progression
- do **not** let it infer node transitions
- do **not** let it infer major actor movement

However, a separate or refactored AI step may still be useful for **condition resolution**, as long as it returns only structured authored-condition results.

Possible roles it can still keep:

- extracting small visible world events for chat history
- translating deterministic combat/world changes into event records
- handling low-risk incidental state updates

If it stays, it should become subordinate to orchestrator decisions.

## Adherence Strategies

### Minimum viable approach

- deterministic orchestrator
- stronger scene package
- stronger narrator prompt
- no story-critical post-hoc inference

This is the best first step.

### Stronger future approach

- orchestrator produces a structured response manifest
- narrator prose is generated from that manifest
- optional validator checks prose for forbidden reveals

Example manifest:

```json
{
  "node": "floor1_shadow_game",
  "objective": "Track the rival party without exposing your band.",
  "events": [
    "The rival party has already cleared the antechamber.",
    "The novices were heard nearby but not seen."
  ],
  "must_mention": ["fresh signs of battle", "waning torch smoke"],
  "must_not_reveal": ["the lich's exact condition", "the rival party's full plan"]
}
```

## Streaming Consideration

Because narration is streamed, post-generation validation is weaker.

That means the best safety mechanisms are **pre-generation**:

- tightly scoped inputs
- committed state first
- narrator sees only what it should know
- narrator cannot mutate state

## Debugging Requirement

When a response feels off, you should be able to inspect:

- active node
- current objective
- actor rail states
- scene package
- narrator prompt block

If these are logged clearly, drift becomes much easier to diagnose.

## Success Criteria

The narrator is adhering correctly when:

- the prose feels alive
- the campaign stays on authored rails
- surprises come from valid branches, not canon breaks
- reconnecting later still makes perfect sense because state is explicit

## Key Principle

> [!important]
> The narrator is not the game engine.
> The narrator is the presentation layer for a stateful authored campaign.
