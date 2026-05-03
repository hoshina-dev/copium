.PHONY: install test test-unit test-watch cover lint mocks run build deps swagger format tidy

# --- dev loop ---

install:
	go mod download

deps tidy:
	go mod tidy

# Run every test with race detector. This is the canonical command.
test:
	go test -race -count=1 ./...

# Skip tests tagged "integration" (testcontainers, mailhog) - fast feedback loop.
test-unit:
	go test -race -count=1 -tags='!integration' ./...

# Continuous re-run on file changes. Requires gotestsum.
# Install once: go install gotest.tools/gotestsum@latest
test-watch:
	gotestsum --watch --format=testname -- -race -count=1 ./...

# Generate coverage profile and enforce per-package thresholds.
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 20

lint:
	golangci-lint run ./...

# Regenerate testify mocks from .mockery.yaml
mocks:
	go run github.com/vektra/mockery/v2@latest

# --- run ---

run:
	go run ./cmd/server

run-worker:
	go run ./cmd/worker

build:
	mkdir -p bin
	go build -o bin/copium ./cmd/server
	go build -o bin/copium-worker ./cmd/worker

# --- code gen / docs ---

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs --parseDependency --parseInternal
	@if grep -q "github_com" docs/swagger.json; then \
		echo ""; \
		echo "WARNING: Found 'github_com' in docs/swagger.json - some types are missing @name aliases."; \
		exit 1; \
	fi
	@echo "swagger: OK"

format:
	go fmt ./...
	gofmt -s -w .

.DEFAULT_GOAL = test
