# HyperSDK → hyper2kvm Workflow Integration

## Overview

This document outlines the integration workflow between HyperSDK (Go daemon) and hyper2kvm (Python migration toolkit) for VM export and conversion operations.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    hyper2kvm (Python)                        │
│              Main Migration Orchestration                    │
│                 REST API Client                              │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTP/JSON
                         │
┌────────────────────────▼────────────────────────────────────┐
│              HyperSDK hypervisord Daemon                     │
│                    (Go Service)                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              REST API Server                         │   │
│  │          (Port: 8080, Configurable)                  │   │
│  └──────────────────────────────────────────────────────┘   │
│                         │                                    │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │            Job Manager & Queue                        │  │
│  │       (Async processing with progress tracking)       │  │
│  └──────────────────────┬────────────────────────────────┘  │
│                         │                                    │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │          Provider Abstraction Layer                   │  │
│  │  ┌──────┬──────┬──────┬──────┬──────┬──────┬──────┐  │  │
│  │  │vSphere│ AWS │Azure │ GCP  │Hyper-V│ OCI  │Others│  │  │
│  │  └──────┴──────┴──────┴──────┴──────┴──────┴──────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
│                         │                                    │
│  ┌──────────────────────▼────────────────────────────────┐  │
│  │         Export Pipeline (VM Export)                   │  │
│  │    ┌──────────────────────────────────────┐           │  │
│  │    │ OVA/OVF/VMDK Export to Filesystem    │           │  │
│  │    └──────────────────────────────────────┘           │  │
│  └──────────────────────┬────────────────────────────────┘  │
└─────────────────────────┼───────────────────────────────────┘
                          │
                          │ Output: OVF/VMDK files
                          │
┌─────────────────────────▼───────────────────────────────────┐
│           hyper2kvm Conversion Pipeline                      │
│    ┌─────────────────────────────────────────────────┐      │
│    │  1. Parse VMDK/OVF metadata                     │      │
│    │  2. Convert virtual disk to raw/qcow2          │      │
│    │  3. Generate libvirt domain XML                │      │
│    │  4. Register with KVM/libvirt                  │      │
│    └─────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## Workflow Steps

### Phase 1: Initial Setup

#### 1.1 Start HyperSDK Daemon

```bash
# Start the daemon with configuration
hypervisord --config /etc/hypersdk/config.yaml

# Or as systemd service
sudo systemctl start hypervisord
sudo systemctl enable hypervisord
```

#### 1.2 Verify Daemon Status

```bash
# Check health
curl http://localhost:8080/health

# Expected response:
# {"status": "healthy", "version": "1.0.0"}

# Check capabilities
curl http://localhost:8080/capabilities
```

### Phase 2: VM Discovery

#### 2.1 List Available VMs

```bash
# List VMs from connected provider (e.g., vSphere)
curl -X POST http://localhost:8080/vms/list \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "vsphere",
    "filter": {
      "datacenter": "DC1",
      "status": "poweredOn"
    }
  }'
```

#### 2.2 Get VM Details

```bash
# Get specific VM information
curl http://localhost:8080/vms/info?identifier=vm-123&provider=vsphere
```

### Phase 3: Export Job Submission

#### 3.1 Submit Export Job

```bash
# Submit VM export job
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_identifier": "vm-123",
    "provider": "vsphere",
    "export_method": "ova",
    "export_path": "/exports/vm-123",
    "options": {
      "shutdown_vm": false,
      "remove_cdrom": true,
      "compress": true
    },
    "hyper2kvm_integration": {
      "enabled": true,
      "auto_convert": false,
      "daemon_mode": true,
      "instance": "vsphere"
    }
  }'

# Response:
# {
#   "job_id": "job-abc-123",
#   "status": "pending",
#   "created_at": "2026-01-29T10:00:00Z"
# }
```

#### 3.2 Monitor Job Progress

```bash
# Query job status
curl http://localhost:8080/jobs/query?job_id=job-abc-123

# Get real-time progress
curl http://localhost:8080/jobs/progress/job-abc-123

# Stream job logs
curl http://localhost:8080/jobs/logs/job-abc-123

# Get ETA
curl http://localhost:8080/jobs/eta/job-abc-123
```

### Phase 4: hyper2kvm Integration

#### 4.1 Automatic Conversion (Daemon Mode)

When `hyper2kvm_integration.daemon_mode: true`, HyperSDK automatically:

1. Detects running hyper2kvm systemd service
2. Creates manifest file in watch directory
3. hyper2kvm daemon picks up the job
4. Conversion happens automatically

```bash
# Check if hyper2kvm daemon is active
systemctl is-active hyper2kvm@vsphere.service

# Monitor hyper2kvm logs
journalctl -u hyper2kvm@vsphere.service -f
```

#### 4.2 Manual Conversion (Direct Call)

When `auto_convert: true` and `daemon_mode: false`:

```bash
# HyperSDK calls hyper2kvm directly
# POST /convert endpoint
curl -X POST http://localhost:8080/convert \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/exports/vm-123/vm-123.ovf",
    "output_path": "/kvm/vm-123",
    "conversion_options": {
      "format": "qcow2",
      "compression": true
    }
  }'
```

#### 4.3 Import to KVM/libvirt

```bash
# Import converted VM to KVM
curl -X POST http://localhost:8080/import-to-kvm \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "vm-123",
    "disk_path": "/kvm/vm-123/disk.qcow2",
    "memory_mb": 4096,
    "vcpus": 2,
    "network": "default",
    "autostart": true
  }'
```

### Phase 5: Workflow Automation

#### 5.1 Create Scheduled Export

```bash
# Schedule regular exports
curl -X POST http://localhost:8080/schedules/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "nightly-vm-backup",
    "cron": "0 2 * * *",
    "job_definition": {
      "vm_identifier": "vm-123",
      "provider": "vsphere",
      "export_method": "ova",
      "export_path": "/exports/nightly",
      "hyper2kvm_integration": {
        "enabled": true,
        "daemon_mode": true
      }
    }
  }'
```

#### 5.2 Create Backup Policy

```bash
# Define backup policy
curl -X POST http://localhost:8080/backup-policies/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-vms",
    "vm_filter": {
      "tags": ["production"],
      "datacenter": "DC1"
    },
    "schedule": "0 2 * * *",
    "retention": {
      "count": 7,
      "age_days": 30
    },
    "export_options": {
      "hyper2kvm_integration": {
        "enabled": true,
        "daemon_mode": true
      }
    }
  }'
```

### Phase 6: Monitoring & Webhooks

#### 6.1 Configure Webhooks

```bash
# Set up webhook notifications
curl -X POST http://localhost:8080/webhooks/configure \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://monitoring.example.com/hypersdk-events",
    "events": [
      "job.started",
      "job.completed",
      "job.failed"
    ],
    "headers": {
      "Authorization": "Bearer token123"
    }
  }'
```

#### 6.2 WebSocket Real-Time Updates

```javascript
// Connect to WebSocket for live updates
const ws = new WebSocket('ws://localhost:8080/websocket');

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log(`Job ${update.job_id}: ${update.status} - ${update.progress}%`);
};
```

## Integration Patterns

### Pattern 1: Direct Pipeline (Single VM)

```
User → Submit Job → HyperSDK Export → hyper2kvm Convert → KVM Import → Done
```

**Use Case**: One-time migration of a single VM

**Code Flow**:
1. Submit job via `/jobs/submit`
2. Monitor progress via WebSocket
3. Auto-convert via hyper2kvm
4. Auto-import to libvirt
5. VM ready in KVM

### Pattern 2: Daemon Queue (Batch Processing)

```
User → Multiple Jobs → HyperSDK Queue → hyper2kvm Daemon (watch dir) → Batch Convert
```

**Use Case**: Migrating multiple VMs overnight

**Code Flow**:
1. Submit multiple jobs
2. Jobs queued in HyperSDK
3. Exports written to filesystem
4. Manifests created in watch directory
5. hyper2kvm daemon processes queue
6. Conversions happen asynchronously

### Pattern 3: Scheduled Backup

```
Cron Schedule → HyperSDK Auto-Export → hyper2kvm Convert → Archive → Cleanup Old
```

**Use Case**: Regular VM backups with retention policy

**Code Flow**:
1. Schedule created via `/schedules/create`
2. Cron triggers export at specified time
3. Export completes
4. hyper2kvm converts to KVM format
5. Old backups cleaned up based on retention policy

## Configuration Examples

### HyperSDK Config (config.yaml)

```yaml
# HyperSDK daemon configuration
daemon:
  address: "0.0.0.0:8080"
  log_level: "info"
  workers: 4

providers:
  vsphere:
    enabled: true
    host: "vcenter.example.com"
    username: "administrator@vsphere.local"
    password: "${VSPHERE_PASSWORD}"
    insecure: false
    connection_pool_size: 5

hyper2kvm:
  # Path to hyper2kvm executable
  path: "/usr/local/bin/hyper2kvm"

  # Daemon mode settings
  daemon_enabled: true
  watch_dir: "/var/lib/hyper2kvm/watch"
  output_dir: "/var/lib/kvm/images"

  # Conversion defaults
  default_format: "qcow2"
  compression: true

  # Polling settings
  poll_interval: "5s"
  job_timeout: "2h"

export:
  base_path: "/exports"
  chunk_size: 10485760  # 10MB
  max_retries: 3
  concurrent_exports: 2

database:
  path: "/var/lib/hypervisord/hypersdk.db"

webhooks:
  - url: "https://slack.example.com/webhook"
    events: ["job.failed"]
```

### hyper2kvm Systemd Service

```ini
# /etc/systemd/system/hyper2kvm@.service
[Unit]
Description=hyper2kvm Conversion Daemon (%i)
After=network.target libvirtd.service

[Service]
Type=simple
User=root
Environment="HYPER2KVM_WATCH_DIR=/var/lib/hyper2kvm/watch/%i"
Environment="HYPER2KVM_OUTPUT_DIR=/var/lib/kvm/images"
ExecStart=/usr/local/bin/hyper2kvm daemon \
  --watch-dir ${HYPER2KVM_WATCH_DIR} \
  --output-dir ${HYPER2KVM_OUTPUT_DIR} \
  --format qcow2 \
  --log-level info
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Integration Test Script

```bash
#!/bin/bash
# test-integration.sh - Test HyperSDK → hyper2kvm workflow

set -e

DAEMON_URL="http://localhost:8080"
VM_ID="test-vm-001"

echo "=== HyperSDK → hyper2kvm Integration Test ==="

# 1. Check daemon health
echo "1. Checking daemon health..."
curl -f ${DAEMON_URL}/health || exit 1

# 2. List VMs
echo "2. Listing available VMs..."
VMS=$(curl -s -X POST ${DAEMON_URL}/vms/list \
  -H "Content-Type: application/json" \
  -d '{"provider": "vsphere"}')
echo "Found VMs: ${VMS}"

# 3. Submit export job
echo "3. Submitting export job..."
JOB_RESPONSE=$(curl -s -X POST ${DAEMON_URL}/jobs/submit \
  -H "Content-Type: application/json" \
  -d "{
    \"vm_identifier\": \"${VM_ID}\",
    \"provider\": \"vsphere\",
    \"export_method\": \"ova\",
    \"export_path\": \"/tmp/exports/${VM_ID}\",
    \"options\": {
      \"shutdown_vm\": false,
      \"remove_cdrom\": true
    },
    \"hyper2kvm_integration\": {
      \"enabled\": true,
      \"daemon_mode\": true,
      \"instance\": \"vsphere\"
    }
  }")

JOB_ID=$(echo ${JOB_RESPONSE} | jq -r '.job_id')
echo "Job submitted: ${JOB_ID}"

# 4. Monitor job progress
echo "4. Monitoring job progress..."
while true; do
  STATUS=$(curl -s ${DAEMON_URL}/jobs/query?job_id=${JOB_ID} | jq -r '.status')
  PROGRESS=$(curl -s ${DAEMON_URL}/jobs/progress/${JOB_ID} | jq -r '.percentage')

  echo "Status: ${STATUS} - Progress: ${PROGRESS}%"

  if [ "$STATUS" = "completed" ]; then
    echo "Export completed successfully!"
    break
  elif [ "$STATUS" = "failed" ]; then
    echo "Export failed!"
    curl -s ${DAEMON_URL}/jobs/logs/${JOB_ID}
    exit 1
  fi

  sleep 5
done

# 5. Check hyper2kvm conversion
echo "5. Checking hyper2kvm daemon status..."
systemctl status hyper2kvm@vsphere.service

# 6. Verify output
echo "6. Verifying converted files..."
ls -lh /var/lib/kvm/images/${VM_ID}/

echo "=== Integration test completed successfully! ==="
```

## API Endpoints Summary

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/health` | GET | Health check |
| `/jobs/submit` | POST | Submit export job |
| `/jobs/query` | GET | Query job status |
| `/jobs/progress/{id}` | GET | Get job progress |
| `/jobs/logs/{id}` | GET | Stream job logs |
| `/vms/list` | POST | List VMs from provider |
| `/vms/info` | GET | Get VM details |
| `/convert` | POST | Convert VM to KVM format |
| `/import-to-kvm` | POST | Import to libvirt |
| `/parse-vmdk` | GET | Parse VMDK metadata |
| `/schedules/create` | POST | Create scheduled job |
| `/webhooks/configure` | POST | Configure webhooks |
| `/websocket` | WS | Real-time updates |

## Error Handling

### Common Errors

1. **Provider Connection Failed**
   - Check credentials in config
   - Verify network connectivity
   - Validate SSL certificates

2. **Export Failed**
   - Check disk space
   - Verify VM permissions
   - Review export logs

3. **hyper2kvm Conversion Failed**
   - Check hyper2kvm daemon status
   - Verify watch directory permissions
   - Review conversion logs

4. **Import to KVM Failed**
   - Verify libvirt connectivity
   - Check storage pool availability
   - Validate network configuration

### Debugging

```bash
# Check HyperSDK logs
journalctl -u hypervisord -f

# Check hyper2kvm logs
journalctl -u hyper2kvm@vsphere -f

# Check libvirt logs
journalctl -u libvirtd -f

# Get job logs
curl http://localhost:8080/jobs/logs/{job_id}
```

## Performance Considerations

### Optimization Tips

1. **Parallel Exports**: Configure `concurrent_exports` based on storage IOPS
2. **Chunk Size**: Adjust `chunk_size` for network/disk balance
3. **Connection Pooling**: Use provider connection pools for multiple VMs
4. **Compression**: Enable compression for network transfers, disable for local
5. **Worker Count**: Set daemon workers = CPU cores

### Recommended Settings

| Environment | Workers | Chunk Size | Concurrent |
|-------------|---------|------------|------------|
| Small (1-10 VMs) | 2 | 10MB | 1 |
| Medium (10-50 VMs) | 4 | 20MB | 2 |
| Large (50+ VMs) | 8 | 50MB | 4 |

## Security Best Practices

1. **Credentials**: Use environment variables, not hardcoded passwords
2. **TLS**: Enable HTTPS for daemon API
3. **Authentication**: Configure API authentication
4. **RBAC**: Use role-based access control
5. **Audit**: Enable audit logging for all operations
6. **Network**: Restrict daemon to internal network

## Next Steps

1. Review existing integration code:
   - `/daemon/api/hyper2kvm_integration.go`
   - `/daemon/jobs/pipeline.go`
   - `/providers/common/libvirt.go`

2. Customize configuration for your environment

3. Run integration test script

4. Set up monitoring and alerting

5. Create backup policies for production VMs

## References

- HyperSDK API Documentation: `/docs/api/`
- Provider Documentation: `/docs/providers/`
- Integration Guide: `/docs/integration/daemon-integration.md`
- Configuration Examples: `/config.example.yaml`
