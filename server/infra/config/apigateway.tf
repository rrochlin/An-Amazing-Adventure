# ---------------------------------------------------------------------------
# HTTP API (API Gateway V2) — REST endpoints
# ---------------------------------------------------------------------------
resource "aws_apigatewayv2_api" "http" {
  name          = "${local.prefix}-http"
  protocol_type = "HTTP"

  cors_configuration {
    allow_origins = ["*"]
    allow_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allow_headers = ["Content-Type", "Authorization"]
    max_age       = 300
  }

  tags = local.common_tags
}

# Cognito JWT authorizer for HTTP API
resource "aws_apigatewayv2_authorizer" "cognito" {
  api_id           = aws_apigatewayv2_api.http.id
  authorizer_type  = "JWT"
  identity_sources = ["$request.header.Authorization"]
  name             = "cognito"

  jwt_configuration {
    audience = [aws_cognito_user_pool_client.spa.id]
    issuer   = "https://cognito-idp.us-west-2.amazonaws.com/${aws_cognito_user_pool.main.id}"
  }
}

# Lambda integrations (AWS_PROXY)
resource "aws_apigatewayv2_integration" "http_games" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.http_games.invoke_arn
  payload_format_version = "2.0"
}

resource "aws_apigatewayv2_integration" "http_users" {
  api_id                 = aws_apigatewayv2_api.http.id
  integration_type       = "AWS_PROXY"
  integration_uri        = aws_lambda_function.http_users.invoke_arn
  payload_format_version = "2.0"
}

# Routes
resource "aws_apigatewayv2_route" "get_games" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /api/games"
  target             = "integrations/${aws_apigatewayv2_integration.http_games.id}"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
  authorization_type = "JWT"
}

resource "aws_apigatewayv2_route" "get_game" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /api/games/{uuid}"
  target             = "integrations/${aws_apigatewayv2_integration.http_games.id}"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
  authorization_type = "JWT"
}

resource "aws_apigatewayv2_route" "post_game" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /api/games"
  target             = "integrations/${aws_apigatewayv2_integration.http_games.id}"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
  authorization_type = "JWT"
}

resource "aws_apigatewayv2_route" "delete_game" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "DELETE /api/games/{uuid}"
  target             = "integrations/${aws_apigatewayv2_integration.http_games.id}"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
  authorization_type = "JWT"
}

resource "aws_apigatewayv2_route" "get_worldready" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "GET /api/worldready/{uuid}"
  target             = "integrations/${aws_apigatewayv2_integration.http_games.id}"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
  authorization_type = "JWT"
}

resource "aws_apigatewayv2_route" "post_users" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "POST /api/users"
  target             = "integrations/${aws_apigatewayv2_integration.http_users.id}"
  authorization_type = "NONE"
}

resource "aws_apigatewayv2_route" "put_users" {
  api_id             = aws_apigatewayv2_api.http.id
  route_key          = "PUT /api/users"
  target             = "integrations/${aws_apigatewayv2_integration.http_users.id}"
  authorizer_id      = aws_apigatewayv2_authorizer.cognito.id
  authorization_type = "JWT"
}

resource "aws_apigatewayv2_stage" "http" {
  api_id      = aws_apigatewayv2_api.http.id
  name        = "$default"
  auto_deploy = true

  tags = local.common_tags
}

# Lambda permissions for HTTP API
resource "aws_lambda_permission" "http_games" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.http_games.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

resource "aws_lambda_permission" "http_users" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.http_users.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.http.execution_arn}/*/*"
}

# ---------------------------------------------------------------------------
# WebSocket API (API Gateway V2)
# ---------------------------------------------------------------------------
resource "aws_apigatewayv2_api" "websocket" {
  name                       = "${local.prefix}-ws"
  protocol_type              = "WEBSOCKET"
  route_selection_expression = "$request.body.action"

  tags = local.common_tags
}

# WebSocket Lambda integrations
resource "aws_apigatewayv2_integration" "ws_connect" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.ws_connect.invoke_arn
}

resource "aws_apigatewayv2_integration" "ws_disconnect" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.ws_disconnect.invoke_arn
}

resource "aws_apigatewayv2_integration" "ws_chat" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.ws_chat.invoke_arn
}

resource "aws_apigatewayv2_integration" "ws_game_action" {
  api_id           = aws_apigatewayv2_api.websocket.id
  integration_type = "AWS_PROXY"
  integration_uri  = aws_lambda_function.ws_game_action.invoke_arn
}

# WebSocket routes
resource "aws_apigatewayv2_route" "ws_connect" {
  api_id    = aws_apigatewayv2_api.websocket.id
  route_key = "$connect"
  target    = "integrations/${aws_apigatewayv2_integration.ws_connect.id}"
  # Token is passed as ?token=... query string and validated inside the Lambda
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

  tags = local.common_tags
}

# Lambda permissions for WebSocket API
resource "aws_lambda_permission" "ws_connect" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ws_connect.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}

resource "aws_lambda_permission" "ws_disconnect" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ws_disconnect.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}

resource "aws_lambda_permission" "ws_chat" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ws_chat.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}

resource "aws_lambda_permission" "ws_game_action" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.ws_game_action.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_apigatewayv2_api.websocket.execution_arn}/*/*"
}
