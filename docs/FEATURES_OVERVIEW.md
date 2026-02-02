# HyperSDK - Complete Features Overview

**Version**: 2.0
**Last Updated**: 2026-02-04
**Status**: Production Ready

## 📋 Table of Contents

- [Core Features](#core-features)
- [Recent Advanced Features](#recent-advanced-features)
- [Cloud & Multi-Provider](#cloud--multi-provider)
- [Performance & Optimization](#performance--optimization)
- [Security & Compliance](#security--compliance)
- [Automation & Integration](#automation--integration)
- [Management & Monitoring](#management--monitoring)
- [Documentation & SDKs](#documentation--sdks)

---

## 🎯 Core Features

### Multi-Cloud VM Export System

HyperSDK is a high-performance, daemon-based VM export system that provides a provider layer abstraction for multiple clouds.

**Supported Providers** (9 total):
- ✅ **vSphere** (VMware vCenter/ESXi) - Production Ready
- ✅ **AWS** (Amazon EC2) - Production Ready
- ✅ **Azure** (Microsoft Azure VMs) - Production Ready
- ✅ **GCP** (Google Compute Engine) - Production Ready
- ✅ **Hyper-V** (Microsoft Hyper-V) - Production Ready
- ✅ **OCI** (Oracle Cloud Infrastructure) - Production Ready
- ✅ **OpenStack** (Nova/Swift) - Production Ready
- ✅ **Alibaba Cloud** (Aliyun ECS/OSS) - Production Ready
- ✅ **Proxmox VE** (Proxmox Virtual Environment) - Production Ready

### Three Powerful Tools

| Component | Purpose | Use Case |
|-----------|---------|----------|
| `hyperexport` | Standalone Export Tool | Interactive & scriptable VM exports with CLI flags |
| `hypervisord` | Background Daemon | Automation, REST API, batch processing |
| `hyperctl` | Migration Commander | Interactive TUI migration, daemon control, job management |
| **Web Dashboard** | Browser UI | VM monitoring, console access, job management |

### REST API (57+ Endpoints)

Comprehensive REST API for complete automation:
- Job submission and management
- VM discovery and information
- Schedule and workflow management
- Cost estimation and analysis
- Libvirt/KVM integration
- Real-time progress tracking

---

## 🚀 Recent Advanced Features (2026)

### 1. Multi-Language SDK Clients ⭐⭐⭐

**Implemented**: January 2026
**Status**: Production Ready
**Documentation**: [docs/features/MULTI_LANGUAGE_SDKS.md](features/MULTI_LANGUAGE_SDKS.md)

Complete SDK client libraries for Python and TypeScript with full type safety.

**Features**:
- **Python SDK** (`sdk/python/`) - Type hints, async support, comprehensive docstrings
- **TypeScript SDK** (`sdk/typescript/`) - Full TypeScript types, Promise-based API
- **OpenAPI 3.0 Specification** (`api/openapi/hypersdk.yaml`)
- Auto-generated API documentation
- 50+ SDK methods covering all API endpoints

**Example Usage**:
```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")
job_id = client.submit_job({
    "vm_path": "/datacenter/vm/web-01",
    "output_path": "/exports",
    "format": "ova"
})
```

```typescript
import { HyperSDK } from 'hypersdk';

const client = new HyperSDK('http://localhost:8080');
const jobId = await client.submitJob({
    vm_path: '/datacenter/vm/web-01',
    output_path: '/exports',
    format: 'ova'
});
```

---

### 2. Provider Plugin Hot-Loading ⭐⭐

**Implemented**: January 2026
**Status**: Production Ready
**Documentation**: [docs/features/PLUGIN_HOT_LOADING.md](features/PLUGIN_HOT_LOADING.md)

Dynamic plugin system for cloud providers without daemon restarts.

**Features**:
- Load/unload providers at runtime via REST API
- Zero-downtime provider updates
- Plugin isolation and sandboxing
- Health monitoring and auto-recovery
- Version management and compatibility checking

**Endpoints**:
- `POST /plugins/load` - Load a new provider plugin
- `POST /plugins/unload` - Unload a provider plugin
- `POST /plugins/reload` - Reload a provider plugin
- `GET /plugins/list` - List all loaded plugins
- `GET /plugins/status/{name}` - Get plugin health status

**Example**:
```bash
# Load AWS provider plugin
curl -X POST http://localhost:8080/plugins/load \
  -H "Content-Type: application/json" \
  -d '{
    "name": "aws",
    "path": "/opt/hypersdk/plugins/aws.so",
    "config": {
      "region": "us-east-1"
    }
  }'
```

---

### 3. Native Go Format Converters ⭐⭐

**Implemented**: January 2026
**Status**: Production Ready
**Documentation**: [docs/features/FORMAT_CONVERTERS.md](features/FORMAT_CONVERTERS.md)

Pure Go implementation of VM disk format converters without external dependencies.

**Supported Conversions**:
- **VMDK → QCOW2** (for KVM/QEMU)
- **VMDK → VHD** (for Hyper-V)
- **VMDK → VHDX** (for Hyper-V Gen 2)
- **VMDK → VDI** (for VirtualBox)
- **VMDK → RAW** (universal format)

**Features**:
- Zero external dependencies (no qemu-img required)
- Streaming conversion for memory efficiency
- Progress tracking and ETA estimation
- Automatic format detection
- Compression support
- Concurrent multi-disk conversion

**Example**:
```bash
# Convert VMDK to QCOW2
curl -X POST http://localhost:8080/convert/format \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/exports/vm1.vmdk",
    "target_format": "qcow2",
    "output_path": "/exports/vm1.qcow2"
  }'
```

---

### 4. Incremental Export with CBT ⭐⭐⭐

**Implemented**: January 2026
**Status**: Production Ready
**Documentation**: [docs/features/INCREMENTAL_EXPORT.md](features/INCREMENTAL_EXPORT.md)

Changed Block Tracking (CBT) for 95% faster incremental backups.

**Features**:
- **CBT Integration** - Track changed blocks since last export
- **Incremental Backups** - Export only changed data
- **Base + Delta Model** - Efficient storage with reconstruction capability
- **Automatic Detection** - Smart fallback to full export when needed
- **Progress Tracking** - Real-time tracking of changed blocks

**Performance**:
- **95% faster** than full exports for small changes
- **90% storage savings** for typical workloads
- **Instant recovery** with base + delta reconstruction

**Endpoints**:
- `POST /cbt/enable` - Enable CBT on a VM
- `POST /cbt/disable` - Disable CBT on a VM
- `POST /cbt/status` - Check CBT status and readiness
- `POST /incremental/analyze` - Analyze potential savings

**Example**:
```python
# Enable CBT and analyze savings
client.enable_cbt("/datacenter/vm/web-01")
analysis = client.analyze_incremental_export("/datacenter/vm/web-01")
print(f"Estimated savings: {analysis['estimated_savings_bytes'] / 1e9:.2f} GB")
```

---

### 5. Advanced Scheduling ⭐⭐

**Implemented**: January 2026
**Status**: Production Ready
**Documentation**: [docs/features/ADVANCED_SCHEDULING.md](features/ADVANCED_SCHEDULING.md)

Enterprise-grade job scheduling with dependencies, retries, and time windows.

**Features**:

**Job Dependencies**:
- Wait for other jobs to complete
- Conditional execution based on job state (success/failure/any)
- Timeout support for stalled dependencies
- Automatic dependency resolution

**Retry Policies**:
- Configurable max attempts (1-10)
- Multiple backoff strategies: linear, exponential, fibonacci
- Delay customization (min/max)
- Selective retry on specific error patterns

**Time Windows**:
- Restrict job execution to specific time periods
- Business hours configuration (e.g., 9am-5pm, Mon-Fri)
- Maintenance window support
- Timezone-aware scheduling

**Priority-Based Queue**:
- Priority levels 0-100 (higher = more important)
- Automatic queue management
- Max concurrent job limits
- Skip-if-running option to prevent overlap

**Endpoints**:
- `POST /schedules/advanced/create` - Create advanced schedule
- `GET /schedules/dependencies` - Check dependency status
- `GET /schedules/retry` - Get retry history and next attempt
- `GET /schedules/timewindow` - Check time window status
- `GET /schedules/queue` - View job queue status
- `POST /schedules/validate` - Validate schedule configuration

**Example**:
```typescript
await client.createAdvancedSchedule({
  name: 'weekly-backup',
  schedule: '0 2 * * 0', // 2 AM every Sunday
  jobTemplate: {
    vm_path: '/datacenter/vm/database',
    output_path: '/backups',
    format: 'ova'
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
      days: ['sunday', 'saturday'],
      timezone: 'America/New_York'
    }],
    priority: 80,
    skip_if_running: true
  }
});
```

---

### 6. Cost Estimation ⭐

**Implemented**: February 2026
**Status**: Production Ready
**Documentation**: [docs/features/COST_ESTIMATION.md](features/COST_ESTIMATION.md)

Pre-export cost analysis for cloud storage across AWS S3, Azure Blob, and Google Cloud Storage.

**Features**:
- **Multi-Cloud Pricing** - Realistic pricing data for S3, Azure Blob, GCS (2026-01)
- **Storage Classes** - All tiers supported (Standard, IA, Glacier, Archive, etc.)
- **Detailed Breakdowns** - Storage, transfer, requests, retrieval, early deletion costs
- **Cost Comparison** - Automatic provider comparison with recommendations
- **Yearly Projections** - Monthly breakdown and annual forecasts
- **Export Size Estimation** - Compression-aware size calculations

**Supported Providers & Storage Classes**:

**AWS S3**:
- Standard ($0.023/GB), IA ($0.0125/GB), One Zone IA ($0.01/GB)
- Glacier ($0.004/GB), Deep Archive ($0.00099/GB)

**Azure Blob**:
- Hot ($0.0184/GB), Cool ($0.01/GB), Archive ($0.002/GB)

**Google Cloud Storage**:
- Standard ($0.02/GB), Nearline ($0.01/GB), Coldline ($0.004/GB), Archive ($0.0012/GB)

**Endpoints**:
- `POST /cost/estimate` - Estimate costs for specific provider
- `POST /cost/compare` - Compare costs across all providers
- `POST /cost/project` - Project yearly costs
- `POST /cost/estimate-size` - Estimate export size

**Example**:
```python
# Compare cloud storage costs
comparison = client.compare_providers(
    storage_gb=500,
    transfer_gb=100,
    requests=10000,
    duration_days=90
)

print(f"Cheapest provider: {comparison['cheapest']}")
print(f"Estimated cost: ${comparison['estimates'][0]['total_cost']:.2f}")
print(f"Savings vs most expensive: ${comparison['savings_vs_expensive']:.2f}")
```

---

## ☁️ Cloud & Multi-Provider

### Cloud Provider Support

**9 Production-Ready Providers**:
- vSphere, AWS, Azure, GCP, Hyper-V, OCI, OpenStack, Alibaba Cloud, Proxmox VE

**Direct SDK Integration**:
- Native Go SDK clients (no external binaries)
- Connection pooling (30-40% faster)
- Automatic retry with exponential backoff
- Credential management

### Cloud Storage Integration

**Upload Destinations**:
- AWS S3 (with S3-compatible storage)
- Azure Blob Storage
- Google Cloud Storage
- SFTP servers

**Features**:
- Stream upload (no local copy needed)
- Multi-part uploads for large files
- Progress tracking
- Automatic cleanup

---

## ⚡ Performance & Optimization

### Concurrent Processing

- **Goroutine-based worker pool** - Parallel VM exports
- **Configurable concurrency** - Adjust worker count
- **Connection pooling** - 30-40% performance improvement
- **Resumable downloads** - Automatic retry on failure

### Incremental Backups

- **Changed Block Tracking** - 95% faster backups
- **90% storage savings** - Only export changed data
- **Smart detection** - Automatic fallback to full export

### Format Conversion

- **Native Go converters** - No external dependencies
- **Streaming conversion** - Memory-efficient processing
- **Concurrent multi-disk** - Convert multiple disks in parallel

---

## 🔒 Security & Compliance

### Encryption

**AES-256-GCM**:
- Strong symmetric encryption
- Passphrase-based key derivation (PBKDF2)
- Chunked encryption for large files

**GPG Encryption**:
- Public-key encryption
- Integration with system GPG
- Recipient-based encryption

### Authentication & Authorization

- API key management
- Session-based authentication
- Role-based access control (RBAC)
- Audit logging

### Security Features

- TLS/SSL support
- Encrypted credentials storage
- Secure webhook authentication
- Compliance framework support

---

## 🤖 Automation & Integration

### REST API

**57+ Endpoints** organized by category:
- Core: health, status, capabilities
- Jobs: submit, query, cancel, progress, logs
- VM Management: list, info, operations
- Scheduling: create, list, enable/disable, trigger
- Webhooks: add, test, delete
- Libvirt: domains, snapshots, networks, volumes
- Cost: estimate, compare, project
- Advanced: CBT, format conversion, plugin management

### Webhook Integration

- Real-time job notifications
- Custom event triggers
- Retry logic with backoff
- Authentication support
- Multiple webhook destinations

### Schedule Persistence

- SQLite-based job scheduling
- Cron-style schedule definitions
- Automatic execution
- Job history and statistics

---

## 📊 Management & Monitoring

### Web Dashboard

**React-based Modern UI**:
- Real-time VM monitoring
- WebSocket live updates
- Job management interface
- Console access (VNC, Serial)
- Cost estimation tools
- Schedule management

### Progress Tracking

- Real-time progress bars
- Speed calculation (MB/s)
- ETA estimation
- Per-file progress for multi-disk VMs
- WebSocket streaming updates

### Logging & Monitoring

- Structured logging (JSON)
- Multiple log levels
- Audit trail
- Job history (last 1000 exports)
- Statistical reports
- Alert rules and notifications

### Libvirt/KVM Integration

**Full VM Management**:
- Domain operations (start, stop, reboot, pause, resume)
- Snapshot management (create, revert, delete)
- Network management
- Volume operations
- Resource monitoring (CPU, memory, disk, network I/O)
- Batch operations

---

## 📚 Documentation & SDKs

### Multi-Language SDKs

**Python SDK** (`sdk/python/`):
- Type hints and mypy support
- Async/await support
- Comprehensive docstrings
- PyPI package ready

**TypeScript SDK** (`sdk/typescript/`):
- Full TypeScript types
- Promise-based API
- npm package ready
- Works in Node.js and browsers

**OpenAPI 3.0**:
- Complete API specification
- Auto-generated documentation
- Client code generation support

### Documentation

**Comprehensive Guides**:
- Feature documentation for each major feature
- API reference with examples
- SDK usage guides
- Best practices and tutorials
- Deployment guides (Docker, Kubernetes, OpenShift, Helm)

**Deployment Options**:
- Standalone binary
- Docker containers
- Kubernetes Deployments
- Helm charts
- OpenShift Routes and Templates
- systemd service units

---

## 📈 Statistics & Metrics

### Current Status

- **584+ Tests** - Comprehensive test coverage
- **100% API Handler Coverage** - All endpoints tested
- **57+ REST Endpoints** - Complete automation API
- **9 Cloud Providers** - Multi-cloud support
- **6 Disk Formats** - VMDK, QCOW2, VHD, VHDX, VDI, RAW
- **3 SDKs** - OpenAPI, Python, TypeScript
- **6 Major Features** - Implemented in 2026

### Performance Metrics

- **30-40% faster** - Connection pooling
- **95% faster** - Incremental backups with CBT
- **90% storage savings** - Changed block tracking
- **Zero downtime** - Plugin hot-loading
- **Sub-second** - Cost estimation calculations

---

## 🎯 Use Cases

### Enterprise Backup

- Scheduled incremental backups with CBT
- Multi-cloud redundancy
- Cost-optimized storage selection
- Retention policies
- Encryption at rest

### Cloud Migration

- Multi-cloud VM migration
- Format conversion for target platforms
- Cost analysis before migration
- Progress tracking and monitoring
- Automated workflows

### Disaster Recovery

- Fast incremental backups
- Multi-region replication
- Quick restoration
- Automated failover
- Health monitoring

### Development & Testing

- VM cloning and templates
- Rapid provisioning
- Snapshot-based testing
- Cost-effective storage
- Automated cleanup

### Compliance & Governance

- Audit logging
- Role-based access control
- Encrypted backups
- Compliance framework support
- Retention policies

---

## 🔄 Integration Examples

### Python Workflow

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# 1. Enable CBT for incremental backups
client.enable_cbt("/datacenter/vm/production-db")

# 2. Analyze potential savings
analysis = client.analyze_incremental_export("/datacenter/vm/production-db")
print(f"Changed blocks: {analysis['estimated_savings_bytes'] / 1e9:.2f} GB")

# 3. Estimate cloud storage costs
estimate = client.estimate_cost(
    provider="s3",
    region="us-east-1",
    storage_class="s3-glacier",
    storage_gb=analysis['estimated_savings_bytes'] / 1e9,
    transfer_gb=0,
    requests=100,
    duration_days=365
)
print(f"Annual cost: ${estimate['total_cost']:.2f}")

# 4. Create advanced schedule
schedule = client.create_advanced_schedule(
    name="weekly-incremental-backup",
    schedule="0 2 * * 0",  # 2 AM every Sunday
    job_template={
        "vm_path": "/datacenter/vm/production-db",
        "output_path": "/backups",
        "format": "qcow2",
        "incremental": True
    },
    advanced_config={
        "retry_policy": {
            "max_attempts": 3,
            "backoff_strategy": "exponential"
        },
        "time_windows": [{
            "start_time": "02:00",
            "end_time": "06:00",
            "days": ["sunday"],
            "timezone": "UTC"
        }],
        "priority": 90
    }
)
```

### TypeScript Workflow

```typescript
import { HyperSDK } from 'hypersdk';

const client = new HyperSDK('http://localhost:8080');

// Compare cloud providers for cost
const comparison = await client.compareProviders({
  storageGB: 1000,
  transferGB: 100,
  requests: 5000,
  durationDays: 365
});

console.log(`Cheapest: ${comparison.cheapest}`);
console.log(`Annual cost: $${comparison.estimates[0].total_cost.toFixed(2)}`);

// Submit job with cost-optimized settings
const jobId = await client.submitJob({
  vm_path: '/datacenter/vm/web-cluster',
  output_path: '/exports',
  format: 'qcow2',
  compression: true,
  incremental: true
});

// Track progress
const progress = await client.getJobProgress(jobId);
console.log(`Progress: ${progress.percentage}%`);
```

---

## 🛠️ Configuration Management

### YAML Configuration

```yaml
VCenterURL: "https://vcenter.example.com/sdk"
Username: "administrator@vsphere.local"
Password: "your-password"
Insecure: true
DaemonAddr: "localhost:8080"
LogLevel: "info"
DownloadWorkers: 4
EnableCBT: true
DefaultFormat: "qcow2"
```

### Environment Variables

```bash
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1
export DAEMON_ADDR='localhost:8080'
export LOG_LEVEL='info'
```

---

## 📦 Deployment Options

### Docker

```bash
docker run -d \
  --name hypervisord \
  -p 8080:8080 \
  -v /exports:/exports \
  -e GOVC_URL='https://vcenter/sdk' \
  -e GOVC_USERNAME='admin' \
  -e GOVC_PASSWORD='pass' \
  hypersdk/hypervisord:latest
```

### Kubernetes

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

### Helm

```bash
helm install hypersdk ./helm/hypersdk \
  --set vcenter.url='https://vcenter/sdk' \
  --set vcenter.username='admin' \
  --set vcenter.password='pass'
```

### Systemd

```bash
sudo systemctl start hypervisord
sudo systemctl enable hypervisord
sudo systemctl status hypervisord
```

---

## 🎉 Summary

HyperSDK is a production-ready, enterprise-grade VM export and migration system with:

✅ **6 Major Features** added in 2026
✅ **9 Cloud Providers** supported
✅ **57+ REST API Endpoints**
✅ **3 SDK Languages** (OpenAPI, Python, TypeScript)
✅ **584+ Comprehensive Tests**
✅ **100% API Coverage**
✅ **Multiple Deployment Options**
✅ **Extensive Documentation**

**Perfect for**:
- Enterprise backup and disaster recovery
- Cloud migration projects
- Multi-cloud VM management
- Cost-optimized storage
- Automated workflows
- Development and testing environments

---

## 📞 Support & Resources

- **GitHub**: [https://github.com/ssahani/hypersdk](https://github.com/ssahani/hypersdk)
- **Documentation**: `/docs/`
- **API Reference**: `/api/openapi/hypersdk.yaml`
- **Issue Tracker**: GitHub Issues
- **License**: LGPL v3

---

*Last Updated: 2026-02-04*
*Version: 2.0 - Production Ready*
