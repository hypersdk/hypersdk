# HyperSDK Quick Start Guide

Get up and running with HyperSDK in under 5 minutes!

## 📋 Table of Contents

- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [SDK Quick Start](#sdk-quick-start)
- [Common Workflows](#common-workflows)
- [Next Steps](#next-steps)

---

## 🚀 Installation

### Option 1: Docker (Fastest)

```bash
# Pull and run the daemon
docker run -d \
  --name hypervisord \
  -p 8080:8080 \
  -v /exports:/exports \
  -e GOVC_URL='https://vcenter.example.com/sdk' \
  -e GOVC_USERNAME='admin@vsphere.local' \
  -e GOVC_PASSWORD='your-password' \
  -e GOVC_INSECURE=1 \
  hypersdk/hypervisord:latest

# Verify it's running
curl http://localhost:8080/health
```

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Build the daemon
go build -o hypervisord ./cmd/hypervisord

# Set environment variables
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='admin@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1

# Run the daemon
./hypervisord
```

### Option 3: Kubernetes/Helm

```bash
# Add Helm repository (if available)
helm repo add hypersdk https://charts.hypersdk.io
helm repo update

# Install with values
helm install hypersdk hypersdk/hypersdk \
  --set vcenter.url='https://vcenter.example.com/sdk' \
  --set vcenter.username='admin@vsphere.local' \
  --set vcenter.password='your-password'
```

---

## 🎯 Basic Usage

### 1. Check System Status

```bash
# Health check
curl http://localhost:8080/health

# Get daemon status
curl http://localhost:8080/status

# Check capabilities
curl http://localhost:8080/capabilities
```

### 2. Submit Your First Export Job

```bash
# Export a VM to OVA format
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "/datacenter/vm/my-test-vm",
    "output_path": "/exports",
    "format": "ova",
    "compression": true
  }'

# Response: {"job_ids": ["job-12345"], "accepted": 1, "rejected": 0}
```

### 3. Monitor Job Progress

```bash
# Get job status
curl http://localhost:8080/jobs/job-12345

# Get detailed progress
curl http://localhost:8080/jobs/progress/job-12345

# Get job logs
curl http://localhost:8080/jobs/logs/job-12345
```

### 4. Query All Jobs

```bash
# List all jobs
curl http://localhost:8080/jobs/query?all=true

# Filter by status
curl -X POST http://localhost:8080/jobs/query \
  -H "Content-Type: application/json" \
  -d '{"status": "running"}'
```

---

## 📚 SDK Quick Start

### Python SDK

#### Installation

```bash
pip install hypersdk
# or from source
cd sdk/python
pip install -e .
```

#### Basic Usage

```python
from hypersdk import HyperSDK

# Initialize client
client = HyperSDK("http://localhost:8080")

# Submit a job
job_id = client.submit_job({
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/exports",
    "format": "ova"
})

print(f"Job submitted: {job_id}")

# Monitor progress
import time
while True:
    job = client.get_job(job_id)
    status = job['status']

    if status == 'completed':
        print(f"✅ Export completed!")
        break
    elif status == 'failed':
        print(f"❌ Export failed: {job.get('error')}")
        break

    if job.get('progress'):
        pct = job['progress']['percent_complete']
        print(f"⏳ Progress: {pct:.1f}%")

    time.sleep(5)
```

### TypeScript SDK

#### Installation

```bash
npm install hypersdk
# or
yarn add hypersdk
```

#### Basic Usage

```typescript
import { HyperSDK } from 'hypersdk';

// Initialize client
const client = new HyperSDK('http://localhost:8080');

// Submit a job
const jobId = await client.submitJob({
  vm_path: '/datacenter/vm/my-vm',
  output_path: '/exports',
  format: 'ova'
});

console.log(`Job submitted: ${jobId}`);

// Monitor progress
while (true) {
  const job = await client.getJob(jobId);

  if (job.status === 'completed') {
    console.log('✅ Export completed!');
    break;
  } else if (job.status === 'failed') {
    console.log(`❌ Export failed: ${job.error}`);
    break;
  }

  if (job.progress) {
    console.log(`⏳ Progress: ${job.progress.percent_complete.toFixed(1)}%`);
  }

  await new Promise(resolve => setTimeout(resolve, 5000));
}
```

---

## 💡 Common Workflows

### Workflow 1: Simple VM Export

**Goal**: Export a single VM to OVA format

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# Submit export job
job_id = client.submit_job({
    "vm_path": "/datacenter/vm/web-server-01",
    "output_path": "/exports/backups",
    "format": "ova",
    "compression": true,
    "verify": true
})

# Wait for completion
job = client.wait_for_completion(job_id)
print(f"Export saved to: {job['result']['ovf_path']}")
```

### Workflow 2: Incremental Backup with CBT

**Goal**: Set up fast incremental backups using Changed Block Tracking

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# 1. Enable CBT on the VM
vm_path = "/datacenter/vm/production-db"
client.enable_cbt(vm_path)

# 2. Analyze potential savings
analysis = client.analyze_incremental_export(vm_path)
print(f"Changed data: {analysis['estimated_savings_bytes'] / 1e9:.2f} GB")

# 3. Submit incremental export job
job_id = client.submit_job({
    "vm_path": vm_path,
    "output_path": "/exports/incremental",
    "format": "qcow2",
    "incremental": True
})

print(f"Incremental backup started: {job_id}")
```

### Workflow 3: Cost-Optimized Cloud Backup

**Goal**: Find cheapest cloud storage and upload VM export

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# 1. Estimate export size
size_estimate = client.estimate_export_size(
    disk_size_gb=500,
    format="ova",
    include_snapshots=False
)
export_size = size_estimate['estimated_export_gb']

# 2. Compare cloud providers
comparison = client.compare_providers(
    storage_gb=export_size,
    transfer_gb=0,  # Keep in cloud, no download
    requests=100,
    duration_days=365
)

print(f"Cheapest provider: {comparison['cheapest']}")
print(f"Annual cost: ${comparison['estimates'][0]['total_cost']:.2f}")

# 3. Export and upload to cheapest provider
job_id = client.submit_job({
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/exports",
    "format": "ova",
    "upload_to_cloud": True,
    "cloud_provider": comparison['cheapest'],
    "cloud_config": {
        "bucket": "my-backups",
        "region": "us-east-1"
    }
})
```

### Workflow 4: Automated Scheduled Backups

**Goal**: Set up weekly backups with retry and time windows

```typescript
import { HyperSDK } from 'hypersdk';

const client = new HyperSDK('http://localhost:8080');

// Create advanced schedule
const schedule = await client.createAdvancedSchedule({
  name: 'weekly-production-backup',
  schedule: '0 2 * * 0',  // 2 AM every Sunday
  jobTemplate: {
    vm_path: '/datacenter/vm/production-app',
    output_path: '/exports/weekly',
    format: 'ova',
    compression: true,
    incremental: true
  },
  advancedConfig: {
    retry_policy: {
      max_attempts: 3,
      initial_delay: 300,
      max_delay: 3600,
      backoff_strategy: 'exponential'
    },
    time_windows: [{
      start_time: '02:00',
      end_time: '06:00',
      days: ['sunday'],
      timezone: 'America/New_York'
    }],
    priority: 90,
    skip_if_running: true,
    notify_on_failure: true
  }
});

console.log(`Schedule created: ${schedule.name}`);
```

### Workflow 5: Multi-Platform Migration

**Goal**: Export VM and convert to multiple formats for different platforms

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# 1. Export from vSphere
export_job = client.submit_job({
    "vm_path": "/datacenter/vm/app-server",
    "output_path": "/exports/source",
    "format": "vmdk"
})

# Wait for export
client.wait_for_completion(export_job)

# 2. Convert to multiple formats in parallel
formats = {
    "qcow2": "For KVM/QEMU",
    "vhdx": "For Hyper-V Gen 2",
    "vdi": "For VirtualBox"
}

conversion_jobs = []
for target_format, description in formats.items():
    job_id = client.convert_format({
        "source_path": "/exports/source/app-server.vmdk",
        "target_format": target_format,
        "output_path": f"/exports/{target_format}/app-server.{target_format}"
    })
    conversion_jobs.append((target_format, job_id, description))
    print(f"Started conversion to {target_format}: {description}")

# 3. Wait for all conversions
for fmt, job_id, desc in conversion_jobs:
    client.wait_for_completion(job_id)
    print(f"✅ {desc} ready!")
```

### Workflow 6: Disaster Recovery Setup

**Goal**: Create automated DR backups with verification

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# Critical VMs for DR
critical_vms = [
    "/datacenter/vm/database-primary",
    "/datacenter/vm/app-server-01",
    "/datacenter/vm/load-balancer"
]

# Enable CBT for all critical VMs
for vm_path in critical_vms:
    try:
        client.enable_cbt(vm_path)
        print(f"✅ CBT enabled: {vm_path}")
    except Exception as e:
        print(f"⚠️  CBT already enabled or not supported: {vm_path}")

# Create DR schedule for each VM
for vm_path in critical_vms:
    vm_name = vm_path.split('/')[-1]

    schedule = client.create_advanced_schedule({
        "name": f"dr-backup-{vm_name}",
        "schedule": "0 */6 * * *",  # Every 6 hours
        "job_template": {
            "vm_path": vm_path,
            "output_path": f"/dr-backups/{vm_name}",
            "format": "ova",
            "incremental": True,
            "verify": True
        },
        "advanced_config": {
            "retry_policy": {
                "max_attempts": 5,
                "backoff_strategy": "exponential"
            },
            "priority": 100,  # Highest priority
            "skip_if_running": True,
            "notify_on_failure": True
        }
    })

    print(f"✅ DR schedule created for {vm_name}")

print("\n🎯 Disaster Recovery setup complete!")
print("Backups will run every 6 hours with automatic retry")
```

---

## 🎓 Advanced Examples

### Example 1: Dynamic Provider Management

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# Load AWS provider plugin dynamically
client.load_plugin({
    "name": "aws",
    "path": "/opt/hypersdk/plugins/aws.so",
    "config": {
        "region": "us-east-1"
    }
})

# Check plugin health
status = client.get_plugin_status("aws")
print(f"AWS provider loaded: {status['healthy']}")

# Use the provider
job_id = client.submit_job({
    "vm_path": "/datacenter/vm/my-vm",
    "provider": "aws",
    "target_region": "us-east-1"
})

# Later: Update provider without downtime
client.reload_plugin("aws")
```

### Example 2: Cost Analysis Before Migration

```typescript
import { HyperSDK } from 'hypersdk';

const client = new HyperSDK('http://localhost:8080');

async function planMigration(vmPath: string, diskSizeGB: number) {
  // 1. Estimate export size
  const sizeEst = await client.estimateExportSize({
    diskSizeGB: diskSizeGB,
    format: 'ova',
    includeSnapshots: false
  });

  console.log(`Estimated export size: ${sizeEst.estimated_export_gb} GB`);

  // 2. Compare 1-year storage costs
  const comparison = await client.compareProviders({
    storageGB: sizeEst.estimated_export_gb,
    transferGB: 0,
    requests: 1000,
    durationDays: 365
  });

  console.log(`\nCloud Storage Cost Comparison (1 year):`);
  comparison.estimates.forEach(est => {
    console.log(`  ${est.provider}: $${est.total_cost.toFixed(2)}`);
  });

  console.log(`\nRecommendation: ${comparison.recommended}`);
  console.log(`Savings: $${comparison.savings_vs_expensive.toFixed(2)}`);

  // 3. Get detailed breakdown for recommended provider
  const cheapest = comparison.estimates.find(
    e => e.provider === comparison.cheapest
  );

  if (cheapest) {
    console.log(`\nDetailed Breakdown for ${cheapest.provider}:`);
    console.log(`  Storage:  $${cheapest.breakdown.storage_cost.toFixed(2)}`);
    console.log(`  Transfer: $${cheapest.breakdown.transfer_cost.toFixed(2)}`);
    console.log(`  Requests: $${cheapest.breakdown.request_cost.toFixed(2)}`);
  }

  return comparison.cheapest;
}

// Usage
const bestProvider = await planMigration('/datacenter/vm/my-vm', 500);
console.log(`\nProceed with ${bestProvider} for optimal costs`);
```

### Example 3: Batch Export with Progress Tracking

```python
from hypersdk import HyperSDK
import time
from typing import List, Dict

def batch_export_with_tracking(client: HyperSDK, vm_paths: List[str]) -> Dict:
    """Export multiple VMs and track overall progress"""

    results = {
        "total": len(vm_paths),
        "completed": 0,
        "failed": 0,
        "jobs": {}
    }

    # Submit all jobs
    print(f"Submitting {len(vm_paths)} export jobs...")
    for vm_path in vm_paths:
        try:
            job_id = client.submit_job({
                "vm_path": vm_path,
                "output_path": "/exports/batch",
                "format": "ova",
                "compression": True
            })
            results["jobs"][job_id] = {
                "vm_path": vm_path,
                "status": "submitted"
            }
            print(f"  ✓ {vm_path}: {job_id}")
        except Exception as e:
            print(f"  ✗ {vm_path}: {str(e)}")
            results["failed"] += 1

    # Monitor all jobs
    print(f"\nMonitoring {len(results['jobs'])} jobs...")
    active_jobs = set(results["jobs"].keys())

    while active_jobs:
        for job_id in list(active_jobs):
            job = client.get_job(job_id)
            status = job["status"]
            vm_path = results["jobs"][job_id]["vm_path"]

            if status == "completed":
                print(f"  ✅ Completed: {vm_path}")
                results["completed"] += 1
                results["jobs"][job_id]["status"] = "completed"
                active_jobs.remove(job_id)
            elif status == "failed":
                print(f"  ❌ Failed: {vm_path} - {job.get('error')}")
                results["failed"] += 1
                results["jobs"][job_id]["status"] = "failed"
                active_jobs.remove(job_id)
            elif job.get("progress"):
                pct = job["progress"]["percent_complete"]
                results["jobs"][job_id]["progress"] = pct

        if active_jobs:
            # Show overall progress
            total_progress = sum(
                results["jobs"][jid].get("progress", 0)
                for jid in active_jobs
            ) / len(active_jobs)
            print(f"  ⏳ Overall: {total_progress:.1f}% ({results['completed']}/{results['total']} complete)")
            time.sleep(10)

    return results

# Usage
client = HyperSDK("http://localhost:8080")
vms = [
    "/datacenter/vm/web-01",
    "/datacenter/vm/web-02",
    "/datacenter/vm/web-03",
    "/datacenter/vm/db-01"
]

results = batch_export_with_tracking(client, vms)
print(f"\n📊 Final Results:")
print(f"  Total: {results['total']}")
print(f"  Completed: {results['completed']}")
print(f"  Failed: {results['failed']}")
```

---

## 📱 Dashboard Quick Access

Once the daemon is running, access the web dashboard:

**URL**: http://localhost:8080

**Features**:
- Real-time job monitoring
- VM console access (VNC/Serial)
- Cost estimation calculator
- Schedule management
- Libvirt VM operations

---

## 🔧 Configuration Tips

### Environment Variables

```bash
# vCenter Connection
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='admin@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1  # Skip TLS verification (dev only!)

# Daemon Settings
export DAEMON_ADDR='localhost:8080'
export LOG_LEVEL='info'  # debug, info, warn, error

# Performance
export DOWNLOAD_WORKERS=4  # Concurrent downloads
```

### YAML Configuration

Create `/etc/hypervisord/config.yaml`:

```yaml
VCenterURL: "https://vcenter.example.com/sdk"
Username: "admin@vsphere.local"
Password: "your-password"
Insecure: true
DaemonAddr: "localhost:8080"
LogLevel: "info"
DownloadWorkers: 4
EnableCBT: true
DefaultFormat: "qcow2"
```

---

## 🆘 Troubleshooting

### Connection Issues

```bash
# Test vCenter connectivity
curl -k https://vcenter.example.com/sdk

# Check daemon logs
docker logs hypervisord
# or
journalctl -u hypervisord -f
```

### Permission Issues

```bash
# Ensure export directory exists and is writable
mkdir -p /exports
chmod 755 /exports

# For Docker, check volume mounts
docker inspect hypervisord | grep Mounts -A 10
```

### API Errors

```python
from hypersdk import HyperSDK
from hypersdk.errors import HyperSDKError, AuthenticationError

client = HyperSDK("http://localhost:8080")

try:
    job_id = client.submit_job({...})
except AuthenticationError as e:
    print(f"Authentication failed: {e}")
except HyperSDKError as e:
    print(f"API error: {e}")
```

---

## 📚 Next Steps

### Learn More

1. **Read Feature Docs**: [docs/FEATURES_OVERVIEW.md](FEATURES_OVERVIEW.md)
2. **Explore Examples**: [examples/](../examples/)
3. **API Reference**: [docs/API_ENDPOINTS.md](API_ENDPOINTS.md)
4. **Advanced Scheduling**: [docs/features/ADVANCED_SCHEDULING.md](features/ADVANCED_SCHEDULING.md)
5. **Cost Estimation**: [docs/features/COST_ESTIMATION.md](features/COST_ESTIMATION.md)

### Try Advanced Features

- **Incremental Backups**: Enable CBT for 95% faster backups
- **Cost Optimization**: Compare cloud providers before exporting
- **Automated Scheduling**: Set up recurring backups with dependencies
- **Format Conversion**: Convert VMs for different platforms
- **Plugin Management**: Hot-load providers without downtime

### Join the Community

- **GitHub**: https://github.com/ssahani/hypersdk
- **Issues**: Report bugs or request features
- **Discussions**: Ask questions and share experiences
- **Contributing**: See [CONTRIBUTING.md](../CONTRIBUTING.md)

---

## 🎉 Success!

You're now ready to use HyperSDK! Start with a simple export and gradually explore advanced features like incremental backups, cost estimation, and automated scheduling.

**Quick Commands Recap**:

```bash
# Health check
curl http://localhost:8080/health

# Submit job
curl -X POST http://localhost:8080/jobs/submit -d '{...}'

# Check status
curl http://localhost:8080/jobs/<job-id>

# List all jobs
curl http://localhost:8080/jobs/query?all=true
```

**Happy Exporting!** 🚀

---

*For more information, see the [complete documentation](00-INDEX.md)*
