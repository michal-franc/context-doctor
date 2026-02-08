BINARY_NAME=context-doctor
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Installation directory (can be overridden)
PREFIX?=/usr/local
BINDIR?=$(PREFIX)/bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

.PHONY: all build clean test coverage lint install uninstall help

all: build

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## install: Install to $(BINDIR) (default: /usr/local/bin)
install: build
	@echo "Installing $(BINARY_NAME) to $(BINDIR)..."
	@mkdir -p $(BINDIR)
	@cp $(BINARY_NAME) $(BINDIR)/$(BINARY_NAME)
	@chmod +x $(BINDIR)/$(BINARY_NAME)
	@echo "Done! Run '$(BINARY_NAME)' from anywhere."

## install-user: Install to ~/go/bin (no sudo required)
install-user: build
	@echo "Installing $(BINARY_NAME) to ~/go/bin..."
	@mkdir -p $(HOME)/go/bin
	@cp $(BINARY_NAME) $(HOME)/go/bin/$(BINARY_NAME)
	@chmod +x $(HOME)/go/bin/$(BINARY_NAME)
	@echo "Done! Make sure ~/go/bin is in your PATH."

## uninstall: Remove from $(BINDIR)
uninstall:
	@echo "Removing $(BINARY_NAME) from $(BINDIR)..."
	@rm -f $(BINDIR)/$(BINARY_NAME)
	@echo "Done!"

## clean: Remove build artifacts
clean:
	@rm -f $(BINARY_NAME)
	@echo "Cleaned."

## test: Run tests
test:
	$(GOTEST) -v ./...

## coverage: Run tests with coverage report
coverage:
	$(GOTEST) ./... -coverprofile=coverage.out -covermode=atomic
	$(GOCMD) tool cover -func=coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## lint: Run linter
lint:
	golangci-lint run ./...

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
