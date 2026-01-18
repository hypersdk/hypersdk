# Getting Started with hypersdk

Welcome to **hypersdk** - the Go implementation of the hyper2kvm migration toolkit!

---

## 🎯 What is hypersdk?

A high-performance, daemon-based VM export system that provides:

1. **Interactive CLI** (`hyperexport`) - Beautiful terminal UI for manual exports
2. **Background Daemon** (`hypervisord`) - REST API service for automation
3. **Control CLI** (`hyperctl`) - Manage daemon jobs from command line

Built with:
- **govmomi v0.52.0** - VMware vSphere Go SDK
- **pterm v0.12.82** - Modern terminal UI
- **Go 1.24** - Fast, concurrent, production-ready

---

## ⚡ Quick Start (5 minutes)

### 1. Set Your vCenter Credentials

```bash
export GOVC_URL='https://your-vcenter.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1
```

### 2. Choose Your Workflow

#### Option A: Web Dashboard (Easiest!)

```bash
# Start daemon
cd ~/projects/hypersdk
./build/hypervisord

# Open in browser
xdg-open http://localhost:8080/web/dashboard/
```

**What You Can Do:**
- Submit conversion jobs from web UI
- Monitor job progress in real-time
- Discover VMware VMs
- Access VM consoles (VNC/Serial)
- Manage libvirt VMs
- Take VM screenshots

**Dashboard URLs:**
- Main Dashboard: `http://localhost:8080/web/dashboard/`
- Console Viewer: `http://localhost:8080/web/dashboard/vm-console.html`

#### Option B: Interactive Mode (For CLI Users)

```bash
cd ~/projects/hypersdk
./build/hyperexport
```

Then:
- Browse and select your VM interactively
- Watch real-time progress bars
- Get beautiful status updates

#### Option C: Daemon Mode (For Automation)

**Terminal 1 - Start Daemon:**
```bash
./build/hypervisord
```

**Terminal 2 - Submit Jobs:**
```bash
# Single VM
./build/hyperctl submit -vm "/datacenter/vm/my-vm" -output "/tmp/export"

# Or from YAML
./build/hyperctl submit -file example-job.yaml

# Check status
./build/hyperctl query -all
```

---

## 📝 Your First Export

### Create a Job File

Create `my-export.yaml`:

```yaml
name: "my-first-export"
vm_path: "/datacenter/vm/my-test-vm"  # Change this to your VM path
output_path: "/tmp/my-export"
options:
  parallel_downloads: 4
  remove_cdrom: true
```

### Submit the Job

```bash
./build/hyperctl submit -file my-export.yaml
```

### Monitor Progress

```bash
# Watch all jobs
./build/hyperctl query -all

# Watch daemon status
./build/hyperctl status

# View logs
tail -f /tmp/hypervisord.log
```

---

## 🔧 Common Tasks

### List All VMs

The interactive CLI (`./build/hyperexport`) automatically discovers all VMs.

### Export Multiple VMs (Batch)

Create `batch-export.yaml`:

```yaml
jobs:
  - name: "vm-1"
    vm_path: "/datacenter/vm/vm-1"
    output_path: "/tmp/export-vm-1"

  - name: "vm-2"
    vm_path: "/datacenter/vm/vm-2"
    output_path: "/tmp/export-vm-2"

  - name: "vm-3"
    vm_path: "/datacenter/vm/vm-3"
    output_path: "/tmp/export-vm-3"
```

Submit:
```bash
./build/hyperctl submit -file batch-export.yaml
```

### Cancel a Running Job

```bash
# Get job ID first
./build/hyperctl query -all

# Cancel it
./build/hyperctl cancel -id <job-id>
```

### Check What the Daemon is Doing

```bash
./build/hyperctl status
```

Output:
```
┌────────────┬────────┐
| Metric     | Value  |
|────────────────────|
| Version    | 1.0.0  |
| Uptime     | 5m30s  |
| Total Jobs | 5      |
| Running    | 2      |
| Completed  | 3      |
| Failed     | 0      |
└────────────┴────────┘
```

---

## 🐍 Python Integration

Perfect for integrating with your Python hyper2kvm project:

```python
import requests

# Submit job
resp = requests.post("http://localhost:8080/jobs/submit", json={
    "name": "python-export",
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/tmp/export"
})

job_id = resp.json()["job_ids"][0]

# Check status
status = requests.post("http://localhost:8080/jobs/query",
    json={"job_ids": [job_id]}).json()

print(status["jobs"][0]["status"])  # "running", "completed", etc.
```

---

## 🎨 What Makes This Special?

### Beautiful Terminal UI
```
 ██   ██ ██    ██ ██████  ███████ ██████
 ██   ██  ██  ██  ██   ██ ██      ██   ██
 ███████   ████   ██████  █████   ██████
 ██   ██    ██    ██      ██      ██   ██
 ██   ██    ██    ██      ███████ ██   ██

        Hypervisor to KVM Migration Tool
```

### Real-Time Progress
```
Exporting my-vm: 45% |████████████░░░░░░░| (2/4 files)
 ✓ disk-0.vmdk (23 GB) - Complete
 ✓ disk-1.vmdk (15 GB) - Complete
 → disk-2.vmdk (18 GB) - Downloading 67%
   disk-3.vmdk (8 GB)  - Pending
```

### Concurrent Processing
Multiple VMs export **in parallel** - each in its own goroutine!

---

## 📊 Example Output

### Interactive CLI
```
 SUCCESS  Connected to vSphere successfully!
 SUCCESS  Found 201 virtual machine(s)

┌─ Connection Info ─────────────────────┐
| vCenter: https://10.73.213.134/sdk    |
| User: administrator@vsphere.local     |
└───────────────────────────────────────┘

Select a VM to export [type to search]:
> my-test-vm
  production-web-server
  dev-database
  ...
```

### Daemon Query
```
# Jobs

┌────────────┬─────────┬────────────────┬─────────┬────────────┬─────────┐
| Job ID     | Name    | VM Path        | Status  | Progress   | Started |
|──────────────────────────────────────────────────────────────────────────|
| abc123...  | vm-1    | .../my-vm      | running | export 45% | 10:30:15|
| def456...  | vm-2    | .../other-vm   | complete| 100%       | 10:25:00|
└────────────┴─────────┴────────────────┴─────────┴────────────┴─────────┘
```

---

## 🚀 Production Deployment

### Run as Systemd Service

1. Copy binary:
   ```bash
   sudo cp build/hypervisord /usr/local/bin/
   ```

2. Create service file `/etc/systemd/system/hypervisord.service`:
   ```ini
   [Unit]
   Description=Hypervisord - VM Export Daemon
   After=network.target

   [Service]
   Type=simple
   User=vmexport
   Environment="GOVC_URL=https://vcenter.example.com/sdk"
   Environment="GOVC_USERNAME=admin@vsphere.local"
   Environment="GOVC_PASSWORD=secret"
   Environment="GOVC_INSECURE=1"
   ExecStart=/usr/local/bin/hypervisord
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target
   ```

3. Enable and start:
   ```bash
   sudo systemctl enable hypervisord
   sudo systemctl start hypervisord
   sudo systemctl status hypervisord
   ```

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GOVC_URL` | ✅ Yes | vCenter SDK URL |
| `GOVC_USERNAME` | ✅ Yes | vCenter username |
| `GOVC_PASSWORD` | ✅ Yes | vCenter password |
| `GOVC_INSECURE` | No | Skip TLS verification (1=yes) |
| `GOVC_DATACENTER` | No | Default datacenter |
| `DOWNLOAD_WORKERS` | No | Parallel downloads (default: 4) |
| `LOG_LEVEL` | No | debug/info/warn/error (default: info) |

---

## 🔍 Troubleshooting

### Problem: Cannot Connect to vCenter

**Check:**
```bash
# Test credentials
echo $GOVC_URL
echo $GOVC_USERNAME

# Try direct connection
curl -k $GOVC_URL
```

**Solution:**
- Verify URL ends with `/sdk`
- Check firewall allows HTTPS (443)
- Try `GOVC_INSECURE=1` for self-signed certs

### Problem: Job Stuck in "Running"

**Check daemon logs:**
```bash
tail -f /tmp/hypervisord.log
```

**Common causes:**
- Network interruption
- VM has snapshots (not supported)
- Insufficient disk space
- vCenter under heavy load

### Problem: Download Fails

**Automatic retry** happens 3 times. If still failing:
- Check disk space: `df -h /tmp`
- Check network: `ping vcenter.example.com`
- Reduce workers: Add `parallel_downloads: 1` to job file

---

## 🌐 Using the Web Dashboard

### Access the Dashboard

After starting hypervisord, open your browser to:
```
http://localhost:8080/web/dashboard/
```

### Main Dashboard Features

**Submit Jobs:**
1. Click "Submit Job" card
2. Enter VM path: `/datacenter/vm/my-vm`
3. Enter output directory: `/tmp/export`
4. Click "Submit"

**Monitor Jobs:**
- View active jobs with progress
- See completed jobs
- Track failed jobs with error details
- Auto-refreshes every 5 seconds

**Discover VMs:**
- Click "Refresh VMs" to scan vCenter
- View all VMware VMs with details
- See power state, CPU, memory, OS

### Console Viewer

Access at: `http://localhost:8080/web/dashboard/vm-console.html`

**Features:**
- Grid view of all libvirt VMs
- Status indicators (running=green, stopped=red, paused=yellow)
- **Open VNC** - Browser-based console
- **Open Serial** - Serial console access
- **Screenshot** - Capture VM screenshot
- **Console Info** - View connection details

**Using VNC Console:**
1. Find your VM in the grid
2. Click "Open VNC" button
3. New window opens with VNC viewer
4. Note: VM must be running with VNC graphics configured

### Dashboard API Calls

The dashboard uses these key endpoints:
```bash
# Jobs
GET  /jobs/query?all=true         # List all jobs
POST /jobs/submit                 # Submit new job
GET  /jobs/progress/<id>          # Get job progress

# VMs
GET  /libvirt/domains             # List libvirt VMs
GET  /vms/list                    # List VMware VMs

# Console
GET  /console/info?name=<vm>      # Get console info
GET  /console/vnc?name=<vm>       # Open VNC console
GET  /console/serial?name=<vm>    # Open serial console
GET  /console/screenshot?name=<vm># Take screenshot

# Health
GET  /health                      # Health check
```

## 📚 Learn More

### Core Documentation
- [Main README](../README.md) - Complete project overview
- [Project Summary](PROJECT-SUMMARY.md) - Architecture and features
- [Test Results](TEST-RESULTS.md) - Test coverage

### Dashboard Documentation
- [Dashboard README](../daemon/dashboard/README.md) - Dashboard usage and customization
- [Dashboard Implementation](../DASHBOARD_IMPLEMENTATION_COMPLETE.md) - Implementation details
- [Dashboard Testing](../DASHBOARD_TESTING_REPORT.md) - Test results

### API Documentation
- [API Endpoints](API_ENDPOINTS.md) - Complete API reference (51+ endpoints)
- [API New Features](API_REFERENCE_NEW_FEATURES.md) - Phase 2 features
- [General API Guide](api.md) - API usage patterns

---

## 💡 Pro Tips

1. **Start with interactive mode** to understand the workflow
2. **Use YAML files** for repeatable exports
3. **Run daemon in background** for automation
4. **Monitor with hyperctl** instead of checking logs
5. **Adjust parallel downloads** based on network speed
6. **Remove CD/DVD** before export for cleaner OVF files
7. **Disable web dashboard** for API-only/headless deployments:
   ```bash
   ./hypervisord --disable-web
   # Or in config: web.disabled=true
   ```

---

## ✅ You're Ready!

Try your first export:

```bash
# Interactive mode
./build/hyperexport

# OR daemon mode
./build/hypervisord &
./build/hyperctl submit -vm "/path/to/your/vm" -output "/tmp/export"
./build/hyperctl query -all
```

**Happy migrating! 🎉**

---

*Part of the [hyper2kvm](https://github.com/yourusername/hyper2kvm) project family*
