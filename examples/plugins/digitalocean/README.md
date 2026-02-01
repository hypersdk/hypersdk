# DigitalOcean Provider Plugin Example

This is an example provider plugin for HyperSDK that demonstrates how to build a custom provider plugin.

## Features

- List DigitalOcean Droplets
- Get Droplet information
- Export Droplets to QCOW2/RAW format
- Search Droplets

## Building

```bash
make build
```

This creates `digitalocean.so` plugin file.

## Installing

```bash
# Install to user directory
make install

# Or install system-wide
sudo make install-system
```

## Configuration

Add your DigitalOcean API token to the HyperSDK configuration:

```yaml
# config.yaml
plugins:
  enabled: true
  directories:
    - ~/.hypersdk/plugins

providers:
  digitalocean:
    token: your_api_token_here
```

## Usage

### CLI

```bash
# List droplets
hyperexport list --provider digitalocean

# Export a droplet
hyperexport export \
  --provider digitalocean \
  --vm-id 123456 \
  --output /exports \
  --format qcow2
```

### API

```bash
# Submit export job
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "123456",
    "output_dir": "/exports",
    "vcenter": {
      "server": "api.digitalocean.com",
      "username": "token",
      "password": "your_api_token"
    },
    "format": "qcow2"
  }'
```

### Python SDK

```python
from hypersdk import HyperSDK, JobDefinition

client = HyperSDK("http://localhost:8080")

job_id = client.submit_job(JobDefinition(
    vm_path="123456",  # Droplet ID
    output_dir="/exports",
    vcenter={
        "server": "api.digitalocean.com",
        "username": "token",
        "password": "your_api_token"
    },
    format="qcow2"
))

print(f"Job submitted: {job_id}")
```

## Development

### Project Structure

```
digitalocean/
├── main.go       # Plugin entry point (exports PluginInfo and NewProvider)
├── provider.go   # Provider implementation
├── go.mod        # Go module definition
├── Makefile      # Build automation
└── README.md     # This file
```

### Required Exports

Every plugin MUST export these symbols:

1. **PluginInfo** - Plugin metadata
   ```go
   var PluginInfo = plugin.Metadata{
       Name: "digitalocean",
       Version: "1.0.0",
       // ...
   }
   ```

2. **NewProvider** - Provider factory function
   ```go
   func NewProvider(config providers.ProviderConfig, log logger.Logger) (providers.Provider, error)
   ```

### Implementing the Provider Interface

Your provider must implement all methods from `providers.Provider`:

```go
type Provider interface {
    Name() string
    Type() ProviderType
    Connect(ctx context.Context, config ProviderConfig) error
    Disconnect() error
    ValidateCredentials(ctx context.Context) error
    ListVMs(ctx context.Context, filter VMFilter) ([]*VMInfo, error)
    GetVM(ctx context.Context, identifier string) (*VMInfo, error)
    SearchVMs(ctx context.Context, query string) ([]*VMInfo, error)
    ExportVM(ctx context.Context, identifier string, opts ExportOptions) (*ExportResult, error)
    GetExportCapabilities() ExportCapabilities
}
```

### Testing

```bash
# Unit tests
go test -v

# Integration test (requires actual API token)
export DO_TOKEN=your_token
go test -v -tags=integration
```

## API Reference

### Configuration Fields

The plugin expects these fields in `config.Metadata`:

- `token` (required) - DigitalOcean API token
- `endpoint` (optional) - API endpoint (default: https://api.digitalocean.com/v2)

### VM Identifier Format

Use the Droplet ID as the VM identifier:
- Format: Numeric ID (e.g., "123456")
- Get IDs from `doctl compute droplet list`

## Troubleshooting

### Plugin Not Loading

Check plugin is in the correct directory:
```bash
ls -la ~/.hypersdk/plugins/digitalocean.so
```

Check HyperSDK logs:
```bash
journalctl -u hypervisord | grep digitalocean
```

Validate plugin symbols:
```bash
go tool nm digitalocean.so | grep -E '(PluginInfo|NewProvider)'
```

### Authentication Errors

Test API token:
```bash
curl -H "Authorization: Bearer $DO_TOKEN" \
     https://api.digitalocean.com/v2/account
```

### Build Errors

Ensure Go version matches:
```bash
go version  # Should match hypervisord's Go version
```

## Contributing

This example plugin is meant as a template. To contribute improvements:

1. Fork the repository
2. Create your feature branch
3. Make your changes
4. Submit a pull request

## License

LGPL-3.0-or-later

## Resources

- [DigitalOcean API Documentation](https://docs.digitalocean.com/reference/api/)
- [HyperSDK Plugin Guide](../../../docs/development/PLUGIN_SYSTEM.md)
- [Provider Interface](../../../providers/provider.go)
