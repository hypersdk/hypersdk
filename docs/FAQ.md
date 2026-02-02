# HyperSDK Frequently Asked Questions (FAQ)

Answers to common questions about HyperSDK.

## 📋 Table of Contents

- [General](#general)
- [Installation & Setup](#installation--setup)
- [Features & Capabilities](#features--capabilities)
- [Performance & Optimization](#performance--optimization)
- [Troubleshooting](#troubleshooting)
- [Cost & Pricing](#cost--pricing)
- [Integration & API](#integration--api)
- [Security](#security)

---

## 🌟 General

### What is HyperSDK?

HyperSDK is an enterprise-grade, multi-cloud VM export and migration platform. It provides a unified API for exporting VMs from multiple virtualization platforms (vSphere, AWS, Azure, GCP, etc.) with advanced features like incremental backups, cost estimation, and format conversion.

### Is HyperSDK open source?

Yes! HyperSDK is licensed under LGPL-3.0-or-later. You can use, modify, and distribute it freely.

### What makes HyperSDK different from other tools?

**Key Differentiators**:
- **Multi-cloud native** - Single tool for 9 cloud providers
- **Incremental backups** - 95% faster with CBT
- **Zero-downtime updates** - Hot-loadable provider plugins
- **Cost estimation** - Know costs before exporting
- **Native format conversion** - No external dependencies
- **3 SDK languages** - Python, TypeScript, OpenAPI
- **Advanced scheduling** - Dependencies, retries, time windows

### Who should use HyperSDK?

**Target Users**:
- Enterprise IT teams (data center migrations)
- Cloud service providers (migration services)
- MSPs (managed backup services)
- DevOps teams (infrastructure automation)
- Anyone migrating VMs between platforms

---

## 🚀 Installation & Setup

### What are the system requirements?

**Minimum**:
- Linux, macOS, or Windows
- 2 GB RAM
- 10 GB disk space
- Network access to vCenter/cloud providers

**Recommended**:
- 4 GB RAM
- 100 GB disk space (for exports)
- SSD for better I/O performance

### How do I install HyperSDK?

**Docker (Easiest)**:
```bash
docker run -d -p 8080:8080 \
  -e GOVC_URL='https://vcenter/sdk' \
  -e GOVC_USERNAME='admin' \
  -e GOVC_PASSWORD='pass' \
  hypersdk/hypervisord
```

**From Source**:
```bash
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk
go build -o hypervisord ./cmd/hypervisord
./hypervisord
```

See [Quick Start Guide](QUICK_START.md) for details.

### Do I need to install any dependencies?

**For the daemon**: No! HyperSDK is a single binary with no external dependencies.

**For SDKs**:
- Python: `pip install hypersdk`
- TypeScript: `npm install hypersdk`

### Can I run HyperSDK without root/admin privileges?

Yes! HyperSDK can run as a regular user. However:
- Ensure the user has write access to export directories
- For systemd integration, root is needed to install the service
- For Docker, the user needs Docker permissions

### How do I configure vCenter credentials?

**Environment Variables** (recommended):
```bash
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='admin@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1  # Only for dev/test!
```

**Config File**:
```yaml
# /etc/hypervisord/config.yaml
VCenterURL: "https://vcenter.example.com/sdk"
Username: "admin@vsphere.local"
Password: "your-password"
Insecure: true
```

**Best Practice**: Use environment variables for credentials, config file for other settings.

---

## 💡 Features & Capabilities

### What cloud providers are supported?

**9 Providers** (all production-ready):
1. vSphere (VMware vCenter/ESXi)
2. AWS (Amazon EC2)
3. Azure (Microsoft Azure VMs)
4. GCP (Google Compute Engine)
5. Hyper-V (Microsoft Hyper-V)
6. OCI (Oracle Cloud Infrastructure)
7. OpenStack (Nova/Swift)
8. Alibaba Cloud (Aliyun ECS/OSS)
9. Proxmox VE

### What export formats are supported?

**Output Formats**:
- OVA (compressed virtual appliance)
- OVF (open virtualization format)
- VMDK (VMware virtual disk)

**Conversion Targets**:
- QCOW2 (for KVM/QEMU)
- VHD (for Hyper-V)
- VHDX (for Hyper-V Gen 2)
- VDI (for VirtualBox)
- RAW (universal format)

### Can I export running VMs?

**Yes**, but with considerations:
- **Recommended**: Shut down VM for consistent backup
- **Running VMs**: Possible but may have inconsistencies
- **Snapshots**: Can export from specific snapshot
- **CBT**: Works with running VMs for incremental backups

**Best Practice**: Use VM snapshots for consistent exports of running VMs.

### Does HyperSDK support incremental backups?

**Yes!** Using Changed Block Tracking (CBT):
- **95% faster** than full exports
- **90% storage savings** for typical workloads
- Automatic detection of changed blocks
- Base + delta model for recovery

See [Incremental Export Guide](features/INCREMENTAL_EXPORT.md).

### Can I schedule automated backups?

**Absolutely!** Advanced scheduling features:
- **Cron-style schedules** (e.g., `0 2 * * 0` = 2 AM Sundays)
- **Job dependencies** (wait for other jobs)
- **Retry policies** (automatic retry with backoff)
- **Time windows** (only run during specific hours)
- **Priority queuing** (important jobs first)

See [Advanced Scheduling Guide](features/ADVANCED_SCHEDULING.md).

### How does cost estimation work?

HyperSDK includes realistic pricing data for major cloud providers:
- **AWS S3** (Standard, IA, Glacier, Deep Archive)
- **Azure Blob** (Hot, Cool, Archive)
- **Google Cloud Storage** (Standard, Nearline, Coldline, Archive)

**Features**:
- Estimate costs before exporting
- Compare providers side-by-side
- Yearly cost projections
- Detailed breakdowns (storage, transfer, requests)

See [Cost Estimation Guide](features/COST_ESTIMATION.md).

---

## ⚡ Performance & Optimization

### How fast can HyperSDK export VMs?

**Typical Performance**:
- **Full export**: 100-150 MB/s (network-limited)
- **Incremental export**: 2000+ MB/s equivalent (95% time reduction)
- **With connection pooling**: 30-40% faster

**Example**:
- 500 GB VM: ~83 minutes (full) or ~4 minutes (incremental)

### How can I improve export performance?

**Optimization Tips**:

1. **Use Incremental Backups**:
   - Enable CBT for 95% faster exports
   - Only changed blocks are transferred

2. **Increase Parallel Workers**:
   ```yaml
   DownloadWorkers: 8  # Default: 4
   ```

3. **Use Connection Pooling**:
   - Automatically enabled (30-40% faster)

4. **Use SSD for Export Directory**:
   - Better I/O performance

5. **Network Optimization**:
   - Ensure good network between HyperSDK and vCenter
   - Use 10 Gbps network if possible

6. **Compression**:
   - Enable compression for smaller transfers
   - Trade-off: slight CPU overhead

### How much storage space do I need?

**Full Exports**: 1.2× the total VM disk size
- Buffer for compression and metadata

**Incremental Exports**: 0.1-0.2× the VM disk size
- Only changed blocks (typically 10-20%)

**Example**:
- 10 VMs × 500 GB each = 5 TB
- Full: ~6 TB needed
- Incremental: ~600 GB-1.2 TB needed

### Can HyperSDK handle multiple concurrent exports?

**Yes!** HyperSDK is designed for concurrent operations:
- **Concurrent exports**: Limited by system resources
- **Goroutine-based**: Efficient concurrency
- **Resource management**: Automatic queue management

**Recommended Limits**:
- Small VMs (<50 GB): 10-20 concurrent
- Medium VMs (50-200 GB): 5-10 concurrent
- Large VMs (>200 GB): 2-5 concurrent

---

## 🔧 Troubleshooting

### The API returns "connection refused"

**Possible Causes**:
1. Daemon not running
2. Wrong port
3. Firewall blocking

**Solutions**:
```bash
# Check if daemon is running
curl http://localhost:8080/health

# Check Docker container
docker ps | grep hypervisord
docker logs hypervisord

# Check systemd service
sudo systemctl status hypervisord

# Check firewall
sudo firewall-cmd --list-ports  # Fedora/RHEL
sudo ufw status                  # Ubuntu
```

### Jobs fail with "authentication failed"

**Check**:
1. vCenter credentials are correct
2. User has required permissions
3. Network connectivity to vCenter

**Verify**:
```bash
# Test credentials
curl -k https://vcenter.example.com/sdk

# Check daemon logs
docker logs hypervisord | grep -i auth
journalctl -u hypervisord | grep -i auth
```

**Required Permissions**:
- Read access to VMs
- Access to VM snapshots (for CBT)
- Read access to datastores

### Exports are very slow

**Checklist**:
1. ✅ Network bandwidth adequate?
2. ✅ Using incremental backups?
3. ✅ Connection pooling enabled?
4. ✅ Parallel workers configured?
5. ✅ Export directory on fast storage?

**Diagnostic**:
```bash
# Check job progress
curl http://localhost:8080/jobs/progress/{job-id}

# Look for network issues
docker logs hypervisord | grep -i error

# Monitor network
iftop -i eth0
```

### "Disk space full" error

**Quick Fix**:
```bash
# Check available space
df -h /exports

# Clean old exports
find /exports -type f -mtime +30 -delete

# Move exports to larger volume
rsync -avz /exports/ /mnt/large-volume/exports/
```

**Prevention**:
- Monitor disk usage
- Set up retention policies
- Use incremental backups (90% less space)

### CBT not working / "CBT not enabled"

**Requirements**:
- VMware vSphere 6.5+ (CBT supported)
- VM hardware version 7+
- VM must be powered off to enable CBT (first time)

**Enable CBT**:
```python
from hypersdk import HyperSDK
client = HyperSDK("http://localhost:8080")

# Enable CBT (VM will be powered off temporarily)
result = client.enable_cbt("/datacenter/vm/my-vm")
print(result)
```

**Troubleshoot**:
```bash
# Check CBT status
curl -X POST http://localhost:8080/cbt/status \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'

# Check daemon logs
docker logs hypervisord | grep -i cbt
```

---

## 💰 Cost & Pricing

### Is HyperSDK free?

**Yes!** HyperSDK is open source (LGPL-3.0):
- Free to use
- Free to modify
- Free to distribute
- No licensing fees
- No per-VM costs

**Costs You May Incur**:
- Cloud storage fees (S3, Azure, GCS)
- Network transfer costs
- Infrastructure costs (servers, storage)

### How accurate is the cost estimation?

**Pricing Data**: Based on January 2026 official cloud provider pricing

**Accuracy**:
- ✅ Storage costs: ±2% (very accurate)
- ✅ Transfer costs: ±5% (tiered pricing)
- ⚠️ Request costs: ±10% (usage patterns vary)

**Note**: Estimates are approximations. Always verify with official cloud provider calculators before committing.

### Can I use my own pricing data?

Not directly through the API currently, but:
- Pricing data is in `providers/cost/calculator.go`
- Can modify and rebuild for custom pricing
- Future feature: Custom pricing configuration

**Workaround**:
```python
# Use API estimates as baseline
estimate = client.estimate_cost(...)
custom_rate = 0.015  # Your negotiated rate
custom_cost = estimate['breakdown']['storage_cost'] * (custom_rate / 0.023)
```

### Does HyperSDK charge for API calls?

**No!** All API calls are free. There are no:
- Per-request fees
- Rate limits (besides system capacity)
- Subscription costs
- Hidden charges

---

## 🔌 Integration & API

### Can I integrate HyperSDK with my existing tools?

**Yes!** Multiple integration options:

**REST API**:
- 67+ endpoints
- Standard HTTP/JSON
- OpenAPI 3.0 spec

**SDKs**:
- Python SDK
- TypeScript SDK
- Generate clients from OpenAPI spec

**Automation Tools**:
- Ansible playbooks
- Terraform providers
- Jenkins pipelines
- GitLab CI/CD

See [Integration Examples](INTEGRATION_GUIDE.md).

### Is there a CLI tool?

**Yes!** Multiple CLI tools:

1. **hyperctl** - Migration commander
2. **hyperexport** - Standalone export tool
3. **hypervisord** - Daemon service

**Example**:
```bash
# Using hyperctl
hyperctl submit --vm /datacenter/vm/my-vm --output /exports

# Using curl (REST API)
curl -X POST http://localhost:8080/jobs/submit -d '{...}'

# Using Python SDK
python -c "from hypersdk import HyperSDK; ..."
```

### Can I get webhooks for job events?

**Yes!** Webhook support for:
- Job started
- Job completed
- Job failed
- Progress updates

**Setup**:
```bash
curl -X POST http://localhost:8080/webhooks \
  -d '{
    "url": "https://my-server.com/webhook",
    "events": ["job.completed", "job.failed"]
  }'
```

### How do I monitor HyperSDK?

**Built-in Monitoring**:
- **Health endpoint**: `/health`
- **Status endpoint**: `/status`
- **Metrics**: Prometheus-compatible (planned)

**External Monitoring**:
```bash
# Health check
curl http://localhost:8080/health

# Get daemon status
curl http://localhost:8080/status

# List all jobs
curl http://localhost:8080/jobs/query?all=true
```

**Alerting**:
- Set up webhooks for failures
- Monitor with Prometheus + Alertmanager
- Custom scripts polling API

---

## 🔒 Security

### How secure is HyperSDK?

**Security Features**:
- ✅ TLS/SSL support for API
- ✅ API key authentication
- ✅ Session-based auth
- ✅ RBAC (role-based access control)
- ✅ Audit logging
- ✅ Encrypted credentials storage

### Should I use HTTPS?

**Yes!** For production deployments:

```bash
# Start with TLS
./hypervisord \
  --tls-cert /path/to/cert.pem \
  --tls-key /path/to/key.pem \
  --tls-port 8443
```

**Development**: HTTP on localhost is acceptable

### How are vCenter credentials stored?

**Options**:

1. **Environment Variables** (recommended for containers):
   ```bash
   export GOVC_PASSWORD='password'
   ```

2. **Config File** (should be secured):
   ```bash
   chmod 600 /etc/hypervisord/config.yaml
   ```

3. **Secrets Management** (best for production):
   - Kubernetes Secrets
   - HashiCorp Vault
   - AWS Secrets Manager

**Best Practice**: Never commit credentials to git!

### Can I encrypt exports?

**Yes!** Two options:

1. **AES-256-GCM** (built-in):
   ```bash
   # Export with encryption
   hyperexport --vm /path/to/vm --encrypt --passphrase "secret"
   ```

2. **GPG** (if available):
   ```bash
   hyperexport --vm /path/to/vm --encrypt-method gpg \
     --gpg-recipient admin@example.com
   ```

### Is there an audit log?

**Yes!** All operations are logged:
- Job submissions
- API requests
- Authentication events
- Configuration changes

**Access Logs**:
```bash
# Docker
docker logs hypervisord

# Systemd
journalctl -u hypervisord

# File
tail -f /var/log/hypervisord/audit.log
```

---

## 📚 More Help

### Where can I find more documentation?

- [Quick Start Guide](QUICK_START.md)
- [Features Overview](FEATURES_OVERVIEW.md)
- [API Reference](API_ENDPOINTS.md)
- [Examples](../examples/)
- Feature-specific guides in [docs/features/](features/)

### How do I report a bug?

1. Check existing issues: https://github.com/ssahani/hypersdk/issues
2. Create new issue with:
   - Steps to reproduce
   - Expected vs actual behavior
   - Logs and error messages
   - Environment details

### How can I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md) for:
- Code contributions
- Documentation improvements
- Bug reports
- Feature requests

### Where can I get help?

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and community support
- **Documentation**: Comprehensive guides and examples
- **Examples**: Ready-to-use code samples

---

## 🎯 Quick Reference

### Most Common Tasks

```bash
# Start daemon
docker run -d -p 8080:8080 hypersdk/hypervisord

# Health check
curl http://localhost:8080/health

# Export a VM
curl -X POST http://localhost:8080/jobs/submit \
  -d '{"vm_path": "/datacenter/vm/my-vm", "output_path": "/exports"}'

# Check job status
curl http://localhost:8080/jobs/{job-id}

# Enable CBT
curl -X POST http://localhost:8080/cbt/enable \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'

# Compare cloud costs
curl -X POST http://localhost:8080/cost/compare \
  -d '{"storage_gb": 500, "duration_days": 365}'
```

---

*Last Updated: 2026-02-04*
*For more questions, see the [documentation index](00-INDEX.md)*
