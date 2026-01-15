.PHONY: test test-unit test-integration test-e2e test-db clean help

# Test commands
help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

test: ## Run all tests (unit + integration + e2e)
	@echo "Running all tests..."
	@go test -v ./... -coverprofile=coverage.out

test-unit: ## Run unit tests only (no database required)
	@echo "Running unit tests..."
	@go test -v -short ./...

test-integration: ## Run integration tests with database
	@echo "Running integration tests..."
	@go test -v -run Integration ./...

test-e2e: ## Run end-to-end tests
	@echo "Running end-to-end tests..."
	@go test -v -tags=e2e ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-db: ## Run tests that require a database connection
	@echo "Running database tests..."
	@go test -v -tags=database ./models/... ./handlers/...

clean: ## Clean test artifacts
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@go clean -testcache

build: ## Build the application
	@echo "Building application..."
	@go build -o bin/server ./cmd/server
	@go build -o bin/migrations ./cmd/migrations
	@go build -o bin/cleanup ./cmd/cleanup
	@go build -o bin/reminder ./cmd/reminder
	@go build -o bin/seed ./cmd/seed

lint: ## Run linters
	@echo "Running linters..."
	@go fmt ./...
	@go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

# Docker test environment
test-docker-up: ## Start test database in Docker
	@echo "Starting test database..."
	@docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for database to be ready..."
	@sleep 5

test-docker-down: ## Stop test database
	@echo "Stopping test database..."
	@docker compose -f docker-compose.test.yml down -v

test-docker: test-docker-up ## Run tests with Docker database
	@echo "Running tests with Docker database..."
	@export DATABASE_URL="postgresql://postgres:test123@localhost:5433/test_db?sslmode=disable" && \
	go test -v ./...
	@$(MAKE) test-docker-down
