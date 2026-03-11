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
        Action   = ["dynamodb:PutItem"]
        Resource = var.mutations_table_arn
      },
      {
        # Read per-user RBAC record to enforce AI access and token quota
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
