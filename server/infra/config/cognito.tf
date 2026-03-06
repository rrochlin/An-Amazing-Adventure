# ---------------------------------------------------------------------------
# Cognito User Pool
# Simple email-only setup — no SMS, no social IdPs.
# ---------------------------------------------------------------------------
resource "aws_cognito_user_pool" "main" {
  name = local.prefix

  # Case-insensitive email as the username
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  username_configuration {
    case_sensitive = false
  }

  # Email verification on sign-up (no SMS)
  verification_message_template {
    default_email_option = "CONFIRM_WITH_CODE"
    email_subject        = "Your Amazing Adventure verification code"
    email_message        = "Your verification code is {####}"
  }

  # Password policy
  password_policy {
    minimum_length                   = 8
    require_uppercase                = true
    require_lowercase                = true
    require_numbers                  = true
    require_symbols                  = false
    temporary_password_validity_days = 7
  }

  # Account recovery via email only
  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  # Keep standard email attribute
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

  tags = local.common_tags
}

# ---------------------------------------------------------------------------
# App client — public SRP client for the React SPA
# No client secret; uses SRP auth flow.
# ---------------------------------------------------------------------------
resource "aws_cognito_user_pool_client" "spa" {
  name         = "${local.prefix}-spa"
  user_pool_id = aws_cognito_user_pool.main.id

  # No client secret — this is a public browser client
  generate_secret = false

  explicit_auth_flows = [
    "ALLOW_USER_SRP_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]

  # Token validity
  access_token_validity  = 1  # hours
  id_token_validity      = 1  # hours
  refresh_token_validity = 30 # days

  token_validity_units {
    access_token  = "hours"
    id_token      = "hours"
    refresh_token = "days"
  }

  prevent_user_existence_errors = "ENABLED"
}

# ---------------------------------------------------------------------------
# SSM outputs — Lambda functions read these at startup
# ---------------------------------------------------------------------------
resource "aws_ssm_parameter" "user_pool_id" {
  name  = "/${local.prefix}/cognito/user-pool-id"
  type  = "String"
  value = aws_cognito_user_pool.main.id
  tags  = local.common_tags
}

resource "aws_ssm_parameter" "user_pool_client_id" {
  name  = "/${local.prefix}/cognito/client-id"
  type  = "String"
  value = aws_cognito_user_pool_client.spa.id
  tags  = local.common_tags
}
