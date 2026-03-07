variable "prefix" { type = string }
variable "common_tags" { type = map(string) }
variable "environment" { type = string }
variable "user_pool_id" { type = string }
variable "user_pool_client_id" { type = string }
variable "http_games_invoke_arn" { type = string }
variable "http_users_invoke_arn" { type = string }
variable "ws_connect_invoke_arn" { type = string }
variable "ws_disconnect_invoke_arn" { type = string }
variable "ws_chat_invoke_arn" { type = string }
variable "ws_game_action_invoke_arn" { type = string }
variable "http_games_function_name" { type = string }
variable "http_users_function_name" { type = string }
variable "ws_connect_function_name" { type = string }
variable "ws_disconnect_function_name" { type = string }
variable "ws_chat_function_name" { type = string }
variable "ws_game_action_function_name" { type = string }

data "aws_region" "current" {}

# ── HTTP API ─────────────────────────────────────────────────────────────────
resource "aws_apigatewayv2_api" "http" {
  name          = "${var.prefix}-http"
  protocol_type = "HTTP"
  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers = ["Content-Type", "Authorization"]
    max_age       = 300
  }
  tags = var.common_tags
}

resource "aws_apigatewayv2_authorizer" "cognito" {
  api_id           = aws_apigatewayv2_api.http.id
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  name             = "cognito"
  jwt_configuration {
    audience = [var.user_pool_client_id]
    issuer   = "https://cognito-idp.${data.aws_region.current.name}.amazonaws.com/${var.user_pool_id}"
  }
}

resource "aws_apigatewayv2_integration" "http_games" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = var.http_games_invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "http_users" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = var.http_users_invoke_arn
  payload_format_version = "2.0"
}

locals {
  jwt_auth = {
    authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
    authorization_type = "JWT"
  }
  games_target = "integrations/${aws_apigatewayv2_integration.http_games.id}"
  users_target = "integrations/${aws_apigatewayv2_integration.http_users.id}"
}

resource "aws_apigatewayv2_route" "get_games" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /api/games"
  target             = local.games_target
  authorizer_id      = local.jwt_auth.authorizer_id
  authorization_type = local.jwt_auth.authorization_type
}
resource "aws_apigatewayv2_route" "get_game" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /api/games/{uuid}"
  target             = local.games_target
  authorizer_id      = local.jwt_auth.authorizer_id
  authorization_type = local.jwt_auth.authorization_type
}
resource "aws_apigatewayv2_route" "post_game" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /api/games"
  target             = local.games_target
  authorizer_id      = local.jwt_auth.authorizer_id
  authorization_type = local.jwt_auth.authorization_type
}
resource "aws_apigatewayv2_route" "delete_game" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "DELETE /api/games/{uuid}"
  target             = local.games_target
  authorizer_id      = local.jwt_auth.authorizer_id
  authorization_type = local.jwt_auth.authorization_type
}
resource "aws_apigatewayv2_route" "get_worldready" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /api/worldready/{uuid}"
  target             = local.games_target
  authorizer_id      = local.jwt_auth.authorizer_id
  authorization_type = local.jwt_auth.authorization_type
}
resource "aws_apigatewayv2_route" "post_users" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /api/users"
  target             = local.users_target
  authorization_type = "NONE"
}
resource "aws_apigatewayv2_route" "put_users" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "PUT /api/users"
  target             = local.users_target
  authorizer_id      = local.jwt_auth.authorizer_id
  authorization_type = local.jwt_auth.authorization_type
}

resource "aws_apigatewayv2_stage" "http" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true
  tags        = var.common_tags
}

resource "aws_lambda_permission" "http_games" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.http_games_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}
resource "aws_lambda_permission" "http_users" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.http_users_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

# ── WebSocket API ─────────────────────────────────────────────────────────────
resource "aws_apigatewayv2_api" "websocket" {
  name                       = "${var.prefix}-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"
  tags                       = var.common_tags
}

resource "aws_apigatewayv2_integration" "ws_connect" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = var.ws_connect_invoke_arn
}
resource "aws_apigatewayv2_integration" "ws_disconnect" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = var.ws_disconnect_invoke_arn
}
resource "aws_apigatewayv2_integration" "ws_chat" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = var.ws_chat_invoke_arn
}
resource "aws_apigatewayv2_integration" "ws_game_action" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = var.ws_game_action_invoke_arn
}

resource "aws_apigatewayv2_route" "ws_connect" {
  api_id             = aws_apigatewayv2_api.websocket.id
  route_key          = "$connect"
  target             = "integrations/${aws_apigatewayv2_integration.ws_connect.id}"
  authorization_type = "NONE"
}
resource "aws_apigatewayv2_route" "ws_disconnect" {
  api_id    = aws_apigatewayv2_api.websocket.id
  route_key = "$disconnect"
  target    = "integrations/${aws_apigatewayv2_integration.ws_disconnect.id}"
}
resource "aws_apigatewayv2_route" "ws_chat" {
  api_id    = aws_apigatewayv2_api.websocket.id
  route_key = "chat"
  target    = "integrations/${aws_apigatewayv2_integration.ws_chat.id}"
}
resource "aws_apigatewayv2_route" "ws_game_action" {
  api_id    = aws_apigatewayv2_api.websocket.id
  route_key = "game_action"
  target    = "integrations/${aws_apigatewayv2_integration.ws_game_action.id}"
}

resource "aws_apigatewayv2_stage" "websocket" {
  api_id      = aws_apigatewayv2_api.websocket.id
  name        = var.environment
  auto_deploy = true
  tags        = var.common_tags
}

resource "aws_lambda_permission" "ws_connect" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.ws_connect_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}
resource "aws_lambda_permission" "ws_disconnect" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.ws_disconnect_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}
resource "aws_lambda_permission" "ws_chat" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.ws_chat_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}
resource "aws_lambda_permission" "ws_game_action" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = var.ws_game_action_function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}

# ── Outputs ──────────────────────────────────────────────────────────────────
# Strip https:// for use as CloudFront origin domain names
output "http_api_endpoint" {
  value = replace(aws_apigatewayv2_api.http.api_endpoint, "https://", "")
}
output "websocket_api_endpoint" {
  # Strip both https:// and wss:// prefixes — AWS uses https:// for the management API
  # but the raw endpoint may report either depending on SDK version.
  value = "${replace(replace(aws_apigatewayv2_api.websocket.api_endpoint, "https://", ""), "wss://", "")}/${var.environment}"
}
output "websocket_api_execution_arn" {
  value = aws_apigatewayv2_api.websocket.execution_arn
}
