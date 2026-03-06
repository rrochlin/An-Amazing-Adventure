variable "prefix" { type = string }
variable "common_tags" { type = map(string) }
variable "cloudfront_distribution_arn" { type = string }

resource "aws_s3_bucket" "app" {
  bucket = "${var.prefix}-app"
  tags   = merge(var.common_tags, { Name = "AppAssets" })
}

resource "aws_s3_bucket_versioning" "app" {
  bucket = aws_s3_bucket.app.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "app" {
  bucket = aws_s3_bucket.app.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}

resource "aws_s3_bucket_public_access_block" "app" {
  bucket                  = aws_s3_bucket.app.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_cloudfront_origin_access_control" "app" {
  name                              = "${var.prefix}-oac"
  description                       = "OAC for ${var.prefix} SPA bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_s3_bucket_policy" "app_cf_read" {
  bucket     = aws_s3_bucket.app.id
  depends_on = [aws_s3_bucket_public_access_block.app]

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "AllowCloudFrontServicePrincipal"
      Effect    = "Allow"
      Principal = { Service = "cloudfront.amazonaws.com" }
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.app.arn}/*"
      Condition = {
        StringEquals = { "AWS:SourceArn" = var.cloudfront_distribution_arn }
      }
    }]
  })
}

output "app_bucket_id" { value = aws_s3_bucket.app.id }
output "app_bucket_regional_domain" { value = aws_s3_bucket.app.bucket_regional_domain_name }
output "oac_id" { value = aws_cloudfront_origin_access_control.app.id }
