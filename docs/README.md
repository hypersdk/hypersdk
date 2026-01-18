# HyperSDK Documentation

Complete documentation for HyperSDK - A multi-cloud VM migration framework with focus on vSphere to KVM migration.

## 📚 Documentation Navigation

### New Users Start Here
- **[Getting Started Guide](GETTING-STARTED.md)** - Install and run your first migration (includes dashboard!)
- **[Interactive Mode Guide](user-guides/01-interactive-mode.md)** - Learn the interactive TUI
- **[Dashboard README](../daemon/dashboard/README.md)** - Web dashboard usage and features

### Dashboard Documentation
1. **[Dashboard README](../daemon/dashboard/README.md)** - Complete dashboard usage guide
2. **[Dashboard Implementation](../DASHBOARD_IMPLEMENTATION_COMPLETE.md)** - Implementation details and features
3. **[Dashboard Testing Report](../DASHBOARD_TESTING_REPORT.md)** - Comprehensive testing results (51+ endpoints)

### User Guides
1. **[Interactive Mode](user-guides/01-interactive-mode.md)** - Complete guide to the TUI migration tool
2. **[VM Export Guide](user-guides/02-vm-export-guide.md)** - Step-by-step export procedures
3. **[Integration Guide](user-guides/03-integration.md)** - Integrate HyperSDK into workflows

### Feature Documentation
1. **[Interactive Migration Features](features/01-interactive-migration.md)** - Interactive mode capabilities
2. **[Hyperctl Features](features/02-hyperctl-features.md)** - CLI command reference
3. **[Feature Reference](features/03-feature-reference.md)** - Comprehensive feature catalog

### API & Reference
- **[API Endpoints](API_ENDPOINTS.md)** - Complete API reference (51+ endpoints)
- **[API New Features](API_REFERENCE_NEW_FEATURES.md)** - Phase 2 features documentation
- **[General API Documentation](api.md)** - API usage patterns
- **[Daemon API Reference](api/01-daemon-api.md)** - Legacy daemon API docs
- **[Examples](../examples/README.md)** - Configuration file examples
- **[Project Summary](PROJECT-SUMMARY.md)** - Architecture overview

### Testing & Development
- **[Test Results](TEST-RESULTS.md)** - Coverage and test reports
- **[Test API Script](../scripts/test-api.sh)** - Automated API testing (79 endpoints)
- **[Dashboard Endpoint Testing](../test-dashboard-endpoints.sh)** - Dashboard endpoint tests
- **[Demo Scripts](../demos/README.md)** - Live demonstrations
- **[Recording Guide](../demos/RECORDING-GUIDE.md)** - Creating demo videos

## 🗂️ Directory Structure

```
docs/
├── README.md                    # This file
├── 00-INDEX.md                  # Complete documentation index
│
├── user-guides/                 # Step-by-step tutorials
│   ├── 01-interactive-mode.md   # Interactive TUI guide
│   ├── 02-vm-export-guide.md    # VM export procedures
│   └── 03-integration.md        # Integration patterns
│
├── features/                    # Feature documentation
│   ├── 01-interactive-migration.md  # Interactive features
│   ├── 02-hyperctl-features.md      # CLI features
│   └── 03-feature-reference.md      # Complete feature list
│
└── api/                         # API references
    └── 01-daemon-api.md         # Daemon REST API
```

## 🚀 Quick Links

### Most Common Tasks
- **[Access Web Dashboard](GETTING-STARTED.md#using-the-web-dashboard)** - Browser-based UI
- **[View VM Consoles](../daemon/dashboard/README.md#console-viewer-usage)** - VNC and Serial access
- [List VMs from vCenter](user-guides/01-interactive-mode.md#getting-started)
- [Migrate VMs interactively](user-guides/01-interactive-mode.md#migration-workflow)
- [Export VMs via CLI](user-guides/02-vm-export-guide.md)
- [Search and filter VMs](features/02-hyperctl-features.md)
- **[API Reference](API_ENDPOINTS.md)** - All 51+ endpoints

### Configuration Examples
- [Single VM export](../examples/example-vm-export.yaml)
- [Batch VM export](../examples/example-batch-export.yaml)
- [Daemon configuration](../config.yaml.example)

### Dashboard Access
```bash
# Start daemon
./hypervisord

# Access dashboard
http://localhost:8080/web/dashboard/           # Main dashboard
http://localhost:8080/web/dashboard/vm-console.html  # Console viewer
```

## 📖 Reading Guide

### For First-Time Users
1. Read [Getting Started](../GETTING-STARTED.md)
2. Try [Interactive Mode](user-guides/01-interactive-mode.md)
3. Review [Examples](../examples/README.md)

### For Developers
1. Read [Project Summary](../PROJECT-SUMMARY.md)
2. Review [API Reference](api/01-daemon-api.md)
3. Check [Test Results](../TEST-RESULTS.md)

### For DevOps Engineers
1. Read [Integration Guide](user-guides/03-integration.md)
2. Review [Daemon API](api/01-daemon-api.md)
3. Study [Examples](../examples/README.md)

## 🆘 Getting Help

- **Documentation Index**: See [00-INDEX.md](00-INDEX.md) for complete listing
- **Issues**: Report issues on GitHub
- **Examples**: Check [examples/](../examples/) directory

## 📝 Contributing to Documentation

When adding documentation:
1. Place in appropriate subdirectory (user-guides/, features/, api/)
2. Use numeric prefix for ordering (e.g., `04-new-guide.md`)
3. Update this README and [00-INDEX.md](00-INDEX.md)
4. Follow existing documentation style
5. Include code examples where relevant

## License

Documentation is licensed under LGPL-3.0-or-later.
