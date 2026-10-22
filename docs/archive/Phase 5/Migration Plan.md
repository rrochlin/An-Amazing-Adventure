# Migration Plan: GCP -> AWS

> Historical note:
> This migration plan describes the AWS transition work that has already largely been completed.
> It is retained for background only and is not the current architecture source of truth.
>
> For the current live architecture, see:
> - `Overview.md`
> - `Project Structure.md`
> - `AI Architecture.md`
> - `Game Engine.md`
> - `UI.md`
> - `TODO.md`

This document covers the ordered execution plan to migrate from the current GCP/Gemini/Docker stack to AWS/Bedrock/Lambda. The migration is a full replacement, not an incremental transition, since the data formats (Gemini history → Bedrock history) and auth systems (custom JWT → Cognito) are incompatible.

---

## Phase 5.0 — Bootstrap Infrastructure (no application code changes yet)

Goal: Get AWS infrastructure defined in Terraform and deployed to a dev environment. Application stays on GCP until Phase 5.3.

1. **Create `server/infra/` git subtree** linked to the `rrochlin/terraform-infrastructure` repo on branch `amazing-adventure`.
2. **Write Terraform modules** for (in dependency order):
   - `dynamodb`: sessions table (with `schema_version` + `version` fields), connections table (with TTL).
   - `cognito`: User Pool + app client (SRP, email-only, no SMS).
   - `s3`: app assets bucket (private, versioning on), OAC.
   - `lambdas`: function resources (placeholder code initially), IAM execution roles, environment variables.
   - `api-gateway`: HTTP API + WebSocket API + Cognito JWT authorizer + Lambda integrations + stages.
   - `cloudfront`: distribution with S3 origin + HTTP API origin + WebSocket origin, custom error responses, cache behaviors.
3. **Request Bedrock model access** in `us-west-2` for `anthropic.claude-haiku-4-5` and `anthropic.claude-sonnet-4-6` via the AWS console.
4. **Set up OIDC trust** between the `rrochlin/An-Amazing-Adventure` GitHub repo and an IAM role with permissions to update Lambda code + S3 + CloudFront invalidation.
5. **PR to terraform-infrastructure repo** → pipeline deploys dev environment → verify all resources are created.

Deliverables: All AWS resources exist. CloudFront returns 200 for the SPA (serving a placeholder index.html). Lambda functions exist but are stubs. API Gateway routes exist.

---

## Phase 5.1 — Backend: Lambda Restructure + Bedrock Port

Goal: Port the Go server to Lambda handlers with Bedrock. Does not need to be connected to the real frontend yet — test with curl/wscat.

1. **Restructure Go project** into `server/cmd/<lambda-name>/` packages.
2. **Port `internal/auth/`**: remove custom JWT logic, add Cognito JWT validation (decode the Access Token using the Cognito JWKS endpoint — the JWT authorizer on API GW handles this for HTTP routes, but the WebSocket `ws-connect` Lambda needs to validate it manually).
3. **Port `internal/game/`**: update structs per `Game Engine.md` — add `Container`, `Equipment`, `SchemaVersion`, `Version`. Keep `Area`/`Character`/`Item` logic intact.
4. **Port `internal/ai/`**: replace `google.golang.org/genai` with `github.com/aws/aws-sdk-go-v2/service/bedrockruntime`. Rewrite tool definitions as `types.Tool`. Rewrite `ExecuteTool` dispatcher. Implement `ConverseStream` chat loop with streaming push-back via API GW Management API.
5. **Implement `world-gen` Lambda**: port `world_gen.go` to the new Architect → Engineer → Sub-agent pattern. Invoked asynchronously. On completion, sets `game.Ready = true` and saves to DynamoDB.
6. **Implement `http-games` Lambda**: game CRUD, calls `world-gen` asynchronously on game creation, returns `{ready: false}` immediately.
7. **Implement `http-users` Lambda**: wraps Cognito sign-up/profile update.
8. **Implement `ws-connect` / `ws-disconnect`**: connection record lifecycle.
9. **Implement `ws-chat`**: streaming chat, Bedrock `ConverseStream`, push back frames.
10. **Implement `ws-game-action`**: direct game state mutations (equip, drop, pick up, move).
11. **Update CI**: `deploy-server.yml` builds Go zip artifacts and uploads to Lambda.
12. **Remove** `server/dockerfile`, `server/doppler.yaml` (add local dev instructions using AWS SAM CLI or a thin HTTP shim).

Test each Lambda in isolation with SAM CLI before deploying. Integration test by hitting API GW endpoints directly with curl and wscat.

---

## Phase 5.2 — Frontend: Auth + WebSocket Migration

Goal: Update the React client to use Cognito and WebSocket. The UI visual design is unchanged in this phase — only the data layer changes.

1. **Install** `zustand` + `@aws-sdk/client-cognito-identity-provider`. Remove `@toolpad/core`.
2. **Rewrite `auth.service.ts`**: Cognito SRP auth, store `AccessToken`/`IdToken`/`RefreshToken` in localStorage.
3. **Rewrite `login.tsx` and `signup.tsx`**: add email verification step after sign-up (enter confirmation code).
4. **Remove `api.users.ts`** login/signup calls; replace with direct Cognito SDK calls. Keep `UpdateUser` but route through the HTTP API if needed.
5. **Create `useGameSocket` hook**: manages WebSocket connection lifecycle, reconnect logic, and dispatches incoming frames to the Zustand store.
6. **Create Zustand `gameStore`**: replaces per-component local state and TanStack route loaders for game data.
7. **Rewrite `Chat.tsx`**: remove axios chat POST, send/receive via WebSocket, render streaming narrative chunks as they arrive.
8. **Update `api.service.ts`**: change `VITE_APP_URI` to relative `/api`. Remove the axios 401 interceptor refresh logic (Cognito SDK handles token refresh).
9. **Update `client/.env.production`**: remove GCP URL, set `VITE_COGNITO_USER_POOL_ID` and `VITE_COGNITO_CLIENT_ID`.
10. **Remove `client/dockerfile`** and `client/nginx.conf.template`.
11. **Update CI**: `deploy-client.yml` builds Vite and syncs to S3 + CloudFront invalidation.

---

## Phase 5.3 — Cut Over + Retire GCP

Goal: Switch production traffic from GCP to AWS.

1. Update DNS to point the custom domain to the CloudFront distribution.
2. Update `client/.env.production` if using a custom domain.
3. Delete the GCP Cloud Run services (`an-amazing-adventure` and `an-amazing-adventure-server`).
4. Archive or delete Docker Hub images (`rrochlin/an-amazing-adventure-client`, `rrochlin/an-amazing-adventure-server`).
5. Remove GitHub Actions secrets: `DOCKERHUB_TOKEN`, `GCP_SA_KEY`, `GCP_REGION`, `GCP_PROJECT_ID`.
6. Delete the old GCP workflow (`build-push-deploy.yml`).
7. Drop/archive old DynamoDB tables: `AWS_USERS_TABLE`, `AWS_R_TOKENS_TABLE` (game sessions table data is incompatible with new schema — wipe and start fresh or rename).
8. Remove Doppler projects (or keep for local dev only — no production role).

---

## Phase 5.4 — UI Polish + Game Engine Enhancements

Only begins after 5.3 is stable.

1. Fix map rendering bugs (coordinate calculation, corridor drawing, Z-axis floor selector).
2. Implement `Equipment` panel in `GameInfo.tsx`.
3. Implement inventory interactivity (equip, drop, pick up via `game_action` WebSocket frames).
4. Implement `Container` interaction in the UI.
5. Map pop-out feature.
6. Room panel enhancements (hover effects, item/NPC detail).
7. Context compression for long game sessions (server-side narrative summarization).
8. Campaign template support for world generation.

---

## Decisions / Open Questions

| # | Question | Default Decision |
|---|----------|-----------------|
| 1 | Keep existing users or force re-register? | Force re-register (early alpha, no real user base). Drop old users table. |
| 2 | Custom domain for production? | Yes — needed for CloudFront + ACM cert. Choose domain before Phase 5.0. |
| 3 | Single AWS account or separate dev/prod? | Single account, two Terraform workspaces (`dev`, `prod`). |
| 4 | `react-konva` vs. alternative for map? | Keep Konva, fix the coordinate/rendering bugs. |
| 5 | WebSocket token delivery method? | Query string parameter (`?token=...`) on the `wss://` URL — standard approach since browsers can't set custom headers on WebSocket upgrade requests. Token is extracted and validated in `ws-connect` Lambda. |
| 6 | Lambda provisioned concurrency? | Not initially — add if cold start latency is observed on `ws-connect`. |
| 7 | Lambda Durable Functions for `world-gen`? | **Not viable for Go** — the Durable Execution SDK only supports Node.js, TypeScript, Python, and Java (preview) as of March 2026. Go is not on the roadmap. Use plain async Lambda invoke (`lambda:InvokeAsync` / `InvocationType: Event`) from `http-games`. World generation completes well within the 15-minute Lambda timeout in practice. If orchestration complexity grows (e.g. multi-stage retries, human-in-the-loop), revisit with Step Functions Express Workflows, which have full Go support via the AWS SDK. |
