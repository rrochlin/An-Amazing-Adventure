## S3
- s3 bucket for the application. cloudfront will be hosting the application from here
- index.html file in the application bucket should have no-cache headers set (Cache-Control: no-store)
- Static assets (js/css/fonts) can use long-lived cache with content-hash filenames (Vite does this by default)
- Bucket is private — access via OAC (Origin Access Control), not a public bucket policy
- s3 bucket for generated content like images or other assets that are user specific (defer until image generation is implemented)

## Dynamo
- Three tables total:
  - **Sessions table**: PK = `session_id` (binary UUID). GSI = `user-sessions-index` on `user_id`. Stores full serialized game state.
  - **WebSocket connections table**: PK = `connection_id` (string). Attributes: `user_id`, `game_id`, `connected_at`, TTL = `expires_at` (set to ~24h). Required so Lambda can push back to open connections via the API GW Management API.
  - Cognito replaces the users table and refresh tokens table — those DynamoDB tables are retired.
- Game state schema must include a `schema_version` field (start at `1`) to handle future migrations without breaking existing saves.
- Entire game state is stored as marshalled binary content in the sessions table.

## Cloudfront
- Single distribution with two origins:
  1. **S3 origin** (default `/*`) — serves the SPA static assets. OAC configured.
  2. **HTTP API Gateway origin** (`/api/*`) — proxies REST calls. Cache disabled (TTL=0).
  3. **WebSocket API Gateway origin** (`/ws`) — proxies WebSocket upgrade. Cache disabled.
- Custom error responses: 403 and 404 both return `/index.html` with HTTP 200 (SPA fallback).
- HTTPS only (redirect HTTP → HTTPS).
- Custom domain + ACM certificate in us-east-1 (CloudFront requires ACM certs in us-east-1 regardless of deployment region).
- WebSocket support: CloudFront natively supports WebSocket over HTTP/1.1. The cache behavior for `/ws` must use an origin request policy that forwards `Sec-WebSocket-Key`, `Sec-WebSocket-Version`, `Sec-WebSocket-Protocol`, `Sec-WebSocket-Accept`, and `Sec-WebSocket-Extensions` headers. Use `AllViewer` managed policy or a custom policy with those headers explicitly listed.

## Websocket API (API Gateway V2)
- Using apigatewayv2 WebSocket API.
- Routes: `$connect`, `$disconnect`, `$default`, plus custom routes: `chat`, `game_action`.
- `$connect` route has a Lambda authorizer attached that validates the Cognito JWT from a query string parameter (`?token=...`) since WebSocket upgrade requests cannot carry Authorization headers in most browsers.
- On `$connect` success: Lambda writes `{connection_id, user_id, game_id, expires_at}` to the WebSocket connections DynamoDB table.
- On `$disconnect`: Lambda deletes the connection record. If a prior connection exists for the same `user_id`, it is also cleaned up (one connection per user enforced here).
- `chat` route: triggers the game engine Lambda. While streaming is active, additional `chat` or `game_action` messages from the same connection are rejected with an error frame (enforced via a `streaming_in_progress` flag in the connection record).
- `game_action` route: handles non-chat player interactions (inventory, equip, inspect) that also mutate game state.
- Lambda pushes partial narrative chunks back to the client using the API Gateway Management API (`POST /@connections/{connectionId}`).
- Staging: single `$default` stage.

## HTTP API (API Gateway V2)
- Using apigatewayv2 HTTP API with Cognito JWT authorizer (not a Lambda authorizer).
- Routes and methods:
  - `POST   /api/users`          — create user (no auth, Cognito sign-up)
  - `PUT    /api/users`          — update profile (auth required)
  - `POST   /api/games`          — create new game session (auth required)
  - `DELETE /api/games/{uuid}`   — delete game session (auth required, ownership verified)
  - `GET    /api/games`          — list user's games (auth required)
  - `GET    /api/games/{uuid}`   — get full initial game state to load on page entry (auth required)
  - `GET    /api/worldready/{uuid}` — poll world generation status (auth required)
- Note: `/api/describe` is folded into `GET /api/games/{uuid}` — no need for a separate describe endpoint.
- CORS configured on the API GW itself (not in application code).

## Lambdas
- All Lambda functions are deployed as **zip archives** (not container images) for Go:
  - Build: `GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o bootstrap .`
  - Zip: `zip function.zip bootstrap`
  - Runtime: `provided.al2023` on arm64 (Graviton2 — cheaper and faster for Go than x86).
- Lambda functions:
  - `ws-connect` — handles `$connect`, authorizes token, writes connection record.
  - `ws-disconnect` — handles `$disconnect`, cleans up connection record.
  - `ws-chat` — handles `chat` route, runs full game engine + Bedrock streaming, pushes frames back.
  - `ws-game-action` — handles `game_action` route, executes non-AI game state mutations.
  - `http-games` — handles all `/api/games*` REST routes.
  - `http-users` — handles `/api/users` REST routes.
  - `world-gen` — async world generation invoked asynchronously (not via API GW). Triggered by `http-games` via `lambda:InvokeAsync` after game record is created. Updates game state and sets `ready=true` when complete. Max timeout = 15 minutes.
- Each Lambda has its own IAM execution role following least-privilege.

## Cognito
- Simple User Pool setup:
  - Sign-in with email (case insensitive).
  - Email verification required on sign-up.
  - No SMS recovery — email-only MFA/recovery.
  - No social identity providers or federated login.
  - No Cognito hosted UI — custom login/signup forms in the React app using the Cognito User Pools API directly (via `@aws-sdk/client-cognito-identity-provider`).
- App client: SRP auth flow (`USER_SRP_AUTH`), no client secret (public client for SPA).
- Existing users in the old DynamoDB users table cannot have passwords migrated (bcrypt hashes are not importable). Options:
  - Since this is early alpha with no real user base, drop the old table and require re-registration.
  - If users need to be preserved, use a Cognito Pre-Authentication Lambda trigger to check the old table and migrate on first login.
- JWT authorizer on HTTP API uses the Cognito User Pool issuer URL and the app client ID as audience.

## Bedrock
- Used for all AI access.
- Models:
  - Sub-agents (world-builder, NPC creator, item creator): `anthropic.claude-haiku-4-5` (lighter, faster, cheaper).
  - Architect and Narrator agents: `anthropic.claude-sonnet-4-6` (heavier, higher quality).
- API: use `ConverseStream` for the chat/narrator (streaming to WebSocket). Use `Converse` (non-streaming) for world generation sub-agents where full responses are needed before proceeding.
- Tool/function calling: Bedrock `Converse` API has native `toolConfig` support. Tool definitions are passed in the `toolConfig.tools` array; tool use results are returned as `toolUse` content blocks. Port current `mcp.go` tool definitions to `bedrock.Tool` structs.
- Region: deploy in `us-west-2` (keeps Bedrock and DynamoDB co-located, reduces latency).
- Bedrock model access must be explicitly requested in the AWS console for the target account and region before deployment.

## IAM
- Lambda execution roles (one per function, or shared where appropriate):
  - `ws-connect` / `ws-disconnect`: `dynamodb:PutItem`, `dynamodb:DeleteItem` on connections table.
  - `ws-chat`: `dynamodb:GetItem`, `dynamodb:PutItem` on sessions + connections tables; `bedrock:InvokeModelWithResponseStream`; `execute-api:ManageConnections` on the WebSocket API.
  - `ws-game-action`: `dynamodb:GetItem`, `dynamodb:PutItem` on sessions + connections tables; `execute-api:ManageConnections`.
  - `http-games`: `dynamodb:GetItem`, `dynamodb:PutItem`, `dynamodb:DeleteItem`, `dynamodb:Query` on sessions table; `lambda:InvokeFunction` on `world-gen`.
  - `http-users`: `cognito-idp:AdminUpdateUserAttributes` (if profile updates bypass the app client).
  - `world-gen`: `dynamodb:GetItem`, `dynamodb:PutItem` on sessions table; `bedrock:InvokeModel`.
- No cross-account access.

## Secrets / Configuration
- Doppler is used for local development only.
- In Lambda: all configuration is passed via environment variables set by Terraform:
  - `SESSIONS_TABLE`, `CONNECTIONS_TABLE`, `USER_POOL_ID`, `BEDROCK_REGION`, `WEBSOCKET_API_ENDPOINT`.
- No application-level JWT secret needed — Cognito issues and validates tokens.
- GCP_KEY and the old JWT SECRET are retired.
