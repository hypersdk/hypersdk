.PHONY: all build build-daemon build-ctl build-all clean test test-quick test-short test-verbose bench \
        fmt vet lint lint-fix security check ci \
        docker-build docker-run docker-push \
        install install-daemon install-ctl uninstall \
        release release-snapshot deps deps-update deps-verify vendor \
        run-daemon run-ctl run-debug help version

# ==================================================================================== #
# VARIABLES
# ==================================================================================== #

# Binary names
DAEMON_BINARY=hypervisord
CTL_BINARY=hyperctl
EXPORT_BINARY=hyperexport

# Build directory
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS=-ldflags "\
	-s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildTime=$(BUILD_TIME) \
	-X main.goVersion=$(GO_VERSION)"

# Install paths
INSTALL_PREFIX ?= /usr/local
BIN_DIR=$(INSTALL_PREFIX)/bin
SYSTEMD_DIR=/etc/systemd/system

# Docker parameters
DOCKER_IMAGE ?= hypervisord
DOCKER_TAG ?= $(VERSION)
DOCKER_REGISTRY ?=

# Test parameters
TEST_TIMEOUT ?= 10m
TEST_PARALLEL ?= 4

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

.DEFAULT_GOAL := help

all: clean build test ## Clean, build, and test everything

build: build-daemon build-ctl build-export ## Build all binaries

build-daemon: ## Build hypervisord daemon
	@echo "🔨 Building $(DAEMON_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_BINARY) ./cmd/hypervisord
	@echo "✅ Build complete: $(BUILD_DIR)/$(DAEMON_BINARY)"

build-ctl: ## Build hyperctl CLI
	@echo "🔨 Building $(CTL_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CTL_BINARY) ./cmd/hyperctl
	@echo "✅ Build complete: $(BUILD_DIR)/$(CTL_BINARY)"

build-export: ## Build hyperexport interactive CLI
	@echo "🔨 Building $(EXPORT_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(EXPORT_BINARY) ./cmd/hyperexport
	@echo "✅ Build complete: $(BUILD_DIR)/$(EXPORT_BINARY)"

# ==================================================================================== #
# TESTING
# ==================================================================================== #

test: ## Run tests with race detector and coverage
	@echo "🧪 Running tests..."
	$(GOTEST) -v -race -timeout=$(TEST_TIMEOUT) -coverprofile=coverage.out ./...
	@echo "✅ Tests complete"

test-quick: ## Run tests without race detector (faster)
	@echo "⚡ Running quick tests..."
	$(GOTEST) -timeout=$(TEST_TIMEOUT) -coverprofile=coverage.out ./...
	@echo "✅ Quick tests complete"

test-short: ## Run only short tests
	@echo "⏱️  Running short tests..."
	$(GOTEST) -short -timeout=1m ./...
	@echo "✅ Short tests complete"

test-verbose: ## Run tests with verbose output
	@echo "🔍 Running verbose tests..."
	$(GOTEST) -v -race -timeout=$(TEST_TIMEOUT) ./...

test-coverage: test ## Show detailed test coverage
	@echo "📊 Test coverage:"
	@$(GOCMD) tool cover -func=coverage.out | tail -1

test-coverage-html: test ## Open test coverage in browser
	@echo "📊 Opening test coverage in browser..."
	@$(GOCMD) tool cover -html=coverage.out

bench: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	$(GOTEST) -bench=. -benchmem -run=^$$ ./...
	@echo "✅ Benchmarks complete"

bench-compare: ## Run benchmarks and save for comparison
	@echo "⚡ Running benchmarks..."
	@mkdir -p $(BUILD_DIR)
	$(GOTEST) -bench=. -benchmem -run=^$$ ./... | tee $(BUILD_DIR)/bench.txt
	@echo "✅ Benchmarks saved to $(BUILD_DIR)/bench.txt"

# ==================================================================================== #
# CODE QUALITY
# ==================================================================================== #

fmt: ## Format code
	@echo "🎨 Formatting code..."
	@$(GOFMT) ./...
	@echo "✅ Format complete"

fmt-check: ## Check if code is formatted
	@echo "🎨 Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "❌ Code is not formatted. Run 'make fmt'" && exit 1)
	@echo "✅ Code is properly formatted"

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	@$(GOVET) ./...
	@echo "✅ Vet complete"

lint: ## Run golangci-lint
	@echo "🔍 Running linter..."
	@which golangci-lint > /dev/null || (echo "⚠️  golangci-lint not found. Install from https://golangci-lint.run/usage/install/"; exit 1)
	@golangci-lint run ./... --timeout=5m
	@echo "✅ Lint complete"

lint-fix: ## Run golangci-lint with auto-fix
	@echo "🔧 Running linter with auto-fix..."
	@which golangci-lint > /dev/null || (echo "⚠️  golangci-lint not found. Install from https://golangci-lint.run/usage/install/"; exit 1)
	@golangci-lint run ./... --fix --timeout=5m
	@echo "✅ Lint with auto-fix complete"

security: ## Run security checks with gosec
	@echo "🔒 Running security checks..."
	@which gosec > /dev/null || (echo "⚠️  gosec not found. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; exit 1)
	@gosec -quiet ./...
	@echo "✅ Security check complete"

staticcheck: ## Run staticcheck
	@echo "🔍 Running staticcheck..."
	@which staticcheck > /dev/null || (echo "⚠️  staticcheck not found. Install: go install honnef.co/go/tools/cmd/staticcheck@latest"; exit 1)
	@staticcheck ./...
	@echo "✅ Staticcheck complete"

check: fmt-check vet lint test ## Run all checks (format, vet, lint, test)
	@echo "✅ All checks passed!"

ci: check security ## Run all CI checks
	@echo "✅ All CI checks passed!"

# ==================================================================================== #
# DEPENDENCIES
# ==================================================================================== #

deps: ## Download dependencies
	@echo "📥 Downloading dependencies..."
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "✅ Dependencies downloaded"

deps-update: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	@$(GOGET) -u ./...
	@$(GOMOD) tidy
	@echo "✅ Dependencies updated"

deps-verify: ## Verify dependencies
	@echo "🔍 Verifying dependencies..."
	@$(GOMOD) verify
	@echo "✅ Dependencies verified"

deps-graph: ## Show dependency graph
	@echo "📊 Dependency graph:"
	@go mod graph

deps-why: ## Explain why a dependency is needed (usage: make deps-why PKG=package/name)
	@$(GOMOD) why $(PKG)

vendor: ## Vendor dependencies
	@echo "📦 Vendoring dependencies..."
	@$(GOMOD) vendor
	@echo "✅ Dependencies vendored"

vuln-check: ## Check for known vulnerabilities (requires govulncheck)
	@echo "🔍 Checking for vulnerabilities..."
	@which govulncheck > /dev/null || (echo "⚠️  govulncheck not found. Install: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1)
	@govulncheck ./...
	@echo "✅ Vulnerability check complete"

# ==================================================================================== #
# BUILD VARIANTS
# ==================================================================================== #

build-linux: ## Build for Linux (amd64)
	@echo "🔨 Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_BINARY)-linux-amd64 ./cmd/hypervisord
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CTL_BINARY)-linux-amd64 ./cmd/hyperctl
	@echo "✅ Linux builds complete"

build-all: ## Build for all platforms
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@echo "Building Linux binaries..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_BINARY)-linux-amd64 ./cmd/hypervisord
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_BINARY)-linux-arm64 ./cmd/hypervisord
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CTL_BINARY)-linux-amd64 ./cmd/hyperctl
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CTL_BINARY)-linux-arm64 ./cmd/hyperctl
	@echo "Building macOS binaries..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_BINARY)-darwin-amd64 ./cmd/hypervisord
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_BINARY)-darwin-arm64 ./cmd/hypervisord
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CTL_BINARY)-darwin-amd64 ./cmd/hyperctl
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CTL_BINARY)-darwin-arm64 ./cmd/hyperctl
	@echo "Building Windows binaries..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(DAEMON_BINARY)-windows-amd64.exe ./cmd/hypervisord
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CTL_BINARY)-windows-amd64.exe ./cmd/hyperctl
	@echo "✅ All platform builds complete"
	@ls -lh $(BUILD_DIR)/

build-race: ## Build with race detector enabled
	@echo "🔨 Building with race detector..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -race -o $(BUILD_DIR)/$(DAEMON_BINARY)-race ./cmd/hypervisord
	@echo "✅ Race build complete"

# ==================================================================================== #
# RELEASE
# ==================================================================================== #

release: clean ## Build release binaries with checksums
	@echo "📦 Building release $(VERSION)..."
	@$(MAKE) build-all
	@echo "🔐 Generating checksums..."
	@cd $(BUILD_DIR) && sha256sum * > SHA256SUMS
	@echo "✅ Release $(VERSION) complete"
	@echo "📋 Release artifacts:"
	@ls -lh $(BUILD_DIR)/

release-snapshot: ## Build snapshot release (no git tag required)
	@echo "📦 Building snapshot release..."
	@VERSION=snapshot-$(COMMIT)-$(shell date +%Y%m%d%H%M%S) $(MAKE) release

# ==================================================================================== #
# DOCKER
# ==================================================================================== #

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-build-no-cache: ## Build Docker image without cache
	@echo "🐳 Building Docker image (no cache)..."
	@docker build --no-cache \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	@docker run --rm \
		-e GOVC_URL \
		-e GOVC_USERNAME \
		-e GOVC_PASSWORD \
		-e GOVC_INSECURE \
		-v $(PWD)/exports:/exports \
		-p 8080:8080 \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: ## Push Docker image to registry
	@echo "🐳 Pushing Docker image to registry..."
	@if [ -n "$(DOCKER_REGISTRY)" ]; then \
		docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG); \
		docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG); \
	else \
		docker push $(DOCKER_IMAGE):$(DOCKER_TAG); \
	fi
	@echo "✅ Docker image pushed"

docker-shell: ## Run shell in Docker container
	@docker run --rm -it --entrypoint /bin/sh $(DOCKER_IMAGE):$(DOCKER_TAG)

# ==================================================================================== #
# INSTALLATION
# ==================================================================================== #

install: install-daemon install-ctl ## Install all binaries

install-daemon: build-daemon ## Install hypervisord daemon
	@echo "📦 Installing $(DAEMON_BINARY)..."
	@sudo install -m 755 $(BUILD_DIR)/$(DAEMON_BINARY) $(BIN_DIR)/$(DAEMON_BINARY)
	@sudo install -m 644 systemd/hypervisord.service $(SYSTEMD_DIR)/hypervisord.service
	@echo "✅ Installed $(DAEMON_BINARY) to $(BIN_DIR)"
	@echo "✅ Installed systemd service to $(SYSTEMD_DIR)"
	@echo ""
	@echo "To enable and start the service:"
	@echo "  sudo systemctl daemon-reload"
	@echo "  sudo systemctl enable hypervisord"
	@echo "  sudo systemctl start hypervisord"

install-ctl: build-ctl ## Install hyperctl CLI
	@echo "📦 Installing $(CTL_BINARY)..."
	@sudo install -m 755 $(BUILD_DIR)/$(CTL_BINARY) $(BIN_DIR)/$(CTL_BINARY)
	@echo "✅ Installed $(CTL_BINARY) to $(BIN_DIR)"

uninstall: ## Uninstall all binaries and services
	@echo "🗑️  Uninstalling..."
	@sudo systemctl stop hypervisord 2>/dev/null || true
	@sudo systemctl disable hypervisord 2>/dev/null || true
	@sudo rm -f $(BIN_DIR)/$(DAEMON_BINARY)
	@sudo rm -f $(BIN_DIR)/$(CTL_BINARY)
	@sudo rm -f $(BIN_DIR)/$(EXPORT_BINARY)
	@sudo rm -f $(SYSTEMD_DIR)/hypervisord.service
	@sudo systemctl daemon-reload
	@echo "✅ Uninstall complete"

# ==================================================================================== #
# CLEANUP
# ==================================================================================== #

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out
	@echo "✅ Clean complete"

clean-all: clean ## Deep clean (including vendor and cache)
	@echo "🧹 Deep cleaning..."
	@rm -rf vendor/
	@$(GOCLEAN) -cache -testcache -modcache
	@echo "✅ Deep clean complete"

# ==================================================================================== #
# RUN
# ==================================================================================== #

run-daemon: build-daemon ## Build and run hypervisord
	@echo "🚀 Running $(DAEMON_BINARY)..."
	@$(BUILD_DIR)/$(DAEMON_BINARY)

run-ctl: build-ctl ## Build and run hyperctl
	@echo "🚀 Running $(CTL_BINARY)..."
	@$(BUILD_DIR)/$(CTL_BINARY)

run-debug: build-daemon ## Build and run with debug logging
	@echo "🐛 Running $(DAEMON_BINARY) in debug mode..."
	@LOG_LEVEL=debug $(BUILD_DIR)/$(DAEMON_BINARY)

run-race: build-race ## Build with race detector and run
	@echo "🏃 Running with race detector..."
	@$(BUILD_DIR)/$(DAEMON_BINARY)-race

# ==================================================================================== #
# UTILITIES
# ==================================================================================== #

version: ## Show version information
	@echo "Version:     $(VERSION)"
	@echo "Commit:      $(COMMIT)"
	@echo "Build time:  $(BUILD_TIME)"
	@echo "Go version:  $(GO_VERSION)"

info: version ## Show build information
	@echo ""
	@echo "Build directory: $(BUILD_DIR)"
	@echo "Install prefix:  $(INSTALL_PREFIX)"
	@echo "Docker image:    $(DOCKER_IMAGE):$(DOCKER_TAG)"

pre-commit: fmt vet lint test ## Run pre-commit checks
	@echo "✅ Pre-commit checks passed"

todo: ## Show TODO comments in code
	@echo "📝 TODO items:"
	@grep -r "TODO\|FIXME\|XXX" --include="*.go" --line-number . || echo "No TODO items found"

lines: ## Count lines of code
	@echo "📊 Lines of code:"
	@find . -name '*.go' -not -path "./vendor/*" | xargs wc -l | tail -1

size: build ## Show binary sizes
	@echo "📏 Binary sizes:"
	@ls -lh $(BUILD_DIR)/ | grep -v "^d" | awk '{print $$9 ": " $$5}'

test-rpm: ## Test RPM build locally
	@echo "📦 Testing RPM build..."
	@./test_rpmbuild.sh

watch: ## Watch for changes and rebuild (requires entr)
	@which entr > /dev/null || (echo "❌ entr not found. Install with: apt install entr"; exit 1)
	@echo "👀 Watching for changes..."
	@find . -name '*.go' | entr -c make build

help: ## Show this help
	@echo ""
	@echo "🚀 HyperSDK - Makefile Help"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "📦 Components:"
	@echo "  • $(DAEMON_BINARY)  - Go daemon for multi-cloud VM export"
	@echo "  • $(CTL_BINARY)     - Control CLI for managing exports"
	@echo "  • $(EXPORT_BINARY)  - Interactive CLI for manual VM exports"
	@echo ""
	@echo "🎯 Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'
	@echo ""
	@echo "💡 Examples:"
	@echo "  make build          # Build all binaries"
	@echo "  make test           # Run tests with coverage"
	@echo "  make check          # Run all quality checks"
	@echo "  make ci             # Run all CI checks"
	@echo "  make release        # Build release artifacts"
	@echo ""
	@echo "📌 Current version: $(VERSION)"
	@echo ""
