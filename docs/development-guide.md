# Development Guide

## Table of Contents

1. [Development Environment Setup](#development-environment-setup)
2. [Building from Source](#building-from-source)
3. [Project Structure](#project-structure)
4. [Code Style](#code-style)
5. [Testing](#testing)
6. [Contributing](#contributing)
7. [Release Process](#release-process)

## Development Environment Setup

### Prerequisites

```bash
# Install Go 1.24+
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### Clone Repository

```bash
# Clone repo
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Install dependencies
go mod download
go mod tidy
```

### IDE Setup

#### VSCode

```bash
# Install Go extension
code --install-extension golang.go

# Install tools
go install golang.org/x/tools/gopls@latest
go install github.com/go-delve/delve/cmd/dlv@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/tools/cmd/goimports@latest
```

Create `.vscode/settings.json`:

```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace",
  "go.formatTool": "goimports",
  "editor.formatOnSave": true,
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSave": true
}
```

#### GoLand/IntelliJ

1. Open project
2. Enable Go modules integration
3. Configure GOPATH and GOROOT
4. Enable gofmt on save

## Building from Source

### Build All Binaries

```bash
# Build all
make build

# Or individually
go build -o hyperexport ./cmd/hyperexport
go build -o hypervisord ./cmd/hypervisord
go build -o hyperctl ./cmd/hyperctl
```

### Build with Debug Symbols

```bash
go build -gcflags="all=-N -l" -o hyperexport-debug ./cmd/hyperexport
```

### Build with Race Detector

```bash
go build -race -o hyperexport-race ./cmd/hyperexport
```

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o hyperexport-linux-amd64 ./cmd/hyperexport

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o hyperexport-linux-arm64 ./cmd/hyperexport

# macOS
GOOS=darwin GOARCH=amd64 go build -o hyperexport-darwin-amd64 ./cmd/hyperexport

# Windows
GOOS=windows GOARCH=amd64 go build -o hyperexport-windows-amd64.exe ./cmd/hyperexport
```

### Build RPM Package

```bash
# Install rpm-build
sudo dnf install rpm-build rpmdevtools

# Create build tree
rpmdev-setuptree

# Build RPM
rpmbuild -ba hypersdk.spec

# Output: ~/rpmbuild/RPMS/x86_64/hypersdk-0.2.0-1.fc39.x86_64.rpm
```

## Project Structure

```
hypersdk/
├── cmd/                      # Main applications
│   ├── hyperexport/          # Standalone export tool
│   │   └── main.go
│   ├── hypervisord/          # Daemon service
│   │   └── main.go
│   └── hyperctl/             # Control CLI
│       └── main.go
│
├── providers/                # Cloud provider implementations
│   ├── vsphere/              # vSphere provider
│   │   ├── client.go         # vSphere client
│   │   ├── export.go         # Export logic
│   │   ├── ova.go            # OVA creation
│   │   ├── types.go          # Data structures
│   │   └── vm_operations.go  # VM management
│   ├── aws/                  # AWS provider
│   ├── azure/                # Azure provider
│   ├── gcp/                  # GCP provider
│   └── common/               # Shared provider code
│
├── daemon/                   # Daemon implementation
│   ├── api/                  # REST API
│   │   ├── server.go         # Base server
│   │   ├── enhanced_server.go# Enhanced routes
│   │   ├── job_handlers.go   # Job endpoints
│   │   ├── libvirt_handlers.go # Libvirt endpoints
│   │   └── ...               # More handlers
│   ├── jobs/                 # Job management
│   │   └── manager.go        # Job manager
│   ├── models/               # Data models
│   ├── dashboard/            # Web dashboard
│   ├── scheduler/            # Job scheduler
│   ├── store/                # Database layer
│   └── webhooks/             # Webhook manager
│
├── config/                   # Configuration
│   └── config.go
│
├── logger/                   # Logging
│   └── logger.go
│
├── progress/                 # Progress tracking
│   └── progress.go
│
├── web/                      # Web UI
│   └── dashboard-react/      # React dashboard
│       ├── src/
│       │   ├── components/
│       │   ├── hooks/
│       │   └── types/
│       ├── package.json
│       └── vite.config.ts
│
├── docs/                     # Documentation
│   ├── API_ENDPOINTS.md
│   ├── getting-started.md
│   └── ...
│
├── scripts/                  # Utility scripts
│   └── test-api.sh
│
├── go.mod                    # Go modules
├── go.sum                    # Dependency checksums
├── Makefile                  # Build automation
├── hypersdk.spec             # RPM spec file
├── hypervisord.service       # Systemd unit
└── README.md                 # Main readme
```

## Code Style

### Go Standards

Follow official Go style guide and conventions:

```bash
# Format code
go fmt ./...
gofmt -s -w .

# Or use goimports
goimports -w .

# Lint code
golangci-lint run

# Vet code
go vet ./...
```

### Naming Conventions

```go
// Exported functions: PascalCase
func ExportVM(ctx context.Context, vmPath string) error

// Unexported functions: camelCase
func downloadFile(url string, dest string) error

// Constants: PascalCase
const MaxRetries = 3

// Variables: camelCase
var jobManager *Manager

// Interfaces: PascalCase with -er suffix
type Exporter interface {
    Export(ctx context.Context) error
}

// Struct names: PascalCase
type ExportOptions struct {
    Format   string
    Compress bool
}
```

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to export VM: %w", err)
}

// Don't ignore errors
_, err := doSomething()
if err != nil {
    return err
}

// Use named returns for complex functions
func processExport(vm string) (result *ExportResult, err error) {
    defer func() {
        if err != nil {
            log.Error("export failed", "vm", vm, "error", err)
        }
    }()

    // ... implementation
    return result, nil
}
```

### Logging

```go
// Use structured logging
log.Info("starting export", "vm", vmPath, "format", options.Format)
log.Error("export failed", "vm", vmPath, "error", err)
log.Debug("download progress", "file", filename, "bytes", bytesRead)

// Don't log and return error (choose one)
// Bad:
log.Error("failed", "error", err)
return err

// Good:
return fmt.Errorf("failed to process: %w", err)
```

### Comments

```go
// Package-level comment
// Package vsphere provides VMware vSphere integration.
package vsphere

// Exported function documentation
// ExportVM exports a virtual machine from vSphere to OVF/OVA format.
// It returns the path to the exported files or an error.
//
// Example:
//   result, err := ExportVM(ctx, "/datacenter/vm/myvm", opts)
//   if err != nil {
//       log.Fatal(err)
//   }
func ExportVM(ctx context.Context, vmPath string, opts *ExportOptions) (*ExportResult, error) {
    // Implementation
}

// Unexported functions: brief comment
// downloadFile downloads a file from URL to destination path
func downloadFile(url, dest string) error {
    // Implementation
}
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package
go test ./providers/vsphere/

# Run specific test
go test -run TestExportVM ./providers/vsphere/

# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...
```

### Writing Tests

```go
package vsphere

import (
    "context"
    "testing"
)

func TestExportVM(t *testing.T) {
    tests := []struct {
        name    string
        vmPath  string
        want    *ExportResult
        wantErr bool
    }{
        {
            name:    "valid VM",
            vmPath:  "/datacenter/vm/test",
            want:    &ExportResult{VMName: "test"},
            wantErr: false,
        },
        {
            name:    "invalid VM path",
            vmPath:  "",
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ExportVM(context.Background(), tt.vmPath, nil)
            if (err != nil) != tt.wantErr {
                t.Errorf("ExportVM() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ExportVM() = %v, want %v", got, tt.want)
            }
        })
    }
}

// Test helpers
func TestMain(m *testing.M) {
    // Setup
    setup()

    // Run tests
    code := m.Run()

    // Teardown
    teardown()

    os.Exit(code)
}
```

### Integration Tests

```go
// +build integration

package integration

func TestFullExportWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Integration test implementation
}
```

```bash
# Run only unit tests
go test -short ./...

# Run integration tests
go test -tags=integration ./...
```

### Benchmarks

```go
func BenchmarkExportVM(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ExportVM(context.Background(), "/datacenter/vm/test", nil)
    }
}
```

```bash
# Run benchmarks
go test -bench=. ./...

# Run with memory profiling
go test -bench=. -benchmem ./...

# Compare benchmarks
go test -bench=. -count=5 ./... > new.txt
benchstat old.txt new.txt
```

## Contributing

### Fork and Clone

```bash
# Fork on GitHub
# Clone your fork
git clone https://github.com/YOUR_USERNAME/hypersdk.git
cd hypersdk

# Add upstream remote
git remote add upstream https://github.com/ssahani/hypersdk.git

# Keep your fork synced
git fetch upstream
git checkout main
git merge upstream/main
```

### Create Feature Branch

```bash
# Create branch
git checkout -b feature/my-new-feature

# Make changes
# ...

# Commit with descriptive message
git add .
git commit -m "feat: Add support for XYZ

- Implement XYZ feature
- Add tests for XYZ
- Update documentation

Fixes #123"
```

### Commit Message Convention

```
type(scope): Subject

Body

Footer
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Code style (formatting)
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance

Examples:
```
feat(vsphere): Add OVA compression support

Implement gzip compression for OVA exports to reduce storage requirements.

Fixes #42

---

fix(api): Fix WebSocket connection issue

The response writer wrapper was breaking the http.Hijacker interface.
Now bypass middleware for /ws endpoint.

Fixes #87

---

docs(guide): Add troubleshooting section

Add common issues and solutions to help users debug problems.
```

### Pull Request

```bash
# Push to your fork
git push origin feature/my-new-feature

# Create pull request on GitHub
# Fill in the template
```

PR Template:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Performance improvement

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] Tests pass locally
- [ ] No new warnings
```

## Release Process

### Version Numbering

Follow Semantic Versioning (semver):
- MAJOR.MINOR.PATCH (e.g., 1.2.3)
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

### Creating a Release

```bash
# Update version
VERSION=0.2.0

# Tag release
git tag -a v$VERSION -m "Release v$VERSION"
git push origin v$VERSION

# Build release binaries
make release

# Create GitHub release
gh release create v$VERSION \
  --title "v$VERSION" \
  --notes "Release notes..."
  dist/*
```

### Changelog

Maintain CHANGELOG.md following [Keep a Changelog](https://keepachangelog.com):

```markdown
# Changelog

## [0.2.0] - 2026-01-20

### Added
- OVA export format support
- Compression for exports
- Connection pooling for vSphere

### Fixed
- WebSocket connection issues
- Memory leak in job manager

### Changed
- Improved export performance by 40%

## [0.1.0] - 2025-12-01

### Added
- Initial release
- vSphere VM export
- Basic web dashboard
```

## See Also

- [Contributing Guidelines](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [API Documentation](API_ENDPOINTS.md)
- [Testing Guide](testing-guide.md)
