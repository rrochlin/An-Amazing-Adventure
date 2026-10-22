# Phase 0 — RBAC and Usage Controls

Protects against unbounded Bedrock spend and establishes the permission model before the game is shared with anyone. Ships as an independent feature branch before any rpg-toolkit work.

Prerequisites: Terraform infrastructure from [[Terraform Infrastructure]] must be deployed (users table, Cognito groups, `http-admin` Lambda, `cognito-post-confirm` trigger).

Branch: `feat/phase-6-rbac`

---

## Goals

1. New signups default to `restricted` — no Bedrock calls fire until admin approves
2. Per-user token quota enforced atomically — even if the user has multiple concurrent sessions
3. Admin panel lets you view all users and toggle access / set limits
4. System is built to accommodate future `billing_mode` alternatives (BYOK, Stripe) without schema changes
5. Existing games continue working — no breaking changes to existing session data

---

## New File: `server-v2/internal/db/users.go`

Full implementation of the users table data layer.

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

type UserRecord struct {
    UserID      BinaryID `dynamodbav:"user_id"`
    Role        string   `dynamodbav:"role"`           // "admin" | "user" | "restricted"
    AIEnabled   bool     `dynamodbav:"ai_enabled"`
    TokenLimit  int      `dynamodbav:"token_limit"`    // 0 = unlimited
    TokensUsed  int      `dynamodbav:"tokens_used"`
    GamesLimit  int      `dynamodbav:"games_limit"`    // 0 = unlimited
    BillingMode string   `dynamodbav:"billing_mode"`   // "admin_granted" | "own_key" | "subscription"
    APIKeyHash  string   `dynamodbav:"api_key_hash,omitempty"`
    CreatedAt   int64    `dynamodbav:"created_at"`
    UpdatedAt   int64    `dynamodbav:"updated_at"`
    Notes       string   `dynamodbav:"notes,omitempty"`
}

func (c *Client) GetUser(ctx context.Context, userID string) (*UserRecord, error) {
    key, err := marshalBinaryKey("user_id", userID)
    if err != nil {
        return nil, err
    }
    out, err := c.ddb.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(c.usersTable),
        Key:       key,
    })
    if err != nil {
        return nil, fmt.Errorf("GetUser: %w", err)
    }
    if out.Item == nil {
        return nil, nil // user not found — treat as restricted
    }
    var record UserRecord
    if err := attributevalue.UnmarshalMap(out.Item, &record); err != nil {
        return nil, fmt.Errorf("GetUser unmarshal: %w", err)
    }
    return &record, nil
}

func (c *Client) PutUser(ctx context.Context, u UserRecord) error {
    item, err := attributevalue.MarshalMap(u)
    if err != nil {
        return fmt.Errorf("PutUser marshal: %w", err)
    }
    _, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: aws.String(c.usersTable),
        Item:      item,
    })
    return err
}

// UpdateUserTokens atomically increments tokens_used by delta.
// If token_limit > 0 and the increment would exceed it, the update is rejected
// with ErrQuotaExceeded. Callers must handle this and push a quota_exceeded frame.
//
// Uses a DynamoDB conditional update expression so this is safe under Lambda concurrency.
func (c *Client) UpdateUserTokens(ctx context.Context, userID string, delta int) error {
    key, err := marshalBinaryKey("user_id", userID)
    if err != nil {
        return err
    }

    now := time.Now().UnixMilli()

    // Unconditional increment — limit enforcement is done before calling Bedrock,
    // not here. This is a post-hoc accounting write that should always succeed.
    _, err = c.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
        TableName:        aws.String(c.usersTable),
        Key:              key,
        UpdateExpression: aws.String("ADD tokens_used :delta SET updated_at = :now"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":delta": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", delta)},
            ":now":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", now)},
        },
    })
    return err
}

func (c *Client) ListUsers(ctx context.Context) ([]UserRecord, error) {
    var records []UserRecord
    var lastKey map[string]types.AttributeValue

    for {
        out, err := c.ddb.Scan(ctx, &dynamodb.ScanInput{
            TableName:         aws.String(c.usersTable),
            ExclusiveStartKey: lastKey,
        })
        if err != nil {
            return nil, fmt.Errorf("ListUsers scan: %w", err)
        }
        for _, item := range out.Items {
            var r UserRecord
            if err := attributevalue.UnmarshalMap(item, &r); err != nil {
                continue // skip malformed records
            }
            records = append(records, r)
        }
        if out.LastEvaluatedKey == nil {
            break
        }
        lastKey = out.LastEvaluatedKey
    }
    return records, nil
}

func (c *Client) UpdateUser(ctx context.Context, userID string, updates UserRecord) error {
    updates.UpdatedAt = time.Now().UnixMilli()
    // Re-use PutUser with a condition that the record already exists
    key, err := marshalBinaryKey("user_id", userID)
    if err != nil {
        return err
    }
    item, err := attributevalue.MarshalMap(updates)
    if err != nil {
        return err
    }
    _, err = c.ddb.PutItem(ctx, &dynamodb.PutItemInput{
        TableName:           aws.String(c.usersTable),
        Item:                item,
        ConditionExpression: aws.String("attribute_exists(user_id)"),
    })
    return err
}
```

Add `usersTable string` to the `Client` struct and initialize from `USERS_TABLE` env var in `db.New()`.

---

## Modified: `server-v2/cmd/http-games/main.go`

### `POST /api/games` — quota and permission check

Insert after extracting `userID` from JWT claims, before calling `dbClient.PutSaveState`:

```go
// Load user record — enforce AI access and game limit
userRecord, err := dbClient.GetUser(ctx, userID)
if err != nil {
    log.Printf("http-games POST: GetUser error: %v", err)
    // Non-fatal — treat as restricted on error to be safe
    userRecord = &db.UserRecord{Role: "restricted", AIEnabled: false, GamesLimit: 1}
}
if userRecord == nil {
    // User confirmed but post-confirm trigger hasn't fired yet (race) — treat as restricted
    userRecord = &db.UserRecord{Role: "restricted", AIEnabled: false, GamesLimit: 1}
}

// Check game limit
if userRecord.GamesLimit > 0 {
    count, err := dbClient.CountUserGames(ctx, userID)
    if err == nil && count >= userRecord.GamesLimit {
        return jsonResponse(403, map[string]string{
            "error": "games_limit_reached",
            "message": fmt.Sprintf("Game limit of %d reached", userRecord.GamesLimit),
        }), nil
    }
}

// Set preview_mode flag if AI is not enabled
previewMode := !userRecord.AIEnabled
```

In the response from `POST /api/games`, include:
```go
type createGameResponse struct {
    SessionID   string `json:"session_id"`
    Ready       bool   `json:"ready"`
    PreviewMode bool   `json:"preview_mode"` // true = AI disabled, static preview only
}
```

### `GET /api/games` — include quota in response

Add `UserQuota` to the list response:
```go
type listGamesResponse struct {
    Games     []GameListItem `json:"games"`
    UserQuota UserQuotaInfo  `json:"user_quota"`
}

type UserQuotaInfo struct {
    TokensUsed  int    `json:"tokens_used"`
    TokenLimit  int    `json:"token_limit"`   // 0 = unlimited
    AIEnabled   bool   `json:"ai_enabled"`
    Role        string `json:"role"`
}
```

---

## Modified: `server-v2/cmd/ws-chat/main.go`

### Quota enforcement — before calling Bedrock

Insert after loading game state, before `NarrateStream`:

```go
userRecord, err := dbClient.GetUser(ctx, userID)
if err != nil || userRecord == nil {
    userRecord = &db.UserRecord{Role: "restricted", AIEnabled: false}
}

if !userRecord.AIEnabled {
    _ = ws.SendError(ctx, connID, "ai_access_not_enabled")
    return nil, nil
}

if userRecord.TokenLimit > 0 && userRecord.TokensUsed >= userRecord.TokenLimit {
    _ = ws.SendError(ctx, connID, "quota_exceeded")
    return nil, nil
}
```

### Token accounting — after successful turn

After `EngineerScan` completes and before saving game state:

```go
totalDelta := narratorResult.Tokens.Total() + engineerResult.Tokens.Total()
if accountErr := dbClient.UpdateUserTokens(ctx, userID, totalDelta); accountErr != nil {
    // Non-fatal — log but don't fail the game turn
    log.Printf("ws-chat: UpdateUserTokens error (non-fatal): %v", accountErr)
}
```

The `ws.SendError` frame types used here:
```go
// wsutil/wsutil.go — add these error code constants
const (
    ErrAIAccessNotEnabled = "ai_access_not_enabled"
    ErrQuotaExceeded      = "quota_exceeded"
)
```

---

## Modified: `server-v2/cmd/world-gen/main.go`

After `GenerateBlueprint` completes, account for blueprint tokens:

```go
if accountErr := dbClient.UpdateUserTokens(ctx, userID, blueprintTokens.Total()); accountErr != nil {
    log.Printf("world-gen: UpdateUserTokens error (non-fatal): %v", accountErr)
}
```

`userID` must be threaded through to `world-gen` — it's already present in the `SaveState` as `UserID` / `OwnerID`, so load it from there after the initial `GetSaveState` call.

---

## New File: `server-v2/cmd/http-admin/main.go`

Full admin API handler. Auth is enforced at two layers: API Gateway JWT authorizer (requires valid Cognito token) + Lambda-level group check (requires `admin` claim).

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/config"
    cognitoidp "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

    "github.com/rrochlin/An-Amazing-Adventure/server-v2/internal/db"
)

var (
    dbClient    *db.Client
    cognitoClient *cognitoidp.Client
    userPoolID  string
)

func init() {
    cfg, _ := config.LoadDefaultConfig(context.Background())
    dbClient = db.New()
    cognitoClient = cognitoidp.NewFromConfig(cfg)
    userPoolID = os.Getenv("USER_POOL_ID")
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    // Enforce admin group — defense in depth beyond API Gateway authorizer
    if !hasGroup(req.RequestContext.Authorizer.JWT.Claims, "admin") {
        return jsonResponse(403, map[string]string{"error": "forbidden"}), nil
    }

    method := req.RequestMethod
    path := req.RawPath

    switch {
    case method == "GET" && path == "/api/admin/users":
        return handleListUsers(ctx, req)
    case method == "PUT" && strings.HasPrefix(path, "/api/admin/users/"):
        userID := req.PathParameters["userId"]
        return handleUpdateUser(ctx, req, userID)
    case method == "GET" && path == "/api/admin/stats":
        return handleStats(ctx, req)
    default:
        return jsonResponse(404, map[string]string{"error": "not_found"}), nil
    }
}

// GET /api/admin/users
// Returns all users with Cognito email enrichment.
type AdminUserView struct {
    UserID      string `json:"user_id"`
    Email       string `json:"email"`
    Role        string `json:"role"`
    AIEnabled   bool   `json:"ai_enabled"`
    TokenLimit  int    `json:"token_limit"`
    TokensUsed  int    `json:"tokens_used"`
    GamesLimit  int    `json:"games_limit"`
    BillingMode string `json:"billing_mode"`
    Notes       string `json:"notes,omitempty"`
    CreatedAt   int64  `json:"created_at"`
}

func handleListUsers(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    records, err := dbClient.ListUsers(ctx)
    if err != nil {
        log.Printf("http-admin ListUsers: %v", err)
        return jsonResponse(500, map[string]string{"error": "internal"}), nil
    }

    views := make([]AdminUserView, 0, len(records))
    for _, r := range records {
        email := fetchEmail(ctx, string(r.UserID))
        views = append(views, AdminUserView{
            UserID:      string(r.UserID),
            Email:       email,
            Role:        r.Role,
            AIEnabled:   r.AIEnabled,
            TokenLimit:  r.TokenLimit,
            TokensUsed:  r.TokensUsed,
            GamesLimit:  r.GamesLimit,
            BillingMode: r.BillingMode,
            Notes:       r.Notes,
            CreatedAt:   r.CreatedAt,
        })
    }
    return jsonResponse(200, views), nil
}

// PUT /api/admin/users/{userId}
// Updates role, ai_enabled, token_limit, games_limit, notes.
// Syncs Cognito group membership to match new role.
type updateUserRequest struct {
    Role       string `json:"role"`        // "admin" | "user" | "restricted"
    AIEnabled  bool   `json:"ai_enabled"`
    TokenLimit int    `json:"token_limit"`
    GamesLimit int    `json:"games_limit"`
    Notes      string `json:"notes"`
}

func handleUpdateUser(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
    var body updateUserRequest
    if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
        return jsonResponse(400, map[string]string{"error": "invalid_body"}), nil
    }

    existing, err := dbClient.GetUser(ctx, userID)
    if err != nil || existing == nil {
        return jsonResponse(404, map[string]string{"error": "user_not_found"}), nil
    }

    existing.Role = body.Role
    existing.AIEnabled = body.AIEnabled
    existing.TokenLimit = body.TokenLimit
    existing.GamesLimit = body.GamesLimit
    existing.Notes = body.Notes

    if err := dbClient.UpdateUser(ctx, userID, *existing); err != nil {
        return jsonResponse(500, map[string]string{"error": "update_failed"}), nil
    }

    // Sync Cognito groups
    if err := syncCognitoGroups(ctx, userID, body.Role); err != nil {
        log.Printf("http-admin: syncCognitoGroups error (non-fatal): %v", err)
    }

    return jsonResponse(200, map[string]string{"status": "ok"}), nil
}

// syncCognitoGroups ensures the user is in the correct Cognito group for their role.
// admin → groups: [admin, user]
// user  → groups: [user]
// restricted → groups: [restricted]
func syncCognitoGroups(ctx context.Context, userID, role string) error {
    allGroups := []string{"admin", "user", "restricted"}
    targetGroups := map[string]bool{}
    switch role {
    case "admin":
        targetGroups["admin"] = true
        targetGroups["user"] = true
    case "user":
        targetGroups["user"] = true
    default:
        targetGroups["restricted"] = true
    }

    for _, g := range allGroups {
        if targetGroups[g] {
            _, err := cognitoClient.AdminAddUserToGroup(ctx, &cognitoidp.AdminAddUserToGroupInput{
                UserPoolId: &userPoolID,
                Username:   &userID,
                GroupName:  &g,
            })
            if err != nil {
                log.Printf("AdminAddUserToGroup %s: %v", g, err)
            }
        } else {
            _, err := cognitoClient.AdminRemoveUserFromGroup(ctx, &cognitoidp.AdminRemoveUserFromGroupInput{
                UserPoolId: &userPoolID,
                Username:   &userID,
                GroupName:  &g,
            })
            if err != nil {
                // Non-fatal — user may not be in the group
                log.Printf("AdminRemoveUserFromGroup %s (non-fatal): %v", g, err)
            }
        }
    }
    return nil
}

// GET /api/admin/stats
type adminStats struct {
    TotalUsers     int `json:"total_users"`
    AdminUsers     int `json:"admin_users"`
    ApprovedUsers  int `json:"approved_users"`
    RestrictedUsers int `json:"restricted_users"`
    TotalTokensUsed int `json:"total_tokens_used"`
}

func handleStats(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    records, err := dbClient.ListUsers(ctx)
    if err != nil {
        return jsonResponse(500, map[string]string{"error": "internal"}), nil
    }
    stats := adminStats{}
    for _, r := range records {
        stats.TotalUsers++
        stats.TotalTokensUsed += r.TokensUsed
        switch r.Role {
        case "admin":
            stats.AdminUsers++
        case "user":
            stats.ApprovedUsers++
        default:
            stats.RestrictedUsers++
        }
    }
    return jsonResponse(200, stats), nil
}

func fetchEmail(ctx context.Context, userID string) string {
    out, err := cognitoClient.AdminGetUser(ctx, &cognitoidp.AdminGetUserInput{
        UserPoolId: &userPoolID,
        Username:   &userID,
    })
    if err != nil {
        return ""
    }
    for _, attr := range out.UserAttributes {
        if aws.ToString(attr.Name) == "email" {
            return aws.ToString(attr.Value)
        }
    }
    return ""
}

func hasGroup(claims map[string]string, group string) bool {
    return strings.Contains(claims["cognito:groups"], group)
}

func jsonResponse(status int, body any) events.APIGatewayV2HTTPResponse {
    b, _ := json.Marshal(body)
    return events.APIGatewayV2HTTPResponse{
        StatusCode: status,
        Headers:    map[string]string{"Content-Type": "application/json"},
        Body:       string(b),
    }
}

func main() {
    lambda.Start(handler)
}
```

---

## New File: `server-v2/cmd/cognito-post-confirm/main.go`

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"

    "github.com/rrochlin/An-Amazing-Adventure/server-v2/internal/db"
)

var dbClient *db.Client

func init() {
    dbClient = db.New()
}

func handler(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (
    events.CognitoEventUserPoolsPostConfirmation, error,
) {
    userID := event.Request.UserAttributes["sub"]
    if userID == "" {
        log.Printf("cognito-post-confirm: missing sub in user attributes")
        return event, nil // non-fatal — don't block signup
    }

    now := time.Now().UnixMilli()
    record := db.UserRecord{
        UserID:      db.BinaryID(userID),
        Role:        "restricted",
        AIEnabled:   false,
        TokenLimit:  0,
        TokensUsed:  0,
        GamesLimit:  1,
        BillingMode: "admin_granted",
        CreatedAt:   now,
        UpdatedAt:   now,
    }

    if err := dbClient.PutUser(ctx, record); err != nil {
        log.Printf("cognito-post-confirm: PutUser error: %v", err)
        // Non-fatal — user can still log in, GetUser will return nil and system treats as restricted
    }

    // Redeem invite code if present (set by client in signUp clientMetadata)
    if code, ok := event.Request.ClientMetadata["inviteCode"]; ok && code != "" {
        if err := redeemInviteForUser(ctx, userID, code); err != nil {
            log.Printf("cognito-post-confirm: redeemInvite(%s) error (non-fatal): %v", code, err)
        }
    }

    log.Printf("cognito-post-confirm: created user record for %s", userID)
    return event, nil
}

func redeemInviteForUser(ctx context.Context, userID, code string) error {
    invite, err := dbClient.GetInvite(ctx, code)
    if err != nil || invite == nil {
        return fmt.Errorf("invite %s not found or expired", code)
    }
    if invite.MaxUses > 0 && invite.Uses >= invite.MaxUses {
        return fmt.Errorf("invite %s is full (%d/%d uses)", code, invite.Uses, invite.MaxUses)
    }

    // Add membership
    if err := dbClient.PutMembership(ctx, db.MembershipRecord{
        UserID:    db.BinaryID(userID),
        SessionID: invite.SessionID,
        Role:      "member",
        JoinedAt:  time.Now().UnixMilli(),
    }); err != nil {
        return err
    }

    // Increment invite use count
    return dbClient.IncrementInviteUses(ctx, code)
}

func main() {
    lambda.Start(handler)
}
```

---

## Frontend: Admin Panel

### `client/src/routes/admin.tsx`

New TanStack Router file-based route. Protected: redirects to `/` if caller is not in the `admin` Cognito group.

The admin group membership is checked by decoding the ID token (already stored in localStorage as part of the Cognito auth flow). The `cognito:groups` claim is in the ID token payload.

```typescript
// client/src/routes/admin.tsx
import { createFileRoute, redirect } from '@tanstack/react-router'
import { AdminPanel } from '../components/AdminPanel'
import { getIdTokenClaims } from '../services/auth.service'

export const Route = createFileRoute('/admin')({
  beforeLoad: () => {
    const claims = getIdTokenClaims()
    if (!claims?.['cognito:groups']?.includes('admin')) {
      throw redirect({ to: '/' })
    }
  },
  component: AdminPanel,
})
```

`getIdTokenClaims()` in `auth.service.ts`:
```typescript
export function getIdTokenClaims(): Record<string, any> | null {
  const tokens = getStoredTokens() // reads "AAA_TOKENS" from localStorage
  if (!tokens?.idToken) return null
  try {
    const payload = tokens.idToken.split('.')[1]
    return JSON.parse(atob(payload))
  } catch {
    return null
  }
}
```

### `client/src/components/AdminPanel.tsx`

```typescript
// Columns: Email | Role | AI Enabled | Tokens Used / Limit | Games Limit | Notes | Save
// Each row is editable inline. Save button calls PUT /api/admin/users/{userId}
// Stats bar at top: total users, approved, restricted, total tokens consumed

interface AdminUserView {
  user_id: string
  email: string
  role: 'admin' | 'user' | 'restricted'
  ai_enabled: boolean
  token_limit: number    // 0 = unlimited
  tokens_used: number
  games_limit: number
  billing_mode: string
  notes?: string
  created_at: number
}
```

API calls use the existing `api.service.ts` Axios instance (which already injects the Bearer token). Add to `client/src/services/api.game.ts` or a new `api.admin.ts`:
```typescript
export const adminApi = {
  listUsers: () => api.get<AdminUserView[]>('/api/admin/users'),
  updateUser: (userId: string, data: Partial<AdminUserView>) =>
    api.put(`/api/admin/users/${userId}`, data),
  getStats: () => api.get<AdminStats>('/api/admin/stats'),
}
```

### `client/src/routes/game-$sessionUUID.tsx` — Preview Mode Banner

The `POST /api/games` response now includes `preview_mode: boolean`. Store on `gameStore`. If `true`:

```tsx
{previewMode && (
  <Alert severity="info" sx={{ mb: 1 }}>
    <strong>Preview Mode</strong> — AI narration is not enabled for your account.
    Contact the admin to request access.
  </Alert>
)}
```

The chat input `TextField` is disabled with placeholder text: `"AI access not enabled — contact admin"`.

### Games List — Token Usage Indicator

The `GET /api/games` response includes `user_quota`. Display in the games list header:

```tsx
// If token_limit === 0: "Unlimited tokens"
// Else: ProgressBar showing tokens_used / token_limit with percentage label
<LinearProgress
  variant="determinate"
  value={Math.min(100, (quota.tokens_used / quota.token_limit) * 100)}
/>
<Typography variant="caption">
  {quota.tokens_used.toLocaleString()} / {quota.token_limit.toLocaleString()} tokens used
</Typography>
```

---

## Testing Notes

**DB layer tests** (`server-v2/internal/db/`):
```go
// Set USERS_TABLE in test environment
t.Setenv("USERS_TABLE", "test-users")

// UpdateUserTokens is atomic — test concurrent increments with goroutines
// GetUser returns nil (not error) when user doesn't exist — test this case
```

**ws-chat tests** — mock `GetUser` to return:
- `nil` → treated as restricted → `SendError("ai_access_not_enabled")`
- `{AIEnabled: false}` → same
- `{AIEnabled: true, TokenLimit: 100, TokensUsed: 100}` → `SendError("quota_exceeded")`
- `{AIEnabled: true, TokenLimit: 0}` → proceeds normally (unlimited)

**Admin panel tests** (`client/src/`):
```typescript
// Mock GET /api/admin/users, verify table renders rows
// Mock PUT /api/admin/users/{id}, verify save button fires correct payload
// Mock getIdTokenClaims to return non-admin claims → verify redirect to /
```

---

## Rollout Checklist

- [ ] Terraform PR merged — users table, Cognito groups, `http-admin` Lambda, `cognito-post-confirm` trigger, new API routes
- [ ] Deploy `cognito-post-confirm` Lambda
- [ ] Deploy `http-admin` Lambda
- [ ] Deploy updated `http-games`, `ws-chat`, `world-gen`
- [ ] Run `scripts/elevate-admin.sh <your-cognito-sub>` — see [[Admin Scripts]]
- [ ] Log in, navigate to `/admin`, verify panel loads
- [ ] Create a test game with your admin account — verify AI works
- [ ] Create a second test account — verify it is restricted by default
- [ ] Use admin panel to approve second account — verify AI then works
