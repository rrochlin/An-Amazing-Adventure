output "cloudfront_domain" {
  description = "CloudFront distribution domain name"
  value       = aws_cloudfront_distribution.main.domain_name
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID (used for cache invalidation in CI)"
  value       = aws_cloudfront_distribution.main.id
}

output "app_bucket_name" {
  description = "S3 bucket name for React SPA assets"
  value       = aws_s3_bucket.app.bucket
}

output "http_api_endpoint" {
  description = "HTTP API Gateway endpoint"
  value       = aws_apigatewayv2_api.http.api_endpoint
}

output "websocket_api_endpoint" {
  description = "WebSocket API Gateway endpoint"
  value       = "${aws_apigatewayv2_api.websocket.api_endpoint}/${aws_apigatewayv2_stage.websocket.name}"
}

output "cognito_user_pool_id" {
  description = "Cognito User Pool ID"
  value       = aws_cognito_user_pool.main.id
}

output "cognito_client_id" {
  description = "Cognito App Client ID for the SPA"
  value       = aws_cognito_user_pool_client.spa.id
}

output "sessions_table_name" {
  description = "DynamoDB sessions table name"
  value       = aws_dynamodb_table.sessions.name
}

output "connections_table_name" {
  description = "DynamoDB WebSocket connections table name"
  value       = aws_dynamodb_table.connections.name
}
