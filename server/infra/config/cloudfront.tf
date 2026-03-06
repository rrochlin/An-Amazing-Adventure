# ---------------------------------------------------------------------------
# ACM certificate (must be in us-east-1 for CloudFront)
# Only created when a custom domain is provided.
# ---------------------------------------------------------------------------
resource "aws_acm_certificate" "main" {
  count             = local.has_domain ? 1 : 0
  provider          = aws.us_east_1
  domain_name       = var.domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }

  tags = local.common_tags
}

# ---------------------------------------------------------------------------
# Origin request policy for WebSocket — forwards the required WS headers
# ---------------------------------------------------------------------------
resource "aws_cloudfront_origin_request_policy" "websocket" {
  name    = "${local.prefix}-websocket-headers"
  comment = "Forward WebSocket upgrade headers to API GW origin"

  cookies_config {
    cookie_behavior = "none"
  }

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

  query_strings_config {
    query_string_behavior = "all" # pass ?token= through to ws-connect
  }
}

# ---------------------------------------------------------------------------
# Cache policies
# ---------------------------------------------------------------------------

# API routes — never cache
resource "aws_cloudfront_cache_policy" "no_cache" {
  name        = "${local.prefix}-no-cache"
  comment     = "No caching for API and WebSocket origins"
  min_ttl     = 0
  default_ttl = 0
  max_ttl     = 0

  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config {
      cookie_behavior = "none"
    }
    headers_config {
      header_behavior = "none"
    }
    query_strings_config {
      query_string_behavior = "none"
    }
    enable_accept_encoding_brotli = false
    enable_accept_encoding_gzip   = false
  }
}

# ---------------------------------------------------------------------------
# CloudFront distribution
# Origin 0: S3 (default — serves SPA assets)
# Origin 1: HTTP API Gateway (/api/*)
# Origin 2: WebSocket API Gateway (/ws)
# ---------------------------------------------------------------------------

locals {
  s3_origin_id   = "s3-app"
  http_origin_id = "apigw-http"
  ws_origin_id   = "apigw-ws"

  # Strip the https:// prefix from the API GW endpoint URLs for CF origins
  http_api_host = replace(aws_apigatewayv2_api.http.api_endpoint, "https://", "")
  ws_api_host   = replace(aws_apigatewayv2_api.websocket.api_endpoint, "https://", "")
}

resource "aws_cloudfront_distribution" "main" {
  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = "index.html"
  price_class         = "PriceClass_100" # US, Canada, Europe only — cheapest
  comment             = local.prefix
  tags                = local.common_tags

  aliases = local.has_domain ? [var.domain_name] : []

  # S3 origin for static assets
  origin {
    origin_id                = local.s3_origin_id
    domain_name              = aws_s3_bucket.app.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.app.id
  }

  # HTTP API Gateway origin
  origin {
    origin_id   = local.http_origin_id
    domain_name = local.http_api_host

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  # WebSocket API Gateway origin
  origin {
    origin_id   = local.ws_origin_id
    domain_name = local.ws_api_host

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  # Default behaviour — serve S3 SPA assets
  default_cache_behavior {
    target_origin_id       = local.s3_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD", "OPTIONS"]
    cached_methods         = ["GET", "HEAD"]
    compress               = true
    cache_policy_id        = "658327ea-f89d-4fab-a63d-7e88639e58f6" # AWS managed: CachingOptimized
  }

  # /api/* — forward to HTTP API Gateway, never cache
  ordered_cache_behavior {
    path_pattern           = "/api/*"
    target_origin_id       = local.http_origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = aws_cloudfront_cache_policy.no_cache.id
  }

  # /ws — forward to WebSocket API Gateway with WS headers
  ordered_cache_behavior {
    path_pattern             = "/ws"
    target_origin_id         = local.ws_origin_id
    viewer_protocol_policy   = "redirect-to-https"
    allowed_methods          = ["GET", "HEAD"]
    cached_methods           = ["GET", "HEAD"]
    cache_policy_id          = aws_cloudfront_cache_policy.no_cache.id
    origin_request_policy_id = aws_cloudfront_origin_request_policy.websocket.id
  }

  # SPA fallback: 403/404 from S3 → return index.html with 200
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
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn            = local.has_domain ? aws_acm_certificate.main[0].arn : null
    ssl_support_method             = local.has_domain ? "sni-only" : null
    minimum_protocol_version       = local.has_domain ? "TLSv1.2_2021" : "TLSv1"
    cloudfront_default_certificate = local.has_domain ? false : true
  }
}
