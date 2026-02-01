# HyperSDK Client Libraries

Official client libraries for the HyperSDK VM migration and export platform.

## Available SDKs

### 🐍 Python SDK

Full-featured Python client with type hints and async support.

**Installation:**
```bash
pip install hypersdk
```

**Quick Start:**
```python
from hypersdk import HyperSDK, JobDefinition, VCenterConfig

client = HyperSDK("http://localhost:8080")
job_id = client.submit_job(JobDefinition(
    vm_path="/Datacenter/vm/my-vm",
    output_dir="/exports",
    vcenter=VCenterConfig(
        server="vcenter.example.com",
        username="admin",
        password="password"
    )
))
```

📖 **[Full Python Documentation](python/README.md)**

---

### 📘 TypeScript/JavaScript SDK

Modern TypeScript client with full type definitions.

**Installation:**
```bash
npm install @hypersdk/client
```

**Quick Start:**
```typescript
import { HyperSDK, JobDefinition } from '@hypersdk/client';

const client = new HyperSDK('http://localhost:8080');
const jobId = await client.submitJob({
  vm_path: '/Datacenter/vm/my-vm',
  output_dir: '/exports',
  vcenter: {
    server: 'vcenter.example.com',
    username: 'admin',
    password: 'password'
  }
});
```

📖 **[Full TypeScript Documentation](typescript/README.md)**

---

## Features

Both SDKs provide complete coverage of the HyperSDK API:

### ✅ Core Features
- **Job Management** - Submit, monitor, and cancel VM export jobs
- **VM Operations** - Discover and manage VMs across multiple providers
- **Schedule Management** - Create and manage cron-based scheduled jobs
- **Webhook Integration** - Configure webhooks for job event notifications
- **Authentication** - Session-based and API key authentication

### 🚀 Advanced Features
- **Libvirt Integration** - Manage domains, snapshots, and volumes
- **Hyper2KVM Conversion** - Convert VMs to KVM/QCOW2 format
- **Real-time Progress** - Track job progress with ETA calculations
- **Batch Operations** - Submit and manage multiple jobs simultaneously
- **Error Handling** - Comprehensive error types and handling

### 🌐 Multi-Provider Support
- VMware vSphere / ESXi
- AWS EC2
- Microsoft Azure
- Google Cloud Platform (GCP)
- Microsoft Hyper-V
- Oracle Cloud Infrastructure (OCI)
- OpenStack
- Alibaba Cloud
- Proxmox VE

## Quick Comparison

| Feature | Python SDK | TypeScript SDK |
|---------|-----------|----------------|
| Type Safety | ✅ Type hints | ✅ Full TypeScript |
| Async/Await | ✅ | ✅ |
| Node.js | ❌ | ✅ |
| Browser | ❌ | ✅ |
| Python 3.8+ | ✅ | ❌ |

## API Reference

### Authentication
```python
# Python
client.login("admin", "password")

# TypeScript
await client.login("admin", "password")
```

### Submit a Job
```python
# Python
from hypersdk import JobDefinition, ExportFormat

job_id = client.submit_job(JobDefinition(
    vm_path="/Datacenter/vm/my-vm",
    output_dir="/exports",
    format=ExportFormat.OVF,
    compress=True
))

# TypeScript
import { ExportFormat } from '@hypersdk/client';

const jobId = await client.submitJob({
  vm_path: '/Datacenter/vm/my-vm',
  output_dir: '/exports',
  format: ExportFormat.OVF,
  compress: true
});
```

### Monitor Job Progress
```python
# Python
job = client.get_job(job_id)
print(f"Progress: {job.progress.percent_complete}%")

# TypeScript
const job = await client.getJob(jobId);
console.log(`Progress: ${job.progress.percent_complete}%`);
```

### Create a Schedule
```python
# Python
from hypersdk import ScheduledJob

schedule = client.create_schedule(ScheduledJob(
    name="Daily Backup",
    schedule="0 2 * * *",  # 2 AM daily
    job_template=JobDefinition(
        vm_path="/Datacenter/vm/prod-vm",
        output_dir="/backups"
    )
))

# TypeScript
const schedule = await client.createSchedule({
  name: 'Daily Backup',
  schedule: '0 2 * * *',
  job_template: {
    vm_path: '/Datacenter/vm/prod-vm',
    output_dir: '/backups'
  }
});
```

### Add a Webhook
```python
# Python
from hypersdk import Webhook

client.add_webhook(Webhook(
    url="https://example.com/webhook",
    events=["job_completed", "job_failed"]
))

# TypeScript
await client.addWebhook({
  url: 'https://example.com/webhook',
  events: ['job_completed', 'job_failed']
});
```

## Examples

### Python Examples
- [Submit Job](python/examples/submit_job.py)
- [Monitor Jobs](python/examples/monitor_jobs.py)

### TypeScript Examples
- [Submit Job](typescript/examples/submit-job.ts)
- [Monitor Jobs](typescript/examples/monitor-jobs.ts)

## OpenAPI Specification

The HyperSDK API is fully documented using OpenAPI 3.0:

📄 **[OpenAPI Specification](../openapi.yaml)**

You can use this specification to:
- Generate clients in other languages
- Explore the API with Swagger UI
- Validate API requests and responses
- Generate API documentation

## Development

### Building from Source

**Python:**
```bash
cd sdk/python
pip install -e ".[dev]"
pytest
```

**TypeScript:**
```bash
cd sdk/typescript
npm install
npm run build
npm test
```

### Code Generation

Both SDKs can be regenerated from the OpenAPI specification using standard tools:

**Python:**
```bash
openapi-generator-cli generate \
  -i openapi.yaml \
  -g python \
  -o sdk/python-generated
```

**TypeScript:**
```bash
openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-fetch \
  -o sdk/typescript-generated
```

## Contributing

We welcome contributions! To add support for a new language:

1. Review the [OpenAPI Specification](../openapi.yaml)
2. Generate a client using [OpenAPI Generator](https://openapi-generator.tech/)
3. Add language-specific improvements and tests
4. Submit a pull request

## Support

- **Issues**: https://github.com/ssahani/hypersdk/issues
- **Documentation**: https://github.com/ssahani/hypersdk
- **OpenAPI Spec**: [openapi.yaml](../openapi.yaml)

## License

LGPL-3.0-or-later

---

Made with ❤️ by the HyperSDK team
