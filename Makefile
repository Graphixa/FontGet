.PHONY: build build-dev build-release version clean help

# Default target
.DEFAULT_GOAL := help

# Detect version from git tag, fallback to dev
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -s -w \
	-X fontget/internal/version.Version=$(VERSION) \
	-X fontget/internal/version.GitCommit=$(COMMIT) \
	-X fontget/internal/version.BuildDate=$(DATE)

# Binary name (platform-specific)
BINARY_NAME := fontget
ifeq ($(OS),Windows_NT)
	BINARY_NAME := fontget.exe
endif

help: ## Show this help message
	@echo "FontGet Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build with auto-detected version (from git tag)
	@echo "Building FontGet v$(VERSION)..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Date:    $(DATE)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

build-dev: ## Build for local testing (simple dev version)
	@echo "Building FontGet (local dev build)..."
	@echo "  Version: dev"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Date:    $(DATE)"
	@go build -ldflags "-s -w -X fontget/internal/version.Version=dev -X fontget/internal/version.GitCommit=$(COMMIT) -X fontget/internal/version.BuildDate=$(DATE)" -o $(BINARY_NAME) .

build-release: ## Build with specific version (use VERSION=v2.1.0 make build-release)
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "Error: No git tag found. Use VERSION=v2.1.0 make build-release"; \
		exit 1; \
	fi
	@echo "Building FontGet v$(VERSION) (release)..."
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Date:    $(DATE)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

version: ## Show current version info
	@echo "Version Information:"
	@echo "  Latest Tag:  $(VERSION)"
	@echo "  Commit:      $(COMMIT)"
	@echo "  Build Date:  $(DATE)"
	@if [ -f $(BINARY_NAME) ]; then \
		echo ""; \
		echo "Built Binary:"; \
		./$(BINARY_NAME) version 2>/dev/null || echo "  (run 'make build' first)"; \
	fi

clean: ## Remove built binaries
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME) fontget.exe
	@echo "Done!"

test: ## Run tests
	@go test ./...

