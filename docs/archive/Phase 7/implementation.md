---
tags:
  - phase-7
  - implementation
  - working-plan
---

# Phase 7 Implementation

> [!summary]
> This is the working implementation plan for Phase 7.
>
> It translates the design notes into execution order, code touchpoints, milestones, and done criteria.

Related notes:

- `[[Ideas]]`
- `[[Selected Stack]]`
- `[[Campaign Data Model]]`
- `[[Campaign JSON Schema]]`
- `[[Campaign Save State]]`
- `[[Story]]`
- `[[Narrator Adherence]]`
- `[[Condition Resolution]]`
- `[[Lich's Labyrinth]]`
- `[[Implementation Plan]]`

## Current Objective

Replace the current `character creation -> world-gen -> open-ended AI sandbox` flow with a `campaign selection -> campaign bootstrap -> authored one-shot runtime` flow.

The first shipped implementation targets one campaign: `[[Lich's Labyrinth]]`.

For development and validation, the first executable POC target should be a much smaller campaign package: `server/campaigns/test/campaign.json`.

That POC exists to validate the method, not the full authored experience.

## Settled Decisions

> [!success]
> Settled so far:
> - campaigns are defined in JSON
> - the current D&D creator stays, but campaigns may restrict options
> - `world-gen` is conceptually retired for Phase 7 campaign sessions
> - story-node progression is deterministic
> - AI may resolve authored condition fulfillment, but not choose branches
> - FSM is used for questline / story progression
> - Yarn is used for authored dialogue scenes
> - campaign-specific choices like entrance/timing happen in-story via freeform chat

## Non-Goals

- a full campaign authoring UI
- replacing the entire game runtime with Yarn
- using AI to improvise the main campaign structure
- supporting many campaigns before the first one is solid
- removing legacy sandbox sessions immediately
- fully implementing the complete `Lich's Labyrinth` experience before the runtime is proven

## Implementation Strategy

Build Phase 7 in vertical slices, but in a dependency-aware order.

High-level sequence:

1. Define executable campaign content shape.
2. Add persisted campaign runtime state.
3. Add campaign bootstrap flow.
4. Add FSM-backed quest progression.
5. Add Yarn-backed dialogue runtime.
6. Add AI condition resolution and narrator constraints.
7. Update the client for campaign sessions.
8. Implement `[[Lich's Labyrinth]]`.

## Workstreams

## 1. Campaign Content Format

Goal: define the author-facing content format the engine will load.

Primary outputs:

- `campaign.json` schema
- asset folder conventions
- dialogue asset reference format
- readable condition/action syntax

Likely code touchpoints:

- `server/internal/campaigns/` (new)
- `server/campaigns/` (new content root)

Tasks:

- [ ] define Go structs for `CampaignDefinition`
- [ ] define JSON loader/validator
- [ ] define file layout for campaign assets
- [ ] define node modes: `ai_scene`, `yarn_dialogue`, `hybrid`
- [ ] define author-facing condition syntax
- [ ] define author-facing effect/action syntax

Done when:

- a campaign can be loaded and validated from disk
- invalid references fail fast with useful errors

## 2. Campaign Runtime Persistence

Goal: make campaign progression first-class persisted state.

Primary outputs:

- `SchemaVersion = 5`
- `CampaignRuntimeState`
- save/load support

Likely code touchpoints:

- `server/internal/game/game.go`
- `server/internal/db/db.go`
- `server/internal/game/*_test.go`

Tasks:

- [ ] add `Campaign *CampaignRuntimeState` to `Game`
- [ ] add `Campaign *CampaignRuntimeState` to `SaveState`
- [ ] update `ToSaveState`
- [ ] update `FromSaveState`
- [ ] define migration behavior for old sessions
- [ ] add tests for round-tripping campaign runtime state

Done when:

- campaign sessions persist and restore without losing node state, flags, actor state, or companion state
- old non-campaign sessions still load safely

## 3. Campaign Registry And Selection

Goal: make campaigns discoverable and selectable at game creation time.

Primary outputs:

- campaign manifest API
- campaign-aware create request
- client campaign selection UI

Likely code touchpoints:

- `server/internal/campaigns/registry.go`
- `server/cmd/http-games/main.go`
- `client/src/routes/create.tsx`
- `client/src/services/api.game.ts`
- `client/src/types/types.ts`

Tasks:

- [ ] define campaign manifest response shape
- [ ] add server registry for available campaigns
- [ ] add campaign selection to create flow
- [ ] keep D&D character creation after campaign selection
- [ ] apply campaign restrictions to available creator options

Done when:

- the client can list supported campaigns
- the player can choose a campaign and create a session against it

## 4. Campaign Bootstrap

Goal: replace procedural world generation with authored session instantiation.

Primary outputs:

- campaign bootstrap path
- player + companion + actor placement
- initial node setup
- intro scene startup

Likely code touchpoints:

- `server/cmd/http-games/main.go`
- new bootstrap package or lambda path
- `server/cmd/world-gen/main.go` retirement/removal path

Tasks:

- [ ] define bootstrap request payload
- [ ] load selected campaign JSON during session start
- [ ] instantiate authored rooms/items/actors/companions
- [ ] place player in starting room
- [ ] seed initial campaign runtime state
- [ ] set opening story node
- [ ] decide whether bootstrap stays async or becomes inline

Done when:

- starting a campaign session no longer depends on procedural world generation
- a new campaign session begins at the expected opening node with valid world state

## 5. FSM-Backed Quest Runtime

Goal: make questline / story-node progression deterministic and inspectable.

Primary outputs:

- FSM configuration derived from campaign definition
- deterministic transition evaluation
- hooks around node changes

Likely code touchpoints:

- `server/internal/story/` or `server/internal/campaignruntime/` (new)
- `server/internal/campaigns/`

Dependency choice:

- `github.com/enetx/fsm`

Tasks:

- [ ] define mapping from campaign nodes to FSM states
- [ ] define transition guard evaluation against campaign runtime state
- [ ] define transition hooks for enter/exit side effects
- [ ] define how actor rails advance relative to node transitions
- [ ] add logging/debug output for every node transition

Done when:

- active node changes are deterministic
- illegal transitions are impossible without explicit code bypass
- node transitions are easy to inspect in logs and persisted state

## 6. Yarn Dialogue Runtime

Goal: integrate authored Yarn scenes into campaign nodes.

Primary outputs:

- dialogue asset loader
- runtime execution bridge
- command / variable bridge into campaign state

Likely code touchpoints:

- `server/internal/dialogue/` (new)
- `server/campaigns/**/dialogue/`
- `server/cmd/ws-chat/main.go`

Dependency choice:

- `drjosh.dev/yarn`

Tasks:

- [x] define campaign `dialogue_assets` structure
- [x] load Yarn program + string table assets
- [x] define story node -> dialogue asset mapping
- [x] define how Yarn state is minimally persisted or reconstructed for current node execution
- [ ] define allowed Yarn commands that can affect runtime state
- [x] define variable mirroring between campaign runtime and Yarn variables
- [ ] define interactive player-facing Yarn choice handling

Done when:

- [x] a campaign node can run a Yarn dialogue scene reliably
- [ ] authored dialogue choices can contribute to campaign condition fulfillment

## 7. ws-chat Orchestration

Goal: make the runtime correctly handle `ai_scene`, `yarn_dialogue`, and `hybrid` nodes.

Primary outputs:

- scene-mode-aware chat pipeline
- latest-turn orchestration contract
- unified persistence path

Likely code touchpoints:

- `server/cmd/ws-chat/main.go`
- `server/internal/ai/`
- `server/internal/dialogue/`
- `server/internal/story/`

Tasks:

- [ ] define execution path for `ai_scene`
- [ ] define execution path for `yarn_dialogue`
- [ ] define execution path for `hybrid`
- [ ] define how narrative chunks are streamed for each mode
- [ ] define how chat history is written in each mode
- [ ] define how state deltas are emitted in each mode

Done when:

- a player message routes to the correct scene runtime
- all three scene modes produce coherent state updates and UI updates

## 8. Condition Resolution

Goal: use AI for authored condition classification without surrendering branch control.

Primary outputs:

- structured condition resolver
- validation against campaign vocabulary
- latest-turn-only resolution

Likely code touchpoints:

- `server/internal/ai/`
- `server/internal/story/`
- `server/cmd/ws-chat/main.go`

Tasks:

- [ ] define resolver input contract
- [ ] define resolver output contract
- [ ] validate output against allowed flags/labels/counters
- [ ] reject invalid keys and invalid values
- [ ] apply resolved condition updates to campaign runtime state
- [ ] run deterministic transition evaluation afterward

Constraints:

- resolver uses latest turn only
- resolver emits only flags / labels / counters
- resolver never emits `next_node_id`

Done when:

- freeform player input can reliably trigger authored campaign progression without AI deciding the branch directly

## 9. Narrator Refactor

Goal: constrain AI narration to the current campaign truth.

Primary outputs:

- scene package
- campaign-aware narrator prompt
- hybrid-scene support

Likely code touchpoints:

- `server/internal/ai/bedrock.go`
- `server/internal/story/`

Tasks:

- [ ] define `NarratorScenePackage`
- [ ] include active node, objective, visible actors, forbidden revelations, and allowed outcomes
- [ ] include Yarn-adjacent context for hybrid scenes when needed
- [ ] ensure narrator cannot invent branch outcomes

Done when:

- narration consistently reflects committed campaign state
- reconnecting later still makes sense because story truth is explicit in saved state

## 10. Client UX

Goal: expose campaign identity and progression cleanly in the existing client.

Primary outputs:

- campaign selection UI
- objective/progress UI
- companion status UI
- better authored intro experience

Likely code touchpoints:

- `client/src/routes/create.tsx`
- `client/src/routes/game-{$sessionUUID}.tsx`
- `client/src/components/GameInfo.tsx`
- `client/src/types/types.ts`

Tasks:

- [ ] add campaign selection to create flow
- [ ] surface current objective and campaign title
- [ ] surface companion roster/status
- [ ] decide how Yarn-backed choices render in the chat UI
- [ ] soften or replace `WorldGenTerminal` for authored campaign intros

Done when:

- the client feels like a campaign session, not a procedural sandbox session

## 11. First Campaign Package

Goal: create a real executable content package for `[[Lich's Labyrinth]]`.

POC note:

- first validate the runtime using the simpler `test` campaign
- use `Lich's Labyrinth` as the richer follow-up campaign once the method is proven

Primary outputs:

- `campaign.json`
- dialogue assets
- authored room graph
- actor rails
- initial nodes and transitions

Likely code touchpoints:

- `server/campaigns/lichs-labyrinth/`

Tasks:

- [x] draft first-pass `server/campaigns/lichs-labyrinth/campaign.json`
- [ ] define room graph and entrances
- [ ] define companions
- [ ] define rival party rail
- [ ] define novice party rail
- [ ] define opening nodes and mid-campaign nodes
- [ ] define Yarn dialogue assets for briefing and key encounters

Done when:

- the campaign is playable end-to-end in at least one valid path

Current draft artifact:

- `server/campaigns/lichs-labyrinth/campaign.json`
- `server/campaigns/test/campaign.json`

## 12. Testing And Verification

Goal: keep the Phase 7 runtime debuggable and safe to iterate on.

Tests to add:

- [ ] campaign JSON validation tests
- [ ] save/load tests for campaign runtime state
- [ ] FSM transition tests
- [ ] Yarn asset loading tests
- [ ] condition resolver contract tests
- [ ] ws-chat orchestration tests for all scene modes
- [ ] create flow tests for campaign selection restrictions

Verification commands likely needed later:

- `go test ./...`
- `pnpm test --run`
- `pnpm build`

## Suggested Delivery Milestones

## Milestone 1 - Campaign Session Skeleton

Player can select a campaign, create a session, and load into a bootstrap-created authored world with persisted campaign runtime state.

## Milestone 2 - Deterministic Story Progression

Campaign runtime can evaluate authored node transitions deterministically using resolved flags/labels/counters.

## Milestone 3 - Yarn Scene Support

At least one Yarn-backed scene runs inside a campaign node and affects campaign progression correctly.

## Milestone 4 - First Playable Vertical Slice

`Lich's Labyrinth` supports:

- camp briefing
- entrance/timing choice
- node progression
- objective display
- persistence across reconnects

## Risks

- campaign logic ends up split awkwardly across JSON, FSM config, and ad hoc code
- Yarn commands become an uncontrolled escape hatch into campaign state
- AI condition resolver becomes too broad and starts behaving like a planner
- hybrid scenes become hard to debug if the orchestration order is muddy

## Proposed Core Structs

This section is the first-pass proposal for the actual Go types we will likely implement.

These are not final, but they are concrete enough to guide the first implementation slice.

## Campaign Loading Structs

Suggested package:

- `server/internal/campaigns/`

Suggested types:

```go
type CampaignDefinition struct {
    ID                string                       `json:"id"`
    Version           string                       `json:"version"`
    Title             string                       `json:"title"`
    Premise           string                       `json:"premise"`
    Description       string                       `json:"description,omitempty"`
    Tone              string                       `json:"tone,omitempty"`
    CharacterCreation CharacterCreationRules       `json:"character_creation"`
    PlayerRole        PlayerRoleTemplate           `json:"player_role,omitempty"`
    Party             PartyTemplate                `json:"party"`
    Map               CampaignMap                  `json:"map"`
    Actors            map[string]ActorTemplate     `json:"actors"`
    Objectives        map[string]ObjectiveTemplate `json:"objectives,omitempty"`
    DialogueAssets    map[string]DialogueAsset     `json:"dialogue_assets,omitempty"`
    StoryNodes        map[string]StoryNode         `json:"story_nodes"`
    StartNodeID       string                       `json:"start_node_id"`
    IntroSequence     IntroSequence                `json:"intro_sequence,omitempty"`
}

type CharacterCreationRules struct {
    UseStandardCreator   bool     `json:"use_standard_creator"`
    AllowedClasses       []string `json:"allowed_classes,omitempty"`
    AllowedRaces         []string `json:"allowed_races,omitempty"`
    AllowedSubraces      []string `json:"allowed_subraces,omitempty"`
    LockedBackgroundText string   `json:"locked_background_prompt,omitempty"`
    ThemeHintMode        string   `json:"theme_hint_mode,omitempty"`
    PreferenceMode       string   `json:"preference_mode,omitempty"`
}

type PlayerRoleTemplate struct {
    Name        string `json:"name,omitempty"`
    Premise     string `json:"premise,omitempty"`
    RoleSummary string `json:"role_summary,omitempty"`
}

type PartyTemplate struct {
    MaxHumanPlayers int      `json:"max_human_players"`
    StartingRoomID  string   `json:"starting_room_id"`
    CompanionIDs    []string `json:"companion_ids,omitempty"`
}

type CampaignMap struct {
    Rooms     map[string]CampaignRoom `json:"rooms"`
    Entrances []EntranceOption        `json:"entrances,omitempty"`
}

type CampaignRoom struct {
    Name        string            `json:"name"`
    Floor       int               `json:"floor,omitempty"`
    Type        string            `json:"type,omitempty"`
    Description string            `json:"description,omitempty"`
    Tags        []string          `json:"tags,omitempty"`
    Connections map[string]string `json:"connections,omitempty"`
}

type EntranceOption struct {
    ID           string          `json:"id"`
    Name         string          `json:"name"`
    StartRoomID  string          `json:"start_room_id"`
    InitialFlags map[string]bool `json:"initial_flags,omitempty"`
}

type ActorTemplate struct {
    Name       string    `json:"name"`
    Kind       string    `json:"kind"`
    Goal       string    `json:"goal,omitempty"`
    Importance string    `json:"importance,omitempty"`
    Rail       ActorRail `json:"rail,omitempty"`
}

type ActorRail struct {
    Steps []ActorRailStep `json:"steps,omitempty"`
}

type ActorRailStep struct {
    ID             string       `json:"id"`
    LocationID     string       `json:"location_id,omitempty"`
    AdvanceWhen    []Condition  `json:"advance_when,omitempty"`
    HoldWhen       []Condition  `json:"hold_when,omitempty"`
    RevealWhen     []Condition  `json:"reveal_when,omitempty"`
    OnAdvance      []Action     `json:"on_advance,omitempty"`
}

type ObjectiveTemplate struct {
    Title       string      `json:"title"`
    Description string      `json:"description,omitempty"`
    VisibleWhen []Condition `json:"visible_when,omitempty"`
    SuccessWhen []Condition `json:"success_when,omitempty"`
    FailWhen    []Condition `json:"fail_when,omitempty"`
    Optional    bool        `json:"optional,omitempty"`
}

type DialogueAsset struct {
    Format  string `json:"format"`
    Program string `json:"program"`
    Strings string `json:"strings"`
}

type StoryNode struct {
    Mode              string                 `json:"mode"`
    DialogueAssetID   string                 `json:"dialogue_asset_id,omitempty"`
    DialogueStartNode string                 `json:"dialogue_start_node,omitempty"`
    Title             string                 `json:"title"`
    Summary           string                 `json:"summary,omitempty"`
    ObjectiveText     string                 `json:"objective_text,omitempty"`
    OnEnter           []Action               `json:"on_enter,omitempty"`
    CompletionRules   []TransitionRule       `json:"completion_rules,omitempty"`
    FailureRules      []TransitionRule       `json:"failure_rules,omitempty"`
    NarratorContract  NodeNarratorContract   `json:"narrator_contract,omitempty"`
    AllowedConditions []AllowedConditionSpec `json:"allowed_conditions,omitempty"`
}

type TransitionRule struct {
    Conditions []Condition `json:"conditions"`
    NextNodeID string      `json:"next_node_id"`
    Actions    []Action    `json:"actions,omitempty"`
}

type NodeNarratorContract struct {
    SceneFocus    string   `json:"scene_focus,omitempty"`
    MustMention   []string `json:"must_mention,omitempty"`
    MustNotReveal []string `json:"must_not_reveal,omitempty"`
    ToneNotes     []string `json:"tone_notes,omitempty"`
}

type IntroSequence struct {
    Mode          string              `json:"mode,omitempty"`
    OpeningNodeID string              `json:"opening_node_id,omitempty"`
    OpeningText   string              `json:"opening_text,omitempty"`
    OpeningChoices map[string][]string `json:"opening_choices,omitempty"`
}
```

## Persisted Runtime Structs

Suggested package:

- `server/internal/game/`

Suggested types:

```go
type CampaignRuntimeState struct {
    CampaignID       string                       `json:"campaign_id" dynamodbav:"campaign_id"`
    CampaignVersion  string                       `json:"campaign_version" dynamodbav:"campaign_version"`
    ActiveNodeID     string                       `json:"active_node_id" dynamodbav:"active_node_id"`
    PreviousNodeIDs  []string                     `json:"previous_node_ids,omitempty" dynamodbav:"previous_node_ids,omitempty"`
    BoolFlags        map[string]bool              `json:"bool_flags,omitempty" dynamodbav:"bool_flags,omitempty"`
    Labels           map[string]string            `json:"labels,omitempty" dynamodbav:"labels,omitempty"`
    Counters         map[string]int               `json:"counters,omitempty" dynamodbav:"counters,omitempty"`
    ActiveObjectives []ObjectiveState             `json:"active_objectives,omitempty" dynamodbav:"active_objectives,omitempty"`
    ActorStates      map[string]ActorRuntimeState `json:"actor_states,omitempty" dynamodbav:"actor_states,omitempty"`
    CompanionStates  map[string]CompanionState    `json:"companion_states,omitempty" dynamodbav:"companion_states,omitempty"`
    CampaignClock    int                          `json:"campaign_clock,omitempty" dynamodbav:"campaign_clock,omitempty"`
    ActiveDialogue   *DialogueRuntimeState        `json:"active_dialogue,omitempty" dynamodbav:"active_dialogue,omitempty"`
}

type ObjectiveState struct {
    ID          string `json:"id" dynamodbav:"id"`
    Status      string `json:"status" dynamodbav:"status"`
    VisibleText string `json:"visible_text,omitempty" dynamodbav:"visible_text,omitempty"`
}

type ActorRuntimeState struct {
    ActorID           string `json:"actor_id" dynamodbav:"actor_id"`
    CurrentRailStepID string `json:"current_rail_step_id,omitempty" dynamodbav:"current_rail_step_id,omitempty"`
    CurrentLocationID string `json:"current_location_id,omitempty" dynamodbav:"current_location_id,omitempty"`
    AlertLevel        int    `json:"alert_level,omitempty" dynamodbav:"alert_level,omitempty"`
    Status            string `json:"status,omitempty" dynamodbav:"status,omitempty"`
}

type CompanionState struct {
    CompanionID       string `json:"companion_id" dynamodbav:"companion_id"`
    Status            string `json:"status" dynamodbav:"status"`
    CurrentLocationID string `json:"current_location_id,omitempty" dynamodbav:"current_location_id,omitempty"`
    BondLevel         int    `json:"bond_level,omitempty" dynamodbav:"bond_level,omitempty"`
}

type DialogueRuntimeState struct {
    AssetID        string         `json:"asset_id" dynamodbav:"asset_id"`
    CurrentNode    string         `json:"current_node,omitempty" dynamodbav:"current_node,omitempty"`
    Variables      map[string]any `json:"variables,omitempty" dynamodbav:"variables,omitempty"`
    AwaitingChoice bool           `json:"awaiting_choice,omitempty" dynamodbav:"awaiting_choice,omitempty"`
}
```

## Author-Facing Condition Syntax

Recommendation: use readable named condition types in JSON.

Do not make campaign authors write raw operator tuples as the primary format.

### Supported Phase 7 Condition Types

```json
{ "type": "flag_true", "key": "briefing_complete" }
{ "type": "flag_false", "key": "target_party_alerted" }
{ "type": "label_equals", "key": "entry_route", "value": "hidden_entrance" }
{ "type": "label_in", "key": "timing_strategy", "values": ["before_target", "after_target"] }
{ "type": "counter_gte", "key": "alert_level", "value": 2 }
{ "type": "counter_lte", "key": "mercenary_cohesion", "value": 0 }
{ "type": "actor_at_step", "actor_id": "target_party", "step_id": "floor2_boss_down" }
{ "type": "actor_in_room", "actor_id": "target_party", "room_id": "f2_shrine" }
{ "type": "objective_status", "objective_id": "prepare_ambush", "status": "active" }
```

Suggested Go type:

```go
type Condition struct {
    Type       string   `json:"type"`
    Key        string   `json:"key,omitempty"`
    Value      any      `json:"value,omitempty"`
    Values     []string `json:"values,omitempty"`
    ActorID    string   `json:"actor_id,omitempty"`
    StepID     string   `json:"step_id,omitempty"`
    RoomID     string   `json:"room_id,omitempty"`
    ObjectiveID string  `json:"objective_id,omitempty"`
    Status     string   `json:"status,omitempty"`
}
```

## Author-Facing Action Syntax

Actions are applied deterministically by code after a transition or node hook fires.

### Supported Phase 7 Action Types

```json
{ "type": "set_flag", "key": "briefing_complete", "value": true }
{ "type": "set_label", "key": "entry_route", "value": "hidden_entrance" }
{ "type": "inc_counter", "key": "alert_level", "value": 1 }
{ "type": "set_counter", "key": "campaign_clock", "value": 3 }
{ "type": "activate_objective", "objective_id": "track_target_party" }
{ "type": "complete_objective", "objective_id": "choose_entry_plan" }
{ "type": "fail_objective", "objective_id": "remain_hidden" }
{ "type": "move_actor_to_step", "actor_id": "target_party", "step_id": "floor2_boss_down" }
{ "type": "move_actor_to_room", "actor_id": "novices", "room_id": "beginner_halls" }
{ "type": "start_dialogue", "asset_id": "camp_briefing", "start_node": "CampBriefing" }
{ "type": "end_dialogue" }
```

Suggested Go type:

```go
type Action struct {
    Type        string `json:"type"`
    Key         string `json:"key,omitempty"`
    Value       any    `json:"value,omitempty"`
    ActorID     string `json:"actor_id,omitempty"`
    StepID      string `json:"step_id,omitempty"`
    RoomID      string `json:"room_id,omitempty"`
    ObjectiveID string `json:"objective_id,omitempty"`
    AssetID     string `json:"asset_id,omitempty"`
    StartNode   string `json:"start_node,omitempty"`
}
```

## Allowed Condition Vocabulary

The resolver must not be able to invent arbitrary state keys.

Each node should explicitly declare which authored conditions may be fulfilled during the turn.

Suggested type:

```go
type AllowedConditionSpec struct {
    Type          string   `json:"type"`
    Key           string   `json:"key,omitempty"`
    AllowedValues []string `json:"allowed_values,omitempty"`
}
```

Example:

```json
"allowed_conditions": [
  { "type": "flag_true", "key": "briefing_complete" },
  { "type": "label_equals", "key": "entry_route", "allowed_values": ["target_entrance", "novice_entrance", "hidden_entrance"] },
  { "type": "label_equals", "key": "timing_strategy", "allowed_values": ["before_target", "after_target", "wait_for_return"] }
]
```

## Yarn Integration Contract

Yarn should be a scene runtime, not the top-level campaign runtime.

### Asset Convention

Suggested layout:

```text
server/campaigns/
  lichs-labyrinth/
    campaign.json
    dialogue/
      camp-briefing.yarnc
      camp-briefing.csv
      rival-party-intro.yarnc
      rival-party-intro.csv
```

### Campaign JSON References

Campaign nodes reference a dialogue asset and an entry node.

Example:

```json
"dialogue_assets": {
  "camp_briefing": {
    "format": "yarn",
    "program": "dialogue/camp-briefing.yarnc",
    "strings": "dialogue/camp-briefing.csv"
  }
}
```

### Runtime Responsibilities

Suggested package:

- `server/internal/dialogue/`

Suggested responsibilities:

- load Yarn program assets
- load associated string table
- run a requested Yarn entry node
- expose emitted lines/options/commands to the chat pipeline
- mirror allowed dialogue variables into runtime state

### Supported Yarn Command Boundary

Phase 7 should keep the command bridge narrow.

Recommended supported commands:

- `set_flag <key> <bool>`
- `set_label <key> <value>`
- `inc_counter <key> <int>`
- `activate_objective <id>`
- `complete_objective <id>`
- `fail_objective <id>`
- `start_ai_scene <scene_id>`
- `end_dialogue`

Commands should not be allowed to:

- directly set `next_node_id`
- move arbitrary actors without authored permission
- mutate unknown runtime keys

### Yarn Variable Mirroring

Recommendation for Phase 7:

- keep Yarn variables local by default
- explicitly mirror only a small set of approved campaign values

Likely mirrored values:

- current companion names
- selected entrance/timing labels
- alert level counters
- objective visibility text

### Persistence Strategy

First-pass recommendation:

- persist a lightweight `DialogueRuntimeState`
- store active asset ID, current node, mirrored variables, and choice-waiting state
- do not try to make Yarn the canonical persistence format for the whole campaign

### ws-chat Integration Contract

For a `yarn_dialogue` node:

1. player input is interpreted as dialogue continuation or option selection
2. Yarn runtime emits lines/options/commands
3. allowed commands are translated into runtime actions
4. any condition-relevant results are fed into deterministic transition evaluation

For a `hybrid` node:

1. AI narrator handles framing prose
2. Yarn handles the authored dialogue segment
3. condition resolver inspects the latest turn output
4. deterministic transition engine evaluates the next state

## Recommended First Concrete Deliverables

1. Implement the Go structs above in `server/internal/campaigns` and `server/internal/game`.
2. Finalize the Phase 7 condition/action vocabulary.
3. Validate and iterate on `server/campaigns/lichs-labyrinth/campaign.json`.
4. Define the exact Yarn command parser and runtime bridge.

## Open Questions

- Should Yarn runtime state be persisted directly, or reconstructed from campaign state + current dialogue node?
- What exact Yarn commands are allowed to affect campaign runtime state?
- Do actor rails advance before or after condition resolution for a turn?
- Should `EngineerScan` survive in a narrow role, or be retired entirely for campaign sessions?

## Immediate Next Drafts

1. Draft concrete Go structs for campaign loading and runtime state.
2. Draft the first author-facing condition/action JSON syntax.
3. Draft the Yarn integration contract.
4. Refine the first executable `server/campaigns/lichs-labyrinth/campaign.json`.

## Progress Update

> [!success]
> First-pass executable campaign packages drafted:
> - `server/campaigns/lichs-labyrinth/campaign.json`
> - `server/campaigns/test/campaign.json`
>
> These drafts now pressure-test:
> - top-level campaign metadata
> - room graph authoring
> - companion + actor definitions
> - dialogue asset references
> - authored node modes
> - readable condition/action syntax

The `test` campaign is now the preferred runtime POC target because its conditions are easy to reach and easier to debug.

Remaining gaps in the draft:

- exact Yarn asset files do not exist yet
- condition/action vocabulary is still provisional
- some counters and labels need final naming conventions

## Server POC Progress

> [!success]
> First campaign-backed server slice implemented:
> - embedded campaign asset loading added
> - `server/internal/campaigns` loader + validator implemented
> - `GET /api/campaigns` added to `http-games`
> - `POST /api/games` now accepts `campaign_id`
> - campaign-backed sessions now use a minimal bootstrap path instead of `world-gen`
> - focused local tests pass for `internal/campaigns` and `cmd/http-games`

> [!success]
> Campaign runtime persistence is now scaffolded in server code:
> - `CampaignRuntimeState` added to `game.Game` and `game.SaveState`
> - DynamoDB save/load mapping updated
> - campaign bootstrap seeds active node/objective/actor+companion runtime state
> - `GET /api/games/{id}` now returns a campaign runtime summary

This is not the full Phase 7 runtime yet.

It is the smallest AWS-testable slice that proves:

- authored campaign config can ship inside the Lambda binary
- the backend can expose campaign manifests
- the backend can create a campaign-backed session without procedural generation
