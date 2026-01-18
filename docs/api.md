# HyperSDK REST API Documentation

## Overview

The HyperSDK REST API provides programmatic access to manage VM migration jobs, schedules, webhooks, and more. All endpoints return JSON responses and use standard HTTP methods and status codes.

**Base URL**: `http://localhost:8080` (configurable)

**Authentication**: Currently no authentication (add in production)

## Table of Contents

- [Health & Status](#health--status)
- [Jobs API](#jobs-api)
- [Schedules API](#schedules-api)
- [Webhooks API](#webhooks-api)
- [VM Discovery API](#vm-discovery-api)
- [WebSocket API](#websocket-api)
- [Metrics API](#metrics-api)

---

## Health & Status

### GET /health

Health check endpoint.

**Response**:
```json
{
  "status": "ok"
}
```

### GET /status

Get current daemon status and statistics.

**Response**:
```json
{
  "total_jobs": 42,
  "running_jobs": 3,
  "completed_jobs": 35,
  "failed_jobs": 4,
  "worker_count": 10,
  "active_workers": 3
}
```

---

## Jobs API

### POST /jobs/submit

Submit a new migration job.

**Request Body**:
```json
{
  "vm_path": "Datacenter/vm/production/web-server-1",
  "source": {
    "type": "vsphere",
    "vcenter_url": "vcenter.example.com",
    "username": "admin@vsphere.local",
    "password": "password",
    "datacenter": "Datacenter"
  },
  "destination": {
    "type": "kvm",
    "libvirt_uri": "qemu+ssh://user@host/system"
  },
  "options": {
    "conversion_method": "vddk",
    "network_mapping": {
      "VM Network": "br0"
    }
  }
}
```

**Response**:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "job submitted successfully"
}
```

**Status Codes**:
- `201 Created` - Job submitted successfully
- `400 Bad Request` - Invalid request body
- `500 Internal Server Error` - Failed to submit job

### POST /jobs/query

Query jobs with filters.

**Request Body**:
```json
{
  "all": true,
  "status": "running",
  "limit": 10
}
```

**Response**:
```json
{
  "jobs": [
    {
      "definition": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "vm_path": "Datacenter/vm/production/web-server-1"
      },
      "status": "running",
      "progress": {
        "stage": "conversion",
        "percent_complete": 45,
        "current_bytes": 21474836480,
        "total_bytes": 53687091200
      },
      "started_at": "2026-01-17T10:30:00Z",
      "error": null
    }
  ],
  "total": 1
}
```

**Status Codes**:
- `200 OK` - Query successful
- `400 Bad Request` - Invalid query parameters

### GET /jobs/{id}

Get specific job details.

**URL Parameters**:
- `id` - Job ID (UUID)

**Response**:
```json
{
  "definition": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "vm_path": "Datacenter/vm/production/web-server-1"
  },
  "status": "running",
  "progress": {
    "stage": "conversion",
    "percent_complete": 45
  }
}
```

**Status Codes**:
- `200 OK` - Job found
- `404 Not Found` - Job not found

### POST /jobs/cancel

Cancel jobs by ID or status.

**Request Body**:
```json
{
  "job_ids": ["550e8400-e29b-41d4-a716-446655440000"],
  "status": "running"
}
```

**Response**:
```json
{
  "message": "jobs cancelled successfully",
  "count": 1
}
```

**Status Codes**:
- `200 OK` - Jobs cancelled
- `400 Bad Request` - Invalid request

---

## Schedules API

### GET /schedules

List all scheduled jobs.

**Response**:
```json
{
  "schedules": [
    {
      "id": "daily-backup",
      "name": "Daily VM Backup",
      "schedule": "0 2 * * *",
      "enabled": true,
      "job_template": {
        "vm_path": "Datacenter/vm/production/*",
        "source": {...},
        "destination": {...}
      },
      "last_run": "2026-01-17T02:00:00Z",
      "next_run": "2026-01-18T02:00:00Z",
      "run_count": 30
    }
  ],
  "total": 1,
  "timestamp": "2026-01-17T12:00:00Z"
}
```

### POST /schedules

Create a new scheduled job.

**Request Body**:
```json
{
  "id": "daily-backup",
  "name": "Daily VM Backup",
  "description": "Backup production VMs every night",
  "schedule": "0 2 * * *",
  "enabled": true,
  "job_template": {
    "vm_path": "Datacenter/vm/production/web-server-1",
    "source": {...},
    "destination": {...}
  }
}
```

**Schedule Format**: Cron expression (minute hour day month weekday)
- `0 2 * * *` - Every day at 2:00 AM
- `*/15 * * * *` - Every 15 minutes
- `0 0 * * 0` - Every Sunday at midnight

**Response**:
```json
{
  "message": "schedule created successfully",
  "schedule": {...}
}
```

**Status Codes**:
- `201 Created` - Schedule created
- `400 Bad Request` - Invalid schedule
- `503 Service Unavailable` - Scheduler not enabled

### GET /schedules/{id}

Get specific schedule.

**Response**:
```json
{
  "id": "daily-backup",
  "name": "Daily VM Backup",
  "schedule": "0 2 * * *",
  "enabled": true,
  "last_run": "2026-01-17T02:00:00Z",
  "next_run": "2026-01-18T02:00:00Z"
}
```

### PUT /schedules/{id}

Update schedule.

**Request Body**:
```json
{
  "name": "Updated Name",
  "schedule": "0 3 * * *",
  "enabled": true
}
```

**Response**:
```json
{
  "message": "schedule updated successfully"
}
```

### DELETE /schedules/{id}

Delete schedule.

**Response**:
```json
{
  "message": "schedule deleted successfully"
}
```

### POST /schedules/{id}/enable

Enable schedule.

**Response**:
```json
{
  "message": "schedule enabled successfully"
}
```

### POST /schedules/{id}/disable

Disable schedule.

**Response**:
```json
{
  "message": "schedule disabled successfully"
}
```

### POST /schedules/{id}/trigger

Manually trigger schedule execution.

**Response**:
```json
{
  "message": "schedule triggered successfully"
}
```

### GET /schedules/stats

Get schedule statistics.

**Response**:
```json
{
  "total_schedules": 5,
  "enabled_schedules": 3,
  "disabled_schedules": 2,
  "total_runs": 150,
  "successful_runs": 145,
  "failed_runs": 5
}
```

---

## Webhooks API

### GET /webhooks

List all configured webhooks.

**Response**:
```json
{
  "webhooks": [
    {
      "url": "https://hooks.example.com/notify",
      "events": ["job.completed", "job.failed"],
      "enabled": true,
      "timeout": 30,
      "retry_count": 3
    }
  ],
  "total": 1,
  "timestamp": "2026-01-17T12:00:00Z"
}
```

### POST /webhooks

Add new webhook.

**Request Body**:
```json
{
  "url": "https://hooks.example.com/notify",
  "events": ["job.completed", "job.failed"],
  "enabled": true,
  "timeout": 30,
  "retry_count": 3,
  "headers": {
    "Authorization": "Bearer token123"
  }
}
```

**Available Events**:
- `job.submitted` - Job submitted
- `job.started` - Job started
- `job.completed` - Job completed successfully
- `job.failed` - Job failed
- `job.progress` - Job progress update
- `schedule.triggered` - Schedule triggered
- `schedule.completed` - Scheduled job completed

**Response**:
```json
{
  "message": "webhook added successfully",
  "webhook": {...}
}
```

### DELETE /webhooks/{index}

Delete webhook by index.

**URL Parameters**:
- `index` - Webhook index (0-based)

**Response**:
```json
{
  "message": "webhook deleted successfully"
}
```

### POST /webhooks/test

Test webhook delivery.

**Request Body**:
```json
{
  "url": "https://hooks.example.com/notify",
  "event": "test"
}
```

**Response**:
```json
{
  "message": "test webhook sent"
}
```

---

## VM Discovery API

### GET /vms/list

List VMs from vCenter.

**Query Parameters**:
- `vcenter_url` - vCenter URL
- `username` - vCenter username
- `password` - vCenter password
- `datacenter` - Datacenter name (optional)

**Response**:
```json
{
  "vms": [
    {
      "name": "web-server-1",
      "path": "Datacenter/vm/production/web-server-1",
      "power_state": "poweredOn",
      "num_cpu": 4,
      "memory_mb": 8192,
      "storage": 107374182400,
      "guest_os": "Ubuntu Linux (64-bit)",
      "ip_address": "192.168.1.10"
    }
  ],
  "total": 1
}
```

### GET /vms/info

Get detailed VM information.

**Query Parameters**:
- `vcenter_url` - vCenter URL
- `username` - vCenter username
- `password` - vCenter password
- `vm_path` - VM inventory path

**Response**:
```json
{
  "name": "web-server-1",
  "path": "Datacenter/vm/production/web-server-1",
  "uuid": "42329208-8c5a-3254-1234-567890abcdef",
  "power_state": "poweredOn",
  "hardware": {
    "num_cpu": 4,
    "memory_mb": 8192,
    "num_virtual_disks": 2
  },
  "disks": [
    {
      "label": "Hard disk 1",
      "capacity_kb": 104857600,
      "file_path": "[datastore1] web-server-1/web-server-1.vmdk"
    }
  ]
}
```

### POST /vms/shutdown

Gracefully shutdown VM.

**Request Body**:
```json
{
  "vcenter_url": "vcenter.example.com",
  "username": "admin@vsphere.local",
  "password": "password",
  "vm_path": "Datacenter/vm/production/web-server-1"
}
```

**Response**:
```json
{
  "message": "VM shutdown initiated"
}
```

### POST /vms/poweroff

Force power off VM.

**Request Body**: Same as shutdown

**Response**:
```json
{
  "message": "VM powered off"
}
```

### POST /vms/remove-cdrom

Remove CD-ROM devices from VM.

**Request Body**: Same as shutdown

**Response**:
```json
{
  "message": "CD-ROM devices removed"
}
```

---

## WebSocket API

### WS /ws

WebSocket endpoint for real-time updates.

**Connection**:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(message);
};
```

**Message Format**:
```json
{
  "type": "status|jobs|job_update|schedule_event",
  "timestamp": "2026-01-17T12:00:00Z",
  "data": {...}
}
```

**Message Types**:

1. **status** - Status update (every 2 seconds)
```json
{
  "type": "status",
  "data": {
    "total_jobs": 42,
    "running_jobs": 3,
    "completed_jobs": 35,
    "failed_jobs": 4
  }
}
```

2. **jobs** - Initial job list on connection
```json
{
  "type": "jobs",
  "data": {
    "jobs": [...]
  }
}
```

3. **job_update** - Real-time job update
```json
{
  "type": "job_update",
  "data": {
    "definition": {...},
    "status": "running",
    "progress": {...}
  }
}
```

4. **schedule_event** - Schedule event
```json
{
  "type": "schedule_event",
  "data": {
    "schedule_id": "daily-backup",
    "event": "triggered|completed|failed"
  }
}
```

**Features**:
- Automatic reconnection with exponential backoff
- Heartbeat ping/pong (every 54 seconds)
- Initial data sent on connection
- Broadcasts only when clients connected (efficient)

---

## Metrics API

### GET /metrics

Prometheus metrics endpoint.

**Note**: Must be enabled in configuration:
```yaml
metrics:
  enabled: true
  port: 8080
```

**Response**: Prometheus text format
```
# HELP hypersdk_api_requests_total Total API requests
# TYPE hypersdk_api_requests_total counter
hypersdk_api_requests_total{method="GET",path="/status",status="OK"} 1234

# HELP hypersdk_api_request_duration_seconds API request duration
# TYPE hypersdk_api_request_duration_seconds histogram
hypersdk_api_request_duration_seconds_bucket{method="GET",path="/status",le="0.1"} 1200

# HELP hypersdk_jobs_total Total jobs by status
# TYPE hypersdk_jobs_total gauge
hypersdk_jobs_total{status="completed"} 35
hypersdk_jobs_total{status="running"} 3
hypersdk_jobs_total{status="failed"} 4
```

**Available Metrics**:
- `hypersdk_api_requests_total` - Total API requests (counter)
- `hypersdk_api_request_duration_seconds` - Request duration (histogram)
- `hypersdk_jobs_total` - Total jobs by status (gauge)
- `hypersdk_workers_active` - Active workers (gauge)
- `hypersdk_build_info` - Build information (gauge)

---

## Error Responses

All endpoints use standard HTTP status codes and return errors in this format:

```json
{
  "error": "descriptive error message"
}
```

**Common Status Codes**:
- `200 OK` - Success
- `201 Created` - Resource created
- `400 Bad Request` - Invalid request
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error
- `503 Service Unavailable` - Service not enabled

---

## Configuration

### Example Configuration File

```yaml
# config.yaml
database:
  path: "/var/lib/hypersdk/jobs.db"

metrics:
  enabled: true
  port: 8080

webhooks:
  - url: "https://hooks.example.com/notify"
    events:
      - job.completed
      - job.failed
    enabled: true
    timeout: 30
    retry_count: 3
    headers:
      Authorization: "Bearer token123"
```

---

## Client Examples

### cURL Examples

**Submit Job**:
```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "Datacenter/vm/test-vm",
    "source": {"type": "vsphere", ...},
    "destination": {"type": "kvm", ...}
  }'
```

**Query Jobs**:
```bash
curl -X POST http://localhost:8080/jobs/query \
  -H "Content-Type: application/json" \
  -d '{"all": true}'
```

**Create Schedule**:
```bash
curl -X POST http://localhost:8080/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "nightly-backup",
    "schedule": "0 2 * * *",
    "job_template": {...}
  }'
```

### Python Example

```python
import requests

# Submit job
response = requests.post('http://localhost:8080/jobs/submit', json={
    'vm_path': 'Datacenter/vm/test-vm',
    'source': {'type': 'vsphere', ...},
    'destination': {'type': 'kvm', ...}
})

job_id = response.json()['job_id']
print(f"Job submitted: {job_id}")

# Monitor job
while True:
    job = requests.get(f'http://localhost:8080/jobs/{job_id}').json()
    if job['status'] in ['completed', 'failed']:
        break
    print(f"Progress: {job['progress']['percent_complete']}%")
    time.sleep(5)
```

### JavaScript Example

```javascript
// Using axios
const axios = require('axios');

// Submit job
const response = await axios.post('http://localhost:8080/jobs/submit', {
  vm_path: 'Datacenter/vm/test-vm',
  source: {type: 'vsphere', ...},
  destination: {type: 'kvm', ...}
});

const jobId = response.data.job_id;

// Connect WebSocket for real-time updates
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  if (msg.type === 'job_update' &&
      msg.data.definition.id === jobId) {
    console.log('Progress:', msg.data.progress.percent_complete + '%');
  }
};
```

---

## Rate Limiting

Currently no rate limiting is implemented. Consider adding in production:

- Request rate limits per IP
- Concurrent job limits per user
- WebSocket connection limits

---

## Best Practices

1. **Use WebSocket** for real-time updates instead of polling
2. **Handle errors** gracefully with exponential backoff
3. **Set timeouts** on HTTP requests (recommended: 30s)
4. **Validate input** before submitting jobs
5. **Monitor metrics** using Prometheus endpoint
6. **Use schedules** for recurring migrations
7. **Configure webhooks** for event notifications
8. **Test webhooks** before production use

---

## Support

- GitHub Issues: https://github.com/hypersdk/hypersdk/issues
- Documentation: https://github.com/hypersdk/hypersdk/docs
- Examples: https://github.com/hypersdk/hypersdk/examples
