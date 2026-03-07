terraform {
  required_providers {
    aws = {
      source                = "hashicorp/aws"
      version               = "~> 5.0"
      configuration_aliases = [aws.us_east_1]
    }
  }
}

variable "prefix" { type = string }
variable "common_tags" { type = map(string) }
variable "domain_name" {
  type    = string
  default = ""
}
variable "app_bucket_regional_domain" { type = string }
variable "app_bucket_id" { type = string }
variable "http_api_endpoint" { type = string }
variable "websocket_api_endpoint" { type = string }

locals {
  has_domain     = var.domain_name != ""
  s3_origin_id   = "s3-${var.app_bucket_id}"
  http_origin_id = "apigw-http"
  ws_origin_id   = "apigw-ws"
}

resource "aws_acm_certificate" "main" {
  count             = local.has_domain ? 1 : 0
  provider          = aws.us_east_1
  domain_name       = var.domain_name
  validation_method = "DNS"
  lifecycle { create_before_destroy = true }
  tags = var.common_tags
}

resource "aws_cloudfront_origin_request_policy" "websocket" {
  name    = "${var.prefix}-websocket-headers"
  comment = "Forward WebSocket upgrade headers"
  cookies_config { cookie_behavior = "none" }
  headers_config {
    header_behavior = "whitelist"
    headers {
      items = [
        "Sec-WebSocket-Key",
        "Sec-WebSocket-Version",
        "Sec-WebSocket-Protocol",
        "Sec-WebSocket-Accept",
        "Sec-WebSocket-Extensions",
      ]
    }
  }
  query_strings_config { query_string_behavior = "all" }
}

resource "aws_cloudfront_cache_policy" "no_cache" {
  name        = "${var.prefix}-no-cache"
  comment     = "No caching for API and WebSocket origins"
  min_ttl     = 0
  default_ttl = 0
  max_ttl     = 0
  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config { cookie_behavior = "none" }
    headers_config { header_behavior = "none" }
    query_strings_config { query_string_behavior = "none" }
    enable_accept_encoding_brotli = false
    enable_accept_encoding_gzip   = false
  }
}

resource "aws_cloudfront_distribution" "main" {
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  price_class         = "PriceClass_100"
  comment             = var.prefix
  tags                = var.common_tags
  aliases             = local.has_domain ? [var.domain_name] : []

  # S3 origin
  origin {
    origin_id                = local.s3_origin_id
    domain_name              = var.app_bucket_regional_domain
    origin_access_control_id = aws_cloudfront_origin_access_control.app.id
  }

  # HTTP API Gateway origin — domain only, no protocol prefix
  origin {
    origin_id   = local.http_origin_id
    domain_name = var.http_api_endpoint
    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  # WebSocket API Gateway origin — domain only, no protocol prefix
  origin {
    origin_id   = local.ws_origin_id
    domain_name = split("/", var.websocket_api_endpoint)[0]
    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  default_cache_behavior {
    target_origin_id       = local.s3_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true
    # AWS managed CachingOptimized policy
    cache_policy_id = "658327ea-f89d-4fab-a63d-7e88639e58f6"
  }

  ordered_cache_behavior {
    path_pattern           = "/api/*"
    target_origin_id       = local.http_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = aws_cloudfront_cache_policy.no_cache.id
    # Forward all headers except Host so API Gateway sees its own domain, not the CF domain.
    # Without this, API GW returns 403 Forbidden because the Host header is the CF domain.
    origin_request_policy_id = "b689b0a8-53d0-40ab-baf2-68738e2966ac" # Managed-AllViewerExceptHostHeader
  }

  ordered_cache_behavior {
    path_pattern             = "/ws"
    target_origin_id         = local.ws_origin_id
    viewer_protocol_policy   = "redirect-to-https"
    allowed_methods          = ["GET", "HEAD"]
    cached_methods           = ["GET", "HEAD"]
    cache_policy_id          = aws_cloudfront_cache_policy.no_cache.id
    origin_request_policy_id = aws_cloudfront_origin_request_policy.websocket.id
  }

  custom_error_response {
    error_code            = 403
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 0
  }
  custom_error_response {
    error_code            = 404
    response_code         = 200
    response_page_path    = "/index.html"
    error_caching_min_ttl = 0
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }

  viewer_certificate {
    acm_certificate_arn            = local.has_domain ? aws_acm_certificate.main[0].arn : null
    ssl_support_method             = local.has_domain ? "sni-only" : null
    minimum_protocol_version       = local.has_domain ? "TLSv1.2_2021" : "TLSv1"
    cloudfront_default_certificate = !local.has_domain
  }
}

resource "aws_cloudfront_origin_access_control" "app" {
  name                              = "${var.prefix}-oac"
  description                       = "OAC for ${var.prefix} SPA bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

output "distribution_arn" { value = aws_cloudfront_distribution.main.arn }
output "distribution_id" { value = aws_cloudfront_distribution.main.id }
output "distribution_domain" { value = aws_cloudfront_distribution.main.domain_name }
