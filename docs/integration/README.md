# Integration Documentation

This directory contains guides for integrating HyperSDK with various systems and workflows.

## Quick Navigation

### For Meeting Presentations
- **[MEETING_QUICK_START.md](MEETING_QUICK_START.md)** - 5-minute overview and demo guide for meetings
  - Architecture diagrams
  - Quick API examples
  - Common use cases

### For hyper2kvm Integration
- **[hyper2kvm-workflow.md](hyper2kvm-workflow.md)** - Complete HyperSDK → hyper2kvm workflow documentation
  - Detailed integration patterns
  - Step-by-step implementation guide
  - API endpoint reference
  - Configuration examples
  - Troubleshooting guide
- **[../examples/hyper2kvm-demo.sh](../../examples/hyper2kvm-demo.sh)** - Interactive demo script

### System Integration
- **[daemon-integration.md](daemon-integration.md)** - Complete guide to daemon integration and setup
- **[systemd-daemon.md](systemd-daemon.md)** - Integrating HyperSDK daemon with systemd services
- **[pipeline-integration.md](pipeline-integration.md)** - CI/CD pipeline integration patterns

## Integration Overview

### HyperSDK → hyper2kvm Workflow

```
Cloud Provider → HyperSDK Export → hyper2kvm Convert → KVM Import
   (vSphere)         (Go daemon)      (Python tool)      (libvirt)
```

**Quick Start:**
```bash
# Run the interactive demo
./examples/hyper2kvm-demo.sh

# Or submit a job via API
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{"vm_identifier": "test-vm", "provider": "vsphere", "hyper2kvm_integration": {"enabled": true}}'
```

## Integration Types

### System Integration
- Systemd service configuration
- Daemon process management
- System resource monitoring
- hyper2kvm daemon integration

### Workflow Integration
- CI/CD pipeline integration
- Automation workflows
- Batch processing
- VM migration workflows

### External Tool Integration
- hyper2kvm (VM conversion to KVM)
- libvirt (KVM import)
- Cloud provider APIs
- Webhook notifications

## Use Cases

1. **Single VM Migration** - Export and convert one VM from vSphere to KVM
2. **Batch Migration** - Migrate 50+ VMs overnight with queue processing
3. **Continuous Backups** - Scheduled daily backups with retention policies

See [MEETING_QUICK_START.md](MEETING_QUICK_START.md) for detailed use case examples.

## Related Documentation

- [User Guides](../user-guides/) - Step-by-step usage guides
- [API Documentation](../api/) - Complete API reference
- [Deployment Guide](../deployment-guide.md) - Production deployment
- [Examples](../../examples/) - Configuration and demo scripts
