output "cloudfront_domain" {
  description = "CloudFront distribution domain name"
  value       = module.cloudfront.distribution_domain
}

output "cloudfront_distribution_id" {
  description = "CloudFront distribution ID (used for cache invalidation in CI)"
  value       = module.cloudfront.distribution_id
}

output "app_bucket_name" {
  description = "S3 bucket name for React SPA assets"
  value       = module.s3.app_bucket_id
}

output "http_api_endpoint" {
  description = "HTTP API Gateway invoke URL"
  value       = module.api_gateway.http_api_endpoint
}

output "websocket_api_endpoint" {
  description = "WebSocket API Gateway endpoint (with stage)"
  value       = module.api_gateway.websocket_api_endpoint
}

output "cognito_user_pool_id" {
  description = "Cognito User Pool ID"
  value       = module.cognito.user_pool_id
}

output "cognito_client_id" {
  description = "Cognito SPA App Client ID"
  value       = module.cognito.user_pool_client_id
}

output "sessions_table_name" {
  description = "DynamoDB sessions table name"
  value       = module.dynamodb.sessions_table_name
}

output "connections_table_name" {
  description = "DynamoDB WebSocket connections table name"
  value       = module.dynamodb.connections_table_name
}

output "users_table_name" {
  description = "DynamoDB users table name (RBAC + quota)"
  value       = module.dynamodb.users_table_name
}

output "memberships_table_name" {
  description = "DynamoDB memberships table name (user ↔ session join)"
  value       = module.dynamodb.memberships_table_name
}
