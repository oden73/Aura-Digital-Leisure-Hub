APP_NAME=core-api

.PHONY: run build test fmt

run:
	go run ./cmd/api

build:
	go build -o bin/$(APP_NAME) ./cmd/api

test:
	go test ./...

fmt:
	go fmt ./...
