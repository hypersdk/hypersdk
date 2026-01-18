# hypersdk

**Multi-cloud VM export provider layer for the hyper2kvm migration toolkit**

[![License: LGPL v3](https://img.shields.io/badge/License-LGPL%20v3-blue.svg)](https://www.gnu.org/licenses/lgpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![Production Ready](https://img.shields.io/badge/status-production%20ready-green)](https://github.com/ssahani/hypersdk)

## 🎯 Overview

`hypersdk` is a high-performance, daemon-based VM export system that provides a **provider layer abstraction** for multiple clouds. It handles VM discovery, export, download, and metadata operations across different cloud platforms.

## 📹 Video Demos

> **Want to see it in action?** Check out our demo videos showing the tools in operation!

<!-- TODO: Add demo videos here after recording
### Quick Start Demo
<video src="https://user-attachments.githubusercontent.com/.../quick-demo.mp4"></video>

### Interactive Export Demo
![hyperexport Demo](demos/hyperexport-demo.gif)

### VM Discovery & Management
[![asciicast](https://asciinema.org/a/XXXXXXX.svg)](https://asciinema.org/a/XXXXXXX)
-->

For recording your own demos, see [demos/RECORDING-GUIDE.md](demos/RECORDING-GUIDE.md)

### Supported Providers

- ✅ **vSphere** (VMware vCenter/ESXi) - Production Ready
- 🔜 **AWS** (Amazon EC2) - Coming Soon
- 🔜 **Azure** (Microsoft Azure VMs) - Coming Soon
- 🔜 **GCP** (Google Compute Engine) - Coming Soon

### Architecture

```mermaid
graph TB
    A[hyper2kvm Python<br/>Main Migration Orchestrator]
    B[hypersdk Go<br/>Provider Layer Abstraction]
    C[vSphere Provider<br/>Ready]
    D[AWS Provider<br/>Planned]
    E[Azure Provider<br/>Planned]
    F[GCP Provider<br/>Planned]

    A -->|REST API| B
    B --> C
    B --> D
    B --> E
    B --> F

    style A fill:#4CAF50,stroke:#2E7D32,color:#fff
    style B fill:#2196F3,stroke:#1565C0,color:#fff
    style C fill:#4CAF50,stroke:#2E7D32,color:#fff
    style D fill:#FFC107,stroke:#F57C00,color:#000
    style E fill:#FFC107,stroke:#F57C00,color:#000
    style F fill:#FFC107,stroke:#F57C00,color:#000
```

## ✨ Features

### Core Capabilities

- **Direct SDK Integration** - Native Go SDK clients (no external binaries)
- **Beautiful Terminal UI** - Modern pterm-based interface with animations
- **REST JSON API** - 51+ endpoints for complete automation
- **Web Dashboard** - Real-time monitoring and VM console access
- **Libvirt Integration** - Full KVM/libvirt management capabilities
- **Daemon Architecture** - Background service with systemd integration
- **Configuration Management** - YAML config files + environment variables
- **Concurrent Processing** - Goroutine-based parallel VM exports
- **Resumable Downloads** - Automatic retry with exponential backoff
- **Progress Tracking** - Real-time progress bars and ETAs
- **Production Ready** - Comprehensive error handling, logging, timeouts

### Three Tools + Dashboard

| Component | Purpose | Use Case |
|-----------|---------|----------|
| `hyperexport` | Interactive CLI | Manual exports with beautiful UI |
| `hypervisord` | Background daemon | Automation, REST API, batch processing |
| `hyperctl` | Migration Commander | Interactive TUI migration, daemon control, job management |
| **Web Dashboard** | Browser UI | VM monitoring, console access, job management |

### Web Dashboard Features

- **VM Console Access** - VNC and serial console viewers
- **Real-time Job Monitoring** - Track conversion progress
- **VM Management** - Start, stop, reboot, pause/resume VMs
- **Snapshot Management** - Create, revert, delete snapshots
- **ISO Management** - Upload ISOs and attach to VMs
- **Backup & Restore** - VM backup operations
- **Network Management** - Configure virtual networks
- **Volume Operations** - Manage storage volumes
- **Batch Operations** - Perform actions on multiple VMs

## 🚀 Quick Start

### Installation from Source

```bash
# Clone repository
git clone https://github.com/ssahani/hypersdk
cd hypersdk

# Build binaries
go build -o hyper2kvm ./cmd/hyper2kvm
go build -o hypervisord ./cmd/hypervisord
go build -o hyperctl ./cmd/hyperctl

# Install (requires root)
sudo ./install.sh
```

### Installation from RPM (Fedora/RHEL/CentOS)

```bash
# Install package
sudo dnf install hypersdk

# Configure
sudo vi /etc/hypervisord/config.yaml

# Start daemon
sudo systemctl start hypervisord
sudo systemctl enable hypervisord
```

### Configuration

#### Option 1: Environment Variables

```bash
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1
export DAEMON_ADDR='localhost:8080'
export LOG_LEVEL='info'
```

#### Option 2: Configuration File

Create `/etc/hypervisord/config.yaml`:

```yaml
VCenterURL: "https://vcenter.example.com/sdk"
Username: "administrator@vsphere.local"
Password: "your-password"
Insecure: true
DaemonAddr: "localhost:8080"
LogLevel: "info"
DownloadWorkers: 4
```

## 📖 Usage Examples

### Interactive CLI

```bash
# Launch interactive mode
./hyperexport

# Beautiful UI will guide you through:
# 1. VM discovery
# 2. Interactive selection
# 3. Real-time progress tracking
```

### Daemon Mode

```bash
# Start daemon
./hypervisord --config /etc/hypervisord/config.yaml

# Or with flags
./hypervisord --addr localhost:8080 --log-level debug
```

### Control CLI

```bash
# Submit single VM export
hyperctl submit -vm "/datacenter/vm/my-vm" -output "/tmp/export"

# Submit from YAML file
hyperctl submit -file job.yaml

# Query all jobs
hyperctl query -all

# Get daemon status
hyperctl status

# Cancel a job
hyperctl cancel -id <job-id>
```

### Job Definition (YAML)

**Single VM** (`job.yaml`):
```yaml
name: "vm-export-1"
vm_path: "/datacenter/vm/my-vm"
output_path: "/tmp/export"
options:
  parallel_downloads: 4
  remove_cdrom: true
```

**Batch** (`batch.yaml`):
```yaml
jobs:
  - name: "vm-1"
    vm_path: "/datacenter/vm/vm-1"
    output_path: "/tmp/export-1"

  - name: "vm-2"
    vm_path: "/datacenter/vm/vm-2"
    output_path: "/tmp/export-2"
```

### Web Dashboard

Access the dashboard in your browser:

```bash
# Main dashboard
http://localhost:8080/web/dashboard/

# VM console viewer
http://localhost:8080/web/dashboard/vm-console.html
```

**Dashboard Features:**
- Monitor running jobs with real-time progress
- Submit new conversion jobs
- View VMware VM inventory
- Access VM consoles (VNC/Serial)
- Manage libvirt VMs
- Take VM screenshots
- Monitor system health

### REST API

```bash
# Health check
curl http://localhost:8080/health

# Daemon status
curl http://localhost:8080/status

# List libvirt VMs
curl http://localhost:8080/libvirt/domains

# Submit job
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-export",
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/tmp/export"
  }'

# Query jobs (GET or POST)
curl http://localhost:8080/jobs/query?all=true

# Get job progress
curl http://localhost:8080/jobs/progress/<job-id>

# VM operations
curl -X POST http://localhost:8080/libvirt/domain/start \
  -d '{"name": "my-vm"}'

# Create snapshot
curl -X POST http://localhost:8080/libvirt/snapshot/create \
  -d '{"name": "my-vm", "snapshot_name": "snap1"}'

# Get console info
curl http://localhost:8080/console/info?name=my-vm
```

**51+ API Endpoints Available:**
- Core services (health, status, capabilities)
- Job management (submit, query, cancel, progress, logs, ETA)
- VMware VM operations (list, info, shutdown, poweroff)
- Libvirt domains (list, start, stop, reboot, pause, resume)
- Console access (VNC, serial, screenshots)
- Snapshots (create, revert, delete, list)
- Networks (list, create, delete, start, stop)
- Volumes (create, clone, resize, delete, upload)
- ISO management (list, upload, attach, detach)
- Backups (create, restore, verify, list, delete)
- Monitoring (stats, CPU, memory, disk, network)
- Batch operations (bulk start/stop/reboot/snapshot/delete)
- Cloning & templates (clone, deploy, export)
- Workflows (conversion, status)
- Schedules (list, create, update, delete, enable/disable)
- Webhooks (list, add, delete, test)

See [API documentation](docs/API_ENDPOINTS.md) for complete reference.

### Python Integration

```python
import requests
import time

BASE_URL = "http://localhost:8080"

# Submit job
response = requests.post(f"{BASE_URL}/jobs/submit", json={
    "name": "python-export",
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/tmp/export",
    "options": {
        "parallel_downloads": 4,
        "remove_cdrom": True
    }
})

job_id = response.json()["job_ids"][0]
print(f"Job submitted: {job_id}")

# Poll for completion
while True:
    response = requests.post(f"{BASE_URL}/jobs/query",
        json={"job_ids": [job_id]})

    job = response.json()["jobs"][0]
    status = job["status"]

    if status == "completed":
        print(f"✅ Export completed: {job['result']['ovf_path']}")
        break
    elif status == "failed":
        print(f"❌ Export failed: {job.get('error')}")
        break

    if job.get("progress"):
        progress = job["progress"]
        print(f"⏳ {progress['phase']}: {progress['percent_complete']:.1f}%")

    time.sleep(5)
```

## 🏗️ Architecture

### Directory Structure

```
hypersdk/
├── cmd/
│   ├── hyper2kvm/          # Interactive CLI
│   ├── hypervisord/        # Daemon service
│   └── hyperctl/           # Control CLI
│
├── providers/
│   ├── vsphere/            # vSphere provider (ready)
│   │   ├── client.go       # Connection management
│   │   ├── export.go       # Export logic
│   │   ├── vm_operations.go# VM management
│   │   └── types.go        # Data structures
│   ├── aws/                # AWS provider (planned)
│   ├── azure/              # Azure provider (planned)
│   └── common/             # Shared provider code
│
├── daemon/
│   ├── api/                # REST API server (51+ endpoints)
│   │   ├── server.go       # Base API server
│   │   ├── enhanced_server.go  # Enhanced server with all routes
│   │   ├── *_handlers.go   # Endpoint handlers (24 files)
│   │   ├── libvirt_handlers.go # Libvirt VM management
│   │   ├── console_handlers.go # Console access (VNC/Serial)
│   │   ├── snapshot_handlers.go # Snapshot operations
│   │   ├── iso_handlers.go  # ISO management
│   │   ├── backup_handlers.go # Backup/restore
│   │   ├── progress_handlers.go # Job progress tracking
│   │   └── ... (17 more handler files)
│   ├── jobs/               # Job management
│   ├── models/             # Data models
│   └── dashboard/          # Web dashboard
│       └── static/         # Static HTML/CSS/JS files
│
├── web/
│   └── dashboard/          # Dashboard web UI
│       ├── index.html      # Main dashboard
│       └── vm-console.html # Console viewer
│
├── config/                 # Configuration management
├── logger/                 # Structured logging
├── progress/               # Progress tracking
│
├── docs/                   # Complete documentation
│   ├── API_ENDPOINTS.md    # Full API reference
│   ├── API_REFERENCE_NEW_FEATURES.md # Phase 2 features
│   ├── GETTING-STARTED.md  # Quick start guide
│   └── PROJECT-SUMMARY.md  # Architecture overview
│
├── scripts/                # Utility scripts
│   └── test-api.sh         # API testing script
│
├── config.yaml.example     # Example configuration
├── hypervisord.service     # Systemd unit file
├── hypersdk.spec           # RPM spec file
└── install.sh              # Installation script
```

### Technology Stack

- **Language:** Go 1.24+
- **vSphere SDK:** govmomi v0.52.0
- **Libvirt:** virsh command-line integration
- **Terminal UI:** pterm v0.12.82
- **Progress Bars:** progressbar v3.19.0
- **Config:** YAML (gopkg.in/yaml.v3)
- **HTTP Server:** Go standard library
- **Dashboard:** Pure HTML/CSS/JavaScript (no frameworks)

### Concurrency Model

- **Goroutines** for job execution (not threads/forks)
- **Channels** for communication
- **Mutexes** for shared state protection
- **Worker pools** for parallel downloads
- **Context** for cancellation and timeouts

## 📊 Performance

- **Connection Time:** ~1-2 seconds
- **VM Discovery:** ~1 second (200+ VMs)
- **Download Speed:** Network-limited
- **Memory Usage:** Low (streaming downloads)
- **Concurrent Jobs:** Unlimited (goroutine-based)
- **API Response:** < 50ms

## 🔒 Security

### Disable Web Dashboard (API-Only Mode)

For security-conscious deployments, embedded systems, or pure automation use cases, you can disable the web dashboard:

```bash
# Option 1: CLI flag
./hypervisord --disable-web

# Option 2: Config file
# config.yaml
web:
  disabled: true
```

**Benefits:**
- Reduced attack surface (no web UI)
- Lower resource usage
- Perfect for headless/embedded systems
- API-only automation environments

When disabled, the daemon runs in API-only mode with all 51+ REST endpoints still available via CLI or scripts.

### Systemd Hardening

The systemd service includes security hardening:
- `NoNewPrivileges=true`
- `PrivateTmp=true`
- `ProtectSystem=strict`
- `ProtectHome=true`
- `ProtectKernelTunables=true`

### Credentials

- Store credentials in config file: `/etc/hypervisord/config.yaml`
- Protect config file: `chmod 600 /etc/hypervisord/config.yaml`
- Or use environment variables for temporary use
- Never commit credentials to git

## 🛠️ Development

### Build

```bash
# Build all binaries
go build -o hyper2kvm ./cmd/hyper2kvm
go build -o hypervisord ./cmd/hypervisord
go build -o hyperctl ./cmd/hyperctl
```

### Test

```bash
# Run tests
go test ./...

# Test with coverage
go test -cover ./...
```

### Format

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...
```

### Dependencies

```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy
```

## 📚 Documentation

### Getting Started
- [Getting Started Guide](docs/GETTING-STARTED.md) - Quick start tutorial with dashboard access
- [Documentation Index](docs/00-INDEX.md) - Complete documentation catalog
- [Project Summary](docs/PROJECT-SUMMARY.md) - Architecture and design overview

### API Reference
- [API Endpoints](docs/API_ENDPOINTS.md) - Complete API reference (51+ endpoints)
- [API Reference - New Features](docs/API_REFERENCE_NEW_FEATURES.md) - Phase 2 features documentation
- [General API Documentation](docs/api.md) - API usage guide

### Testing & Implementation
- [Dashboard Testing Report](DASHBOARD_TESTING_REPORT.md) - Comprehensive endpoint testing results
- [Dashboard Implementation](DASHBOARD_IMPLEMENTATION_COMPLETE.md) - Implementation details and features
- [Test Results](docs/TEST-RESULTS.md) - Test coverage and results
- [Test API Script](scripts/test-api.sh) - Automated API testing

### Features & Guides
- [Interactive Mode Guide](docs/user-guides/01-interactive-mode.md) - Interactive TUI usage
- [VM Export Guide](docs/user-guides/02-vm-export-guide.md) - Step-by-step export procedures
- [Integration Guide](docs/user-guides/03-integration.md) - Integrating HyperSDK into workflows

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

LGPL-3.0-or-later - see [LICENSE](LICENSE) file for details

## 👤 Author

**Susant Sahani** <ssahani@gmail.com>

## 🔗 Related Projects

- [hyper2kvm](https://github.com/ssahani/hyper2kvm) - Python migration orchestrator
- [govmomi](https://github.com/vmware/govmomi) - VMware vSphere Go SDK
- [pterm](https://github.com/pterm/pterm) - Modern terminal UI library

## 🎉 Acknowledgments

Built with ❤️ using:
- [govmomi](https://github.com/vmware/govmomi) - VMware SDK
- [pterm](https://github.com/pterm/pterm) - Terminal UI
- [progressbar](https://github.com/schollz/progressbar) - Progress bars
- Go standard library

---

**Made by Susant Sahani**
*Part of the hyper2kvm project family*
