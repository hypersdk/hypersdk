# Tutorial: End-to-End VM Migration with Pipeline Integration

## Overview

This tutorial demonstrates how to use HyperSDK's complete pipeline to migrate VMs from vSphere to KVM, automatically converting disk formats and preparing VMs for libvirt.

**Pipeline Flow:**
```
vSphere VM → HyperSDK Export → Artifact Manifest → hyper2kvm → libvirt KVM
```

## Prerequisites

### 1. Install Required Software

```bash
# Install HyperSDK
cd /home/ssahani/go/github/hypersdk
go build ./cmd/hyperexport
go build ./cmd/hyperd
sudo cp hyperexport hyperd /usr/local/bin/

# Install hyper2kvm
cd /home/tt/hyper2kvm
go build
sudo cp hyper2kvm /usr/local/bin/

# Install libvirt and QEMU/KVM
sudo apt update
sudo apt install -y qemu-kvm libvirt-daemon-system \
  libvirt-clients bridge-utils virt-manager ovmf
```

### 2. Verify Installation

```bash
# Check HyperSDK
hyperexport --version

# Check hyper2kvm
hyper2kvm --version

# Check libvirt
virsh --version
sudo systemctl status libvirtd
```

### 3. Configure Permissions

```bash
# Add user to libvirt group
sudo usermod -aG libvirt $USER

# Add user to kvm group
sudo usermod -aG kvm $USER

# Log out and back in for groups to take effect
```

## Part 1: Basic Pipeline Export (CLI)

### Step 1: Export VM with Pipeline

Export a VM from vSphere and convert it to KVM format:

```bash
hyperexport \
  --vcenter vcenter.company.com \
  --username administrator@vsphere.local \
  --password 'YourPassword' \
  --vm "/DC1/vm/Ubuntu-Server" \
  --output /var/lib/libvirt/images/ubuntu-server \
  --format ova \
  --manifest \
  --pipeline \
  --hyper2kvm-path /usr/local/bin/hyper2kvm \
  --libvirt \
  --libvirt-uri "qemu:///system" \
  --libvirt-autostart
```

**What happens:**
1. ✅ HyperSDK connects to vSphere
2. ✅ Downloads VM files to `/var/lib/libvirt/images/ubuntu-server/`
3. ✅ Generates Artifact Manifest v1.0 (`manifest.json`)
4. ✅ Calls hyper2kvm to process the VM:
   - **INSPECT**: Detects Ubuntu OS, analyzes drivers
   - **FIX**: Updates fstab, GRUB, initramfs for KVM
   - **CONVERT**: Converts VMDK → qcow2 with compression
   - **VALIDATE**: Verifies image integrity
5. ✅ Defines VM in libvirt
6. ✅ Configures VM for auto-start

**Output:**
```
[INFO] Connecting to vCenter vcenter.company.com...
[INFO] Finding VM /DC1/vm/Ubuntu-Server...
[INFO] Exporting VM Ubuntu-Server...
[PROGRESS] Downloading disk ubuntu-server-disk1.vmdk: 45% [=====>    ] 4.5 GB/10 GB
...
[INFO] Export completed in 8m32s
[INFO] Generating artifact manifest...
[INFO] Manifest saved to /var/lib/libvirt/images/ubuntu-server/manifest.json
[INFO] Starting hyper2kvm pipeline...
[hyper2kvm] INSPECT: Detecting guest OS...
[hyper2kvm] INSPECT: OS detected: Ubuntu 20.04 LTS
[hyper2kvm] FIX: Fixing fstab entries (UUID → /dev/vda)...
[hyper2kvm] FIX: Updating GRUB configuration...
[hyper2kvm] FIX: Rebuilding initramfs with virtio drivers...
[hyper2kvm] CONVERT: Converting VMDK → qcow2 (compression level 6)...
[hyper2kvm] CONVERT: Output: /var/lib/libvirt/images/ubuntu-server.qcow2
[hyper2kvm] VALIDATE: Checking image integrity...
[hyper2kvm] VALIDATE: Image validation successful
[INFO] Pipeline completed in 12m15s
[INFO] Defining VM in libvirt...
[INFO] VM 'ubuntu-server' defined successfully
[INFO] Auto-start enabled

✓ Migration completed successfully!

  Output directory: /var/lib/libvirt/images/ubuntu-server
  Converted image:   /var/lib/libvirt/images/ubuntu-server.qcow2
  Libvirt domain:    ubuntu-server
  Total time:        20m47s

Next steps:
  1. Start the VM: virsh start ubuntu-server
  2. Connect via console: virsh console ubuntu-server
  3. Or use virt-manager for graphical access
```

### Step 2: Start the VM

```bash
# List all VMs
virsh list --all

# Start the VM
virsh start ubuntu-server

# Check VM status
virsh list

# Connect to console
virsh console ubuntu-server
```

### Step 3: Verify the VM

```bash
# Inside the VM console, verify network
ip addr show

# Check disk
lsblk
df -h

# Verify VirtIO drivers
lsmod | grep virtio

# Exit console (Ctrl+])
```

## Part 2: Web Dashboard Pipeline Export

### Step 1: Start the Daemon

```bash
# Start hyperd in background
hyperd &

# Or use systemd
sudo systemctl start hyperd
```

### Step 2: Open Web Dashboard

```bash
# Open browser
firefox http://localhost:8080
```

### Step 3: Configure Export Job

1. **Provider Configuration:**
   - Provider: vSphere
   - vCenter URL: `vcenter.company.com`
   - Username: `administrator@vsphere.local`
   - Password: `YourPassword`
   - Datacenter: `DC1`

2. **VM Selection:**
   - VM Path: `/DC1/vm/Windows-Server`

3. **Export Options:**
   - Output Directory: `/var/lib/libvirt/images/windows-server`
   - Format: `OVA`
   - ✓ Generate Artifact Manifest

4. **Pipeline Integration:** (Enable checkbox)
   - hyper2kvm Path: `/usr/local/bin/hyper2kvm`
   - Compression Level: `6`
   - ✓ INSPECT (detect OS)
   - ✓ FIX (fstab, grub)
   - ✓ CONVERT (→ qcow2)
   - ✓ VALIDATE (integrity)

5. **Libvirt Integration:** (Enable checkbox)
   - Libvirt URI: `qemu:///system`
   - Network Bridge: `virbr0`
   - Storage Pool: `default`
   - ✓ Enable VM auto-start

### Step 4: Submit Job

Click **"Submit Job"** button.

**Job Status Updates:**
```
[10:00:00] Job queued (ID: job-abc123)
[10:00:05] Connecting to vCenter...
[10:00:10] Exporting VM... 0%
[10:05:30] Exporting VM... 50%
[10:10:45] Export completed, generating manifest...
[10:11:00] Running hyper2kvm pipeline: INSPECT stage
[10:12:15] Running hyper2kvm pipeline: FIX stage
[10:15:30] Running hyper2kvm pipeline: CONVERT stage
[10:25:00] Running hyper2kvm pipeline: VALIDATE stage
[10:26:00] Defining VM in libvirt...
[10:26:15] Job completed successfully
```

### Step 5: Check Results

Click on job ID to view details:

```json
{
  "id": "job-abc123",
  "status": "completed",
  "progress": 100,
  "result": {
    "output_path": "/var/lib/libvirt/images/windows-server",
    "format": "ova",
    "size": 21474836480,
    "duration": "26m15s",
    "metadata": {
      "manifest_path": "/var/lib/libvirt/images/windows-server/manifest.json",
      "pipeline_success": true,
      "pipeline_duration": "15m15s",
      "converted_path": "/var/lib/libvirt/images/windows-server.qcow2",
      "libvirt_domain": "windows-server",
      "libvirt_uri": "qemu:///system",
      "pipeline_stages": {
        "inspect": "completed",
        "fix": "completed",
        "convert": "completed",
        "validate": "completed"
      }
    }
  }
}
```

## Part 3: Batch Migration

### Step 1: Create Migration Plan

Create `migration-plan.yaml`:

```yaml
jobs:
  - name: "Web Server 01"
    provider: vsphere
    vcenter_url: vcenter.company.com
    datacenter: DC1
    username: administrator@vsphere.local
    password: "YourPassword"
    vm_path: "/DC1/vm/Production/web-01"
    output_dir: /var/lib/libvirt/images/web-01
    format: ova
    options:
      parallel_downloads: 3
      remove_cdrom: true
      enable_pipeline: true
      hyper2kvm_path: /usr/local/bin/hyper2kvm
      pipeline_inspect: true
      pipeline_fix: true
      pipeline_convert: true
      pipeline_validate: true
      pipeline_compress: true
      compress_level: 6
      libvirt_integration: true
      libvirt_uri: "qemu:///system"
      libvirt_autostart: true
      libvirt_bridge: "br0"

  - name: "Web Server 02"
    provider: vsphere
    vcenter_url: vcenter.company.com
    datacenter: DC1
    username: administrator@vsphere.local
    password: "YourPassword"
    vm_path: "/DC1/vm/Production/web-02"
    output_dir: /var/lib/libvirt/images/web-02
    format: ova
    options:
      enable_pipeline: true
      libvirt_integration: true

  - name: "Database Server"
    provider: vsphere
    vcenter_url: vcenter.company.com
    datacenter: DC1
    username: administrator@vsphere.local
    password: "YourPassword"
    vm_path: "/DC1/vm/Production/db-01"
    output_dir: /var/lib/libvirt/images/db-01
    format: ova
    options:
      enable_pipeline: true
      libvirt_integration: true
      libvirt_autostart: false  # Don't auto-start database
```

### Step 2: Submit Batch Jobs

```bash
# Using hyperctl
hyperctl jobs submit migration-plan.yaml

# Or via API
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/yaml" \
  --data-binary @migration-plan.yaml
```

### Step 3: Monitor Progress

```bash
# Watch all jobs
hyperctl jobs list --watch

# Check specific job
hyperctl jobs get job-abc123

# JSON output for automation
hyperctl jobs list --all --json | jq '.jobs[] | {name, status, progress}'
```

## Part 4: Advanced Scenarios

### Scenario 1: Windows VM with UEFI

```bash
hyperexport \
  --vm "/DC1/vm/Windows-Server-2022" \
  --output /var/lib/libvirt/images/win2022 \
  --format ova \
  --manifest \
  --pipeline \
  --libvirt
```

**hyper2kvm detects UEFI firmware and:**
- Generates UEFI-compatible domain XML
- Uses OVMF firmware (`/usr/share/OVMF/OVMF_CODE.fd`)
- Configures q35 machine type
- Preserves Secure Boot settings

**Libvirt XML generated:**
```xml
<domain type='kvm'>
  <name>win2022</name>
  <os>
    <type arch='x86_64' machine='q35'>hvm</type>
    <loader readonly='yes' type='pflash'>/usr/share/OVMF/OVMF_CODE.fd</loader>
    <nvram>/var/lib/libvirt/qemu/nvram/win2022_VARS.fd</nvram>
  </os>
  <!-- ... -->
</domain>
```

### Scenario 2: Multi-Disk VM

```bash
hyperexport \
  --vm "/DC1/vm/Database-Server" \
  --output /var/lib/libvirt/images/database \
  --format ova \
  --manifest \
  --pipeline \
  --libvirt
```

**hyper2kvm processes all disks:**
- Converts each VMDK to qcow2
- Preserves boot order
- Maps disks to virtio devices (vda, vdb, vdc, ...)
- Maintains disk roles (boot, data, logs)

**Generated files:**
```
/var/lib/libvirt/images/database/
  ├── database-disk-boot.qcow2     (vda - boot disk)
  ├── database-disk-data.qcow2     (vdb - data disk)
  ├── database-disk-logs.qcow2     (vdc - logs disk)
  └── manifest.json
```

### Scenario 3: Dry-Run Before Migration

Test the pipeline without making changes:

```bash
hyperexport \
  --vm "/DC1/vm/Production-App" \
  --output /tmp/test-export \
  --format ova \
  --manifest \
  --pipeline \
  --pipeline-dry-run
```

**Dry-run output:**
```
[DRY-RUN] Would export VM to /tmp/test-export
[DRY-RUN] Would generate manifest
[DRY-RUN] Would run hyper2kvm pipeline:
  - INSPECT: Detect OS
  - FIX: Update fstab, grub
  - CONVERT: VMDK → qcow2 (compression: 6)
  - VALIDATE: Check integrity
[DRY-RUN] Would define in libvirt as 'production-app'

No changes made (dry-run mode)
```

## Troubleshooting

### Pipeline Fails: hyper2kvm Not Found

**Error:**
```
Error: hyper2kvm not found at /usr/local/bin/hyper2kvm
```

**Solution:**
```bash
# Verify hyper2kvm location
which hyper2kvm

# Update path in command
hyperexport ... --hyper2kvm-path /path/to/hyper2kvm
```

### Libvirt Integration Fails: Permission Denied

**Error:**
```
Error: virsh define failed: permission denied
```

**Solution:**
```bash
# Check user groups
groups

# Add to libvirt group if missing
sudo usermod -aG libvirt $USER

# Log out and back in

# Or use session URI
hyperexport ... --libvirt-uri "qemu:///session"
```

### UEFI Firmware Not Found

**Error:**
```
Error: /usr/share/OVMF/OVMF_CODE.fd not found
```

**Solution:**
```bash
# Install OVMF firmware
sudo apt install ovmf          # Ubuntu/Debian
sudo dnf install edk2-ovmf     # RHEL/Fedora
```

### Disk Conversion Timeout

**Error:**
```
Error: pipeline failed: context deadline exceeded
```

**Solution:**
```bash
# Increase timeout
hyperexport ... --pipeline-timeout 1h
```

## Best Practices

### 1. Use Dedicated Storage Pool

```bash
# Create libvirt storage pool
sudo virsh pool-define-as migrated-vms dir - - - - /var/lib/libvirt/images/migrated
sudo virsh pool-build migrated-vms
sudo virsh pool-start migrated-vms
sudo virsh pool-autostart migrated-vms

# Use in export
hyperexport ... --output /var/lib/libvirt/images/migrated/vm-name
```

### 2. Test with Snapshot First

```bash
# Create snapshot of production VM in vSphere
govc snapshot.create -vm prod-vm test-snapshot

# Export snapshot instead of running VM
hyperexport --vm "/DC1/vm/prod-vm" --snapshot test-snapshot ...
```

### 3. Use Compression for Storage Efficiency

```bash
hyperexport ... --pipeline --compress-level 9
```

**Compression levels:**
- 1: Fastest, least compression
- 6: Default, balanced
- 9: Slowest, maximum compression

### 4. Verify After Migration

```bash
# Start VM
virsh start vm-name

# Check console
virsh console vm-name

# Verify inside VM:
# - Network connectivity
# - Disk mounting
# - Services running
```

## Next Steps

- [API Reference](API.md) - Integrate pipeline with your tools
- [Web Dashboard Guide](WEB_DASHBOARD.md) - Use the browser interface
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
- [Performance Tuning](PERFORMANCE.md) - Optimize migration speed

## Summary

You've learned to:
- ✅ Export VMs from vSphere with automatic conversion
- ✅ Use hyper2kvm pipeline for disk format conversion and guest OS fixes
- ✅ Define VMs in libvirt ready for KVM
- ✅ Use CLI and web dashboard for migrations
- ✅ Batch migrate multiple VMs
- ✅ Handle advanced scenarios (UEFI, multi-disk, Windows)

The complete HyperSDK pipeline automates the entire migration from vSphere to KVM!
