variable "app_name" {
  description = "Application name used as prefix for all resources"
  type        = string
  default     = "amazing-adventure"
}

variable "environment" {
  description = "Deployment environment (prod, dev)"
  type        = string
  default     = "prod"
}

variable "aws_region" {
  description = "Primary AWS deployment region"
  type        = string
  default     = "us-west-2"
}

variable "domain_name" {
  description = "Custom domain for CloudFront (e.g. play.example.com). Leave empty to use the CloudFront default domain."
  type        = string
  default     = ""
}

locals {
  prefix = "${var.app_name}-${var.environment}"
  common_tags = {
    App         = var.app_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
