.PHONY: run build test test-race test-integration fmt swagger docker-up docker-down admin-create admin-archive-legacy-users admin-verify

run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	docker compose --profile test run --rm --build integration-test

fmt:
	gofmt -w cmd internal

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go

docker-up:
	docker compose up --build

docker-down:
	docker compose down

admin-create:
	@test -n "$(USERNAME)" || (echo "Usage: make admin-create USERNAME=superadmin" && exit 1)
	docker compose run --rm --entrypoint /admin api create-superadmin --username "$(USERNAME)"

admin-archive-legacy-users:
	docker compose run --rm --entrypoint /admin api archive-legacy-users --confirm-archive

admin-verify:
	docker compose run --rm --entrypoint /admin api verify-foundation
