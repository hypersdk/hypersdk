# Dashboard Tab Navigation Testing Guide

## Current Tab Structure

The HyperSDK dashboard has **3 tabs**:

1. **📊 Dashboard** - Real-time metrics, charts, and system status
2. **💼 Jobs** - All jobs table with complete job history
3. **📦 Manifest Converter** - One-shot VM export and KVM conversion tool

**Note**: There are NO "Manage" or "Providers" tabs in the current implementation. These may be part of future enhancements (React dashboard migration in Phase 5 of the plan).

## How to Test

### Step 1: Start the Daemon

```bash
cd /home/ssahani/go/github/hypersdk
./bin/hypervisord
```

The daemon should start on port 8080 by default.

### Step 2: Open Dashboard in Browser

Open your web browser and navigate to:
```
http://localhost:8080
```

### Step 3: Test Tab Switching

1. **Click on the "Jobs" tab** - You should see:
   - The "Jobs" button becomes highlighted (blue background)
   - The jobs table appears
   - The dashboard metrics/charts disappear

2. **Click on the "Manifest Converter" tab** - You should see:
   - The "Manifest Converter" button becomes highlighted
   - The conversion form appears
   - Previous content disappears

3. **Click back on the "Dashboard" tab** - You should see:
   - Return to the metrics and charts view

### Step 4: Check Browser Console for Debugging

Press `F12` in your browser to open Developer Tools, then check the **Console** tab for messages:

- You should see: `Tab switcher initialized. Found 3 tab buttons and 3 tab contents`
- When clicking tabs, you should see: `Switching to tab: <tab-name>`
- If there are errors, they will be shown in red

### Common Issues

#### Issue: Tabs not switching
**Possible causes:**
- Browser cache serving old JavaScript
- JavaScript errors preventing event listeners
- Missing console errors

**Solution:**
1. Hard refresh the page: `Ctrl+Shift+R` (or `Cmd+Shift+R` on Mac)
2. Clear browser cache
3. Check browser console for errors (F12)

#### Issue: Seeing old content or different tabs
**Cause:** Browser cache

**Solution:**
```bash
# Rebuild daemon to embed latest files
go build -o bin/hypervisord ./cmd/hypervisord

# Start daemon
./bin/hypervisord

# In browser: Hard refresh (Ctrl+Shift+R)
```

#### Issue: Console shows "Tab content element not found"
**Cause:** Mismatch between button `data-tab` attribute and content element `id`

**Check:**
- Button has `data-tab="dashboard"` → Content should have `id="dashboard-tab"`
- Button has `data-tab="jobs"` → Content should have `id="jobs-tab"`
- Button has `data-tab="manifest"` → Content should have `id="manifest-tab"`

## Verify Installation

Run these commands to verify files exist:

```bash
# Check JavaScript files
ls -la daemon/dashboard/static/js/
# Should show: dashboard.js and manifest.js

# Check CSS
ls -la daemon/dashboard/static/css/
# Should show: dashboard.css

# Check template
ls -la daemon/dashboard/templates/
# Should show: index.html
```

## Testing the Manifest Converter

Once the "Manifest Converter" tab is working:

1. Fill in the form:
   - **VM Path**: `/datacenter/vm/production/test-vm`
   - **Output Directory**: `/tmp/exports/test-vm`
   - **Target Format**: QCOW2 (recommended)

2. Click "🚀 Start Export & Conversion"

3. The status section should appear below showing:
   - Job ID
   - Current status
   - Progress bar
   - Conversion logs

## Expected Browser Console Output

When everything is working correctly:

```
Tab switcher initialized. Found 3 tab buttons and 3 tab contents
WebSocket connected
[Click Jobs tab]
Switching to tab: jobs (element ID: jobs-tab)
Tab switched successfully to: jobs
[Click Manifest tab]
Switching to tab: manifest (element ID: manifest-tab)
Tab switched successfully to: manifest
```

## Debugging Steps

If tabs still don't work after trying above solutions:

1. **Check daemon logs** for any template parsing errors
2. **Verify embed.go directive** is correct in `daemon/dashboard/dashboard.go`:
   ```go
   //go:embed templates/* static/*
   var embeddedFS embed.FS
   ```
3. **Rebuild everything**:
   ```bash
   go clean
   go build -o bin/hypervisord ./cmd/hypervisord
   ```
4. **Test with curl** to verify files are served:
   ```bash
   curl http://localhost:8080/static/js/manifest.js
   curl http://localhost:8080/static/js/dashboard.js
   curl http://localhost:8080/static/css/dashboard.css
   ```

## Support

If issues persist:
- Check browser console for JavaScript errors
- Verify all files in `daemon/dashboard/` exist and are not corrupted
- Test with different browser (Chrome, Firefox, Edge)
- Clear all browser cache and cookies for localhost
