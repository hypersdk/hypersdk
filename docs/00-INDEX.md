# HyperSDK Documentation Index

Welcome to the HyperSDK documentation. This guide helps you navigate all available documentation.

## Quick Start

- **[Getting Started](GETTING-STARTED.md)** - Quick start guide for new users
- **[Quickstart Guide](quickstart-guide.md)** - Fast-track setup and usage
- **[README](../README.md)** - Project overview and introduction

## User Guides

### Core Guides
1. **[Interactive Mode User Guide](user-guides/01-interactive-mode.md)** - Complete guide to the interactive TUI for VM migration
2. **[VM Export Guide](user-guides/02-vm-export-guide.md)** - Step-by-step VM export procedures
3. **[Integration Guide](user-guides/03-integration.md)** - Integrating HyperSDK into your workflow
4. **[Manifest Integration with hyper2kvm](user-guides/04-manifest-integration.md)** - Artifact Manifest examples and integration patterns

### Deployment & Configuration
- **[Deployment Guide](deployment-guide.md)** - Production deployment guidelines
- **[Multi-Cloud Guide](multi-cloud-guide.md)** - Multi-cloud migration workflows

### Feature Documentation
1. **[Interactive Migration Features](features/01-interactive-migration.md)** - Detailed interactive migration features
2. **[Hyperctl Features](features/02-hyperctl-features.md)** - Command-line interface features
3. **[Feature Reference](features/03-feature-reference.md)** - Comprehensive feature catalog
4. **[Features Complete](features/features-complete.md)** - Complete feature documentation

## Reference Documentation

### Architecture & API
- **[Daemon API](api/01-daemon-api.md)** - Daemon REST API reference
- **[Project Summary](PROJECT-SUMMARY.md)** - High-level architecture overview

### Examples
- **[Examples Index](../examples/README.md)** - Configuration file examples
- **[Demo Scripts](../demos/README.md)** - Live demonstration scripts

## Testing & Development

### Testing Documentation
- **[Test Results](TEST-RESULTS.md)** - Test coverage and results
- **[Testing Guide](testing/testing-guide.md)** - Comprehensive testing guide
- **[Bug Fixes and Tests](testing/bug-fixes-and-tests.md)** - Bug fixes and test cases
- **[Dashboard Testing](testing/dashboard-testing.md)** - Web dashboard testing guide
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

## Project Organization

- **[Directory Organization](ORGANIZATION.md)** - Repository structure and organization guidelines
- **[Roadmap](ROADMAP.md)** - Future features and development plans
- **[New Features](NEW-FEATURES.md)** - Recent feature additions and changes

## Documentation Structure

```
hypersdk/
├── README.md                   # Project overview
├── CHANGELOG.md               # Version history
├── SECURITY.md                # Security policy
├── docs/
│   ├── 00-INDEX.md            # This file
│   ├── GETTING-STARTED.md     # Quick start guide
│   ├── quickstart-guide.md    # Fast-track setup
│   ├── deployment-guide.md    # Deployment guide
│   ├── multi-cloud-guide.md   # Multi-cloud workflows
│   ├── PROJECT-SUMMARY.md     # Architecture overview
│   ├── TEST-RESULTS.md        # Test coverage and results
│   ├── user-guides/           # Step-by-step guides
│   │   ├── 01-interactive-mode.md
│   │   ├── 02-vm-export-guide.md
│   │   ├── 03-integration.md
│   │   └── 04-manifest-integration.md
│   ├── features/              # Feature documentation
│   │   ├── 01-interactive-migration.md
│   │   ├── 02-hyperctl-features.md
│   │   ├── 03-feature-reference.md
│   │   └── features-complete.md
│   ├── api/                   # API references
│   │   └── 01-daemon-api.md
│   ├── testing/               # Testing documentation
│   │   ├── testing-guide.md
│   │   ├── bug-fixes-and-tests.md
│   │   └── dashboard-testing.md
│   ├── integration/           # Integration guides
│   │   ├── daemon-integration.md
│   │   ├── systemd-daemon.md
│   │   └── pipeline-integration.md
│   └── development/           # Development notes
│       ├── CODE_REVIEW.md
│       ├── IMPLEMENTATION_SUMMARY.md
│       └── ...
├── examples/                  # Configuration examples
│   ├── README.md
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
1. Place user guides in `docs/user-guides/` with numeric prefix (e.g., `05-new-guide.md`)
2. Place feature docs in `docs/features/` with numeric prefix
3. Place API docs in `docs/api/` with numeric prefix
4. Place testing docs in `docs/testing/`
5. Place integration guides in `docs/integration/`
6. Place development notes in `docs/development/`
7. Update this index file to include the new document
8. Follow the existing documentation style and format

## License

All documentation is licensed under LGPL-3.0-or-later.
