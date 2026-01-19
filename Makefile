PROJECT_NAME := xenorchestra-go-sdk
GO := go
GOFLAGS :=
MOCKGEN_VERSION := 1.6.0

# Colors
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: all
all: mod test lint ## Run mod tidy, tests, and lint

.PHONY: help
help: ## Show this help message
	@echo "$(BLUE)Makefile for $(PROJECT_NAME)$(NC)"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@echo "  make $(GREEN)<target>$(NC)"
	@echo ""
	@echo "$(YELLOW)Targets:$(NC)"
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

.PHONY: mod
mod: ## Run go mod tidy
	@echo "$(BLUE)Running go mod tidy...$(NC)"
	$(GO) mod tidy

.PHONY: test
test: ## Run full test suite with coverage
	@echo "$(BLUE)Running tests...$(NC)"
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated at coverage.html$(NC)"

.PHONY: test-v1
test-v1: ## Run v1 client tests
	@echo "$(BLUE)Running v1 client tests...$(NC)"
	$(GO) test -race ./client/...

.PHONY: test-v2
test-v2: ## Run v2 client and pkg tests (excluding integration)
	@echo "$(BLUE)Running v2 client tests...$(NC)"
	$(GO) test -race $(shell go list ./v2/... | grep -v -E "/integration$$") ./pkg/... ./internal/...

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	$(GO) test -timeout 5m  ./v2/integration/... 

.PHONY: lint
lint: ## Run golangci-lint
	@echo "$(BLUE)Running linter...$(NC)"
	golangci-lint run

.PHONY: install-mockgen
install-mockgen: ## Install mockgen tool
	@echo "$(BLUE)Installing mockgen v$(MOCKGEN_VERSION)...$(NC)"
	$(GO) install github.com/golang/mock/mockgen@v$(MOCKGEN_VERSION)

.PHONY: mock
mock: install-mockgen ## Generate mocks
	@echo "$(BLUE)Generating mocks...$(NC)"
	$(GO) generate ./...

.PHONY: run-example-v1
run-example-v1: ## Run v1 example program
	@echo "$(BLUE)Running v1 example...$(NC)"
	$(GO) run ./examples/v1/user_demo.go

.PHONY: run-example-v2
run-example-v2: ## Run v2 example program
	@echo "$(BLUE)Running v2 example...$(NC)"
	$(GO) run ./examples/v2/user_demo.go

.PHONY: run-examples
run-examples: run-example-v1 run-example-v2 ## Run all examples
	@echo "$(GREEN)All examples executed successfully$(NC)"

.PHONY: clean
clean: ## Clean build artifacts and caches
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -f coverage.out coverage.html
	$(GO) clean -cache -testcache

.PHONY: vuln
vuln: ## Run vulnerability scan with govulncheck
	@echo "$(BLUE)Checking for vulnerabilities...$(NC)"
	govulncheck ./...

.PHONY: precommit
precommit: ## Run pre-commit hooks on all files
	@echo "$(BLUE)Running pre-commit checks...$(NC)"
	pre-commit run --all-files
	@echo "$(GREEN)Pre-commit checks passed!$(NC)"