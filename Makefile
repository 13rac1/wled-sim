# WLED Simulator Makefile

# Binary name
BINARY_NAME=wled-sim

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build directory
BUILD_DIR=build

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

# Build with version info
.PHONY: build-release
build-release:
	@echo "Building $(BINARY_NAME) with version info..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd

# Generate the WLED Client
.PHONY: gen
gen:
	# Not used
	openapi-generator-cli generate -g go -i docs/openapi-wled.json -o internal/openapi

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run the simulator with default config
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run the simulator with custom config
.PHONY: run-demo
run-demo: build
	@echo "Running $(BINARY_NAME) with demo config..."
	./$(BUILD_DIR)/$(BINARY_NAME) -rows 4 -cols 5 -wiring col -init "#00FF00"

# Run in headless mode
.PHONY: run-headless
run-headless: build
	@echo "Running $(BINARY_NAME) in headless mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) -headless -v

# Install the binary to GOPATH/bin
.PHONY: install
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-release - Build optimized binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  run           - Build and run with default config"
	@echo "  run-demo      - Build and run with demo config"
	@echo "  run-headless  - Build and run in headless mode"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  help          - Show this help message" 
