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

.DEFAULT_GOAL := help