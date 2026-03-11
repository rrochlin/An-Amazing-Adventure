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
        # Read connections table to find active WebSocket connections for the game
        Effect   = "Allow"
        Action   = ["dynamodb:Query"]
        Resource = var.connections_table_index_arn
      },
      {
        # Deduct tokens from user quota after world generation
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
        # Push progress frames to connected clients via API Gateway Management API
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
