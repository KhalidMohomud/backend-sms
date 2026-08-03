.PHONY: run build test fmt swagger docker-up docker-down

run:
	go run ./cmd/api

build:
	go build ./...

test:
	go test ./...

fmt:
	gofmt -w cmd internal

swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go

docker-up:
	docker compose up --build

docker-down:
	docker compose down
