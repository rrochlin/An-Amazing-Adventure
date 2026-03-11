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
        # Read/write memberships to track session ownership and party membership
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:DeleteItem", "dynamodb:Query"]
        Resource = [var.memberships_table_arn, "${var.memberships_table_arn}/index/*"]
      },
      {
        # Read per-user RBAC record to enforce games limit
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
      USERS_TABLE       = var.users_table_name
      MEMBERSHIPS_TABLE = var.memberships_table_name
      WORLD_GEN_ARN     = aws_lambda_function.world_gen.arn
    }
  }
  depends_on = [aws_cloudwatch_log_group.http_games]
  tags       = var.common_tags
}
