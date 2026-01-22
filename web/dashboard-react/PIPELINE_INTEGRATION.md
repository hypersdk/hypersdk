# Web Dashboard Pipeline Integration

## Overview

The web dashboard now supports full hyper2kvm pipeline integration and libvirt VM definition through an enhanced job submission form.

## UI Components

### Pipeline Integration Section

Located in `JobSubmissionForm.tsx`, after the "Export options" section.

#### Main Features:

1. **Enable Pipeline Checkbox**
   - Toggles the entire pipeline integration section
   - When disabled, only standard export occurs
   - When enabled, shows pipeline configuration options

2. **hyper2kvm Configuration**
   - **hyper2kvm Path**: Path to the hyper2kvm executable (default: `/home/tt/hyper2kvm/hyper2kvm`)
   - **Compression Level**: qcow2 compression level 1-9 (default: 6)

3. **Pipeline Stages** (4 checkboxes)
   - **INSPECT**: Detect OS, analyze drivers, collect guest info
   - **FIX**: Fix fstab, GRUB, initramfs, remove VMware tools
   - **CONVERT**: Convert VMDK → qcow2 with compression
   - **VALIDATE**: Check image integrity after conversion

4. **Libvirt Integration**
   - **Enable Checkbox**: Toggles libvirt VM definition
   - **Libvirt URI**: Connection URI (default: `qemu:///system`)
   - **Network Bridge**: Network bridge name (default: `virbr0`)
   - **Storage Pool**: Storage pool name (default: `default`)
   - **Auto-start Checkbox**: Enable VM auto-start on boot

## Form Data Structure

```typescript
{
  // Standard export fields
  name: string,
  provider: string,
  output_dir: string,
  format: string,
  compress: boolean,

  // Pipeline integration
  enable_pipeline: boolean,
  hyper2kvm_path: string,
  pipeline_inspect: boolean,
  pipeline_fix: boolean,
  pipeline_convert: boolean,
  pipeline_validate: boolean,
  pipeline_compress: boolean,
  compress_level: number,

  // Libvirt integration
  libvirt_integration: boolean,
  libvirt_uri: string,
  libvirt_autostart: boolean,
  libvirt_bridge: string,
  libvirt_pool: string
}
```

## API Integration

### Job Submission Endpoint

`POST /jobs/submit`

```json
{
  "name": "My VM Export",
  "vm_path": "/DC1/vm/my-vm",
  "output_dir": "/var/lib/libvirt/images/my-vm",
  "format": "ova",
  "options": {
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
    "libvirt_autostart": false,
    "libvirt_bridge": "virbr0",
    "libvirt_pool": "default"
  }
}
```

### Backend Processing

The daemon API receives the job submission and passes the pipeline options to the vSphere provider, which:

1. Exports the VM from vSphere
2. Generates Artifact Manifest v1.0
3. Executes hyper2kvm pipeline (if `enable_pipeline: true`)
4. Defines VM in libvirt (if `libvirt_integration: true`)

## Job Status Updates

Pipeline progress is tracked in job status updates:

```json
{
  "status": "running",
  "progress": 75,
  "message": "Running hyper2kvm pipeline: CONVERT stage",
  "metadata": {
    "pipeline_stage": "convert",
    "converted_files": ["/path/to/output.qcow2"]
  }
}
```

Upon completion:

```json
{
  "status": "completed",
  "progress": 100,
  "result": {
    "output_path": "/var/lib/libvirt/images/my-vm",
    "format": "ova",
    "size": 10737418240,
    "metadata": {
      "pipeline_success": true,
      "pipeline_duration": "15m30s",
      "converted_path": "/var/lib/libvirt/images/my-vm.qcow2",
      "libvirt_domain": "my-vm",
      "libvirt_uri": "qemu:///system"
    }
  }
}
```

## UI Flow

### Standard Export (No Pipeline)

1. User fills provider configuration (vSphere, AWS, etc.)
2. User fills export options (name, output dir, format)
3. User submits job
4. VM is exported to the specified format
5. Job completes

### Pipeline-Enabled Export

1. User fills provider configuration
2. User fills export options
3. **User enables "Pipeline integration"**
4. User configures pipeline stages (default: all enabled)
5. User optionally enables "Libvirt integration"
6. User configures libvirt settings
7. User submits job
8. Export → Manifest generation → hyper2kvm pipeline → Libvirt definition
9. Job completes with converted qcow2 image and libvirt domain ready

## Visual Design

The pipeline section uses:
- **Dark background** (`#1a1a1a`) matching provider configuration sections
- **Orange accent** (`#f0583a`) for the "(hyper2kvm + libvirt)" badge
- **Collapsible layout**: Shows/hides based on `enable_pipeline` checkbox
- **Nested collapsible**: Libvirt options show/hide based on `libvirt_integration` checkbox
- **Grid layout**: 2-column grid for input fields
- **Checkbox groups**: Pipeline stages in 2x2 grid

## Error Handling

### Pipeline Failures

If hyper2kvm fails, the export job itself still succeeds (non-fatal):

```json
{
  "status": "completed_with_warnings",
  "result": {
    "output_path": "/path/to/export",
    "metadata": {
      "pipeline_success": false,
      "pipeline_error": "hyper2kvm failed: disk conversion error"
    }
  }
}
```

### Libvirt Failures

Libvirt integration failures are also non-fatal:

```json
{
  "status": "completed_with_warnings",
  "result": {
    "output_path": "/path/to/export",
    "metadata": {
      "pipeline_success": true,
      "converted_path": "/path/to/output.qcow2",
      "libvirt_error": "virsh define failed: permission denied"
    }
  }
}
```

## Provider Support

Pipeline integration currently supports:
- ✅ **vSphere**: Full support
- ⚠️ **AWS**: Manifest generation only (no pipeline)
- ⚠️ **Azure**: Manifest generation only (no pipeline)
- ⚠️ **GCP**: Manifest generation only (no pipeline)
- ⚠️ **Hyper-V**: Manifest generation only (no pipeline)

Future updates will add pipeline support for additional providers.

## Testing

### Test in Browser Console

```javascript
// Enable pipeline
document.getElementById('enable_pipeline').checked = true;

// Check pipeline stages
console.log({
  inspect: document.getElementById('pipeline_inspect').checked,
  fix: document.getElementById('pipeline_fix').checked,
  convert: document.getElementById('pipeline_convert').checked,
  validate: document.getElementById('pipeline_validate').checked
});

// Enable libvirt
document.getElementById('libvirt_integration').checked = true;

// Check libvirt settings
console.log({
  uri: document.querySelector('[name="libvirt_uri"]').value,
  bridge: document.querySelector('[name="libvirt_bridge"]').value,
  pool: document.querySelector('[name="libvirt_pool"]').value,
  autostart: document.getElementById('libvirt_autostart').checked
});
```

### Submit Test Job

```javascript
// Submit a test job with pipeline enabled
fetch('/jobs/submit', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: 'Test Pipeline Export',
    provider: 'vsphere',
    vcenter_url: 'vcenter.example.com',
    username: 'administrator@vsphere.local',
    password: 'password',
    vm_path: '/DC1/vm/test-vm',
    output_dir: '/tmp/test-export',
    format: 'ova',
    options: {
      enable_pipeline: true,
      pipeline_inspect: true,
      pipeline_fix: true,
      pipeline_convert: true,
      pipeline_validate: true,
      libvirt_integration: true,
      libvirt_uri: 'qemu:///system'
    }
  })
})
.then(r => r.json())
.then(console.log);
```

## Screenshot Locations

(Future: Add screenshots of the pipeline integration UI)

1. Pipeline section collapsed (enable_pipeline: false)
2. Pipeline section expanded (enable_pipeline: true)
3. Libvirt section collapsed (libvirt_integration: false)
4. Libvirt section expanded (libvirt_integration: true)
5. Full form with all pipeline options enabled

## See Also

- **Pipeline Integration Documentation**: `/PIPELINE_INTEGRATION.md`
- **Artifact Manifest Spec**: `/manifest/types.go`
- **Job API Documentation**: `/daemon/api/server.go`
- **Frontend Component**: `/web/dashboard-react/src/components/JobSubmissionForm.tsx`
