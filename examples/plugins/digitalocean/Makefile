# DigitalOcean Provider Plugin Makefile

PLUGIN_NAME = digitalocean
PLUGIN_FILE = $(PLUGIN_NAME).so
GO_VERSION = 1.24

# Installation directories
USER_PLUGIN_DIR = $(HOME)/.hypersdk/plugins
SYSTEM_PLUGIN_DIR = /usr/local/lib/hypersdk/plugins

# Build flags
BUILD_FLAGS = -buildmode=plugin
LDFLAGS = -s -w

.PHONY: all build clean install install-system uninstall test lint

all: build

build:
	@echo "Building $(PLUGIN_NAME) plugin..."
	@go build $(BUILD_FLAGS) -ldflags="$(LDFLAGS)" -o $(PLUGIN_FILE) .
	@echo "Plugin built: $(PLUGIN_FILE)"

clean:
	@echo "Cleaning..."
	@rm -f $(PLUGIN_FILE)
	@go clean

install: build
	@echo "Installing plugin to user directory..."
	@mkdir -p $(USER_PLUGIN_DIR)
	@cp $(PLUGIN_FILE) $(USER_PLUGIN_DIR)/
	@echo "Plugin installed to $(USER_PLUGIN_DIR)/$(PLUGIN_FILE)"

install-system: build
	@echo "Installing plugin system-wide..."
	@mkdir -p $(SYSTEM_PLUGIN_DIR)
	@cp $(PLUGIN_FILE) $(SYSTEM_PLUGIN_DIR)/
	@echo "Plugin installed to $(SYSTEM_PLUGIN_DIR)/$(PLUGIN_FILE)"

uninstall:
	@echo "Uninstalling plugin..."
	@rm -f $(USER_PLUGIN_DIR)/$(PLUGIN_FILE)
	@rm -f $(SYSTEM_PLUGIN_DIR)/$(PLUGIN_FILE)
	@echo "Plugin uninstalled"

test:
	@echo "Running tests..."
	@go test -v ./...

test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

lint:
	@echo "Running linter..."
	@golangci-lint run

# Validate plugin exports
validate: build
	@echo "Validating plugin exports..."
	@go tool nm $(PLUGIN_FILE) | grep PluginInfo || (echo "ERROR: PluginInfo not found" && exit 1)
	@go tool nm $(PLUGIN_FILE) | grep NewProvider || (echo "ERROR: NewProvider not found" && exit 1)
	@echo "Plugin exports validated successfully"

# Development helpers
dev-install: build
	@echo "Installing for development..."
	@mkdir -p ./testplugins
	@cp $(PLUGIN_FILE) ./testplugins/
	@echo "Plugin installed to ./testplugins/"

help:
	@echo "DigitalOcean Provider Plugin Build System"
	@echo ""
	@echo "Usage:"
	@echo "  make build             Build the plugin"
	@echo "  make install           Install to user directory (~/.hypersdk/plugins)"
	@echo "  make install-system    Install system-wide (/usr/local/lib/hypersdk/plugins)"
	@echo "  make uninstall         Remove installed plugin"
	@echo "  make clean             Remove built files"
	@echo "  make test              Run unit tests"
	@echo "  make test-integration  Run integration tests"
	@echo "  make validate          Validate plugin exports"
	@echo "  make lint              Run linter"
	@echo "  make dev-install       Install to local test directory"
