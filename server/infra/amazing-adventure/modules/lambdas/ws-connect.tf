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
        Action   = ["dynamodb:PutItem", "dynamodb:DeleteItem", "dynamodb:Query"]
        Resource = [var.connections_table_arn, var.connections_table_index_arn]
      },
      {
        # Validate that the session exists before accepting the connection
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
