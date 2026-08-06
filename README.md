# user-service

Accounts and authentication for weeb.vip: registration, sign-in, JWT issuing
and the user records behind them.

A Go GraphQL service built with [gqlgen](https://gqlgen.com), backed by MySQL.
It publishes user events to Kafka so the rest of the system can react to a new
account without this service knowing who is listening, and uses MinIO for
object storage.

The tokens it issues are what `gateway-proxy` validates on the way in.

## Running it

Requires Go and a MySQL instance. `.db_init.sql` sets up the schema a local
database needs.

```sh
make migrate                                  # bring the database up to date
go run cmd/cli/main.go server                 # GraphQL server, port 3002 by default
air                                           # the same, with live reload
go run cmd/cli/main.go user-created-event     # the Kafka consumer
```

`docker-compose.local.yaml` brings up the dependencies for local work.

## Schema and generated code

The GraphQL schema lives under `graph/`, and the resolvers around it are
generated:

```sh
make gql      # regenerate from the schema
make mocks    # regenerate the test mocks
```

Both outputs are committed, so a schema change means regenerating and
committing the result with it.

## Migrations

```sh
make create-migration name=add_something
make migrate
```

Migration files live in `internal/migrations/scripts`, run by
[golang-migrate](https://github.com/golang-migrate/migrate).

## Configuration

`config/config.go` defines the shape; `config/config.dev.json` and
`config/config.docker.json` are the checked-in variants, and environment
variables override them.

## Checks

```sh
go test ./...
golangci-lint run     # config in .golangci.json
```

`.pre-commit-config.yaml` wires the same checks into a pre-commit hook.
