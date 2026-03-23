# Makefile for auto-customer-service

.PHONY: help test-unit test-integration test-gateway test-all run-mock-chat run-gateway build clean

# Colors for output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
NC     := \033[0m

help:
	@echo "$(GREEN)Available commands:$(NC)"
	@echo "  make test-unit         - Run unit tests"
	@echo "  make test-integration  - Run integration tests"
	@echo "  make test-gateway      - Run gateway tests"
	@echo "  make test-all          - Run all tests"
	@echo "  make run-mock-chat     - Run mock chat application"
	@echo "  make run-gateway       - Run gateway server"
	@echo "  make build             - Build all packages"
	@echo "  make clean             - Clean build artifacts"

# Unit tests (fast, no external dependencies)
test-unit:
	@echo "$(YELLOW)Running unit tests...$(NC)"
	go test -v ./tests/unit/... -timeout 30s

# Integration tests (may require external dependencies)
test-integration:
	@echo "$(YELLOW)Running integration tests...$(NC)"
	go test -v ./tests/integration/... -timeout 60s

# Gateway tests (WebSocket server tests)
test-gateway:
	@echo "$(YELLOW)Running gateway tests...$(NC)"
	go test -v ./internal/gateway/... -timeout 60s

# All tests
test-all: test-unit test-integration test-gateway

# Run mock chat application
run-mock-chat:
	@echo "$(YELLOW)Starting mock chat application...$(NC)"
	go run ./cmd/mock-chat/main.go

# Run gateway server
run-gateway:
	@echo "$(YELLOW)Starting gateway server...$(NC)"
	go run ./cmd/gateway/main.go

# Build all packages
build:
	@echo "$(YELLOW)Building all packages...$(NC)"
	go build ./...

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	go clean ./...
