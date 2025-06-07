.PHONY: all vet test colortest clitests build integration help

# Default target shows help
.DEFAULT_GOAL := help

all: vet test build clitests

help: ## Show this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Use 'make <target>' to run a specific target."

vet: ## Run go vet on all packages
	go list ./... | grep -v vendor | xargs go vet

test: ## Run all unit tests
	go list ./... | grep -v vendor | xargs go test

colortest: build ## Test color output functionality
	./assets/color_tester

clitests: build ## Run CLI integration tests
	./assets/cli_tests

build: ## Build the graft binary
	go build ./cmd/graft

integration: build ## Run integration tests with external services (Vault, NATS)
	@echo "Running integration tests..."
	@scripts/integration
