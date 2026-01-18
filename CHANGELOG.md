# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-01-20

### Added

**Phase 2 Features - Complete API & Dashboard Implementation**

- **Enhanced API Server** with 51+ production-ready endpoints
  - Job management (submit, query, cancel, progress, logs, ETA)
  - VMware VM operations (list, info, shutdown, poweroff, CD-ROM removal)
  - Libvirt domain management (start, stop, reboot, pause, resume)
  - Console access (VNC, serial, screenshots)
  - Snapshot operations (create, revert, delete, list)
  - Network management (list, create, delete, start, stop)
  - Volume operations (create, clone, resize, delete, upload, wipe)
  - ISO management (list, upload, attach, detach)
  - Backup & restore (create, restore, verify, list, delete)
  - Batch operations (bulk start/stop/reboot/snapshot/delete)
  - VM cloning & templates (clone, deploy, export)
  - Resource monitoring (stats, CPU, memory, disk, network)
  - Workflows (conversion, status)
  - Schedules (list, create, update, delete, enable/disable)
  - Webhooks (list, add, delete, test)

- **Web Dashboard** - Browser-based UI
  - Main dashboard at `/web/dashboard/`
  - Real-time job monitoring with auto-refresh
  - VM console viewer (VNC and serial consoles)
  - VM management interface
  - Job submission and tracking
  - System health monitoring
  - Optional disable via `--disable-web` flag for API-only mode

- **Security Enhancements**
  - TLS certificate validation with configurable skip option
  - Path traversal protection in file operations
  - Input sanitization for VM names
  - Constant-time API key comparison
  - Request size limits (configurable, default 10MB)
  - Private IP blocking for webhooks (configurable)
  - Authentication middleware with session tokens
  - Trusted proxy configuration

- **Configuration Options**
  - `web.disabled` config option for API-only deployments
  - `--disable-web` CLI flag for disabling web dashboard
  - Enhanced security configuration section
  - Webhook configuration support
  - Metrics configuration (Prometheus)

- **Documentation**
  - Complete API reference (API_ENDPOINTS.md) with all 51+ endpoints
  - Enhanced Getting Started guide with dashboard instructions
  - Project summary with Phase 2 architecture
  - Dashboard implementation guide
  - Dashboard testing report
  - Security fixes documentation

### Changed

- Enhanced server architecture with middleware chain
- Improved error handling across all API handlers
- Updated build system with version info injection
- Documentation updates for all new features

### Fixed

- Critical TLS certificate validation issues
- Path traversal vulnerabilities in file operations
- Timing attack vulnerabilities in authentication
- Input validation in VM operations

## [0.1.0] - 2026-01-17

### Added

**Initial Release - Core VMware Export Functionality**

- **Three Command-Line Tools**
  - `hyperexport` - Interactive CLI with beautiful terminal UI
  - `hypervisord` - Background daemon with REST API
  - `hyperctl` - Control CLI for daemon management

- **vSphere Integration**
  - Direct SDK integration via govmomi v0.52.0
  - VM discovery and enumeration
  - OVF/OVA export functionality
  - Parallel download support (configurable workers)
  - Resumable downloads with automatic retry
  - CD/DVD device removal during export

- **Core API Endpoints** (6 initial endpoints)
  - `/health` - Health check
  - `/status` - Daemon status
  - `/capabilities` - Export capabilities detection
  - `/jobs/submit` - Submit export jobs (JSON/YAML)
  - `/jobs/query` - Query job status
  - `/jobs/cancel` - Cancel running jobs

- **Job Management**
  - Concurrent job processing with goroutines
  - Job status tracking (pending, running, completed, failed)
  - Progress reporting with ETAs
  - Job persistence (SQLite optional)
  - Batch job submission support

- **Configuration System**
  - YAML configuration file support
  - Environment variable support (GOVC_* variables)
  - CLI flag overrides
  - Flexible logging levels (debug, info, warn, error)

- **Terminal UI Features**
  - Beautiful ASCII art banner
  - Interactive VM selection
  - Real-time progress bars
  - Color-coded status messages
  - Capability detection display

- **Build & Deployment**
  - Makefile build system
  - RPM spec file for Fedora/RHEL/CentOS
  - Systemd service unit
  - Installation script
  - GitHub Actions CI/CD workflows

- **Documentation**
  - Comprehensive README with quick start
  - Getting Started guide
  - Project summary and architecture
  - API usage examples (curl, Python)
  - Example configuration files

### Technical Details

- **Language:** Go 1.24+
- **Dependencies:**
  - govmomi v0.52.0 (VMware SDK)
  - pterm v0.12.82 (Terminal UI)
  - progressbar v3.19.0 (Progress tracking)
  - gopkg.in/yaml.v3 (Config)

- **Architecture:**
  - Goroutine-based concurrency
  - Channel-based communication
  - Worker pool pattern for downloads
  - Context-based cancellation
  - Mutex-protected shared state

[0.2.0]: https://github.com/ssahani/hypersdk/releases/tag/v0.2.0
[0.1.0]: https://github.com/ssahani/hypersdk/releases/tag/v0.1.0
