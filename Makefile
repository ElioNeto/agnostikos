.PHONY: help build test clean install test-iso lint fmt deps

BINARY_NAME=agnostic
BUILD_DIR=build
GO=go
LDFLAGS=-ldflags "-X github.com/ElioNeto/agnostikos/cmd/agnostic.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev) -X github.com/ElioNeto/agnostikos/cmd/agnostic.Commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the CLI binary
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

test: ## Run unit tests
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-iso: ## Test ISO in QEMU
	@bash scripts/run-qemu.sh $(BUILD_DIR)/agnostikos-latest.iso

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go code
	$(GO) fmt ./...

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR) coverage.out

deps: ## Download Go dependencies
	$(GO) mod download
	$(GO) mod tidy

dev: ## Run in development mode
	@$(GO) run . --help

install: build ## Install binary to /usr/local/bin
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)

.DEFAULT_GOAL := help