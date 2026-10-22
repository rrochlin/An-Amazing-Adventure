# TODO — An Amazing Adventure

Persistent backlog of identified work. Each item captures phase, context, decisions made, and open questions.
Updated as work progresses. Checked items are done and merged to `main`.

---

## Current Direction (as of 2026-06-24)

- Phase 7 implementation is largely complete in `main` (types, mutation persistence, ws-chat wiring, client events UI).
- Phase 8 architecture (Narrator + Engineer split) is also implemented in `main`.
- The highest-value next work is test hardening for the new turn flow and env-var guards.

### Next item to work on

1. Start **Milestone A** in `plan.md`: interactive Yarn choice support on the `test` campaign.

---

## How to read this

- **Phase** — the logical grouping (Infrastructure / Backend / Frontend / AI)
- **Status** — `[ ]` pending · `[~]` in progress (has open PR) · `[x]` done (merged)
- **Branch** — feature branch or PR reference where applicable
- **Decisions** — choices already made that should not be re-litigated without good reason
- **Open** — questions still to be answered before or during implementation

---

## Active PRs

No active PRs at the moment.

Recently merged:
- `An-Amazing-Adventure` PR #61 — fix: preserve chat state and correct hidden-probe scan gating
- `An-Amazing-Adventure` PR #18 — fix: correct WEBSOCKET_API_ENDPOINT double-stage in CLAUDE.md
- `An-Amazing-Adventure` PR #17 — fix: WebSocket chat, streaming bubble, map spacing
- `terraform-infrastructure` PR #10 — fix(infra): remove double-stage from WEBSOCKET_API_ENDPOINT (closed)

---

## Phase 6 — Bug Fixes & UI Polish

### [x] WS-1 · WebSocket chat broken in production
**Branch:** `feat/ws-fix-chat-ui` (PR #17, merged)
**Root causes identified and fixed:**
1. `wsutil.New()` was not prepending `https://` to the schemeless `WEBSOCKET_API_ENDPOINT` env var → `unsupported protocol scheme ""` on every `PostToConnection` call. Fixed in `server/internal/wsutil/wsutil.go`.
2. `WEBSOCKET_API_ENDPOINT` env var was `domain/prod/prod` (double-stage) because the `api-gateway` module output already includes the stage and `lambdas/main.tf` appended it again. Fixed via Terraform (PR #10) + `terraform apply` already run in production.
**Cleanup done:** Deleted 3 stale IAM drift policies (`game-chat-permissions` on ws-chat, `dynamo-connections` on ws-connect/ws-disconnect). Deleted 2 orphaned API Gateway instances from a previous Terraform apply (`26xtekh6ek` WS, `d3szg6bpc0` HTTP).

---

### [x] UI-1 · Streaming chat bubble
**Branch:** `feat/ws-fix-chat-ui` (PR #17, merged)
**What:** When the AI streams a narrative response, display a dedicated `StreamingChatMessage` bubble immediately rather than the shimmer-only `LoadingMessage`. The bubble shows a Cinzel shimmering flavor-text header ("The tale unfolds...") and the streaming text below with a blinking gold cursor. Switches to `react-markdown` rendering on `narrative_end`.
**Decision:** Render as plain text while streaming (avoids broken markdown mid-stream). Switch to `react-markdown` only after the message finalizes.
**Files:** `client/src/components/Chat.tsx`, `client/src/routes/game-{$sessionUUID}.tsx`

---

### [x] UI-2 · Map node spacing too compact
**Branch:** `feat/ws-fix-chat-ui` (PR #17, merged)
**What:** `worldUnitToPx` increased from 70→110px (corridor gap 20→66px). `baseRoomSize` reduced from 50→44px to keep map from sprawling while giving rooms room to breathe.
**File:** `client/src/components/RoomMap.tsx`

---

### [x] UI-3 · Duplicate chat messages after stream ends
**Context:** Two code paths both add the finalized narrative to `chatMessages`:
1. `narrative_end` → `finalizeStreamingMessage()` appends `{type:"narrative", content: streamingMessage}` to `chatMessages`.
2. `state_delta` → `applyDelta()` appends `delta.new_message` to `chatMessages`.
The server sends both frames. The result is every AI response appearing twice.
**Decision:** Remove `NewMessage` from the `StateDelta` returned by `ws-chat`. The client already has the text from the stream. `state_delta` is for world state only (room/player). Remove `new_message` handling from `applyDelta` on the client side.
**Files:**
- `server/cmd/ws-chat/main.go` — remove `delta.NewMessage` assignment
- `client/src/store/gameStore.ts` — remove `new_message` branch from `applyDelta`
- `client/src/routes/game-{$sessionUUID}.tsx` — confirm/remove stale `displayMessages` spread
**Done:** `state_delta` no longer appends narrative text on the client (`new_message` path removed) and ws-chat uses streaming narrative + world-state delta separation.

---

### [x] UI-4 · Player message disappeared (intermittent)
**Context:** Player message is added optimistically via `addChatMessage` in the game route. If `setGameState` fires (from a `game_state_update` frame) it resets `chatMessages` from `state.chat_history`, wiping the optimistically-added message. Most likely this is a visual flicker caused by the duplicate message bug (UI-3) rather than a true disappearance — fixing UI-3 likely resolves this.
**Done:** Chat reconciliation preserves optimistic player messages when server state refreshes.

---

## Phase 7 — World Events & Mutation Log

This phase implements two distinct features that share infrastructure:
1. A **mutation log** — durable, machine-readable audit trail of all game state changes
2. **Player-facing world events** — filtered, human-readable signals of what the player can observe, attached to chat history and rendered in a collapsible UI accordion

### Context

Every game state mutation already flows through `DispatchTool` in `server/internal/ai/tools.go`. This single chokepoint is where both the log entries and player-visible events should originate — through code, not agent reasoning. The tool dispatch layer knows exactly what happened with precision (who was damaged, by how much, what their new health is) and can generate accurate, player-perspective messages deterministically.

### Decisions

- **Mutation log storage:** Separate DynamoDB table (`amazing-adventure-prod-mutations`), not embedded in the game session item. Avoids DynamoDB 400KB item limit. PK: `session_id` (B type, matches sessions table). SK: `ts` (N, unix milliseconds — unique within a Lambda execution since tool calls are sequential).
- **Player events persistence:** Events are attached to the `narrative` `ChatMessage` in `ChatHistory` (new optional `events []WorldEvent` field on `ChatMessage`). This means events survive reconnection/reload — they're part of the permanent chat record.
- **Client display:** Collapsible MUI Accordion below each narrative bubble when events are non-empty. Collapsed by default. Label: "World events (N)".
- **Visibility filter:** Code-determined, not AI-determined. Uses `g.Player.LocationID` at dispatch time to decide if the player can observe the mutation. See visibility table below.
- **Event messages:** Hardcoded Go templates, not AI-generated. Examples: "You take 12 damage. ❤ 73/100" / "Iron Key added to your inventory." / "Goblin has been slain."
- **saveStateDB drift fix:** Included in this phase. `saveStateDB` in `db.go` is missing `Title`, `Theme`, `QuestGoal`, `TotalTokens`, `ConversationCount`, `CreationParams` — these are stored but not loaded back. Fixes the game details page silently showing empty data.
- **Narration quality issue (AI describing tool calls) is NOT in this phase.** That requires the Narrator+Engineer split and is tracked separately in Phase 8.

### Visibility filter

| Tool | Player sees event if... |
|---|---|
| `damage_character` (player) | Always |
| `damage_character` (NPC) | NPC's room == player's room |
| `heal_character` (player) | Always |
| `heal_character` (NPC) | NPC's room == player's room |
| `set_character_alive` false (player) | Always |
| `set_character_alive` false (NPC) | NPC's room == player's room |
| `set_character_alive` true (revive) | NPC's room == player's room |
| `give_item_to_player` | Always |
| `take_item_from_player` | Always |
| `create_item` (placed in player room) | Always |
| `create_item` (placed elsewhere) | Never |
| `place_item_in_room` (player's room) | Always |
| `place_item_in_room` (elsewhere) | Never |
| `create_character` (placed in player room) | Always |
| `create_character` (elsewhere) | Never |
| `move_character` (arrives at player room) | Always |
| `move_character` (departs from player room) | Always |
| `move_character` (neither room) | Never |
| `update_room` (player's room) | Never (description change visible in next narrative) |
| `create_room` | Never |
| `get_room_info` | Never (read-only) |

### Tasks

#### INFRA-1 · New mutations DynamoDB table
**Status:** `[x]`
**What:**
- Add `aws_dynamodb_table.mutations` to `dynamodb/main.tf`: hash key `session_id` (B), sort key `ts` (N), PAY_PER_REQUEST. Optional: TTL on `ts` (90-day expiry).
- Add outputs: `mutations_table_name`, `mutations_table_arn`
- Add `MUTATIONS_TABLE` env var to `ws-chat` and `world-gen` Lambdas in `lambdas/main.tf`
- Add `dynamodb:PutItem` on mutations ARN to `ws-chat` and `world-gen` IAM policies
- Wire new dynamodb outputs through root `main.tf` to lambdas module
- Subtree-push to `terraform-infrastructure`, open PR, apply locally after merge
**Open:** TTL policy still optional; current table is active without TTL expiry.

#### EVENTS-1 · Core types in game package
**Status:** `[x]`
**What (server/internal/game/game.go):**
```go
type WorldEvent struct {
    Type    string `json:"type"`    // "damage", "heal", "death", "revive", "item_gained",
                                    // "item_lost", "item_appeared", "character_arrived",
                                    // "character_departed", "room_changed"
    Message string `json:"message"` // human-readable, player's perspective
}

// Extend ChatMessage:
type ChatMessage struct {
    Type    string       `json:"type"`            // "player" | "narrative"
    Content string       `json:"content"`
    Events  []WorldEvent `json:"events,omitempty"`
}

// Extend StateDelta (remove NewMessage, add Events):
type StateDelta struct {
    CurrentRoom  *RoomView           `json:"current_room,omitempty"`
    Player       *CharacterView      `json:"player,omitempty"`
    UpdatedRooms map[string]RoomView `json:"updated_rooms,omitempty"`
    Events       []WorldEvent        `json:"events,omitempty"`
    // NewMessage removed — duplicate fix (see UI-3)
}

// New: MutationEntry for the audit log
type MutationEntry struct {
    SessionID string         `json:"session_id" dynamodbav:"session_id"`
    Ts        int64          `json:"ts"         dynamodbav:"ts"`
    Turn      int            `json:"turn"       dynamodbav:"turn"`
    Tool      string         `json:"tool"       dynamodbav:"tool"`
    Input     map[string]any `json:"input"      dynamodbav:"input"`
    Result    string         `json:"result"     dynamodbav:"result"`
}
```
**Also:** Fix `saveStateDB` drift — add `Title`, `Theme`, `QuestGoal`, `TotalTokens`, `ConversationCount`, `CreationParams` to `saveStateDB` and update `toDBState` / `fromDBState`.

#### EVENTS-2 · DispatchTool returns WorldEvent + MutationEntry
**Status:** `[x]`
**What (implemented in `server/internal/ai/tools.go` + `server/internal/ai/bedrock.go`):**
- Change `DispatchTool` signature: `func DispatchTool(g *game.Game, name string, input map[string]any) (result string, event *game.WorldEvent, err error)`
- Each `exec*` function returns `(string, *game.WorldEvent, error)`
- Visibility filter uses `g.Player.LocationID` inline — no helper needed
- Message templates are Go format strings, hardcoded per tool+target combination
- `EngineerScan` call site handles new return values; accumulates `[]WorldEvent` and `[]MutationEntry`

#### EVENTS-3 · db.PutMutation
**Status:** `[x]`
**What (server/internal/db/db.go):**
- Add `mutationsTable string` field to `Client`
- Load from `MUTATIONS_TABLE` env var in `New()`
- Add `requireMutationsTable()` guard
- Add `PutMutation(ctx context.Context, entry game.MutationEntry) error`
- `MutationEntry.SessionID` must marshal as `BinaryID` — use a `mutationEntryDB` wrapper struct (same pattern as `saveStateDB`)

#### EVENTS-4 · ws-chat: wire events into turn flow
**Status:** `[~]`
**What (server/cmd/ws-chat/main.go):**
1. After `NarrateStream` returns, `result.Events` contains all visible player events for the turn
2. Batch-write `MutationEntry` items to mutations table (sequential `PutMutation` calls — small count, no batch write needed)
3. Populate `delta.Events = result.Events`
4. Remove `delta.NewMessage` (fixes UI-3)
5. Append `result.Events` to the `narrative` `ChatMessage` before appending to `ChatHistory`: `history = append(history, game.ChatMessage{Type: "narrative", Content: result.Narrative, Events: result.Events})`
6. `MUTATIONS_TABLE` env var required — add explicit startup-time guard in ws-chat handler path (currently enforced when `PutMutation` executes)

#### EVENTS-5 · Client: types + store + socket
**Status:** `[x]`
**What:**

`client/src/types/types.ts`:
```typescript
export interface WorldEvent {
  type: string;
  message: string;
}
// Extend ChatMessage:
events?: WorldEvent[];
// Extend StateDelta:
events?: WorldEvent[];
// Remove new_message from StateDelta
```

`client/src/store/gameStore.ts`:
- Remove `new_message` handling from `applyDelta` (fixes UI-3)
- Add `attachEventsToLastMessage(events: WorldEvent[])` action: patches `events` onto the last `chatMessages` entry if it is type `"narrative"`

`client/src/hooks/useGameSocket.ts`:
- In `state_delta` handler: after calling `applyDelta(delta)`, if `delta.events?.length`, call `attachEventsToLastMessage(delta.events)`
- **Ordering note:** `narrative_end` fires before `state_delta`, so `finalizeStreamingMessage` (which commits the message) always runs before `attachEventsToLastMessage` (which patches events onto it). The ordering is correct.

#### EVENTS-6 · Client: WorldEventsAccordion in Chat.tsx
**Status:** `[x]`
**What (client/src/components/Chat.tsx):**
- Add `WorldEventsAccordion` component: MUI `Accordion` with `defaultExpanded={false}`
- Summary label: "World events ({n})" in small Cinzel text, muted gold
- Each event row: event message string in Crimson Text, small font
- Render below the `Paper` bubble in `ChatMessage` when `message.events?.length > 0`
- On reconnect/reload, events are loaded from `ChatHistory` via `setGameState` → present in `chatMessages` from the start

#### EVENTS-7 · Tests
**Status:** `[x]`
**What:**
- Go: `DispatchTool` unit tests — `damage_character` on player always returns event; on NPC in different room returns nil; on NPC in same room returns event. `PutMutation` db test (with `t.Setenv`). `ws-chat` handler test updated for new `NarratorResult` shape and removed `delta.NewMessage`.
- TS: `gameStore` — `applyDelta` does NOT add to `chatMessages`; `attachEventsToLastMessage` patches events onto last narrative message. `Chat.test.tsx` — accordion renders when `events` non-empty; does not render when empty.

**Done:** Added ws-chat tests for `delta.events` propagation and mutation-write behavior on the non-deterministic path, plus explicit `MUTATIONS_TABLE` env-var expectation coverage strategy.

---

## Phase 8 — AI Narration Quality (Narrator + Engineer Split)

### Context

The narrator currently has access to all 16 world-mutation tools and runs an agentic loop where it both produces narrative prose and calls tools. This causes the model to describe its own planning ("I will now move the goblin...") rather than writing immersive prose. The model conflates DM planning reasoning with player-facing text.

### Proposed Architecture

**Two-step turn inside `ws-chat`:**

1. **Narrator** (`claude-sonnet-4-6`, streaming, zero tools) — Given game state + player input + conversation history, produces pure narrative prose only. Streams to the client exactly as today. Cannot mention tool calls because it has none.

2. **Engineer** (`claude-haiku-4-5`, synchronous, all 16 tools) — Given the finished narrative text + current game state (room contents, player inventory, NPC list), determines what world mutations are implied by the narrative and executes them. Returns updated game state + tool call summaries. Runs after `narrative_end` is sent, so the client sees no added latency in the narrative stream.

**Why this works:**
- Narrator writes prose by construction — it has no tools to plan around
- Engineer is focused: "execute the mutations implied by this text" — a structured extraction task that Haiku handles well
- Tool call summaries from the Engineer feed directly into the WorldEvents layer (Phase 7) — event generation moves to the Engineer's dispatch rather than the Narrator's loop
- Conversation history stays clean: only narrative text enters the history, never tool call JSON

### Decisions made

- **Narrator has zero tools** — enforced by removing `ToolConfig` from the `ConverseStreamInput`
- **Engineer uses Haiku** — fast, cheap, focused on structured extraction
- **Engineer receives both narrative and game state context** — allows accurate mutation detection (knows what items/NPCs exist to reference by name)
- **Tool call summaries are ephemeral for now** — not yet surfaced to the client beyond the world events accordion (already handled by Phase 7). The "thinking steps" collapsible showing raw tool names can come later if desired.
- **Conversation history change:** The Narrator's `NarrativeMessage` history stores only text turns. The Engineer's tool calls are never stored in history (they'd pollute the narrative context with JSON).

### Risks

- **Engineer may miss mutations** — if narrative says "the bridge collapses" the Engineer must infer `update_room` or `delete_room`. Haiku may miss subtle implications. Mitigated by: explicit prompt with examples, and the fact that missed mutations are survivable (narrative still delivered; state is wrong but not catastrophic).
- **Latency after `narrative_end`** — Engineer adds ~1-3s before `state_delta` is sent. Phase 7's world events accordion fills this gap visually (player sees events populating after narrative ends).
- **Prompt engineering required** — Engineer prompt needs to be carefully tuned. Plan to iterate.

### Tasks

#### AI-1 · NarrateStream: remove tools
**Status:** `[x]`
**What (server/internal/ai/bedrock.go):**
- Remove `ToolConfig` from `ConverseStreamInput` in `NarrateStream`
- Remove the tool execution block from the agentic loop (loop now ends immediately — Narrator never calls tools)
- Update system prompt: remove "Use the provided tools to mutate the game world as the story demands. Call tools when the world should change." Replace with: "Write immersive, engaging prose narration only. Never describe what you are about to do — only narrate what is happening in the story."
- `NarratorResult` history remains text-only (no tool_use/tool_result blocks are emitted by NarrateStream).

#### AI-2 · New EngineerScan function
**Status:** `[x]`
**What (server/internal/ai/bedrock.go):**
```go
func (c *Client) EngineerScan(
    ctx context.Context,
    g *game.Game,
    narrative string,
) (EngineerResult, error)
```
- Synchronous `Converse` call using `ModelSubAgent` (Haiku)
- Has all 16 tools from `NarratorTools()`
- System prompt: "You are a game world engineer. Read the narrative below and execute the world mutations it implies. Call tools only — do not output narrative text. Do not invent mutations not clearly implied by the narrative."
- User message includes: the narrative text + compact game state (current room name, room items, player inventory, NPCs with health/location)
- On response: execute tool calls via `DispatchTool`, collect `WorldEvent` and `MutationEntry` per call
- Returns accumulated events and mutations

#### AI-3 · ws-chat: wire Engineer into turn flow
**Status:** `[x]`
**What (server/cmd/ws-chat/main.go):**
1. `NarrateStream` — streams narrative (no tool calls)
2. `ws.SendNarrativeEnd`
3. `EngineerScan(ctx, g, result.Narrative)` — executes world mutations
4. Batch-write `MutationEntry` items (replaces Phase 7's mutation write from the Narrator loop)
5. Populate `delta.Events` from Engineer events
6. `ws.SendDelta`

#### AI-4 · Tests
**Status:** `[x]`
**What:**
- `NarrateStream` unit test: confirm no tool calls in output
- `EngineerScan` unit test: mock Bedrock response with tool calls, verify `g` is mutated and events returned
- `ws-chat` handler test: updated for two-step flow

**Done:** Added focused `EngineerScan` tool-loop unit coverage (mocked Bedrock Converse rounds + tool dispatch assertions) and updated ws-chat handler coverage for two-step flow expectations.

---

## Phase 9 — Future UI Improvements

Captured from `UI.md` notes and RoomMap.tsx TODO comments. No decisions made yet — these are ideas for future sprints.

### UI-FUT-1 · Map: pop-out/modal expand
Expand the map to fill more screen real estate on demand. Button in map panel header opens a modal with a larger canvas.

### UI-FUT-2 · Map: auto-zoom to current room
When the player moves, auto-pan and zoom the map canvas to keep the current room visible and centered.

### UI-FUT-3 · Map: highlight connected exits
Highlight adjacent rooms (connected to current) in a distinct color (e.g. teal glow) so the player can see movement options on the map.

### UI-FUT-4 · Map: room panel correlation
Hovering or clicking a map node updates the Room Info panel on the right to show that room's details, not just the current room.

### UI-FUT-5 · Map: character tokens
Render NPC/character tokens on the map at their room positions so the player can see where enemies and allies are.

### UI-FUT-6 · Map: fog-of-war persistence
Currently only the current room and its direct connections are revealed. Persist a "visited" set in game state so rooms stay revealed after the player leaves them.

### UI-FUT-7 · Chat: per-message timestamps
Small timestamp on each chat message, shown on hover.

### UI-FUT-8 · Chat: reconnection indicator
Show a toast/banner when the WebSocket reconnects after a drop, so the player knows they're back.

---

## Completed

### [x] Phase 5 — Game Creation Refactor (PR #16, merged)
- Multi-step `/create` wizard (character details + adventure prefs)
- Retired REST `/api/worldready` polling; WebSocket `world_gen_ready` delivery
- `/game-{uuid}/details` page: character, quest, preferences, stats
- Blueprint metadata: title, theme, quest goal, token usage
- PR #9 (terraform-infrastructure): removed dead `GET /api/worldready/{uuid}` route
