# Admin Scripts

Operational scripts for managing the game in production. These live in `scripts/` in the monorepo root and are run manually from a developer workstation with AWS credentials configured.

---

## `scripts/elevate-admin.sh`

Promotes a Cognito user to admin. Run this once after creating your first account to bootstrap admin access.

**Usage:**
```bash
./scripts/elevate-admin.sh <cognito-user-sub> [aws-profile]

# Examples:
./scripts/elevate-admin.sh abc123-def456-...
./scripts/elevate-admin.sh abc123-def456-... personal
AWS_PROFILE=personal ./scripts/elevate-admin.sh abc123-def456-...
```

**What it does:**
1. Resolves `USER_POOL_ID` from SSM (or env var override)
2. Adds user to Cognito `admin` group
3. Adds user to Cognito `user` group
4. Upserts the DynamoDB user record: `role=admin`, `ai_enabled=true`, `token_limit=0`

**Full script** (`scripts/elevate-admin.sh`):
```bash
#!/usr/bin/env bash
set -euo pipefail

# -----------------------------------------------------------------------
# elevate-admin.sh — Promote a Cognito user to admin for An Amazing Adventure
#
# Usage: ./scripts/elevate-admin.sh <cognito-user-sub> [aws-profile]
#
# The Cognito user sub is the UUID in the "sub" claim of the user's JWT.
# Find it at: AWS Console → Cognito → User Pools → Users → click user → sub attribute
# Or: decode the access token at jwt.io and look at the "sub" field.
# -----------------------------------------------------------------------

USER_SUB="${1:?Error: Cognito user sub is required as first argument}"
AWS_PROFILE="${2:-${AWS_PROFILE:-default}}"
REGION="${AWS_REGION:-us-west-2}"
ENV="${ENVIRONMENT:-prod}"
PREFIX="amazing-adventure-${ENV}"
USERS_TABLE="${PREFIX}-users"

export AWS_PROFILE
export AWS_REGION="${REGION}"

echo "==> Resolving Cognito User Pool ID..."
POOL_ID="${COGNITO_USER_POOL_ID:-}"
if [ -z "${POOL_ID}" ]; then
  POOL_ID=$(aws ssm get-parameter \
    --name "/amazing-adventure/${ENV}/cognito/user-pool-id" \
    --query "Parameter.Value" \
    --output text 2>/dev/null || true)
fi
if [ -z "${POOL_ID}" ]; then
  echo "Error: Could not resolve USER_POOL_ID."
  echo "Set COGNITO_USER_POOL_ID env var or ensure SSM parameter exists."
  exit 1
fi
echo "    Pool ID: ${POOL_ID}"
echo "    User sub: ${USER_SUB}"
echo ""

echo "==> Adding user to Cognito 'admin' group..."
aws cognito-idp admin-add-user-to-group \
  --user-pool-id "${POOL_ID}" \
  --username "${USER_SUB}" \
  --group-name "admin"
echo "    Done."

echo "==> Adding user to Cognito 'user' group..."
aws cognito-idp admin-add-user-to-group \
  --user-pool-id "${POOL_ID}" \
  --username "${USER_SUB}" \
  --group-name "user"
echo "    Done."

echo "==> Upserting DynamoDB user record (role=admin, ai_enabled=true, token_limit=0)..."
NOW_MS=$(date +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")

# BinaryID: the user_id is stored as a DynamoDB Binary type.
# The Cognito sub UUID is stored as raw bytes (the string encoded as UTF-8 bytes).
# aws cli requires Binary values as base64-encoded strings.
USER_SUB_B64=$(echo -n "${USER_SUB}" | base64)

aws dynamodb put-item \
  --table-name "${USERS_TABLE}" \
  --item "{
    \"user_id\":      {\"B\": \"${USER_SUB_B64}\"},
    \"role\":         {\"S\": \"admin\"},
    \"ai_enabled\":   {\"BOOL\": true},
    \"token_limit\":  {\"N\": \"0\"},
    \"tokens_used\":  {\"N\": \"0\"},
    \"games_limit\":  {\"N\": \"0\"},
    \"billing_mode\": {\"S\": \"admin_granted\"},
    \"created_at\":   {\"N\": \"${NOW_MS}\"},
    \"updated_at\":   {\"N\": \"${NOW_MS}\"},
    \"notes\":        {\"S\": \"Bootstrapped by elevate-admin.sh\"}
  }"
echo "    Done."

echo ""
echo "==> Verification:"
echo "    Cognito groups for user ${USER_SUB}:"
aws cognito-idp admin-list-groups-for-user \
  --user-pool-id "${POOL_ID}" \
  --username "${USER_SUB}" \
  --query "Groups[].GroupName" \
  --output table

echo ""
echo "SUCCESS: User ${USER_SUB} is now an admin."
echo "Log in and navigate to /admin to access the admin panel."
echo ""
echo "Note: You must log out and back in for the new Cognito group claims"
echo "to appear in your JWT tokens."
```

Make it executable after creation:
```bash
chmod +x scripts/elevate-admin.sh
```

---

## How to find your Cognito user sub

**Option A — From the AWS Console:**
1. AWS Console → Cognito → User Pools → `amazing-adventure-prod` → Users
2. Click your user → copy the `sub` attribute value

**Option B — Decode your JWT:**
1. Log in to the game
2. Open browser DevTools → Application → Local Storage → `AAA_TOKENS`
3. Copy the `accessToken` value
4. Go to https://jwt.io and paste it
5. The `sub` field in the payload is your Cognito user sub

**Option C — AWS CLI:**
```bash
aws cognito-idp list-users \
  --user-pool-id <POOL_ID> \
  --filter "email = \"your@email.com\"" \
  --query "Users[0].Username" \
  --output text
```
The `Username` returned by Cognito for email-based pools IS the `sub` UUID.

---

## `scripts/list-users.sh`

Quick operational script to list all users and their current status without opening the admin panel.

```bash
#!/usr/bin/env bash
set -euo pipefail

AWS_PROFILE="${1:-${AWS_PROFILE:-default}}"
REGION="${AWS_REGION:-us-west-2}"
ENV="${ENVIRONMENT:-prod}"
PREFIX="amazing-adventure-${ENV}"
USERS_TABLE="${PREFIX}-users"

export AWS_PROFILE
export AWS_REGION="${REGION}"

echo "Users in ${USERS_TABLE}:"
echo ""

aws dynamodb scan \
  --table-name "${USERS_TABLE}" \
  --projection-expression "user_id, #r, ai_enabled, token_limit, tokens_used, billing_mode" \
  --expression-attribute-names '{"#r": "role"}' \
  --output json \
  | jq -r '.Items[] | [
      (.user_id.B | @base64d),
      .role.S,
      (if .ai_enabled.BOOL then "AI:YES" else "AI:NO" end),
      ("tokens: " + (.tokens_used.N // "0") + "/" + (if .token_limit.N == "0" then "∞" else .token_limit.N end))
    ] | @tsv' \
  | column -t
```

---

## `scripts/reset-user-tokens.sh`

Emergency script to reset a user's token counter (e.g. if a bug caused incorrect accounting).

```bash
#!/usr/bin/env bash
# Usage: ./scripts/reset-user-tokens.sh <cognito-user-sub> [new-count]
# new-count defaults to 0

USER_SUB="${1:?Error: user sub required}"
NEW_COUNT="${2:-0}"
ENV="${ENVIRONMENT:-prod}"
USERS_TABLE="amazing-adventure-${ENV}-users"
USER_SUB_B64=$(echo -n "${USER_SUB}" | base64)
NOW_MS=$(date +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")

aws dynamodb update-item \
  --table-name "${USERS_TABLE}" \
  --key "{\"user_id\": {\"B\": \"${USER_SUB_B64}\"}}" \
  --update-expression "SET tokens_used = :n, updated_at = :ts" \
  --expression-attribute-values "{\":n\": {\"N\": \"${NEW_COUNT}\"}, \":ts\": {\"N\": \"${NOW_MS}\"}}"

echo "Reset tokens_used to ${NEW_COUNT} for user ${USER_SUB}"
```

---

## Notes on BinaryID encoding

The `user_id` in DynamoDB is stored as Binary type (`B`), matching the `BinaryID` type in `server-v2/internal/db/binaryid.go`. When using the AWS CLI to write or query these records, the binary value must be provided as a base64-encoded string.

The Cognito `sub` UUID is stored as the raw UTF-8 bytes of the string (not parsed as a UUID binary). So:
```bash
# Correct encoding:
USER_SUB_B64=$(echo -n "abc123-def456-..." | base64)

# This is what the Go BinaryID type does:
# type BinaryID string
# func (id BinaryID) MarshalDynamoDBAttributeValue() (types.AttributeValue, error) {
#     return &types.AttributeValueMemberB{Value: []byte(id)}, nil
# }
```

This is the same pattern as all existing scripts and admin operations that interact with the sessions or connections tables.
