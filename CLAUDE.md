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

# Run the user-created consumer (NATS in production; `eventing user-created` is the Kafka twin)
go run cmd/cli/main.go eventing user-created-nats

# Publish outbox_events rows (follow-graph events) to NATS JetStream
go run cmd/cli/main.go relay outbox
```

### Local toolchain notes (macOS)
- Run `gqlgen` and `go mod tidy` with `GOTOOLCHAIN=go1.24.0` so the `go` line in go.mod is not bumped.
- Linking the binary needs the full Xcode toolchain on recent macOS: `DEVELOPER_DIR=/Applications/Xcode.app/Contents/Developer go build ./cmd/cli`. The Command Line Tools linker rejects the SDK stubs the cgo Kafka driver links against.
- Until go-outbox-lib is published, build with a `go.work` that includes `../go-outbox-lib`.

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./internal/jwt/...

# Unit tests that need Postgres (users, follows repositories) read DBHOST/DBPORT/DBUSER/DBPASSWORD/DBNAME/DBSSL
# and expect the migrations to be applied (`go run cmd/cli/main.go db migrate`).

# End-to-end tests (build tag `integration`): boot the real GraphQL handler over the real database
make test-integration

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
    - `users/`: profiles, the `follow_approval_required` flag
    - `follows/`: the follow graph (request/approval, visibility rule), its `user-follow` outbox events
  - `resolvers/`: GraphQL resolver implementations
- `handlers/`: long-running commands: the user-created consumer and the outbox relay
- `integration-tests/follows/`: end-to-end tests for the follow graph (in-process handler + Postgres)

### Configuration
The service uses environment-based configuration loaded from `config/config.{env}.json` files. Key configurations:
- **APP_ENV**: Controls which config file to load (dev/docker/prod)
- **Database**: Postgres connection configured via DB* environment variables
- **Kafka/NATS**: NATS JetStream is what production uses (`NATSURL`); the Kafka consumer is the legacy twin
- **Outbox**: `OUTBOX_*` tune the relay (see go-outbox-lib)
- **MinIO**: Object storage endpoint and credentials
- **JWT**: Token validity duration and key rotation settings

### GraphQL Federation
The service is configured for Apollo Federation with entity resolution support. Schema files are in `graph/*.graphqls` with generated code in `graph/generated/`.

### Follow graph
- `user_follows(follower_id, followee_id, status, created_at, accepted_at)`; status is `pending` or `accepted`.
- A user with `follow_approval_required` gets pending requests and hides their follower/following lists from non-followers; counts stay public.
- Every change writes a row to `outbox_events` in the same transaction (subject `user-follow`, types `follow_requested | follow_accepted | follow_declined | unfollowed | follower_removed`). `relay outbox` publishes them with `Nats-Msg-Id` = event id.
- `PublicUser` is a federation entity (`@key(fields: "id")`) so other subgraphs can return `PublicUser{id}`.
- `followerIDs(userID, after, limit)` is the keyset page notifications-service uses for fan-out.

### Important Notes
- The service includes a health check endpoint at `/readyz` and `/livez`
- GraphQL playground is available at root path `/` in development
- JWT signing keys rotate automatically based on configured duration (minimum 5 minutes)
- MinIO storage implementation has incorrect imports that need fixing (references image-sync instead of user service)