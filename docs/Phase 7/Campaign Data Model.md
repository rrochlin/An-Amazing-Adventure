---
tags:
  - phase-7
  - data-model
  - campaigns
---

# Campaign Data Model

> [!summary]
> A campaign should be an authored, versioned package with two separate layers:
> - **definition**: static authored content
> - **runtime state**: what changed during a specific playthrough

This note describes the **general schema for campaigns as a system**.
`[[Lich's Labyrinth]]` is the first campaign that should implement this schema.

Related notes:

- `[[Ideas]]`
- `[[Story]]`
- `[[Campaign Save State]]`
- `[[Narrator Adherence]]`
- `[[Lich's Labyrinth]]`

## Design Rule

Do **not** use freeform markdown as the runtime source of truth.

Use Obsidian markdown for design and writing, but use a typed JSON schema for executable campaign data.

## Recommended Storage Shape

### Authoring / code layout

Suggested layout:

```text
Phase 7/
  Ideas.md
  Story.md
  Lich's Labyrinth.md

server/internal/campaigns/
  registry.go
  loader.go
  schema.go
  defaults.go

server/campaigns/
  lichs-labyrinth/
    campaign.json
```

This keeps campaign content versioned, reviewable, and decoupled from code.

## JSON-First Decision

Confirmed direction:

- campaigns should be defined in JSON
- the server should load and validate them
- Obsidian notes remain the design/writing space, not the executable format
- campaign logic should be **data-heavy and code-evaluated**
- setup choices like entrance and timing should happen **inside the story intro**
- companions should be authored **inside campaign JSON**
- the physical campaign map should be a **full room graph in JSON**

## Top-Level Campaign Definition

Suggested JSON shape:

```json
{
  "id": "lichs-labyrinth",
  "version": "1",
  "title": "Lich's Labyrinth",
  "premise": "You are part of a mercenary crew hired to ambush a rival adventuring party.",
  "description": "An authored one-shot focused on pursuit, timing, and ambush inside a dangerous dungeon.",
  "tone": "dark-fantasy",
  "player_role": {},
  "party": {},
  "map": {},
  "actors": {},
  "objectives": {},
  "story_nodes": {},
  "start_node_id": "camp_briefing",
  "intro_sequence": {}
}
```

Suggested Go load shape:

```go
type CampaignDefinition struct {
    ID            string
    Version       string
    Title         string
    Premise       string
    Description   string
    Tone          string
    PlayerRole    PlayerRoleTemplate
    Party         PartyTemplate
    Map           CampaignMap
    Actors        map[string]ActorTemplate
    Objectives    map[string]ObjectiveTemplate
    StoryNodes    map[string]StoryNode
    StartNodeID   string
    IntroSequence IntroSequence
}
```

The Go struct exists to validate and execute the JSON, not to hand-author campaigns.

## Why This Level Exists

The campaign definition must answer all of these without consulting AI:

- What is this story about?
- Who is the player?
- Who are the important actors?
- What spaces exist?
- What are the major beats?
- What are valid transitions?
- What counts as success or failure?

## Player Role Template

```go
type PlayerRoleTemplate struct {
    Name              string
    Premise           string
    AllowedClasses    []string
    AllowedRaces      []string
    AllowedSubraces   []string
    RequiredBackstory []string
    StartingLoadoutID string
}
```

Phase 7 should keep the current character creator, but campaign data can lock it down.

## Character Creation Constraints

Suggested JSON section:

```json
"character_creation": {
  "use_standard_creator": true,
  "allowed_classes": ["fighter", "barbarian", "monk"],
  "allowed_races": ["human", "half-orc", "dwarf"],
  "allowed_subraces": ["hill-dwarf", "mountain-dwarf"],
  "locked_background_prompt": "You are a sellsword hired for a morally dubious contract.",
  "theme_hint_mode": "campaign_default",
  "preference_mode": "campaign_default"
}
```

This preserves the current creator UX while allowing each campaign to feel authored.

## Party Template

```go
type PartyTemplate struct {
    MaxHumanPlayers int
    CompanionIDs    []string
    StartingRoomID  string
    StartingStatus  string
}
```

For the first campaign:

- `MaxHumanPlayers = 1`
- companions are authored
- party identity is fixed

## Campaign Map

Reuse the existing room/node model where possible, but treat it as the physical layer under the story graph.

```go
type CampaignMap struct {
    Rooms         map[string]CampaignRoom
    Entrances     []EntranceOption
    Chokepoints   []string
    ObservationPoints []string
}

type CampaignRoom struct {
    ID            string
    Name          string
    Floor         int
    Type          string
    Description   string
    Connections   map[string]string
    Tags          []string
    StoryHooks    []string
}

type EntranceOption struct {
    ID            string
    Name          string
    StartRoomID   string
    Summary       string
    InitialFlags  map[string]bool
}
```

## Actor Template

Actors are important named groups or NPCs with authored intent.

```go
type ActorTemplate struct {
    ID              string
    Name            string
    Kind            string
    Goal            string
    Importance      string
    Rail            ActorRail
    CombatProfileID string
    DialogueStyle   string
}

type ActorRail struct {
    Steps []ActorRailStep
}

type ActorRailStep struct {
    ID                string
    LocationID        string
    AdvanceWhen       []Condition
    HoldWhen          []Condition
    RevealWhen        []Condition
    OnAdvanceEffects  []Effect
}
```

## Objective Template

```go
type ObjectiveTemplate struct {
    ID          string
    Title       string
    Description string
    VisibleFrom []Condition
    SuccessWhen []Condition
    FailWhen    []Condition
    Optional    bool
}
```

## Story Node

This is the core authored unit.

```go
type StoryNode struct {
    ID                   string
    Title                string
    Summary              string
    ObjectiveText        string
    EntryConditions      []Condition
    OnEnterEffects       []Effect
    CompletionRules      []TransitionRule
    FailureRules         []TransitionRule
    AllowedActorAdvance  []string
    NarratorContract     NodeNarratorContract
}

type TransitionRule struct {
    ConditionSet []Condition
    NextNodeID   string
    Effects      []Effect
}
```

Node transitions should remain deterministic.

That means:

- JSON defines the legal conditions and transitions
- code evaluates whether those conditions produce a transition
- AI does **not** choose the next node directly

## Conditions and Effects

Phase 7 should avoid a huge generic scripting engine.

Keep a small, explicit vocabulary.

This is the core Phase 7 compromise:

- JSON stores the authored rules and references
- Go owns the evaluator and allowed operator vocabulary
- campaigns do not execute arbitrary code

Condition fulfillment may still be resolved by AI, but only as structured classification against a fixed vocabulary.

Example:

- AI may determine `entry_route = hidden_entrance`
- AI may determine `briefing_complete = true`
- AI may determine `target_party_alerted = true`
- AI may **not** choose `next_node_id = floor2_pressure_setup`

The transition engine chooses the next node after evaluating the resolved condition state.

```go
type Condition struct {
    Kind  string
    Key   string
    Op    string
    Value string
}

type Effect struct {
    Kind  string
    Key   string
    Value string
}
```

Examples:

- condition: `flag entry_hidden == true`
- condition: `actor_progress target_party >= floor2_boss`
- effect: `set_flag target_party_alerted true`
- effect: `set_objective prepare_ambush active`
- effect: `move_actor novices intersection_retreat`

## Node Narrator Contract

Each node should tell the narrator what is allowed.

```go
type NodeNarratorContract struct {
    SceneFocus         string
    MustMention        []string
    MustNotReveal      []string
    ToneNotes          []string
    AllowedOutcomeTags []string
}
```

This is crucial for keeping narration aligned with authored progression.

## Example: Lich's Labyrinth Node

```go
StoryNode{
    ID: "choose_entry_plan",
    Title: "Choose Entry Plan",
    Summary: "The mercenary band decides how to enter the labyrinth.",
    ObjectiveText: "Choose an entrance and timing before the target party gains too much ground.",
    CompletionRules: []TransitionRule{
        {
            ConditionSet: []Condition{{Kind: "flag", Key: "entry_selected", Op: "==", Value: "true"}},
            NextNodeID:   "floor1_shadow_game",
        },
    },
    NarratorContract: NodeNarratorContract{
        SceneFocus:    "planning tension and party dynamics",
        MustMention:   []string{"three entry options", "risk of losing timing"},
        MustNotReveal: []string{"future lich chamber details", "floor 2 boss outcome"},
    },
}
```

## Recommendation: Keep Runtime Authoring Narrow

Do not try to build a universal narrative scripting language in Phase 7.

Instead:

- define a small number of condition kinds
- define a small number of effect kinds
- hardcode campaign evaluation logic where needed
- keep the first campaign concrete and inspectable

## API Surface Implication

The client will eventually need a campaign manifest for selection.

Suggested response shape:

```ts
type CampaignManifestItem = {
  id: string
  version: string
  title: string
  premise: string
  player_role: string
  difficulty: string
  estimated_length: string
}
```

## Loader / Validation Requirements

The loader should:

- parse `campaign.json`
- validate required fields
- validate references between nodes, actors, objectives, and rooms
- reject bad transitions or missing IDs at load time

Practical recommendation:

- load all campaigns at process start when possible
- fail fast on invalid campaign definitions

## Settled Decision

> [!success]
> Campaign definitions should live in JSON.

Recommended workflow:

- design in Obsidian
- encode executable campaign content in JSON
- validate into Go structs on load

## Settled Implementation Direction

> [!success]
> Phase 7 should be **data-heavy, code-evaluated**.

Meaning:

- authored nodes, rails, flags, companions, and rooms live in JSON
- Go evaluates conditions, transitions, and effects with a fixed engine
- campaign-specific code hooks should be avoided in the first implementation unless a real gap appears
