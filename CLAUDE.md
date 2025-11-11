# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

An AI-powered text adventure game using Google's Gemini AI as a Dungeon Master. The game features dynamic world generation, real-time narrative responses, and persistent game state stored in DynamoDB.

**Current Status:** Early alpha build - UI and game components are subject to significant changes.

## Tech Stack

### Backend (Go)
- **Language:** Go 1.24
- **AI:** Google Gemini API (gemini-2.5-flash model)
- **Database:** AWS DynamoDB
- **Auth:** JWT tokens with refresh token rotation
- **Framework:** Standard library with `net/http`

### Frontend (React)
- **Framework:** React 19 + TypeScript
- **Build Tool:** Vite
- **Routing:** TanStack Router (file-based routing)
- **UI Library:** Material UI (MUI)
- **HTTP Client:** Axios
- **Testing:** Vitest + Testing Library

### Infrastructure
- **Secrets Management:** Doppler CLI
- **Deployment:** Docker containers on GCP
- **CORS:** Configured via rs/cors package

## Development Setup

### Prerequisites
1. Go 1.24+ installed
2. Node.js with pnpm installed
3. Doppler CLI installed and authenticated
4. Access to the Doppler project for secrets

### Initial Setup

```bash
# Authenticate with Doppler
doppler login

# Setup server secrets
cd server
doppler setup

# Setup client secrets
cd ../client
doppler setup
```

### Running Locally

**Terminal 1 - Start Backend:**
```bash
cd server
doppler run -- go run .
```

**Terminal 2 - Start Frontend:**
```bash
cd client
doppler run -- pnpm dev
```

The client runs on `http://localhost:5173` and proxies API requests to the Go server.

## Common Commands

### Client (Frontend)
```bash
cd client

# Development
pnpm dev              # Start dev server on port 5173
pnpm start            # Alias for dev

# Building
pnpm build            # Build for production (Vite + tsc)
pnpm serve            # Preview production build

# Testing
pnpm test             # Run tests with Vitest
```

### Server (Backend)
```bash
cd server

# Development
doppler run -- go run .           # Run with Doppler secrets
go run .                          # Run directly (requires env vars)

# Dependencies
go mod download                   # Download dependencies
go mod tidy                       # Clean up dependencies
go mod vendor                     # Vendor dependencies

# Building
go build -o bin/server .          # Build binary
```

## Architecture

### Backend Structure (`/server`)

**Core Game Components:**
- `main.go` - HTTP server setup, routing, and configuration
- `game.go` - Core game state management and logic
- `mcp.go` - Gemini AI client and tool execution system
- `world_gen.go` - AI-driven world generation logic

**Domain Models:**
- `area.go` - Room management and connections
- `character.go` - Player and NPC system
- `item.go` - Item properties and management

**API Layer:**
- `handlers.go` - Game-related endpoints
- `handlers_chat.go` - AI chat interaction endpoint
- `handlers_login.go` - Authentication endpoints
- `handlers_refresh.go` - Token refresh logic
- `handlers_users.go` - User management endpoints

**Infrastructure:**
- `dynamodb_actions.go` - Database operations
- `errors.go` - Error handling utilities
- `logger.go` - HTTP request logging middleware

**Key API Endpoints:**
- `POST /api/games/{uuid}` - Start new game
- `DELETE /api/games/{uuid}` - Delete game
- `GET /api/games` - List user's games
- `POST /api/chat/{uuid}` - Chat with AI DM
- `GET /api/describe/{uuid}` - Get current room description
- `GET /api/worldready/{uuid}` - Check if world generation complete
- `POST /api/login` - User authentication
- `POST /api/refresh` - Refresh access token
- `POST /api/revoke` - Revoke refresh token
- `POST /api/users` - Create new user
- `PUT /api/users` - Update user profile

### Frontend Structure (`/client/src`)

**Routing** (`/routes`):
- Uses TanStack Router with file-based routing
- `__root.tsx` - Root layout
- `index.tsx` - Home page
- `game-{$sessionUUID}.tsx` - Game session page (dynamic route)
- `login.tsx`, `signup.tsx` - Authentication pages
- `profile.tsx` - User profile page

**Services** (`/services`):
- `api.service.ts` - Base Axios configuration
- `api.game.ts` - Game API calls
- `api.users.ts` - User/auth API calls

**Components** (`/components`):
- `Chat.tsx` - Chat interface with AI DM
- `RoomMap.tsx` - Visual room/map display
- `GameInfo.tsx` - Game state information
- `Header.tsx` - App header/navigation
- `AccountPanel.tsx` - User account menu
- `WaitForWorld.ts` - World generation loading state

**Types** (`/types`):
- `api.types.ts` - TypeScript definitions for API responses

**Import Alias:**
- `@/*` maps to `./src/*` for cleaner imports

## Environment Variables (via Doppler)

**Server:**
- `GCP_KEY` - Google Cloud API key for Gemini
- `HOST_URL` - Server host (e.g., localhost)
- `PORT` - Server port (e.g., 8080)
- `SECRET` - JWT signing secret
- `AWS_USERS_TABLE` - DynamoDB users table name
- `AWS_SESSION_TABLE` - DynamoDB sessions table name
- `AWS_R_TOKENS_TABLE` - DynamoDB refresh tokens table name

**Client:**
- Configuration varies based on deployment environment

## Game Architecture

### AI Integration
The game uses Google's Gemini AI as a Dungeon Master with a custom tool system:
- System instructions define the AI's role and capabilities
- The AI can execute game actions via function calling
- Tools are defined in `mcp.go` and executed in `ExecuteTool()`
- World generation happens asynchronously when starting a new game

### Data Flow
1. Client sends chat message to `/api/chat/{uuid}`
2. Server retrieves game state from DynamoDB
3. Request sent to Gemini with game context and available tools
4. AI responds with narrative and/or tool calls
5. Server executes tools, updates game state
6. Response sent back to client
7. Updated state persisted to DynamoDB

### Authentication
- JWT access tokens (short-lived)
- Refresh token rotation pattern
- Tokens stored in DynamoDB with expiration
- Password hashing with bcrypt

## Deployment

The project uses GitHub Actions for CI/CD:
- Workflow: `.github/workflows/build-push-deploy.yml`
- Builds Docker images for server
- Deploys to Google Cloud Platform
- Triggered on pushes to main branch

## Testing

**Client Tests:**
- Framework: Vitest with jsdom
- Library: @testing-library/react
- Run: `pnpm test` in client directory

**Server Tests:**
- Go's standard testing package (if tests exist)
- Run: `go test ./...` in server directory

## Notes for Development

- The game is in early alpha - expect significant UI/UX changes
- Game mechanics and components will evolve
- Focus on the AI integration and core game loop as the stable foundation
- Authentication and session management are production-ready
- World generation can take time - use `/api/worldready/{uuid}` to poll status
