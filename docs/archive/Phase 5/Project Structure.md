Terraform is managed in a separate repo rrochlin/terraform-infrastructure with automatic build pipelines accessible in the terminal via `gh`. The signed in user is the owner of this repo. PR's to this repo have automated building — refer to #TerraformRepoIntegration for more.

## Repository Layout (target state)

```
An-Amazing-Adventure/
├── client/                  # React SPA (unchanged directory name)
├── server/                  # Go Lambda handlers (replaces the monolithic server)
│   ├── cmd/
│   │   ├── ws-connect/      # Lambda: WebSocket $connect
│   │   ├── ws-disconnect/   # Lambda: WebSocket $disconnect
│   │   ├── ws-chat/         # Lambda: WebSocket chat route
│   │   ├── ws-game-action/  # Lambda: WebSocket game_action route
│   │   ├── http-games/      # Lambda: REST /api/games*
│   │   ├── http-users/      # Lambda: REST /api/users
│   │   └── world-gen/       # Lambda: async world generation
│   ├── internal/
│   │   ├── auth/            # Cognito JWT validation helpers
│   │   ├── game/            # Game engine (area, character, item, container, equipment)
│   │   ├── ai/              # Bedrock client + tool dispatch (replaces mcp.go)
│   │   └── db/              # DynamoDB actions
│   ├── go.mod
│   └── go.sum
│   └── infra/               # Terraform config (git subtree → terraform-infrastructure repo)
│       ├── main.tf
│       ├── variables.tf
│       ├── outputs.tf
│       └── modules/
│           ├── api-gateway/ # HTTP and WebSocket API GW resources
│           ├── cognito/     # User Pool and app client
│           ├── dynamodb/    # Table definitions
│           ├── lambdas/     # Lambda function resources + IAM roles
│           ├── s3/          # App bucket + OAC
│           └── cloudfront/  # Distribution, behaviors, custom errors
└── .github/
    └── workflows/
        ├── deploy-client.yml   # Build Vite → S3 sync + CF invalidation
        └── deploy-server.yml   # Build Go Lambdas → zip → upload to S3 → update Lambda code
```

## Terraform Subtree (git subtree workflow)
The `server/infra/` folder is a git subtree linked to a branch on the `rrochlin/terraform-infrastructure` repo.

```bash
# Initial setup (run once):
git subtree add --prefix server/infra \
  https://github.com/rrochlin/terraform-infrastructure.git \
  amazing-adventure --squash

# Push infra changes back to the infra repo:
git subtree push --prefix server/infra \
  https://github.com/rrochlin/terraform-infrastructure.git \
  amazing-adventure

# Then open a PR from the terminal:
gh pr create --repo rrochlin/terraform-infrastructure \
  --head amazing-adventure --base main \
  --title "..." --body "..."
```

## CI/CD Pipeline

### Client (deploy-client.yml)
Triggered on push to `main` (path filter: `client/**`).
Steps:
1. Checkout
2. Setup Node + pnpm
3. `pnpm install --frozen-lockfile`
4. `pnpm test --run` (fail fast)
5. `pnpm build` (Vite outputs to `client/dist/`)
6. Configure AWS credentials (OIDC — no long-lived keys; use `aws-actions/configure-aws-credentials` with an IAM role that has S3+CloudFront permissions)
7. `aws s3 sync client/dist/ s3://$BUCKET_NAME/ --delete`
8. `aws cloudfront create-invalidation --distribution-id $CF_DIST_ID --paths "/*"`

### Server (deploy-server.yml)
Triggered on push to `main` (path filter: `server/**`, excludes `server/infra/**`).
Steps:
1. Checkout
2. Setup Go 1.24
3. For each Lambda in `server/cmd/*/`:
   - `GOARCH=arm64 GOOS=linux go build -tags lambda.norpc -o bootstrap ./cmd/<name>/`
   - `zip <name>.zip bootstrap`
4. Configure AWS credentials (OIDC)
5. For each zip: `aws lambda update-function-code --function-name <name> --zip-file fileb://<name>.zip`
6. Wait for update to complete before next Lambda (sequential to avoid throttle)

### Docker is removed entirely
The existing `client/dockerfile`, `server/dockerfile`, `client/nginx.conf.template`, and the Docker Hub push steps are all retired. No Docker images are built or pushed.

## GitHub Actions Secrets Needed
- `AWS_DEPLOY_ROLE_ARN` — IAM role ARN for OIDC (replaces DOCKERHUB_TOKEN, GCP_SA_KEY, GCP_REGION, GCP_PROJECT_ID)
- `AWS_REGION` — deployment region (e.g. `us-west-2`)
- `S3_BUCKET_NAME` — client assets bucket
- `CF_DISTRIBUTION_ID` — CloudFront distribution ID
- Lambda function names can be hardcoded in the workflow or output from Terraform into SSM and fetched in CI.

## Environment Variable Migration
| Old (Doppler/GCP) | New (Lambda env var via Terraform) |
|---|---|
| `GCP_KEY` | retired (Bedrock uses IAM role) |
| `HOST_URL` / `PORT` | retired (Lambda has no listener) |
| `SECRET` (JWT signing) | retired (Cognito) |
| `AWS_USERS_TABLE` | retired (Cognito) |
| `AWS_SESSION_TABLE` | `SESSIONS_TABLE` |
| `AWS_R_TOKENS_TABLE` | retired (Cognito handles refresh) |
| — | `CONNECTIONS_TABLE` (new) |
| — | `USER_POOL_ID` |
| — | `WEBSOCKET_API_ENDPOINT` (for Management API push-back) |
| — | `BEDROCK_REGION` |

Doppler remains for local development only. For local Lambda testing, use AWS SAM CLI or a thin HTTP wrapper around the Lambda handler.
