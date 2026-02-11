# Makefile for gosearch

.PHONY: all build run test clean fmt vet lint race bench help

# Variables
BINARY_NAME=gosearch
BUILD_DIR=bin
CMD_DIR=./cmd/gosearch
GO_FILES=$(shell find . -name '*.go' -type f)

# Default target
all: fmt vet build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## run: Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

## race: Run tests with race detector
race:
	@echo "Running tests with race detector..."
	@go test -race -v ./...

## bench: Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean ./...
	@echo "Clean complete"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet complete"

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run ./...
	@echo "Lint complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies ready"

## install-deps: Install development tools
install-deps:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

## cover: Run tests with coverage
cover:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
