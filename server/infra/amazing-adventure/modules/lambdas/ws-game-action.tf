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
