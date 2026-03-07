variable "prefix" { type = string }
variable "common_tags" { type = map(string) }

resource "aws_cognito_user_pool" "main" {
  name                     = var.prefix
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  username_configuration {
    case_sensitive = false
  }

  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
    email_subject        = "Your verification seal — An Amazing Adventure"
    email_message        = "Hail, adventurer.\n\nYour verification seal is: {####}\n\nSpeak this code to complete your oath of registration. It expires in 24 hours.\n\nIf you did not attempt to register, you may disregard this scroll.\n\n-- An Amazing Adventure"
  }

  password_policy {
    minimum_length                   = 8
    require_uppercase                = true
    require_lowercase                = true
    require_numbers                  = true
    require_symbols                  = false
    temporary_password_validity_days = 7
  }

  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  schema {
    name                = "email"
    attribute_data_type = "String"
    required            = true
    mutable             = true
    string_attribute_constraints {
      min_length = 5
      max_length = 254
    }
  }

  # Invitation email — sent when an admin creates a user account
  admin_create_user_config {
    allow_admin_create_user_only = false

    invite_message_template {
      email_subject = "⚔ Your invitation to An Amazing Adventure"

      email_message = "Hail, {username}.\n\nYou have been summoned to An Amazing Adventure. Your temporary passphrase is:\n\n{####}\n\nYou will be asked to forge a permanent passphrase upon your first entry. This invitation expires in 7 days.\n\n-- An Amazing Adventure"
      sms_message   = "An Amazing Adventure: {username}, your temporary password is {####}"
    }
  }

  tags = var.common_tags
}

resource "aws_cognito_user_pool_client" "spa" {
  name         = "${var.prefix}-spa"
  user_pool_id = aws_cognito_user_pool.main.id

  generate_secret = false

  explicit_auth_flows = [
    "ALLOW_USER_SRP_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]

  access_token_validity  = 1
  id_token_validity      = 1
  refresh_token_validity = 30

  token_validity_units {
    access_token  = "hours"
    id_token      = "hours"
    refresh_token = "days"
  }

  prevent_user_existence_errors = "ENABLED"
}

resource "aws_ssm_parameter" "user_pool_id" {
  name  = "/${var.prefix}/cognito/user-pool-id"
  type  = "String"
  value = aws_cognito_user_pool.main.id
  tags  = var.common_tags
}

resource "aws_ssm_parameter" "user_pool_client_id" {
  name  = "/${var.prefix}/cognito/client-id"
  type  = "String"
  value = aws_cognito_user_pool_client.spa.id
  tags  = var.common_tags
}

output "user_pool_id" { value = aws_cognito_user_pool.main.id }
output "user_pool_arn" { value = aws_cognito_user_pool.main.arn }
output "user_pool_client_id" { value = aws_cognito_user_pool_client.spa.id }
