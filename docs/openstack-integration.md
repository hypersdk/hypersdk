# OpenStack Integration Guide

## Overview

HyperSDK provides comprehensive OpenStack integration supporting both **Nova** (Compute) for instance management and **Swift** (Object Storage) for backup storage. The implementation uses the official Gophercloud SDK with intelligent retry mechanisms and network monitoring.

## Features

### OpenStack Nova (Compute)
- ✅ List all instances across the cloud
- ✅ Get detailed instance information
- ✅ Stop and start instances
- ✅ Create snapshots (images) from instances
- ✅ Export instances via Glance image download
- ✅ Upload custom images to Glance
- ✅ Wait for image status (active/error detection)
- ✅ Delete images with cleanup

### OpenStack Swift (Object Storage)
- ✅ Upload files and streams with progress tracking
- ✅ Download objects with retry support
- ✅ List objects with pagination
- ✅ Delete objects and cleanup
- ✅ Check object existence
- ✅ Native `swift://` URL support

### Reliability Features
- **Retry with Exponential Backoff** - 5 attempts, 2s→4s→8s→16s→30s delays
- **Network-Aware Retry** - Pauses during network outages, resumes automatically
- **Smart Error Detection** - Distinguishes retryable (5xx, timeouts) from non-retryable (404, 403) errors
- **Progress Tracking** - Real-time progress callbacks during uploads/downloads
- **Automatic Cleanup** - Handles temporary resources and failed operations

## Installation

### Prerequisites

```bash
# Install Go 1.24+
sudo dnf install golang  # Fedora/RHEL
sudo apt install golang   # Ubuntu/Debian

# Install HyperSDK
git clone https://github.com/ssahani/hypersdk
cd hypersdk
go build ./cmd/hyperexport
go build ./cmd/hypervisord
go build ./cmd/hyperctl
```

### Dependencies

The OpenStack integration uses Gophercloud:

```bash
go get github.com/gophercloud/gophercloud
go get github.com/gophercloud/gophercloud/openstack
go get github.com/gophercloud/gophercloud/openstack/compute/v2/servers
go get github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop
go get github.com/gophercloud/gophercloud/openstack/imageservice/v2/images
go get github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects
```

All dependencies are automatically managed via `go.mod`.

## Configuration

### Method 1: Configuration File

Create `/etc/hypervisord/config.yaml`:

```yaml
openstack:
  # Keystone Authentication
  auth_url: "https://openstack.example.com:5000/v3"
  username: "admin"
  password: "your-secure-password"
  tenant_name: "admin"        # Project name
  tenant_id: ""               # Optional: Use ID instead of name
  domain_name: "Default"      # Domain for v3 auth
  region: "RegionOne"

  # Swift Object Storage
  container: "vm-backups"     # Swift container for backups

  # Export Settings
  export_format: "qcow2"      # Image format: qcow2, vmdk, raw
  identity_version: "v3"      # Keystone version: v2.0 or v3

  enabled: true
```

### Method 2: Environment Variables

```bash
# Keystone Authentication
export OS_AUTH_URL="https://openstack.example.com:5000/v3"
export OS_USERNAME="admin"
export OS_PASSWORD="your-secure-password"
export OS_TENANT_NAME="admin"
export OS_DOMAIN_NAME="Default"
export OS_REGION_NAME="RegionOne"

# Cloud Storage URL
export CLOUD_STORAGE_URL="swift://vm-backups/exports/"
```

### Method 3: OpenStack RC File

Source the standard OpenStack RC file:

```bash
# Download from Horizon dashboard or create manually
source openstack-admin-openrc.sh

# Then use with HyperSDK
./hyperexport -vm my-instance -upload swift://vm-backups/
```

## Usage Examples

### List Instances

```bash
# Using hyperctl
./hyperctl -provider openstack list

# Using API
curl http://localhost:8080/openstack/instances
```

### Export Instance

```bash
# Interactive export
./hyperexport -provider openstack -vm my-instance

# Non-interactive with options
./hyperexport \
  -provider openstack \
  -vm my-instance \
  -output /backup/exports \
  -format qcow2 \
  -compress
```

### Create Snapshot

```bash
# Create snapshot/image from instance
curl -X POST http://localhost:8080/openstack/instances/snapshot \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "abc123...",
    "snapshot_name": "my-instance-backup-2024-01-21"
  }'
```

### Upload to Swift Object Storage

```bash
# Upload after export
./hyperexport \
  -vm my-instance \
  -upload swift://vm-backups/2024-01-21/

# Direct streaming upload (no local storage)
./hyperexport \
  -vm my-instance \
  -upload swift://vm-backups/ \
  -stream-upload \
  --keep-local=false
```

### Download from Swift

```bash
# Download to local directory
curl -X POST http://localhost:8080/cloud/download \
  -H "Content-Type: application/json" \
  -d '{
    "url": "swift://vm-backups/my-backup.qcow2",
    "output_path": "/restore/my-backup.qcow2"
  }'
```

## Advanced Features

### Retry Mechanism

The OpenStack client includes intelligent retry with exponential backoff:

```yaml
# Customize retry behavior (optional)
retry:
  max_attempts: 5
  initial_delay: 2s
  max_delay: 30s
  multiplier: 2.0
  jitter: true
```

**Retry Logic:**
- Attempt 1: Immediate
- Attempt 2: 2s delay
- Attempt 3: 4s delay
- Attempt 4: 8s delay
- Attempt 5: 16s delay

**Retryable Errors:**
- 5xx server errors
- Network timeouts
- Connection refused
- Temporary DNS failures

**Non-Retryable Errors:**
- 404 Not Found
- 403 Forbidden
- 401 Unauthorized
- 400 Bad Request

### Network Monitoring

When network monitoring is enabled, operations automatically pause during network outages:

```go
// Enable network monitoring
monitor := network.NewMonitor(logger)
monitor.Start(ctx)

client.SetNetworkMonitor(monitor)

// Operations will pause during outages and resume automatically
```

### Progress Tracking

All upload/download operations support progress callbacks:

```go
progress := func(transferred, total int64) {
    percentage := float64(transferred) / float64(total) * 100
    fmt.Printf("Progress: %.1f%% (%d/%d bytes)\n", percentage, transferred, total)
}

err := swiftStorage.Upload(ctx, localPath, remotePath, progress)
```

### Swift Pagination

List operations automatically handle pagination:

```go
// Automatically fetches all pages
files, err := swiftStorage.List(ctx, "exports/")

// Returns all objects, handling pagination internally
for _, file := range files {
    fmt.Printf("%s - %d bytes\n", file.Path, file.Size)
}
```

## API Reference

### Nova Compute Client

```go
import "hypersdk/providers/openstack"

// Create client
client, err := openstack.NewClient(cfg, logger)

// List instances
instances, err := client.ListInstances(ctx)

// Get instance details
instance, err := client.GetInstance(ctx, instanceID)

// Power management
err = client.StopInstance(ctx, instanceID)
err = client.StartInstance(ctx, instanceID)

// Snapshots
imageID, err := client.CreateSnapshot(ctx, instanceID, "snapshot-name")
err = client.WaitForImageStatus(ctx, imageID, "active", 30*time.Minute)
err = client.DeleteImage(ctx, imageID)
```

### Swift Object Storage

```go
import "hypersdk/cmd/hyperexport"

// Create Swift storage from URL
storage, err := NewCloudStorage("swift://container/prefix", logger)

// Upload file
err = storage.Upload(ctx, localPath, remotePath, progressCallback)

// Download file
err = storage.Download(ctx, remotePath, localPath, progressCallback)

// List objects
files, err := storage.List(ctx, "prefix/")

// Delete object
err = storage.Delete(ctx, remotePath)

// Check existence
exists, err := storage.Exists(ctx, remotePath)
```

## Error Handling

### Common Errors

**Authentication Failed:**
```
Error: authenticate to OpenStack: Invalid credentials
Solution: Check OS_USERNAME, OS_PASSWORD, and OS_AUTH_URL
```

**Container Not Found:**
```
Error: container not found: vm-backups
Solution: Create container: swift post vm-backups
```

**Insufficient Permissions:**
```
Error: 403 Forbidden
Solution: Ensure user has required roles (Member, admin, etc.)
```

**Network Timeout:**
```
Error: request timeout
Solution: Check network connectivity, firewall rules, and endpoint accessibility
```

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
# Set log level to debug
export LOG_LEVEL=debug
./hyperexport -provider openstack -vm my-instance

# Or in config file
log_level: debug
```

## Integration Examples

### Python Integration

```python
import requests
import time

BASE_URL = "http://localhost:8080"

# Create snapshot
response = requests.post(f"{BASE_URL}/openstack/instances/snapshot", json={
    "instance_id": "abc123...",
    "snapshot_name": "automated-backup"
})

image_id = response.json()["image_id"]

# Wait for snapshot completion
while True:
    response = requests.get(f"{BASE_URL}/openstack/images/{image_id}")
    status = response.json()["status"]

    if status == "active":
        print("Snapshot ready!")
        break
    elif status in ["error", "killed", "deleted"]:
        print(f"Snapshot failed: {status}")
        break

    time.sleep(10)

# Download snapshot
requests.post(f"{BASE_URL}/cloud/download", json={
    "url": f"swift://backups/snapshot-{image_id}.qcow2",
    "output_path": f"/restore/snapshot-{image_id}.qcow2"
})
```

### Ansible Playbook

```yaml
---
- name: Export OpenStack Instance
  hosts: localhost
  tasks:
    - name: Submit export job
      uri:
        url: "http://localhost:8080/jobs/submit"
        method: POST
        body_format: json
        body:
          name: "openstack-export"
          provider: "openstack"
          vm_path: "{{ instance_id }}"
          output_path: "/backup/{{ inventory_hostname }}"
          options:
            format: "qcow2"
            compress: true
      register: job_result

    - name: Wait for completion
      uri:
        url: "http://localhost:8080/jobs/progress/{{ job_result.json.job_ids[0] }}"
        method: GET
      register: progress
      until: progress.json.status in ['completed', 'failed']
      retries: 120
      delay: 30
```

### Bash Script

```bash
#!/bin/bash

INSTANCE_ID="your-instance-id"
BACKUP_NAME="backup-$(date +%Y%m%d)"

# Create snapshot
echo "Creating snapshot..."
IMAGE_ID=$(curl -s -X POST http://localhost:8080/openstack/instances/snapshot \
  -H "Content-Type: application/json" \
  -d "{\"instance_id\":\"$INSTANCE_ID\",\"snapshot_name\":\"$BACKUP_NAME\"}" \
  | jq -r '.image_id')

echo "Snapshot created: $IMAGE_ID"

# Wait for snapshot
echo "Waiting for snapshot to be ready..."
while true; do
  STATUS=$(curl -s http://localhost:8080/openstack/images/$IMAGE_ID | jq -r '.status')

  if [ "$STATUS" = "active" ]; then
    echo "Snapshot ready!"
    break
  elif [ "$STATUS" = "error" ]; then
    echo "Snapshot failed!"
    exit 1
  fi

  sleep 10
done

# Upload to Swift
echo "Uploading to Swift..."
swift upload vm-backups /path/to/export --object-name="$BACKUP_NAME.qcow2"

echo "Backup complete!"
```

## Troubleshooting

### Check Connectivity

```bash
# Test Keystone authentication
curl -X POST $OS_AUTH_URL/auth/tokens \
  -H "Content-Type: application/json" \
  -d '{
    "auth": {
      "identity": {
        "methods": ["password"],
        "password": {
          "user": {
            "name": "'$OS_USERNAME'",
            "domain": {"name": "'$OS_DOMAIN_NAME'"},
            "password": "'$OS_PASSWORD'"
          }
        }
      }
    }
  }'

# Test Swift endpoint
swift stat

# Test Nova endpoint
openstack server list
```

### Verify Configuration

```bash
# Check HyperSDK config
./hyperctl status

# Test OpenStack provider
curl http://localhost:8080/providers/openstack/health
```

### Common Issues

**Issue: "Unable to authenticate"**
- Verify OS_AUTH_URL points to Keystone endpoint (includes /v3 or /v2.0)
- Check username format for v3: "username" not "username@domain"
- Ensure domain is set for Keystone v3

**Issue: "Container not found"**
- Create container: `swift post vm-backups`
- Or create via Horizon dashboard: Project → Object Store → Containers

**Issue: "Insufficient quota"**
- Check OpenStack quotas: `openstack quota show`
- Request quota increase from cloud admin

## Performance Tuning

### Parallel Downloads

```yaml
# Increase parallel workers for faster downloads
download_workers: 8  # Default: 4
```

### Connection Pooling

```yaml
# Increase connection pool for concurrent operations
connection_pool:
  max_connections: 10
  idle_timeout: 5m
```

### Swift Timeouts

```yaml
# Adjust timeouts for large objects
swift:
  timeout: 600s      # 10 minutes for large uploads
  chunk_size: 5MB    # Upload chunk size
```

## Security Best Practices

1. **Use Application Credentials** instead of password auth (OpenStack Stein+):
   ```bash
   openstack application credential create hypersdk --role Member
   ```

2. **Encrypt Swift Objects** with server-side encryption:
   ```bash
   swift post vm-backups -H "X-Container-Meta-Encryption: true"
   ```

3. **Use Private Networks** for Swift access when possible

4. **Rotate Credentials** regularly using Keystone policies

5. **Enable Audit Logging** in OpenStack for compliance

## Migration from Other Providers

See [MIGRATION.md](MIGRATION.md) for guides on migrating from:
- VMware vSphere → OpenStack
- AWS EC2 → OpenStack
- Azure VMs → OpenStack

## Support and Resources

- **OpenStack Documentation**: https://docs.openstack.org/
- **Gophercloud SDK**: https://github.com/gophercloud/gophercloud
- **HyperSDK Issues**: https://github.com/ssahani/hypersdk/issues
- **OpenStack ML**: http://lists.openstack.org/

## License

LGPL-3.0-or-later - See LICENSE file for details
