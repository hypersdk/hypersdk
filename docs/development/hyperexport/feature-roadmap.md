# HyperExport Feature Roadmap

**Last Updated:** 2026-01-23

## Current Status

✅ **Completed:**
- Concurrent exports for all cloud providers (AWS, Azure, GCP, Hyper-V, vSphere)
- Live progress tracking with callbacks
- TUI with 23 enhancements (vSphere)
- Export queue management
- Export history
- Advanced filtering and search
- Snapshot management
- Resource planning
- Migration wizard

---

## Feature Categories

### 🔴 High Priority (Quick Wins)
Features that provide immediate value with reasonable effort.

### 🟡 Medium Priority (Valuable Enhancements)
Features that significantly improve functionality but require more effort.

### 🟢 Low Priority (Nice to Have)
Features that are useful but not critical.

---

## 🔴 High Priority Features

### 1. Export Resumption & Checkpoint Recovery
**Status:** Partial implementation (TODOs found)
**Effort:** Medium
**Value:** High

**Current State:**
```go
// TODO: Implement checkpoint persistence (parallel_download.go:326)
// TODO: Implement checkpoint loading (parallel_download.go:334)
// TODO: Implement actual resumeable download (parallel_download.go:351)
```

**What to Implement:**
- Checkpoint file format (JSON with download state)
- Save progress periodically (every 10 MB or 10 seconds)
- Resume from last checkpoint on failure
- HTTP Range header support for partial downloads
- Integrity verification after resume

**Benefits:**
- Recover from network interruptions
- Resume large exports (>100 GB)
- Save time and bandwidth
- Improve reliability

**Example Usage:**
```go
opts := vsphere.ExportOptions{
    EnableCheckpoints: true,
    CheckpointInterval: 10 * time.Second,
    CheckpointPath: "/tmp/checkpoints/",
}

// On failure, resume:
opts.ResumeFromCheckpoint = true
opts.CheckpointFile = "/tmp/checkpoints/vm-web-01.checkpoint"
```

---

### 2. Export Encryption (At Rest)
**Status:** Mentioned but not implemented
**Effort:** Medium
**Value:** High (Security)

**What to Implement:**
- AES-256-GCM encryption during export
- GPG encryption option
- Password-based encryption
- Key management (file-based, KMS integration)
- Transparent decryption on import

**Example Usage:**
```go
opts := vsphere.ExportOptions{
    Encrypt: true,
    EncryptionMethod: "aes256",  // or "gpg"
    EncryptionKey: "/path/to/key.pem",
    // Or password-based:
    EncryptionPassword: "secure-password",
}
```

**Implementation:**
```go
// Wrap writer with encryption
encWriter := NewAES256Writer(file, encryptionKey)
io.Copy(encWriter, downloadReader)
```

---

### 3. Bandwidth Throttling & Rate Limiting
**Status:** Not implemented
**Effort:** Low
**Value:** High

**What to Implement:**
- Per-export rate limiting
- Global bandwidth cap
- Time-based throttling (e.g., slower during business hours)
- QoS priority levels

**Example Usage:**
```go
opts := vsphere.ExportOptions{
    BandwidthLimit: 50 * 1024 * 1024, // 50 MB/s
    BandwidthBurst: 10 * 1024 * 1024, // 10 MB burst
    AdaptiveThrottling: true,          // Adjust based on network
}
```

**Implementation:**
```go
import "golang.org/x/time/rate"

// Create rate limiter
limiter := rate.NewLimiter(rate.Limit(bytesPerSecond), burstSize)

// Wrap reader
throttledReader := &ThrottledReader{
    reader: downloadReader,
    limiter: limiter,
}
```

---

### 4. Export Validation & Verification
**Status:** Partial (TODO in validation.go:435)
**Effort:** Low-Medium
**Value:** High (Data Integrity)

**What to Implement:**
- SHA-256 checksum calculation during export
- Automatic verification after export
- OVF/OVA manifest validation
- Disk integrity checks
- Detailed validation reports

**Example Usage:**
```go
opts := vsphere.ExportOptions{
    ValidateChecksum: true,
    GenerateManifest: true,
    VerifyManifest: true,
    ValidationLevel: "strict", // strict, standard, minimal
}

// After export:
report, err := ValidateExport(exportPath)
if !report.AllPassed {
    log.Printf("Validation failed: %v", report.Failures)
}
```

**Validation Checks:**
- ✓ File checksums match manifest
- ✓ All expected files present
- ✓ OVF descriptor is valid XML
- ✓ Disk sizes match specifications
- ✓ No corruption detected

---

### 5. Incremental Export Implementation
**Status:** Placeholder (incremental.go has TODOs)
**Effort:** High
**Value:** Very High (Bandwidth & Time Savings)

**Current State:**
```go
// TODO: Implement GetVMDisks in vsphere.VSphereClient (incremental.go:147)
result.Reason = "Disk change detection not yet implemented"
```

**What to Implement:**
- Changed Block Tracking (CBT) integration
- Differential exports (only changed blocks)
- Snapshot-based incremental backups
- Incremental metadata tracking
- Full vs. incremental decision logic

**Example Usage:**
```go
// Initial full export
fullExport := vsphere.ExportOptions{
    IncrementalMode: false,
    BaselineSnapshot: "baseline-2026-01",
}

// Later incremental export
incrementalExport := vsphere.ExportOptions{
    IncrementalMode: true,
    BaselineSnapshot: "baseline-2026-01",
    IncrementalSnapshot: "daily-2026-01-23",
    ExportOnlyChanges: true,
}
```

**Benefits:**
- 10-100x faster for small changes
- Minimal bandwidth usage
- Frequent backups possible
- Lower storage requirements

---

### 6. Snapshot Integration (vSphere Client)
**Status:** TODOs in snapshot.go
**Effort:** Medium
**Value:** High

**Current TODOs:**
```go
// TODO: Implement CreateSnapshot in vsphere.VSphereClient (snapshot.go:72)
// TODO: Implement DeleteSnapshot in vsphere.VSphereClient (snapshot.go:103)
// TODO: Implement ListSnapshots in vsphere.VSphereClient (snapshot.go:118)
// TODO: Implement RevertToSnapshot in vsphere.VSphereClient (snapshot.go:170)
```

**What to Implement:**
- Full vSphere snapshot API integration
- Snapshot creation before export
- Automatic snapshot cleanup
- Snapshot tree management
- Quiesce filesystem option

**Example:**
```go
// In providers/vsphere/client.go
func (c *VSphereClient) CreateSnapshot(ctx context.Context, vm *object.VirtualMachine, name, description string, memory, quiesce bool) (*types.ManagedObjectReference, error) {
    task, err := vm.CreateSnapshot(ctx, name, description, memory, quiesce)
    if err != nil {
        return nil, err
    }

    taskInfo, err := task.WaitForResult(ctx)
    if err != nil {
        return nil, err
    }

    return taskInfo.Result.(*types.ManagedObjectReference), nil
}
```

---

## 🟡 Medium Priority Features

### 7. Multi-Cloud TUI Integration
**Status:** Not started (vSphere-only TUI exists)
**Effort:** High
**Value:** High

**What to Implement:**
- Unified VM interface across providers
- Provider detection and switching
- Multi-cloud concurrent exports in TUI
- Provider-specific configuration screens
- Cloud credential management UI

**Architecture:**
```go
// Unified VM interface
type CloudVM interface {
    GetName() string
    GetProvider() string    // "vsphere", "aws", "azure", "gcp", "hyperv"
    GetID() string
    GetPowerState() string
    GetResources() ResourceInfo
    Export(ctx context.Context, opts ExportOptions) error
}

// Implement for each provider
type VSphereVM struct { /* ... */ }
type AWSEC2Instance struct { /* ... */ }
type AzureVM struct { /* ... */ }
```

**TUI Changes:**
```go
type tuiModel struct {
    provider       string      // Current active provider
    providers      []string    // Available providers
    vms            []CloudVM   // Provider-agnostic VMs
    activeExports  map[string]*activeExportState
}
```

---

### 8. Export Templates & Profiles
**Status:** Not implemented
**Effort:** Medium
**Value:** Medium-High

**What to Implement:**
- Save export configurations as templates
- Load templates by name
- Template library management
- Template sharing (JSON export/import)
- Batch operations from templates

**Example:**
```yaml
# template-web-servers.yaml
name: "Web Server Backup"
description: "Standard backup for web servers"
providers:
  vsphere:
    format: "ova"
    compress: true
    cleanup_ovf: true
  aws:
    format: "vmdk"
    s3_bucket: "backup-web"
filters:
  tags:
    - "env:production"
    - "tier:web"
schedule: "0 2 * * *"  # 2 AM daily
retention: 30d
```

**Usage:**
```bash
hyperexport --template web-servers --output /backups/web/
hyperexport --list-templates
hyperexport --save-template my-config --output /tmp/my-config.yaml
```

---

### 9. Export Scheduling & Automation
**Status:** Partial (daemon exists but not fully integrated)
**Effort:** Medium
**Value:** High

**What to Implement:**
- Cron-style scheduling
- Recurring export policies
- Retention policies (keep last N, keep for X days)
- Automatic cleanup of old exports
- Email notifications on completion/failure
- Webhook integration

**Example:**
```yaml
# schedule-config.yaml
schedules:
  - name: "daily-web-backup"
    cron: "0 2 * * *"  # 2 AM daily
    vms: ["web-01", "web-02", "web-03"]
    template: "web-server-backup"
    retention:
      count: 7
      days: 30
    notifications:
      email: "admin@company.com"
      webhook: "https://hooks.slack.com/..."

  - name: "weekly-full-backup"
    cron: "0 3 * * 0"  # 3 AM Sunday
    vms: ["*"]  # All VMs
    template: "full-backup"
    incremental: false
```

**Usage:**
```bash
hyperexport-daemon --schedule /etc/hyperexport/schedules.yaml
hyperexport-daemon --list-jobs
hyperexport-daemon --run-now daily-web-backup
```

---

### 10. REST API Server
**Status:** Partial (daemon API exists)
**Effort:** Medium
**Value:** Medium-High

**What to Implement:**
- Full REST API for all operations
- Export job submission
- Progress streaming via Server-Sent Events
- WebSocket support for real-time updates
- Authentication (API keys, OAuth)
- API documentation (Swagger/OpenAPI)

**API Endpoints:**
```
POST   /api/v1/exports              - Create export job
GET    /api/v1/exports              - List export jobs
GET    /api/v1/exports/{id}         - Get export status
DELETE /api/v1/exports/{id}         - Cancel export
GET    /api/v1/exports/{id}/stream  - SSE progress stream
GET    /api/v1/vms                  - List VMs
POST   /api/v1/vms/{id}/export      - Export specific VM
GET    /api/v1/templates            - List templates
POST   /api/v1/templates            - Create template
```

**Example:**
```bash
# Submit export job
curl -X POST http://localhost:8080/api/v1/exports \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "provider": "vsphere",
    "vms": ["web-01", "web-02"],
    "options": {
      "format": "ova",
      "output_path": "/backups"
    }
  }'

# Stream progress
curl -N http://localhost:8080/api/v1/exports/job-123/stream
```

---

### 11. Performance Metrics & Monitoring
**Status:** Not implemented
**Effort:** Medium
**Value:** Medium

**What to Implement:**
- Prometheus metrics endpoint
- Grafana dashboard templates
- Export performance metrics
- Resource usage tracking
- Alert rules

**Metrics:**
```
# Export metrics
hyperexport_exports_total{provider="vsphere",status="success"}
hyperexport_exports_total{provider="vsphere",status="failed"}
hyperexport_export_duration_seconds{provider="vsphere"}
hyperexport_export_bytes_total{provider="vsphere"}

# Performance metrics
hyperexport_download_speed_bytes_per_second{vm="web-01"}
hyperexport_active_exports{provider="vsphere"}
hyperexport_queue_length

# Resource metrics
hyperexport_cpu_usage_percent
hyperexport_memory_usage_bytes
hyperexport_disk_usage_bytes
```

**Grafana Dashboard:**
- Export success/failure rate
- Average export duration
- Bandwidth utilization
- Queue depth over time
- Top exported VMs

---

### 12. Cloud Upload Integration (Streaming)
**Status:** Partial implementation
**Effort:** Medium-High
**Value:** Medium

**What to Implement:**
- Direct streaming to cloud storage (no local disk)
- Multi-part upload for large files
- Parallel chunk uploads
- Resume interrupted uploads
- Cloud provider optimization

**Example:**
```go
opts := vsphere.ExportOptions{
    StreamUpload: true,           // Don't save locally
    UploadTarget: "s3://my-backup-bucket/exports/",
    UploadConcurrency: 5,        // 5 parallel upload streams
    ChunkSize: 10 * 1024 * 1024, // 10 MB chunks
}

// For very large VMs, this saves local disk space
```

**Supported Targets:**
- `s3://bucket/path`
- `azure://container/path`
- `gs://bucket/path`
- `sftp://host/path`

---

## 🟢 Low Priority Features

### 13. Export Compression Options
**Status:** Basic OVA compression exists
**Effort:** Low-Medium
**Value:** Medium

**What to Implement:**
- Multiple compression algorithms (gzip, zstd, lz4, bzip2)
- Compression level control
- Parallel compression
- Transparent decompression on import

**Example:**
```go
opts := vsphere.ExportOptions{
    Compress: true,
    CompressionAlgorithm: "zstd",  // gzip, zstd, lz4, bzip2
    CompressionLevel: 9,            // 1-9
    ParallelCompression: 4,         // 4 threads
}
```

---

### 14. Export Format Conversion
**Status:** hyper2kvm exists for conversion
**Effort:** Medium
**Value:** Medium

**What to Implement:**
- In-place format conversion during export
- VMDK → QCOW2 conversion
- VHD → VHDX conversion
- Thin → Thick disk conversion
- Integration with qemu-img

**Example:**
```go
opts := vsphere.ExportOptions{
    Format: "ova",
    ConvertDisksTo: "qcow2",      // Convert VMDKs to QCOW2
    OptimizeDisks: true,           // Thin provision, zero-fill
    ConversionTool: "qemu-img",    // or "hyper2kvm"
}
```

---

### 15. Detailed Export Reports
**Status:** Basic history exists
**Effort:** Low
**Value:** Low-Medium

**What to Implement:**
- PDF export reports
- HTML dashboards
- Export statistics
- Cost analysis (cloud bandwidth, storage)
- Performance trends

**Report Contents:**
- Export summary (success/failure)
- File sizes and compression ratios
- Export duration and speed
- Disk breakdown
- Validation results
- Cost estimates

---

### 16. Multi-Tenancy Support
**Status:** Not implemented
**Effort:** High
**Value:** Low (Enterprise)

**What to Implement:**
- Tenant isolation
- Per-tenant quotas
- RBAC integration
- Audit logging
- Tenant-specific configurations

---

### 17. Disaster Recovery Features
**Status:** Not implemented
**Effort:** High
**Value:** Medium (Enterprise)

**What to Implement:**
- Geo-redundant backups
- Automatic failover
- Cross-region replication
- Recovery time objective (RTO) tracking
- Recovery point objective (RPO) tracking

---

## Implementation Priority Matrix

| Feature | Value | Effort | Priority | Quick Win |
|---------|-------|--------|----------|-----------|
| Export Resumption | High | Medium | 🔴 High | ✓ |
| Encryption | High | Medium | 🔴 High | ✓ |
| Bandwidth Throttling | High | Low | 🔴 High | ✓✓ |
| Export Validation | High | Low-Med | 🔴 High | ✓ |
| Incremental Exports | V.High | High | 🔴 High | - |
| Snapshot Integration | High | Medium | 🔴 High | ✓ |
| Multi-Cloud TUI | High | High | 🟡 Medium | - |
| Export Templates | Med-High | Medium | 🟡 Medium | ✓ |
| Scheduling | High | Medium | 🟡 Medium | ✓ |
| REST API | Med-High | Medium | 🟡 Medium | ✓ |
| Monitoring/Metrics | Medium | Medium | 🟡 Medium | - |
| Streaming Upload | Medium | Med-High | 🟡 Medium | - |
| Compression Options | Medium | Low-Med | 🟢 Low | ✓ |
| Format Conversion | Medium | Medium | 🟢 Low | - |
| Export Reports | Low-Med | Low | 🟢 Low | ✓✓ |

**Legend:**
- ✓✓ = Very Quick Win (< 2 days)
- ✓ = Quick Win (< 1 week)
- - = Longer effort

---

## Recommended Implementation Order

### Phase 1: Reliability & Security (1-2 weeks)
1. **Bandwidth Throttling** (2 days) - Immediate value
2. **Export Validation** (3 days) - Data integrity
3. **Encryption** (5 days) - Security requirement
4. **Export Resumption** (5 days) - Reliability

### Phase 2: Efficiency & Automation (2-3 weeks)
5. **Snapshot Integration** (1 week) - Complete existing TODOs
6. **Export Templates** (3 days) - Reusability
7. **Scheduling** (1 week) - Automation
8. **Incremental Exports** (1.5 weeks) - Huge efficiency gain

### Phase 3: Integration & Monitoring (2 weeks)
9. **REST API** (1 week) - External integration
10. **Monitoring/Metrics** (1 week) - Observability

### Phase 4: Advanced Features (3-4 weeks)
11. **Multi-Cloud TUI** (2 weeks) - Unified experience
12. **Streaming Upload** (1 week) - Storage optimization
13. **Compression Options** (3 days) - Space savings

---

## Quick Wins (Can Start Immediately)

### 1. Bandwidth Throttling (2 days)
```bash
# Implement rate limiter wrapper
providers/common/throttled_reader.go
cmd/hyperexport/bandwidth.go
```

### 2. Export Reports (1 day)
```bash
# Generate PDF/HTML reports
cmd/hyperexport/report.go
cmd/hyperexport/templates/report.html
```

### 3. Compression Options (3 days)
```bash
# Add zstd, lz4 support
providers/common/compression.go
```

---

## Which Would You Like to Implement First?

**Top Recommendations:**

1. **🚀 Bandwidth Throttling** - Quick, high value, universally useful
2. **🔒 Export Encryption** - Important for security compliance
3. **✅ Export Validation** - Ensures data integrity
4. **⚡ Incremental Exports** - Huge efficiency gains

**Or tell me:**
- Your specific use case or pain point
- Which features interest you most
- Time constraints (quick win vs. major feature)

I can dive into implementation details for any of these features!
