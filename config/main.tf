terraform {
  backend "s3" {
    bucket         = "roberts-personal-tf-bucket"
    key            = "03-basics/web-app/terraform.tfstate"
    region         = "us-west-2"
    dynamodb_table = "terraform-state-locking"
    encrypt        = true
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  region = "us-west-2"
}

resource "aws_dynamodb_table" "users" {
  name         = "users"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "user_id"

  attribute {
    name = "user_id"
    type = "B"
  }

  attribute {
    name = "email"
    type = "S"
  }

  global_secondary_index {
    name            = "email-index"
    hash_key        = "email"
    projection_type = "ALL"
  }

  tags = {
    Name = "Users"
  }
}

resource "aws_dynamodb_table" "amazing_adventure_data" {
  name         = "amazing-adventure-data"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "session_id"

  attribute {
    name = "session_id"
    type = "B"
  }

  attribute {
    name = "user_id"
    type = "B"
  }

  global_secondary_index {
    name            = "user-sessions-index"
    hash_key        = "user_id"
    projection_type = "ALL"
  }

  tags = {
    Name = "GameSessions"
  }
}

resource "aws_dynamodb_table" "refresh_tokens" {
  name         = "refresh-tokens"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "token"

  attribute {
    name = "token"
    type = "S"
  }

  attribute {
    name = "user_id"
    type = "B"
  }

  # Query all tokens for a user
  global_secondary_index {
    name            = "user-tokens-index"
    hash_key        = "user_id"
    projection_type = "ALL"
  }

  # Auto-expire items based on epoch seconds stored in the "expires_at" attribute
  ttl {
    attribute_name = "expires_at"
    enabled        = true
  }

  tags = {
    Name = "RefreshTokens"
  }
}

# S3 Bucket for AI-generated map images
resource "aws_s3_bucket" "map_images" {
  bucket = "amazing-adventure-map-images"

  tags = {
    Name        = "MapImages"
    Environment = "production"
    Purpose     = "AI-generated game map storage"
  }
}

# Enable versioning (useful if AI regenerates maps)
resource "aws_s3_bucket_versioning" "map_images" {
  bucket = aws_s3_bucket.map_images.id

  versioning_configuration {
    status = "Enabled"
  }
}

# Server-side encryption with S3-managed keys
resource "aws_s3_bucket_server_side_encryption_configuration" "map_images" {
  bucket = aws_s3_bucket.map_images.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# CORS configuration for React client
resource "aws_s3_bucket_cors_configuration" "map_images" {
  bucket = aws_s3_bucket.map_images.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "HEAD"]
    allowed_origins = ["*"] # Can be tightened to specific domains later
    expose_headers  = ["ETag"]
    max_age_seconds = 3600
  }
}

# Block public ACLs (modern best practice)
resource "aws_s3_bucket_public_access_block" "map_images" {
  bucket = aws_s3_bucket.map_images.id

  block_public_acls       = true
  block_public_policy     = false # Allow bucket policy for public read
  ignore_public_acls      = true
  restrict_public_buckets = false # Allow public read via bucket policy
}

# Bucket policy for public read access
resource "aws_s3_bucket_policy" "map_images_public_read" {
  bucket = aws_s3_bucket.map_images.id

  # Ensure public access block is configured first
  depends_on = [aws_s3_bucket_public_access_block.map_images]

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicReadGetObject"
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource  = "${aws_s3_bucket.map_images.arn}/*"
      }
    ]
  })
}

# IAM policy for server to upload images
resource "aws_iam_policy" "map_images_upload" {
  name        = "AmazingAdventureMapImagesUpload"
  description = "Allow server to upload map images to S3"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:PutObjectAcl",
          "s3:GetObject",
          "s3:DeleteObject"
        ]
        Resource = "${aws_s3_bucket.map_images.arn}/*"
      },
      {
        Effect = "Allow"
        Action = [
          "s3:ListBucket"
        ]
        Resource = aws_s3_bucket.map_images.arn
      }
    ]
  })

  tags = {
    Name = "MapImagesUploadPolicy"
  }
}

# IAM user for game server (imported from existing)
resource "aws_iam_user" "game_server" {
  name = "AnAmazingAdventureServer"

  tags = {
    Name        = "GameServer"
    Environment = "production"
    Purpose     = "Game server service account"
  }
}

# Attach S3 upload policy to game server user
resource "aws_iam_user_policy_attachment" "game_server_s3_upload" {
  user       = aws_iam_user.game_server.name
  policy_arn = aws_iam_policy.map_images_upload.arn
}
