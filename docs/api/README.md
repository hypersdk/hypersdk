# API Documentation

This directory contains comprehensive API documentation for HyperSDK.

## 🚀 Quick Reference

**Total Endpoints:** 51+ REST endpoints
**API Version:** v1
**Base URL:** `http://localhost:8080`
**Format:** JSON
**Test Coverage:** 40.8% (daemon/api package)

## API References

1. **[API Overview](00-overview.md)** - Introduction to HyperSDK APIs and architecture
2. **[Daemon API](01-daemon-api.md)** - REST API reference for the hypervisord daemon
3. **[API Endpoints](02-endpoints.md)** - Complete endpoint reference with examples
4. **[New Features](03-new-features.md)** - Recently added API features and capabilities

## API Categories

### Job Management
- Submit VM export jobs
- Query job status and progress
- Cancel running jobs
- Job scheduling with cron expressions
- Batch job operations

### VM Discovery & Export
- List VMs from vCenter
- Export VM details and metadata
- Download VM files (VMDK, OVF)
- Verify exports with checksums

### Libvirt Integration
- Domain lifecycle (start, stop, pause, resume, reboot, destroy)
- Snapshot management (create, list, revert, delete)
- Storage pools and volumes
- Network management

### Console Access
- VNC console proxy
- Serial console access
- VM screenshots
- Send keys to VM

### Cloud Provider Integrations
- vSphere (VMware vCenter/ESXi)
- AWS (Amazon EC2)
- Azure (Microsoft Azure VMs)
- GCP (Google Compute Engine)
- Hyper-V
- OCI (Oracle Cloud)
- OpenStack
- Alibaba Cloud
- Proxmox VE

### Monitoring & Analytics
- Job progress tracking with ETAs
- Real-time metrics (CPU, memory, goroutines)
- Cost tracking and budgets
- Historical data and trends
- WebSocket live updates

### Additional Features
- ISO management (upload, attach, detach)
- VM cloning and templates
- Configuration generation
- Backup and restore
- Hyper-V to KVM conversion
- Manifest-based workflows

## Authentication

Currently, the API endpoints are accessible without authentication. For production deployments:

```bash
# Future: Authentication will be required
curl -H "Authorization: Bearer <token>" \
  http://localhost:8080/api/v1/jobs
```

See [Security Best Practices](../security-best-practices.md) for credential management.

## Quick Start

### 1. Start the Daemon
```bash
# Using binary
./hypervisord

# Using systemd
sudo systemctl start hypervisord

# Check it's running
curl http://localhost:8080/api/v1/status
```

### 2. Submit a Job
```bash
# Submit single VM export
curl -X POST http://localhost:8080/api/v1/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/tmp/export"
  }'
```

### 3. Monitor Progress
```bash
# Get job status
curl http://localhost:8080/api/v1/jobs/{job_id}

# List all jobs
curl http://localhost:8080/api/v1/jobs

# Watch job progress with ETA
curl http://localhost:8080/api/v1/jobs/{job_id}/progress
```

### 4. Access Web Dashboard
```bash
# Open dashboard in browser
xdg-open http://localhost:8080/web/dashboard/

# View VM console
xdg-open http://localhost:8080/web/dashboard/vm-console.html
```

## Common Operations

### List Available VMs
```bash
curl http://localhost:8080/api/v1/vms/discover
```

### Query Jobs by Status
```bash
# Get running jobs
curl http://localhost:8080/api/v1/jobs/query?status=running

# Get failed jobs
curl http://localhost:8080/api/v1/jobs/query?status=failed
```

### Create Scheduled Export
```bash
curl -X POST http://localhost:8080/api/v1/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "name": "daily-backup",
    "cron": "0 2 * * *",
    "job_definition": {
      "vm_path": "/datacenter/vm/prod/web01",
      "output_path": "/backup/daily"
    }
  }'
```

### Manage Libvirt VMs
```bash
# List domains
curl http://localhost:8080/api/v1/libvirt/domains

# Start a VM
curl -X POST http://localhost:8080/api/v1/libvirt/domain/start \
  -d '{"name": "my-vm"}'

# Create snapshot
curl -X POST http://localhost:8080/api/v1/libvirt/snapshot/create \
  -d '{
    "domain_name": "my-vm",
    "snapshot_name": "backup-2026-01-26",
    "description": "Pre-update backup"
  }'
```

## WebSocket Support

Real-time updates via WebSocket:

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws/metrics');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Metrics update:', data);
};
```

## Error Handling

All API errors return JSON with standard format:

```json
{
  "error": "job not found",
  "status": 404,
  "timestamp": "2026-01-26T10:30:00Z"
}
```

Common HTTP status codes:
- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid input
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

## Rate Limiting

API endpoints implement rate limiting:
- Default: 100 requests per minute per IP
- Burst: 200 requests
- Headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`

## Testing the API

### Run API Tests
```bash
# Run all API handler tests
go test -v ./daemon/api

# Test specific handlers
go test -v ./daemon/api -run TestHandle.*Libvirt

# With coverage
go test -coverprofile=coverage.out ./daemon/api
```

### Use Test Script
```bash
# Test 79+ endpoints
./scripts/test-api.sh

# Test dashboard endpoints
./test-dashboard-endpoints.sh
```

## API Versioning

**Current Version:** v1

HyperSDK uses semantic versioning:
- **v1.x.x** - Current stable API
- **v2.x.x** - Future major version (breaking changes)

Breaking changes will result in a new API version while maintaining backward compatibility for existing versions.

## OpenAPI Specification

OpenAPI/Swagger spec available at:
```
http://localhost:8080/openapi.yaml
http://localhost:8080/swagger-ui/
```

## Related Documentation

- **[Getting Started](../getting-started.md)** - Setup and initial configuration
- **[Test Results](../test-results.md)** - API test coverage and results
- **[Integration Guides](../integration/)** - Integrating with external systems
- **[Examples](../../examples/)** - API usage examples
- **[Dashboard README](../../daemon/dashboard/README.md)** - Web dashboard documentation

## Support

- **GitHub Issues:** Report bugs and request features
- **Test Scripts:** Use `scripts/test-api.sh` for endpoint validation
- **Documentation:** Check API endpoint documentation for detailed examples

## License

API documentation is licensed under LGPL-3.0-or-later.
