# Web Dashboard Workflow Integration

## Overview

The HyperSDK web dashboard now includes a complete **Login → Auto-Discover → Export** workflow for all 9 supported cloud providers. This feature allows users to visually manage providers, discover VMs, and configure exports through a modern React-based UI.

## Quick Links

### Web Dashboard Components
- **Main Documentation:** [web/dashboard-react/README_WORKFLOW.md](../web/dashboard-react/README_WORKFLOW.md)
- **Integration Summary:** [web/dashboard-react/WORKFLOW_INTEGRATION_SUMMARY.md](../web/dashboard-react/WORKFLOW_INTEGRATION_SUMMARY.md)
- **Backend Guide:** [web/dashboard-react/BACKEND_INTEGRATION_GUIDE.md](../web/dashboard-react/BACKEND_INTEGRATION_GUIDE.md)

### hyper2kvm Integration (for meeting)
- **Quick Start:** [docs/integration/MEETING_QUICK_START.md](./integration/MEETING_QUICK_START.md)
- **Full Workflow:** [docs/integration/hyper2kvm-workflow.md](./integration/hyper2kvm-workflow.md)
- **Demo Script:** [examples/hyper2kvm-demo.sh](../examples/hyper2kvm-demo.sh)

## Features

### 1. Provider Manager
Configure and manage connections to all 9 cloud providers:
- ☁️ VMware vSphere/ESXi
- 🌐 Amazon AWS EC2
- 🔷 Microsoft Azure VMs
- 🔶 Google Cloud Platform
- 💻 Microsoft Hyper-V
- 🟧 Oracle Cloud Infrastructure
- 🌀 OpenStack
- 🟠 Alibaba Cloud
- 🔧 Proxmox VE

### 2. VM Browser
Auto-discover and browse virtual machines with:
- Real-time VM discovery
- Search and filter capabilities
- Detailed VM information (CPU, memory, OS, IP)
- Sortable table view
- One-click selection for export

### 3. Export Workflow
Guided 3-step process:
1. **Select Provider** - Configure and test cloud connections
2. **Discover VMs** - Auto-discover and browse VMs
3. **Configure Export** - Set export options (pre-filled with VM details)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    React Web UI                              │
│                  (Port: 5173 dev)                           │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTP/JSON
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              HyperSDK API Endpoints                         │
│                  (Port: 8080)                               │
│                                                             │
│  GET  /api/providers/list    - List providers              │
│  POST /api/providers/add     - Add provider                │
│  POST /api/providers/test    - Test connection             │
│  POST /api/vms/list          - Discover VMs                │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Provider Abstraction Layer                     │
│         (vSphere, AWS, Azure, GCP, etc.)                   │
└─────────────────────────────────────────────────────────────┘
```

## Getting Started

### Frontend Development

```bash
cd web/dashboard-react
npm install
npm run dev
```

Visit: `http://localhost:5173`

Click "📤 Export Workflow" in the top navigation.

### Backend Integration

Implement these 3 endpoints:

1. **GET /api/providers/list** - List configured providers
2. **POST /api/providers/add** - Add new provider connection
3. **POST /api/vms/list** - Discover VMs from provider

See [BACKEND_INTEGRATION_GUIDE.md](../web/dashboard-react/BACKEND_INTEGRATION_GUIDE.md) for complete implementation examples.

## Component Overview

### New Components

| Component | Size | Purpose |
|-----------|------|---------|
| **VMBrowser.tsx** | 17 KB | Auto-discover and browse VMs |
| **ProviderManager.tsx** | 19 KB | Manage provider connections |
| **ExportWorkflow.tsx** | 15 KB | 3-step guided workflow |

### Updated Components

| Component | Changes |
|-----------|---------|
| **App.tsx** | Added navigation, enhanced auth |
| **JobSubmissionForm.tsx** | Added pre-fill support |

## Use Cases

### Use Case 1: First-Time Setup
1. Login to dashboard
2. Click "Export Workflow"
3. Click "Add Provider"
4. Configure vSphere credentials
5. Click "Test Connection"
6. Click "Browse VMs"
7. Select VM
8. Configure export
9. Submit job

### Use Case 2: Regular Export
1. Login to dashboard
2. Click "Export Workflow"
3. Click "Browse VMs" on existing provider
4. Select VM
5. Submit job (previous settings remembered)

### Use Case 3: Multi-Provider Management
1. Add multiple providers (vSphere, AWS, Azure)
2. Switch between providers
3. Discover VMs from each
4. Export from different clouds

## API Examples

### List Providers
```bash
curl http://localhost:8080/api/providers/list
```

Response:
```json
[
  {
    "provider": "vsphere",
    "enabled": true,
    "connected": true,
    "config": {
      "host": "vcenter.example.com"
    }
  }
]
```

### Add Provider
```bash
curl -X POST http://localhost:8080/api/providers/add \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "vsphere",
    "config": {
      "host": "vcenter.example.com",
      "username": "admin@vsphere.local",
      "password": "secret"
    }
  }'
```

### List VMs
```bash
curl -X POST http://localhost:8080/api/vms/list \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "vsphere",
    "filter": {}
  }'
```

Response:
```json
[
  {
    "id": "vm-123",
    "name": "web-server-01",
    "provider": "vsphere",
    "status": "running",
    "cpu_count": 4,
    "memory_mb": 8192,
    "os": "Ubuntu 20.04",
    "ip_address": "10.0.1.50"
  }
]
```

## Integration with hyper2kvm

The web workflow seamlessly integrates with hyper2kvm:

1. User selects VM via web UI
2. Export job submitted with hyper2kvm integration enabled
3. HyperSDK exports VM
4. hyper2kvm daemon converts to KVM format
5. VM imported to libvirt

See [hyper2kvm-workflow.md](./integration/hyper2kvm-workflow.md) for complete integration guide.

## Security Considerations

### Frontend
- Token-based authentication
- Session storage with localStorage
- Secure logout
- API token validation

### Backend
- Encrypt provider credentials in database
- Don't return passwords in API responses
- Implement RBAC for provider access
- Add audit logging

See [BACKEND_INTEGRATION_GUIDE.md](../web/dashboard-react/BACKEND_INTEGRATION_GUIDE.md) for implementation details.

## Testing

### Manual Testing
```bash
# Start frontend
cd web/dashboard-react
npm run dev

# In another terminal, start backend
cd ../..
./hypervisord --config config.yaml
```

### Component Testing
```bash
cd web/dashboard-react
npm run test
```

### Integration Testing
See [test-integration.sh](./integration/hyper2kvm-workflow.md#integration-test-script) for automated tests.

## Troubleshooting

### VMs Not Showing
- Check provider connection status
- Verify `/api/vms/list` is implemented
- Check backend logs
- Test connection via "Test Connection" button

### Provider Connection Failed
- Verify credentials
- Check network connectivity
- Review firewall rules
- Check SSL certificate (vSphere)

### Export Job Fails
- Check job logs
- Verify disk space
- Check provider permissions
- Review daemon logs

## Production Deployment

### Build Frontend
```bash
cd web/dashboard-react
npm run build
# Output: daemon/dashboard/static-react/
```

### Configure Backend
```yaml
# config.yaml
daemon:
  address: "0.0.0.0:8080"

providers:
  storage:
    type: sqlite
    path: /var/lib/hypervisord/providers.db
```

### Start Services
```bash
sudo systemctl start hypervisord
sudo systemctl enable hypervisord
```

## Documentation

### Complete Documentation
- **Web Workflow:** [web/dashboard-react/README_WORKFLOW.md](../web/dashboard-react/README_WORKFLOW.md)
- **Backend Integration:** [web/dashboard-react/BACKEND_INTEGRATION_GUIDE.md](../web/dashboard-react/BACKEND_INTEGRATION_GUIDE.md)
- **Summary:** [web/dashboard-react/WORKFLOW_INTEGRATION_SUMMARY.md](../web/dashboard-react/WORKFLOW_INTEGRATION_SUMMARY.md)

### hyper2kvm Integration
- **Meeting Guide:** [docs/integration/MEETING_QUICK_START.md](./integration/MEETING_QUICK_START.md)
- **Workflow:** [docs/integration/hyper2kvm-workflow.md](./integration/hyper2kvm-workflow.md)
- **Demo:** [examples/hyper2kvm-demo.sh](../examples/hyper2kvm-demo.sh)

### API Documentation
- **Overview:** [docs/api/00-overview.md](./api/00-overview.md)
- **Daemon API:** [docs/api/01-daemon-api.md](./api/01-daemon-api.md)
- **Endpoints:** [docs/api/02-endpoints.md](./api/02-endpoints.md)

## Contributing

To add support for new features:

1. Create components in `web/dashboard-react/src/components/`
2. Update `App.tsx` navigation
3. Implement backend endpoints
4. Add tests
5. Update documentation

## Support

### Issues
- Check troubleshooting guide
- Review backend logs
- Test API endpoints manually

### Questions
- Review component API documentation
- Check backend integration guide
- See examples in documentation

## License

See main project LICENSE file.

---

**Last Updated:** 2026-01-29

**Version:** 2.0.0 (Web Workflow Integration)

**Status:** ✅ Frontend Complete | ⏳ Backend Integration Pending
