# CLAUDE.md

Guidance for AI assistants working in this repository.

## Project Overview

An AI-powered text adventure game. Claude (via AWS Bedrock) acts as the Dungeon Master. The game features dynamic world generation, real-time streaming narrative over WebSocket, and persistent game state in DynamoDB.

**Current Status:** Alpha — core loop working in production.

## Tech Stack

### Backend (`server-v2/` — Go 1.24, AWS Lambda)
- **AI:** AWS Bedrock — `us.anthropic.claude-sonnet-4-6` (narrator + architect), `us.anthropic.claude-haiku-4-5-20251001-v1:0` (sub-agents)
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
- **IaC:** Terraform in `server/infra/` (git subtree → `rrochlin/terraform-infrastructure`)
- **Region:** `us-west-2`, Account: `292826404083`
- **CloudFront:** `d1ctll9l3g8cf4.cloudfront.net`

## Repository Layout

```
An-Amazing-Adventure/
├── client/                  # React frontend
│   ├── src/
│   │   ├── routes/          # TanStack file-based routes
│   │   ├── components/      # Chat, GameInfo, RoomMap, WorldGenTerminal, ...
│   │   ├── hooks/           # useGameSocket
│   │   ├── services/        # api.service, api.game, api.users, auth.service
│   │   ├── store/           # gameStore.ts (Zustand)
│   │   └── types/           # types.ts
│   └── vite.config.ts
├── server-v2/               # Go Lambda handlers
│   ├── cmd/
│   │   ├── http-games/      # GET/POST/DELETE /api/games
│   │   ├── http-users/      # PUT /api/users (profile update via Cognito)
│   │   ├── world-gen/       # Async Lambda: blueprint → world build → WS progress
│   │   ├── ws-connect/      # WebSocket $connect
│   │   ├── ws-disconnect/   # WebSocket $disconnect
│   │   ├── ws-chat/         # WebSocket chat action → Bedrock streaming
│   │   └── ws-game-action/  # WebSocket game_action (move/pick_up/drop/equip/unequip)
│   └── internal/
│       ├── ai/              # Bedrock client, NarrateStream, GenerateBlueprint, BuildWorldFromBlueprint
│       ├── db/              # DynamoDB client (BinaryID type for B-typed keys)
│       ├── game/            # Game engine (Area, Character, Item, Game, SaveState)
│       └── wsutil/          # WebSocket frame push helpers
└── server/infra/            # Terraform (git subtree → rrochlin/terraform-infrastructure)
    └── amazing-adventure/
        └── modules/         # dynamodb, cognito, s3, lambdas, api-gateway, cloudfront
```

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
cd server-v2
go build ./...
go test ./...
```

### Terraform
```bash
cd server/infra/amazing-adventure
doppler run --project terraform-personal-infra --config dev_personal -- terraform plan
doppler run --project terraform-personal-infra --config dev_personal -- terraform apply
```

### Push Terraform changes to infra repo
```bash
git subtree push --prefix server/infra git@github.com:rrochlin/terraform-infrastructure.git <branch-name>
# Then open PR on rrochlin/terraform-infrastructure
```

## Architecture

### Game Flow
1. User fills out `/create` wizard (character + adventure prefs) → `POST /api/games` with `AdventureCreationParams` → returns `session_id`
2. Client navigates immediately to `/game-{uuid}`; `useGameSocket` connects; `WorldGenTerminal` shown inline
3. `http-games` fires `world-gen` Lambda async with all creation params
4. `world-gen` calls `GenerateBlueprint` (enriched prompt), builds world deterministically, persists `Title`/`Theme`/`QuestGoal`/`TotalTokens`/`CreationParams` to `SaveState`
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
- `us.anthropic.claude-sonnet-4-6` — narrator and architect
- `us.anthropic.claude-haiku-4-5-20251001-v1:0` — sub-agents
Bare model IDs without prefix are rejected with `ValidationException`.

## Working Process

### Starting a new phase — MANDATORY first step
**Before creating a feature branch or writing any code, always sync with the remote:**

```bash
git fetch origin
git log --oneline origin/main -5   # confirm any recently merged PRs are visible
git checkout -b feat/<phase-name> origin/main
```

**Why this matters:** If a previous PR was merged while the last session was running, your local `main` is stale. Starting a branch from stale `main` means your Phase N branch includes Phase N-1 commits. When you later rebase/PR, git sees those commits as divergent (not as the squash-merge from main) and either raises conflicts or duplicates the work.

**If you forgot and already have commits on a stale branch:**
```bash
# Identify your Phase N-only commits (the ones NOT in origin/main)
git log --oneline origin/main..HEAD

# Create a clean branch and cherry-pick only your commits
git checkout -b feat/<phase-name>-rebased origin/main
git cherry-pick <sha1> <sha2> ...   # your Phase N commits only

# Push and open a new PR; close the stale one
git push -u origin feat/<phase-name>-rebased
gh pr close <old-pr-number> --comment "Superseded by rebased branch"
gh pr create ...
```

### Commit cadence
**Commit early and often — do not accumulate a large diff before committing.**

Every logical phase of work should be its own commit before moving to the next:
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
- **This monorepo** (client + server-v2): open a PR to `main` via `gh pr create`. GitHub Actions runs tests on every PR and deploys on merge to `main`.
- **Infrastructure changes** (new AWS resources, API Gateway routes, Lambda env vars, DynamoDB tables, etc.): these live in a **separate repo** `rrochlin/terraform-infrastructure`, linked here as a git subtree at `server/infra/`. Push changes with:
  ```bash
  git subtree push --prefix server/infra git@github.com:rrochlin/terraform-infrastructure.git <branch-name>
  gh pr create --repo rrochlin/terraform-infrastructure ...
  ```
- **Never commit compiled Lambda binaries** (`server-v2/http-games`, `server-v2/world-gen`, etc.) — they are in `.gitignore`. CI builds them from source on every deploy.

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

## Testing Notes

- `db.New()` requires `SESSIONS_TABLE` env var — set with `t.Setenv()` in handler tests
- WS tests: use `act(() => Promise.resolve())` to flush `useEffect`, not `setTimeout`-based `flushPromises`
- `vi.mock` factories are hoisted — use `vi.hoisted()` for mock functions needed in factory scope
- `scrollIntoView` is not available in jsdom — guard with `typeof ... === "function"`
