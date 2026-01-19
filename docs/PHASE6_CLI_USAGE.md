# Phase 6 CLI Integration - Usage Guide

## Overview

Phase 6 features are now integrated into the `hyperexport` CLI tool, providing comprehensive monitoring, progress tracking, metrics collection, and audit logging capabilities.

## New CLI Flags

### Orchestration Flags

#### `--orchestrate`
Enable Phase 6 migration orchestration framework.

```bash
hyperexport --vm web-server-01 --orchestrate
```

#### `--progress-api PORT`
Start the Progress API server on the specified port for real-time progress tracking.

```bash
hyperexport --vm web-server-01 --progress-api :8080
```

**Endpoints available:**
- `GET /api/v1/progress` - Get all active migrations
- `GET /api/v1/progress/{taskId}` - Get specific task progress
- `GET /api/v1/stream/{taskId}` - Stream real-time progress updates (SSE)

#### `--metrics-api PORT`
Start the Metrics API server on the specified port for Prometheus scraping.

```bash
hyperexport --vm web-server-01 --metrics-api :9090
```

**Endpoints available:**
- `GET /metrics` - Prometheus metrics (text format)
- `GET /stats` - JSON statistics

#### `--audit-log PATH`
Enable audit logging to the specified file path.

```bash
hyperexport --vm web-server-01 --audit-log /var/log/hypersdk/audit.log
```

### Webhook Notification Flags

#### `--webhook-url URL`
Send webhook notifications to the specified URL.

```bash
hyperexport --vm web-server-01 --webhook-url https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

#### `--webhook-type TYPE`
Specify the webhook type: `slack`, `discord`, or `generic` (default).

```bash
hyperexport --vm web-server-01 \
  --webhook-url https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  --webhook-type slack
```

#### `--webhook-on-start`
Send webhook notification when migration starts (default: true).

#### `--webhook-on-complete`
Send webhook notification when migration completes successfully (default: true).

#### `--webhook-on-error`
Send webhook notification when migration fails (default: true).

## Usage Examples

### Example 1: Basic Export with Progress Tracking

Monitor export progress via HTTP API:

```bash
hyperexport --vm web-server-01 --progress-api :8080
```

In another terminal, query progress:

```bash
# Get all migrations
curl http://localhost:8080/api/v1/progress

# Stream real-time updates
curl http://localhost:8080/api/v1/stream/export-web-server-01-1234567890
```

### Example 2: Export with Prometheus Metrics

Enable metrics collection for Prometheus scraping:

```bash
hyperexport --vm db-server-01 \
  --metrics-api :9090
```

Configure Prometheus to scrape:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'hypersdk'
    static_configs:
      - targets: ['localhost:9090']
```

Query metrics:

```bash
# Prometheus format
curl http://localhost:9090/metrics

# JSON statistics
curl http://localhost:9090/stats
```

### Example 3: Comprehensive Monitoring Setup

Enable all monitoring features:

```bash
hyperexport --vm app-server-01 \
  --progress-api :8080 \
  --metrics-api :9090 \
  --audit-log /var/log/hypersdk/audit.log \
  --convert \
  --format ova \
  --compress
```

Features enabled:
- Real-time progress tracking (port 8080)
- Prometheus metrics (port 9090)
- Audit logging to file
- Automatic conversion after export
- OVA packaging with compression

### Example 4: Export with Slack Notifications

Send notifications to Slack:

```bash
hyperexport --vm prod-web-01 \
  --webhook-url https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  --webhook-type slack \
  --webhook-on-start \
  --webhook-on-complete \
  --webhook-on-error
```

You'll receive Slack messages when:
- Migration starts
- Migration completes successfully
- Migration fails with error details

### Example 5: Production Migration with Full Observability

Complete production-ready migration with all Phase 6 features:

```bash
# Create audit log directory
sudo mkdir -p /var/log/hypersdk
sudo chown $USER /var/log/hypersdk

# Run export with full monitoring
hyperexport --vm production-app-server \
  --output /backups/production-app-server \
  --format ova \
  --compress \
  --verify \
  --convert \
  --manifest \
  --progress-api :8080 \
  --metrics-api :9090 \
  --audit-log /var/log/hypersdk/audit.log \
  --webhook-url https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  --webhook-type slack
```

Monitor in real-time from another terminal:

```bash
# Watch progress
watch -n 1 'curl -s http://localhost:8080/api/v1/progress | jq'

# View metrics
curl http://localhost:9090/metrics | grep hypersdk

# Tail audit log
tail -f /var/log/hypersdk/audit.log | jq
```

### Example 6: Batch Migration with Monitoring

Export multiple VMs with comprehensive tracking:

```bash
# Create VM list file
cat > vms.txt <<EOF
production-web-01
production-web-02
production-db-01
production-app-01
EOF

# Run batch export with monitoring
hyperexport --batch vms.txt \
  --progress-api :8080 \
  --metrics-api :9090 \
  --audit-log /var/log/hypersdk/batch-audit.log \
  --webhook-url https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  --webhook-type slack
```

## Monitoring Dashboards

### Grafana Dashboard

Import the pre-built Grafana dashboard for hypersdk metrics:

```bash
# TODO: Add dashboard JSON file location
```

**Panels include:**
- Migration success rate
- Export duration over time
- Data transferred
- Active migrations
- Failure reasons

### Custom Queries

**Prometheus queries:**

```promql
# Total migrations
hypersdk_migrations_total

# Success rate
rate(hypersdk_migrations_total{status="success"}[5m]) / rate(hypersdk_migrations_total[5m])

# Average export duration
rate(hypersdk_export_duration_seconds_sum[5m]) / rate(hypersdk_export_duration_seconds_count[5m])

# Total bytes exported
hypersdk_bytes_exported_total
```

## Audit Log Format

Audit logs are written as JSON lines:

```json
{
  "timestamp": "2026-01-21T15:30:00Z",
  "event_type": "migration_start",
  "task_id": "export-web-server-01-1737473400",
  "vm_name": "web-server-01",
  "provider": "vsphere",
  "user": "admin",
  "ip_address": "",
  "metadata": {}
}
```

**Query audit logs:**

```bash
# Show all events for a specific VM
cat /var/log/hypersdk/audit.log | jq 'select(.vm_name == "web-server-01")'

# Show only failures
cat /var/log/hypersdk/audit.log | jq 'select(.event_type == "migration_failed")'

# Show migrations from a specific user
cat /var/log/hypersdk/audit.log | jq 'select(.user == "admin")'

# Count migrations by status
cat /var/log/hypersdk/audit.log | jq -r .event_type | sort | uniq -c
```

## Integration with Existing Features

Phase 6 features work seamlessly with all existing hyperexport functionality:

### With Automatic Conversion (Phase 2)

```bash
hyperexport --vm web-server-01 \
  --convert \
  --progress-api :8080 \
  --metrics-api :9090
```

Progress tracking includes:
- Export phase
- Conversion phase (tracked automatically)
- Overall completion

### With Manifest Generation (Phase 1)

```bash
hyperexport --vm web-server-01 \
  --manifest \
  --manifest-target qcow2 \
  --audit-log /var/log/hypersdk/audit.log
```

Audit log records:
- Export start/complete
- Manifest generation
- Checksums computed

### With Cloud Upload

```bash
hyperexport --vm web-server-01 \
  --upload s3://my-bucket/backups \
  --progress-api :8080 \
  --webhook-url https://hooks.slack.com/... \
  --webhook-type slack
```

Notifications include:
- Upload destination in metadata
- Total time including upload

## Troubleshooting

### Progress API not accessible

**Problem:** Cannot connect to progress API on specified port

**Solution:**
```bash
# Check if port is in use
sudo netstat -tlnp | grep 8080

# Try a different port
hyperexport --vm web-server-01 --progress-api :8081
```

### Metrics not appearing in Prometheus

**Problem:** Prometheus not scraping metrics

**Solution:**
1. Verify metrics endpoint is accessible:
   ```bash
   curl http://localhost:9090/metrics
   ```

2. Check Prometheus configuration:
   ```yaml
   scrape_configs:
     - job_name: 'hypersdk'
       static_configs:
         - targets: ['localhost:9090']
       scrape_interval: 15s
   ```

3. Reload Prometheus configuration:
   ```bash
   curl -X POST http://localhost:9090/-/reload
   ```

### Audit log permission denied

**Problem:** Cannot write to audit log file

**Solution:**
```bash
# Create directory with correct permissions
sudo mkdir -p /var/log/hypersdk
sudo chown $USER:$USER /var/log/hypersdk

# Or use a user-writable location
hyperexport --vm web-server-01 \
  --audit-log $HOME/.hypersdk/audit.log
```

### Webhook notifications not received

**Problem:** Webhooks not being sent

**Solution:**
1. Test webhook URL manually:
   ```bash
   curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
     -H 'Content-Type: application/json' \
     -d '{"text":"Test message"}'
   ```

2. Check hyperexport logs for webhook errors:
   ```bash
   # Enable debug logging
   export LOG_LEVEL=debug
   hyperexport --vm web-server-01 \
     --webhook-url https://hooks.slack.com/... \
     --webhook-type slack
   ```

3. Verify webhook type matches the service:
   ```bash
   # For Slack
   --webhook-type slack

   # For Discord
   --webhook-type discord

   # For generic webhooks
   --webhook-type generic
   ```

## Performance Considerations

### Memory Usage

- **Progress tracking:** ~1-2 MB per active migration
- **Metrics collection:** ~500 KB for counters/histograms
- **Audit logging:** Minimal (buffered writes to disk)

Total overhead: < 5 MB for typical usage

### Network Usage

- **Progress API:** ~1 KB per query
- **Metrics API:** ~10-20 KB per scrape
- **Webhooks:** ~500 bytes per notification

### Disk Usage

- **Audit logs:** ~1-2 KB per migration event
- **Log rotation:** Recommended for long-running deployments

Configure logrotate:

```bash
# /etc/logrotate.d/hypersdk
/var/log/hypersdk/audit.log {
    daily
    rotate 30
    compress
    delaycompress
    notifempty
    create 0644 user user
}
```

## Best Practices

1. **Always enable audit logging in production**
   ```bash
   --audit-log /var/log/hypersdk/audit.log
   ```

2. **Use progress tracking for long-running migrations**
   ```bash
   --progress-api :8080
   ```

3. **Integrate with Prometheus for metrics**
   ```bash
   --metrics-api :9090
   ```

4. **Configure webhooks for critical migrations**
   ```bash
   --webhook-url https://... --webhook-type slack
   ```

5. **Monitor resource usage**
   ```bash
   # Check server memory
   ps aux | grep hyperexport

   # Monitor audit log size
   du -h /var/log/hypersdk/audit.log
   ```

## Next Steps

- Configure Grafana dashboards for visualization
- Set up Prometheus alerts for failed migrations
- Integrate audit logs with SIEM systems
- Create custom webhook handlers for automation

## See Also

- [PHASE6_CLI_INTEGRATION.md](../PHASE6_CLI_INTEGRATION.md) - Technical implementation details
- [Phase 5 Monitoring Guide](./PHASE5_MONITORING_REPORTING.md) - Core monitoring features
- [Prometheus Metrics Reference](./PROMETHEUS_METRICS.md) - Available metrics
