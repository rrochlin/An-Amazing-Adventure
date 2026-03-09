#!/usr/bin/env bash
set -euo pipefail

# -----------------------------------------------------------------------
# reset-user-tokens.sh — Reset a user's token counter
#
# Usage: ./scripts/reset-user-tokens.sh <cognito-user-sub> [new-count] [aws-profile]
# new-count defaults to 0
# -----------------------------------------------------------------------

USER_SUB="${1:?Error: user sub required}"
NEW_COUNT="${2:-0}"
AWS_PROFILE="${3:-${AWS_PROFILE:-default}}"
REGION="${AWS_REGION:-us-west-2}"
ENV="${ENVIRONMENT:-prod}"
USERS_TABLE="amazing-adventure-${ENV}-users"

export AWS_PROFILE
export AWS_REGION="${REGION}"

USER_SUB_B64=$(echo -n "${USER_SUB}" | base64)
NOW_MS=$(date +%s%3N 2>/dev/null || python3 -c "import time; print(int(time.time()*1000))")

aws dynamodb update-item \
  --table-name "${USERS_TABLE}" \
  --key "{\"user_id\": {\"B\": \"${USER_SUB_B64}\"}}" \
  --update-expression "SET tokens_used = :n, updated_at = :ts" \
  --expression-attribute-values "{\":n\": {\"N\": \"${NEW_COUNT}\"}, \":ts\": {\"N\": \"${NOW_MS}\"}}"

echo "Reset tokens_used to ${NEW_COUNT} for user ${USER_SUB}"
