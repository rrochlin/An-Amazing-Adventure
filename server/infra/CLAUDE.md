# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This repository manages AWS infrastructure for game development projects using Terraform. The infrastructure supports "Amazing Adventure" with DynamoDB tables for user authentication, game sessions, and refresh token management.

## Secrets Management

This project uses **Doppler** for secrets management. AWS credentials are injected via Doppler.

**Setup** (one-time):
```bash
doppler setup
```

**Running Commands**: All Terraform commands must be prefixed with `doppler run --` to inject AWS credentials.

## Terraform Commands

All Terraform commands must be run from the `config/` directory with Doppler:

```bash
cd config

# Initialize Terraform (required after cloning or adding new providers)
doppler run -- terraform init

# Validate configuration syntax
doppler run -- terraform validate

# Format Terraform files
doppler run -- terraform fmt

# Check formatting without making changes
doppler run -- terraform fmt -check

# Preview changes
doppler run -- terraform plan

# Apply changes
doppler run -- terraform apply

# Apply without confirmation (use with caution)
doppler run -- terraform apply -auto-approve

# Show current state
doppler run -- terraform show

# List all resources in state
doppler run -- terraform state list

# Destroy all resources (use with caution)
doppler run -- terraform destroy
```

## Infrastructure Architecture

### State Management
- **Backend**: S3 bucket (`roberts-personal-tf-bucket`)
- **State File Path**: `03-basics/web-app/terraform.tfstate`
- **State Locking**: DynamoDB table (`terraform-state-locking`)
- **Region**: us-west-2
- **Encryption**: Enabled

### DynamoDB Tables

**users**
- Primary Key: `user_id` (Binary)
- GSI: `email-index` on `email` (String)
- Billing: Pay-per-request

**amazing-adventure-data**
- Primary Key: `session_id` (Binary)
- GSI: `user-sessions-index` on `user_id` (Binary)
- Billing: Pay-per-request

**refresh-tokens**
- Primary Key: `token` (String)
- GSI: `user-tokens-index` on `user_id` (Binary)
- TTL: Enabled on `expires_at` attribute (automatic expiration)
- Billing: Pay-per-request

**Important**: All UUID fields (`user_id`, `session_id`) use Binary (B) type, not String (S), to align with Golang's binary UUID storage format.

## CI/CD Pipeline

GitHub Actions workflow (`.github/workflows/terraform.yml`) runs on:
- Pushes to `main` branch
- Release creation

Workflow steps:
1. Format check (`terraform fmt -check`)
2. Initialize (`terraform init`)
3. Plan (on pull requests, posts results as PR comment)
4. Apply (on main branch, auto-approves)

Working directory for all jobs: `config/`

### Required Secrets
Configure in GitHub repository settings:
- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`

## Development Workflow

1. Create feature branch: `git checkout -b feature/description`
2. Make changes to `config/main.tf`
3. Test locally:
   ```bash
   cd config
   doppler run -- terraform fmt
   doppler run -- terraform validate
   doppler run -- terraform plan
   ```
4. Commit and push
5. Create PR to trigger plan workflow
6. Review plan in PR comments
7. Merge to main to auto-apply

## File Structure

```
terraform-infrastructure/
├── config/
│   └── main.tf              # All infrastructure resources
├── .github/
│   └── workflows/
│       └── terraform.yml    # CI/CD pipeline
├── .gitignore
└── README.md
```

## Terraform Configuration

- **Terraform Version**: 1.12.2
- **AWS Provider**: ~> 3.0
- **Default Region**: us-west-2

## Common Issues

**State Lock Errors**: Another Terraform process is running. Wait for it to complete or manually unlock (use with caution).

**Format Check Failures in CI**: Run `terraform fmt` in the `config/` directory before committing.

**Binary vs String Types**: This project uses Binary (B) for UUIDs to match Golang's binary UUID format. Use String (S) only for non-UUID text fields.
