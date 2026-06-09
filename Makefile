.PHONY: help build clean test run setup install lint

# Variables
BINARY_NAME=z9s
VERSION?=v0.55.5
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE?=$(shell date -u '+%Y-%m-%d')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
help:
	@echo "KT9S Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make setup         - Setup development environment"
	@echo "  make build         - Build the binary"
	@echo "  make run           - Build and run"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make test          - Run tests"
	@echo "  make lint          - Run linter"
	@echo "  make install       - Install binary to GOBIN"
	@echo "  make help          - Show this help message"
	@echo ""

# Setup development environment
setup:
	@echo "📦 Setting up development environment..."
	@go mod tidy
	@echo "✅ Setup complete"

# Build the binary
build: setup
	@echo "🔨 Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME)
	@echo "✅ Build complete: ./$(BINARY_NAME)"

# Run the application
run: build
	@echo "🚀 Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean
	@echo "✅ Clean complete"

# Run tests
test: setup
	@echo "🧪 Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@echo "✅ Tests complete"

# Coverage report
coverage: test
	@echo "📊 Coverage report:"
	@go tool cover -func=coverage.out

# Lint code
lint:
	@echo "🔍 Linting..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run ./...
	@echo "✅ Lint complete"

# Install binary
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	@cp $(BINARY_NAME) $(GOPATH)/bin/ || cp $(BINARY_NAME) $(GOROOT)/bin/
	@echo "✅ Installed to GOBIN"

# Development build with race detector
dev-build: setup
	@echo "🔨 Building $(BINARY_NAME) with race detector..."
	@go build $(LDFLAGS) -race -o $(BINARY_NAME)
	@echo "✅ Build complete: ./$(BINARY_NAME)"

# Format code
fmt:
	@echo "📝 Formatting code..."
	@go fmt ./...
	@echo "✅ Format complete"

# Check dependencies
deps:
	@echo "📚 Checking dependencies..."
	@go mod graph | grep -v "^\s*$"

# Show version info
version:
	@echo "KT9S Version Information:"
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Date:    $(DATE)"

# Build for multiple platforms
build-all: setup
	@echo "🔨 Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe
	@echo "✅ Multi-platform builds complete"
	@ls -lh $(BINARY_NAME)-*

# Quick check without full test
check: fmt
	@echo "🔍 Quick check..."
	@go vet ./...
	@echo "✅ Quick check complete"
