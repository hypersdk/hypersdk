# Web UX Pipeline Integration Summary

## Overview

Successfully integrated the hyper2kvm pipeline and libvirt VM definition functionality into the web dashboard, providing a complete end-to-end VM migration workflow through the browser.

## Changes Made

### 1. Frontend (React Dashboard)

**File**: `web/dashboard-react/src/components/JobSubmissionForm.tsx`

#### Added Form State Fields

```typescript
const [formData, setFormData] = useState<Record<string, any>>({
  // ... existing fields ...

  // Pipeline integration (NEW)
  enable_pipeline: false,
  hyper2kvm_path: '/home/tt/hyper2kvm/hyper2kvm',
  pipeline_inspect: true,
  pipeline_fix: true,
  pipeline_convert: true,
  pipeline_validate: true,
  pipeline_compress: true,
  compress_level: 6,

  // Libvirt integration (NEW)
  libvirt_integration: false,
  libvirt_uri: 'qemu:///system',
  libvirt_autostart: false,
  libvirt_bridge: 'virbr0',
  libvirt_pool: 'default',
});
```

#### Added UI Section (after "Export options")

```tsx
{/* Pipeline Integration */}
<div style={{ marginBottom: '12px', padding: '12px', backgroundColor: '#1a1a1a', borderRadius: '4px' }}>
  <h3 style={{ fontSize: '14px', fontWeight: '600', marginBottom: '8px', color: '#fff' }}>
    Pipeline integration <span style={{ fontSize: '11px', fontWeight: 'normal', color: '#f0583a' }}>(hyper2kvm + libvirt)</span>
  </h3>

  {/* Enable Pipeline Checkbox */}
  {/* hyper2kvm Path Input */}
  {/* Compression Level Input */}
  {/* Pipeline Stages Checkboxes (4 stages) */}
  {/* Libvirt Integration Section (collapsible) */}
</div>
```

**Features**:
- ✅ Collapsible pipeline section (toggle with "Enable pipeline" checkbox)
- ✅ hyper2kvm path configuration
- ✅ 4 pipeline stage checkboxes (INSPECT, FIX, CONVERT, VALIDATE)
- ✅ Compression level selector (1-9)
- ✅ Nested libvirt integration section
- ✅ Libvirt URI, bridge, pool, and auto-start configuration
- ✅ Orange accent color for branding (`#f0583a`)
- ✅ Responsive 2-column grid layout

### 2. Backend (Daemon API)

**File**: `daemon/models/job.go`

#### Extended ExportOptions Model

```go
type ExportOptions struct {
    // ... existing fields ...

    // Pipeline integration options (NEW)
    EnablePipeline      bool   `json:"enable_pipeline,omitempty"`
    Hyper2KVMPath       string `json:"hyper2kvm_path,omitempty"`
    PipelineInspect     bool   `json:"pipeline_inspect,omitempty"`
    PipelineFix         bool   `json:"pipeline_fix,omitempty"`
    PipelineConvert     bool   `json:"pipeline_convert,omitempty"`
    PipelineValidate    bool   `json:"pipeline_validate,omitempty"`
    PipelineCompress    bool   `json:"pipeline_compress,omitempty"`
    CompressLevel       int    `json:"compress_level,omitempty"`

    // Libvirt integration options (NEW)
    LibvirtIntegration bool   `json:"libvirt_integration,omitempty"`
    LibvirtURI         string `json:"libvirt_uri,omitempty"`
    LibvirtAutoStart   bool   `json:"libvirt_autostart,omitempty"`
    LibvirtBridge      string `json:"libvirt_bridge,omitempty"`
    LibvirtPool        string `json:"libvirt_pool,omitempty"`
}
```

**Impact**:
- ✅ API accepts pipeline configuration via JSON/YAML job submissions
- ✅ Backward compatible (all fields are optional with `omitempty`)
- ✅ Works with existing job submission endpoint `/jobs/submit`

### 3. Documentation

**New Files**:
1. `web/dashboard-react/PIPELINE_INTEGRATION.md` - Web UI integration guide
2. `WEB_UX_INTEGRATION_SUMMARY.md` - This summary document

## Integration Flow

### End-to-End Pipeline Workflow

```
┌─────────────┐     ┌──────────────┐     ┌────────────┐     ┌──────────┐     ┌──────────┐
│  Browser    │────>│  Daemon API  │────>│  vSphere   │────>│hyper2kvm│────>│ libvirt  │
│   Form      │     │  /jobs/submit│     │  Provider  │     │ Pipeline │     │   KVM    │
└─────────────┘     └──────────────┘     └────────────┘     └────────────┘     └──────────┘
      │                     │                     │                 │                │
   Submit              Validate              Export           Convert         Define VM
   Options             & Queue               VM               Disk            in KVM
```

### Step-by-Step User Flow

1. **User opens web dashboard** → `http://localhost:8080`
2. **User selects cloud provider** → vSphere, AWS, Azure, etc.
3. **User fills provider credentials** → vCenter URL, username, password, VM path
4. **User configures export options** → Output directory, format (OVA/VMDK/qcow2)
5. **User enables "Pipeline integration"** → Checkbox toggle
6. **User configures pipeline stages** → Select which stages to run
7. **User enables "Libvirt integration"** (optional) → Define VM in KVM after conversion
8. **User clicks "Submit job"** → Job sent to daemon API
9. **Daemon exports VM** → From source provider
10. **Daemon generates manifest** → Artifact Manifest v1.0
11. **Daemon runs hyper2kvm** → Converts VMDK → qcow2, fixes guest OS
12. **Daemon defines VM in libvirt** → VM ready to start

## API Request Example

### Job Submission with Pipeline Enabled

**Request**: `POST /jobs/submit`

```json
{
  "name": "Ubuntu Server Migration",
  "provider": "vsphere",
  "vcenter_url": "vcenter.company.com",
  "datacenter": "DC1",
  "username": "administrator@vsphere.local",
  "password": "SecurePassword123",
  "vm_path": "/DC1/vm/Production/ubuntu-server",
  "output_dir": "/var/lib/libvirt/images/ubuntu-server",
  "format": "ova",
  "options": {
    "parallel_downloads": 3,
    "remove_cdrom": true,
    "enable_pipeline": true,
    "hyper2kvm_path": "/home/tt/hyper2kvm/hyper2kvm",
    "pipeline_inspect": true,
    "pipeline_fix": true,
    "pipeline_convert": true,
    "pipeline_validate": true,
    "pipeline_compress": true,
    "compress_level": 6,
    "libvirt_integration": true,
    "libvirt_uri": "qemu:///system",
    "libvirt_autostart": true,
    "libvirt_bridge": "br0",
    "libvirt_pool": "default"
  }
}
```

**Response**: `200 OK`

```json
{
  "job_ids": ["job-abc123"],
  "accepted": 1,
  "rejected": 0,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Job Status Query

**Request**: `GET /jobs/job-abc123`

**Response**:

```json
{
  "id": "job-abc123",
  "status": "running",
  "progress": 65,
  "message": "Running hyper2kvm pipeline: FIX stage - updating GRUB configuration",
  "started_at": "2024-01-15T10:30:00Z",
  "metadata": {
    "export_completed": true,
    "manifest_generated": true,
    "pipeline_stage": "fix",
    "stages_completed": ["inspect", "fix"],
    "stages_remaining": ["convert", "validate"]
  }
}
```

### Job Completion

**Response**:

```json
{
  "id": "job-abc123",
  "status": "completed",
  "progress": 100,
  "message": "Export completed successfully with pipeline integration",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:47:30Z",
  "result": {
    "output_path": "/var/lib/libvirt/images/ubuntu-server",
    "format": "ova",
    "size": 8589934592,
    "duration": "17m30s",
    "files": [
      "/var/lib/libvirt/images/ubuntu-server/ubuntu-server.ovf",
      "/var/lib/libvirt/images/ubuntu-server/ubuntu-server-disk1.vmdk"
    ],
    "metadata": {
      "manifest_path": "/var/lib/libvirt/images/ubuntu-server/manifest.json",
      "pipeline_success": true,
      "pipeline_duration": "12m15s",
      "converted_path": "/var/lib/libvirt/images/ubuntu-server.qcow2",
      "libvirt_domain": "ubuntu-server",
      "libvirt_uri": "qemu:///system",
      "pipeline_stages": {
        "inspect": "completed",
        "fix": "completed",
        "convert": "completed",
        "validate": "completed"
      }
    }
  }
}
```

## UI Screenshots (Conceptual)

### Pipeline Section - Collapsed
```
┌─────────────────────────────────────────────────────────┐
│ Pipeline integration (hyper2kvm + libvirt)              │
│                                                         │
│ ☐ Enable hyper2kvm pipeline after export               │
└─────────────────────────────────────────────────────────┘
```

### Pipeline Section - Expanded
```
┌─────────────────────────────────────────────────────────┐
│ Pipeline integration (hyper2kvm + libvirt)              │
│                                                         │
│ ☑ Enable hyper2kvm pipeline after export               │
│                                                         │
│ hyper2kvm Path          Compression Level (1-9)        │
│ /home/tt/hyper2kvm...   [6]                            │
│                                                         │
│ Pipeline Stages                                         │
│ ☑ INSPECT (detect OS)   ☑ FIX (fstab, grub)           │
│ ☑ CONVERT (→ qcow2)     ☑ VALIDATE (integrity)        │
│                                                         │
│ ─────────────────────────────────────────────────────  │
│                                                         │
│ ☑ Define VM in libvirt after conversion                │
│                                                         │
│ Libvirt URI             Network Bridge                 │
│ qemu:///system          virbr0                         │
│                                                         │
│ Storage Pool            ☑ Enable VM auto-start         │
│ default                                                │
└─────────────────────────────────────────────────────────┘
```

## Testing Instructions

### 1. Start the Daemon

```bash
cd /home/ssahani/go/github/hypersdk
go build ./cmd/hyperd
./hyperd
```

### 2. Open Web Dashboard

```bash
# In browser
http://localhost:8080
```

### 3. Submit Test Job with Pipeline

1. Select **vSphere** provider
2. Fill in vCenter credentials
3. Enter VM path (e.g., `/DC1/vm/test-vm`)
4. Set output directory to `/tmp/test-export`
5. Select format: **OVA**
6. **Enable Pipeline integration** ✓
7. Keep all pipeline stages checked
8. **Enable libvirt integration** ✓
9. Keep default libvirt settings
10. Click **Submit job**

### 4. Monitor Job Progress

```bash
# Watch job status
curl http://localhost:8080/jobs?all=true | jq

# Watch specific job
curl http://localhost:8080/jobs/JOB_ID | jq
```

### 5. Verify Results

```bash
# Check converted qcow2 image
ls -lh /tmp/test-export/*.qcow2

# Check libvirt domain
virsh list --all

# Check manifest
cat /tmp/test-export/manifest.json | jq
```

## Compatibility Matrix

| Provider    | Export | Manifest | Pipeline | Libvirt |
|-------------|--------|----------|----------|---------|
| vSphere     | ✅     | ✅       | ✅       | ✅      |
| AWS         | ✅     | ✅       | ⚠️*      | ⚠️*     |
| Azure       | ✅     | ✅       | ⚠️*      | ⚠️*     |
| GCP         | ✅     | ✅       | ⚠️*      | ⚠️*     |
| Hyper-V     | ✅     | ✅       | ⚠️*      | ⚠️*     |
| OCI         | ✅     | ⚠️       | ❌       | ❌      |
| OpenStack   | ✅     | ⚠️       | ❌       | ❌      |
| Alibaba     | ✅     | ⚠️       | ❌       | ❌      |
| Proxmox     | ✅     | ⚠️       | ❌       | ❌      |

*⚠️ = Supported but requires manual disk download first (cloud exports go to object storage)*

## Error Handling

### Pipeline Failure (Non-Fatal)

If hyper2kvm fails, the export still succeeds:

```json
{
  "status": "completed_with_warnings",
  "result": {
    "output_path": "/tmp/test-export",
    "metadata": {
      "pipeline_success": false,
      "pipeline_error": "hyper2kvm: disk conversion failed: insufficient disk space"
    }
  }
}
```

User can:
1. View the exported VMDK/OVA files
2. Manually run hyper2kvm with the generated manifest
3. Re-submit job with more disk space

### Libvirt Failure (Non-Fatal)

If libvirt VM definition fails, conversion still completes:

```json
{
  "status": "completed_with_warnings",
  "result": {
    "output_path": "/tmp/test-export",
    "metadata": {
      "pipeline_success": true,
      "converted_path": "/tmp/test-export/vm.qcow2",
      "libvirt_error": "virsh define failed: permission denied - run as root or add user to libvirt group"
    }
  }
}
```

User can:
1. Use the converted qcow2 image
2. Manually define VM: `virsh define /path/to/domain.xml`
3. Fix permissions and re-run

## Future Enhancements

- [ ] **Live Progress Streaming** - WebSocket updates for pipeline stages
- [ ] **Visual Stage Indicator** - Progress bars for each pipeline stage
- [ ] **Manifest Preview** - Show generated manifest before pipeline execution
- [ ] **Pipeline Logs Viewer** - Real-time hyper2kvm output in browser
- [ ] **VM Console Access** - Launch VNC/SPICE console after VM is defined
- [ ] **Multi-Provider Pipeline** - Support pipeline for AWS/Azure/GCP exports
- [ ] **Batch Exports** - Submit multiple VMs with pipeline enabled
- [ ] **Template Profiles** - Save/load pipeline configurations as templates

## Build & Deploy

### Development

```bash
cd web/dashboard-react
npm install
npm run dev
```

Access: `http://localhost:5173`

### Production

```bash
cd web/dashboard-react
npm run build
```

Output: `daemon/dashboard/static-react/`

Served by daemon at: `http://localhost:8080`

## Summary of Files Changed

### New Files Created (3)
1. `providers/common/pipeline.go` - Pipeline executor
2. `providers/common/libvirt.go` - Libvirt integrator
3. `web/dashboard-react/PIPELINE_INTEGRATION.md` - Documentation

### Files Modified (3)
1. `web/dashboard-react/src/components/JobSubmissionForm.tsx` - Added pipeline UI
2. `daemon/models/job.go` - Extended ExportOptions model
3. `providers/vsphere/export_options.go` - Added pipeline fields (already done)

### Total Lines Added: ~450
- Frontend (React): ~180 lines
- Backend (Models): ~20 lines
- Infrastructure (Already completed): ~250 lines

## Conclusion

The web UX now provides a complete, user-friendly interface for VM migrations with integrated hyper2kvm conversion and libvirt VM definition. Users can perform end-to-end migrations entirely through the browser, from export to KVM-ready VM in a single operation.

✅ **Implementation Complete**
✅ **Build Verified**
✅ **Documentation Created**
✅ **Ready for Testing**
