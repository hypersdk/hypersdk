# HyperSDK Documentation Index

Welcome to the HyperSDK documentation. This guide helps you navigate all available documentation.

## Quick Start

- **[Getting Started](GETTING-STARTED.md)** - Quick start guide for new users
- **[README](../README.md)** - Project overview and introduction

## User Guides

### Core Guides
1. **[Interactive Mode User Guide](user-guides/01-interactive-mode.md)** - Complete guide to the interactive TUI for VM migration
2. **[VM Export Guide](user-guides/02-vm-export-guide.md)** - Step-by-step VM export procedures
3. **[Integration Guide](user-guides/03-integration.md)** - Integrating HyperSDK into your workflow
4. **[Manifest Integration with hyper2kvm](user-guides/04-manifest-integration.md)** - Artifact Manifest examples and integration patterns

### Feature Documentation
1. **[Interactive Migration Features](features/01-interactive-migration.md)** - Detailed interactive migration features
2. **[Hyperctl Features](features/02-hyperctl-features.md)** - Command-line interface features
3. **[Feature Reference](features/03-feature-reference.md)** - Comprehensive feature catalog

## Reference Documentation

### Architecture & API
- **[Daemon API](api/01-daemon-api.md)** - Daemon REST API reference
- **[Project Summary](PROJECT-SUMMARY.md)** - High-level architecture overview

### Examples
- **[Examples Index](../examples/README.md)** - Configuration file examples
- **[Demo Scripts](../demos/README.md)** - Live demonstration scripts

## Testing & Development

- **[Test Results](TEST-RESULTS.md)** - Test coverage and results
- **[Recording Guide](../demos/RECORDING-GUIDE.md)** - Creating demo recordings

## Project Organization

- **[Directory Organization](ORGANIZATION.md)** - Repository structure and organization guidelines
- **[Roadmap](ROADMAP.md)** - Future features and development plans
- **[New Features](NEW-FEATURES.md)** - Recent feature additions and changes

## Documentation Structure

```
hypersdk/
├── README.md                   # Project overview
├── docs/
│   ├── 00-INDEX.md            # This file
│   ├── GETTING-STARTED.md     # Quick start guide
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
│   │   └── 03-feature-reference.md
│   └── api/                   # API references
│       └── 01-daemon-api.md
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
1. Place user guides in `docs/user-guides/` with numeric prefix (e.g., `04-new-guide.md`)
2. Place feature docs in `docs/features/` with numeric prefix
3. Place API docs in `docs/api/` with numeric prefix
4. Update this index file to include the new document
5. Follow the existing documentation style and format

## License

All documentation is licensed under LGPL-3.0-or-later.
