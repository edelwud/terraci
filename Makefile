.PHONY: build build-xterraci test clean install lint fmt help

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOMOD=$(GOCMD) mod

# Directories
BUILD_DIR=build

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'

## build: Build terraci and xterraci
build: build-terraci build-xterraci

## build-terraci: Build the terraci binary
build-terraci:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/terraci ./cmd/terraci

## build-xterraci: Build the xterraci binary
build-xterraci:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/xterraci ./cmd/xterraci

## build-all: Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/terraci-linux-amd64 ./cmd/terraci
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/terraci-linux-arm64 ./cmd/terraci
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/xterraci-linux-amd64 ./cmd/xterraci
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/xterraci-linux-arm64 ./cmd/xterraci

build-darwin:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/terraci-darwin-amd64 ./cmd/terraci
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/terraci-darwin-arm64 ./cmd/terraci
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/xterraci-darwin-amd64 ./cmd/xterraci
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/xterraci-darwin-arm64 ./cmd/xterraci

build-windows:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/terraci-windows-amd64.exe ./cmd/terraci
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/xterraci-windows-amd64.exe ./cmd/xterraci

## install: Install both binaries to $GOPATH/bin
install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/terraci ./cmd/terraci
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/xterraci ./cmd/xterraci

## test: Run tests
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-short: Run short tests
test-short:
	$(GOTEST) -v -short ./...

## coverage: Show test coverage
coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, running go vet instead"; \
		$(GOVET) ./...; \
	fi

## fmt: Format code
fmt:
	$(GOFMT) ./...

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## clean: Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## deps: Download dependencies
deps:
	$(GOMOD) download

## run: Run terraci
run: build-terraci
	./$(BUILD_DIR)/terraci

## example: Generate example pipeline
example: build-terraci
	./$(BUILD_DIR)/terraci generate --dry-run

## version: Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"
