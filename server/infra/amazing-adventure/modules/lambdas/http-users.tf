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
