# VM Export Guide with hyperctl

Complete guide to exporting VMware VMs using hyperctl and hypervisord.

## 🎯 Overview

The hyperctl tool provides a powerful, user-friendly way to export VMs from vSphere to local storage using VDDK (Virtual Disk Development Kit) for high-performance parallel downloads.

## 📋 Prerequisites

1. **hypervisord daemon running:**
   ```bash
   sudo systemctl status hypervisord
   ```

2. **vCenter credentials configured:**
   ```bash
   export GOVC_URL='https://vcenter.example.com/sdk'
   export GOVC_USERNAME='administrator@vsphere.local'
   export GOVC_PASSWORD='your-password'
   export GOVC_INSECURE=1
   export GOVC_DATACENTER='datacenter1'
   ```

3. **VDDK libraries installed:**
   ```bash
   ls /usr/lib/vmware-vix-disklib/lib64/
   ```

## 🚀 Quick Start

### 1. Discover Available VMs

```bash
# List all VMs
hyperctl list

# Find specific VMs
hyperctl list -filter rhel

# Export list for processing
hyperctl list -json > available-vms.json
```

### 2. Submit Export Job (Command Line)

```bash
hyperctl submit \
  -vm "/data/vm/rhel9.4" \
  -output "/tmp/export-rhel9"
```

Output:
```
✅ Submitted 1 job(s)
✅ Accepted Jobs: 1
  💡 Job ID: 8f9e1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b
```

### 3. Submit Export Job (YAML File)

**Create job file:**
```yaml
# export-vm.yaml
name: export-rhel9
vm_path: /data/vm/rhel9.4
output_path: /tmp/export-rhel9
options:
  parallel_downloads: 4
  remove_cdrom: true
  show_individual_progress: false
```

**Submit:**
```bash
hyperctl submit -file export-vm.yaml
```

### 4. Monitor Job Progress

```bash
# Query specific job
hyperctl query -id 8f9e1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b

# Watch running jobs
watch -n 2 'hyperctl query -status running'

# Get detailed job status
curl -s http://localhost:8080/jobs/8f9e1a2b-3c4d-5e6f-7a8b-9c0d1e2f3a4b | jq
```

### 5. Verify Export

```bash
# Check output directory
ls -lh /tmp/export-rhel9/

# Expected files:
# - *.ovf (OVF descriptor)
# - *.vmdk (virtual disk files)
# - *.mf (manifest file)
```

## 📝 Job Configuration Examples

### Single VM Export (YAML)

```yaml
name: export-production-db
vm_path: /data/vm/production/db01
output_path: /migrations/db01
options:
  parallel_downloads: 8      # Use 8 parallel streams
  remove_cdrom: true          # Remove CD/DVD devices
  show_individual_progress: true  # Show progress per disk
```

### Single VM Export (JSON)

```json
{
  "name": "export-production-db",
  "vm_path": "/data/vm/production/db01",
  "output_path": "/migrations/db01",
  "options": {
    "parallel_downloads": 8,
    "remove_cdrom": true,
    "show_individual_progress": true
  }
}
```

### Batch Export (YAML)

```yaml
jobs:
  - name: export-web1
    vm_path: /data/vm/web-servers/web01
    output_path: /migrations/web01
    options:
      parallel_downloads: 4

  - name: export-web2
    vm_path: /data/vm/web-servers/web02
    output_path: /migrations/web02
    options:
      parallel_downloads: 4

  - name: export-app1
    vm_path: /data/vm/app-servers/app01
    output_path: /migrations/app01
    options:
      parallel_downloads: 8
      remove_cdrom: true
```

**Submit batch:**
```bash
hyperctl submit -file batch-export.yaml
```

### Batch Export (JSON)

```json
{
  "jobs": [
    {
      "name": "export-web1",
      "vm_path": "/data/vm/web-servers/web01",
      "output_path": "/migrations/web01",
      "options": {"parallel_downloads": 4}
    },
    {
      "name": "export-web2",
      "vm_path": "/data/vm/web-servers/web02",
      "output_path": "/migrations/web02",
      "options": {"parallel_downloads": 4}
    },
    {
      "name": "export-app1",
      "vm_path": "/data/vm/app-servers/app01",
      "output_path": "/migrations/app01",
      "options": {
        "parallel_downloads": 8,
        "remove_cdrom": true
      }
    }
  ]
}
```

## ⚙️ Export Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `parallel_downloads` | int | 4 | Number of parallel download streams per disk |
| `remove_cdrom` | bool | false | Remove CD/DVD devices before export |
| `show_individual_progress` | bool | false | Show progress for each disk separately |

### Parallel Downloads

The `parallel_downloads` option controls how many simultaneous connections are used to download each virtual disk:

- **4 streams** (default): Good for standard networks
- **8 streams**: Better for high-bandwidth networks
- **16 streams**: Maximum performance on very fast networks

**Performance Impact:**
```
1 stream:   ~50 MB/s
4 streams:  ~200 MB/s
8 streams:  ~400 MB/s
16 streams: ~800 MB/s
```

*Actual speeds depend on network, storage, and vCenter performance*

## 🔄 Complete Workflow Example

### End-to-End VM Migration

```bash
#!/bin/bash
# migrate-vm.sh - Complete VM migration workflow

VM_PATH="/data/vm/production/app01"
EXPORT_DIR="/migrations/app01"
VM_NAME="app01"

echo "🔍 Step 1: Discover and inspect VM..."
hyperctl vm -op info -path "$VM_PATH"

echo ""
echo "🔌 Step 2: Shutdown VM gracefully..."
hyperctl vm -op shutdown -path "$VM_PATH" -timeout 300

echo ""
echo "💿 Step 3: Remove CD/DVD devices..."
hyperctl vm -op remove-cdrom -path "$VM_PATH"

echo ""
echo "📦 Step 4: Export VM..."
cat > /tmp/export-${VM_NAME}.yaml <<EOF
name: export-${VM_NAME}
vm_path: ${VM_PATH}
output_path: ${EXPORT_DIR}
options:
  parallel_downloads: 8
  remove_cdrom: false  # Already removed above
  show_individual_progress: true
EOF

JOB_OUTPUT=$(hyperctl submit -file /tmp/export-${VM_NAME}.yaml)
JOB_ID=$(echo "$JOB_OUTPUT" | grep "Job ID:" | awk '{print $4}')

echo "Job ID: $JOB_ID"

echo ""
echo "📊 Step 5: Monitor export progress..."
while true; do
  STATUS=$(curl -s http://localhost:8080/jobs/$JOB_ID | jq -r '.status')
  PROGRESS=$(curl -s http://localhost:8080/jobs/$JOB_ID | jq -r '.progress.percent_complete // 0')

  echo "Status: $STATUS - Progress: ${PROGRESS}%"

  if [[ "$STATUS" == "completed" ]]; then
    echo "✅ Export completed!"
    break
  elif [[ "$STATUS" == "failed" ]]; then
    echo "❌ Export failed!"
    curl -s http://localhost:8080/jobs/$JOB_ID | jq '.error'
    exit 1
  fi

  sleep 5
done

echo ""
echo "🔍 Step 6: Verify exported files..."
ls -lh "$EXPORT_DIR"

echo ""
echo "✅ Migration export complete!"
echo "📁 Files location: $EXPORT_DIR"
echo ""
echo "Next steps:"
echo "  1. Convert to qcow2: qemu-img convert -f vmdk -O qcow2 *.vmdk disk.qcow2"
echo "  2. Import to libvirt: virt-install --import --disk disk.qcow2 ..."
```

### Run the workflow:

```bash
chmod +x migrate-vm.sh
./migrate-vm.sh
```

## 🎯 Production Best Practices

### 1. Pre-Export Checklist

```bash
# Verify VM is ready
hyperctl vm -op info -path /data/vm/my-vm

# Check VM has VMware Tools installed
# (for clean shutdown)

# Ensure sufficient disk space
df -h /migrations

# Test connectivity to vCenter
ping vcenter.example.com
```

### 2. Optimize Export Performance

```yaml
# For small VMs (<20GB)
parallel_downloads: 4

# For medium VMs (20-100GB)
parallel_downloads: 8

# For large VMs (>100GB)
parallel_downloads: 16
```

### 3. Monitor System Resources

```bash
# Watch network utilization
watch -n 1 'ifstat -i eth0'

# Monitor disk I/O
iostat -x 2

# Check daemon logs
sudo journalctl -u hypervisord -f
```

### 4. Handle Errors

```bash
# Query failed jobs
hyperctl query -status failed

# Get error details
curl -s http://localhost:8080/jobs/{job-id} | jq '.error'

# Retry failed export
hyperctl submit -vm /data/vm/my-vm -output /migrations/my-vm
```

## 📊 Real-World Examples

### Example 1: Export Development Environment (5 VMs)

```yaml
# dev-environment.yaml
jobs:
  - name: dev-web
    vm_path: /data/vm/dev/web
    output_path: /migrations/dev/web
    options:
      parallel_downloads: 4

  - name: dev-api
    vm_path: /data/vm/dev/api
    output_path: /migrations/dev/api
    options:
      parallel_downloads: 4

  - name: dev-db
    vm_path: /data/vm/dev/db
    output_path: /migrations/dev/db
    options:
      parallel_downloads: 8

  - name: dev-cache
    vm_path: /data/vm/dev/cache
    output_path: /migrations/dev/cache
    options:
      parallel_downloads: 4

  - name: dev-queue
    vm_path: /data/vm/dev/queue
    output_path: /migrations/dev/queue
    options:
      parallel_downloads: 4
```

**Submit:**
```bash
hyperctl submit -file dev-environment.yaml
```

**Monitor:**
```bash
watch -n 5 'hyperctl query -status running'
```

### Example 2: Export Single Large Database (500GB)

```yaml
# large-db-export.yaml
name: export-production-db
vm_path: /data/vm/production/oracle-db
output_path: /migrations/oracle-db
options:
  parallel_downloads: 16
  remove_cdrom: true
  show_individual_progress: true
```

**Submit:**
```bash
# Ensure VM is shutdown first
hyperctl vm -op shutdown -path /data/vm/production/oracle-db -timeout 600

# Submit export
hyperctl submit -file large-db-export.yaml
```

## 🔧 Troubleshooting

### Issue: Export Stuck at 0%

**Solution:**
```bash
# Check daemon logs
sudo journalctl -u hypervisord -n 50

# Verify vCenter connectivity
govc ls /data/vm/my-vm

# Cancel and retry
hyperctl cancel -id {job-id}
hyperctl submit -vm /data/vm/my-vm -output /migrations/my-vm
```

### Issue: Slow Download Speed

**Solution:**
```yaml
# Increase parallel downloads
options:
  parallel_downloads: 16  # Try higher value

# Or check network bottleneck
iftop -i eth0
```

### Issue: Out of Disk Space

**Solution:**
```bash
# Check available space
df -h /migrations

# Clean up old exports
rm -rf /migrations/old-exports/*

# Retry export
hyperctl submit -file export.yaml
```

## 📚 Additional Resources

- [hyperctl Features Overview](H2KVMCTL-FEATURES.md)
- [API Reference](API.md)
- [Daemon Configuration](../DAEMON-README.md)
- [Getting Started Guide](../getting-started.md)

## 💡 Tips & Tricks

1. **Use JSON output for scripting:**
   ```bash
   hyperctl list -json | jq -r '.vms[].path' | while read vm; do
     hyperctl submit -vm "$vm" -output "/migrations/$(basename $vm)"
   done
   ```

2. **Monitor all running jobs:**
   ```bash
   watch -n 2 'hyperctl query -status running'
   ```

3. **Export VMs in batches:**
   ```bash
   # Create batch file from VM list
   hyperctl list -filter production -json | \
     jq -r '.vms[] | {name: .name, vm_path: .path, output_path: ("/migrations/" + .name)}' | \
     jq -s '{jobs: .}' > batch-export.json

   # Submit batch
   hyperctl submit -file batch-export.json
   ```

4. **Automated cleanup:**
   ```bash
   # Remove completed jobs older than 24h
   hyperctl query -status completed | \
     jq -r '.jobs[] | select(.completed_at < (now - 86400)) | .id' | \
     xargs -I{} hyperctl cancel -id {}
   ```

---

**🎉 Happy Migrating!**
