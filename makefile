# Makefile
.PHONY: build run test docker-build docker-up docker-down clean help

# Variables
APP_NAME=cisdi-test-cms
DOCKER_COMPOSE=docker-compose
GO_FILES=$(shell find . -name "*.go" -type f)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@go build -o bin/$(APP_NAME) .

run: ## Run the application
	@echo "Running $(APP_NAME)..."
	@go run .

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./tests/...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v ./tests/... -tags=integration

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME) .

docker-up: ## Start services with Docker Compose
	@echo "Starting services..."
	@$(DOCKER_COMPOSE) up -d

docker-down: ## Stop services with Docker Compose
	@echo "Stopping services..."
	@$(DOCKER_COMPOSE) down

docker-logs: ## Show Docker logs
	@$(DOCKER_COMPOSE) logs -f

docker-restart: docker-down docker-up ## Restart Docker services

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

dev: ## Run in development mode with hot reload (requires air)
	@echo "Running in development mode..."
	@air

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest

setup-db: ## Setup database
	@echo "Setting up database..."
	@createdb cms_db || true
	@createdb cms_test_db || true

migrate: ## Run database migrations
	@echo "Running migrations..."
	@go run . --migrate

seed: ## Seed database with sample data
	@echo "Seeding database..."
	@go run . --seed