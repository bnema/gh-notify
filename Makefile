.PHONY: install-tools generate-mocks test test-coverage clean-mocks build build-dev install help

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X github.com/bnema/gh-notify/cmd.version=$(VERSION) -X github.com/bnema/gh-notify/cmd.commit=$(COMMIT) -X github.com/bnema/gh-notify/cmd.buildDate=$(BUILD_DATE)

# Install development tools
install-tools:
	@echo "Installing mockgen..."
	@go install go.uber.org/mock/mockgen@latest
	@echo "✓ Tools installed"

# Generate all mocks
generate-mocks:
	@echo "Generating mocks..."
	@mkdir -p internal/github/mocks
	@mockgen -source=internal/github/interfaces.go \
		-destination=internal/github/mocks/mock_github.go \
		-package=mocks
	@echo "✓ Mocks generated"

# Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# Clean generated mocks
clean-mocks:
	@echo "Cleaning generated mocks..."
	@rm -rf internal/github/mocks
	@echo "✓ Mocks cleaned"

# Build for current platform
build:
	@echo "Building gh-notify $(VERSION)..."
	@go build -ldflags "$(LDFLAGS)" -o gh-notify .
	@echo "✓ Build complete: ./gh-notify"

# Build without optimizations (for development/debugging)
build-dev:
	@echo "Building gh-notify $(VERSION) (development mode)..."
	@go build -ldflags "-X github.com/bnema/gh-notify/cmd.version=$(VERSION) -X github.com/bnema/gh-notify/cmd.commit=$(COMMIT) -X github.com/bnema/gh-notify/cmd.buildDate=$(BUILD_DATE)" -o gh-notify .
	@echo "✓ Development build complete: ./gh-notify"

# Install to system
install: build
	@echo "Installing gh-notify to /usr/local/bin..."
	@sudo install -m 755 gh-notify /usr/local/bin/
	@echo "✓ Installed: /usr/local/bin/gh-notify"

# Show help
help:
	@echo "Available targets:"
	@echo "  build            - Build gh-notify with version information"
	@echo "  build-dev        - Build without optimizations (for debugging)"
	@echo "  install          - Build and install to /usr/local/bin"
	@echo "  install-tools    - Install mockgen and other dev tools"
	@echo "  generate-mocks   - Generate mock files from interfaces"
	@echo "  test             - Run all tests"
	@echo "  test-coverage    - Run tests and generate coverage report"
	@echo "  clean-mocks      - Remove generated mock files"
	@echo "  help             - Show this help message"
