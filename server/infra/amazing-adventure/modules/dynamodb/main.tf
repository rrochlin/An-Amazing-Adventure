variable "prefix" { type = string }
variable "common_tags" { type = map(string) }

resource "aws_dynamodb_table" "sessions" {
  name         = "${var.prefix}-sessions"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "session_id"

  attribute {
    name = "session_id"
    type = "B"
  }
  attribute {
    name = "user_id"
    type = "B"
  }

  global_secondary_index {
    name            = "user-sessions-index"
    hash_key        = "user_id"
    projection_type = "ALL"
  }

  tags = merge(var.common_tags, { Name = "GameSessions" })
}

resource "aws_dynamodb_table" "connections" {
  name         = "${var.prefix}-connections"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "connection_id"

  attribute {
    name = "connection_id"
    type = "S"
  }
  attribute {
    name = "user_id"
    type = "B"
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

  tags = merge(var.common_tags, { Name = "WsConnections" })
}

output "sessions_table_name" { value = aws_dynamodb_table.sessions.name }
output "sessions_table_arn" { value = aws_dynamodb_table.sessions.arn }
output "connections_table_name" { value = aws_dynamodb_table.connections.name }
output "connections_table_arn" { value = aws_dynamodb_table.connections.arn }
output "connections_table_index_arn" { value = "${aws_dynamodb_table.connections.arn}/index/*" }
