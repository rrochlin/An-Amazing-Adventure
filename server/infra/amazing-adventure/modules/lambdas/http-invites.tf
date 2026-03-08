# ── http-invites ─────────────────────────────────────────────────────────────
# Handles POST /api/invites, GET /api/invites/{code} (no auth),
# and POST /api/invites/{code}/join (auth required).
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
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = var.invites_table_arn
      },
      {
        # Read session to validate party size; update to denormalize invite_code
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:UpdateItem"]
        Resource = var.sessions_table_arn
      },
      {
        # Write membership record when a user joins via invite
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem"]
        Resource = var.memberships_table_arn
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
      USERS_TABLE       = var.users_table_name
      INVITES_TABLE     = var.invites_table_name
      MEMBERSHIPS_TABLE = var.memberships_table_name
    }
  }
  depends_on = [aws_cloudwatch_log_group.http_invites]
  tags       = var.common_tags
}
