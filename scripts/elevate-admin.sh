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
# Hard-coded fallback for prod (avoids needing SSM parameter)
if [ -z "${POOL_ID}" ]; then
  POOL_ID="us-west-2_DDcAHl2E6"
  echo "    (SSM not found — using hardcoded prod pool ID)"
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

# BinaryID: user_id is stored as DynamoDB Binary type (UTF-8 bytes of the sub string, base64 for CLI).
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
