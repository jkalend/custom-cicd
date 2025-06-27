# Custom CI/CD CLI Makefile

.PHONY: build clean test install deps build-all release help

# Variables
BINARY_NAME=cicd
VERSION=1.0.0
BUILD_DIR=build
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

# Default target
all: build

# Build the CLI for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	go clean

# Build for all major platforms
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe .
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=linux GOARCH=386 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-386 .

# Create release packages
release: build-all
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/releases
	
	# Create zip files for Windows
	cd $(BUILD_DIR) && zip releases/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	cd $(BUILD_DIR) && zip releases/$(BINARY_NAME)-$(VERSION)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe
	
	# Create tar.gz files for Unix-like systems
	cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	cd $(BUILD_DIR) && tar -czf releases/$(BINARY_NAME)-$(VERSION)-linux-386.tar.gz $(BINARY_NAME)-linux-386

# Run the CLI with help
run:
	go run . --help

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build for current platform"
	@echo "  build-all  - Build for all major platforms"
	@echo "  clean      - Clean build artifacts"
	@echo "  deps       - Install dependencies"
	@echo "  test       - Run tests"
	@echo "  install    - Install to GOPATH/bin"
	@echo "  release    - Create release packages"
	@echo "  run        - Run with help"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  help       - Show this help" 
