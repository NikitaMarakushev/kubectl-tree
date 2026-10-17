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