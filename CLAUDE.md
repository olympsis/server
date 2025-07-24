# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Olympsis backend server written in Go. It's a REST API server that handles:
- User authentication via Firebase Auth
- Social features (clubs, organizations, posts, events)
- Location-based services (venues, map snapshots)
- Notification system
- Search functionality
- Report management

## Architecture

The codebase follows a modular domain-driven structure:

### Core Components
- **main.go** - Application entry point that initializes all services and APIs
- **server/models.go** - Defines the `ServerInterface` struct that gets injected into all API modules
- **database/database.go** - MongoDB connection and collection management
- **utils/** - Shared utilities, config management, and notification interface

### Domain Modules
Each domain has its own directory with `api.go` (HTTP handlers) and `service/` subdirectory (business logic):
- **auth/** - Authentication and user management
- **user/** - User profiles and operations
- **club/** - Club management, membership, applications
- **organization/** - Organization management and membership
- **post/** - Social posts, comments, reactions
- **event/** - Event management, participants, teams, waitlists
- **venue/** - Location/venue management
- **announcement/** - System announcements
- **report/** - Bug reports and content moderation
- **locales/** - Internationalization support
- **health/** - Health check endpoints

### Dependencies
- **MongoDB** - Primary database using the official Go driver
- **Firebase** - Authentication and possibly push notifications
- **Gorilla Mux** - HTTP routing
- **External packages**: `github.com/olympsis/models` and `github.com/olympsis/search`

## Development Commands

### Build & Run
```bash
make build          # Build the binary
make run            # Run with go run
make dep            # Install dependencies
```

### Testing & Quality
```bash
make test           # Run unit tests
make race           # Run with race detector
make msan           # Run with memory sanitizer
make lint           # Run golint
```

### Docker Development
```bash
make dev-up         # Start local docker-compose stack
make dev-down       # Stop local docker-compose stack
make docker-build   # Build development Docker image
make unsecure-server # Run HTTP server in Docker
make server         # Run HTTPS server with local certs
```

### Production
```bash
make artifact       # Build and push to GCP Artifact Registry
make prod-up        # Start production docker-compose
make update-service # Update Linux service (production deployment)
```

## Key Configuration

The server uses environment-based configuration through `utils.GetServerConfig()`. Key config areas:
- Database connection (MongoDB)
- Firebase credentials path
- Server port and TLS settings
- Notification service URL

## API Structure

Each domain module follows this pattern:
1. `NewXXXAPI(serverInterface)` - Constructor that takes the shared server interface
2. `Ready(client)` - Initialization method called in main.go
3. HTTP handlers registered with the mux router
4. Business logic in the `service/` subdirectory

The `ServerInterface` provides shared access to:
- Logger (logrus)
- Database collections
- Firebase Auth client
- Search service
- Notification interface
- HTTP router

## Testing

Run tests with `make test`. The project uses Go's standard testing framework.

## Important Files

- **tools/compose.dev.yaml** - Local development Docker Compose setup
- **Dockerfile** - Production container build
- **Makefile** - All build, test, and deployment commands
- **files/** - Contains credentials and certificate files (not in version control)