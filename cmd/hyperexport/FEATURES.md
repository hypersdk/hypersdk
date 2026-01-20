# HyperExport New Features

This document describes the new features added to HyperExport.

## Table of Contents

1. [VM Information Display](#vm-information-display)
2. [Snapshot Management](#snapshot-management)
3. [Bandwidth Limiting](#bandwidth-limiting)
4. [Incremental Exports](#incremental-exports)
5. [Email Notifications](#email-notifications)
6. [Export Cleanup](#export-cleanup)

---

## VM Information Display

Display detailed VM information without performing an export. Useful for inspecting VM specifications before export.

### Features

- **Power state**: See if VM is powered on, off, or suspended
- **CPU details**: View number of vCPUs allocated
- **Memory info**: Display memory allocation in MB and GB
- **Storage size**: See total storage used by VM
- **Guest OS**: Identify the guest operating system
- **VM path**: Full vCenter inventory path

### Usage Examples

```bash
# Display VM information (pretty output)
hyperexport --vm-info --vm "MyVM"

# Display VM information (scripting-friendly key=value format)
hyperexport --vm-info --vm "MyVM" --quiet

# Check if VM is powered off before export
hyperexport --vm-info --vm "MyVM" | grep "poweredOff"
```

### Command-Line Flags

- `--vm-info`: Display VM information and exit
- `--vm NAME`: Specify the VM name to inspect
- `--quiet`: Use simple key=value output format for scripting

### Example Output

**Pretty Output:**
```
┌─ VM Information ───────────────────────────────────┐
│ Property      │ Value                              │
├───────────────┼────────────────────────────────────┤
│ 🖥️  VM Name   │ MyVM                               │
│ ⚡ Power State│ ● poweredOff                       │
│ 💿 Guest OS   │ Ubuntu Linux (64-bit)              │
│ 🧠 Memory     │ 8192 MB (8.0 GB)                   │
│ ⚙️  vCPUs     │ 4                                  │
│ 💾 Storage    │ 128.5 GB                           │
│ 📁 Path       │ /Datacenter/vm/production/MyVM     │
└────────────────────────────────────────────────────┘
```

**Quiet Output:**
```
name=MyVM
path=/Datacenter/vm/production/MyVM
power_state=poweredOff
guest_os=Ubuntu Linux (64-bit)
cpu=4
memory_mb=8192
storage_bytes=138047488000
```

### Benefits

- **Quick inspection**: Check VM specs without performing an export
- **Pre-export validation**: Verify VM state before export
- **Scripting support**: Parse VM info in automation scripts
- **Power state check**: Ensure VM is in the correct state for export

---

## Snapshot Management

Create and manage VM snapshots for safe, consistent exports.

### Features

- **Pre-export snapshots**: Create a snapshot before exporting to ensure consistency
- **Automatic cleanup**: Delete snapshots after export completes
- **Memory snapshots**: Optionally include VM memory state
- **Filesystem quiescing**: Ensure filesystem consistency
- **Snapshot retention**: Keep a specific number of snapshots
- **Snapshot consolidation**: Merge all snapshots into base disks

### Usage Examples

```bash
# Create snapshot before export and delete after
hyperexport --vm "MyVM" --snapshot --delete-snapshot

# Create snapshot with custom name
hyperexport --vm "MyVM" --snapshot --snapshot-name "pre-migration-backup"

# Include memory in snapshot
hyperexport --vm "MyVM" --snapshot --snapshot-memory

# Keep last 5 snapshots
hyperexport --vm "MyVM" --snapshot --keep-snapshots 5

# Consolidate all snapshots after export
hyperexport --vm "MyVM" --consolidate-snapshots
```

### Command-Line Flags

- `--snapshot`: Create snapshot before export
- `--delete-snapshot`: Delete snapshot after export (default: true)
- `--snapshot-name`: Custom snapshot name
- `--snapshot-memory`: Include memory in snapshot
- `--snapshot-quiesce`: Quiesce filesystem before snapshot (default: true)
- `--keep-snapshots N`: Keep only N most recent snapshots
- `--consolidate-snapshots`: Consolidate all snapshots

### Benefits

- **Data consistency**: Ensures VM is in a consistent state during export
- **Point-in-time recovery**: Snapshots can be used for rollback if needed
- **Safe exports**: Export from snapshot instead of running VM
- **Automated cleanup**: Prevents snapshot sprawl

---

## Bandwidth Limiting

Control download/upload bandwidth to prevent network saturation.

### Features

- **Fixed bandwidth limits**: Set maximum bytes per second
- **Burst allowance**: Allow temporary bursts above the limit
- **Adaptive limiting**: Automatically adjust based on network conditions
- **Per-transfer limiting**: Apply limits to individual file transfers
- **Statistics tracking**: Monitor actual transfer speeds

### Usage Examples

```bash
# Limit bandwidth to 10 MB/s
hyperexport --vm "MyVM" --bandwidth-limit 10

# Limit to 5 MB/s with 20 MB burst allowance
hyperexport --vm "MyVM" --bandwidth-limit 5 --bandwidth-burst 20

# Enable adaptive bandwidth (auto-adjusts based on network quality)
hyperexport --vm "MyVM" --adaptive-bandwidth

# Combine with cloud upload
hyperexport --vm "MyVM" --bandwidth-limit 10 --upload s3://my-bucket/backups
```

### Command-Line Flags

- `--bandwidth-limit N`: Limit bandwidth to N MB/s (0 = unlimited)
- `--bandwidth-burst N`: Burst allowance in MB
- `--adaptive-bandwidth`: Enable adaptive bandwidth adjustment

### How It Works

**Fixed Limiting**: Uses a token bucket algorithm to enforce a steady rate limit.

**Adaptive Limiting**: Monitors transfer success/error rates and automatically adjusts bandwidth between minimum and maximum speeds.

### Benefits

- **Network friendly**: Prevents export from saturating network
- **QoS support**: Ensures other applications get bandwidth
- **Cost savings**: Prevents excessive cloud egress costs
- **Reliable transfers**: Adaptive mode handles network congestion

---

## Incremental Exports

Export only changed disks to save time and storage.

### Features

- **Change detection**: Identifies which disks have changed since last export
- **State tracking**: Maintains export state for each VM
- **Automatic full exports**: Falls back to full export when needed
- **Savings calculation**: Shows how much data can be skipped
- **Analysis mode**: Preview incremental savings without exporting

### Usage Examples

```bash
# Enable incremental export
hyperexport --vm "MyVM" --incremental

# Show incremental analysis without exporting
hyperexport --vm "MyVM" --incremental-info

# Force full export even if incremental is available
hyperexport --vm "MyVM" --force-full

# Incremental export with cloud upload
hyperexport --vm "MyVM" --incremental --upload s3://my-bucket/backups
```

### Command-Line Flags

- `--incremental`: Enable incremental export
- `--force-full`: Force full export
- `--incremental-info`: Show incremental analysis only

### How It Works

1. **First Export**: Full export of all disks, saves state
2. **Subsequent Exports**: Compares current disks with saved state
3. **Change Detection**: Checks disk sizes and checksums
4. **Selective Export**: Only exports changed/new disks
5. **State Update**: Updates state after successful export

### Triggers for Full Export

- No previous export state found
- Disk topology changed (disks added/removed)
- More than 30 days since last export
- Force full export flag used

### Benefits

- **Faster exports**: Skip unchanged disks
- **Storage savings**: Only store changed data
- **Network efficiency**: Transfer less data
- **Cost reduction**: Lower cloud storage/transfer costs

---

## Email Notifications

Receive email notifications for export events.

### Features

- **Event notifications**: Start, success, and failure events
- **HTML emails**: Beautiful, formatted email templates
- **SMTP support**: Works with any SMTP server
- **Multiple recipients**: Send to multiple email addresses
- **Detailed information**: Includes export statistics and errors

### Usage Examples

```bash
# Enable email notifications
hyperexport --vm "MyVM" \
  --email-notify \
  --email-smtp-host smtp.gmail.com \
  --email-smtp-port 587 \
  --email-from export@example.com \
  --email-to admin@example.com \
  --email-username export@example.com \
  --email-password "your-password"

# Send email only on failure
hyperexport --vm "MyVM" \
  --email-notify \
  --email-on-start=false \
  --email-on-complete=false \
  --email-on-failure=true \
  --email-smtp-host smtp.gmail.com \
  --email-from export@example.com \
  --email-to admin@example.com

# Multiple recipients
hyperexport --vm "MyVM" \
  --email-notify \
  --email-to "admin@example.com,backup@example.com,ops@example.com"
```

### Command-Line Flags

- `--email-notify`: Enable email notifications
- `--email-smtp-host`: SMTP server hostname
- `--email-smtp-port`: SMTP server port (default: 587)
- `--email-from`: From email address
- `--email-to`: To email addresses (comma-separated)
- `--email-username`: SMTP authentication username
- `--email-password`: SMTP authentication password
- `--email-on-start`: Send email when export starts
- `--email-on-complete`: Send email when export completes (default: true)
- `--email-on-failure`: Send email when export fails (default: true)

### Email Templates

**Start Notification**:
- VM name and provider
- Export format
- Start timestamp

**Success Notification**:
- VM name and provider
- Export duration and size
- Number of files
- Verification status
- Cloud upload destination (if applicable)

**Failure Notification**:
- VM name and provider
- Error message
- Start and failure timestamps

### Supported SMTP Servers

- Gmail (smtp.gmail.com:587)
- Office 365 (smtp.office365.com:587)
- SendGrid (smtp.sendgrid.net:587)
- Mailgun (smtp.mailgun.org:587)
- Any standard SMTP server

### Benefits

- **Automated monitoring**: No need to check manually
- **Quick alerts**: Know immediately when exports fail
- **Audit trail**: Email record of all exports
- **Team visibility**: Notify multiple stakeholders

---

## Export Cleanup

Automatically clean up old exports to manage disk space.

### Features

- **Age-based cleanup**: Delete exports older than X days
- **Count-based cleanup**: Keep only N most recent exports
- **Size-based cleanup**: Delete oldest when total size exceeds limit
- **Pattern preservation**: Protect specific exports from deletion
- **Dry-run mode**: Preview what would be deleted
- **Scheduled cleanup**: Run cleanup periodically
- **Low-space triggers**: Cleanup when disk space is low

### Usage Examples

```bash
# Delete exports older than 30 days
hyperexport --cleanup --cleanup-max-age 720h  # 30 days

# Keep only last 10 exports
hyperexport --cleanup --cleanup-max-count 10

# Delete oldest when total size exceeds 100 GB
hyperexport --cleanup --cleanup-max-size 107374182400

# Preview cleanup without deleting
hyperexport --cleanup --cleanup-dry-run --cleanup-max-age 720h

# Run cleanup every 24 hours
hyperexport --cleanup-schedule 24h

# Combine age and count limits
hyperexport --cleanup --cleanup-max-age 720h --cleanup-max-count 20
```

### Command-Line Flags

- `--cleanup`: Enable cleanup
- `--cleanup-max-age`: Delete exports older than this duration
- `--cleanup-max-count`: Keep only N most recent exports
- `--cleanup-max-size`: Delete oldest when total exceeds size (bytes)
- `--cleanup-dry-run`: Preview cleanup without deleting
- `--cleanup-schedule`: Run cleanup every N hours

### Cleanup Logic

1. **Age-based**: Deletes exports older than max age
2. **Count-based**: Deletes oldest exports beyond max count
3. **Size-based**: Deletes oldest exports when total size exceeds limit
4. **Preserved exports**: Never deleted (matched by pattern)

### Benefits

- **Disk space management**: Prevent disk from filling up
- **Automated maintenance**: No manual cleanup needed
- **Cost control**: Manage storage costs
- **Compliance**: Enforce retention policies

---

## Combining Features

All features can be combined for powerful export workflows:

```bash
# Complete automated backup
hyperexport --vm "ProductionDB" \
  --snapshot \
  --delete-snapshot \
  --incremental \
  --bandwidth-limit 20 \
  --email-notify \
  --email-smtp-host smtp.gmail.com \
  --email-from backups@company.com \
  --email-to ops@company.com \
  --upload s3://prod-backups/databases \
  --cleanup \
  --cleanup-max-age 720h \
  --cleanup-max-count 30

# Safe, throttled migration
hyperexport --vm "LegacyApp" \
  --snapshot \
  --snapshot-name "pre-migration-$(date +%Y%m%d)" \
  --bandwidth-limit 10 \
  --adaptive-bandwidth \
  --email-notify \
  --email-to migration-team@company.com \
  --verify

# Automated daily backups
hyperexport --vm "FileServer" \
  --daemon-schedule "daily-backup:0 2 * * *" \
  --snapshot \
  --incremental \
  --upload s3://daily-backups \
  --cleanup-schedule 24h \
  --cleanup-max-count 7
```

---

## Best Practices

1. **Use snapshots** for production VMs to ensure consistency
2. **Enable bandwidth limiting** during business hours
3. **Use incremental exports** for large VMs to save time/storage
4. **Set up email notifications** for critical exports
5. **Enable cleanup** to prevent disk space issues
6. **Test with dry-run** before running automated cleanup
7. **Use profiles** to save complex configurations
8. **Monitor with metrics API** for production workloads

---

## Troubleshooting

### Snapshots fail to create

- Check VM tools are installed and running
- Ensure sufficient datastore space
- Verify permissions for snapshot operations

### Bandwidth limiting too aggressive

- Increase burst allowance
- Try adaptive bandwidth mode
- Check for network bottlenecks

### Incremental exports not working

- Ensure previous export completed successfully
- Check state directory is writable
- Verify VM path hasn't changed

### Email notifications not sending

- Verify SMTP credentials
- Check firewall allows outbound SMTP
- Test with a known-working SMTP server

### Cleanup deleting wrong exports

- Use dry-run mode first
- Add preserve patterns for important exports
- Adjust age/count/size limits

---

## Configuration Files

Save complex configurations as profiles:

```bash
# Create a profile
hyperexport --save-profile "production-backup" \
  --snapshot \
  --incremental \
  --bandwidth-limit 20 \
  --email-notify \
  --cleanup

# Use a profile
hyperexport --profile "production-backup" --vm "MyVM"
```

---

## Environment Variables

Some settings can be configured via environment variables:

```bash
# Email configuration
export HYPEREXPORT_SMTP_HOST=smtp.gmail.com
export HYPEREXPORT_SMTP_PORT=587
export HYPEREXPORT_EMAIL_FROM=backups@company.com
export HYPEREXPORT_EMAIL_TO=ops@company.com

# Bandwidth limiting
export HYPEREXPORT_BANDWIDTH_LIMIT=20

# Cleanup settings
export HYPEREXPORT_CLEANUP_MAX_AGE=720h
export HYPEREXPORT_CLEANUP_MAX_COUNT=30
```

---

## API Access

All features are also available through the daemon API:

```bash
# Submit job with all features
curl -X POST http://localhost:8080/api/v1/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "/Datacenter/vm/MyVM",
    "output_path": "/exports/MyVM",
    "snapshot": true,
    "incremental": true,
    "bandwidth_limit_mbps": 20,
    "email_notify": true
  }'
```

---

## Performance Impact

| Feature | CPU Impact | Memory Impact | Network Impact |
|---------|-----------|---------------|----------------|
| Snapshots | Low | Low | None |
| Bandwidth Limiting | Very Low | Very Low | Controlled |
| Incremental | Low | Low | Reduced |
| Email | Very Low | Very Low | Minimal |
| Cleanup | Low | Low | None |

All features are designed to have minimal performance impact on the export process.
