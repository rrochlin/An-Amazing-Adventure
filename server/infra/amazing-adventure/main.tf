terraform {
  backend "s3" {
    bucket         = "roberts-personal-tf-bucket"
    key            = "amazing-adventure/terraform.tfstate"
    region         = "us-west-2"
    dynamodb_table = "terraform-state-locking"
    encrypt        = true
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  required_version = ">= 1.5.0"
}

provider "aws" {
  region = var.aws_region
}

# ACM certificates for CloudFront must be provisioned in us-east-1
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

# ── Modules ──────────────────────────────────────────────────────────────────

module "dynamodb" {
  source      = "./modules/dynamodb"
  prefix      = local.prefix
  common_tags = local.common_tags
}

module "cognito" {
  source      = "./modules/cognito"
  prefix      = local.prefix
  common_tags = local.common_tags
}

module "s3" {
  source      = "./modules/s3"
  prefix      = local.prefix
  common_tags = local.common_tags
}

# Bucket policy lives in root to break the s3 <-> cloudfront circular dependency.
# CloudFront OAC is created inside the cloudfront module; its ID is passed here.
resource "aws_s3_bucket_policy" "app_cf_read" {
  bucket     = module.s3.app_bucket_id
  depends_on = [module.s3]

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowCloudFrontServicePrincipal"
      Effect    = "Allow"
      Principal = { Service = "cloudfront.amazonaws.com" }
      Action    = "s3:GetObject"
      Resource  = "${module.s3.app_bucket_arn}/*"
      Condition = {
        StringEquals = { "AWS:SourceArn" = module.cloudfront.distribution_arn }
      }
    }]
  })
}

module "lambdas" {
  source                      = "./modules/lambdas"
  prefix                      = local.prefix
  common_tags                 = local.common_tags
  sessions_table_name         = module.dynamodb.sessions_table_name
  connections_table_name      = module.dynamodb.connections_table_name
  sessions_table_arn          = module.dynamodb.sessions_table_arn
  connections_table_arn       = module.dynamodb.connections_table_arn
  connections_table_index_arn = module.dynamodb.connections_table_index_arn
  user_pool_id                = module.cognito.user_pool_id
  user_pool_arn               = module.cognito.user_pool_arn
  websocket_api_execution_arn = module.api_gateway.websocket_api_execution_arn
  websocket_api_endpoint      = module.api_gateway.websocket_api_endpoint
  websocket_stage_name        = var.environment
}

module "api_gateway" {
  source                       = "./modules/api-gateway"
  prefix                       = local.prefix
  common_tags                  = local.common_tags
  environment                  = var.environment
  user_pool_id                 = module.cognito.user_pool_id
  user_pool_client_id          = module.cognito.user_pool_client_id
  http_games_invoke_arn        = module.lambdas.http_games_invoke_arn
  http_users_invoke_arn        = module.lambdas.http_users_invoke_arn
  ws_connect_invoke_arn        = module.lambdas.ws_connect_invoke_arn
  ws_disconnect_invoke_arn     = module.lambdas.ws_disconnect_invoke_arn
  ws_chat_invoke_arn           = module.lambdas.ws_chat_invoke_arn
  ws_game_action_invoke_arn    = module.lambdas.ws_game_action_invoke_arn
  http_games_function_name     = module.lambdas.http_games_function_name
  http_users_function_name     = module.lambdas.http_users_function_name
  ws_connect_function_name     = module.lambdas.ws_connect_function_name
  ws_disconnect_function_name  = module.lambdas.ws_disconnect_function_name
  ws_chat_function_name        = module.lambdas.ws_chat_function_name
  ws_game_action_function_name = module.lambdas.ws_game_action_function_name
}

module "cloudfront" {
  source                     = "./modules/cloudfront"
  prefix                     = local.prefix
  common_tags                = local.common_tags
  domain_name                = var.domain_name
  app_bucket_regional_domain = module.s3.app_bucket_regional_domain
  app_bucket_id              = module.s3.app_bucket_id
  http_api_endpoint          = module.api_gateway.http_api_endpoint
  providers = {
    aws           = aws
    aws.us_east_1 = aws.us_east_1
  }
}
