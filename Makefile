GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=kubectl-tree
INSTALL_DIR=/usr/local/bin

.PHONY: all build clean test run deps fmt lint install help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v

clean:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./..

deps:
	@echo "Download dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	golangci-lint run

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	sudo mv $(BINARY_NAME) $(INSTALL_DIR)/
	@echo "Installed! You can now run: kubectl tree <deployment>"

help:
	@echo "Avaliable targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/^## //' | column -t -s ':'

test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

build-all:
	@echo "Building for multiple platforms..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-linux-amd64 -v
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-darwin-amd64 -v
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_NAME)-darwin-arm64 -v
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)-windows-amd64.exe -v
	@echo "Binaries created in current directory"

.DEFAULT_GOAL := help

# Release targets
.PHONY: snapshot release

snapshot:
	@echo "Building a local snapshot release..."
	goreleaser release --snapshot --clean

release:
	@echo "Creating a release (requires git tag)..."
	goreleaser release --clean