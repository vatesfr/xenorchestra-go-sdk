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
all: mod test lint

.PHONY: help
help:
	@echo "$(BLUE)Makefile for $(PROJECT_NAME)$(NC)"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@echo "  make $(GREEN)<target>$(NC)"
	@echo ""
	@echo "$(YELLOW)Targets:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

.PHONY: mod
mod: 
	@echo "$(BLUE)Running go mod tidy...$(NC)"
	$(GO) mod tidy

.PHONY: test
test:
	@echo "$(BLUE)Running tests...$(NC)"
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated at coverage.html$(NC)"

.PHONY: test-v1
test-v1: 
	@echo "$(BLUE)Running v1 client tests...$(NC)"
	$(GO) test -race ./client/...

.PHONY: test-v2
test-v2: 
	@echo "$(BLUE)Running v2 client tests...$(NC)"
	$(GO) test -race ./v2/... ./pkg/... ./internal/...

.PHONY: lint
lint: 
	@echo "$(BLUE)Running linter...$(NC)"
	golangci-lint run

.PHONY: install-mockgen
install-mockgen: 
	@echo "$(BLUE)Installing mockgen v$(MOCKGEN_VERSION)...$(NC)"
	$(GO) install github.com/golang/mock/mockgen@v$(MOCKGEN_VERSION)

.PHONY: mock
mock: install-mockgen 
	@echo "$(BLUE)Generating mocks...$(NC)"
	$(GO) generate ./...

.PHONY: run-example-v1
run-example-v1: 
	@echo "$(BLUE)Running v1 example...$(NC)"
	$(GO) run ./examples/v1/user_demo.go

.PHONY: run-example-v2
run-example-v2: 
	@echo "$(BLUE)Running v2 example...$(NC)"
	$(GO) run ./examples/v2/user_demo.go

.PHONY: run-examples
run-examples: run-example-v1 run-example-v2 
	@echo "$(GREEN)All examples executed successfully$(NC)"

.PHONY: run-integration-tests
run-integration-tests:
	@echo "$(BLUE)Running integration tests...$(NC)"
	@mkdir -p v2/integration/logs
	$(GO) test -v ./v2/integration 2>&1 | tee v2/integration/integration_tests.log

.PHONY: clean
clean: 
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -f coverage.out coverage.html
	$(GO) clean -cache -testcache

.PHONY: vuln
vuln: 
	@echo "$(BLUE)Checking for vulnerabilities...$(NC)"
	govulncheck ./...

.PHONY: precommit
precommit: 
	@echo "$(BLUE)Running pre-commit checks...$(NC)"
	pre-commit run --all-files
	@echo "$(GREEN)Pre-commit checks passed!$(NC)"