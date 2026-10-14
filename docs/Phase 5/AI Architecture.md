#AIArchitecture

## Migration: Gemini → Bedrock

Current code uses `google.golang.org/genai` with a `*genai.Chat` session pattern and freeform JSON parsing via regex. This is replaced by the AWS Bedrock `Converse` / `ConverseStream` APIs.

### Bedrock SDK (Go)
```go
import "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
import "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
```
- `Converse` — synchronous, returns full response. Used for world generation sub-agents.
- `ConverseStream` — streaming, returns `ConverseStreamOutput` with an event channel. Used for the narrator/chat. Each event is one of: `ContentBlockDelta` (text chunk), `ContentBlockStart` (tool use begins), `ContentBlockStop`, `MessageStop`.
- Tool calling: pass `toolConfig` with `types.Tool` definitions. The model returns `toolUse` content blocks natively — no regex JSON parsing needed.
- Conversation history: `[]types.Message` with `Role: "user" | "assistant"`. Tool use results are returned as `toolResult` blocks in a subsequent user message (Bedrock follows the Anthropic tool use protocol).

### Porting mcp.go → internal/ai/
The existing 13 tools (`create_room`, `create_item`, etc.) map directly to Bedrock `types.Tool` definitions. The `ExecuteTool` switch statement becomes `ExecuteToolCall(toolName string, input map[string]interface{}) (string, error)` and is otherwise structurally identical.

The two-step prompt pattern in `HandlerChat` (planning stage → execution stage returning JSON) is simplified: with native tool calling, one `ConverseStream` call handles both narrative and tool use in a single pass. The AI streams narrative text and may intersperse `toolUse` blocks that the Lambda executes between chunks.

Stored conversation history: `Narrative []types.Message` replaces `[]*genai.Content`. **All existing DynamoDB game saves are incompatible** — they store Gemini's `Content` type. Since this is alpha, the migration path is: increment `schema_version` to `2` and discard/reset any `schema_version=1` records (or simply wipe the sessions table on initial deployment).

---

## Responsibilities
- Narrative
  - Responding to the player with a cohesive narrative
  - Routinely compress the narrative history so the context window stays manageable (summarize older turns into a single assistant message)
- World Management
  - Moving characters to and from locations
  - Moving/Creating items in response to narrative
  - Creating new rooms and connecting them as needed
  - Removing world resources

## Design: Closed-Ended One-Shot

The game is close-ended — a one-shot quest with a clear defined goal, multiple paths to success, and a finite set of rooms and puzzles. This bounds the AI's world building scope and makes testing tractable.

## World Building Flow

### Architect Agent
- Model: `anthropic.claude-sonnet-4-6` (via Bedrock)
- Invoked once at game creation (inside the async `world-gen` Lambda).
- Has **no tool access** — pure generation.
- Input: player name, optional theme hint.
- Output: a structured description block covering:
  - World theme and setting.
  - The one-shot quest goal and multiple success paths.
  - A list of named locations (rooms) with brief descriptions and connections.
  - A list of key characters and their roles.
  - A list of key items and their purposes.
  - The opening scene narrative.
- This output is passed verbatim to the Engineer Agent as its first message.

### Engineer Agent
- Model: `anthropic.claude-sonnet-4-6` (via Bedrock)
- Invoked once, immediately after the Architect completes.
- Scans the Architect's description and decomposes it into ordered tasks for sub-agents.
- Has access to:
  - `createTask(agent: "world"|"npc"|"item", instructions: string) → string` — dispatches a sub-agent synchronously and returns its result summary (< 180 chars).
  - `queryTasks(query: string) → TaskRecord[]` — queries the in-memory task log for this world generation run. Supports filtering by `agent` and keyword matching on `instructions`/`response`. (Full SQL is overkill — implement as a simple filtered scan over a `[]TaskRecord` slice.)
  - `rejectTask(uuid: string) → TransactionLog` — reverts the world mutations made by a specific task. Requires the game engine to support transactional rollback per task (implement via a task-scoped mutation log: record each tool call made during a task so it can be reversed).
- The Engineer loops until all tasks are dispatched and the world matches the Architect's blueprint.

### Sub-Agents
Each is a synchronous Bedrock `Converse` call (not streaming) with a focused tool set.

**World Builder sub-agent**
- Model: `anthropic.claude-haiku-4-5`
- Tools: `createRoom`, `updateRoom`, `deleteRoom`, `connectRooms`

**NPC Creator sub-agent**
- Model: `anthropic.claude-haiku-4-5`
- Tools: `createCharacter`, `updateCharacter`, `setCharacterLocation`

**Item Creator sub-agent**
- Model: `anthropic.claude-haiku-4-5`
- Tools: `createItem`, `updateItem`, `setItemLocation`

### Task Log
In-memory during the `world-gen` Lambda execution (not persisted to DynamoDB mid-generation):
```go
type TaskRecord struct {
    UUID         string
    Agent        string
    Instructions string
    Response     string
    Mutations    []ToolCall  // ordered list of tool calls made, for rollback
    RejectedAt   *time.Time
}
```
After world generation completes, only the final game state is persisted — not the task log.

### Context Compression (Narrator)
The narrator (`ws-chat` Lambda) maintains `[]types.Message` as conversation history. Before each Bedrock call, if the history exceeds a token threshold (check via `ContentTokenCount` from a prior response, or use a rough character count heuristic), compress older messages by sending the history to a summarization call and replacing the summarized turns with a single `assistant` summary message. This keeps the context window from growing unboundedly across long game sessions.

## Vision
The Narrative agent is the main orchestrator of the game and drives decisions and plot. The narrator session begins with a context compression summary from the World Generation architect output, followed by the initial narrative prompt. The world generation summary sticks to high-level world detail (not room-by-room).

## Future: Campaign Templates
It could help to collect templates of pre-guided D&D campaigns and have the Architect build around them rather than generating from scratch. This would improve coherence and give players recognizable story structures to engage with.
