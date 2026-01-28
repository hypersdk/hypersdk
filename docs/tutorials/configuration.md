# HyperSDK Configuration Guide

This guide covers all configuration options for HyperSDK across different deployment methods.

## Configuration Methods

HyperSDK supports three configuration methods (in priority order):

1. **Command-line flags** (highest priority)
2. **Environment variables**
3. **Configuration file** (lowest priority)

## Configuration File

### Location

- Docker/Podman: `/config/config.yaml`
- Kubernetes: Mounted from ConfigMap
- Binary: `./config.yaml` or specify with `--config`

### Basic Configuration

Create `config.yaml`:

```yaml
# Server configuration
server:
  addr: "0.0.0.0:8080"
  metrics_addr: "0.0.0.0:8081"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

# Logging
logging:
  level: info  # debug, info, warn, error
  format: json  # json or text
  output: stdout  # stdout, stderr, or file path

# Database
database:
  path: /data/hypersdk.db
  max_connections: 10
  connection_timeout: 5s
  busy_timeout: 5s

# Job processing
jobs:
  workers: 3
  queue_size: 100
  timeout: 24h
  retry_attempts: 3
  retry_delay: 5m

# Export settings
export:
  chunk_size: 33554432  # 32MB
  buffer_size: 1048576  # 1MB
  compression: true
  compression_level: 6  # 1-9
  verify_checksums: true

# Cache configuration
cache:
  enabled: true
  type: redis  # redis or memory
  ttl: 1h
  max_size: 1000

# Redis configuration (if cache.type=redis)
redis:
  addr: redis:6379
  password: ""
  db: 0
  pool_size: 10
  min_idle_conns: 5
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s
```

### Advanced Configuration

```yaml
# Rate limiting
rate_limit:
  enabled: true
  requests_per_second: 100
  burst: 200

# Timeouts
timeouts:
  api_request: 30s
  export_job: 24h
  provider_connect: 10s
  provider_operation: 5m

# Resource limits
limits:
  max_concurrent_exports: 5
  max_export_size: 1099511627776  # 1TB
  max_vm_disk_size: 549755813888  # 512GB

# Security
security:
  tls_enabled: false
  tls_cert_file: /config/tls/cert.pem
  tls_key_file: /config/tls/key.pem
  api_key_required: false
  api_keys:
    - key: "your-api-key-here"
      description: "Production API key"

# Monitoring
monitoring:
  metrics_enabled: true
  health_check_interval: 30s
  profiling_enabled: false
  profiling_addr: "localhost:6060"

# Audit logging
audit:
  enabled: true
  log_file: /var/log/hypersdk/audit.log
  max_size: 100  # MB
  max_age: 30    # days
  max_backups: 10
```

## Environment Variables

### Daemon Configuration

```bash
# Server
DAEMON_ADDR=0.0.0.0:8080
METRICS_ADDR=0.0.0.0:8081

# Logging
LOG_LEVEL=info           # debug, info, warn, error
LOG_FORMAT=json          # json or text

# Database
DATABASE_PATH=/data/hypersdk.db
DATABASE_MAX_CONNECTIONS=10

# Workers
DOWNLOAD_WORKERS=3
CHUNK_SIZE=33554432      # 32MB
RETRY_ATTEMPTS=3
```

### Cloud Provider Credentials

#### vSphere

```bash
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=administrator@vsphere.local
GOVC_PASSWORD=your-password
GOVC_INSECURE=1                    # Allow self-signed certs
GOVC_DATACENTER=Datacenter1        # Optional: default datacenter
GOVC_DATASTORE=datastore1          # Optional: default datastore
GOVC_RESOURCE_POOL=/Resources      # Optional: resource pool
```

#### AWS

```bash
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
AWS_REGION=us-east-1
AWS_SESSION_TOKEN=...              # Optional: for temporary credentials
```

#### Azure

```bash
AZURE_SUBSCRIPTION_ID=00000000-0000-0000-0000-000000000000
AZURE_TENANT_ID=00000000-0000-0000-0000-000000000000
AZURE_CLIENT_ID=00000000-0000-0000-0000-000000000000
AZURE_CLIENT_SECRET=your-client-secret
AZURE_RESOURCE_GROUP=my-resource-group
```

#### Google Cloud Platform

```bash
GOOGLE_APPLICATION_CREDENTIALS=/config/gcp-service-account.json
GCP_PROJECT_ID=my-project-id
GCP_ZONE=us-central1-a            # Optional: default zone
```

#### Hyper-V

```bash
HYPERV_HOST=hyperv-host.example.com
HYPERV_USERNAME=Administrator
HYPERV_PASSWORD=your-password
HYPERV_PORT=5985                  # Default: 5985 (HTTP) or 5986 (HTTPS)
HYPERV_USE_SSL=false
```

#### Oracle Cloud Infrastructure (OCI)

```bash
OCI_TENANCY_OCID=ocid1.tenancy.oc1..example
OCI_USER_OCID=ocid1.user.oc1..example
OCI_FINGERPRINT=aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99
OCI_KEY_FILE=/config/oci-api-key.pem
OCI_REGION=us-ashburn-1
```

#### OpenStack

```bash
OS_AUTH_URL=https://openstack.example.com:5000/v3
OS_USERNAME=admin
OS_PASSWORD=your-password
OS_PROJECT_NAME=admin
OS_USER_DOMAIN_NAME=Default
OS_PROJECT_DOMAIN_NAME=Default
OS_REGION_NAME=RegionOne
```

#### Alibaba Cloud

```bash
ALIBABA_CLOUD_ACCESS_KEY_ID=LTAI4G...
ALIBABA_CLOUD_ACCESS_KEY_SECRET=your-secret
ALIBABA_CLOUD_REGION_ID=cn-hangzhou
```

#### Proxmox

```bash
PROXMOX_URL=https://proxmox.example.com:8006/api2/json
PROXMOX_USERNAME=root@pam
PROXMOX_PASSWORD=your-password
PROXMOX_NODE=pve                  # Default node
PROXMOX_INSECURE=1                # Allow self-signed certs
```

### Redis Configuration

```bash
REDIS_ENABLED=true
REDIS_ADDR=redis:6379
REDIS_PASSWORD=your-redis-password
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MAX_MEMORY=256mb
```

## Docker Configuration

### Docker Compose Environment File

Create `.env` in `deployments/docker/`:

```bash
# Service version
VERSION=latest

# Daemon ports
DAEMON_PORT=8080
METRICS_PORT=8081

# Daemon configuration
LOG_LEVEL=info
DATABASE_PATH=/data/hypersdk.db
DOWNLOAD_WORKERS=3
CHUNK_SIZE=33554432
RETRY_ATTEMPTS=3

# Redis
REDIS_ENABLED=true
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_MAX_MEMORY=256mb

# vSphere (configure your credentials)
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=administrator@vsphere.local
GOVC_PASSWORD=your-password
GOVC_INSECURE=1

# Monitoring
PROMETHEUS_PORT=9090
PROMETHEUS_RETENTION=15d
GRAFANA_PORT=3000
GRAFANA_USER=admin
GRAFANA_PASSWORD=admin

# Optional: AWS
AWS_ACCESS_KEY_ID=
AWS_SECRET_ACCESS_KEY=
AWS_REGION=us-east-1

# Optional: Azure
AZURE_SUBSCRIPTION_ID=
AZURE_TENANT_ID=
AZURE_CLIENT_ID=
AZURE_CLIENT_SECRET=

# Optional: GCP
GOOGLE_APPLICATION_CREDENTIALS=/config/gcp-credentials.json
GOOGLE_CREDENTIALS_FILE=./config/gcp-credentials.json
GCP_PROJECT_ID=

# Storage
EXPORTS_PATH=./exports
```

### Volume Mounts

Map local directories to container paths:

```yaml
volumes:
  # Configuration files
  - ./config:/config:ro

  # TLS certificates
  - ./certs:/config/tls:ro

  # GCP credentials
  - ./gcp-credentials.json:/config/gcp-credentials.json:ro

  # Custom export location
  - /mnt/storage/exports:/exports

  # Database backup
  - ./backups:/backups
```

## Kubernetes Configuration

### ConfigMap

Create `configmap.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: hypervisord-config
  namespace: hypersdk
data:
  config.yaml: |
    server:
      addr: "0.0.0.0:8080"
      metrics_addr: "0.0.0.0:8081"

    logging:
      level: info
      format: json

    database:
      path: /data/hypersdk.db

    jobs:
      workers: 3
      queue_size: 100

    export:
      chunk_size: 33554432
      compression: true

    redis:
      addr: redis:6379
      db: 0
```

### Secrets

Create `secrets.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cloud-credentials
  namespace: hypersdk
type: Opaque
stringData:
  # vSphere
  vsphere-url: "https://vcenter.example.com/sdk"
  vsphere-username: "administrator@vsphere.local"
  vsphere-password: "your-password"

  # AWS
  aws-access-key-id: "AKIAIOSFODNN7EXAMPLE"
  aws-secret-access-key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  aws-region: "us-east-1"

  # Azure
  azure-subscription-id: "00000000-0000-0000-0000-000000000000"
  azure-tenant-id: "00000000-0000-0000-0000-000000000000"
  azure-client-id: "00000000-0000-0000-0000-000000000000"
  azure-client-secret: "your-client-secret"
```

### Environment Variables in Deployment

Reference secrets in `deployment.yaml`:

```yaml
env:
  # From ConfigMap
  - name: LOG_LEVEL
    valueFrom:
      configMapKeyRef:
        name: hypervisord-config
        key: log_level

  # From Secret
  - name: GOVC_URL
    valueFrom:
      secretKeyRef:
        name: cloud-credentials
        key: vsphere-url

  - name: GOVC_USERNAME
    valueFrom:
      secretKeyRef:
        name: cloud-credentials
        key: vsphere-username

  - name: GOVC_PASSWORD
    valueFrom:
      secretKeyRef:
        name: cloud-credentials
        key: vsphere-password
```

## Performance Tuning

### Worker Configuration

Adjust based on available resources:

```bash
# Conservative (low resources)
DOWNLOAD_WORKERS=1
CHUNK_SIZE=16777216  # 16MB

# Moderate (4GB RAM, 2 CPUs)
DOWNLOAD_WORKERS=3
CHUNK_SIZE=33554432  # 32MB

# Aggressive (8GB+ RAM, 4+ CPUs)
DOWNLOAD_WORKERS=5
CHUNK_SIZE=67108864  # 64MB
```

### Database Optimization

```yaml
database:
  max_connections: 10
  busy_timeout: 5s
  # SQLite PRAGMA settings
  pragma:
    journal_mode: WAL          # Write-Ahead Logging
    synchronous: NORMAL        # Balance performance/durability
    cache_size: -64000         # 64MB cache
    temp_store: MEMORY         # In-memory temp tables
```

### Redis Tuning

```bash
# Memory
REDIS_MAX_MEMORY=512mb
REDIS_MAX_MEMORY_POLICY=allkeys-lru

# Connection pool
REDIS_POOL_SIZE=20
REDIS_MIN_IDLE_CONNS=5

# Persistence
REDIS_SAVE="60 1000"           # Save every 60s if 1000+ keys changed
REDIS_APPENDFSYNC=everysec     # AOF sync every second
```

## Security Configuration

### Enable TLS

Docker:
```bash
docker run -d \
  -e TLS_ENABLED=true \
  -v ./certs/cert.pem:/config/tls/cert.pem:ro \
  -v ./certs/key.pem:/config/tls/key.pem:ro \
  -p 8443:8080 \
  ghcr.io/ssahani/hypersdk-hypervisord:latest
```

Configuration file:
```yaml
security:
  tls_enabled: true
  tls_cert_file: /config/tls/cert.pem
  tls_key_file: /config/tls/key.pem
  tls_min_version: "1.2"
  tls_cipher_suites:
    - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
    - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
```

### API Key Authentication

Enable API keys:

```yaml
security:
  api_key_required: true
  api_keys:
    - key: "prod-key-abc123"
      description: "Production API key"
      rate_limit: 1000

    - key: "dev-key-xyz789"
      description: "Development API key"
      rate_limit: 100
```

Use API key in requests:
```bash
curl -H "X-API-Key: prod-key-abc123" http://localhost:8080/status
```

### Network Policies (Kubernetes)

Restrict traffic in production:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hypervisord-policy
  namespace: hypersdk
spec:
  podSelector:
    matchLabels:
      app: hypervisord
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

## Monitoring Configuration

### Prometheus Metrics

Configure scrape targets in `prometheus.yaml`:

```yaml
scrape_configs:
  - job_name: 'hypersdk'
    scrape_interval: 15s
    static_configs:
      - targets: ['hypervisord:8081']
        labels:
          environment: 'production'
```

### Grafana Data Source

Provisioned in `deployments/docker/config/grafana/provisioning/datasources/prometheus.yaml`:

```yaml
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

## Validation

### Test Configuration

```bash
# Docker - dry run
docker run --rm \
  --env-file hypersdk.env \
  ghcr.io/ssahani/hypersdk-hypervisord:latest \
  --config /config/config.yaml --validate

# Check configuration syntax
hypervisord --config config.yaml --validate
```

### Environment Variable Debug

```bash
# Print all HyperSDK environment variables
docker exec hypersdk env | grep -E '(GOVC|AWS|AZURE|LOG|DAEMON)' | sort

# Kubernetes
kubectl exec -n hypersdk deployment/hypervisord -- env | grep GOVC
```

## Examples

See complete configuration examples:
- [Docker Compose Example](../../deployments/docker/docker-compose.yml)
- [Kubernetes Development](../../deployments/kubernetes/overlays/development/)
- [Kubernetes Production](../../deployments/kubernetes/overlays/production/)

## Next Steps

- [Multi-Cloud Setup](multi-cloud-setup.md) - Configure multiple providers
- [Monitoring Guide](monitoring.md) - Set up alerts and dashboards
- [Production Deployment](production-deployment.md) - Production best practices
