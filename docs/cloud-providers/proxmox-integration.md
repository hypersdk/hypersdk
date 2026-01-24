# Proxmox VE Integration Guide

## Overview

HyperSDK provides comprehensive Proxmox Virtual Environment integration supporting VM management via the **Proxmox VE REST API**. The implementation includes backup creation via **vzdump**, VM control operations, and retry mechanisms for reliable operations.

## Features

### Proxmox VE VM Management
- ✅ List all VMs across all cluster nodes
- ✅ Get detailed VM information
- ✅ Stop and start VMs
- ✅ Get VM configuration and status
- ✅ Multi-node cluster support
- ✅ Task status monitoring

### Backup & Export
- ✅ Create VM backups via vzdump
- ✅ Download backups to local storage
- ✅ Compression support (zstd, gzip, lzo)
- ✅ Snapshot, suspend, and stop backup modes
- ✅ List and delete backups
- ✅ Find latest backup for a VM

### Reliability Features
- **Retry with Exponential Backoff** - 5 attempts, 2s→4s→8s→16s→30s delays
- **Network-Aware Retry** - Pauses during network outages, resumes automatically
- **Task Monitoring** - Automatic polling of long-running tasks
- **Automatic Re-authentication** - Handles expired tickets transparently
- **TLS Verification Control** - Optional certificate verification for self-signed certs

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

## Configuration

### Method 1: Configuration File

Create `/etc/hypervisord/config.yaml`:

```yaml
proxmox:
  # Connection Settings
  host: "proxmox.example.com"    # Proxmox VE host
  port: 8006                     # API port (default: 8006)

  # Authentication
  username: "root@pam"           # Format: user@realm
  password: "your-password"      # Or use API token

  # Optional: API Token Authentication (recommended)
  token_id: "root@pam!token-id"
  token_secret: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

  # Default Settings
  node: "pve1"                   # Default Proxmox node
  storage: "local"               # Storage for backups

  # Security
  verify_ssl: false              # Set to true for production with valid certs

  enabled: true
```

### Method 2: Environment Variables

```bash
# Proxmox Authentication
export PROXMOX_HOST="proxmox.example.com"
export PROXMOX_USERNAME="root@pam"
export PROXMOX_PASSWORD="your-password"
export PROXMOX_NODE="pve1"

# For API tokens
export PROXMOX_TOKEN_ID="root@pam!token-id"
export PROXMOX_TOKEN_SECRET="token-secret"
```

## Usage Examples

### List VMs

```bash
# List all VMs across all nodes
./hyperctl -provider proxmox list

# Using API
curl http://localhost:8080/proxmox/vms
```

### Export VM

```bash
# Interactive export
./hyperexport -provider proxmox -vm 100

# With node specification
./hyperexport -provider proxmox -vm pve1:100

# Non-interactive with options
./hyperexport \
  -provider proxmox \
  -vm 100 \
  -output /backup/exports \
  -compress
```

### Create Backup

```bash
# Create VM backup
curl -X POST http://localhost:8080/proxmox/backup \
  -H "Content-Type: application/json" \
  -d '{
    "node": "pve1",
    "vmid": 100,
    "mode": "snapshot",
    "compress": "zstd"
  }'
```

### VM Power Management

```bash
# Stop VM
curl -X POST http://localhost:8080/proxmox/vms/100/stop

# Start VM
curl -X POST http://localhost:8080/proxmox/vms/100/start
```

## Backup Modes

Proxmox supports three backup modes:

### Snapshot Mode (Recommended)
```yaml
backup_mode: "snapshot"  # Uses LVM/ZFS snapshots, VM stays online
```
- **Pros**: No downtime, consistent state
- **Cons**: Requires snapshot-capable storage (LVM, ZFS)
- **Use Case**: Production VMs that can't be stopped

### Suspend Mode
```yaml
backup_mode: "suspend"   # Suspends VM during backup
```
- **Pros**: Guaranteed consistency
- **Cons**: Brief service interruption
- **Use Case**: VMs that can tolerate short downtime

### Stop Mode
```yaml
backup_mode: "stop"      # Stops VM before backup
```
- **Pros**: Most reliable, works on any storage
- **Cons**: VM is stopped during backup
- **Use Case**: Development/test VMs, scheduled maintenance windows

## Compression Options

Choose compression based on your needs:

### zstd (Recommended)
```yaml
compress: "zstd"  # Best balance of speed and compression
```
- **Ratio**: ~60-70% size reduction
- **Speed**: Fast
- **CPU**: Moderate

### gzip
```yaml
compress: "gzip"  # Standard compression
```
- **Ratio**: ~50-60% size reduction
- **Speed**: Medium
- **CPU**: Moderate

### lzo
```yaml
compress: "lzo"   # Fastest compression
```
- **Ratio**: ~30-40% size reduction
- **Speed**: Very fast
- **CPU**: Low

## Authentication

### Password Authentication

```yaml
proxmox:
  username: "root@pam"  # Format: user@realm
  password: "password"
```

Supported realms:
- `pam` - Linux PAM
- `pve` - Proxmox VE authentication
- `ad` - Active Directory
- `ldap` - LDAP

### API Token (Recommended)

Create an API token in Proxmox Web UI:

1. Go to **Datacenter → Permissions → API Tokens**
2. Click **Add**
3. Select user, enter Token ID
4. Copy the generated secret

```yaml
proxmox:
  username: "root@pam"
  token_id: "root@pam!hypersdk"
  token_secret: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

## API Reference

### Proxmox Client

```go
import "hypersdk/providers/proxmox"

// Create client
client, err := proxmox.NewClient(cfg, logger)

// List nodes
nodes, err := client.ListNodes(ctx)

// List VMs on a node
vms, err := client.ListVMs(ctx, "pve1")

// Get VM details
vm, err := client.GetVM(ctx, "pve1", 100)

// Power management
err = client.StopVM(ctx, "pve1", 100)
err = client.StartVM(ctx, "pve1", 100)

// Backups
result, err := client.ExportVM(ctx, ExportOptions{
    Node:       "pve1",
    VMID:       100,
    OutputPath: "/backup",
    BackupMode: "snapshot",
    Compress:   "zstd",
})
```

## Error Handling

### Common Errors

**Authentication Failed:**
```
Error: authentication failed with status 401
Solution: Check username/password or API token
```

**VM Not Found:**
```
Error: VM with VMID 100 not found on any node
Solution: Verify VMID and node name
```

**Insufficient Permissions:**
```
Error: 403 Forbidden
Solution: Grant required permissions in Proxmox (VM.Backup, VM.PowerMgmt)
```

**Storage Full:**
```
Error: not enough space on storage 'local'
Solution: Clean up old backups or use different storage
```

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
./hyperexport -provider proxmox -vm 100
```

## Permissions

### Required Proxmox Permissions

Create a role with these privileges:

```
VM.Audit     - View VM configuration and status
VM.Backup    - Create and restore backups
VM.PowerMgmt - Start/stop/shutdown VMs
Datastore.Audit - View storage
Datastore.AllocateSpace - Allocate storage for backups
```

### Create Custom Role

```bash
# Via Proxmox shell
pveum role add HyperSDK -privs \
  VM.Audit,VM.Backup,VM.PowerMgmt,Datastore.Audit,Datastore.AllocateSpace

# Assign to user
pveum acl modify / -user hypersdk@pve -role HyperSDK
```

## Integration Examples

### Python Integration

```python
import requests
import time

BASE_URL = "http://localhost:8080"

# Create backup
response = requests.post(f"{BASE_URL}/proxmox/backup", json={
    "node": "pve1",
    "vmid": 100,
    "mode": "snapshot",
    "compress": "zstd"
})

backup_id = response.json()["backup_id"]

# Wait for completion
while True:
    response = requests.get(f"{BASE_URL}/proxmox/tasks/{backup_id}")
    status = response.json()["status"]

    if status == "stopped":
        print("Backup complete!")
        break

    time.sleep(5)
```

### Bash Script

```bash
#!/bin/bash

NODE="pve1"
VMID=100
BACKUP_DIR="/backup/proxmox"

# Create backup
echo "Creating backup of VM $VMID..."
./hyperexport \
  -provider proxmox \
  -vm $NODE:$VMID \
  -output $BACKUP_DIR \
  -compress

echo "Backup complete!"
```

## Troubleshooting

### Check Connectivity

```bash
# Test API access
curl -k https://proxmox.example.com:8006/api2/json/version

# Test authentication
curl -k -X POST https://proxmox.example.com:8006/api2/json/access/ticket \
  -d "username=root@pam&password=yourpassword"
```

### Verify Configuration

```bash
# Check HyperSDK status
./hyperctl status

# Test Proxmox provider
curl http://localhost:8080/providers/proxmox/health
```

## Security Best Practices

1. **Use API Tokens** instead of passwords
2. **Enable TLS verification** in production (`verify_ssl: true`)
3. **Use proper certificate** from Let's Encrypt or internal CA
4. **Least Privilege** - Grant only required permissions
5. **Rotate Credentials** - Change passwords/tokens regularly
6. **Audit Logs** - Monitor `/var/log/pve/tasks/` for suspicious activity

## Performance Tuning

### Parallel Operations

```yaml
# Increase workers for faster multi-VM exports
download_workers: 6  # Default: 4
```

### Backup Storage

Use fast storage for backups:
- **Local NVMe/SSD** - Fastest for temporary backups
- **NFS/CIFS** - Good for network backups
- **PBS (Proxmox Backup Server)** - Best for production backups

### Network Optimization

```yaml
# Adjust timeouts for slow networks
proxmox:
  timeout: 600s  # Increase for large VMs
```

## License

LGPL-3.0-or-later - See LICENSE file for details
