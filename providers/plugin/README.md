# Provider Plugin System

Dynamic provider loading for HyperSDK.

## Overview

The plugin system allows HyperSDK to load provider implementations at runtime without recompiling the daemon. This enables:

✅ **Community Contributions** - Add support for new cloud providers
✅ **Rapid Development** - Test and iterate without daemon restart
✅ **Modularity** - Keep provider code separate from core
✅ **Hot Reload** - Load new plugins without downtime

## Quick Start

### For Users

1. **Download a plugin**:
   ```bash
   wget https://github.com/user/provider-plugin/releases/download/v1.0.0/myprovider.so
   ```

2. **Install the plugin**:
   ```bash
   mkdir -p ~/.hypersdk/plugins
   cp myprovider.so ~/.hypersdk/plugins/
   ```

3. **Enable hot-reload** (optional):
   ```yaml
   # config.yaml
   plugins:
     enabled: true
     hot_reload: true
   ```

4. **Restart daemon**:
   ```bash
   systemctl restart hypervisord
   ```

### For Developers

1. **Create plugin project**:
   ```bash
   mkdir myprovider && cd myprovider
   go mod init hypersdk/plugins/myprovider
   ```

2. **Implement provider**:
   ```go
   // main.go
   package main

   import (
       "hypersdk/logger"
       "hypersdk/providers"
       "hypersdk/providers/plugin"
   )

   var PluginInfo = plugin.Metadata{
       Name: "myprovider",
       Version: "1.0.0",
       ProviderType: "myprovider",
   }

   func NewProvider(config providers.ProviderConfig, log logger.Logger) (providers.Provider, error) {
       return &MyProvider{}, nil
   }
   ```

3. **Build plugin**:
   ```bash
   go build -buildmode=plugin -o myprovider.so
   ```

4. **Test locally**:
   ```bash
   mkdir -p ~/.hypersdk/plugins
   cp myprovider.so ~/.hypersdk/plugins/
   ```

## Architecture

### Components

- **Loader** (`loader.go`) - Discovers and loads `.so` files
- **Manager** (`manager.go`) - Manages plugin lifecycle
- **Watcher** (`watcher.go`) - Monitors files for hot-reload
- **Metadata** (`metadata.go`) - Plugin information structures

### Plugin Discovery

Plugins are discovered in:
1. `/usr/local/lib/hypersdk/plugins/`
2. `/usr/lib/hypersdk/plugins/`
3. `~/.hypersdk/plugins/`
4. `./plugins/` (current directory)
5. `$HYPERSDK_PLUGIN_PATH` (environment variable)

### Required Exports

Every plugin must export:

1. **PluginInfo** - Metadata
   ```go
   var PluginInfo = plugin.Metadata{
       Name: "myprovider",
       Version: "1.0.0",
       ProviderType: "myprovider",
       Capabilities: providers.ExportCapabilities{...},
   }
   ```

2. **NewProvider** - Factory function
   ```go
   func NewProvider(config providers.ProviderConfig, log logger.Logger) (providers.Provider, error)
   ```

## Configuration

```yaml
# config.yaml
plugins:
  enabled: true
  hot_reload: true
  directories:
    - /usr/local/lib/hypersdk/plugins
    - ~/.hypersdk/plugins
  enabled:
    - digitalocean
    - linode
  disabled:
    - legacy-provider
```

## API Endpoints

### List Plugins

```bash
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
      "loaded_at": "2026-02-03T10:30:00Z"
    }
  ]
}
```

### Reload Plugins

```bash
POST /api/plugins/reload
```

### Get Plugin Info

```bash
GET /api/plugins/{name}
```

## Documentation

- **[Plugin System Architecture](../../docs/development/PLUGIN_SYSTEM.md)** - Detailed design
- **[Plugin Development Guide](../../docs/development/PLUGIN_DEVELOPMENT_GUIDE.md)** - How to build plugins
- **[Example Plugin](../../examples/plugins/digitalocean/)** - DigitalOcean provider

## Examples

- [DigitalOcean Provider](../../examples/plugins/digitalocean/) - Complete working example

## Troubleshooting

### Plugin Not Loading

Check logs:
```bash
journalctl -u hypervisord | grep plugin
```

Validate exports:
```bash
go tool nm myprovider.so | grep -E '(PluginInfo|NewProvider)'
```

### Version Mismatch

Ensure Go versions match:
```bash
hypervisord --version  # Shows Go version
go version             # Your Go version
```

Must match exactly for plugins to load.

## Development

### Running Tests

```bash
go test -v ./...
```

### Building

```bash
go build
```

## Contributing

See [Plugin Development Guide](../../docs/development/PLUGIN_DEVELOPMENT_GUIDE.md) for details on creating plugins.

## License

LGPL-3.0-or-later
