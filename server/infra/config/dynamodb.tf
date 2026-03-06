# ---------------------------------------------------------------------------
# Game sessions table (replaces amazing-adventure-data)
# Stores full serialised game state per session.
# ---------------------------------------------------------------------------
resource "aws_dynamodb_table" "sessions" {
  name         = "${local.prefix}-sessions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "session_id"

  attribute {
    name = "session_id"
    type = "B" # binary UUID
  }

  attribute {
    name = "user_id"
    type = "B" # binary UUID
  }

  global_secondary_index {
    name            = "user-sessions-index"
    hash_key        = "user_id"
    projection_type = "ALL"
  }

  tags = merge(local.common_tags, { Name = "GameSessions" })
}

# ---------------------------------------------------------------------------
# WebSocket connections table
# Tracks active API GW WebSocket connections so Lambdas can push back.
# TTL auto-expires stale records after 24 h.
# ---------------------------------------------------------------------------
resource "aws_dynamodb_table" "connections" {
  name         = "${local.prefix}-connections"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "connection_id"

  attribute {
    name = "connection_id"
    type = "S"
  }

  attribute {
    name = "user_id"
    type = "B" # binary UUID — lets us find/clean up prior connections per user
  }

  global_secondary_index {
    name            = "user-connections-index"
    hash_key        = "user_id"
    projection_type = "ALL"
  }

  ttl {
    attribute_name = "expires_at"
    enabled        = true
  }

  tags = merge(local.common_tags, { Name = "WsConnections" })
}
