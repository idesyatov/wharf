# Makefile for Wharf — TUI for Docker Compose

# Variables
APP_NAME := wharf
BUILD_DIR := ./bin
COMPOSE := docker compose -f docker-compose.dev.yml
BUILDFLAGS := -buildvcs=false
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%d 2>/dev/null || echo "unknown")
LDFLAGS := -s -w \
    -X github.com/idesyatov/wharf/internal/version.Version=$(VERSION) \
    -X github.com/idesyatov/wharf/internal/version.Commit=$(COMMIT) \
    -X github.com/idesyatov/wharf/internal/version.BuildDate=$(BUILD_DATE)
LINT_INSTALL := go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null

# Default target
.PHONY: all build build-all run test vet lint clean \
        docker-build docker-build-linux docker-build-darwin-amd64 docker-build-darwin-arm64 docker-build-windows docker-build-all \
        docker-run docker-test docker-vet docker-lint docker-deps docker-shell docker-clean docker-check \
        release help
all: vet test build-all

# =============================================================================
# Build (requires Go)
# =============================================================================

build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/wharf || { echo "Build failed"; exit 1; }

build-all:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/wharf
	@echo "Building for Darwin amd64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/wharf
	@echo "Building for Darwin arm64..."
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/wharf
	@echo "Building for Windows amd64..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/wharf
	@echo "All binaries built in $(BUILD_DIR)/"

run:
	@go run ./cmd/wharf

test:
	@echo "Running tests..."
	@go test ./... || { echo "Tests failed"; exit 1; }

vet:
	@echo "Running go vet..."
	@go vet ./... || { echo "Vet failed"; exit 1; }

lint:
	@echo "Running linter..."
	@golangci-lint run || { echo "Lint failed"; exit 1; }

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR) 2>/dev/null || docker run --rm -v "$(CURDIR)":/app alpine rm -rf /app/bin || echo "Could not remove build directory"

# =============================================================================
# Docker build (no local Go required)
# =============================================================================

docker-build: docker-build-linux

docker-build-linux:
	@echo "Building for Linux amd64..."
	@$(COMPOSE) run --rm -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 \
		-e VERSION=$(VERSION) -e COMMIT=$(COMMIT) -e BUILD_DATE=$(BUILD_DATE) dev \
		sh -c 'mkdir -p $(BUILD_DIR) && go build $(BUILDFLAGS) -ldflags "-s -w -X github.com/idesyatov/wharf/internal/version.Version=$$VERSION -X github.com/idesyatov/wharf/internal/version.Commit=$$COMMIT -X github.com/idesyatov/wharf/internal/version.BuildDate=$$BUILD_DATE" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/wharf && chmod 777 $(BUILD_DIR) && chmod 755 $(BUILD_DIR)/*' \
		|| { echo "Build failed"; exit 1; }

docker-build-darwin-amd64:
	@echo "Building for Darwin amd64..."
	@$(COMPOSE) run --rm -e CGO_ENABLED=0 -e GOOS=darwin -e GOARCH=amd64 \
		-e VERSION=$(VERSION) -e COMMIT=$(COMMIT) -e BUILD_DATE=$(BUILD_DATE) dev \
		sh -c 'mkdir -p $(BUILD_DIR) && go build $(BUILDFLAGS) -ldflags "-s -w -X github.com/idesyatov/wharf/internal/version.Version=$$VERSION -X github.com/idesyatov/wharf/internal/version.Commit=$$COMMIT -X github.com/idesyatov/wharf/internal/version.BuildDate=$$BUILD_DATE" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/wharf && chmod 777 $(BUILD_DIR) && chmod 755 $(BUILD_DIR)/*' \
		|| { echo "Build failed"; exit 1; }

docker-build-darwin-arm64:
	@echo "Building for Darwin arm64..."
	@$(COMPOSE) run --rm -e CGO_ENABLED=0 -e GOOS=darwin -e GOARCH=arm64 \
		-e VERSION=$(VERSION) -e COMMIT=$(COMMIT) -e BUILD_DATE=$(BUILD_DATE) dev \
		sh -c 'mkdir -p $(BUILD_DIR) && go build $(BUILDFLAGS) -ldflags "-s -w -X github.com/idesyatov/wharf/internal/version.Version=$$VERSION -X github.com/idesyatov/wharf/internal/version.Commit=$$COMMIT -X github.com/idesyatov/wharf/internal/version.BuildDate=$$BUILD_DATE" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/wharf && chmod 777 $(BUILD_DIR) && chmod 755 $(BUILD_DIR)/*' \
		|| { echo "Build failed"; exit 1; }

docker-build-windows:
	@echo "Building for Windows amd64..."
	@$(COMPOSE) run --rm -e CGO_ENABLED=0 -e GOOS=windows -e GOARCH=amd64 \
		-e VERSION=$(VERSION) -e COMMIT=$(COMMIT) -e BUILD_DATE=$(BUILD_DATE) dev \
		sh -c 'mkdir -p $(BUILD_DIR) && go build $(BUILDFLAGS) -ldflags "-s -w -X github.com/idesyatov/wharf/internal/version.Version=$$VERSION -X github.com/idesyatov/wharf/internal/version.Commit=$$COMMIT -X github.com/idesyatov/wharf/internal/version.BuildDate=$$BUILD_DATE" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/wharf && chmod 777 $(BUILD_DIR) && chmod 755 $(BUILD_DIR)/*' \
		|| { echo "Build failed"; exit 1; }

docker-build-all: docker-build-linux docker-build-darwin-amd64 docker-build-darwin-arm64 docker-build-windows
	@echo "All binaries built in $(BUILD_DIR)/"

# =============================================================================
# Docker development
# =============================================================================

docker-run:
	@$(COMPOSE) run --rm dev go run $(BUILDFLAGS) ./cmd/wharf

docker-test:
	@echo "Running tests..."
	@$(COMPOSE) run --rm dev go test $(BUILDFLAGS) ./... || { echo "Tests failed"; exit 1; }

docker-vet:
	@echo "Running go vet..."
	@$(COMPOSE) run --rm dev go vet ./... || { echo "Vet failed"; exit 1; }

docker-lint:
	@echo "Running linter..."
	@$(COMPOSE) run --rm dev sh -c '$(LINT_INSTALL); golangci-lint run' || { echo "Lint failed"; exit 1; }

docker-deps:
	@echo "Updating dependencies..."
	@$(COMPOSE) run --rm dev go mod tidy

docker-shell:
	@$(COMPOSE) run --rm dev sh

docker-test-integration:
	@echo "Running integration tests..."
	@$(COMPOSE) run --rm dev go test -tags integration -v $(BUILDFLAGS) ./internal/docker/ || { echo "Integration tests failed"; exit 1; }

docker-clean:
	@echo "Cleaning up..."
	@$(COMPOSE) run --rm dev rm -rf $(BUILD_DIR) 2>/dev/null || true
	@$(COMPOSE) down -v
	@rm -rf $(BUILD_DIR) 2>/dev/null || true

# =============================================================================
# Pre-release check
# =============================================================================

docker-check:
	@echo "Running pre-release checks..."
	@echo "=== Build ==="
	@$(COMPOSE) run --rm dev go build $(BUILDFLAGS) ./cmd/wharf || { echo "❌ Build failed"; exit 1; }
	@echo "✓ Build OK"
	@echo "=== Vet ==="
	@$(COMPOSE) run --rm dev go vet ./... || { echo "❌ Vet failed"; exit 1; }
	@echo "✓ Vet OK"
	@echo "=== Test ==="
	@$(COMPOSE) run --rm dev go test $(BUILDFLAGS) ./... || { echo "❌ Tests failed"; exit 1; }
	@echo "✓ Tests OK"
	@echo "=== Lint ==="
	@$(COMPOSE) run --rm dev sh -c '$(LINT_INSTALL); golangci-lint run' || { echo "❌ Lint failed"; exit 1; }
	@echo "✓ Lint OK"
	@echo ""
	@echo "✅ All checks passed — ready to release"

# =============================================================================
# Release
# =============================================================================

release: docker-check
ifndef VERSION
	$(error VERSION is not set. Usage: make release VERSION=v0.1.0)
endif
	@echo "Releasing $(VERSION)..."
	@git tag $(VERSION)
	@git push origin $(VERSION) || { echo "Push failed"; exit 1; }
	@echo "Tag $(VERSION) pushed. GitHub Actions will build and publish the release."

# =============================================================================
# Help
# =============================================================================

help:
	@echo "Makefile for Wharf — TUI for Docker Compose"
	@echo ""
	@echo "Local (requires Go):"
	@echo "  make build                      - Build for current platform"
	@echo "  make build-all                  - Cross-compile for all platforms"
	@echo "  make run                        - Run TUI"
	@echo "  make test                       - Run unit tests"
	@echo "  make vet                        - Run go vet"
	@echo "  make lint                       - Run golangci-lint"
	@echo "  make clean                      - Remove build artifacts"
	@echo ""
	@echo "Docker (no local Go required):"
	@echo "  make docker-build               - Build for Linux amd64 (default)"
	@echo "  make docker-build-linux         - Build for Linux amd64"
	@echo "  make docker-build-darwin-amd64  - Build for macOS Intel"
	@echo "  make docker-build-darwin-arm64  - Build for macOS Apple Silicon"
	@echo "  make docker-build-windows       - Build for Windows amd64"
	@echo "  make docker-build-all           - Cross-compile for all platforms"
	@echo "  make docker-run                 - Run TUI"
	@echo "  make docker-test                - Run unit tests"
	@echo "  make docker-vet                 - Run go vet"
	@echo "  make docker-lint                - Run golangci-lint"
	@echo "  make docker-deps                - Update dependencies (go mod tidy)"
	@echo "  make docker-shell               - Open shell in dev container"
	@echo "  make docker-test-integration    - Run integration tests"
	@echo "  make docker-clean               - Remove binaries and Docker volumes"
	@echo "  make docker-check               - Pre-release: build + vet + test + lint"
	@echo ""
	@echo "Release:"
	@echo "  make release VERSION=v0.1.0     - Pre-release check + tag and push"
	@echo ""
	@echo "Other:"
	@echo "  make all                        - Vet, test, and build all"
	@echo "  make help                       - Show this message"

.DEFAULT_GOAL := build
