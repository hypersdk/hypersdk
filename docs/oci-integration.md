# Oracle Cloud Infrastructure (OCI) Integration

**Version**: 1.0
**Last Updated**: 2026-01-21

---

## Overview

HyperSDK now supports Oracle Cloud Infrastructure (OCI) for both compute instance management and object storage. This integration allows you to:

- Export OCI compute instances to custom images
- Download exported images to local storage
- Store exports in OCI Object Storage
- Manage OCI resources with automatic retry and network monitoring

---

## Features

### 1. OCI Compute Integration

- **Instance Management**: List, get, start, stop OCI compute instances
- **Custom Image Creation**: Create custom images from running or stopped instances
- **Image Export**: Export custom images to OCI Object Storage or local filesystem
- **Multiple Formats**: Support for QCOW2 and VMDK export formats
- **Automatic Cleanup**: Optional deletion of temporary custom images after export

### 2. OCI Object Storage Integration

- **Upload/Download**: Full support for uploading and downloading files
- **Streaming**: Stream uploads with progress tracking
- **Bucket Operations**: List, delete, and check existence of objects
- **URL Support**: Native `oci://` URL scheme for easy configuration

### 3. Built-in Reliability

- **Automatic Retry**: Exponential backoff with jitter for all OCI operations
- **Network Monitoring**: Pause operations during network outages (optional)
- **Smart Error Detection**: Distinguish retryable vs non-retryable errors
- **Progress Reporting**: Real-time progress updates for long-running operations

---

## Configuration

### Method 1: Configuration File

Add OCI settings to your `hypersdk.yaml`:

```yaml
oci:
  # OCI Authentication
  tenancy_ocid: "ocid1.tenancy.oc1..aaaaaa..."
  user_ocid: "ocid1.user.oc1..aaaaaa..."
  fingerprint: "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
  private_key_path: "~/.oci/oci_api_key.pem"
  region: "us-phoenix-1"

  # Resource Configuration
  compartment_ocid: "ocid1.compartment.oc1..aaaaaa..."

  # Object Storage
  bucket: "vm-exports"
  namespace: "your-namespace"

  # Export Settings
  export_format: "qcow2"  # or "vmdk"
  enabled: true
```

### Method 2: OCI Config File

Use the standard OCI configuration file at `~/.oci/config`:

```ini
[DEFAULT]
user=ocid1.user.oc1..aaaaaa...
fingerprint=aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99
key_file=~/.oci/oci_api_key.pem
tenancy=ocid1.tenancy.oc1..aaaaaa...
region=us-phoenix-1
```

Then reference it in code:

```go
config := &CloudStorageConfig{
    OCIConfigPath: "~/.oci/config",
    OCIProfile:    "DEFAULT",
    Bucket:        "vm-exports",
    OCINamespace:  "your-namespace",
}
```

### Method 3: Environment Variables

```bash
export OCI_CONFIG_PATH=~/.oci/config
export OCI_PROFILE=DEFAULT
export OCI_REGION=us-phoenix-1
```

---

## Usage Examples

### Example 1: List OCI Instances

```go
package main

import (
    "context"
    "fmt"

    "hypersdk/config"
    "hypersdk/logger"
    "hypersdk/providers/oci"
)

func main() {
    log := logger.New("info")

    // Load configuration
    cfg, err := config.FromFile("hypersdk.yaml")
    if err != nil {
        log.Error("failed to load config", "error", err)
        return
    }

    // Create OCI client
    client, err := oci.NewClient(cfg.OCI, log)
    if err != nil {
        log.Error("failed to create OCI client", "error", err)
        return
    }

    // List instances
    ctx := context.Background()
    instances, err := client.ListInstances(ctx)
    if err != nil {
        log.Error("failed to list instances", "error", err)
        return
    }

    for _, inst := range instances {
        fmt.Printf("Instance: %s\n", inst.Name)
        fmt.Printf("  ID: %s\n", inst.ID)
        fmt.Printf("  State: %s\n", inst.State)
        fmt.Printf("  Shape: %s\n", inst.Shape)
        fmt.Printf("  AD: %s\n", inst.AvailabilityDomain)
    }
}
```

### Example 2: Export OCI Instance

```go
package main

import (
    "context"
    "fmt"

    "hypersdk/config"
    "hypersdk/logger"
    "hypersdk/providers/oci"
)

func main() {
    log := logger.New("info")

    // Load configuration
    cfg, err := config.FromFile("hypersdk.yaml")
    if err != nil {
        log.Error("failed to load config", "error", err)
        return
    }

    // Create OCI client
    client, err := oci.NewClient(cfg.OCI, log)
    if err != nil {
        log.Error("failed to create OCI client", "error", err)
        return
    }

    // Export options
    opts := &oci.ExportOptions{
        OutputDir:             "./exports",
        Format:                "qcow2",
        ImageName:             "my-instance-backup",
        ExportToObjectStorage: true,
        Bucket:                cfg.OCI.Bucket,
        Namespace:             cfg.OCI.Namespace,
        ObjectNamePrefix:      "backups/",
        DeleteAfterExport:     true,  // Cleanup temporary image
    }

    // Export instance
    ctx := context.Background()
    instanceID := "ocid1.instance.oc1.phx.aaaaaa..."

    result, err := client.ExportInstance(ctx, instanceID, opts)
    if err != nil {
        log.Error("failed to export instance", "error", err)
        return
    }

    fmt.Printf("Export completed!\n")
    fmt.Printf("  Instance: %s\n", result.InstanceName)
    fmt.Printf("  Image ID: %s\n", result.ImageID)
    fmt.Printf("  Local Path: %s\n", result.LocalPath)
    fmt.Printf("  Size: %d bytes\n", result.Size)
    fmt.Printf("  Duration: %s\n", result.Duration)
}
```

### Example 3: OCI Object Storage Operations

```go
package main

import (
    "context"
    "fmt"

    "hypersdk/logger"
)

func main() {
    log := logger.New("info")

    // Create storage client from URL
    storage, err := NewCloudStorage("oci://my-namespace/vm-exports/backups", log)
    if err != nil {
        log.Error("failed to create storage", "error", err)
        return
    }
    defer storage.Close()

    ctx := context.Background()

    // Upload file
    err = storage.Upload(ctx, "./my-vm.qcow2", "my-vm-backup.qcow2", func(bytes, total int64) {
        pct := float64(bytes) / float64(total) * 100
        fmt.Printf("\rUploading: %.1f%%", pct)
    })
    if err != nil {
        log.Error("upload failed", "error", err)
        return
    }

    fmt.Println("\nUpload completed!")

    // List objects
    files, err := storage.List(ctx, "")
    if err != nil {
        log.Error("list failed", "error", err)
        return
    }

    for _, file := range files {
        fmt.Printf("File: %s (%d bytes)\n", file.Path, file.Size)
    }

    // Download file
    err = storage.Download(ctx, "my-vm-backup.qcow2", "./downloaded.qcow2", func(bytes, total int64) {
        pct := float64(bytes) / float64(total) * 100
        fmt.Printf("\rDownloading: %.1f%%", pct)
    })
    if err != nil {
        log.Error("download failed", "error", err)
        return
    }

    fmt.Println("\nDownload completed!")
}
```

### Example 4: Network-Aware Export with Retry

```go
package main

import (
    "context"
    "fmt"
    "time"

    "hypersdk/config"
    "hypersdk/logger"
    "hypersdk/network"
    "hypersdk/providers/oci"
)

func main() {
    log := logger.New("info")

    // Create network monitor
    monitor := network.NewMonitor(nil, log)
    ctx := context.Background()
    if err := monitor.Start(ctx); err != nil {
        log.Error("failed to start network monitor", "error", err)
        return
    }
    defer monitor.Stop()

    // Load configuration
    cfg, err := config.FromFile("hypersdk.yaml")
    if err != nil {
        log.Error("failed to load config", "error", err)
        return
    }

    // Create OCI client with network monitoring
    client, err := oci.NewClient(cfg.OCI, log)
    if err != nil {
        log.Error("failed to create OCI client", "error", err)
        return
    }
    client.SetNetworkMonitor(monitor)

    // Export will automatically pause if network goes down
    opts := &oci.ExportOptions{
        OutputDir: "./exports",
        Format:    "qcow2",
        ImageName: "resilient-export",
    }

    instanceID := "ocid1.instance.oc1.phx.aaaaaa..."
    result, err := client.ExportInstance(ctx, instanceID, opts)
    if err != nil {
        log.Error("export failed", "error", err)
        return
    }

    fmt.Printf("Export completed: %s\n", result.LocalPath)
}
```

---

## API Reference

### OCI Compute Client

```go
type Client struct {
    // ... internal fields
}

// NewClient creates a new OCI client
func NewClient(cfg *config.OCIConfig, log logger.Logger) (*Client, error)

// SetNetworkMonitor sets the network monitor for retry operations
func (c *Client) SetNetworkMonitor(monitor retry.NetworkMonitor)

// ListInstances lists compute instances in the compartment
func (c *Client) ListInstances(ctx context.Context) ([]InstanceInfo, error)

// GetInstance gets details of a specific instance
func (c *Client) GetInstance(ctx context.Context, instanceID string) (*InstanceInfo, error)

// StopInstance stops a running instance
func (c *Client) StopInstance(ctx context.Context, instanceID string) error

// StartInstance starts a stopped instance
func (c *Client) StartInstance(ctx context.Context, instanceID string) error

// ExportInstance exports an OCI instance to a custom image
func (c *Client) ExportInstance(ctx context.Context, instanceID string, opts *ExportOptions) (*ExportResult, error)

// WaitForInstanceState waits for an instance to reach a specific state
func (c *Client) WaitForInstanceState(ctx context.Context, instanceID string, desiredState core.InstanceLifecycleStateEnum, timeout time.Duration) error
```

### Export Options

```go
type ExportOptions struct {
    OutputDir             string                 // Local output directory
    Format                string                 // Export format: qcow2, vmdk
    ImageName             string                 // Custom image name
    ExportToObjectStorage bool                   // Export to OCI Object Storage
    Bucket                string                 // Object Storage bucket name
    Namespace             string                 // Object Storage namespace
    ObjectNamePrefix      string                 // Prefix for object names
    DeleteAfterExport     bool                   // Delete custom image after export
    ProgressReporter      progress.ProgressReporter // Progress reporter
}
```

### Export Result

```go
type ExportResult struct {
    InstanceID       string
    InstanceName     string
    ImageID          string
    ImageName        string
    ExportFormat     string
    LocalPath        string
    ObjectStorageURL string
    Size             int64
    Duration         time.Duration
}
```

### OCI Object Storage

```go
type OCIStorage struct {
    // ... internal fields
}

// NewOCIStorage creates a new OCI Object Storage client
func NewOCIStorage(cfg *CloudStorageConfig, log logger.Logger) (*OCIStorage, error)

// Upload uploads a file to OCI Object Storage
func (o *OCIStorage) Upload(ctx context.Context, localPath, remotePath string, progress ProgressCallback) error

// UploadStream uploads data from a reader to OCI Object Storage
func (o *OCIStorage) UploadStream(ctx context.Context, reader io.Reader, remotePath string, size int64, progress ProgressCallback) error

// Download downloads a file from OCI Object Storage
func (o *OCIStorage) Download(ctx context.Context, remotePath, localPath string, progress ProgressCallback) error

// List lists objects in OCI Object Storage with a prefix
func (o *OCIStorage) List(ctx context.Context, prefix string) ([]CloudFile, error)

// Delete deletes an object from OCI Object Storage
func (o *OCIStorage) Delete(ctx context.Context, remotePath string) error

// Exists checks if an object exists in OCI Object Storage
func (o *OCIStorage) Exists(ctx context.Context, remotePath string) (bool, error)
```

---

## OCI URL Format

OCI Object Storage uses the following URL format:

```
oci://namespace/bucket/prefix
```

**Examples**:
- `oci://my-namespace/vm-exports/`
- `oci://my-namespace/backups/production/`
- `oci://my-namespace/exports/db-servers/`

**Environment Variables**:
- `OCI_CONFIG_PATH`: Path to OCI config file (default: `~/.oci/config`)
- `OCI_PROFILE`: Profile name in config file (default: `DEFAULT`)
- `OCI_REGION`: OCI region (overrides config file)

---

## Supported Regions

OCI is available in multiple regions worldwide:

**Americas**:
- `us-phoenix-1` (Phoenix, Arizona)
- `us-ashburn-1` (Ashburn, Virginia)
- `ca-toronto-1` (Toronto, Canada)
- `ca-montreal-1` (Montreal, Canada)
- `sa-saopaulo-1` (São Paulo, Brazil)

**EMEA**:
- `eu-frankfurt-1` (Frankfurt, Germany)
- `eu-zurich-1` (Zurich, Switzerland)
- `uk-london-1` (London, UK)
- `me-jeddah-1` (Jeddah, Saudi Arabia)

**Asia Pacific**:
- `ap-tokyo-1` (Tokyo, Japan)
- `ap-osaka-1` (Osaka, Japan)
- `ap-seoul-1` (Seoul, South Korea)
- `ap-mumbai-1` (Mumbai, India)
- `ap-sydney-1` (Sydney, Australia)
- `ap-melbourne-1` (Melbourne, Australia)

---

## Authentication

### API Key Authentication

1. **Generate API Key**:
   ```bash
   mkdir -p ~/.oci
   openssl genrsa -out ~/.oci/oci_api_key.pem 2048
   chmod 600 ~/.oci/oci_api_key.pem
   openssl rsa -pubout -in ~/.oci/oci_api_key.pem -out ~/.oci/oci_api_key_public.pem
   ```

2. **Upload Public Key** to OCI Console:
   - Navigate to: User Settings → API Keys → Add API Key
   - Upload `oci_api_key_public.pem`
   - Copy the fingerprint

3. **Configure** `~/.oci/config`:
   ```ini
   [DEFAULT]
   user=ocid1.user.oc1..aaaaaa...
   fingerprint=aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99
   key_file=~/.oci/oci_api_key.pem
   tenancy=ocid1.tenancy.oc1..aaaaaa...
   region=us-phoenix-1
   ```

### Instance Principal Authentication

For applications running on OCI compute instances:

```go
// Instance principal auth is automatically detected
// when no explicit credentials are provided
config := &CloudStorageConfig{
    Bucket:       "vm-exports",
    OCINamespace: "your-namespace",
}
```

---

## Best Practices

### 1. Use Compartments for Organization

```yaml
oci:
  compartment_ocid: "ocid1.compartment.oc1..production"
  bucket: "prod-vm-exports"
```

### 2. Enable Network Monitoring for Long Exports

```go
monitor := network.NewMonitor(nil, log)
monitor.Start(ctx)
client.SetNetworkMonitor(monitor)
```

### 3. Cleanup Temporary Resources

```go
opts := &oci.ExportOptions{
    DeleteAfterExport: true,  // Remove custom image after export
}
```

### 4. Use Object Name Prefixes

```go
opts := &oci.ExportOptions{
    ObjectNamePrefix: fmt.Sprintf("exports/%s/", time.Now().Format("2006-01-02")),
}
```

### 5. Monitor Export Progress

```go
type ProgressReporter struct{}

func (p *ProgressReporter) Update(pct int64) {
    fmt.Printf("\rExport progress: %d%%", pct)
}

opts.ProgressReporter = &ProgressReporter{}
```

---

## Troubleshooting

### Error: "Authentication failed"

**Cause**: Invalid API key or fingerprint

**Solution**:
```bash
# Verify fingerprint matches OCI console
openssl rsa -pubout -outform DER -in ~/.oci/oci_api_key.pem | openssl md5 -c
```

### Error: "NotAuthorizedOrNotFound"

**Cause**: Missing IAM permissions or invalid OCID

**Solution**: Ensure your user has these policies:
```
Allow group Exporters to manage instances in compartment Production
Allow group Exporters to manage custom-images in compartment Production
Allow group Exporters to manage object-family in compartment Production
```

### Error: "Service error: BucketNotFound"

**Cause**: Bucket doesn't exist or wrong namespace

**Solution**:
```bash
# Verify namespace
oci os ns get

# List buckets
oci os bucket list --compartment-id <compartment-ocid>
```

### Slow Export Performance

**Cause**: Large instance or network bottleneck

**Solution**:
- Use faster shapes for the compute instance
- Export during off-peak hours
- Enable network monitoring to detect issues
- Consider exporting to Object Storage first, then download

---

## Performance

### Instance Export

| Instance Size | Image Creation | Export to Storage | Total Time |
|--------------|----------------|-------------------|------------|
| 50 GB disk | 5-10 minutes | 15-20 minutes | 20-30 minutes |
| 100 GB disk | 10-15 minutes | 30-40 minutes | 40-55 minutes |
| 500 GB disk | 30-45 minutes | 2-3 hours | 2.5-3.5 hours |

### Object Storage

| Operation | Throughput | Latency |
|-----------|------------|---------|
| Upload | 50-100 MB/s | < 100ms |
| Download | 50-100 MB/s | < 100ms |
| List | 1000 objects/sec | < 50ms |
| Delete | 100 objects/sec | < 100ms |

---

## Integration with Retry Mechanism

All OCI operations automatically use the retry mechanism with:

- **Max Attempts**: 5
- **Initial Delay**: 2 seconds
- **Max Delay**: 30 seconds
- **Multiplier**: 2.0 (exponential backoff)
- **Jitter**: Enabled

**Retryable Errors**:
- Network timeouts
- HTTP 5xx errors
- HTTP 429 (rate limiting)
- Service throttling

**Non-Retryable Errors**:
- HTTP 404 (not found)
- HTTP 401/403 (auth errors)
- Invalid OCID format
- Validation errors

---

## Summary

The OCI integration provides:

✅ **Complete Compute Management** - List, manage, and export OCI instances
✅ **Object Storage Support** - Full CRUD operations with OCI Object Storage
✅ **Multiple Export Formats** - QCOW2 and VMDK support
✅ **Automatic Retry** - Built-in exponential backoff with jitter
✅ **Network Awareness** - Pause operations during network outages
✅ **Progress Tracking** - Real-time updates for long operations
✅ **Flexible Authentication** - API key, config file, or instance principal
✅ **Production Ready** - Comprehensive error handling and logging

Use OCI integration for reliable, scalable VM migration and backup workflows!
