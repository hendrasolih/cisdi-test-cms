# Variables
APP_NAME=cms-cisdi
DOCKER_IMAGE=cms-cisdi
GO_VERSION=1.24
BINARY_NAME=main

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build test clean docker-build docker-run lint fmt dev

# Default target
help: ## Show this help message
	@echo "$(BLUE)Available commands:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

# Development
dev: ## Run application in development mode
	@echo "$(YELLOW)Starting development server...$(NC)"
	@air -c .air.toml || go run main.go

install-air: ## Install air for hot reload
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	@echo "$(GREEN)Air installed successfully$(NC)"

# Build
build: ## Build the application
	@echo "$(YELLOW)Building application...$(NC)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -extldflags "-static"' \
		-a -installsuffix cgo \
		-o $(BINARY_NAME) .
	@echo "$(GREEN)Build completed: $(BINARY_NAME)$(NC)"

build-local: ## Build for local OS
	@echo "$(YELLOW)Building for local OS...$(NC)"
	@go build -o $(BINARY_NAME) .
	@echo "$(GREEN)Local build completed$(NC)"

# Testing
test: ## Run tests
	@echo "$(YELLOW)Running tests...$(NC)"
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)Tests completed$(NC)"

test-coverage: test ## Run tests with coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

test-integration: ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@go test -v -run ^TestIntegrationSuite/TestIntegrationSuite$
	@echo "$(GREEN)Integration tests completed$(NC)"

# Code Quality
lint: ## Run linter
	@echo "$(YELLOW)Running linter...$(NC)"
	@golangci-lint run --timeout=5m
	@echo "$(GREEN)Linting completed$(NC)"

fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(NC)"
	@go fmt ./...
	@goimports -w .
	@echo "$(GREEN)Code formatted$(NC)"

install-tools: ## Install development tools
	@echo "$(YELLOW)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(GREEN)Development tools installed$(NC)"

# Dependencies
deps: ## Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@go mod download
	@go mod verify
	@echo "$(GREEN)Dependencies downloaded$(NC)"

deps-update: ## Update dependencies
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

# Docker
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE):latest .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE):latest$(NC)"

docker-run: ## Run Docker container
	@echo "$(YELLOW)Starting Docker container...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)Container started$(NC)"

docker-stop: ## Stop Docker container
	@echo "$(YELLOW)Stopping Docker container...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Container stopped$(NC)"

docker-logs: ## Show Docker logs
	@docker-compose logs -f app

# Database
db-migrate: ## Run database migrations
	@echo "$(YELLOW)Running database migrations...$(NC)"
	@go run main.go -migrate
	@echo "$(GREEN)Migrations completed$(NC)"

db-seed: ## Seed database with sample data
	@echo "$(YELLOW)Seeding database...$(NC)"
	@go run main.go -seed
	@echo "$(GREEN)Database seeded$(NC)"

# Cleanup
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@docker system prune -f
	@echo "$(GREEN)Cleanup completed$(NC)"

# CI/CD Simulation
ci: deps lint test build ## Simulate CI pipeline locally
	@echo "$(GREEN)✅ CI pipeline completed successfully$(NC)"

# Production
prod-build: ## Build for production
	@echo "$(YELLOW)Building for production...$(NC)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags='-w -s -X main.version=$(shell git describe --tags --always)' \
		-a -installsuffix cgo \
		-o $(BINARY_NAME) .
	@echo "$(GREEN)Production build completed$(NC)"