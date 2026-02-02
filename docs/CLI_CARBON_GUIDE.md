# Carbon-Aware CLI Guide

**Complete guide to using carbon-aware features with hyperctl**

---

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Commands](#commands)
  - [carbon -op status](#carbon-status)
  - [carbon -op zones](#carbon-zones)
  - [carbon -op estimate](#carbon-estimate)
  - [carbon -op report](#carbon-report)
- [Examples](#examples)
- [Integration](#integration)
- [Troubleshooting](#troubleshooting)

---

## Overview

The `hyperctl carbon` command provides CLI access to carbon-aware scheduling features, enabling you to:

- **Check grid status** - Real-time carbon intensity for any datacenter zone
- **List zones** - 12 global zones across US, EU, and APAC
- **Estimate savings** - Calculate potential carbon reduction from delaying backups
- **Generate reports** - Carbon footprint analysis for completed jobs

**Environmental Impact**: 30-50% carbon reduction through intelligent backup scheduling

---

## Installation

```bash
# Build hyperctl
cd cmd/hyperctl
go build -o hyperctl

# Install to PATH
sudo mv hyperctl /usr/local/bin/

# Verify installation
hyperctl carbon -op zones
```

**Prerequisites**:
- HyperSDK daemon running (default: `http://localhost:8080`)
- ElectricityMap API key (optional, uses mock provider for testing)

---

## Commands

### carbon -op status

**Check current grid carbon status for a zone**

```bash
hyperctl carbon -op status -zone US-CAL-CISO -threshold 200
```

**Flags**:
- `-zone` - Carbon zone ID (default: `US-CAL-CISO`)
- `-threshold` - Carbon intensity threshold in gCO2/kWh (default: `200.0`)
- `-json` - JSON output for scripting

**Output**:

```
⚡ Grid Carbon Status
┌───────────────────────────┬──────────────────────┐
│ Metric                    │ Value                │
├───────────────────────────┼──────────────────────┤
│ Zone                      │ US-CAL-CISO          │
│ Carbon Intensity          │ 145.2 gCO2/kWh       │
│ Quality                   │ good                 │
│ Renewable Energy          │ 68.5%                │
│ Optimal for Backup        │ ✓ Yes                │
└───────────────────────────┴──────────────────────┘

💡 GOOD time to run backups (145 gCO2/kWh, 69% renewable)

📊 4-Hour Forecast
┌────────┬─────────────────────────┬────────────┐
│ Time   │ Intensity (gCO2/kWh)    │ Quality    │
├────────┼─────────────────────────┼────────────┤
│ 14:00  │ 132.1                   │ good       │
│ 15:00  │ 118.6                   │ good       │
│ 16:00  │ 105.2                   │ excellent  │
│ 17:00  │ 142.8                   │ good       │
└────────┴─────────────────────────┴────────────┘
```

**Quality Levels**:
- `excellent` (green) - < 100 gCO2/kWh (renewables)
- `good` (light green) - 100-200 gCO2/kWh
- `moderate` (yellow) - 200-400 gCO2/kWh
- `poor` (light red) - 400-600 gCO2/kWh
- `very poor` (red) - > 600 gCO2/kWh (coal)

**Use Case**: Check grid status before submitting a backup job to decide whether to run now or delay.

---

### carbon -op zones

**List all available carbon zones**

```bash
hyperctl carbon -op zones
```

**Flags**:
- `-json` - JSON output for scripting

**Output**:

```
🌍 Available Carbon Zones

📍 North America
┌─────────────────┬──────────────────────────┬───────────────────┬──────────────────────────────────────────┐
│ Zone ID         │ Name                     │ Typical Intensity │ Description                              │
├─────────────────┼──────────────────────────┼───────────────────┼──────────────────────────────────────────┤
│ US-CAL-CISO     │ US California (CISO)     │ 200 gCO2/kWh      │ California Independent System Operator   │
│ US-PNW          │ US Pacific Northwest     │ 150 gCO2/kWh      │ Pacific Northwest region (WA, OR)        │
│ US-MISO         │ US Midwest (MISO)        │ 450 gCO2/kWh      │ Midcontinent Independent System Operator │
└─────────────────┴──────────────────────────┴───────────────────┴──────────────────────────────────────────┘

📍 Europe
┌─────────────────┬──────────────────────────┬───────────────────┬──────────────────────────────────────────┐
│ Zone ID         │ Name                     │ Typical Intensity │ Description                              │
├─────────────────┼──────────────────────────┼───────────────────┼──────────────────────────────────────────┤
│ DE              │ Germany                  │ 350 gCO2/kWh      │ German electricity grid                  │
│ GB              │ United Kingdom           │ 250 gCO2/kWh      │ UK National Grid                         │
│ SE              │ Sweden                   │ 50 gCO2/kWh       │ Swedish electricity grid (very clean)    │
└─────────────────┴──────────────────────────┴───────────────────┴──────────────────────────────────────────┘

📍 Asia Pacific
┌─────────────────┬──────────────────────────┬───────────────────┬──────────────────────────────────────────┐
│ Zone ID         │ Name                     │ Typical Intensity │ Description                              │
├─────────────────┼──────────────────────────┼───────────────────┼──────────────────────────────────────────┤
│ SG              │ Singapore                │ 400 gCO2/kWh      │ Singapore electricity grid               │
│ JP-TK           │ Tokyo, Japan             │ 500 gCO2/kWh      │ Tokyo Electric Power Company area        │
│ AU-NSW          │ Sydney, Australia        │ 700 gCO2/kWh      │ New South Wales grid                     │
│ IN-NO           │ North India              │ 650 gCO2/kWh      │ Northern India grid                      │
│ CN-BJ           │ Beijing, China           │ 800 gCO2/kWh      │ Beijing electricity grid                 │
│ CN-SH           │ Shanghai, China          │ 750 gCO2/kWh      │ Shanghai electricity grid                │
└─────────────────┴──────────────────────────┴───────────────────┴──────────────────────────────────────────┘

💡 Use zone ID with: hyperctl carbon status -zone <ZONE_ID>
```

**Use Case**: Discover available zones for your datacenter locations. Use zone IDs in other carbon commands.

---

### carbon -op estimate

**Estimate carbon savings from delaying a backup**

```bash
hyperctl carbon -op estimate -zone US-CAL-CISO -data 500 -hours 2
```

**Flags**:
- `-zone` - Carbon zone ID (default: `US-CAL-CISO`)
- `-data` - Data size in GB (required)
- `-hours` - Duration in hours (default: `2.0`)
- `-json` - JSON output for scripting

**Output**:

```
🔮 Carbon Savings Estimate

┌──────────────────┬────────────────────────┬─────────────────────┐
│ Scenario         │ Intensity (gCO2/kWh)   │ Emissions (kg CO2)  │
├──────────────────┼────────────────────────┼─────────────────────┤
│ Run Now          │ 350.0                  │ 0.079               │
│ Run at Best Time │ 120.0                  │ 0.027               │
└──────────────────┴────────────────────────┴─────────────────────┘

💰 Potential Savings
┌──────────────────┬──────────────┐
│ Metric           │ Value        │
├──────────────────┼──────────────┤
│ CO2 Savings      │ 0.052 kg CO2 │
│ Reduction        │ 65.8%        │
│ Best Time        │ 16:00:00     │
│ Delay Required   │ 2h 45m       │
└──────────────────┴──────────────┘

💡 Delay 165 minutes to save 0.05 kg CO2 (66% reduction)

📊 4-Hour Forecast
┌────────┬─────────────────────────┬────────────┐
│ Time   │ Intensity (gCO2/kWh)    │ Quality    │
├────────┼─────────────────────────┼────────────┤
│ 14:00  │ 320.0                   │ moderate   │
│ 15:00  │ 250.0                   │ moderate   │
│ 16:00  │ 120.0                   │ good       │
│ 17:00  │ 180.0                   │ good       │
└────────┴─────────────────────────┴────────────┘
```

**Use Case**: Before scheduling a backup, check potential carbon savings. Use this to decide whether to delay the backup job.

**Calculation**:
- Energy (kWh) = (Data GB × 0.00015) + (Duration hours × 0.075)
- Emissions (kg CO2) = Energy (kWh) × Carbon Intensity (gCO2/kWh) / 1000

---

### carbon -op report

**Generate carbon footprint report for a completed job**

```bash
hyperctl carbon -op report \
  -job job-123 \
  -data 500 \
  -start "2026-02-04T10:00:00Z" \
  -end "2026-02-04T12:00:00Z" \
  -zone US-CAL-CISO
```

**Flags**:
- `-job` - Job ID (required)
- `-data` - Data size in GB (required)
- `-start` - Start time in RFC3339 format (required)
- `-end` - End time in RFC3339 format (required)
- `-zone` - Carbon zone ID (default: `US-CAL-CISO`)
- `-json` - JSON output for scripting

**Output**:

```
🌿 Carbon Footprint Report

┌──────────────┬──────────┐
│ Metric       │ Value    │
├──────────────┼──────────┤
│ Job ID       │ job-123  │
│ Duration     │ 2.0 hours│
│ Data Size    │ 500.0 GB │
└──────────────┴──────────┘

⚡ Energy & Emissions
┌──────────────────────────┬──────────────────┐
│ Metric                   │ Value            │
├──────────────────────────┼──────────────────┤
│ Energy Used              │ 0.225 kWh        │
│ Carbon Intensity         │ 145.2 gCO2/kWh   │
│ Carbon Emissions         │ 0.033 kg CO2     │
│ Renewable Energy         │ 68.5%            │
│ Savings vs Worst Case    │ 0.192 kg CO2     │
└──────────────────────────┴──────────────────┘

🚗 Equivalent: 0.1 km of driving

✅ Carbon report generated successfully!
```

**Use Case**: After a backup job completes, generate a carbon report for sustainability reporting and ESG compliance.

**Time Format Examples**:
- RFC3339: `2026-02-04T10:00:00Z`
- ISO 8601: `2026-02-04T10:00:00`
- Simple: `2026-02-04 10:00:00`

---

## Examples

### Example 1: Morning Decision Workflow

**Check if it's a good time to run backups:**

```bash
# 1. Check grid status
hyperctl carbon -op status -zone US-CAL-CISO

# Output shows: "POOR time to run backups (450 gCO2/kWh, 25% renewable)"
# Next optimal time: 16:00 (in 4h 30m)

# 2. Estimate savings from waiting
hyperctl carbon -op estimate -zone US-CAL-CISO -data 1000 -hours 3

# Output shows: "Delay 270 minutes to save 0.35 kg CO2 (72% reduction)"

# 3. Decision: Delay the backup until 16:00
# Schedule backup job with carbon_aware: true
```

### Example 2: Post-Backup Reporting

**Generate carbon report for ESG compliance:**

```bash
# 1. Backup completed - get report
hyperctl carbon -op report \
  -job job-789 \
  -data 750 \
  -start "2026-02-04T12:00:00Z" \
  -end "2026-02-04T14:30:00Z" \
  -zone US-CAL-CISO \
  -json > carbon-report-job-789.json

# 2. Extract emissions data
cat carbon-report-job-789.json | jq '.carbon_emissions_kg_co2'
# Output: 0.045

# 3. Add to quarterly ESG report
echo "Job job-789: 0.045 kg CO2" >> q1-2026-emissions.txt
```

### Example 3: Multi-Region Comparison

**Compare carbon intensity across datacenters:**

```bash
# Check all your datacenter zones
echo "US West Coast:"
hyperctl carbon -op status -zone US-CAL-CISO -json | jq '.current_intensity'

echo "EU Germany:"
hyperctl carbon -op status -zone DE -json | jq '.current_intensity'

echo "APAC Singapore:"
hyperctl carbon -op status -zone SG -json | jq '.current_intensity'

# Choose the greenest region for backup
```

### Example 4: Automated Scripting

**Bash script to only run backups when grid is clean:**

```bash
#!/bin/bash

ZONE="US-CAL-CISO"
THRESHOLD=200

# Check carbon status
STATUS=$(hyperctl carbon -op status -zone $ZONE -threshold $THRESHOLD -json)
OPTIMAL=$(echo $STATUS | jq -r '.optimal_for_backup')

if [ "$OPTIMAL" = "true" ]; then
  echo "Grid is clean, starting backup..."
  hyperctl submit -vm /datacenter/vm/prod-db -output /backups
else
  NEXT_TIME=$(echo $STATUS | jq -r '.next_optimal_time')
  echo "Grid is dirty, next optimal time: $NEXT_TIME"
  echo "Scheduling backup for later..."
fi
```

---

## Integration

### With Job Submission

**Enable carbon-aware scheduling for jobs:**

```bash
# Create job with carbon awareness
cat > backup-job.json <<EOF
{
  "name": "nightly-backup",
  "vm_path": "/datacenter/vm/db-prod",
  "output_path": "/backups",
  "metadata": {
    "carbon_aware": true,
    "carbon_max_intensity": 150.0,
    "carbon_max_delay": 7200000000000,
    "carbon_zone": "US-CAL-CISO"
  }
}
EOF

# Submit job
hyperctl submit -file backup-job.json

# Job will be delayed if grid is dirty
# Scheduler checks carbon intensity and delays up to 2 hours
```

### With Cron Jobs

**Daily backup with carbon checking:**

```bash
# Add to crontab
0 2 * * * /usr/local/bin/backup-with-carbon-check.sh

# backup-with-carbon-check.sh:
#!/bin/bash
STATUS=$(hyperctl carbon -op status -zone US-CAL-CISO -json)
if [ "$(echo $STATUS | jq -r '.optimal_for_backup')" = "true" ]; then
  hyperctl submit -vm /dc/vm/prod -output /backups
  echo "Backup started at $(date): Clean grid"
else
  echo "Backup skipped at $(date): Dirty grid"
  # Try again in 1 hour
  echo "0 3 * * * /usr/local/bin/backup-with-carbon-check.sh" | at now + 1 hour
fi
```

### With CI/CD

**GitLab CI carbon-aware pipeline:**

```yaml
backup-prod:
  script:
    # Check carbon status
    - export CARBON_STATUS=$(hyperctl carbon -op status -zone US-CAL-CISO -json)
    - export OPTIMAL=$(echo $CARBON_STATUS | jq -r '.optimal_for_backup')

    # Only run if grid is clean
    - |
      if [ "$OPTIMAL" = "true" ]; then
        echo "Grid is clean, running backup..."
        hyperctl submit -vm /dc/vm/prod -output /backups
      else
        echo "Grid is dirty, failing pipeline"
        exit 1
      fi

  # Retry if grid was dirty
  retry:
    max: 3
    when: script_failure
```

---

## Troubleshooting

### Connection Refused

**Error**: `dial tcp [::1]:8080: connect: connection refused`

**Solution**:
```bash
# Check if daemon is running
hyperctl daemon -op status

# Start daemon if not running
sudo systemctl start hyper2kvm

# Or specify different URL
hyperctl carbon -op zones -daemon http://192.168.1.100:8080
```

### Invalid Time Format

**Error**: `invalid time format: 2026-02-04 10:00`

**Solution**: Use RFC3339 format with timezone:
```bash
# Correct formats:
hyperctl carbon -op report -start "2026-02-04T10:00:00Z"
hyperctl carbon -op report -start "2026-02-04T10:00:00-08:00"

# Not supported:
hyperctl carbon -op report -start "2026-02-04 10:00"  # ❌
```

### Zone Not Found

**Error**: `zone is required`

**Solution**: Specify a valid zone:
```bash
# List available zones first
hyperctl carbon -op zones

# Use a valid zone ID
hyperctl carbon -op status -zone US-CAL-CISO
```

### Missing Required Flags

**Error**: `For report: -job, -data, -start, and -end are required`

**Solution**: Provide all required flags:
```bash
hyperctl carbon -op report \
  -job job-123 \
  -data 500 \
  -start "2026-02-04T10:00:00Z" \
  -end "2026-02-04T12:00:00Z" \
  -zone US-CAL-CISO
```

---

## Performance Tips

1. **Cache Zone List**: Zone data rarely changes
   ```bash
   hyperctl carbon -op zones -json > zones-cache.json
   ```

2. **Parallel Checks**: Check multiple zones simultaneously
   ```bash
   parallel "hyperctl carbon -op status -zone {} -json" ::: US-CAL-CISO DE SE
   ```

3. **JSON Output**: Faster for scripting (no formatting overhead)
   ```bash
   hyperctl carbon -op status -zone US-CAL-CISO -json | jq '.current_intensity'
   ```

4. **Batch Estimates**: Estimate multiple scenarios at once
   ```bash
   for size in 100 500 1000; do
     hyperctl carbon -op estimate -zone US-CAL-CISO -data $size -hours 2 -json
   done
   ```

---

## Next Steps

- **[Carbon API Documentation](CARBON_AWARE_COMPLETE.md)** - Full API reference
- **[Integration Guide](INTEGRATION.md)** - Integrate with other systems
- **[ESG Reporting](ESG_COMPLIANCE.md)** - Generate compliance reports
- **[Best Practices](BEST_PRACTICES.md)** - Optimization strategies

---

**🌿 Start reducing your carbon footprint today!**

```bash
hyperctl carbon -op status -zone US-CAL-CISO
```
