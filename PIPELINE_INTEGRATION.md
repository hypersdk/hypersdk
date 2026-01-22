# HyperSDK Pipeline Integration with hyper2kvm and libvirt

## Overview

The HyperSDK export process now integrates seamlessly with hyper2kvm for VM conversion and libvirt for VM management. This creates a complete end-to-end pipeline:

**Export → Convert → Define in Libvirt**

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐     ┌──────────┐
│   vSphere   │────>│  HyperSDK    │────>│ hyper2kvm  │────>│ libvirt  │
│     VM      │     │   Export     │     │  Convert   │     │   KVM    │
└─────────────┘     └──────────────┘     └────────────┘     └──────────┘
                           │
                           v
                    Artifact Manifest v1.0
```

## Components

### 1. Pipeline Executor (`providers/common/pipeline.go`)

Executes the hyper2kvm pipeline after export completes:
- Locates hyper2kvm binary (searches common paths)
- Executes hyper2kvm with the manifest file
- Captures and streams output
- Handles timeouts and errors gracefully
- Returns pipeline results (success, duration, output path)

### 2. Libvirt Integrator (`providers/common/libvirt.go`)

Integrates converted VMs with libvirt:
- Generates libvirt domain XML from manifest metadata
- Detects firmware type (BIOS/UEFI) and configures appropriately
- Defines VM in libvirt using `virsh define`
- Supports auto-start configuration
- Handles network bridge and storage pool configuration

### 3. Export Options (`providers/vsphere/export_options.go`)

Extended with pipeline configuration:

```go
type ExportOptions struct {
    // ... existing fields ...

    // Pipeline integration
    EnablePipeline         bool
    Hyper2KVMPath          string
    PipelineTimeout        time.Duration
    StreamPipelineOutput   bool
    PipelineDryRun         bool

    // Pipeline stages
    PipelineInspect        bool
    PipelineFix            bool
    PipelineConvert        bool
    PipelineValidate       bool
    PipelineCompress       bool
    PipelineCompressLevel  int

    // Libvirt integration
    LibvirtIntegration     bool
    LibvirtURI             string
    LibvirtAutoStart       bool
    LibvirtNetworkBridge   string
    LibvirtStoragePool     string
}
```

### 4. CLI Flags (`cmd/hyperexport/main.go`)

New command-line options:

```bash
# Pipeline control
--pipeline                     # Enable pipeline integration
--hyper2kvm-path <path>        # Path to hyper2kvm executable
--pipeline-timeout <duration>  # Pipeline execution timeout
--stream-pipeline              # Stream hyper2kvm output
--pipeline-dry-run             # Run in dry-run mode

# Pipeline stages
--pipeline-inspect             # Enable INSPECT stage
--pipeline-fix                 # Enable FIX stage
--pipeline-convert             # Enable CONVERT stage
--pipeline-validate            # Enable VALIDATE stage
--pipeline-compress            # Enable qcow2 compression
--compress-level <1-9>         # Compression level

# Libvirt integration
--libvirt                      # Define VM in libvirt
--libvirt-uri <uri>            # Libvirt connection URI
--libvirt-autostart            # Enable VM auto-start
--libvirt-bridge <bridge>      # Network bridge
--libvirt-pool <pool>          # Storage pool
```

## Usage Examples

### Example 1: Export with Conversion

```bash
hyperexport \
  --vm "Ubuntu-Server" \
  --output /var/lib/libvirt/images/ubuntu \
  --pipeline \
  --hyper2kvm-path /home/tt/hyper2kvm/hyper2kvm \
  --manifest
```

This will:
1. Export the VM from vSphere to `/var/lib/libvirt/images/ubuntu`
2. Generate an Artifact Manifest v1.0
3. Run hyper2kvm to convert VMDK → qcow2
4. Fix fstab, GRUB, and initramfs for KVM
5. Validate the converted image

### Example 2: Export with Libvirt Integration

```bash
hyperexport \
  --vm "Ubuntu-Server" \
  --output /var/lib/libvirt/images/ubuntu \
  --pipeline \
  --libvirt \
  --libvirt-uri "qemu:///system" \
  --libvirt-autostart \
  --libvirt-bridge br0 \
  --manifest
```

This will:
1. Export and convert the VM (as above)
2. Define the VM in libvirt
3. Configure it to use bridge `br0`
4. Enable auto-start on boot
5. VM is ready to start with `virsh start ubuntu-server`

### Example 3: Quick Conversion (No Fix)

```bash
hyperexport \
  --vm "Test-VM" \
  --output /tmp/test \
  --pipeline \
  --pipeline-inspect=false \
  --pipeline-fix=false \
  --pipeline-validate=false \
  --manifest
```

This will only convert the disk format without fixing the guest OS.

### Example 4: Dry-Run Pipeline

```bash
hyperexport \
  --vm "Production-VM" \
  --output /tmp/prod \
  --pipeline \
  --pipeline-dry-run \
  --manifest
```

This will show what the pipeline would do without making any changes.

## Manifest Integration

The pipeline uses the **Artifact Manifest v1.0** as the integration contract:

```json
{
  "manifest_version": "1.0",
  "source": {
    "provider": "vsphere",
    "vm_name": "Ubuntu-Server",
    "datacenter": "DC1"
  },
  "vm": {
    "cpu": 4,
    "mem_gb": 8,
    "firmware": "bios",
    "os_hint": "linux"
  },
  "disks": [
    {
      "id": "disk-0",
      "source_format": "vmdk",
      "bytes": 107374182400,
      "local_path": "/path/to/disk.vmdk",
      "boot_order_hint": 0,
      "disk_type": "boot"
    }
  ],
  "pipeline": {
    "inspect": {"enabled": true},
    "fix": {"enabled": true, "fstab_mode": "stabilize-all"},
    "convert": {"enabled": true, "compress": true},
    "validate": {"enabled": true}
  },
  "output": {
    "directory": "/var/lib/libvirt/images/ubuntu",
    "format": "qcow2"
  }
}
```

## Pipeline Flow

1. **Export Phase** (HyperSDK)
   - Export VM from vSphere
   - Generate Artifact Manifest v1.0
   - Write manifest to `<output>/manifest.json`

2. **Pipeline Execution** (hyper2kvm)
   - Reads manifest from disk
   - Executes configured stages:
     - **INSPECT**: Detect OS, analyze drivers
     - **FIX**: Fix fstab, GRUB, initramfs, remove VMware tools
     - **CONVERT**: Convert VMDK → qcow2 with compression
     - **VALIDATE**: Verify image integrity
   - Writes results to `<output>/report.json`
   - Outputs converted image to `<output>/<vm-name>.qcow2`

3. **Libvirt Integration** (optional)
   - Generate libvirt domain XML
   - Define VM with `virsh define`
   - Configure auto-start if requested
   - VM is ready to use

## Error Handling

The pipeline integration is designed to be **non-fatal**:

- If hyper2kvm is not found, the export succeeds with a warning
- If conversion fails, the export result contains error details
- If libvirt integration fails, it's logged but doesn't fail the export
- All pipeline errors are stored in `result.Metadata`

## Configuration Defaults

```go
// Default pipeline configuration
EnablePipeline:        false  // Opt-in
PipelineTimeout:       30 min
StreamPipelineOutput:  true
PipelineInspect:       true
PipelineFix:           true
PipelineConvert:       true
PipelineValidate:      true
PipelineCompress:      true
PipelineCompressLevel: 6

// Default libvirt configuration
LibvirtURI:            "qemu:///system"
LibvirtNetworkBridge:  "virbr0"
LibvirtStoragePool:    "default"
LibvirtAutoStart:      false
```

## Libvirt XML Generation

The libvirt integrator generates production-ready domain XML:

- **Firmware**: Detects BIOS/UEFI from manifest
- **UEFI**: Uses OVMF firmware (`/usr/share/OVMF/OVMF_CODE.fd`)
- **CPU**: Host passthrough for best performance
- **Disk**: VirtIO SCSI with writeback cache and discard
- **Network**: VirtIO with bridge networking
- **Graphics**: VNC on localhost
- **Devices**: QEMU guest agent, tablet input, video (QXL)
- **RNG**: VirtIO RNG for better entropy

Example generated XML:

```xml
<domain type='kvm'>
  <name>ubuntu-server</name>
  <memory unit='KiB'>8388608</memory>
  <vcpu placement='static'>4</vcpu>
  <os>
    <type arch='x86_64' machine='pc'>hvm</type>
    <boot dev='hd'/>
  </os>
  <cpu mode='host-passthrough'/>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='writeback'/>
      <source file='/var/lib/libvirt/images/ubuntu-server.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <interface type='bridge'>
      <source bridge='virbr0'/>
      <model type='virtio'/>
    </interface>
    <!-- ... more devices ... -->
  </devices>
</domain>
```

## Requirements

### hyper2kvm

The pipeline requires hyper2kvm to be installed. HyperSDK searches:
- `/home/tt/hyper2kvm/hyper2kvm` (default)
- `/usr/local/bin/hyper2kvm`
- `/usr/bin/hyper2kvm`
- `./hyper2kvm`
- `../hyper2kvm/hyper2kvm`
- System PATH

Or specify manually with `--hyper2kvm-path`.

### libvirt

For libvirt integration:
- `virsh` must be in PATH
- libvirt daemon must be running
- User must have permissions to define VMs (or use qemu:///session)
- For UEFI VMs, OVMF firmware must be installed

## Testing

Test the pipeline integration:

```bash
# 1. Export without pipeline (just creates manifest)
hyperexport --vm Test-VM --output /tmp/test --manifest

# 2. Verify manifest
cat /tmp/test/manifest.json

# 3. Export with pipeline (dry-run)
hyperexport --vm Test-VM --output /tmp/test --pipeline --pipeline-dry-run --manifest

# 4. Full pipeline with conversion
hyperexport --vm Test-VM --output /tmp/test --pipeline --manifest

# 5. Pipeline with libvirt integration
hyperexport --vm Test-VM --output /tmp/test --pipeline --libvirt --manifest

# 6. Start the VM
virsh start test-vm
```

## Troubleshooting

### hyper2kvm not found

```
Error: hyper2kvm not found at /home/tt/hyper2kvm/hyper2kvm
```

Solution: Specify path manually:
```bash
--hyper2kvm-path /path/to/hyper2kvm
```

### Libvirt permission denied

```
Error: virsh define failed: permission denied
```

Solution: Use session URI or add user to libvirt group:
```bash
--libvirt-uri "qemu:///session"
# OR
sudo usermod -aG libvirt $USER
```

### Conversion timeout

```
Error: pipeline failed: context deadline exceeded
```

Solution: Increase timeout:
```bash
--pipeline-timeout 1h
```

### UEFI firmware not found

```
Error: /usr/share/OVMF/OVMF_CODE.fd not found
```

Solution: Install OVMF:
```bash
# Ubuntu/Debian
sudo apt install ovmf

# RHEL/Fedora
sudo dnf install edk2-ovmf
```

## Future Enhancements

- [ ] Support for remote libvirt URIs (qemu+ssh://host/system)
- [ ] Storage pool creation and disk image registration
- [ ] VM snapshot creation after successful conversion
- [ ] Parallel VM conversion (batch exports)
- [ ] Progress reporting for hyper2kvm stages
- [ ] Integration with virt-manager for GUI access
- [ ] Cloud-init configuration injection
- [ ] Network configuration preservation

## See Also

- **Artifact Manifest v1.0**: `manifest/types.go`
- **hyper2kvm Documentation**: `/home/tt/hyper2kvm/README.md`
- **libvirt Documentation**: https://libvirt.org/
