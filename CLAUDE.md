# CLAUDE.md

Guidance for AI assistants working in this repository.

## Project Overview

An AI-powered text adventure game. Claude (via AWS Bedrock) acts as the Dungeon Master. The game features dynamic world generation, real-time streaming narrative over WebSocket, and persistent game state in DynamoDB.

**Current Status:** Alpha — core loop working in production.

## Tech Stack

### Backend (`server/` — Go 1.24, AWS Lambda)
- **AI:** AWS Bedrock — `us.anthropic.claude-sonnet-4-6` (narrator + narrative framing), `us.anthropic.claude-haiku-4-5-20251001-v1:0` (sub-agents)
- **Database:** AWS DynamoDB (sessions + connections tables)
- **Auth:** AWS Cognito User Pools (JWT, SRP flow)
- **Transport:** API Gateway HTTP (REST) + API Gateway WebSocket

### Frontend (`client/` — React 19 + TypeScript)
- **Build:** Vite
- **Routing:** TanStack Router (file-based)
- **UI:** Material UI (MUI) — dark fantasy theme (Cinzel/Crimson Text, gold/purple)
- **State:** Zustand (`gameStore.ts`)
- **HTTP:** Axios with Cognito JWT auth header + 401 refresh interceptor
- **WebSocket:** custom `useGameSocket` hook (handles both game frames and world-gen progress)
- **Testing:** Vitest + Testing Library

### Infrastructure
- **Secrets:** Doppler CLI (local dev only)
- **Deployment:** GitHub Actions → S3/CloudFront (client) + Lambda ZIP (server)
- **IaC:** Terraform lives in a separate repo, `rrochlin/terraform-infrastructure` (not in this repo)
- **Region:** `us-west-2`, Account: `292826404083`
- **CloudFront:** `d1ctll9l3g8cf4.cloudfront.net`

## Repository Layout

```
An-Amazing-Adventure/
├── docs/                    # Design notes by topic + decisions.md; phase-era notes in docs/archive/
├── client/                  # React frontend
│   ├── src/
│   │   ├── routes/          # TanStack file-based routes
│   │   ├── components/      # Chat, GameInfo, RoomMap, WorldGenTerminal, ...
│   │   ├── hooks/           # useGameSocket
│   │   ├── services/        # api.service, api.game, api.users, auth.service
│   │   ├── store/           # gameStore.ts (Zustand)
│   │   └── types/           # types.ts
│   └── vite.config.ts
├── server/                  # Go Lambda handlers
│   ├── cmd/
│   │   ├── http-games/      # GET/POST/DELETE /api/games
│   │   ├── http-users/      # PUT /api/users (profile update via Cognito)
│   │   ├── world-gen/       # Async Lambda: procedural world-gen + narrative framing → WS progress
│   │   ├── ws-connect/      # WebSocket $connect
│   │   ├── ws-disconnect/   # WebSocket $disconnect
│   │   ├── ws-chat/         # WebSocket chat action → Bedrock streaming
│   │   └── ws-game-action/  # WebSocket game_action (move/pick_up/drop/equip/unequip)
│   └── internal/
│       ├── ai/              # Bedrock client, NarrateStream, EngineerScan, GenerateNarrativeFraming
│       ├── db/              # DynamoDB client (BinaryID type for B-typed keys)
│       ├── game/            # Game engine (Area, Character, Item, Game, SaveState)
│       └── wsutil/          # WebSocket frame push helpers
```

## Design Notes

Planning happens in the repo's GitHub Project (Kanban). Large outcomes are **epics** (issues labelled
`epic` with sub-issues); research that needs a decision first is a **spike** (`type:spike`). Labels carry
`type:*`, `area:*`, `priority:P0–P2` and `size:S/M/L`. Before starting work, find or create the issue, and
link PRs to it (`Fixes #N`).

`docs/` holds design notes by topic (Obsidian-style `[[wikilinks]]`; git is the source of truth, and the
Obsidian vault folder `~/Documents/notes/Amazing Adventure` is a symlink to it). `docs/decisions.md` is the
decision log, newest first — record a decision there when an issue or spike settles one. `docs/README.md`
explains the layout. Phase-era notes are in `docs/archive/`; its Phase 7 campaign data model and schema
notes still describe the current runtime.

## Common Commands

### Client
```bash
cd client
pnpm dev          # Dev server on :5173
pnpm build        # Production build (Vite + tsc)
pnpm test         # Vitest (all tests)
pnpm test --run   # Vitest (single run, no watch)
```

### Server
```bash
cd server
go build ./...
go test ./...
go run ./tools/yarnc   # compile campaign dialogue: campaigns/**/*.yarn → .yarnc + .csv
go run ./tools/sessiondump SESSION_ID   # turn-by-turn debugging report (see Debugging Sessions)
```

### Terraform
Terraform is not in this repo. It lives in `rrochlin/terraform-infrastructure`,
under `amazing-adventure/`. Clone that repo to make infrastructure changes.

## Architecture

### Game Flow
1. User fills out `/create` wizard (D&D character + adventure prefs) → `POST /api/games` with `CharacterCreationData` → returns `session_id`
2. Client navigates immediately to `/game-{uuid}`; `useGameSocket` connects; `WorldGenTerminal` shown inline
3. `http-games` fires `world-gen` Lambda async with all creation params
4. `world-gen` generates dungeon layout procedurally (rpg-toolkit), calls `GenerateNarrativeFraming`, then persists `Title`/`Theme`/`QuestGoal`/`TotalTokens`/`CreationParams` to `SaveState`
5. `world-gen` emits `world_gen_log` frames → terminal; then `world_gen_ready` → `useGameSocket.onWorldReady` → `LoadGame` refetch → render game
6. Chat messages → `ws-chat` → Bedrock streaming → `narrative_chunk` frames; increments `ConversationCount` + `TotalTokens` on save
7. Game actions (move/pick_up/equip) → `ws-game-action` → state delta frames
8. `/game-{uuid}/details` shows persisted `Title`, `Theme`, `QuestGoal`, `CreationParams`, and stats

### DynamoDB Key Types
Both tables use **Binary (`B`) type** for key attributes (`session_id`, `user_id`, `connection_id`).
The `BinaryID` type in `internal/db/binaryid.go` handles marshaling strings as `B`.
Never use `attributevalue.Marshal(stringVal)` directly for key fields — it produces `S` type and DynamoDB rejects it.

### WebSocket Frames (server → client)
| Type | When |
|---|---|
| `narrative_chunk` | Streaming text from Bedrock |
| `narrative_end` | Streaming complete |
| `game_state_update` | Full state snapshot |
| `state_delta` | Partial update (room/player changed) |
| `world_gen_log` | World-gen progress line |
| `world_gen_ready` | World generation complete |
| `error` | Server error to surface to user |
| `streaming_blocked` | Rejected message (already streaming) |

### Bedrock Model IDs
Must use cross-region inference profile IDs (with `us.` prefix):
- `us.anthropic.claude-sonnet-4-6` — narrator and narrative framing
- `us.anthropic.claude-haiku-4-5-20251001-v1:0` — sub-agents
Bare model IDs without prefix are rejected with `ValidationException`.

## Working Process

### Starting new work — MANDATORY first step
**Before creating a feature branch or writing any code, always sync with the remote:**

```bash
git fetch origin
git log --oneline origin/main -5   # confirm any recently merged PRs are visible
git checkout -b feat/<short-name> origin/main
```

### After merge cleanup — MANDATORY
**When a PR is merged and you are about to start new work, always reset your local branch state first:**

```bash
git checkout main
git pull --ff-only origin main
git branch -d feat/<old-branch>
git checkout -b feat/<new-branch>
```

This avoids stacking work on stale branches and keeps each PR scoped to one section of work.

**Why this matters:** If a previous PR was merged while the last session was running, your local `main` is stale. Starting a branch from stale `main` means your new branch includes the previous branch's commits. When you later rebase/PR, git sees those commits as divergent (not as the squash-merge from main) and either raises conflicts or duplicates the work.

**If you forgot and already have commits on a stale branch:**
```bash
# Identify your own commits (the ones NOT in origin/main)
git log --oneline origin/main..HEAD

# Create a clean branch and cherry-pick only your commits
git checkout -b feat/<short-name>-rebased origin/main
git cherry-pick <sha1> <sha2> ...   # your commits only

# Push and open a new PR; close the stale one
git push -u origin feat/<short-name>-rebased
gh pr close <old-pr-number> --comment "Superseded by rebased branch"
gh pr create ...
```

### Commit cadence
**Commit early and often — do not accumulate a large diff before committing.**

Every logical step of work should be its own commit before moving to the next:
1. Data model changes (Go structs / TS types)
2. Backend API / Lambda handler changes
3. Frontend routes / components
4. Tests and cleanup

A good signal: if `git diff --stat` shows more than ~10 files or ~300 lines, you've waited too long. Typical commit sequence for a feature:
```
feat: extend SaveState with new metadata fields          ← data model
feat: update API handlers and world-gen Lambda           ← backend
feat: new /create wizard and game details page           ← frontend
chore: update tests, gitignore, CLAUDE.md                ← cleanup
```

Always create a feature branch (`git checkout -b feat/...`) before starting any non-trivial work.

### Deploy / PR workflow
- **This monorepo** (client + server): open a PR to `main` via `gh pr create`. GitHub Actions runs tests on every PR and deploys on merge to `main`.
- **Infrastructure changes** (new AWS resources, API Gateway routes, Lambda env vars, DynamoDB tables, etc.): these do **not** live in this repo. They live in `rrochlin/terraform-infrastructure`, under `amazing-adventure/`. Clone that repo and open a PR there. Do not look for Terraform here — there is none.
  - Merging an infra PR **applies nothing**. Apply is an explicit dispatch:
    ```bash
    gh workflow run terraform.yml -f app=amazing-adventure
    ```
  - The CI plan only covers app directories that changed; edits to shared `modules/` or the workflow widen it back to every app.
- **Never commit compiled Lambda binaries** (`server/http-games`, `server/world-gen`, etc.) — they are in `.gitignore`. CI builds them from source on every deploy.

### Infrastructure discipline — MANDATORY
**All infrastructure changes must go through Terraform, in `rrochlin/terraform-infrastructure`.** Never use the AWS CLI or Console to create or modify resources (IAM policies, Lambda env vars, DynamoDB tables, API Gateway routes, Cognito config, etc.). Direct AWS changes create drift that is invisible to Terraform and breaks future applies.

**AWS CLI is permitted only for:**
- Reading logs (`aws logs filter-log-events ...`)
- Inspecting current state for debugging (`aws lambda get-function-configuration`, `aws dynamodb describe-table`, etc.)
- One-time data operations (`aws dynamodb put-item` to seed a record, etc.)

**When adding a new Lambda or table**, the Terraform half of the work happens in `rrochlin/terraform-infrastructure` (paths below are relative to `amazing-adventure/`):
1. Add the `aws_dynamodb_table` resource + outputs to `modules/dynamodb/main.tf`
2. Add the `aws_iam_role` + `aws_iam_role_policy` + `aws_lambda_function` in `modules/lambdas/<lambda-name>.tf` (one file per Lambda), with shared variables declared in `modules/lambdas/main.tf`
3. Add integrations, routes, permissions to `modules/api-gateway/main.tf` if HTTP-facing
4. Wire new outputs through `main.tf`
5. Open a PR on `rrochlin/terraform-infrastructure`, then dispatch the apply — merging does not apply

**In this repo**, add `handler_test.go` with `TestAllRequiredEnvVarsPanic` covering every env var the new Lambda reads. This is the primary guard against drift between the code and the Terraform that is no longer co-located with it.

## CI/CD

Two GitHub Actions workflows:
- `deploy-client.yml` — test (PR+push) + deploy to S3/CloudFront (main only)
- `deploy-server.yml` — test (PR+push) + build arm64 zip + update Lambda (main only)

Branch protection: all changes must go through PRs. Direct push to `main` is blocked.

## Key Known Issues / Discoveries

- **BinaryID**: sessions and connections tables have B-typed keys. See `internal/db/binaryid.go`.
- **CONNECTIONS_TABLE**: `http-games` and `world-gen` don't need it at startup — `db.New()` defers the panic to actual connection methods via `requireConnectionsTable()`.
- **Inference profiles**: Bedrock requires `us.` prefixed profile IDs, not bare model IDs.
- **WebSocket endpoint**: `WEBSOCKET_API_ENDPOINT` env var is `domain/stage` (e.g. `ba2t50m7se.execute-api.us-west-2.amazonaws.com/prod`). The api-gateway Terraform module output already includes the stage — do not append it again. `wsutil.New()` prepends `https://` at runtime.

## Debugging Sessions

Every chat and game-action turn writes a **turn trace** to `TRACES_TABLE` (`internal/trace`). The trace holds
each model call with its full prompt and raw output, the campaign state before and after the turn, the resolver
verdict, the transition, events, every frame sent, and any errors. Traces are kept 30 days. Tracing is
optional: without `TRACES_TABLE` the Lambdas skip it, which is why the variable is not in
`TestAllRequiredEnvVarsPanic`.

Read a session with `go run ./tools/sessiondump [-full] [-json] SESSION_ID` from `server/` (needs read access to
the sessions, mutations and traces tables). See `.github/agents/game-session-debugger.agent.md` for the
full protocol.

`go run ./tools/replay` reruns traced turns through a story pipeline and reports original vs candidate.
`server/testdata/turns/` is the saved **reference turn set** (real failure cases, see its README): run it
with `-from testdata/turns -pipeline recorded` offline, or `-pipeline current` against Bedrock; add turns
from a session with `-save`.

## Testing Notes

- `db.New()` requires `SESSIONS_TABLE` env var — set with `t.Setenv()` in handler tests
- WS tests: use `act(() => Promise.resolve())` to flush `useEffect`, not `setTimeout`-based `flushPromises`
- `vi.mock` factories are hoisted — use `vi.hoisted()` for mock functions needed in factory scope
- `scrollIntoView` is not available in jsdom — guard with `typeof ... === "function"`

### Required env var tests (Go Lambdas)
Every Lambda handler must have a `TestAllRequiredEnvVarsPanic` test that asserts each required env var causes a panic when absent. This is the primary guard against Terraform drift — if an env var is needed by the code but missing from the Lambda's Terraform config, this test documents it and will catch the mismatch during code review.

Pattern:
```go
var requiredEnvVars = []string{"SESSIONS_TABLE", "USERS_TABLE", "MEMBERSHIPS_TABLE"}

func TestAllRequiredEnvVarsPanic(t *testing.T) {
    req := makeValidReq()
    for _, env := range requiredEnvVars {
        env := env
        t.Run(env, func(t *testing.T) {
            assertPanicsWithEnvAbsent(t, env, func() {
                handler(context.Background(), req) //nolint:errcheck
            })
        })
    }
}
```

The `assertPanicsWithEnvAbsent` helper sets the env var to `""`, calls the function, and asserts a panic whose message mentions the var name. It lives in each `cmd/*/handler_test.go`.
