# HyperSDK Configuration Reference

## Table of Contents

1. [Configuration File Location](#configuration-file-location)
2. [Configuration File Format](#configuration-file-format)
3. [vSphere Configuration](#vsphere-configuration)
4. [AWS Configuration](#aws-configuration)
5. [Azure Configuration](#azure-configuration)
6. [GCP Configuration](#gcp-configuration)
7. [Daemon Configuration](#daemon-configuration)
8. [Export Configuration](#export-configuration)
9. [Connection Pool Configuration](#connection-pool-configuration)
10. [Webhook Configuration](#webhook-configuration)
11. [Scheduler Configuration](#scheduler-configuration)
12. [Web Dashboard Configuration](#web-dashboard-configuration)
13. [Logging Configuration](#logging-configuration)
14. [Environment Variables](#environment-variables)

## Configuration File Location

Default locations (searched in order):

1. `/etc/hypervisord/config.yaml` (system-wide)
2. `~/.config/hypersdk/config.yaml` (user-specific)
3. `./config.yaml` (current directory)
4. Custom path via `--config` flag

## Configuration File Format

HyperSDK uses YAML format for configuration. Complete example:

```yaml
# vSphere Provider
vsphere:
  url: "https://vcenter.example.com/sdk"
  username: "administrator@vsphere.local"
  password: "${VCENTER_PASSWORD}"  # Environment variable
  insecure: false
  datacenter: ""  # Optional, filter by datacenter
  folder: ""      # Optional, filter by folder

# AWS Provider
aws:
  region: "us-east-1"
  access_key_id: "${AWS_ACCESS_KEY_ID}"
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"
  s3_bucket: "vm-exports"
  export_format: "vmdk"  # vmdk, vhd, raw

# Azure Provider
azure:
  subscription_id: "${AZURE_SUBSCRIPTION_ID}"
  tenant_id: "${AZURE_TENANT_ID}"
  client_id: "${AZURE_CLIENT_ID}"
  client_secret: "${AZURE_CLIENT_SECRET}"
  resource_group: "vm-exports"
  storage_account: "vmexports"
  container: "exports"

# GCP Provider
gcp:
  project_id: "my-project"
  credentials_file: "/path/to/service-account.json"
  bucket: "vm-exports"
  region: "us-central1"

# Daemon Settings
daemon:
  addr: "0.0.0.0:8080"
  log_level: "info"  # debug, info, warn, error
  download_workers: 4
  max_concurrent_jobs: 10
  job_timeout: "24h"
  enable_profiling: false
  profiling_addr: "localhost:6060"

# Export Settings
export:
  output_dir: "/var/lib/hypersdk/exports"
  default_format: "ova"  # ovf, ova
  compress: true
  compression_level: 6  # 1-9, higher = more compression
  verify_checksums: true
  parallel_downloads: 4
  resume_enabled: true
  keep_partial: false

# Connection Pool
connection_pool:
  max_connections: 5
  idle_timeout: "5m"
  max_lifetime: "1h"
  health_check_interval: "30s"
  health_check_timeout: "10s"

# Webhooks
webhooks:
  - name: "slack-notifications"
    url: "https://hooks.slack.com/services/XXX/YYY/ZZZ"
    events:
      - "job.started"
      - "job.completed"
      - "job.failed"
    headers:
      Content-Type: "application/json"
    timeout: "10s"
    retry: 3
    retry_delay: "5s"
    enabled: true

  - name: "monitoring"
    url: "https://monitoring.example.com/webhook"
    events: ["*"]  # All events
    enabled: true

# Job Scheduler
scheduler:
  enabled: true
  timezone: "UTC"
  max_concurrent_scheduled_jobs: 5

# Web Dashboard
web:
  enabled: true
  static_dir: "/usr/share/hypersdk/web"
  listen_addr: "0.0.0.0:8080"
  tls_enabled: false
  tls_cert: "/etc/hypersdk/tls/cert.pem"
  tls_key: "/etc/hypersdk/tls/key.pem"
  cors_enabled: true
  cors_origins:
    - "http://localhost:3000"
    - "https://dashboard.example.com"

# Logging
logging:
  level: "info"
  format: "json"  # json, text
  output: "stdout"  # stdout, file
  file_path: "/var/log/hypersdk/hypervisord.log"
  max_size_mb: 100
  max_backups: 10
  max_age_days: 30
  compress: true

# Database (for scheduler persistence)
database:
  type: "sqlite"  # sqlite, postgres
  path: "/var/lib/hypersdk/hypersdk.db"
  # For PostgreSQL:
  # host: "localhost"
  # port: 5432
  # username: "hypersdk"
  # password: "${DB_PASSWORD}"
  # database: "hypersdk"

# Metrics
metrics:
  enabled: true
  prometheus_enabled: true
  prometheus_path: "/metrics"
```

## vSphere Configuration

### Basic Authentication

```yaml
vsphere:
  url: "https://vcenter.example.com/sdk"
  username: "administrator@vsphere.local"
  password: "SecurePassword123"
  insecure: false  # Verify SSL certificates
```

### Self-Signed Certificates

```yaml
vsphere:
  url: "https://vcenter.example.com/sdk"
  username: "administrator@vsphere.local"
  password: "SecurePassword123"
  insecure: true  # Skip SSL verification (not recommended)
```

### Domain Users

```yaml
vsphere:
  # Active Directory user
  url: "https://vcenter.example.com/sdk"
  username: "DOMAIN\\username"
  password: "password"

  # Or with @ notation
  username: "username@domain.com"
```

### Filtering Options

```yaml
vsphere:
  url: "https://vcenter.example.com/sdk"
  username: "administrator@vsphere.local"
  password: "password"
  datacenter: "Production-DC"  # Only this datacenter
  folder: "/Production/WebServers"  # Only this folder
  cluster: "Cluster-01"  # Only this cluster
```

## AWS Configuration

### Basic Configuration

```yaml
aws:
  region: "us-east-1"
  access_key_id: "AKIAIOSFODNN7EXAMPLE"
  secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

### S3 Export Configuration

```yaml
aws:
  region: "us-west-2"
  access_key_id: "${AWS_ACCESS_KEY_ID}"
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"
  s3_bucket: "my-vm-exports"
  s3_prefix: "exports/"
  export_format: "vmdk"  # vmdk, vhd, raw
  kms_key_id: "arn:aws:kms:us-west-2:123456789012:key/xxx"  # Optional encryption
```

### IAM Role (EC2 Instance)

```yaml
aws:
  region: "us-east-1"
  # No credentials needed, uses IAM instance role
  s3_bucket: "vm-exports"
```

## Azure Configuration

### Service Principal Authentication

```yaml
azure:
  subscription_id: "12345678-1234-1234-1234-123456789012"
  tenant_id: "87654321-4321-4321-4321-210987654321"
  client_id: "abcdef12-3456-7890-abcd-ef1234567890"
  client_secret: "your-client-secret"
  resource_group: "vm-exports-rg"
  storage_account: "vmexportstorage"
  container: "vhd-exports"
```

### Managed Identity (Azure VM)

```yaml
azure:
  subscription_id: "12345678-1234-1234-1234-123456789012"
  resource_group: "vm-exports-rg"
  # No credentials needed, uses managed identity
```

## GCP Configuration

### Service Account

```yaml
gcp:
  project_id: "my-gcp-project"
  credentials_file: "/etc/hypersdk/gcp-service-account.json"
  bucket: "vm-exports-bucket"
  region: "us-central1"
  export_format: "vmdk"  # vmdk, vhdx, qcow2
```

### Application Default Credentials

```yaml
gcp:
  project_id: "my-gcp-project"
  # Uses GOOGLE_APPLICATION_CREDENTIALS env var
  bucket: "vm-exports"
```

## Daemon Configuration

### Basic Settings

```yaml
daemon:
  addr: "0.0.0.0:8080"  # Listen address
  log_level: "info"
  download_workers: 4
  max_concurrent_jobs: 10
```

### Advanced Settings

```yaml
daemon:
  addr: "0.0.0.0:8080"
  log_level: "debug"
  download_workers: 8
  max_concurrent_jobs: 20
  job_timeout: "48h"
  graceful_shutdown_timeout: "30s"
  enable_profiling: true
  profiling_addr: "localhost:6060"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
```

## Export Configuration

### Default Export Settings

```yaml
export:
  output_dir: "/var/lib/hypersdk/exports"
  default_format: "ova"
  compress: true
  compression_level: 6
  verify_checksums: true
```

### Advanced Export Settings

```yaml
export:
  output_dir: "/exports"
  default_format: "ova"
  compress: true
  compression_level: 9  # Maximum compression
  verify_checksums: true
  parallel_downloads: 8
  resume_enabled: true
  keep_partial: false  # Delete partial downloads on failure
  buffer_size: 8388608  # 8MB buffer
  checksum_algorithm: "sha256"  # sha256, sha512, md5

  # Naming template
  naming_template: "{{.VMName}}-{{.Date}}"  # VM-2024-01-20

  # Auto cleanup
  cleanup_on_completion: false
  cleanup_after_days: 7
```

## Connection Pool Configuration

```yaml
connection_pool:
  max_connections: 10  # Max concurrent vSphere connections
  min_connections: 2   # Keep at least 2 warm connections
  idle_timeout: "10m"  # Close idle connections after 10 min
  max_lifetime: "2h"   # Reconnect after 2 hours
  health_check_interval: "1m"
  health_check_timeout: "15s"
  connection_timeout: "30s"
```

## Webhook Configuration

### Basic Webhook

```yaml
webhooks:
  - url: "https://hooks.example.com/hypersdk"
    events: ["job.completed", "job.failed"]
    enabled: true
```

### Advanced Webhook with Authentication

```yaml
webhooks:
  - name: "production-alerts"
    url: "https://alerts.example.com/webhook"
    events:
      - "job.started"
      - "job.completed"
      - "job.failed"
      - "job.cancelled"
    headers:
      Authorization: "Bearer ${WEBHOOK_TOKEN}"
      Content-Type: "application/json"
      X-Service: "hypersdk"
    timeout: "15s"
    retry: 5
    retry_delay: "10s"
    retry_backoff: "exponential"  # exponential, linear
    enabled: true
    filter:
      # Only trigger for production VMs
      vm_path_regex: ".*/Production/.*"
```

### Multiple Webhooks

```yaml
webhooks:
  - name: "slack"
    url: "https://hooks.slack.com/services/XXX"
    events: ["job.failed"]
    enabled: true

  - name: "email"
    url: "https://api.example.com/email"
    events: ["job.completed"]
    enabled: true

  - name: "monitoring"
    url: "https://prometheus-pushgateway/metrics"
    events: ["*"]  # All events
    enabled: true
```

## Scheduler Configuration

### Basic Scheduler

```yaml
scheduler:
  enabled: true
  timezone: "America/New_York"
```

### Advanced Scheduler

```yaml
scheduler:
  enabled: true
  timezone: "UTC"
  max_concurrent_scheduled_jobs: 10
  missed_job_handling: "skip"  # skip, run, queue
  overlap_handling: "skip"     # skip, queue, run
  persistence_enabled: true
  persistence_interval: "1m"
```

## Web Dashboard Configuration

### HTTP Dashboard

```yaml
web:
  enabled: true
  static_dir: "/usr/share/hypersdk/web"
  listen_addr: "0.0.0.0:8080"
```

### HTTPS Dashboard

```yaml
web:
  enabled: true
  static_dir: "/usr/share/hypersdk/web"
  listen_addr: "0.0.0.0:8443"
  tls_enabled: true
  tls_cert: "/etc/hypersdk/tls/fullchain.pem"
  tls_key: "/etc/hypersdk/tls/privkey.pem"
  tls_min_version: "1.2"
  tls_ciphers:
    - "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
    - "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
```

### CORS Configuration

```yaml
web:
  enabled: true
  cors_enabled: true
  cors_origins:
    - "https://dashboard.example.com"
    - "http://localhost:3000"
  cors_methods:
    - "GET"
    - "POST"
    - "PUT"
    - "DELETE"
  cors_headers:
    - "Authorization"
    - "Content-Type"
  cors_credentials: true
```

## Logging Configuration

### Console Logging

```yaml
logging:
  level: "info"
  format: "text"
  output: "stdout"
  colored: true  # Colorize output
```

### File Logging with Rotation

```yaml
logging:
  level: "debug"
  format: "json"
  output: "file"
  file_path: "/var/log/hypersdk/hypervisord.log"
  max_size_mb: 100    # Rotate after 100MB
  max_backups: 10     # Keep 10 old files
  max_age_days: 30    # Delete files older than 30 days
  compress: true      # Compress rotated files
```

### Multiple Outputs

```yaml
logging:
  level: "info"
  outputs:
    - type: "stdout"
      format: "text"
      colored: true

    - type: "file"
      format: "json"
      path: "/var/log/hypersdk/app.log"
      max_size_mb: 100

    - type: "syslog"
      network: "udp"
      address: "localhost:514"
      tag: "hypersdk"
```

## Environment Variables

All configuration options can be overridden with environment variables:

### vSphere Variables

```bash
export GOVC_URL="https://vcenter.example.com/sdk"
export GOVC_USERNAME="administrator@vsphere.local"
export GOVC_PASSWORD="password"
export GOVC_INSECURE="1"
```

### AWS Variables

```bash
export AWS_REGION="us-east-1"
export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export AWS_S3_BUCKET="vm-exports"
```

### Azure Variables

```bash
export AZURE_SUBSCRIPTION_ID="12345678-1234-1234-1234-123456789012"
export AZURE_TENANT_ID="87654321-4321-4321-4321-210987654321"
export AZURE_CLIENT_ID="abcdef12-3456-7890-abcd-ef1234567890"
export AZURE_CLIENT_SECRET="your-client-secret"
```

### GCP Variables

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export GCP_PROJECT_ID="my-project"
export GCP_BUCKET="vm-exports"
```

### Daemon Variables

```bash
export DAEMON_ADDR="0.0.0.0:8080"
export LOG_LEVEL="info"
export DOWNLOAD_WORKERS="4"
export MAX_CONCURRENT_JOBS="10"
```

## Configuration Validation

Validate configuration before starting:

```bash
# Dry-run to validate config
hypervisord --config /etc/hypervisord/config.yaml --validate

# Check config syntax
yamllint /etc/hypervisord/config.yaml
```

## Configuration Examples

### Minimal Configuration

```yaml
vsphere:
  url: "https://vcenter.example.com/sdk"
  username: "admin@vsphere.local"
  password: "password"
  insecure: true

daemon:
  addr: "localhost:8080"
```

### Production Configuration

```yaml
vsphere:
  url: "https://vcenter.prod.example.com/sdk"
  username: "${VCENTER_USER}"
  password: "${VCENTER_PASSWORD}"
  insecure: false

daemon:
  addr: "0.0.0.0:8080"
  log_level: "info"
  max_concurrent_jobs: 20

export:
  output_dir: "/exports"
  default_format: "ova"
  compress: true
  verify_checksums: true

connection_pool:
  max_connections: 10
  idle_timeout: "10m"

webhooks:
  - url: "${SLACK_WEBHOOK_URL}"
    events: ["job.failed"]
    enabled: true

web:
  enabled: true
  tls_enabled: true
  tls_cert: "/etc/hypersdk/tls/cert.pem"
  tls_key: "/etc/hypersdk/tls/key.pem"

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "/var/log/hypersdk/app.log"
  max_size_mb: 100
  max_backups: 10
```

## See Also

- [Installation Guide](installation-guide.md)
- [API Reference](API_ENDPOINTS.md)
- [User Guides](user-guides/)
- [Webhook Reference](webhook-reference.md)
