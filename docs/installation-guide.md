# HyperSDK Installation Guide

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Installation Methods](#installation-methods)
3. [Post-Installation Configuration](#post-installation-configuration)
4. [Verification](#verification)
5. [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements

- **OS**: Linux (Fedora, RHEL, CentOS, Ubuntu, Debian)
- **Architecture**: x86_64 (AMD64)
- **RAM**: 2GB minimum, 4GB recommended
- **Disk Space**: 500MB for binaries, additional space for VM exports
- **Go Version**: 1.24+ (for building from source)
- **Network**: Connectivity to vCenter/cloud providers

### Recommended Requirements

- **RAM**: 8GB or more for concurrent exports
- **CPU**: 4+ cores for parallel processing
- **Disk**: SSD with 100GB+ free space
- **Network**: 1Gbps+ for fast VM transfers

### Required Permissions

- **vSphere**: Read-only access minimum, VM export permissions
- **KVM/Libvirt**: Root or libvirt group membership
- **Network**: Firewall rules for API port (default 8080)

## Installation Methods

### Method 1: RPM Package (Recommended for Fedora/RHEL/CentOS)

```bash
# Download RPM package
wget https://github.com/ssahani/hypersdk/releases/latest/download/hypersdk-0.2.0-1.fc39.x86_64.rpm

# Install package
sudo dnf install -y hypersdk-0.2.0-1.fc39.x86_64.rpm

# Package includes:
# - /usr/bin/hyperexport
# - /usr/bin/hypervisord
# - /usr/bin/hyperctl
# - /etc/hypervisord/config.yaml
# - /usr/lib/systemd/system/hypervisord.service
```

### Method 2: Build from Source

```bash
# Install dependencies
sudo dnf install -y golang git make

# Clone repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Build all binaries
make build

# Or build individually
go build -o hyperexport ./cmd/hyperexport
go build -o hypervisord ./cmd/hypervisord
go build -o hyperctl ./cmd/hyperctl

# Install (requires root)
sudo make install
```

### Method 3: Docker/Podman Container

HyperSDK provides comprehensive containerization with Docker/Podman support:

```bash
# Clone repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Build container images
./deployments/scripts/build-images.sh --builder podman

# Start full stack (hypervisord + Redis + Prometheus + Grafana)
cd deployments/docker
podman compose up -d

# Access services:
# - API:       http://localhost:8080
# - Dashboard: http://localhost:8080/web/dashboard/
# - Grafana:   http://localhost:3000
# - Prometheus: http://localhost:9090
```

See [Docker/Podman Deployment Guide](../deployments/docker/README.md) for details.

### Method 4: Kubernetes

Deploy to Kubernetes with Kustomize (supports dev/staging/prod environments):

```bash
# Clone repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Configure secrets
cp deployments/kubernetes/base/secrets.yaml.example \
   deployments/kubernetes/overlays/development/secrets.yaml
vim deployments/kubernetes/overlays/development/secrets.yaml

# Deploy to development
./deployments/scripts/deploy-k8s.sh development

# Access API
kubectl port-forward -n hypersdk svc/hypervisord 8080:8080
```

See [Kubernetes Deployment Guide](../deployments/kubernetes/README.md) for details.

### Method 5: Binary Download

```bash
# Download binaries
wget https://github.com/ssahani/hypersdk/releases/latest/download/hyperexport-linux-amd64
wget https://github.com/ssahani/hypersdk/releases/latest/download/hypervisord-linux-amd64
wget https://github.com/ssahani/hypersdk/releases/latest/download/hyperctl-linux-amd64

# Make executable
chmod +x hyperexport-linux-amd64 hypervisord-linux-amd64 hyperctl-linux-amd64

# Move to PATH
sudo mv hyperexport-linux-amd64 /usr/local/bin/hyperexport
sudo mv hypervisord-linux-amd64 /usr/local/bin/hypervisord
sudo mv hyperctl-linux-amd64 /usr/local/bin/hyperctl
```

## Post-Installation Configuration

### 1. Create Configuration Directory

```bash
sudo mkdir -p /etc/hypervisord
sudo mkdir -p /var/lib/hypersdk/exports
sudo mkdir -p /var/log/hypersdk
```

### 2. Configure vSphere Connection

Create `/etc/hypervisord/config.yaml`:

```yaml
# vSphere Configuration
vsphere:
  url: "https://vcenter.example.com/sdk"
  username: "administrator@vsphere.local"
  password: "your-secure-password"
  insecure: false  # Set to true for self-signed certs

# Daemon Configuration
daemon:
  addr: "0.0.0.0:8080"
  log_level: "info"
  download_workers: 4
  max_concurrent_jobs: 10

# Export Configuration
export:
  output_dir: "/var/lib/hypersdk/exports"
  default_format: "ova"
  compress: true
  verify_checksums: true

# Connection Pool
connection_pool:
  max_connections: 5
  idle_timeout: "5m"
  health_check_interval: "30s"

# Webhooks (optional)
webhooks:
  - url: "https://hooks.example.com/hypersdk"
    events: ["job.started", "job.completed", "job.failed"]
    headers:
      Authorization: "Bearer your-token"
    timeout: "10s"
    retry: 3
    enabled: true

# Web Dashboard
web:
  enabled: true
  static_dir: "/usr/share/hypersdk/web"
```

### 3. Secure Configuration File

```bash
# Set restrictive permissions
sudo chmod 600 /etc/hypervisord/config.yaml
sudo chown root:root /etc/hypervisord/config.yaml
```

### 4. Configure Systemd Service

```bash
# Enable service
sudo systemctl enable hypervisord

# Start service
sudo systemctl start hypervisord

# Check status
sudo systemctl status hypervisord
```

### 5. Configure Firewall

```bash
# Fedora/RHEL/CentOS
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# Ubuntu/Debian (UFW)
sudo ufw allow 8080/tcp
sudo ufw reload

# Direct iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
sudo iptables-save > /etc/iptables/rules.v4
```

### 6. Environment Variables (Alternative to Config File)

```bash
# Add to ~/.bashrc or /etc/environment
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1
export DAEMON_ADDR='localhost:8080'
export LOG_LEVEL='info'
```

## Verification

### 1. Verify Installation

```bash
# Check binary versions
hyperexport -version
hypervisord -version
hyperctl -version

# Check binary locations
which hyperexport
which hypervisord
which hyperctl
```

### 2. Test vSphere Connection

```bash
# Using hyperexport
hyperexport -vm "/datacenter/vm/test-vm" -dry-run

# Using hyperctl
hyperctl status
```

### 3. Test Daemon

```bash
# Check daemon status
sudo systemctl status hypervisord

# Check daemon logs
sudo journalctl -u hypervisord -n 50

# Test API endpoint
curl http://localhost:8080/health

# Test dashboard
curl http://localhost:8080/web/dashboard/
```

### 4. Test Export

```bash
# Small test export
hyperexport -vm "/datacenter/vm/small-test-vm" \
  -output /tmp/test-export \
  -verify

# Verify exported files
ls -lh /tmp/test-export/
```

## Troubleshooting

### Issue: Binary Not Found

```bash
# Check PATH
echo $PATH

# Add to PATH
export PATH=$PATH:/usr/local/bin

# Make permanent in ~/.bashrc
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
```

### Issue: Permission Denied

```bash
# Check file permissions
ls -l /usr/bin/hyperexport

# Fix permissions
sudo chmod +x /usr/bin/hyperexport
sudo chmod +x /usr/bin/hypervisord
sudo chmod +x /usr/bin/hyperctl
```

### Issue: vSphere Connection Failed

```bash
# Test network connectivity
ping vcenter.example.com
telnet vcenter.example.com 443

# Test with curl
curl -k https://vcenter.example.com/sdk

# Verify credentials
# Check username format: user@domain or domain\user
# For vCenter: administrator@vsphere.local
```

### Issue: Daemon Won't Start

```bash
# Check logs
sudo journalctl -u hypervisord -xe

# Common causes:
# 1. Port already in use
sudo netstat -tlnp | grep 8080
sudo lsof -i :8080

# 2. Config file errors
sudo hypervisord --config /etc/hypervisord/config.yaml --log-level debug

# 3. Missing dependencies
ldd /usr/bin/hypervisord
```

### Issue: Web Dashboard Not Loading

```bash
# Verify web files exist
ls /usr/share/hypersdk/web/

# Check if web is enabled in config
grep -A 2 "^web:" /etc/hypervisord/config.yaml

# Test API directly
curl http://localhost:8080/health
curl http://localhost:8080/status

# Check browser console for errors
# Verify WebSocket connection at ws://localhost:8080/ws
```

### Issue: SELinux Blocking (Fedora/RHEL)

```bash
# Check SELinux status
getenforce

# View denials
sudo ausearch -m avc -ts recent

# Create custom policy (if needed)
sudo ausearch -m avc -ts recent | audit2allow -M hypersdk
sudo semodule -i hypersdk.pp

# Or set to permissive (not recommended for production)
sudo setenforce 0
```

## Upgrade

### Upgrading from Previous Version

```bash
# Stop daemon
sudo systemctl stop hypervisord

# Backup configuration
sudo cp /etc/hypervisord/config.yaml /etc/hypervisord/config.yaml.backup

# Upgrade RPM
sudo dnf upgrade hypersdk

# Or rebuild from source
cd hypersdk
git pull
make build
sudo make install

# Restart daemon
sudo systemctl start hypervisord

# Verify
hypervisord -version
```

## Uninstallation

### Remove RPM Package

```bash
# Stop and disable service
sudo systemctl stop hypervisord
sudo systemctl disable hypervisord

# Remove package
sudo dnf remove hypersdk

# Remove data (optional)
sudo rm -rf /var/lib/hypersdk
sudo rm -rf /etc/hypervisord
sudo rm -rf /var/log/hypersdk
```

### Remove Manual Installation

```bash
# Stop daemon
sudo systemctl stop hypervisord
sudo systemctl disable hypervisord

# Remove binaries
sudo rm /usr/bin/hyperexport
sudo rm /usr/bin/hypervisord
sudo rm /usr/bin/hyperctl

# Remove systemd service
sudo rm /usr/lib/systemd/system/hypervisord.service
sudo systemctl daemon-reload

# Remove configuration and data
sudo rm -rf /etc/hypervisord
sudo rm -rf /var/lib/hypersdk
```

## Next Steps

After successful installation:

1. Read the [Getting Started Guide](getting-started.md)
2. Review [Configuration Reference](configuration-reference.md)
3. Explore [API Documentation](API_ENDPOINTS.md)
4. Try [Interactive Mode Guide](user-guides/01-interactive-mode.md)
5. Check [Migration Workflows](migration-workflows.md)

## Support

- **GitHub Issues**: https://github.com/ssahani/hypersdk/issues
- **Documentation**: https://github.com/ssahani/hypersdk/docs
- **Email**: ssahani@gmail.com
