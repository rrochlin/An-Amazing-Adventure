variable "prefix" { type = string }
variable "common_tags" { type = map(string) }
variable "sessions_table_name" { type = string }
variable "connections_table_name" { type = string }
variable "sessions_table_arn" { type = string }
variable "connections_table_arn" { type = string }
variable "connections_table_index_arn" { type = string }
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
    Statement = [{
      Effect   = "Allow"
      Action   = ["dynamodb:PutItem", "dynamodb:DeleteItem", "dynamodb:Query"]
      Resource = [var.connections_table_arn, var.connections_table_index_arn]
    }]
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
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:DeleteItem", "dynamodb:Query"]
        Resource = [var.sessions_table_arn, "${var.sessions_table_arn}/index/*"]
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
      SESSIONS_TABLE = var.sessions_table_name
      WORLD_GEN_ARN  = aws_lambda_function.world_gen.arn
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
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = var.sessions_table_arn
      },
      {
        # Read connections table to find the user's active WebSocket connection
        Effect   = "Allow"
        Action   = ["dynamodb:Query"]
        Resource = var.connections_table_index_arn
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
output "ws_connect_invoke_arn" { value = aws_lambda_function.ws_connect.invoke_arn }
output "ws_disconnect_invoke_arn" { value = aws_lambda_function.ws_disconnect.invoke_arn }
output "ws_chat_invoke_arn" { value = aws_lambda_function.ws_chat.invoke_arn }
output "ws_game_action_invoke_arn" { value = aws_lambda_function.ws_game_action.invoke_arn }
output "http_games_function_name" { value = aws_lambda_function.http_games.function_name }
output "http_users_function_name" { value = aws_lambda_function.http_users.function_name }
output "ws_connect_function_name" { value = aws_lambda_function.ws_connect.function_name }
output "ws_disconnect_function_name" { value = aws_lambda_function.ws_disconnect.function_name }
output "ws_chat_function_name" { value = aws_lambda_function.ws_chat.function_name }
output "ws_game_action_function_name" { value = aws_lambda_function.ws_game_action.function_name }
