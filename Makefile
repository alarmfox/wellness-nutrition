.PHONY: test test-unit test-integration test-e2e test-db clean help dev dev-down

# Test commands
help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

test: test-docker-up ## Run all tests with Docker test services
	@echo "Running all tests..."
	@trap '$(MAKE) test-docker-down' EXIT; \
	DATABASE_URL="postgresql://postgres:test123@localhost:5433/test_db?sslmode=disable" \
	EMAIL_SERVER_HOST="localhost" \
	EMAIL_SERVER_PORT="1025" \
	EMAIL_SERVER_USER="" \
	EMAIL_SERVER_PASSWORD="" \
	EMAIL_SERVER_FROM="no-reply@example.test" \
	EMAIL_NOTIFY_ADDRESS="no-reply@example.test" \
	AUTH_URL="http://localhost:3000" \
	go test -v ./... -coverprofile=coverage.out

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
	@go build -o bin/nextjs2go ./cmd/nextjs2go

lint: ## Run linters
	@echo "Running linters..."
	@go fmt ./...
	@go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

dev: ## Build and start the development environment
	@docker compose -f docker-compose.dev.yml up --build

dev-down: ## Stop the development environment
	@docker compose -f docker-compose.dev.yml down

# Docker test environment
test-docker-up: ## Start test services in Docker
	@echo "Starting test services..."
	@docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for test services to be ready..."
	@sleep 5

test-docker-down: ## Stop test services
	@echo "Stopping test services..."
	@docker compose -f docker-compose.test.yml down -v

test-docker: test-docker-up ## Run tests with Docker database
	@echo "Running tests with Docker database..."
	@trap '$(MAKE) test-docker-down' EXIT; \
	DATABASE_URL="postgresql://postgres:test123@localhost:5433/test_db?sslmode=disable" \
	EMAIL_SERVER_HOST="localhost" \
	EMAIL_SERVER_PORT="1025" \
	EMAIL_SERVER_USER="" \
	EMAIL_SERVER_PASSWORD="" \
	EMAIL_SERVER_FROM="no-reply@example.test" \
	EMAIL_NOTIFY_ADDRESS="no-reply@example.test" \
	AUTH_URL="http://localhost:3000" \
	go test -v ./...
