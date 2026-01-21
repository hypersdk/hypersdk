# Interactive Mode User Guide

The interactive mode provides a powerful Terminal User Interface (TUI) for discovering, filtering, and migrating VMs from vSphere to KVM.

## Table of Contents

- [Getting Started](#getting-started)
- [Navigation Controls](#navigation-controls)
- [Search and Filter](#search-and-filter)
- [Sorting](#sorting)
- [Selection](#selection)
- [Detail View](#detail-view)
- [Split-Screen Mode](#split-screen-mode)
- [Export Queue Management](#export-queue-management)
- [Export History View](#export-history-view)
- [Live Logs Viewer](#live-logs-viewer)
- [Dry-Run Mode](#dry-run-mode)
- [Migration Workflow](#migration-workflow)
- [Keyboard Reference](#keyboard-reference)

## Getting Started

Launch interactive mode:

```bash
# Using daemon API
hyperctl migrate

# Using direct vCenter connection
export GOVC_URL='https://vcenter.example.com/sdk'
export GOVC_USERNAME='administrator@vsphere.local'
export GOVC_PASSWORD='your-password'
export GOVC_INSECURE=1
hyperctl migrate
```

## Navigation Controls

### Basic Movement

- **↑ / k** - Move cursor up
- **↓ / j** - Move cursor down
- **Space** - Select/deselect current VM
- **Enter** - Proceed to next step
- **Esc / b** - Go back to previous step
- **q** - Quit

The interface automatically displays a scrolling window when you have more VMs than can fit on screen. The cursor stays centered as you navigate through long lists.

## Search and Filter

### Search (`/` key)

Press `/` to enter search mode. Search is **case-insensitive** and matches against:

- VM name
- VM path
- Guest OS type

**Example searches:**
- `ubuntu` - Finds all VMs with "ubuntu" in name, path, or OS
- `web-server` - Finds VMs with "web-server" in their name
- `windows` - Finds all Windows VMs

**Search controls:**
- Type to search
- **Backspace** - Delete last character
- **Enter** - Apply search and return to VM list
- **Esc** - Cancel search

The search provides **live preview** showing:
- Number of matching VMs
- First 10 matching VMs
- Count of additional matches

### Power State Filter (`f` key)

Press `f` repeatedly to cycle through power state filters:

1. **All VMs** (no filter)
2. **Powered ON only** - Shows only running VMs
3. **Powered OFF only** - Shows only stopped VMs

Current filter is shown in the status bar.

### Clear Filters (`c` key)

Press `c` to clear all active filters and searches at once, returning to the full VM list.

## Sorting

Press `s` to cycle through sort modes:

1. **Name** (A-Z) - Alphabetical by VM name
2. **CPU** - By CPU count (highest first)
3. **Memory** - By memory allocation (highest first)
4. **Storage** - By storage size (largest first)
5. **Power** - Powered ON VMs first, then powered OFF

Current sort mode is displayed in the help text.

**Performance note:** Sorting uses efficient O(n log n) algorithms and is instant even with 1000+ VMs.

## Selection

### Select Individual VMs

- Position cursor on VM
- Press **Space** to toggle selection
- Selected VMs are marked with **[✓]**
- Unselected VMs are marked with **[ ]**

### Bulk Selection

- **a** - Select all visible VMs (respects filters)
- **n** - Deselect all visible VMs

**Important:** Bulk selection only affects currently visible VMs. If you've applied filters, only the filtered VMs are selected/deselected.

### Selection Status

The status bar shows:
```
📊 Total VMs: 50 | Visible: 20 | ✅ Selected: 5
```

- **Total VMs** - All VMs in vCenter
- **Visible** - VMs matching current filters
- **Selected** - VMs selected for migration (across all filters)

## Detail View

Press `d` or `i` to view detailed information about the VM under the cursor.

**Displays:**
- Full VM name (no truncation)
- vCenter path
- Power state
- Guest OS
- CPU count
- Memory (in GB and MB)
- Storage (formatted and bytes)
- Selection status

**Controls in detail view:**
- **Space / Enter** - Toggle selection
- **Esc / b** - Return to VM list

## Split-Screen Mode

Press `v` to toggle split-screen mode, which displays the VM list and VM details side-by-side for efficient navigation and inspection.

**Layout:**
- **Wide terminals (≥80 columns):** Horizontal split with list on left (45%) and details on right (50%)
- **Narrow terminals (<80 columns):** Vertical split with list on top and details on bottom

**Visual Indicators:**
- Active pane highlighted with colored border (teal)
- Inactive pane dimmed with gray border
- Pane titles show "(ACTIVE)" or "(inactive)" status
- Footer displays keyboard shortcuts

**Controls in split-screen mode:**
- **Tab** - Switch between list and details panes
- **↑ / ↓** - Navigate VMs (when list pane is active)
- **Space** - Select/deselect VM
- **Enter** - Proceed to export
- **v** - Exit split-screen mode
- **Esc / b** - Go back
- **q** - Quit

**Details Pane Shows:**
- VM name and path
- Power state (color-coded: green for on, yellow for off)
- Guest OS
- CPU cores
- Memory (MB)
- Storage (GB)

**Benefits:**
- See VM details without leaving the list
- Compare VMs quickly while navigating
- Reduce context switching
- Inspect VM specs while maintaining your position in the list
- Perfect for reviewing large VM inventories

**Example Workflow:**
1. Press `v` to enter split-screen mode
2. Navigate through VMs with ↑/↓ keys
3. Details automatically update in right pane
4. Press **Tab** to focus details pane for reading
5. Press **Tab** again to return to list navigation
6. Press **Space** to select interesting VMs
7. Press **v** to exit split-screen when done

## Export Queue Management

Press `Q` (capital Q) to open the export queue manager, which allows you to reorder VMs and set export priorities.

**Opening the Queue:**
- Select one or more VMs in the main selection view
- Press **Q** to open the queue manager
- Queue is built from currently selected VMs only

**Queue Controls:**
- **↑ / k** - Navigate up in queue
- **↓ / j** - Navigate down in queue
- **K / Shift+↑** - Move current VM up in queue order
- **J / Shift+↓** - Move current VM down in queue order
- **P** - Cycle priority (Normal → High → Low → Normal)
- **Enter** - Confirm queue and proceed to export
- **Esc** - Cancel and return to VM selection

**Priority Levels:**
- **[HIGH]** (red) - Export first, highest priority
- **[NORM]** (blue) - Normal priority (default)
- **[LOW]** (gray) - Export last, lowest priority

**Visual Indicators:**
- Position numbers (#1, #2, #3...) show export order
- Color-coded priority badges
- VM name and storage size displayed
- Active cursor (❯) shows current selection
- Summary shows counts: "X High | Y Normal | Z Low"

**Use Cases:**
- **Critical VMs first:** Set production VMs to [HIGH] priority
- **Size-based ordering:** Move large VMs to end of queue ([LOW])
- **Dependency ordering:** Reorder based on VM dependencies
- **Testing:** Export test VMs first ([HIGH]) before production

**Example Workflow:**
1. Select 10 VMs for export
2. Press **Q** to open queue manager
3. Navigate to production database VM
4. Press **P** to set to [HIGH] priority
5. Press **K** multiple times to move it to position #1
6. Repeat for other critical VMs
7. Press **Enter** to confirm and start exports in order

**Benefits:**
- Full control over export sequence
- Prioritize critical VMs
- Optimize based on VM size or importance
- Visual feedback before starting
- Change your mind before exporting

## Export History View

Press `H` (capital H) to open the export history viewer, which shows all past export operations with comprehensive filtering capabilities.

**Opening the History:**
- Press **H** from the main VM selection screen
- History is loaded from `~/.hyperexport/history.json`
- Shows all past exports in reverse chronological order (most recent first)

**History Display:**
The history view shows:
- **Status indicator:** ✓ (green) for successful exports, ✗ (red) for failed
- **Timestamp:** Date and time of export (MM/DD HH:MM format)
- **VM name:** Name of the exported VM
- **Format:** Export format (OVF, OVA, etc.)
- **Size:** Total export size in GB
- **Duration:** How long the export took

**Summary Statistics:**
At the top of the history view, you'll see:
```
📊 50 Total | ✓ 45 Success | ✗ 5 Failed | 💾 2.5 TB | ⏱ 45m avg
```
- **Total:** Number of exports matching current filters
- **Success/Failed:** Count of successful vs failed exports
- **Size:** Total data exported
- **Avg duration:** Average export time

**Filtering Options:**

1. **Status Filter (F key):**
   - Press **F** to cycle through: all → success → failed → all
   - Filter by export success/failure status
   - Active filter shown at top: `Status: success` or `Status: failed`

2. **Date Range Filter (D key):**
   - Press **D** to cycle through: all → today → week → month → all
   - **today:** Exports from last 24 hours
   - **week:** Exports from last 7 days
   - **month:** Exports from last 30 days
   - Active filter shown at top: `Time: today`, `Time: week`, or `Time: month`

**History Controls:**
- **↑ / k** - Navigate up in history
- **↓ / j** - Navigate down in history
- **F** - Cycle status filter (all/success/failed)
- **D** - Cycle date filter (all/today/week/month)
- **R** - Refresh history from disk
- **Esc / b** - Return to VM selection
- **q** - Quit application

**Detailed View:**
When you select a history entry, detailed information appears at the bottom:
- Full VM name and path
- Provider (vsphere, etc.)
- Export format
- Total size
- Duration
- Number of files exported
- Timestamp
- Output directory
- Error message (if failed)

**Use Cases:**

1. **Audit trail:** Review what VMs were exported and when
2. **Troubleshooting:** Find failed exports and view error messages
3. **Capacity planning:** See total data exported and average times
4. **Verification:** Confirm a VM was successfully exported
5. **Re-export decisions:** Check if a VM needs to be re-exported

**Example Workflow:**
1. Press **H** to open history
2. Press **F** to filter for failed exports only
3. Navigate through failed exports with ↑/↓
4. Review error messages in detailed view
5. Press **Esc** to return to VM selection
6. Re-export failed VMs

**Example: Finding Recent Exports**
1. Press **H** to open history
2. Press **D** twice to filter by "today"
3. Review all exports from last 24 hours
4. Check success rate and any failures

**Example: Viewing Large Exports**
1. Press **H** to open history
2. Navigate through history - largest exports show higher GB values
3. Review duration to estimate time for similar VMs
4. Use this data to plan future export batches

**Performance Notes:**
- History is loaded once when opening the view
- Press **R** to reload if history.json was updated externally
- Filtering is instant even with hundreds of entries
- Last 15 entries displayed at a time with smooth scrolling

**Troubleshooting:**

**"No export history found":**
- No exports have been completed yet
- History file doesn't exist at `~/.hyperexport/history.json`
- Perform an export first to populate history

**History not updating:**
- Press **R** to refresh from disk
- Check that exports are completing successfully
- Verify `~/.hyperexport/history.json` exists and is writable

**Filters showing no results:**
- Press **F** and **D** to cycle back to "all" filters
- Check that you have exports matching the filter criteria
- For date filters, ensure exports exist in the time range

## Live Logs Viewer

Press `L` (capital L) to open the live logs viewer, which displays real-time export logs with filtering and scrolling capabilities.

**Opening the Logs Viewer:**
- Press **L** from the main VM selection screen or during exports
- Logs viewer shows all log messages in chronological order
- Maximum of 1000 log entries kept in memory (oldest pruned automatically)

**Log Display:**
The logs viewer shows:
- **Level indicator:** ℹ (info-teal), ⚠ (warn-yellow), ✗ (error-red), 🐛 (debug-gray)
- **Timestamp:** Time of log entry in HH:MM:SS format
- **VM name:** Associated VM name (if applicable) in orange brackets
- **Message:** Log message (truncated if too long for terminal width)

**Summary Statistics:**
At the top of the logs view, you'll see:
```
📊 50 Total | ℹ 35 Info | ⚠ 10 Warn | ✗ 3 Error | 🐛 2 Debug
```
- **Total:** Number of log entries matching current filter
- **Info/Warn/Error/Debug:** Count by log level
- Real-time updates as new logs are added

**Filtering Options:**

1. **Log Level Filter (L key):**
   - Press **L** to cycle through: all → info → warn → error → debug → all
   - Filter by log severity level
   - Active filter shown at top: `Level: INFO` or `Level: ERROR`

2. **Auto-Scroll (A key):**
   - Press **A** to toggle auto-scroll on/off
   - **ON:** Automatically jumps to newest log when added (default)
   - **OFF:** Stays at current position for manual review
   - Status shown at top: `[AUTO-SCROLL ON]` or `[AUTO-SCROLL OFF]`

**Logs Controls:**
- **↑ / k** - Navigate up in logs
- **↓ / j** - Navigate down in logs
- **g** - Jump to top (oldest log)
- **G** - Jump to bottom (newest log) and enable auto-scroll
- **L** - Cycle log level filter (all/info/warn/error/debug)
- **A** - Toggle auto-scroll on/off
- **C** - Clear all logs
- **Esc / b** - Return to previous view
- **q** - Quit application

**Log Scrolling:**
- Shows last 20 log entries at a time
- Scroll up/down to view older/newer logs
- Indicators show "X more above" or "X more below" when applicable
- Selected log highlighted with cursor (❯)

**Color Coding:**
Log levels are color-coded for quick identification:
- **INFO** (ℹ teal): Normal informational messages
- **WARN** (⚠ yellow): Warnings that don't stop execution
- **ERROR** (✗ red): Errors requiring attention
- **DEBUG** (🐛 gray): Detailed debugging information

**Use Cases:**

1. **Real-time monitoring:** Watch export progress live during migrations
2. **Error diagnosis:** Filter to ERROR level to see only failures
3. **Performance analysis:** Review timestamps to identify slow operations
4. **Debugging:** Enable DEBUG level for detailed troubleshooting
5. **VM tracking:** See which logs belong to which VM

**Example Workflow:**
1. Start an export operation
2. Press **L** to open logs viewer
3. Watch real-time logs with auto-scroll ON
4. If error occurs, logs continue scrolling
5. Press **A** to disable auto-scroll
6. Press **L** twice to filter ERROR only
7. Navigate through errors with ↑/↓
8. Press **g** to jump to first error
9. Review error details
10. Press **Esc** to return to export view

**Example: Debugging Failed Export**
1. Export fails
2. Press **L** to open logs
3. Press **L** twice to filter ERROR level only
4. Review error messages
5. Note VM name and timestamp
6. Press **G** to see most recent error
7. Press **C** to clear logs
8. Retry export and monitor new logs

**Example: Monitoring Multiple Exports**
1. Select multiple VMs for export
2. Press **L** before starting
3. Auto-scroll ON to follow progress
4. See each VM's logs tagged with [VMName]
5. Watch for any warnings or errors
6. Jump to top (g) to review start
7. Jump to bottom (G) to see latest

**Performance Notes:**
- Logs limited to 1000 entries (oldest auto-pruned)
- Filtering is instant even with full log buffer
- Last 20 entries displayed at a time for performance
- Auto-scroll updates cursor only when enabled

**Integration with Exports:**
- Logs automatically populate during export operations
- Export progress messages appear in logs
- File download progress logged
- Errors and warnings captured automatically
- VM-specific logs tagged with VM name

**Troubleshooting:**

**"No log entries found":**
- No exports have been run yet in this session
- Logs were cleared with **C** key
- All logs filtered out by current level filter
- Press **L** to cycle back to "all" level filter

**Logs not updating:**
- Auto-scroll may be disabled - press **A** to enable
- Check that export operations are running
- Some operations may not generate logs

**Can't see recent logs:**
- Press **G** to jump to bottom
- Enable auto-scroll with **A** key
- Check log level filter - may be filtering out info logs

## Dry-Run Mode

Press `r` to toggle dry-run mode.

**What is dry-run mode?**
- Preview migration without executing
- Test filters and selections
- Verify settings before committing
- See exactly what would be migrated

**Visual indicators:**
- `[DRY-RUN]` badge in help text
- Prominent warnings in confirmation screens
- Systemd service creation blocked

**Perfect for:**
- Testing complex filter combinations
- Verifying multi-VM selections
- Training and demonstrations
- Planning migration batches

## Migration Workflow

### 1. VM Selection Screen

**Status Bar:**
```
📊 Total VMs: 100 | Visible: 45 | ✅ Selected: 8
🔍 Search: ubuntu | ⚡ Power: on
```

**Actions:**
- Browse, filter, sort, search
- Select VMs for migration
- View details
- Press **Enter** when ready

### 2. Confirmation Screen

**Shows:**
- List of selected VMs with full details
- Total resources (CPUs, memory, storage)
- Export settings
- Migration pipeline (export → convert → import)

**Controls:**
- **y/Y** - Confirm and proceed
- **n/N / Esc / b** - Go back to selection
- **q** - Quit

### 3. Execution Mode Selection

Choose how to run the migration:

#### Terminal Mode (Interactive)
```
✓ Watch progress in real-time
✓ See immediate feedback
✓ Requires keeping terminal open
⚠  Terminal must stay active during migration
```

**Use when:**
- You want to monitor progress
- Migration is short (< 1 hour)
- You're on a stable connection

#### Systemd Service (Background)
```
✓ Runs in background
✓ Can close terminal and come back later
✓ Survives SSH disconnections
✓ Check status with: journalctl -u vm-migration@<job-id>
ℹ  Perfect for long migrations or remote work
```

**Use when:**
- Migration will take hours
- You're on an unstable connection
- You need to disconnect and check back later

### 4. Migration Execution

The migration proceeds in stages:

1. **Export** - VM downloaded as OVF from vSphere
2. **Convert** - OVF converted to qcow2 (if enabled)
3. **Import** - qcow2 imported to libvirt (if enabled)

## Keyboard Reference

### Navigation
| Key | Action |
|-----|--------|
| ↑ / k | Move up |
| ↓ / j | Move down |
| Space | Select/deselect VM |
| Enter | Continue/confirm |
| Esc / b | Go back |
| q | Quit |

### Search & Filter
| Key | Action |
|-----|--------|
| / | Enter search mode |
| s | Cycle sort mode |
| f | Toggle power filter |
| c | Clear all filters |

### View & Selection
| Key | Action |
|-----|--------|
| a | Select all visible |
| n | Deselect all visible |
| d / i | Show detail view |
| v | Toggle split-screen mode |
| Tab | Switch pane (in split-screen) |
| Q | Open export queue manager |
| H | Open export history |
| L | Open live logs viewer |
| r | Toggle dry-run mode |

### Queue Management (in queue view)
| Key | Action |
|-----|--------|
| K / Shift+↑ | Move VM up in queue |
| J / Shift+↓ | Move VM down in queue |
| p | Change priority |
| Enter | Confirm queue |
| Esc | Cancel and return |

### History View (in history)
| Key | Action |
|-----|--------|
| ↑ / k | Navigate up |
| ↓ / j | Navigate down |
| F | Cycle status filter |
| D | Cycle date filter |
| R | Refresh history |
| Esc | Return to selection |

### Logs View (in logs)
| Key | Action |
|-----|--------|
| ↑ / k | Navigate up |
| ↓ / j | Navigate down |
| g | Jump to top (oldest) |
| G | Jump to bottom (newest) |
| L | Cycle log level filter |
| A | Toggle auto-scroll |
| C | Clear all logs |
| Esc | Return to previous view |

## Examples

### Example 1: Find and migrate all Ubuntu VMs

1. Press `/`
2. Type `ubuntu`
3. Press **Enter**
4. Review filtered list
5. Press `a` to select all
6. Press **Enter** to continue

### Example 2: Migrate only powered-off Windows VMs

1. Press `f` twice (to filter powered OFF)
2. Press `/`
3. Type `windows`
4. Press **Enter**
5. Press `a` to select all matching
6. Press **Enter** to continue

### Example 3: Preview large VM migration without executing

1. Press `s` multiple times until sorting by "storage"
2. Select top 5 VMs (largest storage)
3. Press `r` to enable dry-run
4. Press **Enter** to review
5. Confirm to see what would happen
6. Press `r` again to disable dry-run if you want to proceed

### Example 4: Detailed inspection before migration

1. Navigate to interesting VM
2. Press `d` to view details
3. Review all specs
4. Press **Space** to select
5. Press **Esc** to return to list
6. Repeat for other VMs

## Performance Notes

- **Search/Filter:** Instant for 1000+ VMs
- **Sorting:** O(n log n) performance, optimized for large lists
- **Selection:** Map-based lookups for O(n) bulk operations
- **Rendering:** Shows 20 VMs at a time, scrolls smoothly

## Troubleshooting

### "No VMs found"

**Check:**
1. Daemon is running: `sudo systemctl status hyper2kvmd`
2. vCenter credentials are correct
3. Environment variables are set (for direct connection)
4. Your user has permission to list VMs

### Search returns no results

**Try:**
1. Clear filters with `c`
2. Check search term spelling
3. Search is case-insensitive, but spelling must match
4. Try searching by path or OS instead of name

### VMs not visible after filtering

**Solution:**
- Press `c` to clear all filters
- VMs may be filtered out by power state or search

### Can't select VMs

**Ensure:**
- You're in selection mode (not search or detail view)
- Press **Esc** if you're in another mode
- Use **Space** on the VM under the cursor

## Best Practices

1. **Start with search/filter** - Narrow down before bulk operations
2. **Use detail view** - Verify specs before migration
3. **Test with dry-run** - Always preview first for critical migrations
4. **Sort strategically** - Use CPU/memory sort to group similar VMs
5. **Select incrementally** - Filter → select → filter again → select more
6. **Use systemd service** - For any migration longer than 30 minutes

## Advanced Workflows

### Batch Migration Strategy

1. **Group by size:** Sort by storage
2. **Migrate small first:** Select bottom 10 VMs
3. **Test migration:** Run in terminal mode
4. **If successful:** Migrate larger batches via systemd service

### Staged Migration

1. **Day 1:** Filter powered OFF, migrate all
2. **Day 2:** Filter by development environment, migrate
3. **Day 3:** Filter by production, schedule maintenance window
4. **Migrate:** Use systemd service for production VMs

### Selective Migration

```bash
# Find all database servers
Press / → type "mysql" OR "postgres" OR "mongodb"

# Review each
Press d on each VM to inspect

# Select only production
Space on each production DB server

# Dry-run to verify
Press r → Enter → review → Esc

# Execute
Press r → Enter → y
```

## See Also

- [Migration Architecture](../architecture.md)
- [CLI Reference](../cli-reference.md)
- [Daemon API](../api-reference.md)
- [Troubleshooting Guide](../troubleshooting.md)
