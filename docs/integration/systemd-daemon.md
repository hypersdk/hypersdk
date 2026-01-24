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

### ✅ Completed (Continued)

5. **CLI Flags** (`cmd/hyperexport/main.go`)
   - Added 6 command-line flags:
     ```bash
     --hyper2kvm-daemon              # Enable daemon mode
     --hyper2kvm-instance <name>     # Instance name (e.g., "vsphere-prod")
     --hyper2kvm-watch-dir <path>    # Watch directory (default: /var/lib/hyper2kvm/queue)
     --hyper2kvm-output-dir <path>   # Output directory (default: /var/lib/hyper2kvm/output)
     --hyper2kvm-poll-interval <sec> # Poll interval (default: 5)
     --hyper2kvm-daemon-timeout <min># Timeout (default: 60)
     ```
   - Wired flags to export options

6. **Interactive TUI** (`cmd/hyperexport/interactive_huh.go`)
   - Added daemon configuration phase with conditional display
   - Daemon mode toggle (Yes/No confirmation)
   - Advanced daemon configuration (hidden until user opts in)
   - 5 configuration inputs with validation:
     * Daemon instance name
     * Watch directory
     * Output directory
     * Poll interval (1-60 seconds)
     * Timeout (1-240 minutes)
   - Daemon settings shown in export summary
   - Full integration with export workflow

7. **Web Dashboard** (`web/dashboard-react/src/components/JobSubmissionForm.tsx`)
   - Added daemon configuration section in Pipeline Integration
   - Checkbox to enable daemon mode
   - Collapsible daemon settings (shown when enabled):
     * Instance name input with hint text
     * Watch directory input
     * Output directory input
     * Poll interval number input (1-60)
     * Timeout number input (1-240)
   - Added 6 daemon fields to form state with defaults
   - Submitted with job data to API

8. **Daemon Management Commands** (`cmd/hyperctl/daemon_commands.go`, `cmd/hyperctl/main.go`)
   - Added `hyperctl daemon` command with two operations:
     * `hyperctl daemon -op status` - Show status of all daemon instances
     * `hyperctl daemon -op status -instance <name>` - Show specific instance status
     * `hyperctl daemon -op list` - List all daemon instances
   - Displays instance information:
     * Instance name and service name
     * Active/inactive status
     * Process ID (PID)
     * Uptime
     * Watch and output directories
   - Uses systemctl commands to query daemon status
   - Supports both default (hyper2kvm.service) and named instances (hyper2kvm@name.service)
   - Added documentation to hyperctl help output

9. **Systemd Unit Files** (`systemd/` directory)
   - Created production-ready systemd service files:
     * `hyper2kvm.service` - Default daemon instance
     * `hyper2kvm@.service` - Template for named instances
     * `hyper2kvm.target` - Target to manage all instances
   - Example configuration files:
     * `hyper2kvm.conf.example` - Default configuration
     * `hyper2kvm-vsphere.conf.example` - vSphere-specific config
     * `hyper2kvm-aws.conf.example` - AWS-specific config
   - Security hardening features:
     * Runs as non-root `hyper2kvm` user
     * Resource limits (memory, CPU)
     * Restricted filesystem access
     * Minimal capabilities
     * System call filtering
   - Created `install.sh` script for automated deployment
   - Comprehensive `README.md` with:
     * Installation instructions
     * Usage examples
     * Management commands
     * Troubleshooting guide
     * Multi-instance setup

### 🚧 In Progress

10. **Documentation**
   - TODO: Update PIPELINE_INTEGRATION.md
   - TODO: Add troubleshooting section to main docs

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

- [x] Add CLI flags to `cmd/hyperexport/main.go`
- [x] Add TUI options to `cmd/hyperexport/interactive_huh.go`
- [x] Add web dashboard UI to `JobSubmissionForm.tsx`
- [ ] Update documentation

### Medium Priority

- [x] Add daemon status command (`hyperctl daemon status`)
- [x] Add daemon instance listing (`hyperctl daemon list`)
- [ ] Add job queue monitoring

### Low Priority

- [x] Add systemd unit files to repository
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

# Check daemon status with hyperctl
hyperctl daemon -op status                    # Show all instances
hyperctl daemon -op status -instance vsphere  # Show specific instance
hyperctl daemon -op list                      # List all instances

# Start/stop daemon instances
sudo systemctl start hyper2kvm.service              # Start default
sudo systemctl start hyper2kvm@vsphere.service      # Start named instance
sudo systemctl stop hyper2kvm@vsphere.service       # Stop named instance
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
