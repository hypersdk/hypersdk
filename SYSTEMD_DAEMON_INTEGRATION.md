# hyper2kvm Systemd Daemon Integration

## Status: 🚧 In Progress

## Overview

Integration of HyperSDK with hyper2kvm systemd daemon for production deployments. Provides automatic detection, queue-based submission, and fallback to direct execution.

## Architecture

```
┌─────────────────────────────────────────────┐
│            HyperSDK Export                  │
│    (vSphere, AWS, Azure, GCP, Hyper-V)     │
└──────────────────┬──────────────────────────┘
                   │
                   ▼
         ┌─────────────────────┐
         │ Pipeline Executor   │
         │  detectDaemonMode() │
         └──────┬──────────────┘
                │
        ┌───────┴───────┐
        │               │
        ▼               ▼
┌──────────────┐  ┌──────────────┐
│  Direct Mode │  │  Daemon Mode │
│              │  │              │
│ Execute      │  │ Submit to    │
│ hyper2kvm    │  │ watch dir    │
│ binary       │  │              │
└──────┬───────┘  └──────┬───────┘
       │                 │
       │                 ▼
       │         ┌──────────────┐
       │         │ systemd      │
       │         │ hyper2kvm    │
       │         │ daemon       │
       │         └──────┬───────┘
       │                │
       └────────┬───────┘
                ▼
        ┌──────────────┐
        │ qcow2 Output │
        │ +  Libvirt   │
        └──────────────┘
```

## Implementation Status

### ✅ Completed

1. **Core Pipeline Integration**
   - `providers/common/pipeline.go`:
     - Added `Hyper2KVMConfig` daemon fields
     - Implemented `detectDaemonMode()` - systemctl detection
     - Implemented `ExecuteDirect()` - original direct execution
     - Implemented `ExecuteViaDaemon()` - queue-based submission
     - Auto-fallback to direct mode if daemon unavailable

2. **Export Options**
   - `providers/vsphere/export_options.go`:
     - Added 6 daemon configuration fields
     - Set defaults for daemon paths and timeouts
     - Wired to pipeline executor

3. **API Models**
   - `daemon/models/job.go`:
     - Added daemon options to ExportOptions
     - JSON/YAML serialization support
     - Web API ready

4. **vSphere Provider**
   - `providers/vsphere/export.go`:
     - Daemon options passed to pipeline config
     - Full integration with export workflow

### 🚧 In Progress

5. **CLI Flags** (`cmd/hyperexport/main.go`)
   - TODO: Add command-line flags:
     ```bash
     --hyper2kvm-daemon              # Enable daemon mode
     --hyper2kvm-instance <name>     # Instance name (e.g., "vsphere-prod")
     --hyper2kvm-watch-dir <path>    # Watch directory
     --hyper2kvm-output-dir <path>   # Output directory
     --hyper2kvm-poll-interval <sec> # Poll interval
     --hyper2kvm-daemon-timeout <min># Timeout
     ```

6. **Interactive TUI** (`cmd/hyperctl/interactive.go`)
   - TODO: Add daemon configuration phase
   - TODO: Show daemon status in pipeline options
   - TODO: Allow selection of daemon instance

7. **Web Dashboard** (`web/dashboard-react/src/components/JobSubmissionForm.tsx`)
   - TODO: Add daemon configuration section
   - TODO: Instance selector dropdown
   - TODO: Show daemon status indicator

8. **Documentation**
   - TODO: Update PIPELINE_INTEGRATION.md
   - TODO: Add daemon deployment guide
   - TODO: Add troubleshooting section

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `Hyper2KVMDaemon` | bool | false | Enable daemon mode |
| `Hyper2KVMInstance` | string | "" | Systemd instance name |
| `Hyper2KVMWatchDir` | string | `/var/lib/hyper2kvm/queue` | Watch directory |
| `Hyper2KVMOutputDir` | string | `/var/lib/hyper2kvm/output` | Output directory |
| `Hyper2KVMPollInterval` | int | 5 | Poll interval (seconds) |
| `Hyper2KVMDaemonTimeout` | int | 60 | Timeout (minutes) |

## Usage Examples

### Direct Execution (Current Default)

```bash
hyperexport --vm "Ubuntu-Server" \
  --output /var/lib/libvirt/images/ubuntu \
  --manifest \
  --pipeline \
  --hyper2kvm-path /usr/local/bin/hyper2kvm \
  --libvirt
```

### Daemon Mode (New)

```bash
# Auto-detect daemon
hyperexport --vm "Ubuntu-Server" \
  --output /tmp/export \
  --manifest \
  --pipeline \
  --hyper2kvm-daemon

# Specific instance
hyperexport --vm "Ubuntu-Server" \
  --output /tmp/export \
  --manifest \
  --pipeline \
  --hyper2kvm-daemon \
  --hyper2kvm-instance vsphere-prod
```

### Web API

```json
{
  "name": "Ubuntu Server Migration",
  "vm_path": "/DC1/vm/ubuntu-server",
  "output_dir": "/tmp/export",
  "format": "ova",
  "options": {
    "enable_pipeline": true,
    "hyper2kvm_daemon": true,
    "hyper2kvm_instance": "vsphere-prod",
    "hyper2kvm_watch_dir": "/var/lib/hyper2kvm/vsphere-queue",
    "hyper2kvm_output_dir": "/var/lib/hyper2kvm/vsphere-output",
    "libvirt_integration": true
  }
}
```

## Daemon Detection Logic

1. Check if `hyper2kvm_daemon` option is enabled
2. If enabled:
   - Determine service name:
     - If `instance` specified: `hyper2kvm@{instance}.service`
     - Otherwise: `hyper2kvm.service`
   - Run: `systemctl is-active {service}`
   - If active: Use daemon mode
   - If inactive: Log warning, fall back to direct mode
3. If disabled: Use direct mode

## Queue-Based Submission

### Submit Phase

1. Verify watch directory exists
2. Verify output directory exists
3. Extract VM name from manifest
4. Copy manifest to watch directory: `{watch_dir}/manifest.json`
5. Daemon detects new manifest file

### Polling Phase

1. Poll every `poll_interval` seconds (default: 5s)
2. Check for output file: `{output_dir}/{vm_name}.qcow2`
3. Check for error file: `{output_dir}/{vm_name}.error`
4. Timeout after `daemon_timeout` minutes (default: 60m)
5. If output found: Success
6. If error found: Return error
7. If timeout: Return timeout error

## Deployment Scenarios

### Scenario 1: Single Server (Auto-Detect)

```bash
# Start daemon
sudo systemctl enable --now hyper2kvm.service

# HyperSDK auto-detects and uses daemon
hyperexport --vm test-vm --output /tmp/test \
  --pipeline --hyper2kvm-daemon
```

### Scenario 2: Multi-Instance (Provider-Specific)

```bash
# vSphere instance
sudo systemctl start hyper2kvm@vsphere-prod.service

# Azure instance
sudo systemctl start hyper2kvm@azure-batch.service

# HyperSDK selects instance
hyperexport --provider vsphere --vm test-vm \
  --pipeline --hyper2kvm-daemon \
  --hyper2kvm-instance vsphere-prod
```

### Scenario 3: Dedicated Conversion Server

```bash
# Server 1: Export only (no daemon)
hyperexport --vm test-vm \
  --output /mnt/nfs/queue \
  --manifest \
  --pipeline=false

# Server 2: Daemon watches NFS
# /etc/hyper2kvm/hyper2kvm.conf:
# watch_dir: /mnt/nfs/queue
# output_dir: /mnt/nfs/output

sudo systemctl start hyper2kvm.service
```

## Implementation Tasks

### High Priority

- [ ] Add CLI flags to `cmd/hyperexport/main.go`
- [ ] Add TUI options to `cmd/hyperctl/interactive.go`
- [ ] Add web dashboard UI to `JobSubmissionForm.tsx`
- [ ] Update documentation

### Medium Priority

- [ ] Add daemon status command (`hyperctl daemon status`)
- [ ] Add daemon instance listing (`hyperctl daemon list`)
- [ ] Add job queue monitoring

### Low Priority

- [ ] Add systemd unit files to repository
- [ ] Add RPM/DEB packaging for systemd units
- [ ] Add Ansible playbook for daemon setup

## Benefits

1. **Reliability**: systemd auto-restart on failure
2. **Resource Management**: Memory/CPU limits via systemd
3. **Monitoring**: `systemctl status`, `journalctl` integration
4. **Multi-Tenant**: Multiple instances with different configs
5. **Security**: Runs as dedicated `hyper2kvm` user
6. **Batch Processing**: Queue-based processing of multiple VMs
7. **Separation of Concerns**: Export and conversion can run on different servers

## Testing

```bash
# Test direct mode
hyperexport --vm test-vm --output /tmp/test --pipeline

# Test daemon mode (with daemon running)
hyperexport --vm test-vm --output /tmp/test \
  --pipeline --hyper2kvm-daemon

# Test daemon mode (without daemon - should fallback)
sudo systemctl stop hyper2kvm.service
hyperexport --vm test-vm --output /tmp/test \
  --pipeline --hyper2kvm-daemon
```

## Next Steps

1. **Add CLI Flags**: Complete command-line interface
2. **Add TUI Options**: Interactive mode configuration
3. **Add Web UI**: Browser-based daemon configuration
4. **Documentation**: Update guides and tutorials
5. **Testing**: Integration tests with systemd
6. **Packaging**: RPM/DEB packages with systemd units

## See Also

- [PIPELINE_INTEGRATION.md](PIPELINE_INTEGRATION.md) - Pipeline architecture
- [hyper2kvm systemd units](https://github.com/ssahani/hyper2kvm) - Daemon service files
- [TUTORIAL_PIPELINE.md](docs/TUTORIAL_PIPELINE.md) - End-to-end tutorial
