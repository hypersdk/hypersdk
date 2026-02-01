# HyperSDK Provider Plugin System

## Overview

The HyperSDK plugin system enables dynamic loading of provider implementations without recompiling the daemon. This allows community contributors to add support for new cloud providers, hypervisors, and virtualization platforms through external plugins.

## Architecture

### Components

1. **Plugin Loader** (`providers/plugin/loader.go`)
   - Discovers plugin files (`.so` shared libraries)
   - Loads plugins using Go's `plugin` package
   - Validates plugin interfaces
   - Manages plugin lifecycle

2. **Plugin Manager** (`providers/plugin/manager.go`)
   - Coordinates plugin loading and registration
   - Maintains plugin metadata
   - Handles hot-reload and unloading
   - Provides plugin health monitoring

3. **Plugin Interface** (`providers/plugin/interface.go`)
   - Defines the contract that plugins must implement
   - Export required symbols for plugin discovery
   - Version compatibility checking

4. **Configuration** (`config/plugins.go`)
   - Plugin directory configuration
   - Enable/disable specific plugins
   - Plugin-specific settings

### Plugin Structure

Each provider plugin must be a Go shared library (`.so`) that exports:

```go
// Required exports:
var PluginInfo PluginMetadata
var NewProvider ProviderFactory
```

**PluginMetadata** contains:
- Name and version
- Provider type
- Supported features
- Dependencies
- Author information

**ProviderFactory** is a function that creates provider instances:
```go
func NewProvider(config providers.ProviderConfig, logger logger.Logger) (providers.Provider, error)
```

## Plugin Discovery

Plugins are discovered through:

1. **Static directories**:
   - `/usr/local/lib/hypersdk/plugins/`
   - `$HOME/.hypersdk/plugins/`
   - `./plugins/` (relative to working directory)

2. **Configuration file**:
   ```yaml
   plugins:
     directories:
       - /custom/plugin/path
     enabled:
       - digitalocean
       - linode
     hot_reload: true
   ```

3. **Environment variable**:
   ```bash
   export HYPERSDK_PLUGIN_PATH=/path/to/plugins:/another/path
   ```

## Plugin Loading Sequence

1. **Discovery Phase**:
   - Scan configured directories
   - Find all `.so` files
   - Filter by enable/disable lists

2. **Validation Phase**:
   - Load plugin symbols
   - Check for required exports
   - Validate PluginInfo
   - Verify version compatibility

3. **Registration Phase**:
   - Call plugin's NewProvider factory
   - Register with provider registry
   - Store plugin metadata

4. **Initialization Phase**:
   - Plugin performs setup
   - Validate credentials format
   - Initialize SDK clients

## Hot Reload

Hot reload allows loading/unloading plugins without restarting the daemon:

### Watch Mechanism
- File system watcher on plugin directories
- Detect new `.so` files
- Detect plugin removal

### Reload Strategy
1. **Add Plugin**: Load and register immediately
2. **Update Plugin**:
   - Unregister old version
   - Wait for active connections to drain
   - Load new version
   - Re-register
3. **Remove Plugin**:
   - Mark as deprecated
   - Reject new connections
   - Unload when no active usage

### Safety
- Version locking for active jobs
- Graceful degradation
- Rollback on load failure

## Plugin Development

### Creating a Plugin

1. **Project Structure**:
```
digitalocean-provider/
├── go.mod
├── main.go          # Plugin entry point
├── provider.go      # Provider implementation
├── export.go        # Export logic
└── README.md
```

2. **Entry Point** (`main.go`):
```go
package main

import (
    "hypersdk/logger"
    "hypersdk/providers"
    "hypersdk/providers/plugin"
)

// Plugin metadata
var PluginInfo = plugin.Metadata{
    Name:        "digitalocean",
    Version:     "1.0.0",
    ProviderType: "digitalocean",
    Description: "DigitalOcean Droplet provider",
    Author:      "Community",
    Capabilities: providers.ExportCapabilities{
        SupportedFormats: []string{"raw", "qcow2"},
        SupportsCompression: true,
        SupportsStreaming: true,
    },
}

// Provider factory
func NewProvider(config providers.ProviderConfig, log logger.Logger) (providers.Provider, error) {
    return &DigitalOceanProvider{
        config: config,
        logger: log,
    }, nil
}
```

3. **Provider Implementation** (`provider.go`):
```go
package main

import (
    "context"
    "hypersdk/logger"
    "hypersdk/providers"
)

type DigitalOceanProvider struct {
    config providers.ProviderConfig
    logger logger.Logger
    client *digitalocean.Client
}

func (p *DigitalOceanProvider) Name() string {
    return "DigitalOcean"
}

func (p *DigitalOceanProvider) Type() providers.ProviderType {
    return providers.ProviderType("digitalocean")
}

func (p *DigitalOceanProvider) Connect(ctx context.Context, config providers.ProviderConfig) error {
    // Initialize DigitalOcean client
    token := config.Metadata["token"].(string)
    p.client = digitalocean.NewClient(token)
    return nil
}

// ... implement remaining Provider interface methods
```

4. **Build Plugin**:
```bash
go build -buildmode=plugin -o digitalocean.so
```

5. **Install Plugin**:
```bash
cp digitalocean.so ~/.hypersdk/plugins/
# Or
cp digitalocean.so /usr/local/lib/hypersdk/plugins/
```

### Testing Plugins

1. **Unit Tests**:
```go
package main

import (
    "testing"
    "hypersdk/providers"
)

func TestProviderInterface(t *testing.T) {
    config := providers.ProviderConfig{
        Metadata: map[string]interface{}{
            "token": "test-token",
        },
    }

    provider, err := NewProvider(config, nil)
    if err != nil {
        t.Fatalf("failed to create provider: %v", err)
    }

    if provider.Name() != "DigitalOcean" {
        t.Errorf("unexpected provider name: %s", provider.Name())
    }
}
```

2. **Integration Tests**:
```bash
# Load plugin in test daemon
go test -tags=integration ./test/plugin/
```

## Security Considerations

### Plugin Validation
- Verify plugin signatures (optional)
- Checksum validation
- Whitelist/blacklist mechanisms

### Sandboxing
- Plugins run in same process (Go limitation)
- Resource limits via cgroups (external)
- Audit logging for plugin actions

### Access Control
- Plugins inherit daemon permissions
- No privilege escalation
- Credential isolation per plugin

## Configuration Examples

### Minimal Configuration
```yaml
# config.yaml
plugins:
  enabled: true
  directories:
    - ./plugins
```

### Advanced Configuration
```yaml
plugins:
  enabled: true
  hot_reload: true
  directories:
    - /usr/local/lib/hypersdk/plugins
    - /home/user/.hypersdk/plugins
  enabled:
    - digitalocean
    - linode
    - vultr
  disabled:
    - legacy-provider

  # Plugin-specific config
  providers:
    digitalocean:
      timeout: 30s
      max_concurrent_exports: 5
```

## API Integration

### REST API Endpoints

**List available providers** (including plugins):
```
GET /api/providers
```

**Get provider capabilities**:
```
GET /api/providers/{type}/capabilities
```

**Reload plugins**:
```
POST /api/plugins/reload
```

**List loaded plugins**:
```
GET /api/plugins
```

Response:
```json
{
  "plugins": [
    {
      "name": "digitalocean",
      "version": "1.0.0",
      "type": "digitalocean",
      "status": "loaded",
      "loaded_at": "2026-02-03T10:30:00Z",
      "path": "/usr/local/lib/hypersdk/plugins/digitalocean.so"
    }
  ]
}
```

## Troubleshooting

### Plugin Not Loading

**Check logs**:
```bash
journalctl -u hypervisord -f | grep plugin
```

**Validate plugin**:
```bash
go tool nm digitalocean.so | grep PluginInfo
go tool nm digitalocean.so | grep NewProvider
```

**Test plugin symbols**:
```go
package main

import (
    "fmt"
    "plugin"
)

func main() {
    p, err := plugin.Open("digitalocean.so")
    if err != nil {
        panic(err)
    }

    info, err := p.Lookup("PluginInfo")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Plugin: %+v\n", info)
}
```

### Version Incompatibility

**Issue**: Plugin compiled with different Go version
**Solution**: Recompile with same Go version as daemon

**Check Go version**:
```bash
hypervisord --version  # Shows Go version used
go version             # Current Go version
```

### Symbol Not Found

**Issue**: Missing required exports
**Solution**: Ensure both PluginInfo and NewProvider are exported

```go
// Must be exported (capitalized)
var PluginInfo = ...    // ✓
func NewProvider(...) { // ✓

// Won't work (not exported)
var pluginInfo = ...    // ✗
func newProvider(...) { // ✗
```

## Performance Considerations

### Plugin Loading Time
- Plugins loaded at startup: ~10-50ms per plugin
- Hot reload: ~20-100ms per plugin
- Minimal impact on running jobs

### Memory Overhead
- Plugin metadata: ~1-5 KB per plugin
- Shared library: ~5-50 MB per plugin (depends on dependencies)
- No runtime overhead after loading

### Concurrent Access
- Thread-safe plugin registry
- Multiple goroutines can use same plugin
- Provider instances are separate

## Future Enhancements

1. **Plugin Marketplace**
   - Central registry of community plugins
   - Automated discovery and installation
   - Rating and review system

2. **Plugin SDK**
   - Helper libraries for common operations
   - Testing framework
   - Code generation tools

3. **Advanced Features**
   - Plugin dependencies
   - Plugin communication (inter-plugin)
   - Plugin hooks and events

4. **Monitoring**
   - Plugin performance metrics
   - Usage statistics
   - Error tracking

## References

- [Go Plugin Package](https://pkg.go.dev/plugin)
- [Provider Interface](../providers/provider.go)
- [Plugin Examples](../examples/plugins/)
- [Contributing Guide](../CONTRIBUTING.md)
