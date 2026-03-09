variable "prefix" { type = string }
variable "common_tags" { type = map(string) }
variable "sessions_table_name" { type = string }
variable "connections_table_name" { type = string }
variable "mutations_table_name" { type = string }
variable "users_table_name" { type = string }
variable "memberships_table_name" { type = string }
variable "invites_table_name" { type = string }
variable "sessions_table_arn" { type = string }
variable "connections_table_arn" { type = string }
variable "connections_table_index_arn" { type = string }
variable "mutations_table_arn" { type = string }
variable "users_table_arn" { type = string }
variable "memberships_table_arn" { type = string }
variable "memberships_table_index_arn" { type = string }
variable "invites_table_arn" { type = string }
variable "user_pool_id" { type = string }
variable "user_pool_arn" { type = string }
variable "websocket_api_execution_arn" { type = string }
variable "websocket_api_endpoint" { type = string }

# ── Shared bootstrap placeholder ────────────────────────────────────────────
# CI replaces function code after first deploy. We use a minimal bootstrap
# so Terraform can create the resources without a real artifact.
data "archive_file" "placeholder" {
  type        = "zip"
  output_path = "${path.module}/placeholder.zip"
  source {
    content  = "placeholder"
    filename = "bootstrap"
  }
}

# ── Shared base policy ───────────────────────────────────────────────────────
data "aws_iam_policy_document" "lambda_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "lambda_logs" {
  name = "${var.prefix}-lambda-logs"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
      Resource = "arn:aws:logs:*:*:*"
    }]
  })
  tags = var.common_tags
}

# ── Helper locals ────────────────────────────────────────────────────────────
# ws_endpoint_full is the value passed to WEBSOCKET_API_ENDPOINT on all Lambdas
# that push frames to connected clients. The api-gateway module output already
# includes the stage (e.g. "ba2t50m7se.execute-api.us-west-2.amazonaws.com/prod"),
# so we use it directly. wsutil.New() prepends "https://" at runtime.
locals {
  ws_endpoint_full = var.websocket_api_endpoint
}

# ── ws-connect ───────────────────────────────────────────────────────────────
resource "aws_iam_role" "ws_connect" {
  name               = "${var.prefix}-ws-connect"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "ws_connect_logs" {
  role       = aws_iam_role.ws_connect.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "ws_connect" {
  name = "connections"
  role = aws_iam_role.ws_connect.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem", "dynamodb:GetItem", "dynamodb:DeleteItem", "dynamodb:Query"]
        Resource = [var.connections_table_arn, var.connections_table_index_arn]
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem"]
        Resource = var.sessions_table_arn
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "ws_connect" {
  name              = "/aws/lambda/${var.prefix}-ws-connect"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "ws_connect" {
  function_name    = "${var.prefix}-ws-connect"
  role             = aws_iam_role.ws_connect.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128
  environment {
    variables = {
      SESSIONS_TABLE    = var.sessions_table_name
      CONNECTIONS_TABLE = var.connections_table_name
      USER_POOL_ID      = var.user_pool_id
    }
  }
  depends_on = [aws_cloudwatch_log_group.ws_connect]
  tags       = var.common_tags
}

# ── ws-disconnect ────────────────────────────────────────────────────────────
resource "aws_iam_role" "ws_disconnect" {
  name               = "${var.prefix}-ws-disconnect"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "ws_disconnect_logs" {
  role       = aws_iam_role.ws_disconnect.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "ws_disconnect" {
  name = "connections"
  role = aws_iam_role.ws_disconnect.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["dynamodb:DeleteItem", "dynamodb:GetItem"]
      Resource = var.connections_table_arn
    }]
  })
}
resource "aws_cloudwatch_log_group" "ws_disconnect" {
  name              = "/aws/lambda/${var.prefix}-ws-disconnect"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "ws_disconnect" {
  function_name    = "${var.prefix}-ws-disconnect"
  role             = aws_iam_role.ws_disconnect.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128
  environment {
    variables = { CONNECTIONS_TABLE = var.connections_table_name }
  }
  depends_on = [aws_cloudwatch_log_group.ws_disconnect]
  tags       = var.common_tags
}

# ── ws-chat ──────────────────────────────────────────────────────────────────
resource "aws_iam_role" "ws_chat" {
  name               = "${var.prefix}-ws-chat"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "ws_chat_logs" {
  role       = aws_iam_role.ws_chat.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "ws_chat" {
  name = "chat-permissions"
  role = aws_iam_role.ws_chat.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = [var.sessions_table_arn, var.connections_table_arn]
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:Query"]
        Resource = var.connections_table_index_arn
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem"]
        Resource = var.mutations_table_arn
      },
      {
        # UpdateUserTokens — increment token counter after each narration
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:UpdateItem"]
        Resource = var.users_table_arn
      },
      {
        Effect   = "Allow"
        Action   = ["bedrock:InvokeModelWithResponseStream", "bedrock:InvokeModel"]
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = ["execute-api:ManageConnections"]
        Resource = "${var.websocket_api_execution_arn}/*/*/@connections/*"
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "ws_chat" {
  name              = "/aws/lambda/${var.prefix}-ws-chat"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "ws_chat" {
  function_name    = "${var.prefix}-ws-chat"
  role             = aws_iam_role.ws_chat.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 29
  memory_size      = 256
  environment {
    variables = {
      SESSIONS_TABLE         = var.sessions_table_name
      CONNECTIONS_TABLE      = var.connections_table_name
      MUTATIONS_TABLE        = var.mutations_table_name
      USERS_TABLE            = var.users_table_name
      WEBSOCKET_API_ENDPOINT = local.ws_endpoint_full
      BEDROCK_REGION         = "us-west-2"
    }
  }
  depends_on = [aws_cloudwatch_log_group.ws_chat]
  tags       = var.common_tags
}

# ── ws-game-action ───────────────────────────────────────────────────────────
resource "aws_iam_role" "ws_game_action" {
  name               = "${var.prefix}-ws-game-action"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "ws_game_action_logs" {
  role       = aws_iam_role.ws_game_action.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "ws_game_action" {
  name = "game-action-permissions"
  role = aws_iam_role.ws_game_action.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = [var.sessions_table_arn, var.connections_table_arn]
      },
      {
        Effect   = "Allow"
        Action   = ["dynamodb:Query"]
        Resource = var.connections_table_index_arn
      },
      {
        Effect   = "Allow"
        Action   = ["execute-api:ManageConnections"]
        Resource = "${var.websocket_api_execution_arn}/*/*/@connections/*"
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "ws_game_action" {
  name              = "/aws/lambda/${var.prefix}-ws-game-action"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "ws_game_action" {
  function_name    = "${var.prefix}-ws-game-action"
  role             = aws_iam_role.ws_game_action.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128
  environment {
    variables = {
      SESSIONS_TABLE         = var.sessions_table_name
      CONNECTIONS_TABLE      = var.connections_table_name
      WEBSOCKET_API_ENDPOINT = local.ws_endpoint_full
    }
  }
  depends_on = [aws_cloudwatch_log_group.ws_game_action]
  tags       = var.common_tags
}

# ── http-games ───────────────────────────────────────────────────────────────
resource "aws_iam_role" "http_games" {
  name               = "${var.prefix}-http-games"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "http_games_logs" {
  role       = aws_iam_role.http_games.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "http_games" {
  name = "games-permissions"
  role = aws_iam_role.http_games.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # Sessions: full CRUD + GSI query (ListGamesByOwner) + BatchGet (party sessions)
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:DeleteItem",
          "dynamodb:Query",
          "dynamodb:BatchGetItem",
        ]
        Resource = [var.sessions_table_arn, "${var.sessions_table_arn}/index/*"]
      },
      {
        # Memberships: read party membership lists, write owner record, delete on game delete
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:Query",
          "dynamodb:DeleteItem",
        ]
        Resource = [var.memberships_table_arn, var.memberships_table_index_arn]
      },
      {
        # Users: read quota + role on create, check games limit
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem"]
        Resource = var.users_table_arn
      },
      {
        Effect   = "Allow"
        Action   = ["lambda:InvokeFunction"]
        Resource = aws_lambda_function.world_gen.arn
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "http_games" {
  name              = "/aws/lambda/${var.prefix}-http-games"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "http_games" {
  function_name    = "${var.prefix}-http-games"
  role             = aws_iam_role.http_games.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 15
  memory_size      = 128
  environment {
    variables = {
      SESSIONS_TABLE    = var.sessions_table_name
      MEMBERSHIPS_TABLE = var.memberships_table_name
      USERS_TABLE       = var.users_table_name
      WORLD_GEN_ARN     = aws_lambda_function.world_gen.arn
    }
  }
  depends_on = [aws_cloudwatch_log_group.http_games]
  tags       = var.common_tags
}

# ── http-users ───────────────────────────────────────────────────────────────
resource "aws_iam_role" "http_users" {
  name               = "${var.prefix}-http-users"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "http_users_logs" {
  role       = aws_iam_role.http_users.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "http_users" {
  name = "cognito-user-mgmt"
  role = aws_iam_role.http_users.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["cognito-idp:AdminUpdateUserAttributes", "cognito-idp:AdminGetUser"]
      Resource = var.user_pool_arn
    }]
  })
}
resource "aws_cloudwatch_log_group" "http_users" {
  name              = "/aws/lambda/${var.prefix}-http-users"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "http_users" {
  function_name    = "${var.prefix}-http-users"
  role             = aws_iam_role.http_users.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128
  environment {
    variables = { USER_POOL_ID = var.user_pool_id }
  }
  depends_on = [aws_cloudwatch_log_group.http_users]
  tags       = var.common_tags
}

# ── http-admin ───────────────────────────────────────────────────────────────
resource "aws_iam_role" "http_admin" {
  name               = "${var.prefix}-http-admin"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "http_admin_logs" {
  role       = aws_iam_role.http_admin.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "http_admin" {
  name = "admin-permissions"
  role = aws_iam_role.http_admin.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # Users: list all (Scan), get, update role/limits, update token counter
        Effect = "Allow"
        Action = [
          "dynamodb:Scan",
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
        ]
        Resource = var.users_table_arn
      },
      {
        # Cognito: read email, manage group membership for role sync
        Effect = "Allow"
        Action = [
          "cognito-idp:AdminGetUser",
          "cognito-idp:AdminAddUserToGroup",
          "cognito-idp:AdminRemoveUserFromGroup",
        ]
        Resource = var.user_pool_arn
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "http_admin" {
  name              = "/aws/lambda/${var.prefix}-http-admin"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "http_admin" {
  function_name    = "${var.prefix}-http-admin"
  role             = aws_iam_role.http_admin.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 15
  memory_size      = 128
  environment {
    variables = {
      USERS_TABLE  = var.users_table_name
      USER_POOL_ID = var.user_pool_id
    }
  }
  depends_on = [aws_cloudwatch_log_group.http_admin]
  tags       = var.common_tags
}

# ── http-invites ─────────────────────────────────────────────────────────────
resource "aws_iam_role" "http_invites" {
  name               = "${var.prefix}-http-invites"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "http_invites_logs" {
  role       = aws_iam_role.http_invites.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "http_invites" {
  name = "invites-permissions"
  role = aws_iam_role.http_invites.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # Sessions: read game, update party size on join
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem"]
        Resource = var.sessions_table_arn
      },
      {
        # Invites: create, read, increment use count
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
        ]
        Resource = var.invites_table_arn
      },
      {
        # Memberships: write member record on join
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem", "dynamodb:Query"]
        Resource = [var.memberships_table_arn, var.memberships_table_index_arn]
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "http_invites" {
  name              = "/aws/lambda/${var.prefix}-http-invites"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "http_invites" {
  function_name    = "${var.prefix}-http-invites"
  role             = aws_iam_role.http_invites.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 15
  memory_size      = 128
  environment {
    variables = {
      SESSIONS_TABLE    = var.sessions_table_name
      INVITES_TABLE     = var.invites_table_name
      MEMBERSHIPS_TABLE = var.memberships_table_name
    }
  }
  depends_on = [aws_cloudwatch_log_group.http_invites]
  tags       = var.common_tags
}

# ── cognito-post-confirm ─────────────────────────────────────────────────────
resource "aws_iam_role" "cognito_post_confirm" {
  name               = "${var.prefix}-cognito-post-confirm"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "cognito_post_confirm_logs" {
  role       = aws_iam_role.cognito_post_confirm.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "cognito_post_confirm" {
  name = "post-confirm-permissions"
  role = aws_iam_role.cognito_post_confirm.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # Users: create restricted user record on sign-up
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem"]
        Resource = var.users_table_arn
      },
      {
        # Invites: read invite code, increment use count
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:UpdateItem"]
        Resource = var.invites_table_arn
      },
      {
        # Memberships: write member record if user signed up via invite
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem"]
        Resource = var.memberships_table_arn
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "cognito_post_confirm" {
  name              = "/aws/lambda/${var.prefix}-cognito-post-confirm"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "cognito_post_confirm" {
  function_name    = "${var.prefix}-cognito-post-confirm"
  role             = aws_iam_role.cognito_post_confirm.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128
  environment {
    variables = {
      USERS_TABLE       = var.users_table_name
      INVITES_TABLE     = var.invites_table_name
      MEMBERSHIPS_TABLE = var.memberships_table_name
    }
  }
  depends_on = [aws_cloudwatch_log_group.cognito_post_confirm]
  tags       = var.common_tags
}

# ── world-gen ────────────────────────────────────────────────────────────────
resource "aws_iam_role" "world_gen" {
  name               = "${var.prefix}-world-gen"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "world_gen_logs" {
  role       = aws_iam_role.world_gen.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "world_gen" {
  name = "world-gen-permissions"
  role = aws_iam_role.world_gen.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
        ]
        Resource = var.sessions_table_arn
      },
      {
        # Read connections table to find the user's active WebSocket connection
        Effect   = "Allow"
        Action   = ["dynamodb:Query"]
        Resource = var.connections_table_index_arn
      },
      {
        # UpdateUserTokens — increment token counter after world generation
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:UpdateItem"]
        Resource = var.users_table_arn
      },
      {
        Effect   = "Allow"
        Action   = ["bedrock:InvokeModel", "bedrock:InvokeModelWithResponseStream"]
        Resource = "*"
      },
      {
        # Push progress frames to the connected client via API Gateway Management API
        Effect   = "Allow"
        Action   = ["execute-api:ManageConnections"]
        Resource = "${var.websocket_api_execution_arn}/*"
      }
    ]
  })
}
resource "aws_cloudwatch_log_group" "world_gen" {
  name              = "/aws/lambda/${var.prefix}-world-gen"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "world_gen" {
  function_name    = "${var.prefix}-world-gen"
  role             = aws_iam_role.world_gen.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 900
  memory_size      = 256
  environment {
    variables = {
      SESSIONS_TABLE         = var.sessions_table_name
      CONNECTIONS_TABLE      = var.connections_table_name
      USERS_TABLE            = var.users_table_name
      WEBSOCKET_API_ENDPOINT = local.ws_endpoint_full
      BEDROCK_REGION         = "us-west-2"
    }
  }
  depends_on = [aws_cloudwatch_log_group.world_gen]
  tags       = var.common_tags
}

# ── Outputs ──────────────────────────────────────────────────────────────────
output "http_games_invoke_arn" { value = aws_lambda_function.http_games.invoke_arn }
output "http_users_invoke_arn" { value = aws_lambda_function.http_users.invoke_arn }
output "http_admin_invoke_arn" { value = aws_lambda_function.http_admin.invoke_arn }
output "http_invites_invoke_arn" { value = aws_lambda_function.http_invites.invoke_arn }
output "ws_connect_invoke_arn" { value = aws_lambda_function.ws_connect.invoke_arn }
output "ws_disconnect_invoke_arn" { value = aws_lambda_function.ws_disconnect.invoke_arn }
output "ws_chat_invoke_arn" { value = aws_lambda_function.ws_chat.invoke_arn }
output "ws_game_action_invoke_arn" { value = aws_lambda_function.ws_game_action.invoke_arn }
output "world_gen_invoke_arn" { value = aws_lambda_function.world_gen.invoke_arn }
output "http_games_function_name" { value = aws_lambda_function.http_games.function_name }
output "http_users_function_name" { value = aws_lambda_function.http_users.function_name }
output "http_admin_function_name" { value = aws_lambda_function.http_admin.function_name }
output "http_invites_function_name" { value = aws_lambda_function.http_invites.function_name }
output "ws_connect_function_name" { value = aws_lambda_function.ws_connect.function_name }
output "ws_disconnect_function_name" { value = aws_lambda_function.ws_disconnect.function_name }
output "ws_chat_function_name" { value = aws_lambda_function.ws_chat.function_name }
output "ws_game_action_function_name" { value = aws_lambda_function.ws_game_action.function_name }
output "world_gen_function_name" { value = aws_lambda_function.world_gen.function_name }
output "cognito_post_confirm_function_arn" { value = aws_lambda_function.cognito_post_confirm.arn }
