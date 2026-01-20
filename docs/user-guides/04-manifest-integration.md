# Artifact Manifest Integration with hyper2kvm

This guide provides practical examples of integrating the Artifact Manifest v1.0 system with hyper2kvm for VM migration workflows.

## Table of Contents

1. [Overview](#overview)
2. [Manifest Generation in hypersdk](#manifest-generation-in-hypersdk)
3. [Consuming Manifests in hyper2kvm](#consuming-manifests-in-hyper2kvm)
4. [Complete Integration Workflow](#complete-integration-workflow)
5. [Advanced Examples](#advanced-examples)
6. [Troubleshooting](#troubleshooting)

---

## Overview

The **Artifact Manifest** is a versioned JSON/YAML contract between hypersdk (VM export layer) and hyper2kvm (migration orchestration layer). It provides:

- **VM Metadata**: CPU, memory, firmware, OS information
- **Disk Artifacts**: Paths, formats, checksums, boot order
- **Pipeline Configuration**: Stages to execute (INSPECT, FIX, CONVERT, VALIDATE)
- **Data Integrity**: SHA-256 checksums for disk verification
- **Auditability**: Complete source and export tracking

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      hypersdk (Go)                          │
│  ┌──────────┐     ┌──────────┐     ┌──────────────┐       │
│  │  Export  │────▶│ Manifest │────▶│ manifest.json│       │
│  │   VM     │     │ Builder  │     └──────────────┘       │
│  └──────────┘     └──────────┘                             │
└────────────────────────────┬────────────────────────────────┘
                             │
                             │ manifest.json + disk files
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                    hyper2kvm (Python)                       │
│  ┌──────────────┐   ┌────────┐   ┌────────┐   ┌──────────┐│
│  │Load Manifest │──▶│INSPECT │──▶│  FIX   │──▶│ CONVERT  ││
│  └──────────────┘   └────────┘   └────────┘   └──────────┘│
│                                                  │          │
│                                                  ▼          │
│                                         ┌────────────────┐ │
│                                         │Import to KVM   │ │
│                                         └────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

## Manifest Generation in hypersdk

### Example 1: Basic vSphere Export with Manifest

```bash
# Export VM from vSphere and generate manifest
./hyperexport \
  -vm "/Datacenter/vm/ubuntu-server" \
  -output /var/lib/hypersdk/exports/ubuntu-server \
  -manifest \
  -manifest-checksum \
  -manifest-target qcow2
```

**Generated manifest structure:**

```json
{
  "manifest_version": "1.0",
  "source": {
    "provider": "vsphere",
    "vm_id": "vm-1234",
    "vm_name": "ubuntu-server",
    "datacenter": "Datacenter",
    "export_timestamp": "2026-01-22T10:30:00Z",
    "export_method": "hypersdk-govc"
  },
  "vm": {
    "cpu": 4,
    "mem_gb": 8,
    "firmware": "uefi",
    "os_hint": "ubuntu",
    "os_version": "22.04",
    "secure_boot": false
  },
  "disks": [
    {
      "id": "disk-0",
      "source_format": "vmdk",
      "bytes": 21474836480,
      "local_path": "/var/lib/hypersdk/exports/ubuntu-server/ubuntu-server-disk-0.vmdk",
      "checksum": "sha256:a1b2c3d4e5f6...",
      "boot_order_hint": 0,
      "disk_type": "boot"
    }
  ],
  "pipeline": {
    "inspect": {
      "enabled": true,
      "detect_os": true,
      "detect_kernel": true
    },
    "fix": {
      "enabled": true,
      "inject_drivers": true,
      "fix_fstab": true,
      "fix_grub": true,
      "regenerate_initramfs": true
    },
    "convert": {
      "enabled": true,
      "target_format": "qcow2",
      "compression": false
    },
    "validate": {
      "enabled": true,
      "verify_checksum": true,
      "test_boot": false
    }
  }
}
```

### Example 2: Programmatic Manifest Generation in Go

```go
package main

import (
    "fmt"
    "github.com/hypersdk/manifest"
)

func exportWithManifest(vmPath, outputDir string) error {
    // 1. Export VM (simplified)
    diskPath := fmt.Sprintf("%s/disk-0.vmdk", outputDir)
    // ... perform export ...

    // 2. Build manifest
    builder := manifest.NewBuilder()

    // Set source metadata
    builder.WithSource(
        "vsphere",                    // provider
        "vm-5678",                    // vm_id
        "production-db",              // vm_name
        "Production-DC",              // datacenter
        "hypersdk-govc",              // export_method
    )

    // Set VM metadata
    builder.WithVM(
        8,          // cpu
        16,         // mem_gb
        "uefi",     // firmware
        "rhel",     // os_hint
        "9.3",      // os_version
        true,       // secure_boot
    )

    // Add disk with automatic checksum
    err := builder.AddDiskWithChecksum(
        "disk-0",                     // id
        "vmdk",                       // source_format
        diskPath,                     // local_path
        21474836480,                  // bytes
        0,                            // boot_order_hint
        "boot",                       // disk_type
        true,                         // compute checksum
    )
    if err != nil {
        return fmt.Errorf("failed to add disk: %w", err)
    }

    // Configure pipeline
    builder.WithPipeline(
        true,  // inspect
        true,  // fix
        true,  // convert
        true,  // validate
    )

    // Add metadata
    builder.WithMetadata("0.0.1", "job-123", []string{"production", "database"})

    // Set output configuration
    builder.WithOutput("/var/lib/libvirt/images", "qcow2")

    // Add notes
    builder.AddNote("Exported from Production vCenter")
    builder.AddNote("Target: KVM production cluster")

    // Build and validate
    m, err := builder.Build()
    if err != nil {
        return fmt.Errorf("manifest validation failed: %w", err)
    }

    // Write to file
    manifestPath := fmt.Sprintf("%s/manifest.json", outputDir)
    if err := manifest.WriteToFile(m, manifestPath); err != nil {
        return fmt.Errorf("failed to write manifest: %w", err)
    }

    fmt.Printf("Manifest written to: %s\n", manifestPath)
    return nil
}
```

### Example 3: Multi-Disk VM with Boot Order

```go
func exportMultiDiskVM() error {
    builder := manifest.NewBuilder()
    builder.WithSource("vsphere", "vm-9999", "app-server", "DC1", "hypersdk-govc")
    builder.WithVM(4, 8, "bios", "centos", "7.9", false)

    // Boot disk (SSD)
    builder.AddDiskWithChecksum(
        "boot-ssd",
        "vmdk",
        "/exports/app-server/boot.vmdk",
        53687091200,  // 50 GB
        0,            // First in boot order
        "boot",
        true,
    )

    // Data disk 1 (HDD)
    builder.AddDiskWithChecksum(
        "data-hdd-1",
        "vmdk",
        "/exports/app-server/data1.vmdk",
        214748364800,  // 200 GB
        1,             // Second in boot order (fallback)
        "data",
        true,
    )

    // Data disk 2 (HDD)
    builder.AddDiskWithChecksum(
        "data-hdd-2",
        "vmdk",
        "/exports/app-server/data2.vmdk",
        214748364800,  // 200 GB
        2,             // Third in boot order
        "data",
        true,
    )

    // Configure conversion for all disks
    builder.WithPipeline(true, true, true, true)
    builder.WithOutput("/var/lib/libvirt/images", "qcow2")

    m, _ := builder.Build()
    return manifest.WriteToFile(m, "/exports/app-server/manifest.json")
}
```

---

## Consuming Manifests in hyper2kvm

### Example 4: Python Manifest Parser

```python
# hyper2kvm/manifest.py

import json
import hashlib
from pathlib import Path
from typing import Dict, List, Optional
from dataclasses import dataclass

@dataclass
class DiskArtifact:
    """Represents a disk in the manifest."""
    id: str
    source_format: str
    bytes: int
    local_path: str
    checksum: Optional[str]
    boot_order_hint: int
    disk_type: str

@dataclass
class ArtifactManifest:
    """Artifact Manifest v1.0"""
    manifest_version: str
    source: Dict
    vm: Dict
    disks: List[DiskArtifact]
    pipeline: Dict
    metadata: Optional[Dict] = None

    @classmethod
    def load(cls, path: str) -> 'ArtifactManifest':
        """Load manifest from JSON file."""
        with open(path, 'r') as f:
            data = json.load(f)

        # Validate version
        if data.get('manifest_version') != '1.0':
            raise ValueError(f"Unsupported manifest version: {data.get('manifest_version')}")

        # Parse disks
        disks = [
            DiskArtifact(
                id=d['id'],
                source_format=d['source_format'],
                bytes=d['bytes'],
                local_path=d['local_path'],
                checksum=d.get('checksum'),
                boot_order_hint=d.get('boot_order_hint', 999),
                disk_type=d.get('disk_type', 'unknown')
            )
            for d in data.get('disks', [])
        ]

        return cls(
            manifest_version=data['manifest_version'],
            source=data.get('source', {}),
            vm=data.get('vm', {}),
            disks=disks,
            pipeline=data.get('pipeline', {}),
            metadata=data.get('metadata')
        )

    def verify_checksums(self) -> bool:
        """Verify SHA-256 checksums for all disks."""
        for disk in self.disks:
            if not disk.checksum:
                continue

            expected = disk.checksum.replace('sha256:', '')
            actual = self._compute_sha256(disk.local_path)

            if actual != expected:
                raise ValueError(
                    f"Checksum mismatch for {disk.id}: "
                    f"expected {expected}, got {actual}"
                )

        return True

    def _compute_sha256(self, path: str) -> str:
        """Compute SHA-256 checksum of file."""
        sha256 = hashlib.sha256()
        with open(path, 'rb') as f:
            for chunk in iter(lambda: f.read(8192), b''):
                sha256.update(chunk)
        return sha256.hexdigest()

    def get_boot_disk(self) -> Optional[DiskArtifact]:
        """Get the primary boot disk."""
        boot_disks = [d for d in self.disks if d.disk_type == 'boot']
        if boot_disks:
            return sorted(boot_disks, key=lambda x: x.boot_order_hint)[0]
        return None

    def get_data_disks(self) -> List[DiskArtifact]:
        """Get all data disks."""
        return [d for d in self.disks if d.disk_type == 'data']
```

### Example 5: Integration with hyper2kvm Pipeline

```python
# hyper2kvm/pipeline.py

from hyper2kvm.manifest import ArtifactManifest
from hyper2kvm.stages import InspectStage, FixStage, ConvertStage, ValidateStage

class MigrationPipeline:
    """Main migration pipeline using manifest."""

    def __init__(self, manifest_path: str):
        self.manifest = ArtifactManifest.load(manifest_path)
        self.results = {}

    def run(self):
        """Execute all enabled pipeline stages."""
        print(f"Starting migration pipeline for: {self.manifest.source.get('vm_name')}")

        # Verify checksums first
        if any(d.checksum for d in self.manifest.disks):
            print("Verifying disk checksums...")
            self.manifest.verify_checksums()
            print("✅ All checksums verified")

        # Run INSPECT stage
        if self.manifest.pipeline.get('inspect', {}).get('enabled'):
            print("\n[INSPECT] Detecting guest OS...")
            inspect = InspectStage(self.manifest)
            self.results['inspect'] = inspect.run()
            print(f"  OS: {self.results['inspect']['os_type']}")
            print(f"  Kernel: {self.results['inspect']['kernel_version']}")

        # Run FIX stage
        if self.manifest.pipeline.get('fix', {}).get('enabled'):
            print("\n[FIX] Preparing guest for KVM...")
            fix = FixStage(self.manifest, self.results.get('inspect'))
            self.results['fix'] = fix.run()
            print(f"  Drivers injected: {self.results['fix']['drivers_injected']}")
            print(f"  Boot config updated: {self.results['fix']['boot_updated']}")

        # Run CONVERT stage
        if self.manifest.pipeline.get('convert', {}).get('enabled'):
            print("\n[CONVERT] Converting disk format...")
            convert = ConvertStage(self.manifest)
            self.results['convert'] = convert.run()
            for disk_id, output_path in self.results['convert']['disks'].items():
                print(f"  {disk_id}: {output_path}")

        # Run VALIDATE stage
        if self.manifest.pipeline.get('validate', {}).get('enabled'):
            print("\n[VALIDATE] Validating converted images...")
            validate = ValidateStage(self.manifest, self.results.get('convert'))
            self.results['validate'] = validate.run()
            print(f"  All images valid: {self.results['validate']['all_valid']}")

        print("\n✅ Pipeline complete!")
        return self.results

# Usage
if __name__ == '__main__':
    pipeline = MigrationPipeline('/var/lib/hypersdk/exports/ubuntu-server/manifest.json')
    results = pipeline.run()
```

### Example 6: INSPECT Stage Implementation

```python
# hyper2kvm/stages/inspect.py

import subprocess
from pathlib import Path
from hyper2kvm.manifest import ArtifactManifest

class InspectStage:
    """Detect guest OS and configuration."""

    def __init__(self, manifest: ArtifactManifest):
        self.manifest = manifest

    def run(self) -> dict:
        """Execute inspection."""
        boot_disk = self.manifest.get_boot_disk()
        if not boot_disk:
            raise ValueError("No boot disk found in manifest")

        results = {
            'disk_id': boot_disk.id,
            'disk_path': boot_disk.local_path,
            'disk_format': boot_disk.source_format,
        }

        # Use manifest hints if available
        if self.manifest.vm.get('os_hint'):
            results['os_type'] = self.manifest.vm['os_hint']
            results['os_version'] = self.manifest.vm.get('os_version', 'unknown')
        else:
            # Fallback to virt-inspector
            results.update(self._run_virt_inspector(boot_disk.local_path))

        # Detect kernel
        results['kernel_version'] = self._detect_kernel(boot_disk.local_path)

        # Check firmware type
        results['firmware'] = self.manifest.vm.get('firmware', 'bios')
        results['secure_boot'] = self.manifest.vm.get('secure_boot', False)

        return results

    def _run_virt_inspector(self, disk_path: str) -> dict:
        """Use libguestfs virt-inspector."""
        cmd = ['virt-inspector', '--add', disk_path]
        result = subprocess.run(cmd, capture_output=True, text=True)
        # Parse XML output...
        return {
            'os_type': 'linux',  # Simplified
            'os_version': 'detected'
        }

    def _detect_kernel(self, disk_path: str) -> str:
        """Detect kernel version using guestfish."""
        # Simplified example
        return '5.15.0-generic'
```

### Example 7: CONVERT Stage Implementation

```python
# hyper2kvm/stages/convert.py

import subprocess
from pathlib import Path
from hyper2kvm.manifest import ArtifactManifest

class ConvertStage:
    """Convert disk formats using qemu-img."""

    def __init__(self, manifest: ArtifactManifest):
        self.manifest = manifest

    def run(self) -> dict:
        """Convert all disks to target format."""
        target_format = self.manifest.pipeline.get('convert', {}).get('target_format', 'qcow2')
        output_dir = self.manifest.pipeline.get('convert', {}).get('output_dir', '/tmp')

        converted_disks = {}

        for disk in self.manifest.disks:
            print(f"  Converting {disk.id}...")

            # Build output path
            output_path = Path(output_dir) / f"{disk.id}.{target_format}"

            # Skip if already in target format and same location
            if disk.source_format == target_format:
                print(f"    Already in {target_format} format, copying...")
                subprocess.run(['cp', disk.local_path, str(output_path)], check=True)
            else:
                # Convert using qemu-img
                cmd = [
                    'qemu-img', 'convert',
                    '-f', disk.source_format,
                    '-O', target_format,
                    '-p',  # Show progress
                    disk.local_path,
                    str(output_path)
                ]

                subprocess.run(cmd, check=True)

            converted_disks[disk.id] = str(output_path)
            print(f"    ✅ {output_path}")

        return {
            'disks': converted_disks,
            'format': target_format
        }
```

---

## Complete Integration Workflow

### Example 8: End-to-End Migration Script

```python
#!/usr/bin/env python3
# migrate_vm.py

import sys
import argparse
from pathlib import Path
from hyper2kvm.manifest import ArtifactManifest
from hyper2kvm.pipeline import MigrationPipeline
from hyper2kvm.kvm_importer import KVMImporter

def migrate_from_manifest(manifest_path: str, dry_run: bool = False):
    """Complete migration workflow using manifest."""

    # 1. Load and validate manifest
    print("Loading manifest...")
    manifest = ArtifactManifest.load(manifest_path)

    print(f"VM: {manifest.source.get('vm_name')}")
    print(f"Source: {manifest.source.get('provider')}")
    print(f"Disks: {len(manifest.disks)}")
    print(f"CPU: {manifest.vm.get('cpu')}, RAM: {manifest.vm.get('mem_gb')} GB")
    print()

    # 2. Run pipeline
    pipeline = MigrationPipeline(manifest_path)
    results = pipeline.run()

    if dry_run:
        print("\n[DRY RUN] Stopping before KVM import")
        return

    # 3. Import to KVM
    print("\n[IMPORT] Creating KVM domain...")
    importer = KVMImporter()

    domain_name = manifest.source.get('vm_name', 'imported-vm')

    domain = importer.create_domain(
        name=domain_name,
        cpu=manifest.vm.get('cpu', 2),
        memory_gb=manifest.vm.get('mem_gb', 4),
        firmware=manifest.vm.get('firmware', 'bios'),
        disks=results['convert']['disks'],
        boot_disk_id=manifest.get_boot_disk().id
    )

    print(f"✅ Domain created: {domain_name}")
    print(f"   UUID: {domain.UUIDString()}")

    # 4. Generate migration report
    report_path = Path(manifest_path).parent / 'migration-report.json'
    importer.write_report(results, report_path)
    print(f"📄 Report: {report_path}")

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Migrate VM using Artifact Manifest')
    parser.add_argument('manifest', help='Path to manifest.json')
    parser.add_argument('--dry-run', action='store_true', help='Run without importing to KVM')

    args = parser.parse_args()

    try:
        migrate_from_manifest(args.manifest, args.dry_run)
    except Exception as e:
        print(f"❌ Migration failed: {e}", file=sys.stderr)
        sys.exit(1)
```

**Usage:**

```bash
# Export from vSphere with manifest
cd ~/hypersdk
./hyperexport \
  -vm "/DC/vm/web-server" \
  -output /tmp/export \
  -manifest \
  -manifest-checksum

# Migrate using manifest
cd ~/hyper2kvm
python migrate_vm.py /tmp/export/manifest.json

# Dry run (inspect only)
python migrate_vm.py /tmp/export/manifest.json --dry-run
```

---

## Advanced Examples

### Example 9: Batch Migration with Manifests

```python
# batch_migrate.py

from pathlib import Path
from concurrent.futures import ThreadPoolExecutor, as_completed
from hyper2kvm.manifest import ArtifactManifest
from hyper2kvm.pipeline import MigrationPipeline

def migrate_one(manifest_path: Path) -> dict:
    """Migrate single VM."""
    try:
        manifest = ArtifactManifest.load(str(manifest_path))
        pipeline = MigrationPipeline(str(manifest_path))
        results = pipeline.run()

        return {
            'vm_name': manifest.source.get('vm_name'),
            'status': 'success',
            'results': results
        }
    except Exception as e:
        return {
            'vm_name': str(manifest_path.parent.name),
            'status': 'failed',
            'error': str(e)
        }

def batch_migrate(export_dir: str, max_workers: int = 4):
    """Migrate all VMs with manifests in export directory."""
    export_path = Path(export_dir)
    manifests = list(export_path.glob('*/manifest.json'))

    print(f"Found {len(manifests)} VMs to migrate")

    with ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {
            executor.submit(migrate_one, m): m
            for m in manifests
        }

        results = []
        for future in as_completed(futures):
            manifest_path = futures[future]
            result = future.result()
            results.append(result)

            status_icon = '✅' if result['status'] == 'success' else '❌'
            print(f"{status_icon} {result['vm_name']}: {result['status']}")

    # Summary
    success = sum(1 for r in results if r['status'] == 'success')
    failed = len(results) - success
    print(f"\nSummary: {success} succeeded, {failed} failed")

    return results

if __name__ == '__main__':
    batch_migrate('/var/lib/hypersdk/exports', max_workers=4)
```

### Example 10: Manifest Validation Tool

```python
# validate_manifest.py

import sys
from pathlib import Path
from hyper2kvm.manifest import ArtifactManifest

def validate_manifest(manifest_path: str, strict: bool = False) -> bool:
    """Validate manifest and check disk files."""
    errors = []
    warnings = []

    try:
        manifest = ArtifactManifest.load(manifest_path)
    except Exception as e:
        print(f"❌ Failed to load manifest: {e}")
        return False

    # Check version
    if manifest.manifest_version != '1.0':
        errors.append(f"Unsupported version: {manifest.manifest_version}")

    # Check required fields
    if not manifest.disks:
        errors.append("No disks defined")

    # Validate disk files
    for disk in manifest.disks:
        disk_path = Path(disk.local_path)

        # Check existence
        if not disk_path.exists():
            errors.append(f"Disk file not found: {disk.local_path}")
            continue

        # Check size
        actual_size = disk_path.stat().st_size
        if actual_size != disk.bytes:
            warnings.append(
                f"{disk.id}: Size mismatch "
                f"(manifest: {disk.bytes}, actual: {actual_size})"
            )

        # Verify checksum
        if disk.checksum and strict:
            try:
                manifest._compute_sha256(disk.local_path)
                print(f"✅ {disk.id}: Checksum verified")
            except Exception as e:
                errors.append(f"{disk.id}: Checksum verification failed: {e}")

    # Print results
    if errors:
        print("\n❌ ERRORS:")
        for err in errors:
            print(f"  - {err}")

    if warnings:
        print("\n⚠️  WARNINGS:")
        for warn in warnings:
            print(f"  - {warn}")

    if not errors and not warnings:
        print("✅ Manifest is valid")

    return len(errors) == 0

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: validate_manifest.py <manifest.json> [--strict]")
        sys.exit(1)

    strict = '--strict' in sys.argv
    valid = validate_manifest(sys.argv[1], strict=strict)
    sys.exit(0 if valid else 1)
```

### Example 11: Custom Pipeline Configuration

```python
# custom_pipeline.py

from hyper2kvm.manifest import ArtifactManifest
from hyper2kvm.stages import InspectStage, ConvertStage

def selective_migration(manifest_path: str, skip_fix: bool = False):
    """Run custom pipeline with selective stages."""
    manifest = ArtifactManifest.load(manifest_path)

    # Always inspect
    inspect = InspectStage(manifest)
    os_info = inspect.run()

    print(f"Detected: {os_info['os_type']} {os_info['os_version']}")

    # Conditional FIX based on OS type
    if os_info['os_type'] in ['ubuntu', 'debian'] and not skip_fix:
        from hyper2kvm.stages import DebianFixStage
        fix = DebianFixStage(manifest, os_info)
        fix.run()
    elif os_info['os_type'] in ['rhel', 'centos', 'fedora'] and not skip_fix:
        from hyper2kvm.stages import RHELFixStage
        fix = RHELFixStage(manifest, os_info)
        fix.run()

    # Always convert
    convert = ConvertStage(manifest)
    result = convert.run()

    print(f"Converted disks: {list(result['disks'].keys())}")
    return result
```

---

## Troubleshooting

### Common Issues

#### Issue 1: Checksum Mismatch

**Error:**
```
ValueError: Checksum mismatch for disk-0: expected a1b2c3..., got d4e5f6...
```

**Solution:**
```python
# Recompute checksums in manifest
import hashlib
from hyper2kvm.manifest import ArtifactManifest

manifest = ArtifactManifest.load('manifest.json')
for disk in manifest.disks:
    actual = manifest._compute_sha256(disk.local_path)
    print(f"{disk.id}: sha256:{actual}")
    # Update manifest if needed
```

#### Issue 2: Missing Disk Files

**Error:**
```
FileNotFoundError: /var/lib/hypersdk/exports/vm/disk-0.vmdk
```

**Solution:**
```python
# Update disk paths in manifest
import json

with open('manifest.json', 'r') as f:
    data = json.load(f)

# Fix paths
for disk in data['disks']:
    old_path = disk['local_path']
    disk['local_path'] = old_path.replace('/old/path', '/new/path')

with open('manifest.json', 'w') as f:
    json.dump(data, f, indent=2)
```

#### Issue 3: Unsupported Source Format

**Error:**
```
ValueError: Unsupported source_format: vhdx
```

**Solution:**
```bash
# Convert VHDX to QCOW2 manually first
qemu-img convert -f vhdx -O qcow2 disk.vhdx disk.qcow2

# Update manifest
python << EOF
import json
with open('manifest.json', 'r+') as f:
    data = json.load(f)
    data['disks'][0]['source_format'] = 'qcow2'
    data['disks'][0]['local_path'] = 'disk.qcow2'
    f.seek(0)
    json.dump(data, f, indent=2)
    f.truncate()
EOF
```

---

## Best Practices

1. **Always Enable Checksums**: Use `-manifest-checksum` flag for data integrity
2. **Verify Before Migration**: Run `validate_manifest.py` before processing
3. **Preserve Manifests**: Keep manifests for audit trail and troubleshooting
4. **Use Boot Order Hints**: Properly set `boot_order_hint` for multi-disk VMs
5. **Tag Migrations**: Use `metadata.tags` for categorization and tracking
6. **Handle Errors Gracefully**: Implement retry logic for transient failures
7. **Test Dry Runs**: Use `--dry-run` to validate before actual import

---

## References

- [Artifact Manifest Specification](../../manifest/README.md)
- [hyper2kvm Integration Guide](./03-integration.md)
- [VM Export Guide](./02-vm-export-guide.md)
- [hypersdk API Reference](../api/01-daemon-api.md)

---

**Part of the hypersdk project family**
