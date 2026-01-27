# HyperSDK ↔ hyper2kvm Integration - Meeting Quick Start

## 1-Minute Overview

**What is this integration?**

HyperSDK (Go) provides the **export layer** that pulls VMs from 9 cloud providers, while hyper2kvm (Python) handles the **conversion and import** to KVM/libvirt.

```
Cloud Provider → HyperSDK Export → hyper2kvm Convert → KVM Import
   (vSphere)         (Go daemon)      (Python tool)      (libvirt)
```

## Key Integration Points

### 1. REST API Communication

```bash
# HyperSDK exposes REST API on port 8080
curl http://localhost:8080/jobs/submit -d '{
  "vm_identifier": "vm-123",
  "provider": "vsphere",
  "hyper2kvm_integration": {"enabled": true}
}'
```

### 2. Two Integration Modes

#### Mode A: Daemon Queue (Production)
- HyperSDK exports VM → writes manifest to watch directory
- hyper2kvm daemon picks up job from queue
- Async processing, handles batches

#### Mode B: Direct Call (Testing)
- HyperSDK calls hyper2kvm directly as subprocess
- Synchronous, immediate conversion
- Good for single VM migrations

### 3. Existing Code

**Integration already implemented:**
- `/daemon/api/hyper2kvm_integration.go` (310 lines) - API handlers
- `/daemon/jobs/pipeline.go` (150+ lines) - Pipeline execution
- `/providers/common/libvirt.go` (200+ lines) - KVM import

## 5-Minute Demo

### Prerequisites
```bash
# Start HyperSDK daemon
sudo systemctl start hypervisord

# Start hyper2kvm daemon (daemon mode)
sudo systemctl start hyper2kvm@vsphere.service

# Or run demo script
./examples/hyper2kvm-demo.sh
```

### Live Demo Steps

**Step 1: Check Health**
```bash
curl http://localhost:8080/health
# → {"status": "healthy"}
```

**Step 2: List VMs**
```bash
curl -X POST http://localhost:8080/vms/list \
  -H "Content-Type: application/json" \
  -d '{"provider": "vsphere"}'
# → Returns JSON array of VMs
```

**Step 3: Submit Export Job**
```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_identifier": "test-vm",
    "provider": "vsphere",
    "export_method": "ova",
    "export_path": "/exports/test-vm",
    "hyper2kvm_integration": {
      "enabled": true,
      "daemon_mode": true
    }
  }'
# → {"job_id": "job-abc-123", "status": "pending"}
```

**Step 4: Monitor Progress**
```bash
# Real-time progress
curl http://localhost:8080/jobs/progress/job-abc-123
# → {"percentage": 45, "bytes_transferred": 5242880}

# WebSocket for live updates (JavaScript)
const ws = new WebSocket('ws://localhost:8080/websocket');
ws.onmessage = (event) => console.log(event.data);
```

**Step 5: Check Conversion**
```bash
# Check hyper2kvm daemon
systemctl status hyper2kvm@vsphere.service

# View conversion logs
journalctl -u hyper2kvm@vsphere -f
```

## Architecture Diagram (ASCII)

```
┌─────────────────────────────────────────────────────────────┐
│                  User / Orchestrator                        │
│                  (Python/CLI/Web UI)                        │
└────────────────────────┬────────────────────────────────────┘
                         │ REST API (HTTP/JSON)
                         │
┌────────────────────────▼────────────────────────────────────┐
│               HyperSDK Daemon (Go)                          │
│                  Port: 8080                                 │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │         Job Manager & Queue                         │   │
│  │  ┌──────┬──────┬──────┬──────┐                      │   │
│  │  │Job 1 │Job 2 │Job 3 │Job 4 │  (concurrent)        │   │
│  │  └──────┴──────┴──────┴──────┘                      │   │
│  └─────────────────────────────────────────────────────┘   │
│                         │                                   │
│  ┌─────────────────────▼───────────────────────────────┐   │
│  │       Provider Abstraction Layer                    │   │
│  │  ┌────────┬────────┬────────┬────────┬────────┐    │   │
│  │  │vSphere │  AWS   │ Azure  │  GCP   │ Others │    │   │
│  │  └────────┴────────┴────────┴────────┴────────┘    │   │
│  └─────────────────────────────────────────────────────┘   │
│                         │                                   │
│  ┌─────────────────────▼───────────────────────────────┐   │
│  │     Export Engine (OVA/OVF/VMDK)                    │   │
│  └─────────────────────┬───────────────────────────────┘   │
└────────────────────────┼───────────────────────────────────┘
                         │
                         │ Filesystem
                         ▼
┌─────────────────────────────────────────────────────────────┐
│           /exports/vm-name/*.{ovf,vmdk}                     │
│                  + manifest.yaml                            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         │ Watch Directory / Direct Call
                         ▼
┌─────────────────────────────────────────────────────────────┐
│         hyper2kvm Conversion Daemon (Python)                │
│           systemd service: hyper2kvm@vsphere                │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │   1. Parse OVF/VMDK metadata                        │   │
│  │   2. Convert disk to raw/qcow2                      │   │
│  │   3. Generate libvirt domain XML                    │   │
│  │   4. Register with libvirt                          │   │
│  └─────────────────────┬───────────────────────────────┘   │
└────────────────────────┼───────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                  KVM / libvirt                              │
│          /var/lib/kvm/images/vm-name/*.qcow2               │
└─────────────────────────────────────────────────────────────┘
```

## Use Cases

### Use Case 1: Single VM Migration
**Scenario:** Migrate one VM from vSphere to KVM for testing

```bash
# Submit job with direct conversion
curl -X POST http://localhost:8080/jobs/submit -d '{
  "vm_identifier": "test-app-vm",
  "provider": "vsphere",
  "hyper2kvm_integration": {
    "enabled": true,
    "daemon_mode": false,  # Direct call
    "auto_convert": true
  }
}'
```

**Result:** VM exported and immediately converted in one operation

### Use Case 2: Batch Migration (50 VMs)
**Scenario:** Migrate production VMs overnight

```bash
# Create backup policy for multiple VMs
curl -X POST http://localhost:8080/backup-policies/create -d '{
  "name": "production-migration",
  "vm_filter": {"tags": ["production"]},
  "schedule": "0 2 * * *",  # 2 AM daily
  "hyper2kvm_integration": {
    "enabled": true,
    "daemon_mode": true  # Queue-based
  }
}'
```

**Result:** All VMs exported and queued, hyper2kvm daemon processes in background

### Use Case 3: Continuous DR Backups
**Scenario:** Daily backups of critical VMs

```bash
# Schedule with retention policy
curl -X POST http://localhost:8080/schedules/create -d '{
  "name": "dr-backup",
  "cron": "0 1 * * *",
  "job_definition": {
    "vm_identifier": "db-server",
    "export_method": "ova",
    "hyper2kvm_integration": {"enabled": true}
  },
  "retention": {"count": 7}  # Keep 7 days
}'
```

**Result:** Automated nightly backups with automatic cleanup

## API Endpoints Cheat Sheet

| Endpoint | Method | Purpose | Example |
|----------|--------|---------|---------|
| `/health` | GET | Check daemon status | `curl /health` |
| `/jobs/submit` | POST | Submit export job | See examples above |
| `/jobs/query?job_id=X` | GET | Get job status | `curl /jobs/query?job_id=123` |
| `/jobs/progress/X` | GET | Get progress % | `curl /jobs/progress/123` |
| `/jobs/logs/X` | GET | Stream job logs | `curl /jobs/logs/123` |
| `/vms/list` | POST | List VMs | `curl -X POST /vms/list -d '{...}'` |
| `/convert` | POST | Convert VM | hyper2kvm integration |
| `/import-to-kvm` | POST | Import to libvirt | Final step |
| `/websocket` | WS | Real-time updates | `ws://localhost:8080/websocket` |

## Configuration Example

### HyperSDK Config (`/etc/hypersdk/config.yaml`)

```yaml
daemon:
  address: "0.0.0.0:8080"
  workers: 4

providers:
  vsphere:
    enabled: true
    host: "vcenter.example.com"
    username: "admin@vsphere.local"
    password: "${VSPHERE_PASSWORD}"

hyper2kvm:
  daemon_enabled: true
  watch_dir: "/var/lib/hyper2kvm/watch"
  output_dir: "/var/lib/kvm/images"
  default_format: "qcow2"

export:
  base_path: "/exports"
  concurrent_exports: 2
```

### Start Services

```bash
# Start HyperSDK daemon
sudo systemctl start hypervisord
sudo systemctl enable hypervisord

# Start hyper2kvm daemon for vSphere
sudo systemctl start hyper2kvm@vsphere.service
sudo systemctl enable hyper2kvm@vsphere.service

# Check status
systemctl status hypervisord
systemctl status hyper2kvm@vsphere.service
```

## Monitoring & Troubleshooting

### Check Logs

```bash
# HyperSDK logs
journalctl -u hypervisord -f

# hyper2kvm logs
journalctl -u hyper2kvm@vsphere -f

# libvirt logs
journalctl -u libvirtd -f
```

### Common Issues

**Problem:** Job stuck in "pending"
```bash
# Check worker availability
curl http://localhost:8080/status
# → Shows active workers and queue depth
```

**Problem:** Conversion fails
```bash
# Check hyper2kvm daemon
systemctl status hyper2kvm@vsphere.service

# Check watch directory permissions
ls -la /var/lib/hyper2kvm/watch/
```

**Problem:** Cannot connect to provider
```bash
# Validate credentials
curl -X POST http://localhost:8080/providers/validate -d '{
  "provider": "vsphere"
}'
```

## Performance Tuning

### Small Environment (1-10 VMs)
```yaml
daemon:
  workers: 2
export:
  concurrent_exports: 1
  chunk_size: 10485760  # 10MB
```

### Medium Environment (10-50 VMs)
```yaml
daemon:
  workers: 4
export:
  concurrent_exports: 2
  chunk_size: 20971520  # 20MB
```

### Large Environment (50+ VMs)
```yaml
daemon:
  workers: 8
export:
  concurrent_exports: 4
  chunk_size: 52428800  # 50MB
```

## Security Checklist

- [ ] Use HTTPS for daemon API in production
- [ ] Store credentials in environment variables, not config files
- [ ] Enable API authentication (if available)
- [ ] Restrict daemon to internal network
- [ ] Enable audit logging
- [ ] Use RBAC for multi-user environments
- [ ] Regularly rotate provider credentials
- [ ] Monitor webhook endpoints for security

## Next Steps After Meeting

1. **Review Code:**
   - Explore `/daemon/api/hyper2kvm_integration.go`
   - Review `/daemon/jobs/pipeline.go`
   - Check `/providers/common/libvirt.go`

2. **Test Integration:**
   - Run `./examples/hyper2kvm-demo.sh`
   - Try single VM export
   - Test daemon mode

3. **Production Setup:**
   - Configure provider credentials
   - Set up systemd services
   - Configure monitoring/alerting
   - Create backup policies

4. **Documentation:**
   - Read full workflow guide: `docs/integration/hyper2kvm-workflow.md`
   - API reference: `docs/api/`
   - Integration guide: `docs/integration/daemon-integration.md`

## Questions to Discuss in Meeting

1. **Deployment Model:**
   - Will we use daemon mode or direct calls?
   - Single instance or multiple named instances?

2. **Performance Requirements:**
   - How many VMs to migrate?
   - Time window for migrations?
   - Network bandwidth available?

3. **Storage:**
   - Where to store exports (NFS, local, cloud)?
   - Retention policy for exports?
   - Disk space requirements?

4. **Monitoring:**
   - What metrics to track?
   - Alerting requirements?
   - Webhook integrations (Slack, email, etc.)?

5. **Security:**
   - Authentication requirements?
   - Network isolation needed?
   - Audit logging requirements?

## Resources

- **Main Documentation:** `/docs/integration/hyper2kvm-workflow.md`
- **Demo Script:** `/examples/hyper2kvm-demo.sh`
- **API Docs:** `/docs/api/`
- **Example Config:** `/config.example.yaml`
- **Integration Code:** `/daemon/api/hyper2kvm_integration.go`

---

**Quick Test Command:**

```bash
# Test the entire workflow in one command
./examples/hyper2kvm-demo.sh --daemon-url http://localhost:8080

# Or manually:
curl http://localhost:8080/health && \
  echo "✓ Daemon is healthy - ready for demo!"
```
