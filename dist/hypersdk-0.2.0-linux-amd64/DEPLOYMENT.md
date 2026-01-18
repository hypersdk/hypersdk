# HyperSDK Deployment Guide

**Version**: 0.2.0
**Date**: 2026-01-20
**Status**: Production Ready

---

## 🎯 Overview

This deployment guide covers the complete installation and configuration of HyperSDK with all Phase 1-4 features:

- ✅ **Phase 1**: Connection Pooling, Webhook Integration
- ✅ **Phase 2**: OVA Format, Compression, Schedule Persistence
- ✅ **Phase 3**: Unified Provider Interface
- ✅ **Phase 4**: Multi-Cloud Support (AWS, Azure, GCP, Hyper-V)

---

## 📦 Package Contents

### Binaries
- `bin/hypervisord` - Main daemon server (29MB)
- `bin/hyperctl` - CLI management tool (20MB)
- `bin/hyperexport` - Standalone export utility (20MB)

### Configuration
- `config.example.yaml` - Complete configuration template
- `config.yaml` - Your deployment config (create from example)

### Documentation
- `README.md` - Project overview
- `DEPLOYMENT.md` - This file
- `MULTI_CLOUD_GUIDE.md` - Multi-cloud provider setup
- `API_REFERENCE.md` - Complete API documentation

---

## 🚀 Quick Start (5 Minutes)

### 1. Copy Example Configuration
```bash
cp config.example.yaml config.yaml
```

### 2. Configure vCenter (Required)
Edit `config.yaml`:
```yaml
vcenter_url: "https://vcenter.yourcompany.com"
username: "admin@vsphere.local"
password: "your-secure-password"
insecure: false  # Set true for self-signed certs
```

### 3. Start the Daemon
```bash
./bin/hypervisord --config config.yaml
```

### 4. Verify Installation
```bash
# Check daemon status
curl http://localhost:8080/health

# List VMs
./bin/hyperctl list

# View dashboard
open http://localhost:8080/dashboard
```

---

## 🔧 Production Deployment

### System Requirements

**Minimum**:
- CPU: 2 cores
- RAM: 2GB
- Disk: 50GB (plus storage for exports)
- OS: Linux (Ubuntu 20.04+, RHEL 8+, Fedora 35+)

**Recommended**:
- CPU: 4+ cores
- RAM: 8GB
- Disk: 500GB SSD
- OS: Ubuntu 22.04 LTS or RHEL 9

### Network Requirements
- Outbound HTTPS (443) to vCenter/cloud providers
- Inbound HTTP (8080) for API/dashboard
- Inbound HTTPS (8443) for TLS (optional)

---

## 📝 Configuration

### Essential Settings

```yaml
# vCenter Connection
vcenter_url: "https://vcenter.example.com"
username: "automation@vsphere.local"
password: "SecurePassword123!"
insecure: false
timeout: 3600s

# Export Performance
download_workers: 3
chunk_size: 33554432  # 32MB
retry_attempts: 3
retry_delay: 5s

# Daemon API
daemon_addr: "0.0.0.0:8080"

# Logging
log_level: "info"  # debug, info, warn, error
progress_style: "bar"
show_eta: true
```

### Connection Pooling (Phase 1.1)

```yaml
connection_pool:
  enabled: true
  max_connections: 5
  idle_timeout: 5m
  health_check_interval: 30s
```

**Benefits**:
- 30-40% faster concurrent exports
- Reduced vCenter load
- Automatic connection health monitoring

### Webhook Notifications (Phase 1.2)

```yaml
webhooks:
  # Slack notifications
  - url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
    events:
      - "job.started"
      - "job.completed"
      - "job.failed"
    timeout: 10s
    retry: 3
    enabled: true

  # Custom API endpoint
  - url: "https://api.yourcompany.com/vm-exports"
    events: ["*"]  # All events
    headers:
      Authorization: "Bearer your-api-token"
      X-Environment: "production"
    timeout: 15s
    retry: 5
    enabled: true
```

**Supported Events**:
- `job.started` - Export job begins
- `job.completed` - Export successful
- `job.failed` - Export failed
- `job.progress` - Progress updates (if enabled)
- `job.cancelled` - Job cancelled by user

### Schedule Persistence (Phase 2.3)

```yaml
database_path: "./hypersdk.db"
```

**Features**:
- Schedules persist across daemon restarts
- Execution history tracking
- SQLite-based (no external DB required)

---

## 🌥️ Multi-Cloud Configuration

### AWS EC2 Export (Phase 4.1)

```yaml
aws:
  enabled: true
  region: "us-east-1"
  access_key: "AKIAIOSFODNN7EXAMPLE"
  secret_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  s3_bucket: "vm-exports"
  export_format: "vmdk"  # vmdk, vhd, raw
```

**Usage Example**:
```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "vm_path": "i-1234567890abcdef0",
    "output_dir": "/exports",
    "format": "vmdk"
  }'
```

### Azure VHD Export (Phase 4.2)

```yaml
azure:
  enabled: true
  subscription_id: "00000000-0000-0000-0000-000000000000"
  tenant_id: "00000000-0000-0000-0000-000000000000"
  client_id: "00000000-0000-0000-0000-000000000000"
  client_secret: "your-client-secret"
  resource_group: "vm-exports-rg"
  location: "eastus"
  storage_account: "vmexportsstorage"
  container: "vhd-exports"
  export_format: "vhd"
```

### GCP Export (Phase 4.3)

```yaml
gcp:
  enabled: true
  project_id: "my-gcp-project"
  zone: "us-central1-a"
  region: "us-central1"
  credentials_json: "/path/to/service-account-key.json"
  gcs_bucket: "vm-exports"
  export_format: "vmdk"
```

### Hyper-V Export (Phase 4.4)

```yaml
hyperv:
  enabled: true
  host: "hyperv-server.example.com"
  username: "Administrator"
  password: "HyperVPassword!"
  use_winrm: true
  winrm_port: 5985
  use_https: false
  export_format: "vhdx"
```

---

## 🔐 Security Hardening

### 1. Enable TLS

```yaml
daemon_addr: "0.0.0.0:8443"
tls:
  enabled: true
  cert_file: "/etc/hypersdk/tls/server.crt"
  key_file: "/etc/hypersdk/tls/server.key"
```

### 2. Configure Authentication

```yaml
auth:
  enabled: true
  session_timeout: 24h
  # Add users via API or config
```

### 3. Enable RBAC

```yaml
rbac:
  enabled: true
  policy_file: "/etc/hypersdk/rbac-policy.yaml"
```

### 4. Secure Secrets

**Option 1: Environment Variables**
```bash
export VCENTER_PASSWORD="secure-password"
export AWS_SECRET_KEY="aws-secret"
./bin/hypervisord --config config.yaml
```

**Option 2: Secrets Management**
```yaml
secrets:
  backend: "vault"  # vault, aws-secrets, azure-keyvault
  vault_addr: "https://vault.example.com"
  vault_token: "s.xxxxxxxxxxxx"
```

---

## 🐳 Docker Deployment

### Build Docker Image

```bash
docker build -t hypersdk:latest .
```

### Run with Docker

```bash
docker run -d \
  --name hypersdk \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v $(pwd)/exports:/exports \
  hypersdk:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  hypersdk:
    image: hypersdk:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./exports:/exports
      - ./data:/app/data
    environment:
      - LOG_LEVEL=info
    restart: unless-stopped
```

---

## ☸️ Kubernetes Deployment

### Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hypersdk
  namespace: vm-management
spec:
  replicas: 2
  selector:
    matchLabels:
      app: hypersdk
  template:
    metadata:
      labels:
        app: hypersdk
    spec:
      containers:
      - name: hypersdk
        image: hypersdk:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
        - name: exports
          mountPath: /exports
        env:
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "8Gi"
            cpu: "4000m"
      volumes:
      - name: config
        configMap:
          name: hypersdk-config
      - name: exports
        persistentVolumeClaim:
          claimName: hypersdk-exports
```

### Service & Ingress

```yaml
apiVersion: v1
kind: Service
metadata:
  name: hypersdk
spec:
  selector:
    app: hypersdk
  ports:
  - port: 8080
    targetPort: 8080
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hypersdk
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
  - hosts:
    - hypersdk.example.com
    secretName: hypersdk-tls
  rules:
  - host: hypersdk.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hypersdk
            port:
              number: 8080
```

---

## 🔄 Systemd Service

### Create Service File

```bash
sudo tee /etc/systemd/system/hypervisord.service > /dev/null <<EOF
[Unit]
Description=HyperSDK VM Export Daemon
After=network.target

[Service]
Type=simple
User=hypersdk
Group=hypersdk
WorkingDirectory=/opt/hypersdk
ExecStart=/opt/hypersdk/bin/hypervisord --config /etc/hypersdk/config.yaml
Restart=on-failure
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=hypervisord

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/hypersdk/data /var/log/hypersdk

[Install]
WantedBy=multi-user.target
EOF
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable hypervisord
sudo systemctl start hypervisord
sudo systemctl status hypervisord
```

### View Logs

```bash
sudo journalctl -u hypervisord -f
```

---

## 📊 Monitoring Setup

### Prometheus Integration

HyperSDK exposes metrics at `/metrics`:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'hypersdk'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

**Key Metrics**:
- `hypersdk_jobs_total` - Total jobs by status
- `hypersdk_job_duration_seconds` - Export duration
- `hypersdk_queue_length` - Pending jobs
- `hypersdk_http_requests_total` - API requests
- `hypersdk_memory_bytes` - Memory usage
- `hypersdk_goroutines` - Active goroutines

### Grafana Dashboards

Import pre-built dashboards from `monitoring/grafana/`:
- `hypersdk-overview.json` - System overview
- `job-performance.json` - Job metrics
- `system-resources.json` - Resource usage

```bash
# Start monitoring stack
cd monitoring
docker-compose up -d

# Access Grafana
open http://localhost:3000
# Login: admin / admin
```

---

## 🧪 Testing the Deployment

### 1. Health Check

```bash
curl http://localhost:8080/health
# Expected: {"status":"ok","version":"0.2.0"}
```

### 2. List VMs

```bash
curl http://localhost:8080/vms
```

### 3. Test Export (vSphere)

```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-export",
    "vm_path": "/Datacenter/vm/test-vm",
    "output_dir": "/exports",
    "format": "ova",
    "compress": true
  }'
```

### 4. Test Scheduled Job

```bash
curl -X POST http://localhost:8080/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "daily-backup",
    "name": "Daily VM Backup",
    "schedule": "0 2 * * *",
    "enabled": true,
    "job_template": {
      "vm_path": "/Datacenter/vm/production-*",
      "output_dir": "/backups",
      "format": "ova",
      "compress": true
    }
  }'
```

---

## 🔧 Troubleshooting

### Daemon Won't Start

```bash
# Check logs
./bin/hypervisord --config config.yaml 2>&1 | tee startup.log

# Common issues:
# 1. Port already in use
sudo lsof -i :8080

# 2. Config file not found
ls -l config.yaml

# 3. Permission issues
chmod 644 config.yaml
```

### Connection Pool Issues

```bash
# Enable debug logging
log_level: "debug"

# Check pool stats via API
curl http://localhost:8080/metrics | grep connection_pool
```

### Webhook Not Firing

```bash
# Test webhook manually
curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  -H "Content-Type: application/json" \
  -d '{"text":"Test from HyperSDK"}'

# Check webhook logs
curl http://localhost:8080/webhooks/status
```

### Schedule Not Persisting

```bash
# Check database file
ls -l hypersdk.db

# Verify schedules in DB
sqlite3 hypersdk.db "SELECT * FROM scheduled_jobs;"

# Force schedule reload
curl -X POST http://localhost:8080/schedules/reload
```

---

## 📈 Performance Tuning

### For High-Throughput Environments

```yaml
# Increase workers
download_workers: 10

# Larger chunks
chunk_size: 67108864  # 64MB

# More connections
connection_pool:
  max_connections: 10

# Disable progress updates
progress_style: "quiet"
```

### For Resource-Constrained Systems

```yaml
# Fewer workers
download_workers: 2

# Smaller chunks
chunk_size: 16777216  # 16MB

# Limited connections
connection_pool:
  max_connections: 2
```

---

## 🔄 Upgrade Guide

### From v0.1.x to v0.2.0

1. **Backup Configuration**
```bash
cp config.yaml config.yaml.backup
cp hypersdk.db hypersdk.db.backup
```

2. **Stop Old Version**
```bash
sudo systemctl stop hypervisord
```

3. **Replace Binaries**
```bash
sudo cp bin/* /opt/hypersdk/bin/
```

4. **Update Configuration**
```bash
# Add new Phase 4 sections to config.yaml
# See config.example.yaml for reference
```

5. **Start New Version**
```bash
sudo systemctl start hypervisord
sudo systemctl status hypervisord
```

6. **Verify**
```bash
./bin/hypervisord --version
# Should show: v0.2.0

curl http://localhost:8080/health
```

---

## 🆘 Support & Resources

### Documentation
- [Multi-Cloud Guide](MULTI_CLOUD_GUIDE.md)
- [API Reference](API_REFERENCE.md)
- [Example Configurations](examples/)

### Community
- GitHub Issues: https://github.com/your-org/hypersdk/issues
- Discussions: https://github.com/your-org/hypersdk/discussions

### Commercial Support
- Email: support@hypersdk.io
- Documentation: https://docs.hypersdk.io

---

## ✅ Deployment Checklist

- [ ] Downloaded/compiled binaries
- [ ] Created `config.yaml` from example
- [ ] Configured vCenter credentials
- [ ] Tested vCenter connectivity
- [ ] Enabled connection pooling
- [ ] Configured webhooks (optional)
- [ ] Set up scheduled jobs (optional)
- [ ] Configured cloud providers (optional)
- [ ] Enabled TLS/authentication
- [ ] Set up monitoring (Prometheus/Grafana)
- [ ] Created systemd service
- [ ] Tested export functionality
- [ ] Configured backups
- [ ] Documented deployment

---

**Deployment complete! Your HyperSDK instance is ready for production use.** 🎉
