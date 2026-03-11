# ── http-admin ───────────────────────────────────────────────────────────────
# Handles GET /api/admin/users, PUT /api/admin/users/{userId}, GET /api/admin/stats.
# JWT authorizer + Lambda-level admin group check (defense in depth).
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
        Effect   = "Allow"
        Action   = ["dynamodb:Scan", "dynamodb:GetItem", "dynamodb:PutItem", "dynamodb:UpdateItem"]
        Resource = var.users_table_arn
      },
      {
        # Enrich user list with email, sync group membership on role changes
        Effect = "Allow"
        Action = [
          "cognito-idp:AdminAddUserToGroup",
          "cognito-idp:AdminRemoveUserFromGroup",
          "cognito-idp:AdminGetUser",
          "cognito-idp:ListUsersInGroup"
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
  timeout          = 30
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
