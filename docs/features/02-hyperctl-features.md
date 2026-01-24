# hyperctl - Feature Overview

hyperctl is a powerful CLI tool for managing VMware vSphere VMs and orchestrating migrations to KVM.

## 🎯 Core Features

### 1. VM Discovery & Listing

**Command:**
```bash
hyperctl list                    # Show all VMs
hyperctl list -json              # JSON output for automation
hyperctl list -filter <name>     # Filter by VM name
```

**Features:**
- 📊 Rich summary statistics (total VMs, powered on/off, total resources)
- 💻 Detailed VM table (name, power state, CPU, memory, storage, guest OS)
- 🎨 Color-coded power states (green for powered on, gray for off)
- 🔍 Live animated spinner during discovery
- ⚡ Fast filtering by name
- 📄 Clean JSON output for scripting
- 💡 Built-in helpful tips

**Output Example:**
```
✅ Found 201 VMs

📊 VM Summary
┌──────────────┬────────┐
│ 🖥️  Total VMs  │ 201    │
│ ✅ Powered On  │ 45     │
│ ⭕ Powered Off │ 156    │
│ 💾 Total Memory│ 512 GB │
│ ⚡ Total CPUs  │ 384    │
│ 💿 Total Storage│ 12 TB  │
└──────────────┴────────┘

💻 Virtual Machines
┌───┬─────────────────┬───────────┬─────┬────────┬─────────┬──────────────┐
│ # │ Name            │ Power     │ CPU │ Memory │ Storage │ Guest OS     │
├───┼─────────────────┼───────────┼─────┼────────┼─────────┼──────────────┤
│ 1 │ win2022         │ poweredOn │ 2   │ 4.0 GB │ 90 GB   │ Windows 2022 │
│ 2 │ rhel9.4         │ poweredOff│ 1   │ 2.0 GB │ 16 GB   │ RHEL 9.4     │
...
└───┴─────────────────┴───────────┴─────┴────────┴─────────┴──────────────┘

💡 Tip: Use 'hyperctl list -json' for machine-readable output
💡 Tip: Use 'hyperctl list -filter <name>' to filter VMs
```

### 2. VM Operations

**Graceful Shutdown:**
```bash
hyperctl vm -op shutdown -path /data/vm/my-vm
hyperctl vm -op shutdown -path /data/vm/my-vm -timeout 600  # 10 min timeout
```
- 🔌 Initiates guest OS shutdown
- ⏱️  Configurable timeout
- ✅ Waits for VM to power off

**Force Power Off:**
```bash
hyperctl vm -op poweroff -path /data/vm/my-vm
```
- ⚡ Immediate power off
- 🚨 Use when shutdown fails or VM is unresponsive

**Remove CD/DVD Devices:**
```bash
hyperctl vm -op remove-cdrom -path /data/vm/my-vm
```
- 💿 Removes all CD/DVD devices from VM
- 🎯 Essential before migration to KVM
- ✅ Prevents boot issues

**Get VM Information:**
```bash
hyperctl vm -op info -path /data/vm/my-vm
```
- ℹ️  Detailed VM metadata
- 📊 Power state, resources, guest OS
- 🎯 Quick VM inspection

**Output Example:**
```
✅ Retrieved VM info

📋 VM Information
┌─────────────┬────────────────────────────────────────────┐
│ Property    │ Value                                      │
├─────────────┼────────────────────────────────────────────┤
│ Name        │ Auto-esx8.0-rhel8.9-with-snapshots        │
│ Path        │ /data/vm/Auto-esx8.0-rhel8.9-...           │
│ Power State │ poweredOn                                  │
│ Guest OS    │ Red Hat Enterprise Linux 8 (64-bit)        │
│ CPUs        │ 1                                          │
│ Memory      │ 2.0 GB                                     │
│ Storage     │ 16.0 GB                                    │
└─────────────┴────────────────────────────────────────────┘
```

### 3. Job Management

**Submit Export Job:**
```bash
# From command line
hyperctl submit -vm /data/vm/my-vm -output /tmp/export

# From YAML file
hyperctl submit -file jobs.yaml
```

**Query Jobs:**
```bash
hyperctl query -all                    # All jobs
hyperctl query -id abc123              # Specific job
hyperctl query -status running         # Filter by status
```

**Get Job Details:**
```bash
hyperctl jobs/abc123                   # Job details
```

**Cancel Jobs:**
```bash
hyperctl cancel -id abc123,def456      # Cancel multiple jobs
```

**Daemon Status:**
```bash
hyperctl status
```
- 📊 Uptime, total jobs
- 🔄 Running/completed/failed counts
- ✅ Health check

### 4. Beautiful UX

**Features:**
- 🎨 Color-coded output (success=green, error=red, info=cyan)
- 🔄 Animated spinners with elapsed time
- 📊 Beautiful tables with box drawing
- 😀 Emoji-enhanced messages for better visual feedback
- 💡 Helpful tips and suggestions
- ⚡ Progress indicators
- 📝 Clean JSON output for automation (no ANSI codes)

## 📋 Job Definition Examples

### YAML Format

**Single Job:**
```yaml
name: export-rhel9
vm_path: /data/vm/rhel9.4
output_path: /tmp/export-rhel9
options:
  parallel_downloads: 4
  remove_cdrom: true
  show_individual_progress: false
```

**Batch Jobs:**
```yaml
jobs:
  - name: export-vm1
    vm_path: /data/vm/vm1
    output_path: /tmp/export-vm1

  - name: export-vm2
    vm_path: /data/vm/vm2
    output_path: /tmp/export-vm2
    options:
      parallel_downloads: 8
```

### JSON Format

**Single Job:**
```json
{
  "name": "export-rhel9",
  "vm_path": "/data/vm/rhel9.4",
  "output_path": "/tmp/export-rhel9",
  "options": {
    "parallel_downloads": 4,
    "remove_cdrom": true,
    "show_individual_progress": false
  }
}
```

**Batch Jobs:**
```json
{
  "jobs": [
    {
      "name": "export-vm1",
      "vm_path": "/data/vm/vm1",
      "output_path": "/tmp/export-vm1"
    },
    {
      "name": "export-vm2",
      "vm_path": "/data/vm/vm2",
      "output_path": "/tmp/export-vm2",
      "options": {
        "parallel_downloads": 8
      }
    }
  ]
}
```

## 🔧 API Integration

All hyperctl commands interact with hypervisord daemon via REST API:

- `GET /vms/list` - List all VMs
- `POST /vms/shutdown` - Shutdown VM
- `POST /vms/poweroff` - Power off VM
- `POST /vms/remove-cdrom` - Remove CD/DVD
- `POST /vms/info` - Get VM info
- `POST /jobs/submit` - Submit job
- `POST /jobs/query` - Query jobs
- `POST /jobs/cancel` - Cancel jobs
- `GET /jobs/{id}` - Get job details
- `GET /status` - Daemon status
- `GET /health` - Health check

## 🎯 Use Cases

### 1. VM Discovery Workflow
```bash
# Discover all VMs
hyperctl list

# Find Windows VMs
hyperctl list -filter win

# Export list to JSON for processing
hyperctl list -json > vms.json

# Get details of a specific VM
hyperctl vm -op info -path /data/vm/win2022
```

### 2. Pre-Migration Preparation
```bash
# Shutdown VM gracefully
hyperctl vm -op shutdown -path /data/vm/my-vm

# Remove CD/DVD devices
hyperctl vm -op remove-cdrom -path /data/vm/my-vm

# Verify VM is ready
hyperctl vm -op info -path /data/vm/my-vm
```

### 3. Automated Migration
```bash
# Create job file
cat > migrate-batch.yaml <<EOF
jobs:
  - name: migrate-web1
    vm_path: /data/vm/web1
    output_path: /migrations/web1
    options:
      parallel_downloads: 8
      remove_cdrom: true

  - name: migrate-db1
    vm_path: /data/vm/db1
    output_path: /migrations/db1
    options:
      parallel_downloads: 4
EOF

# Submit batch
hyperctl submit -file migrate-batch.yaml

# Monitor progress
watch -n 5 'hyperctl query -status running'
```

## 🚀 Advantages Over govc

1. **Rich UX**: Beautiful colored output, spinners, progress bars
2. **Migration-Focused**: Built specifically for VM migration workflows
3. **Job Management**: Asynchronous job submission and tracking
4. **Batch Operations**: Submit multiple migrations at once
5. **Integrated**: Works seamlessly with hypervisord daemon
6. **Modern**: Clean JSON/YAML support for automation
7. **User-Friendly**: Helpful tips, examples, and error messages

## 📖 Quick Reference

### Common Commands
```bash
# Discovery
hyperctl list                                    # List all VMs
hyperctl list -filter rhel -json                 # Find RHEL VMs (JSON)

# VM Operations
hyperctl vm -op shutdown -path /data/vm/my-vm    # Shutdown
hyperctl vm -op poweroff -path /data/vm/my-vm    # Power off
hyperctl vm -op remove-cdrom -path /data/vm/my-vm# Remove CD
hyperctl vm -op info -path /data/vm/my-vm        # Get info

# Job Management
hyperctl submit -vm /data/vm/my-vm -output /tmp  # Submit job
hyperctl query -all                              # List jobs
hyperctl query -status running                   # Running jobs
hyperctl cancel -id abc123                       # Cancel job
hyperctl status                                  # Daemon status
```

### Environment Variables
```bash
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='password'
export GOVC_INSECURE=1
export GOVC_DATACENTER='datacenter1'
```

## 🔮 Roadmap

### Planned Features
- [ ] Snapshot management
- [ ] Network configuration
- [ ] Datastore browsing
- [ ] vCenter information
- [ ] ESXi host information
- [ ] Resource pool management
- [ ] VM cloning
- [ ] Template management
- [ ] Performance metrics
- [ ] Event monitoring

## 📚 Documentation

- [Getting Started](../getting-started.md)
- [Daemon Configuration](../DAEMON-README.md)
- [API Reference](../docs/API.md)
- [Job Configuration](../docs/JOBS.md)

## 🤝 Contributing

This tool is part of the hyper2kvm project. Contributions welcome!

---

**Built with ❤️ using Go and pterm**
