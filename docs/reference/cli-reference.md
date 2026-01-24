# CLI Reference

Complete command-line reference for HyperSDK tools.

## Table of Contents

1. [hyperexport](#hyperexport)
2. [hypervisord](#hypervisord)
3. [hyperctl](#hyperctl)
4. [Environment Variables](#environment-variables)

## hyperexport

Standalone VM export tool with interactive and non-interactive modes.

### Synopsis

```bash
hyperexport [flags]
hyperexport -vm VM_PATH [options]
hyperexport -batch VM_LIST_FILE [options]
hyperexport -interactive
```

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-h, --help` | bool | false | Show help message |
| `-version` | bool | false | Show version and exit |
| `-config` | string | | Path to config file |

### Connection Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-url` | string | $GOVC_URL | vCenter SDK URL |
| `-username` | string | $GOVC_USERNAME | vCenter username |
| `-password` | string | $GOVC_PASSWORD | vCenter password |
| `-insecure` | bool | false | Skip SSL verification |
| `-datacenter` | string | | Datacenter name |

### Export Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-vm` | string | | VM name or path to export |
| `-batch` | string | | File with VM list (one per line) |
| `-output` | string | `./export-<vmname>` | Output directory |
| `-format` | string | ovf | Export format: ovf or ova |
| `-compress` | bool | false | Enable compression for OVA |
| `-compression-level` | int | 6 | Compression level (1-9) |
| `-verify` | bool | false | Verify export with checksums |
| `-parallel` | int | 4 | Number of parallel downloads |

### Workflow Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-dry-run` | bool | false | Preview without executing |
| `-power-off` | bool | false | Auto power off VM before export |
| `-quiet` | bool | false | Minimal output for scripting |
| `-resume` | bool | false | Resume interrupted export |

### Advanced Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-folder` | string | | Filter VMs by folder path |
| `-filter` | string | | Filter VMs by tag (key=value) |
| `-interactive, -tui` | bool | false | Launch interactive TUI mode |
| `-validate-only` | bool | false | Only run validation checks |
| `-history` | bool | false | Show export history |
| `-history-limit` | int | 10 | Number of history entries |
| `-report` | bool | false | Generate statistics report |
| `-report-file` | string | stdout | Save report to file |
| `-clear-history` | bool | false | Clear export history |

### Cloud Upload Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-upload` | string | | Upload to cloud (s3://, azure://, gs://, sftp://) |
| `-stream-upload` | bool | false | Stream directly to cloud |
| `-keep-local` | bool | true | Keep local copy after upload |

### Encryption Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-encrypt` | bool | false | Encrypt export files |
| `-encrypt-method` | string | aes256 | Encryption: aes256 or gpg |
| `-passphrase` | string | | Encryption passphrase |
| `-keyfile` | string | | Encryption key file |
| `-gpg-recipient` | string | | GPG recipient for encryption |

### Profile Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-profile` | string | | Use saved export profile |
| `-save-profile` | string | | Save current settings as profile |
| `-list-profiles` | bool | false | List available profiles |
| `-delete-profile` | string | | Delete a saved profile |
| `-create-default-profiles` | bool | false | Create default profiles |

### Examples

```bash
# Basic export
hyperexport -vm "/datacenter/vm/web-server-01"

# Export as compressed OVA with verification
hyperexport -vm myvm -format ova -compress -verify

# Batch export from file
hyperexport -batch vms.txt -format ova -compress -output /exports

# Dry run preview
hyperexport -vm myvm -dry-run

# With graceful shutdown
hyperexport -vm myvm -power-off -format ova

# Quiet mode for scripts
hyperexport -vm myvm -format ova -compress -verify -quiet

# Interactive TUI mode
hyperexport -interactive

# Filter by folder
hyperexport -folder /Production/WebServers

# Export history
hyperexport -history
hyperexport -report

# Cloud upload
hyperexport -vm myvm -upload s3://bucket/exports/ -keep-local=false

# Encrypted export
hyperexport -vm myvm -encrypt -passphrase "secret"

# Using profiles
hyperexport -list-profiles
hyperexport -vm myvm -profile production-backup
hyperexport -vm myvm -format ova -compress -save-profile my-backup
```

## hypervisord

Background daemon service providing REST API and job management.

### Synopsis

```bash
hypervisord [flags]
```

### Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-h, --help` | bool | false | Show help message |
| `-version` | bool | false | Show version and exit |
| `-config` | string | /etc/hypervisord/config.yaml | Path to config file |
| `-addr` | string | 0.0.0.0:8080 | Listen address |
| `-log-level` | string | info | Log level: debug, info, warn, error |
| `-validate` | bool | false | Validate config and exit |
| `-disable-web` | bool | false | Disable web dashboard |

### Examples

```bash
# Start with default config
hypervisord

# Custom config file
hypervisord --config /etc/hypervisord/custom.yaml

# Custom listen address
hypervisord --addr localhost:9000

# Debug logging
hypervisord --log-level debug

# Validate configuration
hypervisord --config /etc/hypervisord/config.yaml --validate

# API-only mode (no web dashboard)
hypervisord --disable-web

# Show version
hypervisord -version
```

### Systemd Service

```bash
# Start daemon
sudo systemctl start hypervisord

# Stop daemon
sudo systemctl stop hypervisord

# Restart daemon
sudo systemctl restart hypervisord

# Enable on boot
sudo systemctl enable hypervisord

# View status
sudo systemctl status hypervisord

# View logs
sudo journalctl -u hypervisord -f
```

## hyperctl

Command-line interface for controlling the daemon and managing jobs.

### Synopsis

```bash
hyperctl [command] [flags]
```

### Global Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-h, --help` | bool | false | Show help message |
| `-daemon-url` | string | http://localhost:8080 | Daemon API URL |
| `-output` | string | table | Output format: table, json, yaml |
| `-quiet` | bool | false | Minimal output |

### Commands

#### status

Show daemon status.

```bash
hyperctl status

# JSON output
hyperctl status --output json
```

#### submit

Submit export job to daemon.

```bash
hyperctl submit [flags]
```

Flags:

| Flag | Type | Description |
|------|------|-------------|
| `-vm` | string | VM path to export |
| `-output` | string | Output directory |
| `-format` | string | Export format (ovf/ova) |
| `-compress` | bool | Enable compression |
| `-file` | string | Job definition file (YAML) |

Examples:

```bash
# Submit single VM export
hyperctl submit -vm "/datacenter/vm/web-01" -output /exports/web-01

# Submit with options
hyperctl submit -vm myvm -format ova -compress

# Submit from YAML file
hyperctl submit -file job.yaml
```

#### query

Query job status.

```bash
hyperctl query [flags]
```

Flags:

| Flag | Type | Description |
|------|------|-------------|
| `-id` | string | Job ID to query |
| `-all` | bool | Query all jobs |
| `-status` | string | Filter by status |

Examples:

```bash
# Query specific job
hyperctl query -id job-12345

# Query all jobs
hyperctl query -all

# Filter by status
hyperctl query -all -status running

# JSON output
hyperctl query -all --output json
```

#### cancel

Cancel running job.

```bash
hyperctl cancel -id JOB_ID
```

Examples:

```bash
# Cancel job
hyperctl cancel -id job-12345

# Force cancel
hyperctl cancel -id job-12345 -force
```

#### logs

View job logs.

```bash
hyperctl logs -id JOB_ID [flags]
```

Flags:

| Flag | Type | Description |
|------|------|-------------|
| `-follow, -f` | bool | Follow log output |
| `-tail` | int | Number of lines to show |

Examples:

```bash
# View job logs
hyperctl logs -id job-12345

# Follow logs
hyperctl logs -id job-12345 -f

# Last 100 lines
hyperctl logs -id job-12345 -tail 100
```

#### list-vms

List VMs from vSphere.

```bash
hyperctl list-vms [flags]
```

Flags:

| Flag | Type | Description |
|------|------|-------------|
| `-folder` | string | Filter by folder |
| `-datacenter` | string | Filter by datacenter |

Examples:

```bash
# List all VMs
hyperctl list-vms

# Filter by folder
hyperctl list-vms -folder /Production

# JSON output
hyperctl list-vms --output json
```

#### schedules

Manage scheduled jobs.

```bash
# List schedules
hyperctl schedules list

# Create schedule
hyperctl schedules create -file schedule.yaml

# Delete schedule
hyperctl schedules delete -id schedule-12345

# Enable/disable schedule
hyperctl schedules enable -id schedule-12345
hyperctl schedules disable -id schedule-12345
```

#### webhooks

Manage webhooks.

```bash
# List webhooks
hyperctl webhooks list

# Add webhook
hyperctl webhooks add -url https://hooks.example.com/hypersdk

# Delete webhook
hyperctl webhooks delete -id webhook-12345

# Test webhook
hyperctl webhooks test -id webhook-12345
```

### Interactive TUI Mode

Launch interactive terminal UI:

```bash
hyperctl

# Or explicit
hyperctl --interactive
```

### Examples

```bash
# Check daemon status
hyperctl status

# Submit export job
hyperctl submit -vm "/dc/vm/web-01" -output /exports -format ova

# Monitor job progress
hyperctl query -id job-12345

# Follow job logs
hyperctl logs -id job-12345 -f

# List all jobs
hyperctl query -all

# Cancel job
hyperctl cancel -id job-12345

# List VMs
hyperctl list-vms

# Interactive mode
hyperctl
```

## Environment Variables

### vSphere Connection

```bash
export GOVC_URL="https://vcenter.example.com/sdk"
export GOVC_USERNAME="administrator@vsphere.local"
export GOVC_PASSWORD="password"
export GOVC_INSECURE="1"  # Skip SSL verification
export GOVC_DATACENTER="Datacenter"
```

### AWS Configuration

```bash
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="AKIA..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_S3_BUCKET="vm-exports"
```

### Azure Configuration

```bash
export AZURE_SUBSCRIPTION_ID="..."
export AZURE_TENANT_ID="..."
export AZURE_CLIENT_ID="..."
export AZURE_CLIENT_SECRET="..."
```

### GCP Configuration

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export GCP_PROJECT_ID="my-project"
export GCP_BUCKET="vm-exports"
```

### Daemon Configuration

```bash
export DAEMON_ADDR="0.0.0.0:8080"
export LOG_LEVEL="info"
export DOWNLOAD_WORKERS="4"
export MAX_CONCURRENT_JOBS="10"
```

### Export Configuration

```bash
export EXPORT_OUTPUT_DIR="/var/lib/hypersdk/exports"
export EXPORT_FORMAT="ova"
export EXPORT_COMPRESS="true"
export EXPORT_VERIFY="true"
export PARALLEL_DOWNLOADS="8"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Connection error |
| 3 | Authentication error |
| 4 | VM not found |
| 5 | Export failed |
| 6 | Validation failed |
| 7 | Configuration error |

## See Also

- [Getting Started Guide](getting-started.md)
- [Configuration Reference](configuration-reference.md)
- [API Reference](API_ENDPOINTS.md)
- [Migration Workflows](migration-workflows.md)
