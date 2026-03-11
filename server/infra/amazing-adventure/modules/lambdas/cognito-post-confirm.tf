# ── cognito-post-confirm ─────────────────────────────────────────────────────
# Triggered by Cognito Post Confirmation. Creates a default restricted UserRecord
# for every new signup. If clientMetadata.inviteCode is present, also redeems
# the invite and writes a membership record for the session.
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
        # Create the default UserRecord on first signup
        Effect   = "Allow"
        Action   = ["dynamodb:PutItem"]
        Resource = var.users_table_arn
      },
      {
        # Redeem invite code if present in clientMetadata
        Effect   = "Allow"
        Action   = ["dynamodb:GetItem", "dynamodb:UpdateItem"]
        Resource = var.invites_table_arn
      },
      {
        # Write membership record when joining via invite at signup
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
      SESSIONS_TABLE    = var.sessions_table_name
      USERS_TABLE       = var.users_table_name
      INVITES_TABLE     = var.invites_table_name
      MEMBERSHIPS_TABLE = var.memberships_table_name
    }
  }
  depends_on = [aws_cloudwatch_log_group.cognito_post_confirm]
  tags       = var.common_tags
}
