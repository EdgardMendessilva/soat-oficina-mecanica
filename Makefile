.PHONY: build test lint cover

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

lint:
	golangci-lint run

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
