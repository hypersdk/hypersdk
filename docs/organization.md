# HyperSDK Directory Organization

## Overview

This document describes the organization of the HyperSDK repository and the changes made to improve structure and maintainability.

## Directory Structure

```
hypersdk/
├── build/                      # Build artifacts (gitignored)
│   ├── hyperctl               # hyperctl binary
│   ├── hyperexport            # hyperexport binary
│   └── hypervisord            # hypervisord binary
├── cmd/                        # Command-line applications
│   ├── hyperctl/              # Interactive TUI
│   ├── hyperexport/           # Export tool
│   └── hypervisord/           # Daemon service
├── config/                     # Config package (Go source)
├── daemon/                     # Daemon implementation
├── demos/                      # Demo scripts and recordings
│   ├── README.md
│   ├── RECORDING-GUIDE.md
│   └── *.sh                   # Demo scripts
├── docs/                       # Documentation
│   ├── 00-INDEX.md            # Documentation index
│   ├── getting-started.md     # Quick start guide
│   ├── project-summary.md     # Architecture overview
│   ├── test-results.md        # Test coverage
│   ├── README.md              # Docs overview
│   ├── ROADMAP.md             # Future plans
│   ├── NEW-FEATURES.md        # Feature changelog
│   ├── api.md                 # API reference
│   ├── user-guides/           # Step-by-step guides
│   ├── features/              # Feature documentation
│   └── api/                   # API references
├── examples/                   # Configuration examples
│   ├── README.md
│   ├── example-vm-export.yaml
│   ├── example-vm-export.json
│   ├── example-batch-export.yaml
│   └── example-batch-export.json
├── logger/                     # Logger package
├── man/                        # Man pages
├── progress/                   # Progress reporting
├── providers/                  # Cloud provider implementations
│   └── vsphere/               # vSphere provider
├── systemd/                    # systemd service files
├── web/                        # Web UI (future)
├── .github/                    # GitHub workflows
├── .git/                       # Git repository
├── .gitignore                 # Git ignore rules
├── config.yaml.example        # Example configuration
├── Dockerfile                 # Container build
├── go.mod                     # Go module definition
├── go.sum                     # Go dependency checksums
├── hypersdk.spec              # RPM spec file
├── install.sh                 # Installation script
├── LICENSE                    # LGPL-3.0 license
├── Makefile                   # Build automation
├── README.md                  # Project overview
└── test_rpmbuild.sh           # RPM build test script
```

## Recent Organization Changes

### 1. Cleaned Up Root Directory

**Removed:**
- `hyperctl` (duplicate, kept in `build/`)
- `hyperexport` (duplicate, kept in `build/`)
- `hypervisord` (duplicate, kept in `build/`)
- `coverage.out` (test artifact)

**Result:** Cleaner root with only essential project files.

### 2. Consolidated Documentation

**Moved to `docs/`:**
- `getting-started.md` → `docs/getting-started.md`
- `project-summary.md` → `docs/project-summary.md`
- `test-results.md` → `docs/test-results.md`

**Updated References:**
- `docs/00-INDEX.md` - Updated all documentation links
- `README.md` - Updated documentation section

**Result:** All documentation now in `docs/` directory for better organization.

### 3. Updated .gitignore

**Added:**
- `coverage.out` - Explicit coverage file ignore

**Result:** Better coverage of temporary build artifacts.

## File Organization Guidelines

### Root Directory

**Should Contain:**
- Essential project files (README.md, LICENSE, Makefile)
- Configuration files (config.yaml.example, Dockerfile)
- Build scripts (install.sh, test_rpmbuild.sh)
- Go module files (go.mod, go.sum)
- Package spec files (hypersdk.spec)

**Should NOT Contain:**
- Build artifacts (use `build/` directory)
- Documentation (use `docs/` directory)
- Temporary files (add to `.gitignore`)

### Documentation (`docs/`)

**Organization:**
1. **Index** - `00-INDEX.md` - Central documentation catalog
2. **Quick Start** - `getting-started.md` - New user guide
3. **Architecture** - `project-summary.md` - System design
4. **Testing** - `test-results.md` - Test coverage
5. **Guides** - `user-guides/` - Step-by-step tutorials
6. **Features** - `features/` - Feature documentation
7. **API** - `api/` - API references

**Naming Conventions:**
- Use descriptive, lowercase filenames with hyphens
- Prefix numbered guides with `01-`, `02-`, etc.
- Use `.md` extension for all markdown files

### Build Artifacts (`build/`)

**Contains:**
- Compiled binaries (`hyperctl`, `hyperexport`, `hypervisord`)
- Generated files from build process
- Temporary build artifacts

**Note:** This directory is gitignored - do not commit contents.

### Examples (`examples/`)

**Contains:**
- Sample configuration files
- Reference implementations
- Working examples for users

**Guidelines:**
- Use realistic, working examples
- Include both YAML and JSON formats
- Add comments explaining options

### Demos (`demos/`)

**Contains:**
- Demo scripts showing features
- Screen recordings (when available)
- Recording guides

**Guidelines:**
- Keep scripts simple and focused
- Add descriptive comments
- Include README with usage

## Maintenance

### Adding New Documentation

1. Create file in appropriate `docs/` subdirectory
2. Use numeric prefix for ordered content (e.g., `04-new-guide.md`)
3. Update `docs/00-INDEX.md` to include the new document
4. Add link to main `README.md` if it's important
5. Follow existing documentation style

### Cleaning Build Artifacts

```bash
# Remove all build artifacts
make clean

# Or manually
rm -rf build/*
rm -f coverage.out
```

### Updating Documentation Links

When moving documentation:
1. Update `docs/00-INDEX.md` links
2. Update `README.md` references
3. Search for old links: `grep -r "old-file.md" .`
4. Update all references found

## Benefits of Organization

### Cleaner Root Directory

- ✅ Easy to navigate
- ✅ Clear project structure
- ✅ Professional appearance
- ✅ Better for new contributors

### Consolidated Documentation

- ✅ All docs in one place
- ✅ Easy to find information
- ✅ Better organization
- ✅ Improved discoverability

### Better Build Management

- ✅ Build artifacts in dedicated directory
- ✅ Clean separation from source
- ✅ Easy to clean up
- ✅ Proper gitignore coverage

## Quick Reference

### Essential Files

| File | Purpose | Location |
|------|---------|----------|
| README.md | Project overview | Root |
| LICENSE | License text | Root |
| Makefile | Build automation | Root |
| go.mod | Go dependencies | Root |
| config.yaml.example | Config template | Root |

### Documentation

| Document | Purpose | Location |
|----------|---------|----------|
| 00-INDEX.md | Doc catalog | docs/ |
| getting-started.md | Quick start | docs/ |
| project-summary.md | Architecture | docs/ |
| test-results.md | Test coverage | docs/ |
| README.md | Docs overview | docs/ |

### Build Commands

```bash
# Build all binaries (outputs to build/)
make build

# Build specific binary
make hyperctl

# Clean build artifacts
make clean

# Run tests
make test

# Run tests with coverage
make coverage
```

## Contributing

When contributing to HyperSDK:

1. **Follow the structure** - Place files in appropriate directories
2. **Update documentation** - Keep docs in sync with changes
3. **Clean up** - Remove temporary files before committing
4. **Test builds** - Ensure `make clean && make build` works
5. **Update links** - Fix any broken documentation references

## Related Documentation

- [Documentation Index](00-INDEX.md) - Complete documentation catalog
- [Getting Started](getting-started.md) - New user guide
- [Project Summary](project-summary.md) - Architecture overview
- [Contributing Guide](../README.md#contributing) - Contribution guidelines

## License

This documentation is licensed under LGPL-3.0-or-later, the same as the project.
