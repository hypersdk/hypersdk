# HyperSDK Troubleshooting Guide

Comprehensive guide to diagnosing and resolving common issues with HyperSDK.

## 📋 Table of Contents

- [Connection Issues](#connection-issues)
- [Authentication Problems](#authentication-problems)
- [Export Failures](#export-failures)
- [Performance Issues](#performance-issues)
- [CBT Issues](#cbt-issues)
- [API Errors](#api-errors)
- [Resource Problems](#resource-problems)
- [Diagnostic Tools](#diagnostic-tools)

---

## 🔌 Connection Issues

### Problem: "Connection Refused" when accessing API

**Symptoms**:
```bash
$ curl http://localhost:8080/health
curl: (7) Failed to connect to localhost port 8080: Connection refused
```

**Diagnosis**:
```bash
# 1. Check if daemon is running
docker ps | grep hypervisord
# or
sudo systemctl status hypervisord
# or
ps aux | grep hypervisord

# 2. Check if port is in use
netstat -tuln | grep 8080
# or
lsof -i :8080

# 3. Check firewall
sudo firewall-cmd --list-ports  # Fedora/RHEL
sudo ufw status                  # Ubuntu
```

**Solutions**:

**A. Daemon Not Running**:
```bash
# Docker
docker start hypervisord
# or restart
docker restart hypervisord

# Systemd
sudo systemctl start hypervisord

# Manual
./hypervisord &
```

**B. Port Already in Use**:
```bash
# Find process using port
sudo lsof -i :8080
# Kill it
sudo kill -9 <PID>

# Or use different port
docker run -p 8081:8080 hypersdk/hypervisord
./hypervisord --port 8081
```

**C. Firewall Blocking**:
```bash
# Fedora/RHEL
sudo firewall-cmd --add-port=8080/tcp --permanent
sudo firewall-cmd --reload

# Ubuntu
sudo ufw allow 8080/tcp
sudo ufw reload
```

---

### Problem: "Cannot connect to vCenter"

**Symptoms**:
```
Error: Failed to connect to vCenter: connection timeout
```

**Diagnosis**:
```bash
# Test vCenter connectivity
curl -k https://vcenter.example.com/sdk

# Check network route
traceroute vcenter.example.com

# Check DNS resolution
nslookup vcenter.example.com

# Verify credentials
cat ~/.govmomirc
# or
echo $GOVC_URL
```

**Solutions**:

**A. Network Connectivity**:
```bash
# Test connection
ping vcenter.example.com

# Test HTTPS
curl -k https://vcenter.example.com/sdk

# Check proxy settings
echo $HTTP_PROXY
echo $HTTPS_PROXY
```

**B. TLS Certificate Issues**:
```bash
# For development, skip TLS verification
export GOVC_INSECURE=1

# For production, add CA certificate
cp vcenter-ca.crt /etc/pki/ca-trust/source/anchors/
update-ca-trust
```

**C. Incorrect URL**:
```bash
# Correct format
export GOVC_URL='https://vcenter.example.com/sdk'

# NOT
export GOVC_URL='vcenter.example.com'  # Missing https://
export GOVC_URL='https://vcenter.example.com'  # Missing /sdk
```

---

## 🔐 Authentication Problems

### Problem: "Authentication failed" errors

**Symptoms**:
```json
{
  "error": "Authentication failed",
  "timestamp": "2026-02-04T10:30:00Z"
}
```

**Diagnosis**:
```bash
# Check credentials
echo "URL: $GOVC_URL"
echo "User: $GOVC_USERNAME"
echo "Pass: [hidden]"  # Don't echo password!

# Test with govc (if installed)
govc about

# Check daemon logs
docker logs hypervisord | grep -i auth
journalctl -u hypervisord | grep -i auth
```

**Solutions**:

**A. Wrong Credentials**:
```bash
# Verify username format
export GOVC_USERNAME='administrator@vsphere.local'  # Correct
# NOT
export GOVC_USERNAME='administrator'  # Missing domain

# Update credentials
docker stop hypervisord
docker rm hypervisord
docker run -d \
  -e GOVC_USERNAME='correct-username' \
  -e GOVC_PASSWORD='correct-password' \
  hypersdk/hypervisord
```

**B. Expired Session**:
```bash
# Restart daemon to create new session
docker restart hypervisord
sudo systemctl restart hypervisord
```

**C. Account Locked**:
```bash
# Check vCenter for locked accounts
# In vSphere Client: Administration > Users and Groups
# Unlock account or use different credentials
```

---

### Problem: "Permission denied" for VM operations

**Symptoms**:
```
Error: Permission denied: Cannot read VM configuration
```

**Required Permissions**:
- **Datastore**: Browse datastore, Low level file operations
- **Virtual Machine**:
  - Configuration > Settings
  - Snapshot management > Create snapshot
  - Provisioning > Allow disk access

**Solutions**:

**A. Grant Required Permissions**:
1. In vSphere Client: Menu > Administration > Access Control > Roles
2. Create custom role with required permissions
3. Assign role to user on VM or folder level

**B. Use Administrator Account** (development only):
```bash
export GOVC_USERNAME='administrator@vsphere.local'
```

---

## ❌ Export Failures

### Problem: Export job fails immediately

**Symptoms**:
```json
{
  "status": "failed",
  "error": "VM not found: /datacenter/vm/my-vm"
}
```

**Diagnosis**:
```bash
# List available VMs
curl http://localhost:8080/vms/list

# Check exact path
govc ls /datacenter/vm

# Get job error
curl http://localhost:8080/jobs/{job-id} | jq '.error'
```

**Solutions**:

**A. VM Path Incorrect**:
```bash
# Correct path format
/datacenter-name/vm/folder-name/vm-name

# Example
/DC1/vm/Production/web-server-01  # Correct
web-server-01                      # Wrong: missing path
/DC1/vm/web-server-01             # Wrong: missing folder
```

**B. VM Doesn't Exist**:
```bash
# List VMs to find correct path
curl http://localhost:8080/vms/list | jq -r '.vms[].path'
```

**C. Insufficient Disk Space**:
```bash
# Check available space
df -h /exports

# Free up space
rm -rf /exports/old-backups
# or increase volume size
```

---

### Problem: Export job stalls at certain percentage

**Symptoms**:
- Job stuck at 23%, 45%, or 78%
- Progress not updating for > 10 minutes

**Diagnosis**:
```bash
# Check job progress
curl http://localhost:8080/jobs/progress/{job-id}

# Check daemon logs for errors
docker logs --tail 100 hypervisord

# Check network activity
iftop -i eth0

# Check disk I/O
iostat -x 5
```

**Solutions**:

**A. Network Timeout**:
```bash
# Increase timeout in config
cat > /etc/hypervisord/config.yaml <<EOF
DownloadTimeout: 600  # 10 minutes (default: 120)
EOF

# Restart daemon
docker restart hypervisord
```

**B. Slow Disk I/O**:
```bash
# Use faster storage for exports
mount /dev/nvme0n1 /exports  # SSD

# Or reduce concurrent exports
# (decreases I/O contention)
```

**C. Large VMDK Files**:
```bash
# This is expected for large disks
# Progress updates are per-file
# Wait for current file to complete
```

---

### Problem: "Disk space full" during export

**Symptoms**:
```
Error: write /exports/vm1.vmdk: no space left on device
```

**Immediate Fix**:
```bash
# 1. Stop the failing job
curl -X POST http://localhost:8080/jobs/cancel \
  -d '{"job_ids": ["job-id"]}'

# 2. Free up space
df -h
rm -rf /exports/old-exports

# 3. Retry export
```

**Long-term Solutions**:

**A. Pre-flight Check** (automatic in v2.0+):
```bash
# Daemon checks space before export
# Needs 1.2× VM size available
```

**B. Use Compression**:
```bash
# Reduces export size by ~30%
{
  "compression": true
}
```

**C. Use Incremental Exports**:
```bash
# Enable CBT for 90% space savings
curl -X POST http://localhost:8080/cbt/enable \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'
```

**D. Increase Storage**:
```bash
# Expand volume
lvextend -L +500G /dev/vg/exports
resize2fs /dev/vg/exports

# Or add new volume
mount /dev/sdb1 /exports2
```

---

## ⚡ Performance Issues

### Problem: Exports are very slow

**Symptoms**:
- Export speed < 10 MB/s
- ETA shows many hours for small VMs

**Diagnosis**:
```bash
# Check current speed
curl http://localhost:8080/jobs/progress/{job-id} | jq '.speed_mbps'

# Check network bandwidth
iftop -i eth0

# Check disk I/O
iostat -x 1

# Check CPU usage
top
```

**Solutions**:

**A. Network Bottleneck**:
```bash
# Test network speed
iperf3 -c vcenter.example.com

# Solutions:
# 1. Use faster network (1 Gbps → 10 Gbps)
# 2. Reduce network latency
# 3. Check for packet loss
ping -c 100 vcenter.example.com | grep loss
```

**B. Disk I/O Bottleneck**:
```bash
# Use SSD for exports
mount /dev/nvme0n1 /exports

# Enable write caching
hdparm -W1 /dev/sda

# Use RAID for better performance
```

**C. Increase Parallel Workers**:
```bash
# Edit config
cat > /etc/hypervisord/config.yaml <<EOF
DownloadWorkers: 8  # Default: 4
EOF

# Restart daemon
docker restart hypervisord
```

**D. Use Incremental Exports**:
```bash
# 95% faster for subsequent backups
curl -X POST http://localhost:8080/cbt/enable \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'
```

---

### Problem: High CPU usage during exports

**Symptoms**:
- CPU at 100%
- System slow/unresponsive

**Diagnosis**:
```bash
# Check CPU usage
top
# Press '1' to see per-core usage

# Check daemon process
ps aux | grep hypervisord

# Check if compression is enabled
curl http://localhost:8080/jobs/{job-id} | jq '.compression'
```

**Solutions**:

**A. Compression Overhead**:
```bash
# Disable compression if CPU-limited
{
  "compression": false
}

# Trade-off: larger export size, less CPU
```

**B. Reduce Concurrent Jobs**:
```bash
# Limit concurrent exports
# In scheduler config
{
  "max_concurrent": 2
}
```

**C. Increase CPU Resources**:
```bash
# Docker: increase CPU limit
docker run --cpus="4" hypersdk/hypervisord

# Kubernetes: increase resources
resources:
  limits:
    cpu: "4"
```

---

## 🔄 CBT Issues

### Problem: "CBT not enabled" on VM

**Symptoms**:
```json
{
  "can_incremental": false,
  "reason": "CBT is not enabled on this VM"
}
```

**Diagnosis**:
```bash
# Check CBT status
curl -X POST http://localhost:8080/cbt/status \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'

# Expected output
{
  "cbt_enabled": false,
  "disks": [...]
}
```

**Solutions**:

**A. Enable CBT**:
```bash
# Method 1: API
curl -X POST http://localhost:8080/cbt/enable \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'

# Method 2: Python SDK
from hypersdk import HyperSDK
client = HyperSDK("http://localhost:8080")
client.enable_cbt("/datacenter/vm/my-vm")

# Note: VM will be powered off briefly
```

**B. Check VM Requirements**:
- vSphere 6.5+ required
- VM hardware version 7+
- VM must be powered off to enable CBT (first time only)

---

### Problem: Incremental export falls back to full export

**Symptoms**:
```
Warning: Incremental export not possible, performing full export
Reason: No previous export found
```

**This is Expected** when:
- First export of a VM
- CBT was disabled and re-enabled
- Base export was deleted
- VM was moved/renamed

**Solutions**:

**A. Perform Initial Full Export**:
```bash
# First export is always full
# Subsequent exports will be incremental
```

**B. Keep Base Exports**:
```bash
# Don't delete base exports!
# They're needed for incremental backups

# Retention policy
keep_base_export: true
delete_incremental_after: 30  # days
```

**C. Verify CBT is Working**:
```bash
# After first export, check
curl -X POST http://localhost:8080/incremental/analyze \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'

# Should show:
{
  "can_incremental": true,
  "last_export": {...}
}
```

---

## 🚨 API Errors

### Problem: "400 Bad Request" errors

**Symptoms**:
```json
{
  "error": "invalid request: vm_path is required",
  "timestamp": "2026-02-04T10:30:00Z"
}
```

**Common Causes**:

**A. Missing Required Fields**:
```bash
# Wrong
curl -X POST http://localhost:8080/jobs/submit \
  -d '{"output_path": "/exports"}'

# Correct
curl -X POST http://localhost:8080/jobs/submit \
  -d '{
    "vm_path": "/datacenter/vm/my-vm",
    "output_path": "/exports"
  }'
```

**B. Invalid JSON**:
```bash
# Wrong (missing quotes)
curl -X POST http://localhost:8080/jobs/submit \
  -d '{vm_path: /datacenter/vm/my-vm}'

# Correct
curl -X POST http://localhost:8080/jobs/submit \
  -H 'Content-Type: application/json' \
  -d '{"vm_path": "/datacenter/vm/my-vm"}'
```

**C. Invalid Field Values**:
```bash
# Wrong (invalid format)
{
  "format": "xyz"  # Invalid
}

# Correct
{
  "format": "ova"  # Valid: ova, ovf, vmdk
}
```

---

### Problem: "404 Not Found" for valid job ID

**Symptoms**:
```json
{
  "error": "job not found: job-12345"
}
```

**Diagnosis**:
```bash
# List all jobs
curl http://localhost:8080/jobs/query?all=true

# Check if job ID is correct
# Job IDs are case-sensitive
```

**Solutions**:

**A. Job ID Expired**:
```bash
# Jobs are kept for limited time (default: 1000 last jobs)
# Older jobs are removed from history
```

**B. Daemon Restarted**:
```bash
# In-memory jobs are lost on restart
# Use persistent storage (future feature)
```

**C. Wrong API URL**:
```bash
# Check API endpoint
curl http://localhost:8080/jobs/job-12345  # Correct
curl http://localhost:8080/job/job-12345   # Wrong
```

---

## 💾 Resource Problems

### Problem: Out of Memory (OOM) errors

**Symptoms**:
```
docker: Error response from daemon: OOM command not allowed when used memory > 'maxmemory'
```

**Diagnosis**:
```bash
# Check memory usage
free -h

# Check Docker container memory
docker stats hypervisord

# Check daemon logs
docker logs hypervisord | grep -i "out of memory"
```

**Solutions**:

**A. Increase Docker Memory Limit**:
```bash
# docker run
docker run -m 4g hypersdk/hypervisord

# docker-compose
services:
  hypervisord:
    mem_limit: 4g
```

**B. Reduce Concurrent Operations**:
```bash
# Reduce parallel workers
DownloadWorkers: 2  # Default: 4

# Reduce concurrent jobs
max_concurrent: 1
```

**C. Increase System Memory**:
```bash
# Add swap space
fallocate -l 4G /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
```

---

### Problem: File descriptor limit reached

**Symptoms**:
```
Error: too many open files
```

**Diagnosis**:
```bash
# Check current limits
ulimit -n

# Check process limits
cat /proc/$(pgrep hypervisord)/limits | grep "open files"
```

**Solutions**:

**A. Increase User Limits**:
```bash
# Edit /etc/security/limits.conf
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Logout and login again
```

**B. Increase Docker Limits**:
```bash
# docker run
docker run --ulimit nofile=65536:65536 hypersdk/hypervisord
```

**C. Increase Systemd Limits**:
```bash
# Edit /etc/systemd/system/hypervisord.service
[Service]
LimitNOFILE=65536

# Reload and restart
sudo systemctl daemon-reload
sudo systemctl restart hypervisord
```

---

## 🔍 Diagnostic Tools

### Health Check Script

```bash
#!/bin/bash
# HyperSDK Health Check

API_URL="${1:-http://localhost:8080}"

echo "=== HyperSDK Health Check ==="
echo ""

# 1. API Connectivity
echo "1. Checking API connectivity..."
if curl -sf "$API_URL/health" > /dev/null; then
    echo "   ✅ API is reachable"
else
    echo "   ❌ API is not reachable"
    exit 1
fi

# 2. Daemon Status
echo "2. Checking daemon status..."
STATUS=$(curl -s "$API_URL/status" | jq -r '.status')
echo "   Status: $STATUS"

# 3. vCenter Connection
echo "3. Checking vCenter connection..."
CAPS=$(curl -s "$API_URL/capabilities")
if echo "$CAPS" | jq -e '.capabilities' > /dev/null; then
    echo "   ✅ vCenter connected"
else
    echo "   ❌ vCenter connection failed"
fi

# 4. Disk Space
echo "4. Checking disk space..."
USAGE=$(df -h /exports | tail -1 | awk '{print $5}' | sed 's/%//')
echo "   Disk usage: $USAGE%"
if [ "$USAGE" -gt 90 ]; then
    echo "   ⚠️  Warning: Disk usage > 90%"
fi

# 5. Running Jobs
echo "5. Checking running jobs..."
JOBS=$(curl -s "$API_URL/jobs/query" -d '{"status": "running"}' | jq '.total')
echo "   Running jobs: $JOBS"

echo ""
echo "=== Health Check Complete ==="
```

### Log Analysis Script

```bash
#!/bin/bash
# Analyze HyperSDK logs

LOG_FILE="${1:-/var/log/hypervisord/daemon.log}"

echo "=== Log Analysis ==="
echo ""

# Errors in last hour
echo "Errors in last hour:"
journalctl -u hypervisord --since "1 hour ago" | grep -i error | wc -l

# Most common errors
echo ""
echo "Top 5 errors:"
journalctl -u hypervisord | grep -i error | \
  awk '{print $NF}' | sort | uniq -c | sort -rn | head -5

# Job statistics
echo ""
echo "Job statistics (today):"
echo "  Submitted: $(journalctl -u hypervisord --since today | grep -c 'job submitted')"
echo "  Completed: $(journalctl -u hypervisord --since today | grep -c 'job completed')"
echo "  Failed: $(journalctl -u hypervisord --since today | grep -c 'job failed')"
```

---

## 📞 Getting Help

If you can't resolve your issue:

1. **Check Documentation**:
   - [FAQ](FAQ.md)
   - [Quick Start](QUICK_START.md)
   - [API Reference](API_ENDPOINTS.md)

2. **Gather Information**:
   ```bash
   # Daemon version
   curl http://localhost:8080/status | jq '.version'

   # System info
   uname -a
   docker version

   # Recent logs
   docker logs --tail 100 hypervisord > logs.txt
   ```

3. **Create GitHub Issue**:
   - URL: https://github.com/ssahani/hypersdk/issues
   - Include: Steps to reproduce, logs, environment details

4. **Community Support**:
   - GitHub Discussions
   - Documentation examples

---

*Last Updated: 2026-02-04*
*For more help, see the [FAQ](FAQ.md) and [documentation](00-INDEX.md)*
