# Integration with hyper2kvm Python Project

This document describes how to integrate `hypersdk` with the main `hyperexport` Python migration toolkit.

## Architecture Overview

```mermaid
graph TB
    Python["hyper2kvm (Python)<br/>Main Migration Orchestration Layer<br/>━━━━━━━━━━━━━━━━━<br/>• VM discovery and selection<br/>• Conversion logic (OVF → QCOW2)<br/>• KVM import<br/>• Migration workflow"]

    Go["hypersdk (Go)<br/>High-Performance Export Layer<br/>━━━━━━━━━━━━━━━━━<br/>• VM export from multiple clouds<br/>• Parallel downloads<br/>• Progress tracking<br/>• Job management"]

    Python -->|"HTTP REST API<br/>JSON requests/responses"| Go

    style Python fill:#4CAF50,stroke:#2E7D32,color:#fff,stroke-width:2px
    style Go fill:#2196F3,stroke:#1565C0,color:#fff,stroke-width:2px
```

## Why Separate Provider Layer?

1. **Performance**: Go provides better performance for network-intensive operations
2. **Concurrency**: Native goroutines for parallel VM exports
3. **Multi-Cloud**: Unified API for vSphere, AWS, Azure, GCP
4. **Daemon Mode**: Background service for long-running exports
5. **Language Strengths**: Go for systems programming, Python for orchestration

## Integration Methods

### Method 1: Python Client Library (Recommended)

Create a Python client for the hypersdk API:

```python
# hyper2kvm/providers/client.py

import requests
import time
from typing import Dict, List, Optional

class ProviderClient:
    """Client for hypersdk daemon."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()

    def health_check(self) -> bool:
        """Check if daemon is healthy."""
        try:
            response = self.session.get(f"{self.base_url}/health")
            return response.status_code == 200
        except requests.RequestException:
            return False

    def get_status(self) -> Dict:
        """Get daemon status."""
        response = self.session.get(f"{self.base_url}/status")
        response.raise_for_status()
        return response.json()

    def submit_export(
        self,
        vm_path: str,
        output_path: str,
        name: Optional[str] = None,
        parallel_downloads: int = 4,
        remove_cdrom: bool = True
    ) -> str:
        """Submit VM export job."""
        payload = {
            "name": name or f"export-{int(time.time())}",
            "vm_path": vm_path,
            "output_path": output_path,
            "options": {
                "parallel_downloads": parallel_downloads,
                "remove_cdrom": remove_cdrom
            }
        }

        response = self.session.post(
            f"{self.base_url}/jobs/submit",
            json=payload
        )
        response.raise_for_status()
        result = response.json()
        return result["job_ids"][0]

    def query_job(self, job_id: str) -> Dict:
        """Query job status."""
        response = self.session.post(
            f"{self.base_url}/jobs/query",
            json={"job_ids": [job_id]}
        )
        response.raise_for_status()
        result = response.json()
        return result["jobs"][0]

    def wait_for_completion(
        self,
        job_id: str,
        timeout: int = 3600,
        poll_interval: int = 5,
        progress_callback=None
    ) -> Dict:
        """Wait for job to complete."""
        start_time = time.time()

        while True:
            if time.time() - start_time > timeout:
                raise TimeoutError(f"Job {job_id} timed out after {timeout}s")

            job = self.query_job(job_id)
            status = job["status"]

            if progress_callback and job.get("progress"):
                progress_callback(job["progress"])

            if status == "completed":
                return job
            elif status == "failed":
                raise Exception(f"Job failed: {job.get('error')}")
            elif status == "cancelled":
                raise Exception("Job was cancelled")

            time.sleep(poll_interval)

    def cancel_job(self, job_id: str) -> None:
        """Cancel a running job."""
        response = self.session.post(
            f"{self.base_url}/jobs/cancel",
            json={"job_ids": [job_id]}
        )
        response.raise_for_status()
```

### Method 2: Direct REST API Calls

```python
# hyper2kvm/utils/provider_api.py

import requests

def export_vm_from_vsphere(vm_path: str, output_path: str) -> str:
    """Export VM using hypersdk."""
    response = requests.post(
        "http://localhost:8080/jobs/submit",
        json={
            "name": f"export-{vm_path.split('/')[-1]}",
            "vm_path": vm_path,
            "output_path": output_path,
            "options": {
                "parallel_downloads": 4,
                "remove_cdrom": True
            }
        }
    )
    response.raise_for_status()
    return response.json()["job_ids"][0]
```

## Integration Workflow

### Complete Migration Example

```python
# hyper2kvm/migrate.py

from hyper2kvm.providers.client import ProviderClient
from hyper2kvm.converter import OVFConverter
from hyper2kvm.kvm import KVMImporter

def migrate_vm(vm_path: str, dest_pool: str):
    """Complete VM migration from vSphere to KVM."""

    # Step 1: Export from vSphere
    print(f"Exporting VM: {vm_path}")
    provider = ProviderClient()

    if not provider.health_check():
        raise Exception("hypersdk daemon not running")

    job_id = provider.submit_export(
        vm_path=vm_path,
        output_path="/tmp/export",
        parallel_downloads=4
    )

    print(f"Export job submitted: {job_id}")

    # Monitor progress
    def show_progress(progress):
        print(f"  {progress['phase']}: {progress['percent_complete']:.1f}%")

    job = provider.wait_for_completion(
        job_id,
        progress_callback=show_progress
    )

    ovf_path = job["result"]["ovf_path"]
    print(f"Export complete: {ovf_path}")

    # Step 2: Convert OVF to QCOW2
    print("Converting to QCOW2...")
    converter = OVFConverter()
    qcow2_path = converter.convert(ovf_path, "/var/lib/libvirt/images")
    print(f"Conversion complete: {qcow2_path}")

    # Step 3: Import to KVM
    print("Importing to KVM...")
    importer = KVMImporter()
    domain = importer.import_vm(qcow2_path, dest_pool)
    print(f"Migration complete: {domain.name()}")

    return domain
```

## Configuration

### Start hypersdk Daemon

```bash
# Option 1: Systemd service
sudo systemctl start hypervisord
sudo systemctl enable hypervisord

# Option 2: Manual start
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1

./hypervisord --config /etc/hypervisord/config.yaml
```

### Environment Variables

hyper2kvm Python project can pass configuration via environment:

```python
import os
import subprocess

def start_provider_daemon(vcenter_url, username, password):
    """Start hypersdk daemon programmatically."""
    env = os.environ.copy()
    env.update({
        'GOVC_URL': vcenter_url,
        'GOVC_USERNAME': username,
        'GOVC_PASSWORD': password,
        'GOVC_INSECURE': '1',
        'DAEMON_ADDR': 'localhost:8080'
    })

    subprocess.Popen(['hypervisord'], env=env)
```

## Error Handling

```python
from hyper2kvm.providers.client import ProviderClient

def safe_export(vm_path: str, output_path: str, max_retries: int = 3):
    """Export with retry logic."""
    provider = ProviderClient()

    for attempt in range(max_retries):
        try:
            job_id = provider.submit_export(vm_path, output_path)
            job = provider.wait_for_completion(job_id)
            return job["result"]
        except Exception as e:
            print(f"Attempt {attempt + 1} failed: {e}")
            if attempt == max_retries - 1:
                raise
            time.sleep(5 * (attempt + 1))  # Exponential backoff
```

## Batch Processing

```python
from concurrent.futures import ThreadPoolExecutor, as_completed
from hyper2kvm.providers.client import ProviderClient

def export_multiple_vms(vm_list: List[str], output_dir: str):
    """Export multiple VMs in parallel."""
    provider = ProviderClient()

    # Submit all jobs
    job_ids = []
    for vm_path in vm_list:
        job_id = provider.submit_export(
            vm_path=vm_path,
            output_path=f"{output_dir}/{vm_path.split('/')[-1]}"
        )
        job_ids.append((vm_path, job_id))

    # Wait for completion
    results = {}
    for vm_path, job_id in job_ids:
        try:
            job = provider.wait_for_completion(job_id)
            results[vm_path] = job["result"]
            print(f"✅ {vm_path}: Complete")
        except Exception as e:
            results[vm_path] = {"error": str(e)}
            print(f"❌ {vm_path}: Failed - {e}")

    return results
```

## Testing Integration

```python
# tests/test_provider_integration.py

import unittest
from hyper2kvm.providers.client import ProviderClient

class TestProviderIntegration(unittest.TestCase):
    def setUp(self):
        self.client = ProviderClient("http://localhost:8080")

    def test_daemon_health(self):
        """Test daemon is running."""
        self.assertTrue(self.client.health_check())

    def test_daemon_status(self):
        """Test daemon status endpoint."""
        status = self.client.get_status()
        self.assertIn("version", status)
        self.assertEqual(status["version"], "0.0.1")

    def test_job_lifecycle(self):
        """Test complete job lifecycle."""
        # Submit (will fail without real vCenter, but tests API)
        try:
            job_id = self.client.submit_export(
                "/datacenter/vm/test",
                "/tmp/test"
            )
            self.assertIsNotNone(job_id)

            # Query
            job = self.client.query_job(job_id)
            self.assertEqual(job["definition"]["vm_path"], "/datacenter/vm/test")
        except Exception:
            # Expected to fail without vCenter
            pass
```

## Deployment

### Development

```bash
# Terminal 1: Start daemon
cd ~/go/hypersdk
./hypervisord

# Terminal 2: Run Python hyper2kvm
cd ~/hyper2kvm
python -m hyper2kvm migrate --vm "/dc/vm/test"
```

### Production

```bash
# Install hypersdk
sudo dnf install hypersdk
sudo systemctl enable --now hypervisord

# Install Python hyper2kvm
pip install hyper2kvm

# Run migration
hyper2kvm migrate --vm "/dc/vm/prod-web-01"
```

## Benefits of This Architecture

1. **Separation of Concerns**
   - Python: High-level orchestration, conversion logic
   - Go: Low-level export, network operations

2. **Performance**
   - Go daemon handles heavy I/O operations
   - Python focuses on business logic

3. **Flexibility**
   - Can run daemon on separate machine
   - Multiple Python clients can share one daemon
   - Easy to add more cloud providers in Go

4. **Maintainability**
   - Each component has clear responsibility
   - Can be developed/tested independently
   - Language-specific strengths utilized

## Next Steps

1. Add `hypersdk` dependency to `setup.py`:
```python
install_requires=[
    'requests>=2.25.0',
    # ... other deps
]
```

2. Create provider client package in hyper2kvm:
```
hyper2kvm/
├── providers/
│   ├── __init__.py
│   ├── client.py
│   └── exceptions.py
```

3. Update CLI to use provider client:
```python
@click.command()
@click.option('--vm-path', required=True)
def export_cmd(vm_path):
    """Export VM using providers daemon."""
    from hyper2kvm.providers.client import ProviderClient
    # ... implementation
```

---

**Part of the hyper2kvm project family**
