# hypervisord & hyperctl - Daemon-Based VM Export System

A high-performance daemon-based architecture for exporting VMs from vSphere to KVM format.

## Architecture

### Components

1. **hypervisord** - Background daemon service that:
   - Accepts job submissions via REST API (JSON/YAML)
   - Manages concurrent VM export tasks using goroutines
   - Tracks progress for each job in real-time
   - Provides JSON API for status queries
   - Listens on HTTP (default: localhost:8080)

2. **hyperctl** - Command-line control tool that:
   - Submits jobs to the daemon
   - Queries job status and progress
   - Lists all running/completed jobs
   - Cancels running jobs
   - Beautiful terminal UI with pterm

3. **hyper2kvm** - Interactive CLI tool (original)
   - User-friendly interactive VM export
   - Progress bars and visual feedback
   - Single VM at a time

## Quick Start

### 1. Start the Daemon

```bash
# Set vCenter credentials (used as defaults)
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1

# Start daemon
./build/hypervisord

# Or specify custom address
./build/hypervisord -addr localhost:9090
```

The daemon will display available API endpoints and wait for jobs.

### 2. Submit Jobs

#### Single VM (Command Line)

```bash
./build/hyperctl submit -vm "/data/vm/test-vm" -output "/tmp/export-test"
```

#### From YAML File

```bash
./build/hyperctl submit -file example-job.yaml
```

#### Batch Jobs (Multiple VMs)

```bash
./build/hyperctl submit -file example-batch.yaml
```

### 3. Query Status

#### All Jobs

```bash
./build/hyperctl query -all
```

#### Specific Job

```bash
./build/hyperctl query -id abc123
```

#### Filter by Status

```bash
./build/hyperctl query -status running
./build/hyperctl query -status completed,failed
```

### 4. Daemon Status

```bash
./build/hyperctl status
```

### 5. Cancel Jobs

```bash
./build/hyperctl cancel -id abc123
./build/hyperctl cancel -id abc123,def456,ghi789
```

## Job File Format

### YAML (Single Job)

```yaml
name: "my-vm-export"
vm_path: "/datacenter/vm/my-vm"
output_path: "/tmp/export-my-vm"
options:
  parallel_downloads: 4
  remove_cdrom: true
  show_individual_progress: false

# Optional: Override vCenter credentials for this job
vcenter_url: "https://other-vcenter.com/sdk"
username: "admin@vsphere.local"
password: "password"
insecure: true
```

### YAML (Batch)

```yaml
jobs:
  - name: "vm-1"
    vm_path: "/datacenter/vm/vm-1"
    output_path: "/tmp/export-vm-1"
    options:
      parallel_downloads: 4

  - name: "vm-2"
    vm_path: "/datacenter/vm/vm-2"
    output_path: "/tmp/export-vm-2"
    options:
      parallel_downloads: 8
      remove_cdrom: true
```

### JSON (Single Job)

```json
{
  "name": "my-vm-export",
  "vm_path": "/datacenter/vm/my-vm",
  "output_path": "/tmp/export-my-vm",
  "options": {
    "parallel_downloads": 4,
    "remove_cdrom": true,
    "show_individual_progress": false
  }
}
```

### JSON (Batch)

```json
{
  "jobs": [
    {
      "name": "vm-1",
      "vm_path": "/datacenter/vm/vm-1",
      "output_path": "/tmp/export-vm-1"
    },
    {
      "name": "vm-2",
      "vm_path": "/datacenter/vm/vm-2",
      "output_path": "/tmp/export-vm-2"
    }
  ]
}
```

## REST API Endpoints

The daemon exposes a REST API on `http://localhost:8080` (default):

### Health Check

```bash
curl http://localhost:8080/health
```

### Get Status

```bash
curl http://localhost:8080/status
```

Response:
```json
{
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "total_jobs": 42,
  "running_jobs": 3,
  "completed_jobs": 38,
  "failed_jobs": 1,
  "timestamp": "2024-01-16T10:30:00Z"
}
```

### Submit Job

```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d @example-job.json
```

Or with YAML:
```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/x-yaml" \
  -d @example-job.yaml
```

Response:
```json
{
  "job_ids": ["abc123", "def456"],
  "accepted": 2,
  "rejected": 0,
  "timestamp": "2024-01-16T10:30:00Z"
}
```

### Query Jobs

```bash
curl -X POST http://localhost:8080/jobs/query \
  -H "Content-Type: application/json" \
  -d '{"all": true}'
```

Filter by status:
```bash
curl -X POST http://localhost:8080/jobs/query \
  -H "Content-Type: application/json" \
  -d '{"status": ["running", "pending"]}'
```

Get specific jobs:
```bash
curl -X POST http://localhost:8080/jobs/query \
  -H "Content-Type: application/json" \
  -d '{"job_ids": ["abc123", "def456"]}'
```

### Get Specific Job

```bash
curl http://localhost:8080/jobs/abc123
```

Response:
```json
{
  "definition": {
    "id": "abc123",
    "name": "my-vm-export",
    "vm_path": "/data/vm/test-vm",
    "output_path": "/tmp/export-test",
    "created_at": "2024-01-16T10:00:00Z"
  },
  "status": "running",
  "progress": {
    "phase": "exporting",
    "current_file": "disk-0001.vmdk",
    "files_downloaded": 2,
    "total_files": 5,
    "bytes_downloaded": 1073741824,
    "total_bytes": 5368709120,
    "percent_complete": 45.5
  },
  "started_at": "2024-01-16T10:00:05Z",
  "updated_at": "2024-01-16T10:15:30Z"
}
```

### Cancel Job

```bash
curl -X POST http://localhost:8080/jobs/cancel \
  -H "Content-Type: application/json" \
  -d '{"job_ids": ["abc123", "def456"]}'
```

Response:
```json
{
  "cancelled": ["abc123"],
  "failed": ["def456"],
  "errors": {
    "def456": "job not found"
  },
  "timestamp": "2024-01-16T10:30:00Z"
}
```

## Integration with Python hyper2kvm

This Go daemon can be used alongside the Python hyper2kvm project:

```python
import requests
import yaml

# Submit job from Python
job = {
    "name": "python-export",
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/tmp/export"
}

resp = requests.post("http://localhost:8080/jobs/submit", json=job)
job_id = resp.json()["job_ids"][0]

# Query status
status = requests.post("http://localhost:8080/jobs/query",
                       json={"job_ids": [job_id]})
print(status.json())
```

## Architecture Benefits

### Concurrent Processing
- Multiple VMs can be exported simultaneously
- Each job runs in its own goroutine
- Efficient use of system resources

### Job Persistence
- All jobs tracked in memory (could be extended to database)
- Query status of any job at any time
- Historical job data available

### Flexible Job Submission
- Submit single jobs or batches
- YAML or JSON format
- File-based or API-based submission
- Override credentials per job

### Real-Time Progress
- Track export progress in real-time
- Phase information (connecting, discovering, exporting)
- File-level and byte-level progress
- Estimated time remaining (future enhancement)

### Clean API
- RESTful JSON API
- Consistent response format
- Error handling and validation
- Extensible for future features

## Building from Source

```bash
# Build all binaries
cd ~/go/hyper2kvm
go build -o build/hypervisord ./cmd/hypervisord
go build -o build/hyperctl ./cmd/hyperctl
go build -o build/hyper2kvm ./cmd/hyper2kvm

# Run tests
go test ./...

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o build/hypervisord-linux ./cmd/hypervisord
GOOS=linux GOARCH=amd64 go build -o build/hyperctl-linux ./cmd/hyperctl
```

## Systemd Service

Create `/etc/systemd/system/hypervisord.service`:

```ini
[Unit]
Description=Hyper2KVM Export Daemon
After=network.target

[Service]
Type=simple
User=vmexport
Environment="GOVC_URL=https://vcenter.example.com/sdk"
Environment="GOVC_USERNAME=administrator@vsphere.local"
Environment="GOVC_PASSWORD=your-password"
Environment="GOVC_INSECURE=1"
ExecStart=/usr/local/bin/hypervisord -addr localhost:8080
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable hypervisord
sudo systemctl start hypervisord
sudo systemctl status hypervisord
```

## Future Enhancements

- [ ] Job persistence to database (SQLite/PostgreSQL)
- [ ] Webhook notifications on job completion
- [ ] Job scheduling (cron-like)
- [ ] Resource limits (max concurrent jobs, bandwidth throttling)
- [ ] Metrics and monitoring (Prometheus)
- [ ] Web UI dashboard
- [ ] Job templates
- [ ] Retry failed jobs automatically
- [ ] Email notifications
- [ ] Multi-tenant support

## License

Same as hyper2kvm project

## Author

Susant Sahani (alongside Python hyper2kvm)
