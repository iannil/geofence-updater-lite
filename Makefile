# Makefile for Geofence-Updater-Lite

# Variables
BINARY_NAME=publisher
SDK_EXAMPLE_NAME=sdk-example
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GO=go
GOFLAGS=-v
PROTOC=protoc

# Directories
CMD_DIR=./cmd
PKG_DIR=./pkg
BUILD_DIR=./bin
SCRIPTS_DIR=./scripts

# Platform specific settings
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    PLATFORM := darwin
endif
ifeq ($(UNAME_S),Linux)
    PLATFORM := linux
endif

.PHONY: all
all: build

## build: Build the publisher binary
.PHONY: build
build:
	@echo "Building publisher..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/publisher

## build-sdk: Build the SDK example binary
.PHONY: build-sdk
build-sdk:
	@echo "Building SDK example..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(SDK_EXAMPLE_NAME) $(CMD_DIR)/sdk-example

## build-all: Build all binaries
.PHONY: build-all
build-all: build build-sdk

## proto: Generate Protobuf code
.PHONY: proto
proto:
	@echo "Generating Protobuf code..."
	@if command -v $(PROTOC) >/dev/null 2>&1; then \
		$(SCRIPTS_DIR)/generate_proto.sh; \
	else \
		echo "Error: protoc not found. Please install Protocol Buffers compiler."; \
		exit 1; \
	fi

## keys: Generate a new key pair
.PHONY: keys
keys:
	@echo "Generating new key pair..."
	@$(SCRIPTS_DIR)/gen_keys.sh

## test: Run all tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

## test-short: Run short tests only
.PHONY: test-short
test-short:
	@echo "Running short tests..."
	$(GO) test -v -short ./...

## test-bench: Run benchmarks
.PHONY: test-bench
test-bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

## test-coverage: Run tests with coverage report
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linters
.PHONY: lint
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Run 'make install-tools' to install."; \
	fi

## fmt: Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

## vet: Run go vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## tidy: Tidy go.mod
.PHONY: tidy
tidy:
	@echo "Tidying go.mod..."
	$(GO) mod tidy

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## clean-all: Clean all generated files
.PHONY: clean-all
clean-all: clean
	@echo "Cleaning all generated files..."
	rm -rf $(PKG_DIR)/protocol/protobuf/*.pb.go

## install-tools: Install development tools
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

## cross-compile: Cross-compile for multiple platforms
.PHONY: cross-compile
cross-compile:
	@echo "Cross-compiling..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/publisher
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/publisher
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/publisher
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/publisher
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/publisher
	@echo "Built binaries:"
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)-*

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t gul-publisher:$(VERSION) .

## docker-run: Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -it --rm gul-publisher:$(VERSION)

## docker-build: Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t gul-publisher:$(VERSION) .

## docker-run: Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -it --rm gul-publisher:$(VERSION)

## help: Show this help message
.PHONY: help
help:
	@echo "Geofence-Updater-Lite Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

	@echo ""
	@echo "Quick start:"
	@echo "  make build        # Build binaries"
	@echo "  make test         # Run tests"
	@	@echo "  make lint         # Run linters"
	@	@echo "  make fmt          # Format code"
	@	@@echo "  make clean        # Clean build files"
	@	@@echo "  make test-coverage # Run tests with coverage report"
	@	@echo "  make run-example   # Run SDK example"
	@	@@echo "  make docs         # Show documentation"
