# HyperSDK New Features Guide

## Table of Contents
1. [HyperCTL Enhancements](#hyperctl-enhancements)
2. [HyperExport Daemon Integration](#hyperexport-daemon-integration)
3. [Web Dashboard Manifest Converter](#web-dashboard-manifest-converter)
4. [Usage Examples](#usage-examples)

---

## HyperCTL Enhancements

HyperCTL has been enhanced with powerful new commands for managing schedules, webhooks, and monitoring jobs in real-time.

### Schedule Management

Create, manage, and monitor scheduled VM export jobs with cron expressions.

#### List Schedules
```bash
hyperctl schedules list
```

Displays all scheduled jobs with:
- Schedule ID
- Name
- Cron schedule
- Enabled/disabled status
- Last run time
- Next run time

#### Create Schedule
```bash
hyperctl schedules create <name> <cron-schedule> -vm <vm-path> [-output <output-dir>]
```

**Example:**
```bash
# Daily backup at 2 AM
hyperctl schedules create daily-backup "0 2 * * *" -vm /dc/vm/prod/web01 -output /backups/web01

# Weekly backup every Sunday at midnight
hyperctl schedules create weekly-backup "0 0 * * 0" -vm /dc/vm/prod/db01

# Hourly backup during business hours (9 AM - 5 PM, weekdays)
hyperctl schedules create business-hours "0 9-17 * * 1-5" -vm /dc/vm/dev/app01
```

**Cron Format:**
```
* * * * *
│ │ │ │ │
│ │ │ │ └─── Day of week (0-6, Sunday=0)
│ │ │ └───── Month (1-12)
│ │ └─────── Day of month (1-31)
│ └───────── Hour (0-23)
└─────────── Minute (0-59)
```

#### Enable/Disable Schedule
```bash
# Enable a schedule
hyperctl schedules enable <schedule-id>

# Disable a schedule
hyperctl schedules disable <schedule-id>
```

#### Trigger Schedule Immediately
```bash
hyperctl schedules trigger <schedule-id>
```

Manually triggers a scheduled job without waiting for the next scheduled time.

#### Delete Schedule
```bash
hyperctl schedules delete <schedule-id>
```

---

### Webhook Management

Configure webhook notifications for job lifecycle events.

#### List Webhooks
```bash
hyperctl webhooks list
```

Shows all configured webhooks with URL, subscribed events, and enabled status.

#### Add Webhook
```bash
hyperctl webhooks add <url> [events...]
```

**Example:**
```bash
# Add Slack webhook for all job events
hyperctl webhooks add https://hooks.slack.com/services/xxx job.started job.completed job.failed

# Add webhook for completed jobs only
hyperctl webhooks add https://example.com/webhook job.completed

# Add webhook with default events (job.started, job.completed, job.failed)
hyperctl webhooks add https://example.com/webhook
```

**Available Events:**
- `job.started` - Job execution started
- `job.completed` - Job completed successfully
- `job.failed` - Job failed with error
- `job.progress` - Job progress update
- `schedule.triggered` - Scheduled job triggered

#### Delete Webhook
```bash
hyperctl webhooks delete <index>
```

#### Test Webhook
```bash
hyperctl webhooks test <index>
```

Sends a test payload to verify webhook configuration.

---

### Real-Time Job Monitoring

#### Watch Job Progress
```bash
hyperctl watch <job-id>
```

Monitors a job in real-time with live progress updates. Press Ctrl+C to exit (job continues running).

**Output Example:**
```
[10:30:15] running 45.2% - Exporting disks
[10:30:17] running 52.8% - Downloading disk1.vmdk
[10:30:19] completed - Export finished
✅ Job completed successfully!
```

#### View Job Logs
```bash
# View all logs
hyperctl logs <job-id>

# Follow logs in real-time
hyperctl logs -f <job-id>

# View last 50 lines
hyperctl logs -tail 50 <job-id>
```

---

## HyperExport Daemon Integration

HyperExport can now submit jobs to the daemon instead of running exports directly, enabling centralized management and scheduling.

### Submit Job to Daemon
```bash
hyperexport --use-daemon -vm <vm-name> [options]
```

**Example:**
```bash
# Submit export job to daemon
hyperexport --use-daemon -vm /dc/vm/prod/web01 -output /exports/web01 -format ova

# Submit with automatic progress watching
hyperexport --use-daemon -vm /dc/vm/prod/db01 --watch

# Submit with custom daemon URL
hyperexport --use-daemon --daemon-url http://remote-host:8080 -vm /dc/vm/app01
```

### Watch Daemon Job
```bash
hyperexport --daemon-watch <job-id>
```

Monitor a specific job submitted to the daemon.

### List Daemon Jobs
```bash
# List all jobs
hyperexport --daemon-list all

# List running jobs only
hyperexport --daemon-list running

# List completed jobs
hyperexport --daemon-list completed

# List failed jobs
hyperexport --daemon-list failed
```

**Quiet Mode (for scripting):**
```bash
hyperexport --daemon-list running --quiet
# Output: job-id\tstatus\tvm-path
```

### Check Daemon Status
```bash
hyperexport --daemon-status
```

Displays daemon health, uptime, active jobs, and system metrics.

### Create Scheduled Export
```bash
hyperexport --daemon-schedule "name:cron" -vm <vm-name>
```

**Example:**
```bash
# Daily backup at 3 AM
hyperexport --daemon-schedule "daily-backup:0 3 * * *" -vm /dc/vm/prod/web01

# Weekly Sunday backup
hyperexport --daemon-schedule "weekly:0 0 * * 0" -vm /dc/vm/prod/db01
```

---

## Web Dashboard Manifest Converter

The web dashboard now includes a powerful "Manifest Converter" tab for one-shot export and conversion.

### Accessing the Manifest Converter

1. Open the dashboard: `http://localhost:8080`
2. Click the "📦 Manifest Converter" tab
3. Fill in the export configuration
4. Click "🚀 Start Export & Conversion"

### Features

#### Step 1: Export Configuration
- **VM Path**: Path to the VM to export (e.g., `/datacenter/vm/production/web01`)
- **Output Directory**: Where to save exported files (e.g., `/exports/web01`)
- **Target Format**: Choose conversion target
  - **QCOW2** (Recommended) - KVM/QEMU native format
  - **RAW** - Universal raw disk image
  - **VDI** - VirtualBox format
- **Auto-convert**: Automatically convert after export
- **Compress**: Compress exported OVA
- **Verify**: Verify checksums after conversion

#### Step 2: Conversion Options
- **Hyper2KVM Binary Path**: Path to hyper2kvm tool (auto-detected if empty)
- **Conversion Timeout**: Maximum time for conversion (default: 120 minutes)
- **Stream Output**: Stream conversion logs to dashboard

### How It Works

1. **Export**: VM exported from vSphere in OVF/OVA format
2. **Manifest Generation**: Artifact Manifest v1.0 created with disk metadata
3. **Conversion**: Hyper2KVM converts disks to target format
4. **Verification**: Checksums verified for data integrity
5. **Result**: KVM-ready disk images with conversion report

### Artifact Manifest Structure

```json
{
  "version": "1.0",
  "source": {
    "provider": "vsphere",
    "vm_name": "production-web-01",
    "export_date": "2026-01-22T10:30:00Z"
  },
  "disks": [
    {
      "path": "production-web-01-disk1.vmdk",
      "format": "vmdk",
      "size": 107374182400,
      "checksum": "sha256:abc123def456..."
    }
  ],
  "conversion": {
    "target_format": "qcow2",
    "verify_checksums": true
  }
}
```

### Monitoring Conversion

The dashboard displays real-time status:
- **Job ID**: Unique identifier for tracking
- **Status**: Current state (pending/running/completed/failed)
- **Progress**: Visual progress bar with percentage
- **Logs**: Streaming conversion output

---

## Usage Examples

### Example 1: Scheduled Backups with Webhooks

Setup automated backups with Slack notifications:

```bash
# 1. Add Slack webhook
hyperctl webhooks add https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
    job.started job.completed job.failed

# 2. Create daily backup schedule
hyperctl schedules create prod-backup "0 2 * * *" \
    -vm /dc/vm/prod/db01 \
    -output /backups/db01

# 3. Create weekly archive
hyperctl schedules create weekly-archive "0 0 * * 0" \
    -vm /dc/vm/prod/web01 \
    -output /archives/web01
```

Now you'll receive Slack notifications whenever backups run!

### Example 2: Batch Export via Daemon

Export multiple VMs using the daemon:

```bash
# Create VM list
cat > vms.txt << EOF
/dc/vm/prod/web01
/dc/vm/prod/web02
/dc/vm/prod/app01
/dc/vm/prod/db01
EOF

# Submit batch export
for vm in $(cat vms.txt); do
    hyperexport --use-daemon -vm "$vm" -format ova -compress
done

# Monitor all jobs
hyperexport --daemon-list running

# Watch specific job
hyperexport --daemon-watch job-12345
```

### Example 3: One-Shot Conversion

Export and convert in a single command:

```bash
hyperexport \
    -vm /dc/vm/prod/app01 \
    -output /exports/app01 \
    -format ova \
    -manifest \
    -manifest-target qcow2 \
    -convert \
    -verify
```

This will:
1. Export VM as OVA
2. Generate Artifact Manifest
3. Convert to QCOW2 format
4. Verify checksums
5. Output KVM-ready disks

### Example 4: Web Dashboard Workflow

1. **Open Dashboard**: Navigate to `http://localhost:8080`
2. **Monitor Jobs**: Check the "Dashboard" tab for active jobs
3. **View All Jobs**: Switch to "Jobs" tab for complete job history
4. **Start Conversion**: Use "Manifest Converter" tab for new exports:
   - Enter VM path: `/dc/vm/prod/web01`
   - Set output: `/exports/web01`
   - Choose format: QCOW2
   - Enable auto-convert
   - Click "Start Export & Conversion"
5. **Monitor Progress**: Watch real-time progress and logs
6. **Download Results**: Access converted files in output directory

### Example 5: Troubleshooting Failed Jobs

```bash
# List failed jobs
hyperctl query -status failed

# Get detailed job info
hyperctl query -id job-12345

# View error logs
hyperctl logs job-12345

# Check daemon status
hyperctl status
```

---

## Best Practices

### Schedule Management
- Use descriptive schedule names: `prod-daily-backup` instead of `backup1`
- Set realistic schedules to avoid overlapping jobs
- Monitor schedule execution: `hyperctl schedules list`
- Disable unused schedules instead of deleting them

### Webhook Configuration
- Test webhooks after creation: `hyperctl webhooks test <index>`
- Subscribe only to necessary events to reduce noise
- Use webhook URLs with authentication/security tokens
- Monitor webhook delivery failures in logs

### Daemon Integration
- Use `--use-daemon` for long-running exports to enable monitoring
- Enable `--watch` flag for immediate feedback on job progress
- Check daemon status regularly: `hyperexport --daemon-status`
- Use quiet mode (`--quiet`) for scripting and automation

### Manifest Conversion
- Always enable checksum verification for production exports
- Use QCOW2 format for best compression and features
- Set appropriate conversion timeout based on VM size
- Enable streaming output to troubleshoot conversion issues

---

## Troubleshooting

### Schedule Not Running
```bash
# Check if schedule is enabled
hyperctl schedules list

# Enable schedule
hyperctl schedules enable <schedule-id>

# Manually trigger to test
hyperctl schedules trigger <schedule-id>

# Check daemon logs
journalctl -u hypervisord -f
```

### Webhook Not Firing
```bash
# List webhooks
hyperctl webhooks list

# Test webhook
hyperctl webhooks test <index>

# Check webhook configuration
curl -X POST <webhook-url> \
    -H "Content-Type: application/json" \
    -d '{"event":"test","data":{}}'
```

### Daemon Connection Failed
```bash
# Check daemon is running
systemctl status hypervisord

# Verify daemon URL
hyperexport --daemon-status --daemon-url http://localhost:8080

# Check firewall
sudo firewall-cmd --list-ports
sudo firewall-cmd --add-port=8080/tcp --permanent
```

### Conversion Failed
```bash
# Check hyper2kvm is installed
which hyper2kvm

# Verify manifest was generated
ls -l /exports/*/manifest.json

# Check conversion logs
hyperctl logs -f <job-id>

# Manually test hyper2kvm
hyper2kvm -manifest /exports/vm01/manifest.json -output /test
```

---

## API Reference

### Schedule Endpoints
- `GET /schedules` - List all schedules
- `POST /schedules` - Create new schedule
- `GET /schedules/{id}` - Get schedule details
- `PUT /schedules/{id}` - Update schedule
- `DELETE /schedules/{id}` - Delete schedule
- `POST /schedules/{id}/enable` - Enable schedule
- `POST /schedules/{id}/disable` - Disable schedule
- `POST /schedules/{id}/trigger` - Trigger schedule now

### Webhook Endpoints
- `GET /webhooks` - List all webhooks
- `POST /webhooks` - Add new webhook
- `DELETE /webhooks/{index}` - Delete webhook
- `POST /webhooks/{index}/test` - Test webhook

### Job Endpoints
- `POST /jobs/submit` - Submit new job
- `GET /jobs/query` - Query jobs
- `GET /jobs/{id}` - Get job details
- `GET /jobs/logs/{id}` - Get job logs
- `POST /jobs/{id}/cancel` - Cancel job

---

## Migration from Previous Versions

If upgrading from HyperSDK v0.1.x:

1. **Update Configuration**
   ```yaml
   # Add to config.yaml
   schedules: []
   webhooks: []
   ```

2. **Restart Daemon**
   ```bash
   systemctl restart hypervisord
   ```

3. **Verify New Features**
   ```bash
   hyperctl schedules list
   hyperctl webhooks list
   hyperexport --daemon-status
   ```

4. **Migrate Existing Cron Jobs**
   ```bash
   # Old: crontab -e
   0 2 * * * /usr/local/bin/hyperexport -vm /dc/vm/prod/web01 -output /backups

   # New: hyperctl schedule
   hyperctl schedules create daily-backup "0 2 * * *" -vm /dc/vm/prod/web01 -output /backups
   ```

---

## Conclusion

These new features significantly enhance HyperSDK's capabilities:

- **Schedule Management**: Automate recurring exports without external cron
- **Webhook Integration**: Get instant notifications on job events
- **Real-Time Monitoring**: Watch jobs progress live
- **Daemon Integration**: Centralized job management
- **One-Shot Conversion**: Export and convert in a single operation

For more information, see:
- [Installation Guide](installation-guide.md)
- [Configuration Reference](configuration-reference.md)
- [Migration Workflows](migration-workflows.md)
- [API Documentation](../daemon/openapi/spec.yaml)
