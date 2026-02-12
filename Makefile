.PHONY: run build test test-coverage linter

build:
	  go build -o bin/adminapi .

test:
	  go test ./...

test-race:
	  go test -race ./...

test-coverage:
	  go test -v ./... -coverprofile=coverage.out
	  go tool cover -html=coverage.out -o coverage.html

linter:
	  golangci-lint run
