# Phase 1 — Party Sessions + Invite System

Adds multi-player party support. Multiple players can share a game session, each with their own character. The dungeon is shared. All state changes are broadcast to all connected party members in real time.

Prerequisites: [[Terraform Infrastructure]] deployed (memberships table, game_id GSI on connections, invites table, `http-invites` Lambda).

Branch: `feat/phase-6-party`

---

## Goals

1. Multiple players can connect to the same game session simultaneously
2. Each player has their own character (not yet a D&D character — that's Phase 2; still a `game.Character` for now)
3. All `StateDelta` frames broadcast to every connected party member
4. Three join paths: invite link, join code, new-user onboarding flow
5. `GET /api/games` returns sessions the user owns AND sessions they've joined as a member
6. Backward compatible: existing solo sessions (SchemaVersion 1) still load and play

---

## Part 1: Session Model Changes

### `server-v2/internal/game/game.go`

The `Game` struct gains party-aware fields. The single `Player Character` field is replaced by a `Players map[string]Character` keyed by `userID`. This is the foundational change that all other party logic builds on.

```go
// Before (Phase 5):
type Game struct {
    ID                string
    UserID            string        // owner
    Player            Character     // single player
    // ...
}

// After (Phase 6 Party):
type Game struct {
    ID                string
    OwnerID           string                // session owner — only they can delete
    UserID            string                // preserved for backward compat (= OwnerID)
    Players           map[string]Character  // keyed by userID — all party members
    PartySize         int                   // max party members (default 4)
    // Rooms, Items, NPCs, etc. unchanged
    // ...
}
```

**Accessor helper** — replaces direct `g.Player` reads throughout the codebase:
```go
// GetPlayerCharacter returns the character for a specific user.
// Returns zero-value Character and false if user is not in this game.
func (g *Game) GetPlayerCharacter(userID string) (Character, bool) {
    if g.Players == nil {
        return Character{}, false
    }
    c, ok := g.Players[userID]
    return c, ok
}

// SetPlayerCharacter writes a character for a user.
func (g *Game) SetPlayerCharacter(userID string, c Character) {
    if g.Players == nil {
        g.Players = make(map[string]Character)
    }
    g.Players[userID] = c
}

// PlayerInGame returns true if the given user is a party member (owner or joined).
func (g *Game) PlayerInGame(userID string) bool {
    _, ok := g.Players[userID]
    return ok
}
```

### `SaveState` — SchemaVersion 2

```go
type SaveState struct {
    SessionID         string            `dynamodbav:"session_id"`
    OwnerID           string            `dynamodbav:"owner_id"`
    UserID            string            `dynamodbav:"user_id"`     // preserved — equals OwnerID
    SchemaVersion     int               `dynamodbav:"schema_version"`
    Version           int               `dynamodbav:"version"`
    Players           map[string]Character `dynamodbav:"players"`   // NEW — keyed by userID
    Player            Character         `dynamodbav:"player"`       // v1 compat only
    PartySize         int               `dynamodbav:"party_size"`
    InviteCode        string            `dynamodbav:"invite_code,omitempty"`
    Rooms             []Area            `dynamodbav:"rooms"`
    Items             []Item            `dynamodbav:"items"`
    NPCs              []Character       `dynamodbav:"npcs"`
    Narrative         []NarrativeMessage `dynamodbav:"narrative"`
    ChatHistory       []ChatMessage     `dynamodbav:"chat_history"`
    Ready             bool              `dynamodbav:"ready"`
    Title             string            `dynamodbav:"title"`
    Theme             string            `dynamodbav:"theme"`
    QuestGoal         string            `dynamodbav:"quest_goal"`
    TotalTokens       int               `dynamodbav:"total_tokens"`
    ConversationCount int               `dynamodbav:"conversation_count"`
    CreationParams    AdventureCreationParams `dynamodbav:"creation_params"`
}
```

### Migration: v1 → v2

`FromSaveState` handles both versions:

```go
func FromSaveState(ss SaveState) (*Game, error) {
    if ss.SchemaVersion > 2 {
        return nil, fmt.Errorf("unsupported schema version %d", ss.SchemaVersion)
    }

    g := &Game{
        ID:        ss.SessionID,
        OwnerID:   ss.OwnerID,
        UserID:    ss.UserID,
        PartySize: ss.PartySize,
        // ... all other fields
    }

    if g.OwnerID == "" {
        g.OwnerID = ss.UserID // v1 migration
    }
    if g.PartySize == 0 {
        g.PartySize = 4 // default
    }

    // v1 had a single Player; migrate it into Players map
    if ss.SchemaVersion <= 1 {
        ownerID := ss.UserID
        g.Players = map[string]Character{ownerID: ss.Player}
    } else {
        g.Players = ss.Players
        if g.Players == nil {
            g.Players = make(map[string]Character)
        }
    }

    return g, nil
}
```

### `BuildGameStateView` — caller-aware

```go
// BuildGameStateView builds the game state for a specific caller.
// Self is the caller's own character. Party contains all other members.
func (g *Game) BuildGameStateView(callerUserID string) GameStateView {
    self, _ := g.GetPlayerCharacter(callerUserID)
    selfView := buildCharacterView(self, g)

    var party []CharacterView
    for uid, char := range g.Players {
        if uid == callerUserID {
            continue
        }
        party = append(party, buildCharacterView(char, g))
    }

    return GameStateView{
        Self:        selfView,
        Party:       party,
        CurrentRoom: buildRoomView(g.Rooms[self.LocationID], g),
        // ... other fields
    }
}
```

Updated view types:
```go
type GameStateView struct {
    Self        CharacterView   `json:"self"`
    Party       []CharacterView `json:"party"`        // other party members
    CurrentRoom RoomView        `json:"current_room"`
    // ... existing fields
}

type CharacterView struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Health      int    `json:"health"`
    Alive       bool   `json:"alive"`
    LocationID  string `json:"location_id"`
    // ... existing fields
}
```

---

## Part 2: DB Layer Changes

### `server-v2/internal/db/memberships.go` (new file)

```go
package db

import (
    "context"
    "fmt"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type MembershipRecord struct {
    UserID    BinaryID `dynamodbav:"user_id"`    // PK
    SessionID BinaryID `dynamodbav:"session_id"` // SK
    Role      string   `dynamodbav:"role"`        // "owner" | "member"
    JoinedAt  int64    `dynamodbav:"joined_at"`   // Unix ms
}

func (c *Client) PutMembership(ctx context.Context, m MembershipRecord) error {
    item, err := attributevalue.MarshalMap(m)
    if err != nil {
        return err
    }
    _, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(c.membershipsTable),
        Item:      item,
    })
    return err
}

// GetMemberSessions returns all session IDs the user belongs to (owned + joined).
func (c *Client) GetMemberSessions(ctx context.Context, userID string) ([]string, error) {
    key, _ := marshalBinaryKey("user_id", userID)
    out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
        TableName:              aws.String(c.membershipsTable),
        KeyConditionExpression: aws.String("user_id = :uid"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":uid": key["user_id"],
        },
    })
    if err != nil {
        return nil, fmt.Errorf("GetMemberSessions: %w", err)
    }
    var sessionIDs []string
    for _, item := range out.Items {
        var m MembershipRecord
        if err := attributevalue.UnmarshalMap(item, &m); err != nil {
            continue
        }
        sessionIDs = append(sessionIDs, string(m.SessionID))
    }
    return sessionIDs, nil
}

// GetSessionMembers returns all memberships for a session (for access control checks).
func (c *Client) GetSessionMembers(ctx context.Context, sessionID string) ([]MembershipRecord, error) {
    key, _ := marshalBinaryKey("session_id", sessionID)
    out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
        TableName:              aws.String(c.membershipsTable),
        IndexName:              aws.String("session-members-index"),
        KeyConditionExpression: aws.String("session_id = :sid"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":sid": key["session_id"],
        },
    })
    if err != nil {
        return nil, fmt.Errorf("GetSessionMembers: %w", err)
    }
    var members []MembershipRecord
    for _, item := range out.Items {
        var m MembershipRecord
        _ = attributevalue.UnmarshalMap(item, &m)
        members = append(members, m)
    }
    return members, nil
}

func (c *Client) DeleteMembership(ctx context.Context, userID, sessionID string) error {
    uk, _ := marshalBinaryKey("user_id", userID)
    sk, _ := marshalBinaryKey("session_id", sessionID)
    _, err := c.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
        TableName: aws.String(c.membershipsTable),
        Key: map[string]types.AttributeValue{
            "user_id":    uk["user_id"],
            "session_id": sk["session_id"],
        },
    })
    return err
}
```

### `server-v2/internal/db/invites.go` (new file)

```go
package db

type InviteRecord struct {
    Code         string   `dynamodbav:"code"`
    SessionID    BinaryID `dynamodbav:"session_id"`
    CreatedBy    BinaryID `dynamodbav:"created_by"`
    ExpiresAt    int64    `dynamodbav:"expires_at"`  // Unix epoch SECONDS for DynamoDB TTL
    MaxUses      int      `dynamodbav:"max_uses"`
    Uses         int      `dynamodbav:"uses"`
    RedirectGame bool     `dynamodbav:"redirect_game"`
}

func (c *Client) PutInvite(ctx context.Context, invite InviteRecord) error { /* PutItem */ }
func (c *Client) GetInvite(ctx context.Context, code string) (*InviteRecord, error) { /* GetItem by code */ }
func (c *Client) IncrementInviteUses(ctx context.Context, code string) error {
    // UpdateItem: ADD uses :one, ConditionExpression: uses < max_uses OR max_uses = 0
}
```

### `server-v2/internal/db/db.go` — connection changes

Rename `DeleteUserConnections` → `DeleteUserConnectionForGame`. Add `GetConnectionsByGameID`.

```go
// DeleteUserConnectionForGame removes the connection record for a specific user+game pair.
// Called on $connect to clean up any stale connection for this user in this session.
func (c *Client) DeleteUserConnectionForGame(ctx context.Context, userID, gameID string) error {
    // Query user-connections-index for this userID
    // Filter results by game_id == gameID
    // Delete matching records
}

// GetConnectionsByGameID returns all active connections for a game session.
// Used for broadcast fanout.
func (c *Client) GetConnectionsByGameID(ctx context.Context, gameID string) ([]Connection, error) {
    out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
        TableName:              aws.String(c.connectionsTable),
        IndexName:              aws.String("game-connections-index"),
        KeyConditionExpression: aws.String("game_id = :gid"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":gid": &types.AttributeValueMemberS{Value: gameID},
        },
    })
    // unmarshal and return []Connection
}
```

**`GetConnectionByUserID` is NOT removed** — `world-gen` still uses it to find the owner's connection for initial world-gen log streaming. The one-connection-per-user invariant is relaxed (a user can have connections to multiple sessions simultaneously), so `world-gen` should use `GetConnectionsByGameID` instead and broadcast to all members. Update `world-gen` accordingly — see Part 5.

---

## Part 3: WebSocket Broadcast

### `server-v2/internal/wsutil/wsutil.go`

Add `Broadcast`:

```go
// Broadcast sends a frame to multiple connections.
// Returns the connection IDs of stale connections (410 Gone from API Gateway).
// Stale connections should be deleted by the caller.
func (s *Sender) Broadcast(ctx context.Context, connectionIDs []string, frame Frame) (stale []string, err error) {
    data, err := json.Marshal(frame)
    if err != nil {
        return nil, err
    }

    for _, connID := range connectionIDs {
        _, postErr := s.mgmt.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{
            ConnectionId: aws.String(connID),
            Data:         data,
        })
        if postErr != nil {
            var gone *apigatewaymanagementapi_types.GoneException
            if errors.As(postErr, &gone) {
                stale = append(stale, connID)
            } else {
                log.Printf("wsutil.Broadcast: send to %s: %v", connID, postErr)
            }
        }
    }
    return stale, nil
}

// BroadcastAndCleanStale sends to all connections and deletes stale ones from DynamoDB.
// This is the common pattern in ws-chat and ws-game-action.
func (s *Sender) BroadcastAndCleanStale(ctx context.Context, gameID string, frame Frame, dbClient interface {
    GetConnectionsByGameID(context.Context, string) ([]db.Connection, error)
    DeleteConnection(context.Context, string) error
}) error {
    conns, err := dbClient.GetConnectionsByGameID(ctx, gameID)
    if err != nil {
        return err
    }
    connIDs := make([]string, len(conns))
    for i, c := range conns {
        connIDs[i] = c.ConnectionID
    }
    stale, _ := s.Broadcast(ctx, connIDs, frame)
    for _, connID := range stale {
        _ = dbClient.DeleteConnection(ctx, connID)
    }
    return nil
}
```

---

## Part 4: WebSocket Handler Changes

### `server-v2/cmd/ws-connect/main.go`

Three changes:
1. Scope cleanup to `(userID, gameID)` pair
2. Validate caller is owner or member before writing connection
3. Reject empty `gameId`

```go
func handler(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (
    events.APIGatewayProxyResponse, error,
) {
    token := req.QueryStringParameters["token"]
    gameID := req.QueryStringParameters["gameId"]
    connID := req.RequestContext.ConnectionID

    // Validate inputs
    if token == "" || gameID == "" {
        return events.APIGatewayProxyResponse{StatusCode: 400}, nil
    }

    // Validate JWT (existing logic — decode, check exp, extract sub)
    userID, err := validateToken(token)
    if err != nil {
        return events.APIGatewayProxyResponse{StatusCode: 401}, nil
    }

    // Authorize: must be owner or party member
    saveState, err := dbClient.GetSaveState(ctx, gameID)
    if err != nil || saveState == nil {
        return events.APIGatewayProxyResponse{StatusCode: 404}, nil
    }
    if !isAuthorizedForSession(saveState, userID) {
        return events.APIGatewayProxyResponse{StatusCode: 403}, nil
    }

    // Clean up any stale connection for this user+game (scoped, not global)
    if err := dbClient.DeleteUserConnectionForGame(ctx, userID, gameID); err != nil {
        log.Printf("ws-connect: cleanup stale connection: %v", err)
        // Non-fatal
    }

    // Write new connection record
    // ... existing PutConnection logic
}

func isAuthorizedForSession(ss *db.SaveState, userID string) bool {
    if ss.UserID == userID || ss.OwnerID == userID {
        return true
    }
    for _, m := range ss.Members {
        if m.UserID == userID {
            return true
        }
    }
    return false
}
```

Note: `isAuthorizedForSession` requires the `Members` list to be on the `SaveState`. For v1 records (no Members field), only the owner can connect. This is correct — v1 games are solo.

### `server-v2/cmd/ws-chat/main.go`

After computing the `StateDelta`, broadcast to all connected party members. Narrative chunks and `narrative_end` also broadcast — all party members see the story stream in real time.

```go
// After NarrateStream produces a chunk, broadcast to all connections
// (not just the sender). Replace direct ws.SendNarrativeChunk calls:
gameConns, err := dbClient.GetConnectionsByGameID(ctx, gameID)
connIDs := extractConnectionIDs(gameConns)

onChunk := func(chunk string) {
    _ = ws.BroadcastFrame(ctx, connIDs, wsutil.Frame{
        Type:    "narrative_chunk",
        Payload: chunk,
    })
}

// narrative_end — broadcast
_ = ws.Broadcast(ctx, connIDs, wsutil.Frame{Type: "narrative_end"})

// StateDelta — broadcast caller's view AND each party member's view
// Each player gets a delta from their own character's perspective
for _, conn := range gameConns {
    memberDelta := g.BuildStateDelta(conn.UserID) // caller-aware delta
    _ = ws.Send(ctx, conn.ConnectionID, wsutil.Frame{
        Type:    "state_delta",
        Payload: memberDelta,
    })
}
```

**`streaming` flag is per-session, not per-connection**: when one player's message is being processed, all other players receive `streaming_blocked` if they try to send. The existing `streaming` field on the connection record is used — but the check should be against the session's connections, not just the current connection.

```go
// Check if session is already streaming (any connection in this game)
gameConns, err := dbClient.GetConnectionsByGameID(ctx, g.ID)
for _, conn := range gameConns {
    if conn.Streaming {
        _ = ws.SendError(ctx, connID, "streaming_blocked")
        return nil, nil
    }
}
// Set streaming = true on THIS connection only (other players can still do game_actions)
```

### `server-v2/cmd/ws-game-action/main.go`

After applying any game action, broadcast the delta to all connected party members:

```go
// Replace single ws.SendDelta(ctx, connID, delta) with broadcast:
gameConns, _ := dbClient.GetConnectionsByGameID(ctx, gameID)
for _, conn := range gameConns {
    memberDelta := g.BuildStateDelta(conn.UserID)
    _ = ws.Send(ctx, conn.ConnectionID, wsutil.Frame{
        Type:    "state_delta",
        Payload: memberDelta,
    })
}
```

### `server-v2/cmd/world-gen/main.go`

Replace `GetConnectionByUserID` with `GetConnectionsByGameID` for progress log broadcast:

```go
// Find all connections for this session (all party members see world-gen progress)
gameConns, err := dbClient.GetConnectionsByGameID(ctx, sessionID)
if err != nil || len(gameConns) == 0 {
    log.Printf("world-gen: no connections for session %s — proceeding silently", sessionID)
}

emit := func(line string) {
    for _, conn := range gameConns {
        _ = ws.SendWorldGenLog(ctx, conn.ConnectionID, line)
    }
}

// ... world generation ...

// world_gen_ready — broadcast to all party members
for _, conn := range gameConns {
    _ = ws.SendWorldGenReady(ctx, conn.ConnectionID)
}
```

---

## Part 5: HTTP Games API Changes

### `server-v2/cmd/http-games/main.go`

**`POST /api/games`** — caller becomes `OwnerID`; write membership record:

```go
// In handleCreateGame, after PutSaveState:
if err := dbClient.PutMembership(ctx, db.MembershipRecord{
    UserID:    db.BinaryID(userID),
    SessionID: db.BinaryID(sessionID),
    Role:      "owner",
    JoinedAt:  time.Now().UnixMilli(),
}); err != nil {
    log.Printf("http-games: PutMembership error (non-fatal): %v", err)
}
```

**`GET /api/games`** — union of owned + joined sessions:

```go
func handleListGames(ctx context.Context, userID string, ...) {
    // Query 1: owned sessions (existing user-sessions-index GSI)
    ownedIDs, err := dbClient.ListGamesByOwner(ctx, userID)

    // Query 2: all sessions user is a member of (memberships table)
    memberSessionIDs, err := dbClient.GetMemberSessions(ctx, userID)

    // Merge, deduplicate, batch-get
    allSessionIDs := deduplicate(append(ownedIDs, memberSessionIDs...))
    sessions, err := dbClient.BatchGetSessions(ctx, allSessionIDs)

    // Build list response
}
```

**`GET /api/games/{uuid}`** — allow if owner OR member:

```go
// Replace: if saveState.UserID != userID { return 403 }
// With:
if !isAuthorizedForSession(saveState, userID) {
    return jsonResponse(403, map[string]string{"error": "forbidden"}), nil
}
```

**`DELETE /api/games/{uuid}`** — owner only; clean up memberships:

```go
// Only owner can delete
if saveState.OwnerID != userID && saveState.UserID != userID {
    return jsonResponse(403, map[string]string{"error": "forbidden"}), nil
}
// After DeleteSaveState:
members, _ := dbClient.GetSessionMembers(ctx, sessionID)
for _, m := range members {
    _ = dbClient.DeleteMembership(ctx, string(m.UserID), sessionID)
}
```

---

## Part 6: Invite API

### `server-v2/cmd/http-invites/main.go`

```go
// POST /api/invites — requires auth
// Request body:
type createInviteRequest struct {
    SessionID string `json:"session_id"`
    MaxUses   int    `json:"max_uses"`   // 0 = unlimited; default 10
    TTLDays   int    `json:"ttl_days"`   // default 7
}
// Response:
type createInviteResponse struct {
    Code    string `json:"code"`
    URL     string `json:"url"`     // https://{domain}/join/{code}
    Expires int64  `json:"expires"` // Unix ms
}

// Validation: caller must be owner of the session
// Generate code: generateInviteCode() — 6-char alphanumeric
// Write InviteRecord with expires_at = now + TTLDays * 86400 (seconds, not ms — DynamoDB TTL)
// Denormalize invite_code onto SaveState record for quick lookup

// GET /api/invites/{code} — no auth
// Response:
type resolveInviteResponse struct {
    Code         string `json:"code"`
    GameTitle    string `json:"game_title"`
    OwnerName    string `json:"owner_name"`    // email of owner (from Cognito)
    PartyCurrent int    `json:"party_current"` // current member count
    PartyMax     int    `json:"party_max"`
    Expired      bool   `json:"expired"`
}

// POST /api/invites/{code}/join — requires auth
// 1. GetInvite — check not expired, not full
// 2. GetSaveState — check party not full
// 3. Check user not already a member
// 4. PutMembership (role=member)
// 5. Add player Character stub to SaveState.Players[userID]
// 6. IncrementInviteUses
// Response: { session_id: string }
// Client redirects to /create?session={session_id} for character creation
```

**Join code input on games list page**: a `TextField` where users paste a 6-character code → `POST /api/invites/{code}/join` → on success, redirect to character creation for that session.

---

## Part 7: Frontend Changes

### `client/src/types/types.ts` — new types

```typescript
export interface PartyMember {
  id: string
  name: string
  health: number
  alive: boolean
  location_id: string
  // Phase 2 will expand this with full D&D character fields
}

export interface GameStateView {
  self: CharacterView          // caller's own character
  party: PartyMember[]         // other party members
  current_room: RoomView
  // ... existing fields
}

export interface UserQuota {
  tokens_used: number
  token_limit: number          // 0 = unlimited
  ai_enabled: boolean
  role: 'admin' | 'user' | 'restricted'
}

export interface InviteInfo {
  code: string
  game_title: string
  owner_name: string
  party_current: number
  party_max: number
  expired: boolean
}
```

### `client/src/store/gameStore.ts`

```typescript
interface GameStore {
  // Existing fields...
  self: CharacterView | null
  party: PartyMember[]
  userQuota: UserQuota | null

  setGameState: (state: GameStateView) => void  // now sets self + party
  applyDelta: (delta: StateDelta) => void        // handles party member updates
  setUserQuota: (quota: UserQuota) => void
}
```

`applyDelta` must handle `delta.party` updates — when another party member's health changes or they move rooms, the delta will contain their updated `PartyMember` view.

### `client/src/routes/join.$code.tsx` (new route)

```typescript
// TanStack Router file-based route: /join/$code
// 1. On mount: GET /api/invites/{code} → show InviteInfo card
// 2. If not authenticated:
//    - Store code in sessionStorage("pendingInviteCode")
//    - Show "Create account" and "Sign in" buttons → navigate to /signup or /login
// 3. If authenticated:
//    - Show "Join [game_title]" button
//    - On click: POST /api/invites/{code}/join → { session_id }
//    - Navigate to /create?session={session_id}

// In auth.service.ts signup flow:
// After successful confirmRegistration(), check sessionStorage("pendingInviteCode")
// If present: pass it as clientMetadata.inviteCode to Cognito signUp call
// (Cognito post-confirm trigger will redeem it automatically)
```

Passing `clientMetadata` to Cognito `signUp`:
```typescript
// auth.service.ts
export async function signUp(email: string, password: string, inviteCode?: string) {
  const clientMetadata = inviteCode ? { inviteCode } : undefined
  await pool.signUp(email, password, [emailAttr], [], clientMetadata)
}
```

### `client/src/routes/game-$sessionUUID.tsx` — party panel

The right `GameInfo` column gains a **Party** section showing all connected party members with their name and HP. For the session owner, an **Invite** section shows the join code and a "Copy invite link" button.

```tsx
// Party panel (inside GameInfo or as a separate collapsible section)
{party.length > 0 && (
  <Box>
    <Typography variant="overline">Party</Typography>
    {party.map(member => (
      <Box key={member.id} sx={{ display: 'flex', justifyContent: 'space-between' }}>
        <Typography>{member.name}</Typography>
        <LinearProgress
          variant="determinate"
          value={member.health}
          color={member.health > 50 ? 'success' : member.health > 25 ? 'warning' : 'error'}
          sx={{ width: 80 }}
        />
      </Box>
    ))}
  </Box>
)}

// Invite section (owner only)
{isOwner && (
  <Box>
    <Typography variant="overline">Invite</Typography>
    <Typography variant="mono">{inviteCode || 'No active invite'}</Typography>
    <Button onClick={generateInvite}>Generate Invite Link</Button>
    <Button onClick={() => navigator.clipboard.writeText(inviteUrl)}>
      Copy Link
    </Button>
  </Box>
)}
```

### `client/src/routes/create.tsx` — join-game path

The creation wizard must handle two scenarios:
1. **New game** (existing flow) — owner creates the world + character
2. **Join existing game** (new) — party member creates only their character; world already exists

Detect via URL param: `/create?session={sessionId}` → join path.

In join path:
- Skip the "Adventure Theme" / world-gen preference steps
- Only show: Name → Class → Ability Scores → Skills → Review
- On submit: `POST /api/games/{sessionId}/join-character` (new endpoint on `http-games`) with the character creation data
- No world-gen Lambda invocation — just add the character to `SaveState.Players[userID]` and redirect to `/game-{sessionId}`

---

## Testing Notes

**DB layer tests:**
```go
// Test GetConnectionsByGameID returns multiple connections for same game_id
// Test DeleteUserConnectionForGame only deletes the matching (user, game) pair
// Test GetMemberSessions returns owned + joined sessions
// Test BatchGetSessions handles missing sessions gracefully
```

**ws-chat broadcast test:**
```go
// Mock GetConnectionsByGameID to return 3 connections
// Send a chat message
// Assert all 3 connections received narrative_chunk frames
// Assert all 3 received state_delta
// Mock one connection as GoneException → assert that connection is deleted from DB
```

**ws-connect auth test:**
```go
// v1 SaveState (no Members) — only owner can connect
// v2 SaveState — owner and any member in Members list can connect
// user not in Members → 403
```

**Frontend party tests:**
```typescript
// Render game route with party: [memberA, memberB]
// Assert party panel shows both members
// Assert applyDelta with party member health update reflects in panel
// Render join route with valid invite code → shows invite card
// Render join route with expired invite → shows "expired" message
```

---

## Rollout Checklist

- [ ] Terraform deployed: memberships table, game_id GSI on connections, invites table, http-invites Lambda
- [ ] Deploy updated ws-connect, ws-chat, ws-game-action, world-gen
- [ ] Deploy updated http-games
- [ ] Deploy http-invites
- [ ] Test solo session: existing game still loads (v1 migration)
- [ ] Test party: two accounts, owner creates game, invites second player via link
- [ ] Test join code: second player enters code in games list
- [ ] Test broadcast: second player sees narrative chunks in real time
- [ ] Test stale connection cleanup: disconnect one player mid-session, other still receives updates
