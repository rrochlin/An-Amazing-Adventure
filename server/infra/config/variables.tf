variable "app_name" {
  description = "Application name used as a prefix for all resources"
  type        = string
  default     = "amazing-adventure"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "prod"
}

variable "domain_name" {
  description = "Custom domain for the CloudFront distribution (e.g. play.example.com). Leave empty to use the CloudFront default domain."
  type        = string
  default     = ""
}

locals {
  prefix     = "${var.app_name}-${var.environment}"
  has_domain = var.domain_name != ""
  common_tags = {
    App         = var.app_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
