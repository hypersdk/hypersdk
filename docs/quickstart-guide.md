# HyperSDK Quick Start Guide

## New Features Available (Phases 1-3)

### 1. Connection Pooling

Reduce connection overhead for concurrent exports by reusing vSphere connections.

```go
import "hypersdk/providers/vsphere"

// Create connection pool
poolConfig := &vsphere.PoolConfig{
    MaxConnections:      5,
    IdleTimeout:         5 * time.Minute,
    HealthCheckInterval: 30 * time.Second,
}

pool := vsphere.NewConnectionPool(cfg, poolConfig, logger)
defer pool.Close()

// Get connection from pool
client, err := pool.Get(ctx)
if err != nil {
    log.Fatal(err)
}

// Use client...
result, err := client.ExportOVF(ctx, vmPath, opts)

// Return to pool
pool.Put(client)

// View statistics
stats := pool.Stats()
fmt.Printf("Reuse ratio: %.2f%%\n", stats["reuse_ratio"].(float64)*100)
```

### 2. Webhook Notifications

Get real-time notifications for all job lifecycle events.

**Configuration** (`config.yaml`):
```yaml
webhooks:
  - url: "https://your-server.com/webhook"
    events: ["job.started", "job.completed", "job.failed"]
    headers:
      Authorization: "Bearer your-token"
    timeout: 10s
    retry: 3
    enabled: true
```

**Webhook Payload Example**:
```json
{
  "event": "job.completed",
  "timestamp": "2026-01-20T10:30:00Z",
  "data": {
    "job_id": "abc123",
    "job_name": "Export Production VM",
    "vm_path": "/Datacenter/vm/prod-web-01",
    "duration_seconds": 1234.56,
    "ovf_path": "/exports/prod-web-01.ovf"
  }
}
```

### 3. OVA Export with Compression

Export VMs as compressed OVA archives.

```go
opts := vsphere.ExportOptions{
    Format:           "ova",
    Compress:         true,
    CompressionLevel: 6,        // 0-9, default is 6
    CleanupOVF:       true,     // Remove intermediate .ovf/.vmdk files
    OutputPath:       "/exports",
}

result, err := client.ExportOVF(ctx, vmPath, opts)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("OVA created: %s\n", result.OVAPath)
fmt.Printf("Format: %s\n", result.Format)
fmt.Printf("Size: %d bytes\n", result.TotalSize)
```

**File naming**:
- Uncompressed: `vm-name.ova`
- Compressed: `vm-name.ova.gz`

### 4. Persistent Job Scheduling

Schedule recurring exports that survive daemon restarts.

```go
import "hypersdk/daemon/scheduler"

// Create scheduler with persistence
store := store.NewSQLiteStore("./hypersdk.db")
sched := scheduler.NewScheduler(jobManager, logger)
sched.SetStore(store)

// Define scheduled job
scheduledJob := &scheduler.ScheduledJob{
    ID:          "daily-backup",
    Name:        "Daily Production Backup",
    Description: "Export production VMs every night",
    Schedule:    "0 2 * * *", // 2 AM daily (cron format)
    Enabled:     true,
    JobTemplate: models.JobDefinition{
        Name:       "prod-backup",
        VMPath:     "/Datacenter/vm/prod-*",
        OutputPath: "/backups",
        Format:     "ova",
        Compress:   true,
    },
}

// Add to scheduler
err := sched.AddScheduledJob(scheduledJob)

// Start scheduler
sched.Start()

// On daemon startup - restore all schedules
sched.LoadSchedules()
```

**Query execution history**:
```go
history, err := store.GetExecutionHistory("daily-backup", 10)
for _, exec := range history {
    fmt.Printf("%s: Job %s - %s (%.2fs)\n",
        exec.ExecutedAt,
        exec.JobID,
        exec.Status,
        exec.DurationSeconds)
}
```

### 5. Unified Provider Interface

Work with multiple cloud providers using a single interface.

```go
import "hypersdk/providers"

// Create provider registry
registry := providers.NewRegistry()

// Register providers
registry.Register(providers.ProviderVSphere, vsphere.NewProvider)
registry.Register(providers.ProviderAWS, aws.NewProvider)
registry.Register(providers.ProviderAzure, azure.NewProvider)

// Create provider instance
config := providers.ProviderConfig{
    Type:     providers.ProviderVSphere,
    Endpoint: "https://vcenter.example.com",
    Username: "admin@vsphere.local",
    Password: "password",
    Insecure: true,
}

provider, err := registry.Create(providers.ProviderVSphere, config)
defer provider.Disconnect()

// Connect
err = provider.Connect(ctx, config)

// List VMs
vms, err := provider.SearchVMs(ctx, "web-server")

// Export VM
opts := providers.ExportOptions{
    OutputPath: "/exports",
    Format:     "ova",
    Compress:   true,
}

result, err := provider.ExportVM(ctx, vms[0].ID, opts)
```

**Check provider capabilities**:
```go
caps := provider.GetExportCapabilities()
fmt.Printf("Supported formats: %v\n", caps.SupportedFormats)
fmt.Printf("Compression: %v\n", caps.SupportsCompression)
fmt.Printf("Streaming: %v\n", caps.SupportsStreaming)
```

---

## Complete Example: Daemon with All Features

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "hypersdk/config"
    "hypersdk/daemon/jobs"
    "hypersdk/daemon/scheduler"
    "hypersdk/daemon/store"
    "hypersdk/daemon/webhooks"
    "hypersdk/logger"
    "hypersdk/providers"
    "hypersdk/providers/vsphere"
)

func main() {
    // Load configuration
    cfg, err := config.FromFile("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Create logger
    log := logger.New(cfg.LogLevel)

    // Initialize database
    db, err := store.NewSQLiteStore("./hypersdk.db")
    if err != nil {
        log.Fatal("Failed to open database:", err)
    }
    defer db.Close()

    // Create job manager
    jobManager := jobs.NewManager(log, detector)

    // Setup webhooks
    if len(cfg.Webhooks) > 0 {
        webhookMgr := webhooks.NewManager(cfg.Webhooks, log)
        jobManager.SetWebhookManager(webhookMgr)
        log.Info("Webhooks enabled", "count", len(cfg.Webhooks))
    }

    // Setup connection pool
    if cfg.ConnectionPool != nil && cfg.ConnectionPool.Enabled {
        poolCfg := &vsphere.PoolConfig{
            MaxConnections:      cfg.ConnectionPool.MaxConnections,
            IdleTimeout:         cfg.ConnectionPool.IdleTimeout,
            HealthCheckInterval: cfg.ConnectionPool.HealthCheckInterval,
        }
        pool := vsphere.NewConnectionPool(cfg, poolCfg, log)
        defer pool.Close()
        log.Info("Connection pool enabled", "max", poolCfg.MaxConnections)
    }

    // Setup scheduler with persistence
    sched := scheduler.NewScheduler(jobManager, log)
    sched.SetStore(db)

    // Load existing schedules
    if err := sched.LoadSchedules(); err != nil {
        log.Error("Failed to load schedules", "error", err)
    }

    sched.Start()
    defer sched.Stop()

    // Setup provider registry
    registry := providers.NewRegistry()
    registry.Register(providers.ProviderVSphere, vsphere.NewProvider)
    log.Info("Provider registry initialized")

    // Start API server
    // api.Start(cfg.DaemonAddr, jobManager, sched)

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Info("Shutting down gracefully...")
    jobManager.Shutdown()
}
```

---

## Testing New Features

### Test Connection Pooling
```bash
# Run with multiple concurrent jobs
for i in {1..10}; do
    curl -X POST http://localhost:8080/jobs \
        -H "Content-Type: application/json" \
        -d '{
            "vm_path": "/Datacenter/vm/test-'$i'",
            "output_dir": "/exports",
            "format": "ova"
        }' &
done

# Check pool statistics
curl http://localhost:8080/stats/pool
```

### Test Webhooks
```bash
# Start a webhook receiver
python3 -m http.server 8090 &

# Configure webhook in config.yaml
# Submit a job and watch for webhook calls
tail -f webhook-receiver.log
```

### Test OVA Compression
```bash
# Export with compression
hypersdk export \
    --vm "/Datacenter/vm/large-vm" \
    --format ova \
    --compress \
    --compression-level 9 \
    --cleanup-ovf \
    --output /exports

# Compare sizes
ls -lh /exports/*.ova*
```

### Test Schedule Persistence
```bash
# Add a schedule
curl -X POST http://localhost:8080/schedules \
    -H "Content-Type: application/json" \
    -d '{
        "id": "test-schedule",
        "name": "Test Schedule",
        "schedule": "*/5 * * * *",
        "enabled": true,
        "job_template": {
            "vm_path": "/Datacenter/vm/test",
            "output_dir": "/exports"
        }
    }'

# Restart daemon
systemctl restart hypervisord

# Verify schedule restored
curl http://localhost:8080/schedules
```

---

## Configuration Reference

### Complete config.yaml
```yaml
# vCenter Connection
vcenter:
  url: "https://vcenter.example.com"
  username: "admin@vsphere.local"
  password: "SecurePassword123"
  insecure: true
  timeout: 3600

# Connection Pooling (Phase 1.1)
connection_pool:
  enabled: true
  max_connections: 5
  idle_timeout: 5m
  health_check_interval: 30s

# Webhooks (Phase 1.2)
webhooks:
  - url: "https://example.com/webhook"
    events: ["*"]  # or specific: ["job.started", "job.completed"]
    headers:
      Authorization: "Bearer token123"
      X-Custom-Header: "value"
    timeout: 10s
    retry: 3
    enabled: true

  - url: "https://backup.example.com/notify"
    events: ["job.completed"]
    enabled: true

# Export Defaults (Phase 2.1, 2.2)
export:
  format: "ova"              # ovf or ova
  compress: true             # Enable gzip compression
  compression_level: 6       # 0-9 (0=none, 9=best)
  cleanup_ovf: true          # Remove intermediate OVF files
  parallel_downloads: 3      # Concurrent file downloads
  remove_cdrom: true         # Remove CD/DVD devices

# Database (Phase 2.3)
database:
  path: "./hypersdk.db"
  enable_wal: true

# Daemon
daemon:
  address: "0.0.0.0:8080"
  log_level: "info"
```

---

## Troubleshooting

### Connection Pool Issues
```bash
# Check pool statistics
curl http://localhost:8080/stats/pool

# Look for:
# - reuse_ratio < 0.5: Pool may be too small
# - total_connections == max_connections: Pool exhausted
```

### Webhook Delivery Failures
```bash
# Check daemon logs
journalctl -u hypervisord -f | grep webhook

# Common issues:
# - Network connectivity
# - Invalid URL
# - Timeout too short
# - Server rejecting requests
```

### OVA Validation Failures
```bash
# Validate OVA structure
hypersdk validate-ova /exports/vm-name.ova

# Extract for inspection
tar -tzf vm-name.ova | head
# First file should be .ovf
```

### Schedule Not Restoring
```bash
# Check database
sqlite3 hypersdk.db "SELECT * FROM scheduled_jobs;"

# Check daemon startup logs
journalctl -u hypervisord -n 100 | grep schedule
```

---

## API Endpoints

### Jobs
- `POST /jobs` - Submit new job
- `GET /jobs` - List all jobs
- `GET /jobs/:id` - Get job details
- `DELETE /jobs/:id` - Cancel job

### Schedules
- `POST /schedules` - Create scheduled job
- `GET /schedules` - List all schedules
- `GET /schedules/:id` - Get schedule details
- `PUT /schedules/:id` - Update schedule
- `DELETE /schedules/:id` - Delete schedule
- `POST /schedules/:id/trigger` - Manually trigger
- `GET /schedules/:id/history` - Get execution history

### Statistics
- `GET /stats` - Daemon statistics
- `GET /stats/pool` - Connection pool statistics
- `GET /stats/schedules` - Schedule statistics

---

## Next Steps

1. **Test the new features** with your production VMs
2. **Configure webhooks** for your monitoring system
3. **Set up schedules** for regular backups
4. **Monitor pool statistics** to optimize max_connections
5. **Review execution history** to identify failures

For more information, see [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md).
