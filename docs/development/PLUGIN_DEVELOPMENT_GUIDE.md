# HyperSDK Plugin Development Guide

Complete guide for developing provider plugins for HyperSDK.

## Table of Contents

1. [Introduction](#introduction)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Plugin Structure](#plugin-structure)
5. [Provider Interface](#provider-interface)
6. [Building and Testing](#building-and-testing)
7. [Best Practices](#best-practices)
8. [Examples](#examples)
9. [Publishing](#publishing)

## Introduction

HyperSDK supports dynamic provider plugins that can be loaded at runtime without recompiling the daemon. This enables:

- **Community Contributions** - Add support for new cloud providers
- **Custom Integrations** - Internal/private cloud platforms
- **Rapid Development** - Test and iterate without daemon restart
- **Modularity** - Keep provider code separate from core

## Prerequisites

### Development Environment

- **Go 1.24+** - Must match hypervisord's Go version exactly
- **Linux** - Go plugins are Linux-only (macOS/Windows not supported)
- **Build Tools** - make, git, etc.

### Knowledge Requirements

- Go programming language
- Cloud provider APIs
- VM/virtualization concepts

### Check Go Version Compatibility

```bash
# Check hypervisord Go version
hypervisord --version

# Check your Go version
go version

# They MUST match exactly for plugins to load
```

## Quick Start

### 1. Create Plugin Project

```bash
mkdir -p myprovider
cd myprovider

# Initialize Go module
go mod init hypersdk/plugins/myprovider
```

### 2. Create Entry Point (`main.go`)

```go
package main

import (
    "hypersdk/logger"
    "hypersdk/providers"
    "hypersdk/providers/plugin"
)

// Plugin metadata - MUST be exported
var PluginInfo = plugin.Metadata{
    Name:         "myprovider",
    Version:      "1.0.0",
    Description:  "My Custom Provider",
    Author:       "Your Name",
    License:      "LGPL-3.0-or-later",
    ProviderType: providers.ProviderType("myprovider"),
    Capabilities: providers.ExportCapabilities{
        SupportedFormats: []string{"qcow2", "raw"},
        SupportsCompression: true,
    },
}

// Provider factory - MUST be exported
func NewProvider(config providers.ProviderConfig, log logger.Logger) (providers.Provider, error) {
    return &MyProvider{
        config: config,
        logger: log,
    }, nil
}
```

### 3. Implement Provider (`provider.go`)

```go
package main

import (
    "context"
    "hypersdk/logger"
    "hypersdk/providers"
)

type MyProvider struct {
    config providers.ProviderConfig
    logger logger.Logger
}

func (p *MyProvider) Name() string {
    return "MyProvider"
}

func (p *MyProvider) Type() providers.ProviderType {
    return providers.ProviderType("myprovider")
}

// Implement remaining Provider interface methods...
```

### 4. Build Plugin

```bash
go build -buildmode=plugin -o myprovider.so
```

### 5. Install Plugin

```bash
mkdir -p ~/.hypersdk/plugins
cp myprovider.so ~/.hypersdk/plugins/
```

### 6. Restart Daemon

```bash
systemctl restart hypervisord
# Or
hypervisord
```

## Plugin Structure

### Recommended Project Layout

```
myprovider/
├── go.mod              # Go module definition
├── main.go             # Plugin entry point
├── provider.go         # Provider implementation
├── export.go           # Export logic
├── client.go           # API client
├── types.go            # Data structures
├── Makefile            # Build automation
├── README.md           # Documentation
├── LICENSE             # License file
└── test/
    ├── provider_test.go
    └── integration_test.go
```

### Required Files

1. **go.mod** - Module definition
   ```go
   module hypersdk/plugins/myprovider

   go 1.24

   require (
       hypersdk v0.0.1
   )

   replace hypersdk => ../..
   ```

2. **main.go** - Plugin entry with exports
   ```go
   package main

   var PluginInfo = plugin.Metadata{...}
   func NewProvider(...) (providers.Provider, error) {...}
   ```

3. **provider.go** - Provider implementation
   ```go
   package main

   type MyProvider struct {...}
   // Implement Provider interface
   ```

## Provider Interface

Your provider must implement the `providers.Provider` interface:

```go
type Provider interface {
    // Identity
    Name() string
    Type() ProviderType

    // Connection lifecycle
    Connect(ctx context.Context, config ProviderConfig) error
    Disconnect() error
    ValidateCredentials(ctx context.Context) error

    // VM Discovery
    ListVMs(ctx context.Context, filter VMFilter) ([]*VMInfo, error)
    GetVM(ctx context.Context, identifier string) (*VMInfo, error)
    SearchVMs(ctx context.Context, query string) ([]*VMInfo, error)

    // VM Export
    ExportVM(ctx context.Context, identifier string, opts ExportOptions) (*ExportResult, error)
    GetExportCapabilities() ExportCapabilities
}
```

### Method Implementation Guide

#### Name() and Type()

```go
func (p *MyProvider) Name() string {
    return "MyCloudProvider"
}

func (p *MyProvider) Type() providers.ProviderType {
    return providers.ProviderType("mycloud")
}
```

#### Connect()

Establish connection to provider API:

```go
func (p *MyProvider) Connect(ctx context.Context, config providers.ProviderConfig) error {
    // 1. Extract credentials from config
    apiKey := config.Metadata["api_key"].(string)

    // 2. Initialize API client
    p.client = mycloud.NewClient(apiKey)

    // 3. Validate connection
    if err := p.ValidateCredentials(ctx); err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }

    p.logger.Info("connected successfully")
    return nil
}
```

#### ListVMs()

List VMs with filtering:

```go
func (p *MyProvider) ListVMs(ctx context.Context, filter providers.VMFilter) ([]*providers.VMInfo, error) {
    // 1. Call provider API
    instances, err := p.client.ListInstances(ctx)
    if err != nil {
        return nil, err
    }

    // 2. Convert to VMInfo format
    var vms []*providers.VMInfo
    for _, inst := range instances {
        // Apply filters
        if filter.NamePattern != "" {
            if !matchPattern(inst.Name, filter.NamePattern) {
                continue
            }
        }

        vm := &providers.VMInfo{
            Provider:    p.Type(),
            ID:          inst.ID,
            Name:        inst.Name,
            State:       inst.Status,
            PowerState:  inst.PowerState,
            MemoryMB:    inst.RAM,
            NumCPUs:     inst.CPUs,
            IPAddresses: inst.IPs,
        }
        vms = append(vms, vm)
    }

    return vms, nil
}
```

#### ExportVM()

Export a VM:

```go
func (p *MyProvider) ExportVM(ctx context.Context, identifier string, opts providers.ExportOptions) (*providers.ExportResult, error) {
    p.logger.Info("exporting VM", "id", identifier)

    // 1. Get VM info
    vm, err := p.GetVM(ctx, identifier)
    if err != nil {
        return nil, err
    }

    // 2. Create snapshot
    snapshot, err := p.client.CreateSnapshot(ctx, identifier)
    if err != nil {
        return nil, err
    }

    // 3. Download snapshot
    outputFile := filepath.Join(opts.OutputPath, vm.Name+"."+opts.Format)
    if err := p.downloadSnapshot(ctx, snapshot.ID, outputFile); err != nil {
        return nil, err
    }

    // 4. Calculate checksum
    checksum, err := calculateSHA256(outputFile)
    if err != nil {
        return nil, err
    }

    // 5. Return result
    return &providers.ExportResult{
        Provider:   p.Type(),
        VMName:     vm.Name,
        VMID:       identifier,
        Format:     opts.Format,
        OutputPath: outputFile,
        Files:      []string{outputFile},
        Checksum:   checksum,
    }, nil
}
```

## Building and Testing

### Build Command

```bash
go build -buildmode=plugin -o myprovider.so .
```

### Build Flags

- `-buildmode=plugin` - Required for plugin mode
- `-ldflags="-s -w"` - Strip debug info (optional, smaller file)
- `-tags=production` - Build tags (optional)

### Unit Testing

```go
package main

import (
    "context"
    "testing"

    "hypersdk/providers"
)

func TestProviderName(t *testing.T) {
    provider := &MyProvider{}
    if provider.Name() != "MyProvider" {
        t.Errorf("unexpected name: %s", provider.Name())
    }
}

func TestConnect(t *testing.T) {
    config := providers.ProviderConfig{
        Metadata: map[string]interface{}{
            "api_key": "test-key",
        },
    }

    provider := &MyProvider{}
    err := provider.Connect(context.Background(), config)
    if err != nil {
        t.Fatalf("connect failed: %v", err)
    }
}
```

Run tests:
```bash
go test -v ./...
```

### Integration Testing

```go
//go:build integration
package main

import (
    "context"
    "os"
    "testing"
)

func TestRealAPI(t *testing.T) {
    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        t.Skip("API_KEY not set")
    }

    config := providers.ProviderConfig{
        Metadata: map[string]interface{}{
            "api_key": apiKey,
        },
    }

    provider := &MyProvider{}
    if err := provider.Connect(context.Background(), config); err != nil {
        t.Fatal(err)
    }

    vms, err := provider.ListVMs(context.Background(), providers.VMFilter{})
    if err != nil {
        t.Fatal(err)
    }

    t.Logf("Found %d VMs", len(vms))
}
```

Run integration tests:
```bash
export API_KEY=your_key
go test -v -tags=integration ./...
```

### Validation

Verify plugin exports:

```bash
go tool nm myprovider.so | grep -E '(PluginInfo|NewProvider)'
```

Expected output:
```
xxxxxx T main.NewProvider
xxxxxx D main.PluginInfo
```

## Best Practices

### 1. Error Handling

Always return descriptive errors:

```go
func (p *MyProvider) Connect(ctx context.Context, config providers.ProviderConfig) error {
    apiKey, ok := config.Metadata["api_key"].(string)
    if !ok {
        return fmt.Errorf("missing required configuration: api_key")
    }

    if apiKey == "" {
        return fmt.Errorf("api_key cannot be empty")
    }

    client := mycloud.NewClient(apiKey)
    if err := client.Ping(ctx); err != nil {
        return fmt.Errorf("failed to connect to API: %w", err)
    }

    return nil
}
```

### 2. Logging

Use structured logging:

```go
p.logger.Info("starting export",
    "vm_id", identifier,
    "format", opts.Format,
    "output", opts.OutputPath)

p.logger.Debug("downloading file",
    "url", downloadURL,
    "size_mb", sizeMB)

p.logger.Warn("retrying request",
    "attempt", attempt,
    "error", err)

p.logger.Error("export failed",
    "vm_id", identifier,
    "error", err)
```

### 3. Context Handling

Respect context cancellation:

```go
func (p *MyProvider) ExportVM(ctx context.Context, id string, opts ExportOptions) (*ExportResult, error) {
    // Check context before long operations
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Pass context to API calls
    snapshot, err := p.client.CreateSnapshot(ctx, id)
    if err != nil {
        return nil, err
    }

    // Check again
    select {
    case <-ctx.Done():
        p.client.DeleteSnapshot(context.Background(), snapshot.ID)
        return nil, ctx.Err()
    default:
    }

    return result, nil
}
```

### 4. Resource Cleanup

Always clean up resources:

```go
func (p *MyProvider) ExportVM(ctx context.Context, id string, opts ExportOptions) (*ExportResult, error) {
    snapshot, err := p.client.CreateSnapshot(ctx, id)
    if err != nil {
        return nil, err
    }

    // Ensure snapshot cleanup
    defer func() {
        if err := p.client.DeleteSnapshot(context.Background(), snapshot.ID); err != nil {
            p.logger.Warn("failed to cleanup snapshot",
                "snapshot_id", snapshot.ID,
                "error", err)
        }
    }()

    // ... rest of export logic
}
```

### 5. Progress Reporting

Report progress for long operations:

```go
func (p *MyProvider) downloadFile(ctx context.Context, url, dest string, progress providers.ProgressReporter) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    file, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer file.Close()

    totalSize := resp.ContentLength
    var downloaded int64

    buf := make([]byte, 32*1024)
    for {
        n, err := resp.Body.Read(buf)
        if n > 0 {
            file.Write(buf[:n])
            downloaded += int64(n)

            // Report progress
            percent := float64(downloaded) / float64(totalSize) * 100
            progress.Update("downloading", percent, fmt.Sprintf("%d/%d bytes", downloaded, totalSize))
        }

        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }

    return nil
}
```

## Examples

See complete examples in:

- [`examples/plugins/digitalocean/`](../../examples/plugins/digitalocean/) - DigitalOcean provider
- More examples coming soon...

## Publishing

### 1. Documentation

Create comprehensive README with:
- Installation instructions
- Configuration examples
- API token requirements
- Supported features
- Troubleshooting guide

### 2. Versioning

Follow Semantic Versioning:
- `1.0.0` - Initial release
- `1.1.0` - New features
- `1.0.1` - Bug fixes
- `2.0.0` - Breaking changes

### 3. Distribution

Options for distributing plugins:

**GitHub Releases:**
```bash
# Tag version
git tag v1.0.0
git push origin v1.0.0

# Create release with plugin binary
gh release create v1.0.0 myprovider.so
```

**Package Repository:**
```bash
# Create DEB/RPM package
fpm -s dir -t deb -n hypersdk-myprovider -v 1.0.0 \
    --prefix /usr/local/lib/hypersdk/plugins \
    myprovider.so
```

### 4. Registry (Future)

A central plugin registry is planned for:
- Discovery
- Automated installation
- Version management
- Ratings and reviews

## Troubleshooting

### Plugin Not Loading

**Symptom:** Plugin file exists but doesn't load

**Solutions:**
1. Check Go version compatibility
2. Verify exports: `go tool nm plugin.so | grep -E '(PluginInfo|NewProvider)'`
3. Check logs: `journalctl -u hypervisord | grep plugin`
4. Validate metadata

### Symbol Not Found

**Symptom:** `symbol not found` error

**Solutions:**
1. Ensure exports are capitalized (exported)
2. Check function signatures match exactly
3. Rebuild with same Go version as daemon

### Version Mismatch

**Symptom:** `plugin was built with a different version of package`

**Solutions:**
1. Check Go versions: `go version` vs `hypervisord --version`
2. Rebuild plugin with matching version
3. Use exact same `go.mod` dependencies

## Support

- **Documentation:** [Plugin System Guide](./PLUGIN_SYSTEM.md)
- **Examples:** [`examples/plugins/`](../../examples/plugins/)
- **Issues:** [GitHub Issues](https://github.com/ssahani/hypersdk/issues)
- **Discussions:** [GitHub Discussions](https://github.com/ssahani/hypersdk/discussions)

## License

Plugins can use any license compatible with LGPL-3.0-or-later.

Recommended licenses:
- LGPL-3.0-or-later
- MIT
- Apache-2.0
