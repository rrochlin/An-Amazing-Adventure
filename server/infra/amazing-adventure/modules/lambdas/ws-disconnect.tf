# ── ws-disconnect ────────────────────────────────────────────────────────────
resource "aws_iam_role" "ws_disconnect" {
  name               = "${var.prefix}-ws-disconnect"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume.json
  tags               = var.common_tags
}
resource "aws_iam_role_policy_attachment" "ws_disconnect_logs" {
  role       = aws_iam_role.ws_disconnect.name
  policy_arn = aws_iam_policy.lambda_logs.arn
}
resource "aws_iam_role_policy" "ws_disconnect" {
  name = "connections"
  role = aws_iam_role.ws_disconnect.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["dynamodb:DeleteItem", "dynamodb:GetItem"]
      Resource = var.connections_table_arn
    }]
  })
}
resource "aws_cloudwatch_log_group" "ws_disconnect" {
  name              = "/aws/lambda/${var.prefix}-ws-disconnect"
  retention_in_days = 7
  tags              = var.common_tags
}
resource "aws_lambda_function" "ws_disconnect" {
  function_name    = "${var.prefix}-ws-disconnect"
  role             = aws_iam_role.ws_disconnect.arn
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  handler          = "bootstrap"
  filename         = data.archive_file.placeholder.output_path
  source_code_hash = data.archive_file.placeholder.output_base64sha256
  timeout          = 10
  memory_size      = 128
  environment {
    variables = { CONNECTIONS_TABLE = var.connections_table_name }
  }
  depends_on = [aws_cloudwatch_log_group.ws_disconnect]
  tags       = var.common_tags
}
