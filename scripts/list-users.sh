#!/usr/bin/env bash
set -euo pipefail

# -----------------------------------------------------------------------
# list-users.sh — List all users and their current status
#
# Usage: ./scripts/list-users.sh [aws-profile]
# -----------------------------------------------------------------------

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
      ("tokens: " + (.tokens_used.N // "0") + "/" + (if .token_limit.N == "0" then "unlimited" else .token_limit.N end))
    ] | @tsv' \
  | column -t
