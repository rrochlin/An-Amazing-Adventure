The current UI's aesthetic is quite nice despite a few visual bugs. The map layout and spacing is in serious need of an overhaul and the system in general is quite broken.

## Migration: GCP → AWS (breaking changes to address first)

### API endpoint
- `VITE_APP_URI` in `client/.env.production` currently points to the GCP Cloud Run URL. Change to the CloudFront domain (e.g. `https://play.your-domain.com`) and use relative paths (`/api`) so the SPA and API share the same origin through CloudFront.
- Update `api.service.ts` base URL accordingly. CloudFront will route `/api/*` to the HTTP API Gateway and `/ws` to the WebSocket API Gateway.

### Authentication (auth.service.ts rewrite)
Current: custom JWT stored in `localStorage["AAA_JWT"]`, manual refresh token rotation against DynamoDB.
New: Cognito User Pools via the AWS SDK for JavaScript.
- Install: `@aws-sdk/client-cognito-identity-provider`
- Replace `Login()` with `InitiateAuth` using `USER_SRP_AUTH` flow (SRP handles the password exchange without transmitting it in plaintext).
- Replace `CreateNewUser()` with `SignUp` + `ConfirmSignUp` (email verification code flow).
- Cognito returns `AccessToken`, `IdToken`, `RefreshToken`. Store in `localStorage` (same key structure is fine, update fields).
- Replace `refreshToken()` with `InitiateAuth` using `REFRESH_TOKEN_AUTH` flow.
- Replace `revokeToken()` with `RevokeToken`.
- Replace `ClearUserAuth()` with `GlobalSignOut` (invalidates all sessions) + `localStorage.removeItem`.
- `getAuthHeaders()`: use the `AccessToken` (not `IdToken`) in the `Authorization: Bearer` header for API calls. The HTTP API Gateway JWT authorizer validates against the Cognito User Pool.
- `isAuthenticated()`: check `AccessToken` expiry from the decoded JWT (`exp` claim), same pattern as today.
- `login.tsx` and `signup.tsx` forms remain custom — no Cognito Hosted UI.

### Chat: HTTP polling → WebSocket (Chat.tsx rewrite)
Current: `axios.post('/api/chat/{uuid}')` returns the full AI response as one payload.
New: persistent WebSocket connection receives streaming chunks.

WebSocket lifecycle (manage in a React context or custom hook `useGameSocket`):
1. On game page mount: open `wss://your-domain.com/ws?token=<AccessToken>&gameId=<uuid>`.
2. On `open`: connection is established. Server writes connection record to DynamoDB.
3. On `message`: parse JSON frame. Frame types:
   - `{type: "narrative_chunk", content: string}` — append to current AI message being built.
   - `{type: "narrative_end"}` — finalize the AI message, re-enable input.
   - `{type: "game_state_update", state: GameState}` — merge into React state (tool calls resolved).
   - `{type: "error", message: string}` — display error toast.
   - `{type: "streaming_blocked"}` — inform user a chat is in progress (server-side enforcement).
4. On `close` / `error`: attempt reconnect with exponential backoff (max 3 retries before showing disconnect banner).
5. On game page unmount: close the WebSocket gracefully.

Send a chat message: `ws.send(JSON.stringify({action: "chat", content: inputText}))`.

State management: introduce a lightweight state store (Zustand recommended — small bundle, no boilerplate) to hold `GameState`, `chatMessages`, `isStreaming`, `wsStatus`. All components read from this store rather than local state. This replaces the current pattern of per-route loaders + local component state.

### Nginx / Docker removal
The `client/dockerfile` and `client/nginx.conf.template` are deleted. The client becomes pure static assets deployed to S3.

---

## Map
### Issues to fix
- Nodes have a poor layout — room coordinate calculation from BFS needs fixing (the `CalculateRoomCoordinates` spacing and direction vectors produce overlapping nodes).
- Connections double back on each other — corridor rendering logic needs to correctly draw line segments between calculated node centers, not use rotated rectangles.
- Z-axis floor selector is broken — the floor chip filtering logic needs a reliable comparison.
- Click/Hover effects not working — Konva event handlers need to be properly attached to the correct layer.
- Auto zoom/snap to current room doesn't work reliably.
### Enhancements
- Pop-out feature: a button to expand the map to a larger modal/drawer overlay.
- Fog of war: rooms not yet visited remain obscured (currently partially implemented but inconsistent).
- Highlight connected exits from current room.
### Decision needed
Keep `react-konva` or switch? Konva is viable if the coordinate and rendering bugs are fixed. The library is not the problem — the coordinate calculation and corridor drawing logic is. Recommend fixing the existing implementation rather than rewriting with a different library. Revisit if the pop-out/3D feature is added later.

## Inventory
### Issues to fix
- Tool calls don't always get executed correctly — this is a server-side issue (AI JSON parsing fragility in `handlers_chat.go`). The Bedrock port should use structured `toolUse` blocks natively instead of regex-parsing JSON from freeform text.
- State fails to update/sync with server — with WebSocket + `game_state_update` frames, this becomes event-driven rather than pulled after each POST.
### Enhancements
- Interactivity: equip, drop, discard, inspect actions. These send `game_action` frames over WebSocket rather than making HTTP calls, keeping all game state changes through one channel.
- Equipment slots panel (head/chest/legs/hands/feet/back) once the `Equipment` struct is implemented server-side.

## Room
### Issues to fix
- Data shown is often inconsistent — same root cause as inventory sync. Resolved by the WebSocket `game_state_update` event driving a single source of truth in the Zustand store.
### Enhancements
- Item/NPC descriptions with hover tooltips.
- Room graphic or ambient illustration (future — defer until image generation is implemented).
- Correlate room panel info to highlighted node on the map.

## State Management (new section)
Install `zustand`:
```
pnpm add zustand
```
Store shape:
```typescript
interface GameStore {
  gameState: GameState | null
  chatMessages: ChatMessageType[]
  isStreaming: boolean
  wsStatus: 'connecting' | 'connected' | 'disconnected' | 'error'
  setGameState: (state: GameState) => void
  mergeGameState: (partial: Partial<GameState>) => void
  appendChatMessage: (msg: ChatMessageType) => void
  appendNarrativeChunk: (chunk: string) => void
  finalizeNarrativeMessage: () => void
  setStreaming: (v: boolean) => void
  setWsStatus: (s: GameStore['wsStatus']) => void
}
```

## New Dependencies
```
pnpm add zustand @aws-sdk/client-cognito-identity-provider
pnpm remove @toolpad/core   # was only used for auth provider type shim
```
