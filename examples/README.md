# hyperctl Examples

This directory contains example configuration files for hyperctl VM export operations.

## 📋 Available Examples

### Single VM Export

**YAML Format** (`example-vm-export.yaml`):
```yaml
name: export-rhel9
vm_path: /data/vm/Auto-esx8.0-rhel8.9-with-multiple-snapshots
output_path: /tmp/test-export
options:
  parallel_downloads: 4
  remove_cdrom: true
  show_individual_progress: false
```

**JSON Format** (`example-vm-export.json`):
```json
{
  "name": "export-rhel9",
  "vm_path": "/data/vm/Auto-esx8.0-rhel8.9-with-multiple-snapshots",
  "output_path": "/tmp/test-export",
  "options": {
    "parallel_downloads": 4,
    "remove_cdrom": true,
    "show_individual_progress": false
  }
}
```

**Usage:**
```bash
# Using YAML
hyperctl submit -file examples/example-vm-export.yaml

# Using JSON
hyperctl submit -file examples/example-vm-export.json
```

### Batch VM Export

**YAML Format** (`example-batch-export.yaml`):
```yaml
jobs:
  - name: export-win2022
    vm_path: /data/vm/Auto-esx8.0-win2022
    output_path: /tmp/exports/win2022
    options:
      parallel_downloads: 8
      remove_cdrom: true
      show_individual_progress: true

  - name: export-rhel8
    vm_path: /data/vm/Auto-esx8.0-rhel8.9-with-multiple-snapshots
    output_path: /tmp/exports/rhel8
    options:
      parallel_downloads: 4
      remove_cdrom: true

  - name: export-opensuse
    vm_path: /data/vm/Auto-esx8.0-opensuse15.6-x86_64-efi
    output_path: /tmp/exports/opensuse
    options:
      parallel_downloads: 4
```

**JSON Format** (`example-batch-export.json`):
```json
{
  "jobs": [
    {
      "name": "export-win2022",
      "vm_path": "/data/vm/Auto-esx8.0-win2022",
      "output_path": "/tmp/exports/win2022",
      "options": {
        "parallel_downloads": 8,
        "remove_cdrom": true,
        "show_individual_progress": true
      }
    },
    {
      "name": "export-rhel8",
      "vm_path": "/data/vm/Auto-esx8.0-rhel8.9-with-multiple-snapshots",
      "output_path": "/tmp/exports/rhel8",
      "options": {
        "parallel_downloads": 4,
        "remove_cdrom": true
      }
    },
    {
      "name": "export-opensuse",
      "vm_path": "/data/vm/Auto-esx8.0-opensuse15.6-x86_64-efi",
      "output_path": "/tmp/exports/opensuse",
      "options": {
        "parallel_downloads": 4
      }
    }
  ]
}
```

**Usage:**
```bash
# Using YAML
hyperctl submit -file examples/example-batch-export.yaml

# Using JSON
hyperctl submit -file examples/example-batch-export.json
```

## 🚀 Complete Migration Workflow

### 1. Discover Available VMs

```bash
# List all VMs
hyperctl list

# Find specific VMs
hyperctl list -filter rhel

# Export to JSON for automation
hyperctl list -json > available-vms.json
```

### 2. Prepare VM for Migration

```bash
# Get VM information
hyperctl vm -op info -path /data/vm/my-vm

# Shutdown VM gracefully (optional but recommended)
hyperctl vm -op shutdown -path /data/vm/my-vm -timeout 300

# Remove CD/DVD devices (required for clean KVM import)
hyperctl vm -op remove-cdrom -path /data/vm/my-vm
```

### 3. Submit Export Job

**Option A: Command Line**
```bash
hyperctl submit \
  -vm /data/vm/my-vm \
  -output /tmp/export-my-vm
```

**Option B: Using YAML File**
```bash
# Create job file
cat > my-export.yaml <<EOF
name: export-my-vm
vm_path: /data/vm/my-vm
output_path: /tmp/export-my-vm
options:
  parallel_downloads: 8
  remove_cdrom: false  # Already removed in step 2
  show_individual_progress: true
EOF

# Submit job
hyperctl submit -file my-export.yaml
```

### 4. Monitor Export Progress

```bash
# Query specific job (use Job ID from submit output)
hyperctl query -id abc123-def456-...

# Watch all running jobs
watch -n 5 'hyperctl query -status running'

# Get detailed status via API
curl -s http://localhost:8080/jobs/{job-id} | jq
```

### 5. Verify Exported Files

```bash
# Check output directory
ls -lh /tmp/export-my-vm/

# Expected files:
# - *.ovf    (OVF descriptor)
# - *.vmdk   (virtual disk files)
# - *.mf     (manifest file)
```

### 6. Convert to qcow2 (for KVM)

```bash
# Convert VMDK to qcow2
qemu-img convert -f vmdk -O qcow2 \
  /tmp/export-my-vm/*.vmdk \
  /tmp/export-my-vm/disk.qcow2

# Verify conversion
qemu-img info /tmp/export-my-vm/disk.qcow2
```

### 7. Import to KVM/libvirt

```bash
# Create new VM with converted disk
virt-install \
  --name my-vm \
  --memory 4096 \
  --vcpus 2 \
  --disk /tmp/export-my-vm/disk.qcow2,bus=virtio \
  --import \
  --os-variant rhel9.0 \
  --network bridge=virbr0 \
  --graphics vnc

# Verify VM
virsh list --all
virsh dominfo my-vm
```

## 📊 Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `name` | string | - | Human-readable job name |
| `vm_path` | string | - | Full path to VM in vCenter (e.g., /data/vm/my-vm) |
| `output_path` | string | - | Local directory to save exported files |
| `options.parallel_downloads` | int | 4 | Number of parallel download streams (1-16) |
| `options.remove_cdrom` | bool | false | Remove CD/DVD devices before export |
| `options.show_individual_progress` | bool | false | Show progress for each disk separately |

## 💡 Tips & Best Practices

1. **Performance Tuning:**
   - Use `parallel_downloads: 4` for small VMs (<20GB)
   - Use `parallel_downloads: 8` for medium VMs (20-100GB)
   - Use `parallel_downloads: 16` for large VMs (>100GB)

2. **CD/DVD Handling:**
   - Always remove CD/DVD devices before migration
   - Use `remove_cdrom: true` in options OR
   - Run `hyperctl vm -op remove-cdrom` before export

3. **Batch Operations:**
   - Group similar VMs together in batch files
   - Stagger exports to avoid overloading storage
   - Monitor network and disk I/O during exports

4. **Error Handling:**
   - Check daemon logs: `sudo journalctl -u hypervisord -f`
   - Query failed jobs: `hyperctl query -status failed`
   - Retry failed exports with same configuration

## 🔗 Related Documentation

- [Interactive Mode Guide](../docs/user-guides/01-interactive-mode.md)
- [Hyperctl Features](../docs/features/02-hyperctl-features.md)
- [VM Export Guide](../docs/user-guides/02-vm-export-guide.md)
- [Daemon API Reference](../docs/api/01-daemon-api.md)
- [Getting Started](../GETTING-STARTED.md)
- [Documentation Index](../docs/00-INDEX.md)

## 📝 Creating Your Own Job Files

### From VM List Output

```bash
# Generate job file from VM list
hyperctl list -filter production -json | \
  jq -r '.vms[] | {
    name: .name,
    vm_path: .path,
    output_path: ("/migrations/" + .name),
    options: {parallel_downloads: 4, remove_cdrom: true}
  }' | \
  jq -s '{jobs: .}' > production-export.json
```

### Custom Template

```bash
# Create template function
create_export_job() {
  local vm_name=$1
  local vm_path=$2
  local output_path=$3

  cat > "export-${vm_name}.yaml" <<EOF
name: export-${vm_name}
vm_path: ${vm_path}
output_path: ${output_path}
options:
  parallel_downloads: 8
  remove_cdrom: true
  show_individual_progress: true
EOF
}

# Use template
create_export_job "web-server" "/data/vm/web-01" "/migrations/web-01"
hyperctl submit -file export-web-server.yaml
```

---

**🎉 Happy Migrating with hyperctl!**
