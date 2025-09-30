# FontGet Build Configuration

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS = -X fontget/internal/version.Version=$(VERSION) \
          -X fontget/internal/version.GitCommit=$(GIT_COMMIT) \
          -X fontget/internal/version.BuildDate=$(BUILD_DATE)

# Default target
.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o fontget.exe

# Development build (no version injection)
.PHONY: dev
dev:
	go build -o fontget.exe

# Clean build artifacts
.PHONY: clean
clean:
	rm -f fontget.exe

# Install to GOPATH/bin
.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)"

# Run tests
.PHONY: test
test:
	go test ./...

# Show version info that would be injected
.PHONY: version-info
version-info:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Help
.PHONY: help
help:
	@echo "FontGet Build Targets:"
	@echo "  build        Build with version injection (recommended)"
	@echo "  dev          Quick development build (no version injection)"
	@echo "  install      Install to GOPATH/bin with version injection"
	@echo "  test         Run tests"
	@echo "  clean        Remove build artifacts"
	@echo "  version-info Show version information that would be injected"
	@echo "  help         Show this help message"
