# Terraform Infrastructure 

This repository contains the Terraform configuration for managing AWS infrastructure to support personal game development projects. The infrastructure is designed to provide scalable, cost-effective cloud resources for game deployment and data management.

## 🎮 Project Overview

This Terraform configuration sets up AWS infrastructure to support "Amazing Adventure" - a game development project. The infrastructure includes:

- **DynamoDB Tables**: For user authentication, game session data, and refresh token management
- **State Management**: Remote state storage with S3 backend and DynamoDB locking
- **CI/CD Pipeline**: Automated deployment via GitHub Actions
- **Secrets Management**: Doppler for local development credential injection

## 🏗️ Infrastructure Architecture

### Current Resources

#### DynamoDB Tables

**users**
- **Purpose**: User account information
- **Primary Key**: `user_id` (Binary - UUID)
- **GSI**: `email-index` on `email` field (String)
- **Billing Mode**: Pay-per-request

**amazing-adventure-data**
- **Purpose**: Game session data
- **Primary Key**: `session_id` (Binary - UUID)
- **GSI**: `user-sessions-index` on `user_id` field (Binary - UUID)
- **Billing Mode**: Pay-per-request

**refresh-tokens**
- **Purpose**: OAuth refresh token storage
- **Primary Key**: `token` (String)
- **GSI**: `user-tokens-index` on `user_id` field (Binary - UUID)
- **TTL**: Enabled on `expires_at` attribute for automatic expiration
- **Billing Mode**: Pay-per-request

**Note**: UUID fields use Binary (B) type to align with Golang's binary UUID storage format.

### State Management
- **Backend**: S3 bucket (`roberts-personal-tf-bucket`)
- **State File**: `03-basics/web-app/terraform.tfstate`
- **Region**: `us-west-2`
- **Locking**: DynamoDB table (`terraform-state-locking`)
- **Encryption**: Enabled for security

## 🔄 CI/CD Pipeline

### GitHub Actions Workflow

The project includes an automated CI/CD pipeline (`terraform.yml`) that:

1. **Triggers on**:
   - Push to `main` branch
   - Release creation

2. **Workflow Steps**:
   - **Format Check**: Ensures Terraform code follows style guidelines
   - **Initialization**: Sets up Terraform backend and providers
   - **Planning**: Creates execution plan (on pull requests)
   - **Comment**: Posts plan results to PR comments
   - **Apply**: Automatically applies changes (on main branch)

### Environment Variables

The following secrets must be configured in your GitHub repository:

- `AWS_ACCESS_KEY_ID`: Your AWS access key
- `AWS_SECRET_ACCESS_KEY`: Your AWS secret key
- `GITHUB_TOKEN`: Automatically provided by GitHub

## 📁 Project Structure

```
├── config/
│   ├── main.tf                 # Main Terraform configuration
├── .github/
│   └── workflows/
│       └── terraform.yml       # CI/CD pipeline configuration
├── .gitignore                  # Git ignore rules
└── README.md                   # This file
```

## 🔧 Configuration Details

### Terraform Configuration

- **Provider**: AWS (~> 3.0)
- **Region**: us-west-2
- **Terraform Version**: 1.12.2

### DynamoDB Table Schemas

```hcl
Table: users
├── Primary Key: user_id (Binary - UUID)
├── Attributes:
│   ├── user_id (Binary, Hash Key)
│   └── email (String)
└── Global Secondary Index:
    └── email-index
        ├── Hash Key: email
        └── Projection: ALL

Table: amazing-adventure-data
├── Primary Key: session_id (Binary - UUID)
├── Attributes:
│   ├── session_id (Binary, Hash Key)
│   └── user_id (Binary - UUID)
└── Global Secondary Index:
    └── user-sessions-index
        ├── Hash Key: user_id
        └── Projection: ALL

Table: refresh-tokens
├── Primary Key: token (String)
├── Attributes:
│   ├── token (String, Hash Key)
│   ├── user_id (Binary - UUID)
│   └── expires_at (Number - Epoch seconds)
├── Global Secondary Index:
│   └── user-tokens-index
│       ├── Hash Key: user_id
│       └── Projection: ALL
└── TTL: expires_at (automatic deletion)
```

## 💰 Cost Optimization

- **DynamoDB**: Uses pay-per-request billing to minimize costs for variable workloads
- **S3 Backend**: Minimal storage costs for state files
- **DynamoDB Locking**: Low-cost table for state locking

## 🔒 Security Considerations

- **State Encryption**: Enabled on S3 backend
- **Access Control**: Uses AWS IAM for authentication
- **Secrets Management**:
  - GitHub Actions: AWS credentials stored as GitHub secrets
  - Local Development: Doppler for credential injection
- **Token Expiration**: Automatic cleanup via DynamoDB TTL on refresh tokens
- **Network Security**: Resources deployed in private subnets where applicable

## 🛠️ Development Workflow

### Setup

**One-time setup** for local development:
```bash
# Install Doppler CLI (if not already installed)
# On Arch Linux:
yay -S doppler

# Configure Doppler for this project
cd /path/to/terraform-infrastructure
doppler setup
```

### Making Changes

1. **Create a feature branch**:
   ```bash
   git checkout -b feature/new-resource
   ```

2. **Make your changes** to `config/main.tf`

3. **Test locally** (with Doppler injecting AWS credentials):
   ```bash
   cd config
   doppler run -- terraform init
   doppler run -- terraform plan
   ```

4. **Commit and push**:
   ```bash
   git add .
   git commit -m "Add new resource"
   git push origin feature/new-resource
   ```

5. **Create a Pull Request** to trigger the CI/CD pipeline

### Best Practices

- Always prefix terraform commands with `doppler run --` for local development
- Always run `terraform plan` before applying changes
- Use meaningful commit messages
- Review the Terraform plan output in PR comments
- Test changes in a development environment first

## 🚨 Troubleshooting

### Common Issues

1. **State Lock Errors**: Check if another process is running Terraform
2. **Permission Errors**: Verify AWS credentials and IAM permissions
3. **Provider Version Conflicts**: Run `terraform init -upgrade`

### Useful Commands

All commands require Doppler for credential injection:

```bash
cd config

# Check Terraform version
doppler run -- terraform version

# Validate configuration
doppler run -- terraform validate

# Format code
doppler run -- terraform fmt

# Show current state
doppler run -- terraform show

# List resources
doppler run -- terraform state list
```

## 📈 Future Enhancements

Potential additions to consider:

- **VPC and Networking**: Custom VPC with subnets and security groups
- **Application Load Balancer**: For game server load balancing
- **Auto Scaling Groups**: For dynamic scaling based on demand
- **CloudWatch Monitoring**: For performance and cost monitoring
- **Route 53**: For custom domain management
- **CloudFront**: For content delivery optimization

## 📞 Support

For issues or questions:

1. Check the troubleshooting section above
2. Review Terraform documentation
3. Check AWS service documentation
4. Create an issue in this repository

## 📄 License

This project is for personal use. Please ensure compliance with AWS terms of service and your organization's policies.

---

**Last Updated**: $(date)
**Terraform Version**: 1.12.2
**AWS Provider Version**: ~> 3.0
