variable "prefix" { type = string }
variable "common_tags" { type = map(string) }
# cloudfront_distribution_arn and oac_id are passed in from root after CF is created.
# Bucket policy is applied via aws_s3_bucket_policy resource in root to break
# the circular dependency between s3 and cloudfront modules.

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

output "app_bucket_id" { value = aws_s3_bucket.app.id }
output "app_bucket_arn" { value = aws_s3_bucket.app.arn }
output "app_bucket_regional_domain" { value = aws_s3_bucket.app.bucket_regional_domain_name }
