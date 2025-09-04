# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based user authentication and management service built with GraphQL (using gqlgen). The service handles JWT-based authentication, user management, and integrates with Kafka for event streaming and MinIO for object storage.

## Common Development Commands

### Code Generation
```bash
# Generate GraphQL code from schema
make gql
# Or directly:
go run github.com/99designs/gqlgen generate

# Generate mocks for testing
make mocks
```

### Database Operations
```bash
# Run database migrations
make migrate
# Or using CLI:
go run cmd/cli/main.go db migrate

# Create a new migration
make create-migration name=<migration_name>
```

### Running the Service
```bash
# Start the GraphQL server (runs on port configured in config, default 3002)
go run cmd/cli/main.go server

# Start with live reloading (Air is configured)
air

# Run the Kafka consumer for user events
go run cmd/cli/main.go user-created-event
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/jwt/...

# Run image service tests with verbose output
go test ./internal/services/image -v

# Run tests with coverage report
go test ./internal/services/image -cover -v
```

### Code Quality
```bash
# Run golangci-lint (configured via .golangci.json)
golangci-lint run

# Run with auto-fix where possible
golangci-lint run --fix
```

## Architecture Overview

### Service Structure
- **GraphQL API**: Federation-enabled GraphQL service exposing user authentication and management operations
- **JWT Authentication**: Rotating signing keys with configurable validity periods
- **Event-Driven**: Kafka integration for publishing/consuming user events
- **Storage**: MinIO integration for object storage needs

### Key Directories
- `cmd/cli/`: CLI entry point with commands for server, migrations, and event processing
- `graph/`: GraphQL schema definitions and resolvers
- `internal/`: Core business logic organized by domain:
  - `jwt/`: JWT token generation and validation
  - `keypair/`: Rotating signing key management
  - `db/`: Database connection and utilities
  - `migrations/`: Database migration scripts
  - `storage/`: Storage abstraction with MinIO implementation
  - `services/`: Business logic services
  - `resolvers/`: GraphQL resolver implementations

### Configuration
The service uses environment-based configuration loaded from `config/config.{env}.json` files. Key configurations:
- **APP_ENV**: Controls which config file to load (dev/docker/prod)
- **Database**: MySQL/MariaDB connection configured via DB* environment variables
- **Kafka**: Bootstrap servers and consumer group configuration
- **MinIO**: Object storage endpoint and credentials
- **JWT**: Token validity duration and key rotation settings

### GraphQL Federation
The service is configured for Apollo Federation with entity resolution support. Schema files are in `graph/*.graphqls` with generated code in `graph/generated/`.

### Important Notes
- The service includes a health check endpoint at `/readyz` and `/livez`
- GraphQL playground is available at root path `/` in development
- JWT signing keys rotate automatically based on configured duration (minimum 5 minutes)
- MinIO storage implementation has incorrect imports that need fixing (references image-sync instead of user service)