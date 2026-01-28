# HyperSDK Documentation Index

Welcome to the HyperSDK documentation. This guide helps you navigate all available documentation.

## Quick Start

- **[Getting Started](getting-started.md)** - Quick start guide for new users
- **[Quickstart Guide](quickstart-guide.md)** - Fast-track setup and usage
- **[README](../README.md)** - Project overview and introduction
- **[Docker Quick Start](../examples/docker-quickstart.md)** - Get running with Docker in 5 minutes
- **[Kubernetes Quick Start](../examples/kubernetes-quickstart.md)** - Deploy to Kubernetes in 10 minutes
- **[K3d Quick Start](../examples/k3d-quickstart.md)** - Local k3d cluster testing and development

## Tutorials

Step-by-step tutorials for common tasks:
- **[Getting Started Tutorial](tutorials/getting-started.md)** - Your first VM export with Docker, Kubernetes, or Docker Compose
- **[Configuration Tutorial](tutorials/configuration.md)** - Advanced configuration options for all deployment methods
- **[Multi-Cloud Setup](tutorials/multi-cloud-setup.md)** - Configure and use multiple cloud providers simultaneously
- **[NFS Shared Storage](tutorials/nfs-shared-storage.md)** - Deploy with NFS for cross-environment access (Kubernetes + Native Linux)

## User Guides

### Core Guides
1. **[Interactive Mode User Guide](user-guides/01-interactive-mode.md)** - Complete guide to the interactive TUI for VM migration
2. **[VM Export Guide](user-guides/02-vm-export-guide.md)** - Step-by-step VM export procedures
3. **[Integration Guide](user-guides/03-integration.md)** - Integrating HyperSDK into your workflow
4. **[Manifest Integration with hyper2kvm](user-guides/04-manifest-integration.md)** - Artifact Manifest examples and integration patterns

### Deployment & Configuration
- **[Deployment Guide](deployment-guide.md)** - Production deployment guidelines
- **[Docker/Podman Deployment](../deployments/docker/README.md)** - Container deployment with Docker Compose
- **[Kubernetes Deployment](../deployments/kubernetes/README.md)** - Kubernetes with Kustomize (dev/staging/prod)
- **[Systemd Integration](../systemd/README.md)** - Native Linux systemd service
- **[Multi-Cloud Guide](multi-cloud-guide.md)** - Multi-cloud migration workflows

### Cloud Provider Integrations
- **[Cloud Providers Overview](cloud-providers/README.md)** - All supported cloud platforms
- **[Alibaba Cloud](cloud-providers/alibabacloud-integration.md)** - Alibaba Cloud integration
- **[AWS Migration](cloud-providers/aws-migration-guide.md)** - Amazon Web Services migration
- **[Oracle Cloud (OCI)](cloud-providers/oci-integration.md)** - Oracle Cloud Infrastructure
- **[OpenStack](cloud-providers/openstack-integration.md)** - OpenStack cloud platform
- **[Proxmox VE](cloud-providers/proxmox-integration.md)** - Proxmox Virtual Environment

### Feature Documentation
1. **[Interactive Migration Features](features/01-interactive-migration.md)** - Detailed interactive migration features
2. **[Hyperctl Features](features/02-hyperctl-features.md)** - Command-line interface features
3. **[Feature Reference](features/03-feature-reference.md)** - Comprehensive feature catalog
4. **[Features Complete](features/features-complete.md)** - Complete feature documentation
5. **[Hyperexport Features](features/hyperexport/README.md)** - Hyperexport-specific features
   - Export resumption, bandwidth throttling, TUI guides, and more

## Reference Documentation

### Technical References
- **[Reference Documentation](reference/README.md)** - Complete technical reference
  - **[CLI Reference](reference/cli-reference.md)** - Command-line interface
  - **[Configuration Reference](reference/configuration-reference.md)** - Config file options
  - **[Performance Tuning](reference/performance-tuning.md)** - Optimization guide
  - **[Troubleshooting Guide](reference/troubleshooting-guide.md)** - Problem solving

### Architecture & API
- **[API Documentation](api/README.md)** - Complete API reference
  - **[API Overview](api/00-overview.md)** - API introduction and architecture
  - **[Daemon API](api/01-daemon-api.md)** - REST API reference
  - **[API Endpoints](api/02-endpoints.md)** - Endpoint reference
  - **[New Features](api/03-new-features.md)** - Recent API additions
- **[Architecture Documentation](architecture/)** - System architecture and design
  - **[NFS Deployment Architecture](architecture/nfs-deployment-architecture.md)** - Complete NFS architecture with diagrams
- **[Project Summary](project-summary.md)** - High-level architecture overview

### Examples
- **[Examples Index](../examples/README.md)** - Configuration file examples
- **[Demo Scripts](../demos/README.md)** - Live demonstration scripts

## Testing & Development

### Testing Documentation
- **[Testing Overview](testing/00-testing-overview.md)** - Test structure and organization
- **[Testing Guide](testing/testing-guide.md)** - Practical testing guide with examples
- **[Test Results](test-results.md)** - Test coverage and results
- **[Bug Fixes and Tests](testing/bug-fixes-and-tests.md)** - Bug fixes and test cases
- **[Dashboard Testing](testing/dashboard-testing.md)** - Web dashboard testing guide
- **[Hyperexport Testing](testing/hyperexport-testing.md)** - Hyperexport test guide
- **[Hyperexport Quick Test](testing/hyperexport-quicktest.md)** - Quick test procedures
- **[Recording Guide](../demos/RECORDING-GUIDE.md)** - Creating demo recordings

### Integration Guides
- **[Daemon Integration](integration/daemon-integration.md)** - Daemon integration complete guide
- **[Systemd Daemon](integration/systemd-daemon.md)** - Systemd service integration
- **[Pipeline Integration](integration/pipeline-integration.md)** - CI/CD pipeline integration

### Development Notes
- **[Code Review](development/CODE_REVIEW.md)** - Code review documentation
- **[Code Review Detailed](development/CODE_REVIEW_DETAILED.md)** - Detailed code review
- **[Code Review Report](development/CODE_REVIEW_REPORT.md)** - Code review report
- **[Code Review Summary](development/CODE_REVIEW_SUMMARY.md)** - Code review summary
- **[Implementation Summary](development/IMPLEMENTATION_SUMMARY.md)** - Implementation notes
- **[Final Changes Summary](development/FINAL_CHANGES_SUMMARY.md)** - Final changes
- **[Task 15 Completion](development/TASK_15_COMPLETION_SUMMARY.md)** - Task completion notes
- **[TUI Crash Fix](development/TUI_CRASH_FIX.md)** - TUI crash fix documentation
- **[Web UX Integration](development/WEB_UX_INTEGRATION_SUMMARY.md)** - Web UX integration summary
- **[Hyperexport Development](development/hyperexport/README.md)** - Hyperexport dev notes
  - Implementation details, roadmaps, and enhancement tracking

## Project Organization

- **[Directory Organization](organization.md)** - Repository structure and organization guidelines
- **[Roadmap](roadmap.md)** - Future features and development plans
- **[New Features](new-features.md)** - Recent feature additions and changes

## Documentation Structure

```
hypersdk/
├── README.md                   # Project overview
├── CHANGELOG.md               # Version history
├── SECURITY.md                # Security policy
├── docs/
│   ├── 00-INDEX.md            # This file
│   ├── getting-started.md     # Quick start guide
│   ├── quickstart-guide.md    # Fast-track setup
│   ├── deployment-guide.md    # Deployment guide
│   ├── multi-cloud-guide.md   # Multi-cloud workflows
│   ├── project-summary.md     # Architecture overview
│   ├── test-results.md        # Test coverage and results
│   ├── tutorials/             # Step-by-step tutorials
│   │   ├── getting-started.md  # First VM export tutorial
│   │   ├── configuration.md    # Configuration tutorial
│   │   ├── multi-cloud-setup.md # Multi-cloud setup tutorial
│   │   └── nfs-shared-storage.md # NFS shared storage setup
│   ├── user-guides/           # Step-by-step guides
│   │   ├── 01-interactive-mode.md
│   │   ├── 02-vm-export-guide.md
│   │   ├── 03-integration.md
│   │   └── 04-manifest-integration.md
│   ├── features/              # Feature documentation
│   │   ├── 01-interactive-migration.md
│   │   ├── 02-hyperctl-features.md
│   │   ├── 03-feature-reference.md
│   │   ├── features-complete.md
│   │   └── hyperexport/       # Hyperexport features
│   │       ├── README.md
│   │       ├── user-guide.md
│   │       ├── keyboard-shortcuts.md
│   │       └── ...
│   ├── cloud-providers/       # Cloud provider integrations
│   │   ├── README.md
│   │   ├── alibabacloud-integration.md
│   │   ├── aws-migration-guide.md
│   │   ├── oci-integration.md
│   │   ├── openstack-integration.md
│   │   └── proxmox-integration.md
│   ├── architecture/          # Architecture documentation
│   │   └── nfs-deployment-architecture.md # NFS deployment architecture
│   ├── api/                   # API references
│   │   ├── README.md
│   │   ├── 00-overview.md
│   │   ├── 01-daemon-api.md
│   │   ├── 02-endpoints.md
│   │   └── 03-new-features.md
│   ├── reference/             # Technical references
│   │   ├── README.md
│   │   ├── cli-reference.md
│   │   ├── configuration-reference.md
│   │   ├── performance-tuning.md
│   │   └── troubleshooting-guide.md
│   ├── testing/               # Testing documentation
│   │   ├── README.md
│   │   ├── 00-testing-overview.md
│   │   ├── testing-guide.md
│   │   ├── bug-fixes-and-tests.md
│   │   ├── dashboard-testing.md
│   │   ├── hyperexport-testing.md
│   │   └── hyperexport-quicktest.md
│   ├── integration/           # Integration guides
│   │   ├── daemon-integration.md
│   │   ├── systemd-daemon.md
│   │   └── pipeline-integration.md
│   └── development/           # Development notes
│       ├── CODE_REVIEW.md
│       ├── IMPLEMENTATION_SUMMARY.md
│       ├── hyperexport/       # Hyperexport dev notes
│       │   ├── README.md
│       │   ├── implementation-summary.md
│       │   └── ...
│       └── ...
├── examples/                  # Configuration examples
│   ├── README.md
│   ├── docker-quickstart.md    # Docker quick start guide
│   ├── kubernetes-quickstart.md # Kubernetes quick start guide
│   ├── example-vm-export.yaml
│   ├── example-vm-export.json
│   ├── example-batch-export.yaml
│   └── example-batch-export.json
└── demos/                     # Demo scripts
    ├── README.md
    ├── quick-demo.sh
    └── RECORDING-GUIDE.md
```

## Contributing

When adding new documentation:
1. Place tutorials in `docs/tutorials/` (step-by-step guides with examples)
2. Place user guides in `docs/user-guides/` with numeric prefix (e.g., `05-new-guide.md`)
3. Place feature docs in `docs/features/` with numeric prefix
4. Place API docs in `docs/api/` with numeric prefix
5. Place cloud provider integrations in `docs/cloud-providers/`
6. Place testing docs in `docs/testing/`
7. Place integration guides in `docs/integration/`
8. Place development notes in `docs/development/`
9. Place quick start examples in `examples/` (e.g., `docker-quickstart.md`)
10. Update this index file to include the new document
11. Follow the existing documentation style and format

## License

All documentation is licensed under LGPL-3.0-or-later.
