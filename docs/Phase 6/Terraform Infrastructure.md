# Terraform Infrastructure

All infrastructure changes for Phase 6 live in the `rrochlin/terraform-infrastructure` repo, pushed via git subtree from `server/infra/` in the monorepo. These changes are a prerequisite for all Phase 6 application work and must ship first.

```bash
# Push to infra repo
git subtree push --prefix server/infra git@github.com:rrochlin/terraform-infrastructure.git feat/phase-6-infra
# Then open PR on rrochlin/terraform-infrastructure
```

Current Terraform working directory: `server/infra/amazing-adventure/`
Run with: `doppler run --project terraform-personal-infra --config dev_personal -- terraform plan`

---

## Existing Infrastructure (do not change)

| Resource | Module | Notes |
|---|---|---|
| `{prefix}-sessions` DynamoDB | `modules/dynamodb` | PK: `session_id` (B), GSI: `user_id` (B) |
| `{prefix}-connections` DynamoDB | `modules/dynamodb` | PK: `connection_id` (S), GSI: `user_id` (B) |
| `{prefix}-mutations` DynamoDB | `modules/dynamodb` | PK: `session_id` (B), SK: `ts` (N) |
| Cognito User Pool | `modules/cognito` | SRP flow, email auth, no groups currently |
| All existing Lambdas | `modules/lambdas` | `ws-connect`, `ws-disconnect`, `ws-chat`, `ws-game-action`, `http-games`, `http-users`, `world-gen` |
| HTTP API Gateway | `modules/api-gateway` | Routes: `/api/games`, `/api/users` |
| WebSocket API Gateway | `modules/api-gateway` | Routes: `$connect`, `$disconnect`, `chat`, `game_action` |
| CloudFront + S3 | `modules/cloudfront`, `modules/s3` | Client hosting |

---

## New DynamoDB Tables

### `{prefix}-users`

Stores per-user profile, role, and usage quota. The `billing_mode` field is an extension point for future BYOK or subscription payment without a schema change.

```hcl
resource "aws_dynamodb_table" "users" {
  name         = "${var.prefix}-users"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "user_id"

  attribute {
    name = "user_id"
    type = "B"   # Binary — matches Cognito sub UUID stored as BinaryID
  }

  ttl {
    enabled        = false
    attribute_name = "expires_at"
  }

  tags = var.tags
}
```

**Item schema** (Go struct for reference):
```go
// server-v2/internal/db/users.go
type UserRecord struct {
    UserID      db.BinaryID `dynamodbav:"user_id"`       // PK — Cognito sub
    Role        string      `dynamodbav:"role"`           // "admin" | "user" | "restricted"
    AIEnabled   bool        `dynamodbav:"ai_enabled"`
    TokenLimit  int         `dynamodbav:"token_limit"`    // 0 = unlimited
    TokensUsed  int         `dynamodbav:"tokens_used"`    // running total across all games
    GamesLimit  int         `dynamodbav:"games_limit"`    // 0 = unlimited
    BillingMode string      `dynamodbav:"billing_mode"`   // "admin_granted" | "own_key" | "subscription"
    APIKeyHash  string      `dynamodbav:"api_key_hash,omitempty"` // future BYOK — hashed
    CreatedAt   int64       `dynamodbav:"created_at"`     // Unix ms
    UpdatedAt   int64       `dynamodbav:"updated_at"`     // Unix ms
    Notes       string      `dynamodbav:"notes,omitempty"` // admin freetext
}
```

**Default values** for new users (set by `cognito-post-confirm` trigger):
```go
UserRecord{
    Role:        "restricted",
    AIEnabled:   false,
    TokenLimit:  0,         // no tokens until admin grants access
    TokensUsed:  0,
    GamesLimit:  1,         // can create 1 game for preview purposes
    BillingMode: "admin_granted",
}
```

---

### `{prefix}-invites`

Stores party invite codes. TTL causes codes to auto-expire; they are also invalidated after `max_uses` redemptions.

```hcl
resource "aws_dynamodb_table" "invites" {
  name         = "${var.prefix}-invites"
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "code"

  attribute {
    name = "code"
    type = "S"
  }

  ttl {
    enabled        = true
    attribute_name = "expires_at"  # Unix epoch seconds (not ms — DynamoDB TTL uses seconds)
  }

  tags = var.tags
}
```

**Item schema:**
```go
// server-v2/internal/db/invites.go
type InviteRecord struct {
    Code         string      `dynamodbav:"code"`           // PK — 6-char alphanumeric e.g. "AX7K2M"
    SessionID    db.BinaryID `dynamodbav:"session_id"`     // game session this invite is for
    CreatedBy    db.BinaryID `dynamodbav:"created_by"`     // user_id of session owner
    ExpiresAt    int64       `dynamodbav:"expires_at"`     // Unix epoch SECONDS (DynamoDB TTL)
    MaxUses      int         `dynamodbav:"max_uses"`       // 0 = unlimited
    Uses         int         `dynamodbav:"uses"`           // current redemption count
    RedirectGame bool        `dynamodbav:"redirect_game"`  // true = after signup, land in this game
}
```

**Code generation** — 6 characters from `[A-Z0-9]` minus ambiguous chars (O, 0, I, 1):
```go
// 32 unambiguous chars: ABCDEFGHJKLMNPQRSTUVWXYZ23456789
func generateInviteCode() string {
    const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
    b := make([]byte, 6)
    for i := range b {
        n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
        b[i] = chars[n.Int64()]
    }
    return string(b)
}
```

---

### `{prefix}-memberships`

Many-to-many join table between users and game sessions. Enables a user to list all sessions they belong to (owned or joined as party member) without scanning the sessions table.

```hcl
resource "aws_dynamodb_table" "memberships" {
  name         = "${var.prefix}-memberships"
  billing_mode = "PAY_PER_REQUEST"

  hash_key  = "user_id"
  range_key = "session_id"

  attribute {
    name = "user_id"
    type = "B"
  }

  attribute {
    name = "session_id"
    type = "B"
  }

  # Reverse lookup: all members of a session
  global_secondary_index {
    name            = "session-members-index"
    hash_key        = "session_id"
    projection_type = "ALL"
  }

  tags = var.tags
}
```

**Item schema:**
```go
// server-v2/internal/db/memberships.go
type MembershipRecord struct {
    UserID    db.BinaryID `dynamodbav:"user_id"`    // PK
    SessionID db.BinaryID `dynamodbav:"session_id"` // SK
    Role      string      `dynamodbav:"role"`        // "owner" | "member"
    JoinedAt  int64       `dynamodbav:"joined_at"`   // Unix ms
}
```

A membership record is written for the **owner** when they create the game (role=`owner`), and for each **party member** when they redeem an invite code (role=`member`).

This makes `GET /api/games` a simple two-query union:
1. Query `user-sessions-index` on sessions table (owned games — existing GSI)
2. Query `memberships` table by `user_id` to get `session_id`s, then batch-get from sessions

---

## Modified DynamoDB Tables

### `{prefix}-connections` — add `game_id` GSI

Adds an index on `game_id` so all WebSocket connections for a game session can be fetched in one query. Required for broadcast (`wsutil.Broadcast`).

```hcl
# Add to existing connections table definition:
global_secondary_index {
  name            = "game-connections-index"
  hash_key        = "game_id"
  projection_type = "ALL"
}

attribute {
  name = "game_id"
  type = "S"   # String — API Gateway assigns plain string game IDs
}
```

**New DB method:**
```go
// server-v2/internal/db/db.go
func (c *Client) GetConnectionsByGameID(ctx context.Context, gameID string) ([]Connection, error) {
    out, err := c.ddb.Query(ctx, &dynamodb.QueryInput{
        TableName:              aws.String(c.connectionsTable),
        IndexName:              aws.String("game-connections-index"),
        KeyConditionExpression: aws.String("game_id = :gid"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":gid": &types.AttributeValueMemberS{Value: gameID},
        },
    })
    // ... unmarshal []Connection
}
```

---

## New Cognito Groups

Added to `modules/cognito/main.tf`. Note: the correct AWS provider resource name is `aws_cognito_user_group` (not `aws_cognito_user_pool_group`).

```hcl
# modules/cognito/main.tf

resource "aws_cognito_user_group" "admin" {
  name         = "admin"
  user_pool_id = aws_cognito_user_pool.main.id
  precedence   = 1
}

resource "aws_cognito_user_group" "user" {
  name         = "user"
  user_pool_id = aws_cognito_user_pool.main.id
  precedence   = 10
}

resource "aws_cognito_user_group" "restricted" {
  name         = "restricted"
  user_pool_id = aws_cognito_user_pool.main.id
  precedence   = 100
}
```

Group membership surfaces in the JWT as the `cognito:groups` claim (array of group names). API Gateway passes this claim through to Lambda in the JWT authorizer context — readable as:
```go
groups := req.RequestContext.Authorizer.JWT.Claims["cognito:groups"] // e.g. "[admin user]"
```

Note: Cognito encodes the groups claim as a space-separated string in a square-bracket wrapper when multiple groups exist. Parse with:
```go
func hasGroup(claims map[string]string, group string) bool {
    raw := claims["cognito:groups"]
    return strings.Contains(raw, group)
}
```

---

## New Lambdas

### `cognito-post-confirm`

A Cognito **Post Confirmation** trigger Lambda — fires after a user confirms their email (completes sign-up). Creates the default `UserRecord` in the users table. If `clientMetadata` contains an `inviteCode`, the invite is redeemed (adds user to the session's members).

**Trigger configuration (Terraform):**

There is no standalone `aws_cognito_user_pool_lambda_config` resource in the AWS provider. The Lambda trigger must be set as an inline `lambda_config` block on `aws_cognito_user_pool`. To avoid a circular dependency (cognito module needs the Lambda ARN; lambdas module needs `user_pool_id`), the cognito module accepts an optional `post_confirm_lambda_arn` variable and uses a `dynamic` block:

```hcl
# modules/cognito/main.tf

variable "post_confirm_lambda_arn" {
  type    = string
  default = ""
}

resource "aws_cognito_user_pool" "main" {
  # ... existing config ...

  dynamic "lambda_config" {
    for_each = var.post_confirm_lambda_arn != "" ? [1] : []
    content {
      post_confirmation = var.post_confirm_lambda_arn
    }
  }
}
```

The Lambda ARN is passed from the root `main.tf`:
```hcl
# main.tf (root)
module "cognito" {
  source                  = "./modules/cognito"
  prefix                  = local.prefix
  common_tags             = local.common_tags
  post_confirm_lambda_arn = module.lambdas.cognito_post_confirm_function_arn
}
```

Terraform resolves this dependency graph correctly — it is not a circular dependency because the graph edges are: `lambdas` reads `cognito.user_pool_id`; `cognito` reads `lambdas.cognito_post_confirm_function_arn`. These form a DAG, not a cycle.

The `aws_lambda_permission` allowing Cognito to invoke the function lives in root `main.tf` alongside the S3 bucket policy (same pattern for cross-module permissions):
```hcl
# main.tf (root)
resource "aws_lambda_permission" "cognito_post_confirm" {
  statement_id  = "AllowCognitoInvoke"
  action        = "lambda:InvokeFunction"
  function_name = module.lambdas.cognito_post_confirm_function_name
  principal     = "cognito-idp.amazonaws.com"
  source_arn    = module.cognito.user_pool_arn
}
```

**Lambda config** (`modules/lambdas/cognito-post-confirm.tf`):
```hcl
resource "aws_lambda_function" "cognito_post_confirm" {
  function_name = "${var.prefix}-cognito-post-confirm"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 10
  memory_size   = 128

  environment {
    variables = {
      SESSIONS_TABLE    = var.sessions_table_name
      USERS_TABLE       = var.users_table_name
      INVITES_TABLE     = var.invites_table_name
      MEMBERSHIPS_TABLE = var.memberships_table_name
    }
  }
}

# IAM: DynamoDB PutItem on users, GetItem/UpdateItem on invites, PutItem on memberships
```

**Handler entrypoint:** `server-v2/cmd/cognito-post-confirm/main.go`

The event type is `events.CognitoEventUserPoolsPostConfirmation` from `github.com/aws/aws-lambda-go/events`.

```go
func handler(ctx context.Context, event events.CognitoEventUserPoolsPostConfirmation) (
    events.CognitoEventUserPoolsPostConfirmation, error,
) {
    userID := event.Request.UserAttributes["sub"]
    now := time.Now().UnixMilli()

    // Create default user record
    err := dbClient.PutUser(ctx, db.UserRecord{
        UserID:      db.BinaryID(userID),
        Role:        "restricted",
        AIEnabled:   false,
        TokenLimit:  0,
        TokensUsed:  0,
        GamesLimit:  1,
        BillingMode: "admin_granted",
        CreatedAt:   now,
        UpdatedAt:   now,
    })
    if err != nil { return event, err }

    // Add to 'restricted' Cognito group (informational only — actual enforcement is in DB)
    // Skipped here to avoid circular dependency; elevate-admin.sh handles initial admin setup

    // Redeem invite code if present in clientMetadata
    if code, ok := event.Request.ClientMetadata["inviteCode"]; ok && code != "" {
        _ = redeemInvite(ctx, dbClient, userID, code) // non-fatal if invite expired
    }

    return event, nil
}
```

---

### `http-admin`

Admin management API. Enforces `admin` group membership at the Lambda level (not just API Gateway — defense in depth).

**Lambda config:**
```hcl
resource "aws_lambda_function" "http_admin" {
  function_name = "${var.prefix}-http-admin"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 15
  memory_size   = 128

  environment {
    variables = {
      USERS_TABLE    = aws_dynamodb_table.users.name
      USER_POOL_ID   = aws_cognito_user_pool.main.id
    }
  }
}
```

**IAM policy for `http-admin`:**
```hcl
# DynamoDB
dynamodb:Scan, GetItem, PutItem, UpdateItem on users table

# Cognito
cognito-idp:AdminAddUserToGroup
cognito-idp:AdminRemoveUserFromGroup
cognito-idp:AdminListGroupsForUser
cognito-idp:AdminGetUser
cognito-idp:ListUsersInGroup
```

**API Gateway routes (HTTP API):**
```hcl
# All under /api/admin/* — JWT authorizer (Cognito) required
# Lambda-level admin group check is the second enforcement layer

GET  /api/admin/users           → http-admin
PUT  /api/admin/users/{userId}  → http-admin
GET  /api/admin/stats           → http-admin
```

---

### `http-invites`

Invite creation and redemption API.

**Lambda config:**
```hcl
resource "aws_lambda_function" "http_invites" {
  function_name = "${var.prefix}-http-invites"
  handler       = "bootstrap"
  runtime       = "provided.al2023"
  architectures = ["arm64"]
  timeout       = 10
  memory_size   = 128

  environment {
    variables = {
      INVITES_TABLE      = aws_dynamodb_table.invites.name
      SESSIONS_TABLE     = aws_dynamodb_table.sessions.name
      MEMBERSHIPS_TABLE  = aws_dynamodb_table.memberships.name
    }
  }
}
```

**IAM policy for `http-invites`:**
```hcl
# DynamoDB
dynamodb:GetItem, PutItem, UpdateItem on invites table
dynamodb:GetItem, UpdateItem on sessions table
dynamodb:PutItem on memberships table
```

**API Gateway routes:**
```hcl
POST /api/invites              → http-invites  (auth required — creates invite for owned session)
GET  /api/invites/{code}       → http-invites  (no auth — resolve code for join landing page)
POST /api/invites/{code}/join  → http-invites  (auth required — redeem code, join session)
```

---

## Modified Lambda Environment Variables

### `http-games`
```hcl
# Add:
USERS_TABLE       = aws_dynamodb_table.users.name
MEMBERSHIPS_TABLE = aws_dynamodb_table.memberships.name
```
Additional IAM:
```hcl
dynamodb:GetItem on users table
dynamodb:PutItem, GetItem, Query on memberships table
```

### `ws-chat`
```hcl
# Add:
USERS_TABLE = aws_dynamodb_table.users.name
```
Additional IAM:
```hcl
dynamodb:GetItem, UpdateItem on users table
```

### `world-gen`
```hcl
# Add:
USERS_TABLE = aws_dynamodb_table.users.name
```
Additional IAM:
```hcl
dynamodb:UpdateItem on users table   # for UpdateUserTokens after blueprint generation
```

### `ws-connect`
No new env vars — but IAM change: add `dynamodb:GetItem` on sessions table (needed to verify caller is owner/member before writing connection). Currently `ws-connect` has `SESSIONS_TABLE` env var but no IAM permission for it — this was a known gap.

---

## `{prefix}-sessions` Table — New Attributes

No schema migration in DynamoDB (schemaless). New attributes are written on new records only. `FromSaveState` in Go handles absent fields gracefully via `omitempty`.

New attributes added to session items:
```
owner_id        B    (replaces semantic role of user_id; user_id GSI preserved unchanged)
members         L    list of {user_id: B, character_id: S}
invite_code     S    current active invite code (denormalized for fast lookup)
party_size      N    max party size (default: 4)
schema_version  N    bumped to 2 when party fields present
```

The existing `user_id` attribute and `user-sessions-index` GSI are **preserved unchanged** — they remain the mechanism for listing owned games. The memberships table handles joined-but-not-owned game lookups.

---

## Deployment Order

1. `terraform apply` — creates all new tables, Cognito groups, new Lambdas, new API routes
2. Deploy `cognito-post-confirm` Lambda code (CI/CD builds + updates)
3. Deploy `http-admin` Lambda code
4. Deploy `http-invites` Lambda code
5. Deploy updated `http-games`, `ws-chat`, `world-gen`, `ws-connect` Lambda code (Phase 0 application changes)
6. Run `scripts/elevate-admin.sh <your-cognito-sub>` to bootstrap the first admin account

Existing games and user sessions are unaffected — no data migration needed at the Terraform/infrastructure level. The Go `SchemaVersion` guard handles old records at the application level.
