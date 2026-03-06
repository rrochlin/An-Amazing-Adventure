# ---------------------------------------------------------------------------
# Lambda placeholder zip
# The CI pipeline will replace function code after first deploy.
# We create a minimal bootstrap binary placeholder so Terraform can create
# the function resource without a real artifact.
# ---------------------------------------------------------------------------
data "archive_file" "placeholder" {
  type        = "zip"
  output_path = "${path.module}/placeholder.zip"

  source {
    content  = "placeholder"
    filename = "bootstrap"
  }
}

# ---------------------------------------------------------------------------
# Shared Lambda execution role base (each function gets its own role below)
# ---------------------------------------------------------------------------
data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

# Common policy: basic Lambda execution + CloudWatch logs
resource "aws_iam_policy" "lambda_basic" {
  name        = "${local.prefix}-lambda-basic"
  description = "Basic Lambda execution permissions (CloudWatch logs)"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:*:*:*"
      }
    ]
  })

  tags = local.common_tags
}

# ---------------------------------------------------------------------------
# Helper: create a Lambda function + role + CloudWatch log group
# ---------------------------------------------------------------------------

# ws-connect
resource "aws_iam_role" "ws_connect" {
  name               = "${local.prefix}-ws-connect"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "ws_connect_basic" {
  role       = aws_iam_role.ws_connect.name
  policy_arn = aws_iam_policy.lambda_basic.arn
}

resource "aws_iam_role_policy" "ws_connect_inline" {
  name = "dynamo-connections"
  role = aws_iam_role.ws_connect.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem", "dynamodb:DeleteItem", "dynamodb:Query"]
        Resource = [aws_dynamodb_table.connections.arn, "${aws_dynamodb_table.connections.arn}/index/*"]
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "ws_connect" {
  name              = "/aws/lambda/${local.prefix}-ws-connect"
  retention_in_days = 7
  tags              = local.common_tags
}

resource "aws_lambda_function" "ws_connect" {
  function_name    = "${local.prefix}-ws-connect"
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
      CONNECTIONS_TABLE = aws_dynamodb_table.connections.name
      USER_POOL_ID      = aws_cognito_user_pool.main.id
      ENVIRONMENT       = var.environment
    }
  }

  depends_on = [aws_cloudwatch_log_group.ws_connect]
  tags       = local.common_tags
}

# ---------------------------------------------------------------------------
# ws-disconnect
# ---------------------------------------------------------------------------
resource "aws_iam_role" "ws_disconnect" {
  name               = "${local.prefix}-ws-disconnect"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "ws_disconnect_basic" {
  role       = aws_iam_role.ws_disconnect.name
  policy_arn = aws_iam_policy.lambda_basic.arn
}

resource "aws_iam_role_policy" "ws_disconnect_inline" {
  name = "dynamo-connections"
  role = aws_iam_role.ws_disconnect.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:DeleteItem", "dynamodb:GetItem"]
        Resource = aws_dynamodb_table.connections.arn
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "ws_disconnect" {
  name              = "/aws/lambda/${local.prefix}-ws-disconnect"
  retention_in_days = 7
  tags              = local.common_tags
}

resource "aws_lambda_function" "ws_disconnect" {
  function_name    = "${local.prefix}-ws-disconnect"
  role             = aws_iam_role.ws_disconnect.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128

  environment {
    variables = {
      CONNECTIONS_TABLE = aws_dynamodb_table.connections.name
      ENVIRONMENT       = var.environment
    }
  }

  depends_on = [aws_cloudwatch_log_group.ws_disconnect]
  tags       = local.common_tags
}

# ---------------------------------------------------------------------------
# ws-chat
# ---------------------------------------------------------------------------
resource "aws_iam_role" "ws_chat" {
  name               = "${local.prefix}-ws-chat"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "ws_chat_basic" {
  role       = aws_iam_role.ws_chat.name
  policy_arn = aws_iam_policy.lambda_basic.arn
}

resource "aws_iam_role_policy" "ws_chat_inline" {
  name = "game-chat-permissions"
  role = aws_iam_role.ws_chat.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = [
          aws_dynamodb_table.sessions.arn,
          aws_dynamodb_table.connections.arn,
        ]
      },
      {
        Effect   = "Allow"
        Action   = ["bedrock:InvokeModelWithResponseStream", "bedrock:InvokeModel"]
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = ["execute-api:ManageConnections"]
        Resource = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*/@connections/*"
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "ws_chat" {
  name              = "/aws/lambda/${local.prefix}-ws-chat"
  retention_in_days = 7
  tags              = local.common_tags
}

resource "aws_lambda_function" "ws_chat" {
  function_name    = "${local.prefix}-ws-chat"
  role             = aws_iam_role.ws_chat.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 29 # API GW WebSocket max is 29 s per frame
  memory_size      = 256

  environment {
    variables = {
      SESSIONS_TABLE         = aws_dynamodb_table.sessions.name
      CONNECTIONS_TABLE      = aws_dynamodb_table.connections.name
      WEBSOCKET_API_ENDPOINT = "${aws_apigatewayv2_api.websocket.api_endpoint}/${aws_apigatewayv2_stage.websocket.name}"
      BEDROCK_REGION         = "us-west-2"
      ENVIRONMENT            = var.environment
    }
  }

  depends_on = [aws_cloudwatch_log_group.ws_chat]
  tags       = local.common_tags
}

# ---------------------------------------------------------------------------
# ws-game-action
# ---------------------------------------------------------------------------
resource "aws_iam_role" "ws_game_action" {
  name               = "${local.prefix}-ws-game-action"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "ws_game_action_basic" {
  role       = aws_iam_role.ws_game_action.name
  policy_arn = aws_iam_policy.lambda_basic.arn
}

resource "aws_iam_role_policy" "ws_game_action_inline" {
  name = "game-action-permissions"
  role = aws_iam_role.ws_game_action.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = [aws_dynamodb_table.sessions.arn, aws_dynamodb_table.connections.arn]
      },
      {
        Effect   = "Allow"
        Action   = ["execute-api:ManageConnections"]
        Resource = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*/@connections/*"
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "ws_game_action" {
  name              = "/aws/lambda/${local.prefix}-ws-game-action"
  retention_in_days = 7
  tags              = local.common_tags
}

resource "aws_lambda_function" "ws_game_action" {
  function_name    = "${local.prefix}-ws-game-action"
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
      SESSIONS_TABLE         = aws_dynamodb_table.sessions.name
      CONNECTIONS_TABLE      = aws_dynamodb_table.connections.name
      WEBSOCKET_API_ENDPOINT = "${aws_apigatewayv2_api.websocket.api_endpoint}/${aws_apigatewayv2_stage.websocket.name}"
      ENVIRONMENT            = var.environment
    }
  }

  depends_on = [aws_cloudwatch_log_group.ws_game_action]
  tags       = local.common_tags
}

# ---------------------------------------------------------------------------
# http-games
# ---------------------------------------------------------------------------
resource "aws_iam_role" "http_games" {
  name               = "${local.prefix}-http-games"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "http_games_basic" {
  role       = aws_iam_role.http_games.name
  policy_arn = aws_iam_policy.lambda_basic.arn
}

resource "aws_iam_role_policy" "http_games_inline" {
  name = "games-permissions"
  role = aws_iam_role.http_games.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem", "dynamodb:PutItem",
          "dynamodb:DeleteItem", "dynamodb:Query"
        ]
        Resource = [
          aws_dynamodb_table.sessions.arn,
          "${aws_dynamodb_table.sessions.arn}/index/*"
        ]
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
  name              = "/aws/lambda/${local.prefix}-http-games"
  retention_in_days = 7
  tags              = local.common_tags
}

resource "aws_lambda_function" "http_games" {
  function_name    = "${local.prefix}-http-games"
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
      SESSIONS_TABLE = aws_dynamodb_table.sessions.name
      WORLD_GEN_ARN  = aws_lambda_function.world_gen.arn
      ENVIRONMENT    = var.environment
    }
  }

  depends_on = [aws_cloudwatch_log_group.http_games]
  tags       = local.common_tags
}

# ---------------------------------------------------------------------------
# http-users
# ---------------------------------------------------------------------------
resource "aws_iam_role" "http_users" {
  name               = "${local.prefix}-http-users"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "http_users_basic" {
  role       = aws_iam_role.http_users.name
  policy_arn = aws_iam_policy.lambda_basic.arn
}

resource "aws_iam_role_policy" "http_users_inline" {
  name = "cognito-user-mgmt"
  role = aws_iam_role.http_users.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cognito-idp:AdminUpdateUserAttributes",
          "cognito-idp:AdminGetUser"
        ]
        Resource = aws_cognito_user_pool.main.arn
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "http_users" {
  name              = "/aws/lambda/${local.prefix}-http-users"
  retention_in_days = 7
  tags              = local.common_tags
}

resource "aws_lambda_function" "http_users" {
  function_name    = "${local.prefix}-http-users"
  role             = aws_iam_role.http_users.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128

  environment {
    variables = {
      USER_POOL_ID = aws_cognito_user_pool.main.id
      ENVIRONMENT  = var.environment
    }
  }

  depends_on = [aws_cloudwatch_log_group.http_users]
  tags       = local.common_tags
}

# ---------------------------------------------------------------------------
# world-gen  (invoked async — no API GW integration)
# ---------------------------------------------------------------------------
resource "aws_iam_role" "world_gen" {
  name               = "${local.prefix}-world-gen"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
  tags               = local.common_tags
}

resource "aws_iam_role_policy_attachment" "world_gen_basic" {
  role       = aws_iam_role.world_gen.name
  policy_arn = aws_iam_policy.lambda_basic.arn
}

resource "aws_iam_role_policy" "world_gen_inline" {
  name = "world-gen-permissions"
  role = aws_iam_role.world_gen.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = aws_dynamodb_table.sessions.arn
      },
      {
        Effect   = "Allow"
        Action   = ["bedrock:InvokeModel", "bedrock:InvokeModelWithResponseStream"]
        Resource = "*"
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "world_gen" {
  name              = "/aws/lambda/${local.prefix}-world-gen"
  retention_in_days = 7
  tags              = local.common_tags
}

resource "aws_lambda_function" "world_gen" {
  function_name    = "${local.prefix}-world-gen"
  role             = aws_iam_role.world_gen.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 900 # 15 minutes — world gen can be slow
  memory_size      = 256

  environment {
    variables = {
      SESSIONS_TABLE = aws_dynamodb_table.sessions.name
      BEDROCK_REGION = "us-west-2"
      ENVIRONMENT    = var.environment
    }
  }

  depends_on = [aws_cloudwatch_log_group.world_gen]
  tags       = local.common_tags
}
