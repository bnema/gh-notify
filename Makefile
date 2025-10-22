.PHONY: install-tools generate-mocks test test-coverage clean-mocks help

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

# Show help
help:
	@echo "Available targets:"
	@echo "  install-tools    - Install mockgen and other dev tools"
	@echo "  generate-mocks   - Generate mock files from interfaces"
	@echo "  test             - Run all tests"
	@echo "  test-coverage    - Run tests and generate coverage report"
	@echo "  clean-mocks      - Remove generated mock files"
	@echo "  help             - Show this help message"
