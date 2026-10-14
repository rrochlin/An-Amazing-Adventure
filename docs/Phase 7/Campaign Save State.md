---
tags:
  - phase-7
  - persistence
  - schema
---

# Campaign Save State

> [!summary]
> The game already persists world state, chat history, and combat state.
> Phase 7 needs an additional persisted layer for **campaign progression**.

Related notes:

- `[[Campaign Data Model]]`
- `[[Story]]`
- `[[Narrator Adherence]]`

## Current Gap

`server/internal/game/game.go` currently persists world and session details, but not the authored campaign runtime.

That means the system cannot reliably answer:

- which dramatic beat is active
- what branches have been taken
- what the rival party has already done
- what the narrator is allowed to reveal next

## Recommendation

Add a nested campaign runtime object rather than scattering campaign fields across the root save state.

This keeps the Phase 7 system coherent and easier to migrate.

## Proposed Schema Version

Suggested next schema version:

- `SchemaVersion = 5`

Version 5 introduces campaign runtime state.

## Proposed `Game` Addition

```go
type Game struct {
    // existing fields...
    Campaign *CampaignRuntimeState
}
```

## Proposed `SaveState` Addition

```go
type SaveState struct {
    // existing fields...
    Campaign *CampaignRuntimeState `json:"campaign,omitempty" dynamodbav:"campaign,omitempty"`
}
```

## Proposed Runtime State Shape

```go
type CampaignRuntimeState struct {
    CampaignID        string                       `json:"campaign_id" dynamodbav:"campaign_id"`
    CampaignVersion   string                       `json:"campaign_version" dynamodbav:"campaign_version"`
    ActiveNodeID      string                       `json:"active_node_id" dynamodbav:"active_node_id"`
    PreviousNodeIDs   []string                     `json:"previous_node_ids,omitempty" dynamodbav:"previous_node_ids,omitempty"`
    BoolFlags         map[string]bool              `json:"bool_flags,omitempty" dynamodbav:"bool_flags,omitempty"`
    IntCounters       map[string]int               `json:"int_counters,omitempty" dynamodbav:"int_counters,omitempty"`
    Labels            map[string]string            `json:"labels,omitempty" dynamodbav:"labels,omitempty"`
    ActiveObjectives  []ObjectiveState             `json:"active_objectives,omitempty" dynamodbav:"active_objectives,omitempty"`
    ActorStates       map[string]ActorRuntimeState `json:"actor_states,omitempty" dynamodbav:"actor_states,omitempty"`
    CompanionStates   map[string]CompanionState    `json:"companion_states,omitempty" dynamodbav:"companion_states,omitempty"`
    CampaignClock     int                          `json:"campaign_clock,omitempty" dynamodbav:"campaign_clock,omitempty"`
    LastSceneSummary  string                       `json:"last_scene_summary,omitempty" dynamodbav:"last_scene_summary,omitempty"`
}
```

## Why Typed Maps Instead Of `map[string]any`

Avoid `map[string]any` for persistence.

Reasons:

- harder to validate
- easier to corrupt over time
- more annoying with DynamoDB marshaling
- harder to grep and reason about in tests

Using `BoolFlags`, `IntCounters`, and `Labels` is much cleaner for Phase 7.

## Objective State

```go
type ObjectiveState struct {
    ID          string `json:"id" dynamodbav:"id"`
    Status      string `json:"status" dynamodbav:"status"` // hidden | active | completed | failed
    VisibleText string `json:"visible_text,omitempty" dynamodbav:"visible_text,omitempty"`
}
```

## Actor Runtime State

```go
type ActorRuntimeState struct {
    ActorID            string `json:"actor_id" dynamodbav:"actor_id"`
    CurrentRailStepID  string `json:"current_rail_step_id" dynamodbav:"current_rail_step_id"`
    CurrentLocationID  string `json:"current_location_id,omitempty" dynamodbav:"current_location_id,omitempty"`
    AlertLevel         int    `json:"alert_level,omitempty" dynamodbav:"alert_level,omitempty"`
    Status             string `json:"status,omitempty" dynamodbav:"status,omitempty"`
    LastAdvancedClock  int    `json:"last_advanced_clock,omitempty" dynamodbav:"last_advanced_clock,omitempty"`
}
```

## Companion State

```go
type CompanionState struct {
    CompanionID       string `json:"companion_id" dynamodbav:"companion_id"`
    Status            string `json:"status" dynamodbav:"status"`
    CurrentLocationID string `json:"current_location_id,omitempty" dynamodbav:"current_location_id,omitempty"`
    BondLevel         int    `json:"bond_level,omitempty" dynamodbav:"bond_level,omitempty"`
    LastBanterClock   int    `json:"last_banter_clock,omitempty" dynamodbav:"last_banter_clock,omitempty"`
}
```

## Save / Load Rules

### On campaign creation

- instantiate `CampaignRuntimeState`
- set `CampaignID`, `CampaignVersion`
- set `ActiveNodeID = StartNodeID`
- seed initial flags from campaign setup
- seed actor rails and companion states

### On each player turn

- increment `CampaignClock`
- apply deterministic effects
- update actor / companion state
- evaluate node completion or failure
- persist the new campaign runtime state with the rest of the game

### On reconnect / load

- load `CampaignRuntimeState`
- derive current objective text from runtime state and campaign definition
- send campaign summary to the client with normal game data

## Migration Strategy

### Old sessions

Existing non-campaign sessions should continue loading with `Campaign == nil`.

Interpretation:

- `Campaign == nil` -> legacy sandbox session
- `Campaign != nil` -> Phase 7 campaign session

This avoids breaking existing saves.

## API Changes

### `POST /api/games`

Move from a purely `CharacterCreationData` body toward a campaign-aware request.

Suggested shape:

```ts
type CreateCampaignGameRequest = {
  campaign_id: string
  character: CharacterCreationData
  setup_choices?: Record<string, string>
}
```

`setup_choices` is for campaign-specific startup decisions that are not part of normal D&D character creation.

Examples:

- selected entry route
- chosen timing strategy
- party stance / approach

### `GET /api/games/:session`

Include campaign runtime summary:

```ts
type CampaignStateView = {
  campaign_id: string
  campaign_title: string
  active_node_id: string
  active_node_title: string
  current_objective: string
  visible_flags?: Record<string, boolean>
}
```

## What Should Be Visible To The Client

Visible:

- campaign title
- player role
- current objective
- companion roster/status
- optional high-level progress marker

Not necessarily visible:

- hidden branches
- unrevealed actor rail state
- internal alert counters
- forbidden revelations list

## Test Requirements

- `ToSaveState` preserves campaign state
- `FromSaveState` restores campaign state
- v4 saves load with `Campaign == nil`
- campaign sessions survive optimistic locking retries
- actor rails and objectives persist across reconnects

## Practical Recommendation

> [!important]
> Keep campaign runtime state small and explicit.
> If a value is only needed for prompt flavor, do not persist it.

Persist only what is needed for correctness, branching, debugging, and resuming play.

## Bootstrap Implication

Because the world is prebuilt, campaign startup should not be treated as world generation anymore.

Startup now means:

- create the base game session
- load the campaign JSON
- instantiate authored rooms, actors, companions, and initial flags
- place the player in the starting room
- set the active story node
- optionally stream a short intro scene

That is a bootstrap / session-instantiation flow, not procedural generation.
