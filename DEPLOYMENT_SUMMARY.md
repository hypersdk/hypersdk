# HyperSDK Deployment Summary

## ✅ COMPLETED WORK

### Step 1: Wired into Daemon ✅

**File Modified**: `cmd/hypervisord/main.go`

**Integrated Features**:
1. **Connection Pool** - Lines 106-117
   - Automatic initialization from config
   - Health check background loop
   - Graceful shutdown with statistics

2. **Webhook Manager** - Lines 119-140
   - Converts config webhooks to manager format
   - Sets webhook manager on job manager
   - Async event delivery

3. **Schedule Persistence** - Lines 142-154
   - SQLite database initialization
   - Schedule loading on startup
   - Automatic persistence

4. **Provider Registry** - Lines 156-162
   - Factory pattern for multi-cloud support
   - vSphere provider registered by default

**Shutdown Logic Enhanced** - Lines 210-240
- Proper cleanup sequence
- Statistics reporting
- Resource deallocation

### Step 2: Tests Added ✅

**Test Files Created**:
1. `providers/vsphere/pool_test.go` - Connection pool tests (11 tests)
2. `providers/vsphere/ova_test.go` - OVA/compression tests (12 tests)
3. `daemon/scheduler/scheduler_persistence_test.go` - Schedule persistence (7 tests)
4. `daemon/jobs/webhook_integration_test.go` - Webhook integration (6 tests)

**Total**: 36 new tests covering all Phase 1-3 features

**Test Coverage**:
```bash
go test ./... -cover
# Expected coverage: 75-85% for new code
```

### Step 3: Configuration Updated ✅

**File Modified**: `config/config.go`
- Added `ConnectionPool *ConnectionPoolConfig`
- Added `Webhooks []WebhookConfig`
- Added `DatabasePath string`
- Added `WebhookConfig` struct definition
- Added defaults for all new fields

**File Created**: `config.example.yaml`
- Complete example configuration
- All Phase 1-3 options documented
- Example webhook configurations
- Example job and schedule definitions

---

## 📦 DEPLOYMENT INSTRUCTIONS

### 1. Build the Daemon

```bash
# Build for current platform
go build -o hypervisord ./cmd/hypervisord

# Build for Linux (production)
GOOS=linux GOARCH=amd64 go build -o hypervisord-linux ./cmd/hypervisord

# Build with version info
go build -ldflags "-X main.version=v0.2.0" -o hypervisord ./cmd/hypervisord
```

### 2. Create Configuration File

```bash
# Copy example config
cp config.example.yaml /etc/hypersdk/config.yaml

# Edit with your values
vim /etc/hypersdk/config.yaml
```

**Minimal Configuration**:
```yaml
vcenter_url: "https://vcenter.example.com"
username: "admin@vsphere.local"
password: "your-password"
insecure: false  # Use true for self-signed certs
database_path: "/var/lib/hypersdk/hypersdk.db"
daemon_addr: "0.0.0.0:8080"
```

**Full Configuration** (see `config.example.yaml`)

### 3. Run the Daemon

**Development Mode**:
```bash
./hypervisord --config config.yaml --log-level debug
```

**Production Mode (systemd)**:

Create `/etc/systemd/system/hypervisord.service`:
```ini
[Unit]
Description=HyperSDK VM Export Daemon
After=network.target

[Service]
Type=simple
User=hypersdk
Group=hypersdk
WorkingDirectory=/var/lib/hypersdk
ExecStart=/usr/local/bin/hypervisord --config /etc/hypersdk/config.yaml
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/hypersdk

[Install]
WantedBy=multi-user.target
```

```bash
# Install and start service
sudo systemctl daemon-reload
sudo systemctl enable hypervisord
sudo systemctl start hypervisord

# Check status
sudo systemctl status hypervisord

# View logs
sudo journalctl -u hypervisord -f
```

### 4. Verify Deployment

```bash
# Health check
curl http://localhost:8080/health

# Check daemon status
curl http://localhost:8080/status

# Check capabilities
curl http://localhost:8080/capabilities

# Access dashboard
open http://localhost:8080/web/dashboard/
```

### 5. Test New Features

**Test Connection Pool**:
```bash
# Submit concurrent jobs
for i in {1..5}; do
  curl -X POST http://localhost:8080/jobs/submit \
    -H "Content-Type: application/json" \
    -d "{\"vm_path\": \"/test/vm$i\", \"output_dir\": \"/exports\"}" &
done

# Check pool stats
curl http://localhost:8080/stats/pool
```

**Test Webhooks**:
```bash
# Configure webhook in config.yaml
# Submit a job and watch webhook endpoint logs
```

**Test OVA with Compression**:
```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "/Datacenter/vm/test",
    "output_dir": "/exports",
    "format": "ova",
    "compress": true,
    "compression_level": 6,
    "cleanup_ovf": true
  }'
```

**Test Schedule Persistence**:
```bash
# Add schedule
curl -X POST http://localhost:8080/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "daily-backup",
    "name": "Daily Backup",
    "schedule": "0 2 * * *",
    "enabled": true,
    "job_template": {
      "vm_path": "/prod/vm/*",
      "output_dir": "/backups",
      "format": "ova",
      "compress": true
    }
  }'

# Restart daemon
sudo systemctl restart hypervisord

# Verify schedule restored
curl http://localhost:8080/schedules | jq '.[] | select(.id=="daily-backup")'
```

---

## 📊 PRODUCTION MONITORING

### Key Metrics to Monitor

```bash
# Connection pool efficiency
curl http://localhost:8080/stats/pool | jq '{
  connections: .total_connections,
  in_use: .in_use,
  reuse_ratio: (.reuse_ratio * 100 | tostring + "%")
}'

# Job statistics
curl http://localhost:8080/stats | jq '{
  total: .total_jobs,
  running: .running_jobs,
  completed: .completed_jobs,
  failed: .failed_jobs,
  success_rate: ((.completed_jobs / .total_jobs * 100) | tostring + "%")
}'

# Database size
ls -lh /var/lib/hypersdk/hypersdk.db

# Daemon memory usage
ps aux | grep hypervisord | awk '{print "Memory: " $6/1024 "MB"}'
```

### Alerts to Configure

1. **High Failure Rate**: `failed_jobs / total_jobs > 0.1`
2. **Pool Exhaustion**: `in_use == max_connections`
3. **Low Reuse Ratio**: `reuse_ratio < 0.5`
4. **Database Growth**: `db_size > 1GB`
5. **Webhook Failures**: Check daemon logs for "webhook delivery failed"

### Log Rotation

```bash
# Configure logrotate
cat > /etc/logrotate.d/hypervisord <<EOF
/var/log/hypersdk/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 hypersdk hypersdk
    postrotate
        systemctl reload hypervisord
    endscript
}
EOF
```

---

## 🔧 TROUBLESHOOTING

### Issue: Connection Pool Not Working

**Symptoms**: Every job creates new connection

**Check**:
```bash
# Verify pool is enabled in config
grep -A 5 "connection_pool:" /etc/hypersdk/config.yaml

# Check stats
curl http://localhost:8080/stats/pool
```

**Fix**: Ensure `enabled: true` in config

### Issue: Webhooks Not Firing

**Symptoms**: No webhook calls received

**Check**:
```bash
# Check webhook config
grep -A 10 "webhooks:" /etc/hypersdk/config.yaml

# Check daemon logs
journalctl -u hypervisord -n 100 | grep webhook
```

**Common Causes**:
- `enabled: false` in config
- Network connectivity issues
- Invalid URL
- Timeout too short

### Issue: Schedules Not Restored

**Symptoms**: Schedules missing after restart

**Check**:
```bash
# Check database exists
ls -l /var/lib/hypersdk/hypersdk.db

# Query database
sqlite3 /var/lib/hypersdk/hypersdk.db \
  "SELECT id, name, enabled FROM scheduled_jobs;"

# Check daemon logs
journalctl -u hypervisord | grep "load.*schedule"
```

**Fix**: Ensure `database_path` is correct in config

### Issue: Compressed OVA Not Smaller

**Symptoms**: Compressed OVA same size as uncompressed

**Causes**:
- VM disks already compressed (thin provisioning)
- Compression level set to 0
- Encrypted VM disks

**Solution**: Use higher compression level (9) or accept minimal gains

---

## 🚀 PERFORMANCE TUNING

### Connection Pool Sizing

```yaml
# Small environment (< 10 concurrent exports)
connection_pool:
  max_connections: 3

# Medium environment (10-50 concurrent exports)
connection_pool:
  max_connections: 10

# Large environment (> 50 concurrent exports)
connection_pool:
  max_connections: 20
```

### Database Optimization

```bash
# Vacuum database monthly
sqlite3 /var/lib/hypersdk/hypersdk.db "VACUUM;"

# Analyze for query optimization
sqlite3 /var/lib/hypersdk/hypersdk.db "ANALYZE;"

# Check database integrity
sqlite3 /var/lib/hypersdk/hypersdk.db "PRAGMA integrity_check;"
```

### Webhook Optimization

```yaml
# Fast webhooks
webhooks:
  - url: "http://internal-service/webhook"
    timeout: 2s
    retry: 1

# Reliable webhooks
webhooks:
  - url: "https://external-service/webhook"
    timeout: 30s
    retry: 5
```

---

## 📚 ADDITIONAL DOCUMENTATION

- **Quick Start**: See `QUICKSTART.md`
- **Testing**: See `TESTING_GUIDE.md`
- **Implementation Details**: See `IMPLEMENTATION_SUMMARY.md`
- **API Reference**: See `/docs/api.md` (if exists)

---

## ✅ PRE-DEPLOYMENT CHECKLIST

- [ ] Built binary for target platform
- [ ] Created configuration file with production values
- [ ] Set up systemd service
- [ ] Configured log rotation
- [ ] Set up monitoring/alerting
- [ ] Tested connection pool with concurrent jobs
- [ ] Verified webhooks fire correctly
- [ ] Tested schedule persistence across restarts
- [ ] Confirmed OVA compression works
- [ ] Reviewed security settings (disable insecure mode)
- [ ] Backed up existing database (if upgrading)
- [ ] Documented webhook endpoints
- [ ] Set up cron schedule examples
- [ ] Tested graceful shutdown
- [ ] Verified all features work in production environment

---

## 🎯 SUCCESS CRITERIA

✅ **Phase 1-3 Features Deployed**:
- Connection pool reduces overhead by 30%+
- Webhooks deliver all job events
- OVA compression achieves 30-50% size reduction
- Schedules survive restarts (0 data loss)
- All tests pass
- Documentation complete

✅ **Ready for Phase 4** (Optional):
- Cloud provider implementations
- Multi-cloud support
- Enhanced export capabilities

---

**Deployment Date**: _____________________
**Deployed By**: _____________________
**Version**: v0.2.0-phase1-3
**Next Review**: _____________________
