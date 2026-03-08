variable "prefix" { type = string }
variable "common_tags" { type = map(string) }
variable "sessions_table_name" { type = string }
variable "connections_table_name" { type = string }
variable "mutations_table_name" { type = string }
variable "users_table_name" { type = string }
variable "invites_table_name" { type = string }
variable "memberships_table_name" { type = string }
variable "sessions_table_arn" { type = string }
variable "connections_table_arn" { type = string }
variable "connections_table_index_arn" { type = string }
variable "mutations_table_arn" { type = string }
variable "users_table_arn" { type = string }
variable "invites_table_arn" { type = string }
variable "memberships_table_arn" { type = string }
variable "user_pool_id" { type = string }
variable "user_pool_arn" { type = string }
variable "user_pool_client_id" { type = string }
variable "websocket_api_execution_arn" { type = string }
variable "websocket_api_endpoint" { type = string }

# ── Shared bootstrap placeholder ────────────────────────────────────────────
# CI replaces function code after first deploy. We use a minimal bootstrap
# so Terraform can create the resources without a real artifact.
data "archive_file" "placeholder" {
  type        = "zip"
  output_path = "${path.module}/placeholder.zip"
  source {
    content  = "placeholder"
    filename = "bootstrap"
  }
}

# ── Shared base policy ───────────────────────────────────────────────────────
data "aws_iam_policy_document" "lambda_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_policy" "lambda_logs" {
  name = "${var.prefix}-lambda-logs"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"]
      Resource = "arn:aws:logs:*:*:*"
    }]
  })
  tags = var.common_tags
}

# ── Helper locals ────────────────────────────────────────────────────────────
# ws_endpoint_full is the value passed to WEBSOCKET_API_ENDPOINT on all Lambdas
# that push frames to connected clients. The api-gateway module output already
# includes the stage (e.g. "ba2t50m7se.execute-api.us-west-2.amazonaws.com/prod"),
# so we use it directly. wsutil.New() prepends "https://" at runtime.
locals {
  ws_endpoint_full = var.websocket_api_endpoint
}
