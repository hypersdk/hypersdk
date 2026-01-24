# HyperSDK API Endpoints Documentation

Complete list of all API endpoints available in HyperSDK with integration to hyper2kvm.

## Base URL
```
http://localhost:8080
```

## Core Endpoints

### Health & Status
- `GET /health` - Health check
- `GET /status` - Daemon status
- `GET /capabilities` - System capabilities

## Job Management

### Jobs
- `POST /jobs/submit` - Submit new export job
- `GET /jobs/query` - Query jobs (supports ?all=true, ?id=xxx, ?status=running)
- `POST /jobs/cancel` - Cancel running jobs
- `GET /jobs/{id}` - Get specific job details

## VM Management

### VM Discovery
- `GET /vms/list?server=...&username=...&password=...&insecure=true` - List VMs from vCenter
- `GET /vms/info?path=/datacenter/vm/myvm` - Get VM info
- `POST /vms/shutdown` - Shutdown VM gracefully
- `POST /vms/poweroff` - Power off VM
- `POST /vms/remove-cdrom` - Remove CD-ROM from VM

## Scheduling & Automation

### Schedules
- `GET /schedules/list` - List all scheduled jobs
- `POST /schedules/create` - Create new schedule
```json
{
  "name": "Daily Backup",
  "cron_expr": "0 2 * * *",
  "job_template": "backup-template-1"
}
```

### Backup Policies
- `GET /backup-policies/list` - List backup policies
- `POST /backup-policies/create` - Create backup policy
```json
{
  "name": "Production Backup",
  "frequency": "daily",
  "retention": 30,
  "target_tags": ["production", "critical"]
}
```

### Workflows
- `GET /workflows/list` - List all workflows

## User Management & RBAC

### Users
- `GET /users/list` - List all users
- `POST /users/create` - Create new user
```json
{
  "username": "operator2",
  "email": "operator2@example.com",
  "role": "operator"
}
```

### Roles
- `GET /roles/list` - List all roles (Admin, Operator, Viewer, Auditor)

### API Keys
- `GET /api-keys/list` - List all API keys
- `POST /api-keys/generate` - Generate new API key
```json
{
  "name": "Production API"
}
```

### Sessions
- `GET /sessions/list` - List active user sessions

## Notifications & Alerts

### Notifications
- `GET /notifications/config` - Get notification configuration
- `PUT /notifications/update` - Update notification config
```json
{
  "email": {
    "enabled": true,
    "smtp_server": "smtp.gmail.com",
    "port": 587,
    "on_completion": true,
    "on_failure": true
  },
  "slack": {
    "enabled": true,
    "webhook_url": "https://hooks.slack.com/...",
    "channel": "#alerts"
  }
}
```

### Alert Rules
- `GET /alert-rules/list` - List all alert rules
- `POST /alert-rules/create` - Create alert rule
```json
{
  "name": "Storage Almost Full",
  "condition": "storage_usage",
  "threshold": 90,
  "actions": ["email", "slack"]
}
```

### Webhooks
- `POST /webhooks/test` - Test webhook configuration
```json
{
  "url": "https://example.com/webhook",
  "headers": {
    "Authorization": "Bearer token123"
  }
}
```

## Hyper2KVM Integration

### VM Conversion
- `POST /convert/vm` - Convert VM using hyper2kvm
```json
{
  "source_path": "/vmfs/volumes/datastore1/vm1/vm1.vmdk",
  "dest_path": "/var/lib/libvirt/images/vm1.qcow2",
  "format": "qcow2",
  "compression": "gzip",
  "vcenter_creds": {
    "server": "vcenter.example.com",
    "username": "administrator@vsphere.local",
    "password": "password",
    "insecure": true
  }
}
```

- `GET /convert/list` - List conversion jobs
- `GET /convert/status?id=conv-123` - Get conversion status

### KVM Import
- `POST /import/kvm` - Import VM to KVM/libvirt
```json
{
  "image_path": "/var/lib/libvirt/images/vm1.qcow2",
  "vm_name": "imported-vm-1",
  "memory": 2048,
  "cpus": 2,
  "network": "default"
}
```

### VMDK Parser
- `GET /vmdk/parse?path=/path/to/disk.vmdk` - Parse VMDK file

## Cost Management

### Cost Summary
- `GET /cost/summary` - Get cost summary
```json
{
  "current_month": 2450.00,
  "last_month": 1890.00,
  "change": 560.00,
  "change_percent": 29.6,
  "annual_projected": 28500.00,
  "breakdown": {
    "storage": 1200.00,
    "network": 850.00,
    "compute": 400.00
  }
}
```

- `GET /cost/history` - Get cost history

### Budget
- `GET /budget/config` - Get budget configuration
- `PUT /budget/update` - Update budget
```json
{
  "monthly_budget": 3000.00,
  "alert_threshold": 80.0
}
```

## Organization

### Tags
- `GET /tags/list` - List all tags
- `POST /tags/create` - Create tag
```json
{
  "name": "production",
  "category": "environment",
  "color": "green"
}
```

### Collections
- `GET /collections/list` - List VM collections
- `POST /collections/create` - Create collection
```json
{
  "name": "Production Web Servers",
  "description": "All production web servers",
  "vm_ids": ["vm-1", "vm-2", "vm-3"]
}
```

### Saved Searches
- `GET /searches/list` - List saved searches
- `POST /searches/create` - Save search query
```json
{
  "name": "Production Windows VMs",
  "query": "tags:production AND os:windows"
}
```

## Cloud & Integration

### Cloud Providers
- `GET /cloud/providers/list` - List cloud providers (AWS, Azure, GCP)
- `POST /cloud/providers/configure` - Configure cloud provider
```json
{
  "name": "aws",
  "enabled": true,
  "config": {
    "aws": {
      "access_key_id": "AKIAIOSFODNN7EXAMPLE",
      "secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
      "region": "us-east-1",
      "s3_bucket": "hypersdk-exports"
    }
  }
}
```

### Multi-vCenter
- `GET /vcenter/servers/list` - List all vCenter servers
- `POST /vcenter/servers/add` - Add vCenter server
```json
{
  "name": "Production vCenter",
  "hostname": "vcenter-prod.example.com",
  "username": "administrator@vsphere.local",
  "password": "password",
  "insecure": true
}
```

### Integrations
- `GET /integrations/list` - List integrations (Jenkins, Ansible, Terraform, Grafana)
- `POST /integrations/configure` - Configure integration
```json
{
  "name": "Jenkins CI/CD",
  "type": "jenkins",
  "enabled": true,
  "config": {
    "url": "https://jenkins.example.com",
    "token": "jenkins-api-token"
  }
}
```

## Security & Compliance

### Encryption
- `GET /encryption/config` - Get encryption configuration
- `PUT /encryption/update` - Update encryption settings
```json
{
  "at_rest": {
    "enabled": true,
    "algorithm": "AES-256"
  },
  "in_transit": {
    "require_tls13": true,
    "verify_ssl_certs": true
  },
  "key_management": {
    "storage": "vault",
    "vault_url": "https://vault.example.com"
  }
}
```

### Compliance
- `GET /compliance/frameworks` - List compliance frameworks (GDPR, SOC2, HIPAA)

### Audit Logs
- `GET /audit/logs` - Get audit logs (supports filtering)
- `GET /audit/export` - Export audit logs as CSV

## Migration

### Migration Wizard
- `GET /migration/wizard` - Get wizard state
- `POST /migration/wizard` - Update wizard state
```json
{
  "step": 1,
  "source": {
    "type": "vmware",
    "server": "vcenter.example.com"
  },
  "destination": {
    "type": "kvm",
    "location": "/var/lib/libvirt/images"
  },
  "vms": ["web-server-01", "db-server-01"]
}
```

### Compatibility Check
- `POST /migration/compatibility` - Run compatibility check
```json
{
  "vm_id": "vm-123",
  "platform": "kvm"
}
```

### Rollback
- `POST /migration/rollback` - Rollback failed migration
```json
{
  "vm_id": "vm-123",
  "backup_id": "backup-456"
}
```

## Testing the API

### Using curl

1. **Health check:**
```bash
curl http://localhost:8080/health
```

2. **List VMs:**
```bash
curl "http://localhost:8080/vms/list?server=vcenter.example.com&username=admin@vsphere.local&password=secret&insecure=true"
```

3. **Convert VM:**
```bash
curl -X POST http://localhost:8080/convert/vm \
  -H "Content-Type: application/json" \
  -d '{
    "source_path": "/vmfs/volumes/datastore1/vm1/vm1.vmdk",
    "dest_path": "/var/lib/libvirt/images/vm1.qcow2",
    "format": "qcow2",
    "compression": "gzip"
  }'
```

4. **Import to KVM:**
```bash
curl -X POST http://localhost:8080/import/kvm \
  -H "Content-Type: application/json" \
  -d '{
    "image_path": "/var/lib/libvirt/images/vm1.qcow2",
    "vm_name": "imported-vm-1",
    "memory": 2048,
    "cpus": 2,
    "network": "default"
  }'
```

5. **Create schedule:**
```bash
curl -X POST http://localhost:8080/schedules/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Nightly Backup",
    "cron_expr": "0 1 * * *"
  }'
```

## Response Format

All endpoints return JSON responses with the following structure:

### Success Response
```json
{
  "status": "success",
  "data": { ... }
}
```

### Error Response
```json
{
  "error": "Error message description"
}
```

## HTTP Status Codes

- `200 OK` - Request succeeded
- `201 Created` - Resource created
- `400 Bad Request` - Invalid request
- `404 Not Found` - Resource not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

## Hyper2KVM Integration Notes

The API integrates with the hyper2kvm Python tool located at `/home/ssahani/tt/hyper2kvm/`.

### Prerequisites
1. Python 3.x installed
2. hyper2kvm installed and accessible
3. Required permissions for vCenter access
4. Sufficient disk space for conversions

### Conversion Workflow
1. Call `/convert/vm` with source VMDK path
2. hyper2kvm converts to target format (qcow2/raw)
3. Call `/import/kvm` to import to libvirt
4. VM is now running on KVM

### Supported Formats
- **Input:** VMDK, OVF/OVA, RAW
- **Output:** qcow2, raw, VMDK
- **Compression:** none, gzip, xz

## Error Handling

All conversion and import operations return detailed error messages:

```json
{
  "error": "conversion failed: disk not found - /path/to/disk.vmdk"
}
```

Check logs for detailed Python traceback if conversion fails.
