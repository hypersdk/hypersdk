# Job Persistence with SQLite

## Overview

The job persistence layer provides durable storage for migration jobs using SQLite, ensuring that job state survives daemon restarts and providing historical job tracking for compliance and debugging.

## Features

- ✅ **Durable Storage**: Jobs survive daemon restarts
- ✅ **Historical Tracking**: Complete job history with timestamps
- ✅ **Fast Queries**: Indexed queries for filtering and sorting
- ✅ **Concurrent Access**: WAL mode for better concurrency
- ✅ **Automatic Pruning**: Clean up old completed jobs
- ✅ **Statistics**: Real-time job statistics and success rates
- ✅ **Metadata Support**: Store arbitrary job metadata as JSON

## Database Schema

```sql
CREATE TABLE jobs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    vm_name TEXT NOT NULL,
    vm_path TEXT NOT NULL,
    provider TEXT NOT NULL,
    output_dir TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    progress REAL DEFAULT 0,
    error TEXT,
    metadata TEXT,  -- JSON
    user TEXT,
    total_size INTEGER DEFAULT 0,
    files_count INTEGER DEFAULT 0
);

CREATE INDEX idx_status ON jobs(status);
CREATE INDEX idx_created_at ON jobs(created_at DESC);
CREATE INDEX idx_provider ON jobs(provider);
CREATE INDEX idx_user ON jobs(user);
CREATE INDEX idx_vm_name ON jobs(vm_name);
```

## Usage

### Creating a Job Store

```go
import "hypersdk/providers/common"

// Create or open database
store, err := common.NewSQLiteJobStore("/var/lib/hypersdk/jobs.db")
if err != nil {
    return fmt.Errorf("create job store: %w", err)
}
defer store.Close()
```

### Saving a Job

```go
job := &common.Job{
    ID:        "migration-20260121-001",
    Name:      "Export web-server-01",
    VMName:    "web-server-01",
    VMPath:    "/datacenter/vm/web-server-01",
    Provider:  "vsphere",
    OutputDir: "/exports/web-server-01",
    Status:    common.JobStatusPending,
    CreatedAt: time.Now(),
    User:      "admin",
    Metadata: map[string]interface{}{
        "format":   "ova",
        "compress": true,
    },
}

if err := store.SaveJob(job); err != nil {
    return fmt.Errorf("save job: %w", err)
}
```

### Loading a Job

```go
job, err := store.LoadJob("migration-20260121-001")
if err != nil {
    return fmt.Errorf("load job: %w", err)
}

fmt.Printf("Job %s: %s (%s)\n", job.ID, job.VMName, job.Status)
```

### Updating Job Status

```go
// Update to running
if err := store.UpdateJobStatus("migration-20260121-001", common.JobStatusRunning); err != nil {
    return fmt.Errorf("update status: %w", err)
}

// Update progress
if err := store.UpdateJobProgress("migration-20260121-001", 50.0); err != nil {
    return fmt.Errorf("update progress: %w", err)
}

// Update to completed
if err := store.UpdateJobStatus("migration-20260121-001", common.JobStatusCompleted); err != nil {
    return fmt.Errorf("update status: %w", err)
}
```

### Listing Jobs with Filters

```go
// List all jobs
allJobs, err := store.ListJobs(common.JobFilter{})

// List only running jobs
runningJobs, err := store.ListJobs(common.JobFilter{
    Status: common.JobStatusRunning,
})

// List jobs by provider
vsphereJobs, err := store.ListJobs(common.JobFilter{
    Provider: "vsphere",
})

// List jobs by user
userJobs, err := store.ListJobs(common.JobFilter{
    User: "admin",
})

// List recent jobs (last 10)
recentJobs, err := store.ListJobs(common.JobFilter{
    Limit: 10,
})

// List jobs in a time range
rangeJobs, err := store.ListJobs(common.JobFilter{
    Since: time.Now().Add(-24 * time.Hour),
    Until: time.Now(),
})
```

### Getting Job Statistics

```go
stats, err := store.GetJobStats()
if err != nil {
    return fmt.Errorf("get stats: %w", err)
}

fmt.Printf("Total Jobs: %d\n", stats.Total)
fmt.Printf("Completed: %d\n", stats.Completed)
fmt.Printf("Failed: %d\n", stats.Failed)
fmt.Printf("Running: %d\n", stats.Running)
fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate)
```

### Pruning Old Jobs

```go
// Delete completed/failed jobs older than 30 days
deleted, err := store.Prune(30 * 24 * time.Hour)
if err != nil {
    return fmt.Errorf("prune jobs: %w", err)
}

fmt.Printf("Pruned %d old jobs\n", deleted)
```

## Job Statuses

The job store supports the following statuses:

| Status | Description |
|--------|-------------|
| `pending` | Job created but not yet started |
| `running` | Job is currently executing |
| `completed` | Job finished successfully |
| `failed` | Job failed with error |
| `cancelled` | Job was cancelled by user |

## Integration with hypervisord Daemon

```go
// In hypervisord main.go

import "hypersdk/providers/common"

func main() {
    // Create job store
    store, err := common.NewSQLiteJobStore("/var/lib/hypersdk/jobs.db")
    if err != nil {
        log.Fatal("Failed to create job store:", err)
    }
    defer store.Close()

    // Create daemon with job store
    daemon := &Daemon{
        jobStore: store,
    }

    // When creating a new job
    job := &common.Job{
        ID:        generateJobID(),
        VMName:    vmName,
        Status:    common.JobStatusPending,
        CreatedAt: time.Now(),
    }
    store.SaveJob(job)

    // When job starts
    now := time.Now()
    job.StartedAt = &now
    job.Status = common.JobStatusRunning
    store.SaveJob(job)

    // Update progress periodically
    store.UpdateJobProgress(job.ID, 50.0)

    // When job completes
    completed := time.Now()
    job.CompletedAt = &completed
    job.Status = common.JobStatusCompleted
    store.SaveJob(job)
}
```

## Integration with hyperexport CLI

```go
// In hyperexport main.go

// Create job store for tracking exports
store, err := common.NewSQLiteJobStore(filepath.Join(os.Getenv("HOME"), ".hypersdk/jobs.db"))
if err != nil {
    log.Warn("Failed to create job store:", err)
    // Continue without persistence
}
defer store.Close()

// Create job record
if store != nil {
    job := &common.Job{
        ID:        fmt.Sprintf("export-%s-%d", vmName, time.Now().Unix()),
        Name:      fmt.Sprintf("Export %s", vmName),
        VMName:    vmName,
        VMPath:    vmPath,
        Provider:  provider,
        OutputDir: exportDir,
        Status:    common.JobStatusRunning,
        CreatedAt: time.Now(),
        User:      os.Getenv("USER"),
    }
    store.SaveJob(job)

    // On completion
    defer func() {
        if result.Success {
            job.Status = common.JobStatusCompleted
        } else {
            job.Status = common.JobStatusFailed
            job.Error = err.Error()
        }
        now := time.Now()
        job.CompletedAt = &now
        job.TotalSize = result.TotalSize
        job.FilesCount = len(result.Files)
        store.SaveJob(job)
    }()
}
```

## CLI Tool for Job Management

Create a CLI tool to query job history:

```bash
#!/bin/bash
# hyperjobs - Query job database

case "$1" in
    list)
        sqlite3 /var/lib/hypersdk/jobs.db "SELECT id, vm_name, status, datetime(created_at) FROM jobs ORDER BY created_at DESC LIMIT 20"
        ;;
    stats)
        sqlite3 /var/lib/hypersdk/jobs.db "SELECT status, COUNT(*) FROM jobs GROUP BY status"
        ;;
    show)
        sqlite3 /var/lib/hypersdk/jobs.db "SELECT * FROM jobs WHERE id = '$2'"
        ;;
    running)
        sqlite3 /var/lib/hypersdk/jobs.db "SELECT id, vm_name, progress FROM jobs WHERE status = 'running'"
        ;;
    failed)
        sqlite3 /var/lib/hypersdk/jobs.db "SELECT id, vm_name, error FROM jobs WHERE status = 'failed' ORDER BY created_at DESC LIMIT 10"
        ;;
    *)
        echo "Usage: hyperjobs {list|stats|show ID|running|failed}"
        ;;
esac
```

## Querying with SQL

Direct SQL queries for advanced use cases:

```bash
# List all completed migrations
sqlite3 /var/lib/hypersdk/jobs.db "SELECT vm_name, datetime(completed_at), total_size FROM jobs WHERE status = 'completed' ORDER BY completed_at DESC"

# Calculate average export duration
sqlite3 /var/lib/hypersdk/jobs.db "SELECT AVG((julianday(completed_at) - julianday(started_at)) * 86400) as avg_duration_seconds FROM jobs WHERE status = 'completed'"

# Find slow migrations (over 1 hour)
sqlite3 /var/lib/hypersdk/jobs.db "SELECT vm_name, (julianday(completed_at) - julianday(started_at)) * 24 as hours FROM jobs WHERE status = 'completed' AND hours > 1"

# Success rate by provider
sqlite3 /var/lib/hypersdk/jobs.db "SELECT provider, COUNT(*) as total, SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed FROM jobs GROUP BY provider"
```

## Performance Considerations

### Indexes

The job store creates indexes on frequently queried columns:
- `idx_status`: Fast filtering by job status
- `idx_created_at`: Efficient sorting by creation time
- `idx_provider`: Quick filtering by provider
- `idx_user`: Fast user-based queries
- `idx_vm_name`: Efficient VM name lookups

### WAL Mode

The database uses Write-Ahead Logging (WAL) mode for better concurrency:
- Multiple readers can access the database simultaneously
- Readers don't block writers
- Better performance for concurrent workloads

### Pruning

Regularly prune old jobs to prevent database growth:

```go
// Prune monthly
ticker := time.NewTicker(24 * time.Hour)
for range ticker.C {
    deleted, _ := store.Prune(30 * 24 * time.Hour)
    log.Info("Pruned old jobs", "deleted", deleted)
}
```

## Backup and Recovery

### Backup

```bash
# Backup job database
sqlite3 /var/lib/hypersdk/jobs.db ".backup /backups/jobs-$(date +%Y%m%d).db"

# Or use file copy (stop daemon first)
cp /var/lib/hypersdk/jobs.db /backups/jobs-$(date +%Y%m%d).db
```

### Recovery

```bash
# Restore from backup
sqlite3 /var/lib/hypersdk/jobs.db ".restore /backups/jobs-20260121.db"
```

### Export to JSON

```bash
# Export all jobs as JSON
sqlite3 /var/lib/hypersdk/jobs.db -json "SELECT * FROM jobs" > jobs.json
```

## Troubleshooting

### Database Locked Error

**Problem:** `database is locked` error

**Solution:**
```bash
# Check for processes holding the database
lsof /var/lib/hypersdk/jobs.db

# Ensure WAL mode is enabled
sqlite3 /var/lib/hypersdk/jobs.db "PRAGMA journal_mode"
# Should return: wal

# If not, enable it
sqlite3 /var/lib/hypersdk/jobs.db "PRAGMA journal_mode=WAL"
```

### Corrupted Database

**Problem:** Database file is corrupted

**Solution:**
```bash
# Check integrity
sqlite3 /var/lib/hypersdk/jobs.db "PRAGMA integrity_check"

# If corrupted, recover from backup
cp /backups/jobs-latest.db /var/lib/hypersdk/jobs.db

# Or rebuild from audit logs (if available)
```

### Slow Queries

**Problem:** Queries are slow

**Solution:**
```bash
# Analyze query plan
sqlite3 /var/lib/hypersdk/jobs.db "EXPLAIN QUERY PLAN SELECT * FROM jobs WHERE status = 'running'"

# Rebuild indexes
sqlite3 /var/lib/hypersdk/jobs.db "REINDEX"

# Vacuum database
sqlite3 /var/lib/hypersdk/jobs.db "VACUUM"
```

## Best Practices

1. **Always close the store properly**
   ```go
   defer store.Close()
   ```

2. **Handle errors gracefully**
   ```go
   if err := store.SaveJob(job); err != nil {
       log.Error("Failed to save job", "error", err)
       // Don't fail the migration if persistence fails
   }
   ```

3. **Prune old jobs regularly**
   ```go
   go func() {
       ticker := time.NewTicker(24 * time.Hour)
       for range ticker.C {
           store.Prune(30 * 24 * time.Hour)
       }
   }()
   ```

4. **Use transactions for batch operations** (future enhancement)

5. **Monitor database size**
   ```bash
   du -h /var/lib/hypersdk/jobs.db
   ```

## Future Enhancements

- [ ] Transaction support for atomic multi-job operations
- [ ] Database replication for high availability
- [ ] PostgreSQL backend option for enterprise deployments
- [ ] Job dependency tracking
- [ ] Scheduled job execution
- [ ] Job templates and presets

## See Also

- [Phase 6 CLI Integration](./PHASE6_CLI_INTEGRATION.md)
- [Audit Logging](./AUDIT_LOGGING.md)
- [Metrics Collection](./METRICS.md)
