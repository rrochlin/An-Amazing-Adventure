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
- **WebSocket:** custom `useGameSocket` + `useWorldGenSocket` hooks
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
│   │   ├── hooks/           # useGameSocket, useWorldGenSocket
│   │   ├── services/        # api.service, api.game, api.users, auth.service
│   │   ├── store/           # gameStore.ts (Zustand)
│   │   └── types/           # types.ts
│   └── vite.config.ts
├── server-v2/               # Go Lambda handlers
│   ├── cmd/
│   │   ├── http-games/      # GET/POST/DELETE /api/games, /api/worldready
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
git subtree push --prefix server/infra git@github.com:rrochlin/terraform-infrastructure.git phase-5-restructure
# Then open PR on rrochlin/terraform-infrastructure
```

## Architecture

### Game Flow
1. User creates game → `POST /api/games` → returns `session_id`
2. `http-games` fires `world-gen` Lambda async
3. Client opens WebSocket immediately with `session_id`
4. `world-gen` queries connections table → pushes `world_gen_log` frames to client terminal
5. On completion → `world_gen_ready` frame → client navigates to game
6. Client opens persistent WebSocket for game play
7. Chat messages → `ws-chat` → Bedrock streaming → `narrative_chunk` frames
8. Game actions (move/pick_up/equip) → `ws-game-action` → state delta frames

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

## CI/CD

Two GitHub Actions workflows:
- `deploy-client.yml` — test (PR+push) + deploy to S3/CloudFront (main only)
- `deploy-server.yml` — test (PR+push) + build arm64 zip + update Lambda (main only)

Branch protection: all changes must go through PRs. Direct push to `main` is blocked.

## Key Known Issues / Discoveries

- **BinaryID**: sessions and connections tables have B-typed keys. See `internal/db/binaryid.go`.
- **CONNECTIONS_TABLE**: `http-games` and `world-gen` don't need it at startup — `db.New()` defers the panic to actual connection methods via `requireConnectionsTable()`.
- **Inference profiles**: Bedrock requires `us.` prefixed profile IDs, not bare model IDs.
- **WebSocket endpoint**: `WEBSOCKET_API_ENDPOINT` env var contains `domain/stage/stage` (e.g. `ba2t50m7se.execute-api.us-west-2.amazonaws.com/prod/prod`) — this is intentional and matches what API Gateway Management API expects.

## Testing Notes

- `db.New()` requires `SESSIONS_TABLE` env var — set with `t.Setenv()` in handler tests
- WS tests: use `act(() => Promise.resolve())` to flush `useEffect`, not `setTimeout`-based `flushPromises`
- `vi.mock` factories are hoisted — use `vi.hoisted()` for mock functions needed in factory scope
- `scrollIntoView` is not available in jsdom — guard with `typeof ... === "function"`
