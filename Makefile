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

test:
	go test ./...

test-coverage:
	go test -cover ./...

test-integration:
	cd integration-tests && INTEGRATION_TESTS=true go test -v ./...

test-integration-short:
	cd integration-tests && INTEGRATION_TESTS=true go test -v -short ./...

clean-docker:
	docker-compose -p user-service-integration down -v --remove-orphans
