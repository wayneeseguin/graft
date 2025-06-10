# graft Makefile
SHELL := /bin/bash
.DEFAULT_GOAL := help

# Variables
BINARY_NAME := graft
BUILD_DIR := ./cmd/graft
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html
INSTALL_PATH ?= /usr/local/bin

# Phony targets
.PHONY: help build build-examples package clean install
.PHONY: test test-unit test-clean test-verbose test-race test-integration test-all test-cli
.PHONY: integration integration-nats integration-vault integration-aws integration-debug
.PHONY: test-nats-recovery test-nats-monitor test-nats-load
.PHONY: debug-vault test-vault-auth test-vault-kv test-vault-load
.PHONY: fmt vet security gosec trivy coverage coverage-text
.PHONY: bench bench-full
.PHONY: deps check ci install-tools
.PHONY: test-color
.PHONY: hooks hooks-install hooks-uninstall hooks-check pre-commit pre-push

##@ General

help: ## Show this help message
	@echo -e "\033[33mgraft\033[0m - Available \`make T\` (T)argets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[32m\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "Version: $(VERSION)"

##@ Build & Package

build: ## Build the graft binary
	@echo "Building graft..."
	@go build $(LDFLAGS) $(BUILD_DIR)

build-examples: ## Build all example applications
	@echo "Building examples..."
	@for dir in examples/*/; do \
		if [ -f "$$dir/main.go" ]; then \
			echo "Building $$dir..."; \
			(cd "$$dir" && go build); \
		fi \
	done

package: clean build ## Clean, build, and prepare for release
	@echo "Packaging graft..."
	@mkdir -p dist
	@cp $(BINARY_NAME) dist/
	@echo "Package ready in dist/"

install: build ## Install graft binary to INSTALL_PATH (default: /usr/local/bin)
	@echo "Installing graft to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "graft installed to $(INSTALL_PATH)/$(BINARY_NAME)"

clean: ## Remove build artifacts and binaries
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@rm -rf $(COVERAGE_DIR)
	@find examples -name 'main' -type f -delete

##@ Testing

test: test-unit ## Run all unit tests (alias for test-unit)

test-unit: ## Run Go unit tests with coverage (excludes integration tests)
	@echo "Running unit tests..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -v -short -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...

test-clean: ## Run tests without linker warnings (CGO disabled)
	@echo "Running tests with CGO disabled..."
	@CGO_ENABLED=0 go test -short ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	@go test -v -short ./...

test-race: ## Run tests with Go race detector enabled
	@echo "Running tests with race detector..."
	@go test -race -short ./...

test-integration: build ## Run all integration tests (Docker + Go with build tags)
	@echo "Running Go integration tests with build tag..."
	@go test -v -tags=integration ./pkg/graft/operators/...
	@$(MAKE) integration

test-all: vet test-unit test-race test-cli test-color test-integration ## Run all test targets

test-cli: build ## Run CLI integration tests
	@echo "Running CLI tests..."
	@./assets/cli_tests

test-color: build ## Test color output functionality
	@echo "Testing color output..."
	@./assets/color_tester

##@ Integration Testing

integration: build ## Run all integration tests with Docker
	@echo "Running all integration tests..."
	@scripts/integration

integration-nats: build ## Run NATS integration tests only
	@echo "Running NATS integration tests..."
	@scripts/integration nats

integration-vault: build ## Run Vault integration tests only
	@echo "Running Vault integration tests..."
	@scripts/integration vault

integration-aws: build ## Run AWS integration tests only
	@echo "Running AWS integration tests..."
	@scripts/integration aws

integration-debug: build ## Run integration tests (keep containers)
	@echo "Running integration tests in debug mode..."
	@DEBUG=1 scripts/integration

##@ NATS Testing

test-nats-recovery: build ## Run NATS error recovery tests
	@echo "Running NATS error recovery tests..."
	@go test -v ./pkg/graft/operators -run TestNATS.*Recovery

test-nats-monitor: build ## Run NATS monitoring tests
	@echo "Running NATS monitoring tests..."
	@go test -v ./pkg/graft/operators -run TestNATS.*Monitor

test-nats-load: build ## Run NATS load/performance tests
	@echo "Running NATS load tests..."
	@go test -v -bench=BenchmarkNATS ./pkg/graft/operators

##@ Vault Testing

debug-vault: ## Start Vault debug environment
	@echo "Starting Vault debug environment..."
	@docker run -d --name graft-vault-debug \
		-p 8200:8200 \
		-e VAULT_DEV_ROOT_TOKEN_ID=test-token \
		vault:latest
	@echo "Vault available at http://localhost:8200 with token: test-token"
	@echo "Run 'docker stop graft-vault-debug && docker rm graft-vault-debug' to cleanup"

test-vault-auth: build ## Run Vault authentication tests
	@echo "Running Vault authentication tests..."
	@go test -v ./pkg/graft/operators -run TestVault.*Auth

test-vault-kv: build ## Run Vault KV store tests
	@echo "Running Vault KV store tests..."
	@go test -v ./pkg/graft/operators -run TestVault.*KV

test-vault-load: build ## Run Vault load/performance tests
	@echo "Running Vault load tests..."
	@go test -v -bench=BenchmarkVault ./pkg/graft/operators

##@ Code Quality

fmt: ## Format all Go source files using gofmt
	@echo "Formatting code..."
	@gofmt -w $(GO_FILES)

vet: ## Run go vet for static analysis
	@echo "Running go vet..."
	@go vet ./...

security: gosec trivy ## Run all security checks (gosec + trivy)

gosec: ## Run gosec security scanner
	@echo "Running gosec..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt text ./...; \
	else \
		echo "gosec not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi

trivy: ## Run trivy vulnerability scanner
	@echo "Running trivy..."
	@if command -v trivy >/dev/null 2>&1; then \
		trivy fs --security-checks vuln,config .; \
	else \
		echo "trivy not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi

coverage: test-unit ## Generate HTML coverage report
	@echo "Generating coverage report..."
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

coverage-text: test-unit ## Show coverage summary in terminal
	@echo "Coverage summary:"
	@go tool cover -func=$(COVERAGE_FILE)

##@ Performance

bench: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

bench-full: ## Run comprehensive benchmarks (10s)
	@echo "Running comprehensive benchmarks..."
	@go test -bench=. -benchmem -benchtime=10s ./...

##@ Deployment & Dependencies

deps: ## Download and verify Go module dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

check: ## Verify all build prerequisites are installed
	@echo "Checking prerequisites..."
	@command -v go >/dev/null 2>&1 || { echo "go is required but not installed."; exit 1; }
	@command -v git >/dev/null 2>&1 || { echo "git is required but not installed."; exit 1; }
	@command -v docker >/dev/null 2>&1 || { echo "docker is required but not installed."; exit 1; }
	@echo "All prerequisites installed ✓"

ci: vet test-unit build test-cli ## Run full CI pipeline (vet, test, build, test-cli)
	@echo "CI pipeline completed successfully ✓"

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@echo "Installing trivy..."
	@if [[ "$$(uname)" == "Darwin" ]]; then \
		brew install aquasecurity/trivy/trivy; \
	else \
		curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin; \
	fi
	@echo "Development tools installed ✓"

##@ Git Hooks

pre-commit: fmt vet build ## Run all pre-commit checks (fmt, vet, build)
	@echo ""
	@echo "✅ All pre-commit checks passed!"

pre-push: gosec trivy test ## Run all pre-push checks (gosec, trivy, test)
	@echo ""
	@echo "✅ All pre-push checks passed!"

hooks: hooks-install ## Install git hooks (alias for hooks-install)

hooks-install: ## Install pre-commit and pre-push hooks
	@echo "Installing git hooks..."
	@git config core.hooksPath .githooks
	@echo "Git hooks installed successfully ✓"
	@echo "Hooks location: .githooks/"
	@echo "- pre-commit: fmt, vet, build"
	@echo "- pre-push: gosec, trivy, tests"

hooks-uninstall: ## Remove git hooks configuration
	@echo "Removing git hooks..."
	@git config --unset core.hooksPath || true
	@echo "Git hooks uninstalled ✓"

hooks-check: ## Check current git hooks configuration
	@echo "Current git hooks configuration:"
	@echo -n "Hooks path: "
	@git config core.hooksPath || echo "(default .git/hooks/)"
	@if [ -d ".githooks" ]; then \
		echo "Available hooks in .githooks/:"; \
		ls -la .githooks/ | grep -E "pre-commit|pre-push" || echo "  No hooks found"; \
	else \
		echo "No .githooks directory found"; \
	fi

##@ Legacy Targets (for backwards compatibility)

all: ci ## Run vet, test, build, and test-cli (legacy target)
colortest: test-color ## Alias for test-color (legacy)
clitests: test-cli ## Alias for test-cli (legacy)
