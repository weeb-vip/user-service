generate: gql mocks
gql:
	go run github.com/99designs/gqlgen generate
create-migration:
	go run github.com/golang-migrate/migrate/v4/cmd/migrate create -ext sql -dir internal/migrations/scripts $(name)

migrate:
	go run cmd/cli/main.go db migrate

mocks:
	go get github.com/golang/mock/mockgen/model
	go install github.com/golang/mock/mockgen@v1.6.0
	mockgen -destination=./mocks/mock_users.go -package=mocks github.com/weeb-vip/user-service/internal/services/users User
	mockgen -source=./internal/services/follows/interface.go -destination=./mocks/mock_follows.go -package=mocks
	mockgen -source=./internal/services/follows/repositories/follows.go -destination=./mocks/mock_follows_repository.go -package=mocks

test:
	go test ./...

test-coverage:
	go test -cover ./...

# The integration packages carry the `integration` build tag, so without
# -tags they compile to nothing and pass vacuously.
test-integration:
	cd integration-tests && INTEGRATION_TESTS=true go test -tags integration -count=1 -v ./...

test-integration-short:
	cd integration-tests && INTEGRATION_TESTS=true go test -tags integration -count=1 -v -short ./...

clean-docker:
	docker-compose -p user-service-integration down -v --remove-orphans
