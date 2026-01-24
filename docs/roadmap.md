# HyperSDK Feature Roadmap

This document outlines potential features and enhancements for the HyperSDK ecosystem.

## Current State Summary

### ✅ Implemented Features
- **vSphere Provider**: Complete integration with VMware vCenter
- **Interactive TUI**: Advanced VM selection, filtering, sorting, searching
- **REST API**: Complete daemon API for job management
- **CLI Tools**: hyperctl, hyperexport, hypervisord
- **Batch Operations**: Multi-VM export with parallel downloads
- **Progress Tracking**: Real-time progress bars and status updates
- **Advanced Search**: grep and ripgrep-style VM searching

### 🔍 Current Gaps
- Job persistence (in-memory only)
- Multi-cloud providers (only vSphere implemented)
- Observability/metrics export
- Direct KVM import integration
- Web UI dashboard
- Scheduling/automation

---

## Feature Roadmap

### 🚀 Phase 1: Foundation & Reliability (Q1 2026)

#### 1.1 Job Persistence Layer
**Priority: CRITICAL**

**Problem:** Jobs are lost on daemon restart, no historical tracking

**Implementation:**
```go
// Add SQLite database backend
type JobStore interface {
    SaveJob(job *Job) error
    LoadJob(id string) (*Job, error)
    ListJobs(filter JobFilter) ([]*Job, error)
    UpdateJobStatus(id string, status JobStatus) error
}

// Schema
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    name TEXT,
    vm_path TEXT,
    status TEXT,
    created_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    progress_json TEXT,
    error TEXT
);

CREATE INDEX idx_status ON jobs(status);
CREATE INDEX idx_created_at ON jobs(created_at DESC);
```

**Benefits:**
- Survive daemon restarts
- Historical job tracking
- Audit trail for compliance
- Better debugging of past failures

**Effort:** 2-3 weeks

---

#### 1.2 Prometheus Metrics Export
**Priority: HIGH**

**Implementation:**
```go
// Add /metrics endpoint
import "github.com/prometheus/client_golang/prometheus"

var (
    jobsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "hypersdk_jobs_total",
            Help: "Total number of jobs",
        },
        []string{"status", "provider"},
    )

    jobDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "hypersdk_job_duration_seconds",
            Help: "Job duration in seconds",
        },
        []string{"vm_type", "provider"},
    )

    exportedVMs = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "hypersdk_vms_exported_total",
            Help: "Total VMs exported",
        },
    )

    exportedBytes = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "hypersdk_bytes_exported_total",
            Help: "Total bytes exported",
        },
    )
)
```

**Metrics to track:**
- Jobs: total, rate, duration, success rate
- VMs: count, by OS type, by power state
- Export: bytes transferred, speed, failures
- API: request rate, latency, errors
- System: goroutines, memory, CPU

**Grafana Dashboard:** Create pre-built dashboard

**Effort:** 1-2 weeks

---

#### 1.3 OVA Format Support
**Priority: MEDIUM**

**Current:** Export as OVF (separate files)
**Enhancement:** Package as single OVA file

**Implementation:**
```bash
# Create TAR archive with specific structure
ovftool --acceptAllEulas export.ovf export.ova

# Or native implementation:
tar -cf vm.ova vm.ovf vm-disk1.vmdk vm.mf
```

**Benefits:**
- Easier distribution (single file)
- Import compatibility with more platforms
- Reduced confusion (no file scatter)

**Effort:** 1 week

---

### 🌐 Phase 2: Multi-Cloud Support (Q2 2026)

#### 2.1 Hyper-V Provider
**Priority: HIGH**

**Target:** Windows Server Hyper-V

**Implementation:**
```go
// providers/hyperv/client.go
type HyperVProvider struct {
    host     string
    username string
    password string
    client   *winrm.Client
}

// Use WinRM + PowerShell for VM operations
func (h *HyperVProvider) ListVMs() ([]VMInfo, error) {
    script := "Get-VM | ConvertTo-Json"
    return h.runPowerShell(script)
}
```

**Capabilities:**
- List VMs via PowerShell/WinRM
- Export VMs using Export-VM cmdlet
- Power state management
- Snapshot handling

**Testing:** Requires Windows Server lab environment

**Effort:** 3-4 weeks

---

#### 2.2 Proxmox Provider
**Priority: MEDIUM**

**Target:** Proxmox Virtual Environment (PVE)

**Implementation:**
```go
// providers/proxmox/client.go
type ProxmoxProvider struct {
    apiURL   string
    token    string
    client   *proxmox.Client
}

// Use Proxmox REST API
func (p *ProxmoxProvider) ListVMs() ([]VMInfo, error) {
    return p.client.GetVMs()
}
```

**API Endpoints:**
- `/api2/json/cluster/resources?type=vm`
- `/api2/json/nodes/{node}/qemu/{vmid}/config`

**Effort:** 2-3 weeks

---

#### 2.3 AWS EC2 Provider
**Priority: MEDIUM**

**Implementation:**
```go
// providers/aws/client.go
import "github.com/aws/aws-sdk-go-v2/service/ec2"

type AWSProvider struct {
    region string
    client *ec2.Client
}

func (a *AWSProvider) ListVMs() ([]VMInfo, error) {
    result, err := a.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
    // Convert EC2 instances to VMInfo
}

func (a *AWSProvider) ExportVM(vmID string) error {
    // Create AMI
    // Export to S3
    // Download OVF/VMDK
}
```

**Challenges:**
- Different VM model (AMI-based)
- Requires S3 bucket for export
- IAM permissions needed
- Cost considerations

**Effort:** 4-5 weeks

---

### 🔧 Phase 3: Automation & Integration (Q3 2026)

#### 3.1 Webhook Notifications
**Priority: HIGH**

**Configuration:**
```yaml
# config.yaml
webhooks:
  enabled: true
  endpoints:
    - url: https://slack.com/api/incoming/webhook
      events: [job.completed, job.failed]
      headers:
        Authorization: Bearer token123

    - url: https://myapp.com/api/migration-complete
      events: [job.completed]
      retry: 3
      timeout: 10s
```

**Payload:**
```json
{
  "event": "job.completed",
  "timestamp": "2026-01-17T10:30:00Z",
  "job_id": "abc123",
  "vm_name": "web-server-01",
  "status": "completed",
  "duration_seconds": 1234,
  "exported_files": [
    "/exports/web-server-01/vm.ovf",
    "/exports/web-server-01/disk.vmdk"
  ]
}
```

**Effort:** 1-2 weeks

---

#### 3.2 Job Scheduling
**Priority: MEDIUM**

**Use Cases:**
- Nightly VM backups
- Maintenance window migrations
- Recurring snapshots

**Implementation:**
```go
// Add cron-like scheduling
type ScheduledJob struct {
    ID          string
    JobTemplate JobDefinition
    Schedule    string  // "0 2 * * *" (cron format)
    Enabled     bool
    NextRun     time.Time
}

// Scheduler using robfig/cron
func (s *Scheduler) Start() {
    c := cron.New()
    c.AddFunc(job.Schedule, func() {
        s.executeScheduledJob(job)
    })
    c.Start()
}
```

**API Endpoints:**
```
POST   /schedules          # Create scheduled job
GET    /schedules          # List all schedules
GET    /schedules/{id}     # Get specific schedule
DELETE /schedules/{id}     # Delete schedule
PUT    /schedules/{id}     # Update schedule
```

**Effort:** 2-3 weeks

---

#### 3.3 Job Templates & Workflows
**Priority: MEDIUM**

**Templates:**
```yaml
# templates/linux-vm-export.yaml
name: "{{ .VMName }}-export"
vm_path: "{{ .VMPath }}"
output_path: "/exports/{{ .VMName }}"
options:
  parallel_downloads: 8
  remove_cdrom: true
pre_hooks:
  - type: shutdown
    timeout: 300
  - type: snapshot
    name: "pre-migration-{{ .Date }}"
post_hooks:
  - type: convert
    format: qcow2
  - type: upload
    destination: "s3://migrations/{{ .VMName }}"
```

**Workflow DAG:**
```yaml
workflow:
  name: "datacenter-migration"
  steps:
    - id: export-dbs
      type: batch
      vms: [db-01, db-02, db-03]
      parallel: false  # Sequential

    - id: export-apps
      type: batch
      vms: [app-01, app-02]
      parallel: true
      depends_on: [export-dbs]

    - id: verify
      type: script
      command: "./verify-migration.sh"
      depends_on: [export-apps]

    - id: notify
      type: webhook
      url: https://hooks.slack.com/...
      depends_on: [verify]
```

**Effort:** 3-4 weeks

---

### 📊 Phase 4: Observability & Management (Q4 2026)

#### 4.1 Web Dashboard
**Priority: HIGH**

**Technology Stack:**
- Frontend: React + TypeScript
- Backend: Existing Go API
- Real-time: WebSocket for live updates
- Charts: Recharts or Chart.js

**Pages:**
1. **Dashboard Home:**
   - Active jobs with progress
   - Recent exports
   - Statistics (VMs exported, GB transferred)
   - System health

2. **VM Inventory:**
   - Browse vCenter VMs
   - Filter/search interface
   - Bulk selection
   - Export initiation

3. **Job History:**
   - Searchable job log
   - Duration charts
   - Success/failure trends
   - Export size over time

4. **Scheduled Jobs:**
   - Cron schedule management
   - Enable/disable schedules
   - Execution history

5. **Settings:**
   - Provider configuration
   - Webhook management
   - User preferences

**Implementation:**
```
/web
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── VMList.tsx
│   │   │   ├── JobHistory.tsx
│   │   │   └── JobProgress.tsx
│   │   ├── api/
│   │   │   └── client.ts
│   │   └── App.tsx
│   └── package.json
└── embed.go  # Embed static files in Go binary
```

**Effort:** 6-8 weeks

---

#### 4.2 OpenTelemetry Tracing
**Priority: MEDIUM**

**Implementation:**
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func (e *Exporter) ExportVM(ctx context.Context, vm VMInfo) error {
    ctx, span := otel.Tracer("hypersdk").Start(ctx, "export_vm")
    defer span.End()

    span.SetAttributes(
        attribute.String("vm.name", vm.Name),
        attribute.Int("vm.cpus", vm.NumCPU),
        attribute.Int64("vm.memory", vm.MemoryMB),
    )

    // Export logic with child spans
    ctx, downloadSpan := otel.Tracer("hypersdk").Start(ctx, "download_disks")
    // ...
    downloadSpan.End()
}
```

**Spans to track:**
- VM discovery
- Connection establishment
- Disk enumeration
- File downloads (per file)
- OVF generation
- Post-export hooks

**Backend:** Jaeger or Tempo

**Effort:** 2-3 weeks

---

### ⚡ Phase 5: Performance & Scale (2027)

#### 5.1 VDDK Integration
**Priority: HIGH**

**Problem:** govmomi uses HTTPS for downloads (slower)
**Solution:** VMware VDDK (Virtual Disk Development Kit) for native access

**Performance Gain:** 2-5x faster exports

**Implementation:**
```go
// Use CGO to call VDDK C library
// #cgo LDFLAGS: -lvixDiskLib
// #include <vixDiskLib.h>
import "C"

func (v *VDDKExporter) OpenDisk(path string) error {
    connection := C.VixDiskLib_ConnectEx(...)
    handle := C.VixDiskLib_Open(connection, path, ...)
    // Read disk blocks efficiently
}
```

**Challenges:**
- Requires VDDK library installation
- Platform-specific (Linux/Windows)
- Licensing considerations

**Effort:** 4-6 weeks

---

#### 5.2 Connection Pooling
**Priority: MEDIUM**

**Implementation:**
```go
type ConnectionPool struct {
    pools map[string]*vSpherePool
    mu    sync.RWMutex
}

type vSpherePool struct {
    conns chan *vim25.Client
    max   int
}

func (p *ConnectionPool) GetConnection(url string) (*vim25.Client, error) {
    pool := p.getOrCreatePool(url)
    select {
    case conn := <-pool.conns:
        return conn, nil
    default:
        return p.createConnection(url)
    }
}
```

**Benefits:**
- Reduce authentication overhead
- Faster job startup
- Better vCenter resource usage

**Effort:** 1-2 weeks

---

#### 5.3 Incremental Exports
**Priority: MEDIUM**

**Use Case:** Export only changed blocks since last export

**Implementation:**
```go
type IncrementalExport struct {
    BaseSnapshot   string
    CurrentSnapshot string
    ChangeBlockIDs []string
}

// Use vSphere Changed Block Tracking (CBT)
func (e *Exporter) ExportIncremental(vm VMInfo, baseSnap string) error {
    changes := e.queryChangedDiskAreas(vm, baseSnap)
    for _, block := range changes {
        e.downloadBlock(block)
    }
}
```

**Requirements:**
- VM snapshots
- Changed Block Tracking (CBT) enabled
- Snapshot management

**Effort:** 3-4 weeks

---

### 🔐 Phase 6: Security & Compliance (2027)

#### 6.1 RBAC (Role-Based Access Control)
**Priority: HIGH** (for enterprise use)

**Roles:**
```yaml
roles:
  admin:
    - job.create
    - job.cancel
    - job.delete
    - vm.list
    - vm.export
    - settings.manage

  operator:
    - job.create
    - job.view
    - vm.list
    - vm.export

  viewer:
    - job.view
    - vm.list
```

**Implementation:**
```go
type User struct {
    ID       string
    Username string
    Roles    []string
}

func (a *Authorizer) Authorize(user User, action string) bool {
    for _, role := range user.Roles {
        if a.roleHasPermission(role, action) {
            return true
        }
    }
    return false
}
```

**Effort:** 3-4 weeks

---

#### 6.2 Audit Logging
**Priority: HIGH**

**Implementation:**
```go
type AuditLog struct {
    Timestamp time.Time
    User      string
    Action    string
    Resource  string
    Result    string
    IP        string
    Details   map[string]interface{}
}

// Log all API calls
func (a *AuditLogger) Log(ctx context.Context, action string) {
    user := ctx.Value("user")
    a.write(AuditLog{
        Timestamp: time.Now(),
        User:      user.Username,
        Action:    action,
        // ...
    })
}
```

**Storage:** Separate audit database, immutable logs

**Effort:** 2-3 weeks

---

#### 6.3 Encryption at Rest
**Priority: MEDIUM**

**Encrypt:**
- Stored credentials
- Job definitions with sensitive data
- Exported VM files (optional)

**Implementation:**
```go
import "golang.org/x/crypto/nacl/secretbox"

type SecretStore struct {
    key [32]byte
}

func (s *SecretStore) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := randomNonce()
    encrypted := secretbox.Seal(nonce[:], plaintext, &nonce, &s.key)
    return encrypted, nil
}
```

**Effort:** 2 weeks

---

### 🎯 Phase 7: Advanced Features (Future)

#### 7.1 Live Migration Support
**Priority: LOW** (complex, limited use cases)

**Target:** vMotion-style live migration to KVM

**Challenges:**
- Requires shared storage or live disk sync
- Network interruption minimization
- Application quiesce
- Very complex implementation

**Research Phase:** 3-6 months

---

#### 7.2 AI-Powered Migration Planning
**Priority: LOW** (experimental)

**Capabilities:**
- Predict export duration based on VM size/type
- Suggest optimal migration batches
- Detect migration issues before they occur
- Recommend resource allocation

**Implementation:**
```python
# ML model training
import scikit-learn

model = RandomForestRegressor()
model.fit(historical_exports, durations)

# Predict new export duration
predicted_time = model.predict(vm_features)
```

**Research Phase:** 2-4 months

---

#### 7.3 Disaster Recovery Orchestration
**Priority: LOW**

**Features:**
- Automated failover to KVM
- Replication management
- Recovery point objectives (RPO)
- Recovery time objectives (RTO)
- Runbook automation

**Effort:** 6+ months (major feature)

---

## Implementation Priority Matrix

| Feature | Priority | Effort | Impact | Quarter |
|---------|----------|--------|--------|---------|
| Job Persistence | CRITICAL | 2-3w | High | Q1 2026 |
| Prometheus Metrics | HIGH | 1-2w | High | Q1 2026 |
| OVA Format | MEDIUM | 1w | Medium | Q1 2026 |
| Webhook Notifications | HIGH | 1-2w | High | Q3 2026 |
| Hyper-V Provider | HIGH | 3-4w | High | Q2 2026 |
| Proxmox Provider | MEDIUM | 2-3w | Medium | Q2 2026 |
| Web Dashboard | HIGH | 6-8w | Very High | Q4 2026 |
| Job Scheduling | MEDIUM | 2-3w | Medium | Q3 2026 |
| VDDK Integration | HIGH | 4-6w | Very High | 2027 |
| RBAC | HIGH | 3-4w | High | 2027 |
| OpenTelemetry | MEDIUM | 2-3w | Medium | Q4 2026 |
| AWS Provider | MEDIUM | 4-5w | Medium | Q2 2026 |

---

## Quick Wins (Can implement in 1-2 weeks)

1. **Prometheus /metrics endpoint** - Immediate observability
2. **OVA packaging** - Better user experience
3. **Webhook notifications** - Integration enablement
4. **JSON structured logging** - Better debugging
5. **Connection keep-alive** - Performance improvement
6. **Export compression** - Disk space savings

---

## Community Contribution Opportunities

Easy entry points for external contributors:

1. **Provider implementations** (Proxmox, oVirt, XenServer)
2. **Export format support** (VHD, VHDX)
3. **Language bindings** (Python SDK, JavaScript client)
4. **Documentation** (More examples, translations)
5. **Grafana dashboards** (Pre-built templates)
6. **Ansible modules** (Integration with Ansible)

---

## Success Metrics

Track these KPIs as features are added:

- **Adoption:** Downloads, active installations
- **Performance:** Export speed (MB/s), job completion time
- **Reliability:** Success rate, MTBF (mean time between failures)
- **Efficiency:** Resource usage (CPU, memory, network)
- **User Satisfaction:** GitHub stars, issues resolved, user feedback

---

## Conclusion

HyperSDK has a solid foundation with vSphere support and excellent CLI/API design. The roadmap focuses on:

1. **Reliability** (persistence, metrics)
2. **Extensibility** (multi-cloud providers)
3. **Automation** (scheduling, webhooks, workflows)
4. **Observability** (metrics, tracing, dashboard)
5. **Performance** (VDDK, incremental exports)
6. **Security** (RBAC, audit logs, encryption)

**Recommended First Steps:**
1. Implement job persistence (SQLite)
2. Add Prometheus metrics
3. Create web dashboard MVP
4. Complete Hyper-V provider

This provides immediate value (persistence + metrics) while building towards a comprehensive multi-cloud migration platform.
