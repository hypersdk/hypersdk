# HyperExport Interactive TUI - User Guide

**Version:** 1.0
**Last Updated:** 2026-01-23

## Table of Contents

1. [Getting Started](#getting-started)
2. [Core Features](#core-features)
3. [Enhanced Features](#enhanced-features)
4. [Keyboard Shortcuts Reference](#keyboard-shortcuts-reference)
5. [Workflow Examples](#workflow-examples)
6. [Tips and Best Practices](#tips-and-best-practices)
7. [Troubleshooting](#troubleshooting)

---

## Getting Started

### Launching the Interactive TUI

```bash
# Launch interactive mode
hyperexport -interactive

# Or use the alias
hyperexport -tui
```

### First-Time Usage

When you launch the TUI, you'll see:
1. **VM Selection Screen** - List of all available VMs from your vSphere environment
2. **Status Bar** - Shows current filters, sort mode, and selected VM count
3. **Help Footer** - Quick reference for common keyboard shortcuts

### Basic Navigation

- **↑↓ / k/j** - Move cursor up/down
- **Space** - Select/deselect VM
- **Enter** - Proceed to next step (when VMs selected)
- **Esc** - Go back or exit
- **q** - Quit application
- **?/h** - Toggle help panel

---

## Core Features

### VM Selection and Filtering

#### Searching VMs
**Shortcut:** `/`

1. Press `/` to open search
2. Type your search query (searches VM names)
3. Press Enter to apply
4. Press `c` to clear filters

#### Sorting VMs
**Shortcut:** `s`

Cycles through sort modes:
- By Name (alphabetical)
- By CPU (cores)
- By Memory (RAM size)
- By Storage (disk size)
- By Power State

#### Filtering by Power State
**Shortcut:** `p`

- Press once: Show only powered-on VMs
- Press twice: Show only powered-off VMs
- Press third time: Show all VMs

#### Quick Filters
**Shortcut:** `1`, `2`, `3`, `4`

Predefined filters:
- `1` - Production VMs
- `2` - Development VMs
- `3` - Large VMs (>16GB RAM)
- `4` - Small VMs (<4GB RAM)

### Bulk Selection
**Shortcuts:** `a`, `n`

- `a` - Select all visible VMs
- `n` - Deselect all visible VMs
- `Space` - Toggle individual VM selection

### VM Details
**Shortcut:** `i` or `I`

View detailed information about the VM under cursor:
- Basic info (name, path, power state)
- Resources (CPU, memory, storage)
- Guest OS information
- Network configuration
- Disk details

---

## Enhanced Features

### 1. Multi-Pane Split-Screen Layout (Enhancement #8)
**Shortcut:** `v` or `V`

**Description:** View VM list and details side-by-side

**Usage:**
1. Press `v` to toggle split-screen mode
2. Use `Tab` to switch focus between left (list) and right (details) panes
3. Navigate normally in focused pane
4. Press `v` again to exit split-screen

**Benefits:**
- Quick reference while browsing VMs
- No need to enter/exit details view
- Efficient for comparing multiple VMs

---

### 2. Export Queue Management (Enhancement #9)
**Shortcut:** `e` or `E`

**Description:** Manage export queue with priorities and reordering

**Features:**
- View all queued exports
- Reorder with drag/drop controls
- Set priorities (High, Normal, Low)
- Pause/resume exports
- Remove from queue

**Usage:**
1. Select VMs and confirm export
2. Press `e` to view export queue
3. Use `↑↓/k/j` to navigate queue
4. Press `^` to move export up
5. Press `v` to move export down
6. Press `1/2/3` to set priority (High/Normal/Low)
7. Press `d` to remove from queue
8. Press `Enter` to start processing

**Queue Status:**
- **Pending** - Waiting to start
- **Running** - Currently exporting
- **Completed** - Successfully exported
- **Failed** - Export failed (check logs)

---

### 3. Export History View (Enhancement #10)
**Shortcut:** `H`

**Description:** View past export operations with filtering

**Features:**
- Complete export history
- Filter by status (success/failed)
- Filter by date range
- Filter by provider
- Search by VM name
- Export statistics

**Usage:**
1. Press `H` to open history view
2. Use `↑↓/k/j` to browse history
3. Press `f` to cycle status filters (all/success/failed)
4. Press `d` to cycle date filters (all/today/week/month)
5. Press `/` to search
6. Press `Enter` to view details

**History Entry Details:**
- VM name and path
- Start and end times
- Duration
- Status (success/failed)
- File size
- Export path
- Error messages (if failed)

---

### 4. Live Logs Viewer (Enhancement #11)
**Shortcut:** `L`

**Description:** Real-time log monitoring with filtering

**Features:**
- Live log streaming
- Filter by log level (INFO/WARN/ERROR/DEBUG)
- Search within logs
- Auto-scroll toggle
- Configurable log retention

**Usage:**
1. Press `L` to open logs viewer
2. Use `↑↓/k/j` to scroll through logs
3. Press `f` to cycle log level filters
4. Press `/` to search logs
5. Press `a` to toggle auto-scroll
6. Press `c` to clear logs

**Log Levels:**
- **DEBUG** - Detailed diagnostic information
- **INFO** - General informational messages
- **WARN** - Warning messages (non-critical)
- **ERROR** - Error messages (critical)

**Auto-Scroll:**
- Enabled: Automatically scroll to newest logs
- Disabled: Manual scroll control

---

### 5. Folder Tree View (Enhancement #12)
**Shortcut:** `]`

**Description:** Hierarchical folder tree with VM grouping

**Features:**
- Tree view of datacenter structure
- Expand/collapse folders
- VM count per folder
- Nested folder navigation
- Select VMs within folders

**Usage:**
1. Press `]` to switch to tree view
2. Use `↑↓/k/j` to navigate tree
3. Press `Enter` to expand/collapse folders
4. Press `Space` on VM to select
5. Press `Esc` or `b` to return to list view

**Tree Indicators:**
- `▼` - Expanded folder
- `▶` - Collapsed folder
- `├─` - Tree branch
- `└─` - Last item in branch

---

### 6. Real-Time Export Preview (Enhancement #13)
**Shortcut:** `p` or `P`

**Description:** Preview export details before starting

**Features:**
- File size estimates
- Disk breakdown
- Transfer time estimation
- Disk space validation
- File list preview

**Usage:**
1. Select VMs for export
2. Press `p` in confirmation screen
3. Review export details:
   - Total size and file count
   - Estimated duration
   - Disk space check
   - Individual file breakdown
4. Press `y` or `Enter` to proceed
5. Press `Esc` to go back

**Estimates:**
- Based on 50 MB/s average transfer speed
- Includes 10% overhead for safety
- Checks available disk space
- Warns if insufficient space

---

### 7. Quick Actions Menu (Enhancement #14)
**Shortcut:** `x` or `X`

**Description:** Context-sensitive actions for selected VM

**Features:**
- Power operations (on/off/restart)
- Snapshot operations
- Export shortcuts
- VM information

**Actions Available:**

**Power Management:**
- Power On - Start powered-off VM
- Power Off - Gracefully shut down VM
- Force Off - Force power off
- Restart - Reboot VM

**Snapshots:**
- Create Snapshot - Take new snapshot
- View Snapshots - List all snapshots
- Revert - Restore to snapshot

**Export:**
- Quick Export - Export with defaults
- Custom Export - Configure export options

**Information:**
- VM Details - Full VM information
- Performance Stats - Resource usage

**Usage:**
1. Navigate to desired VM
2. Press `x` to open actions menu
3. Use `↑↓/k/j` to select action
4. Press `Enter` to execute
5. Press `Esc` to cancel

**Action Availability:**
- Actions are enabled/disabled based on VM state
- Grayed out actions are not available
- Hover over action to see requirements

---

### 8. Bulk Operations Manager (Enhancement #15)
**Shortcut:** `b` or `B`

**Description:** Perform operations on multiple VMs simultaneously

**Features:**
- Bulk power operations
- Bulk snapshots
- Bulk export
- Progress tracking
- Safety confirmations

**Available Operations:**

**Power Management:**
- Power On All - Start all selected VMs
- Power Off All - Shutdown all selected VMs
- Restart All - Reboot all selected VMs

**Snapshots:**
- Snapshot All - Create snapshots for all VMs
- Consolidate Snapshots - Merge snapshot chains
- Delete Old Snapshots - Remove snapshots older than N days

**Maintenance:**
- Update VMware Tools - Update tools on all VMs
- Apply Tags - Bulk tagging
- Move to Folder - Relocate VMs

**Export:**
- Bulk Export - Export all selected VMs
- Incremental Backup - Backup changed VMs only

**Usage:**
1. Select multiple VMs (use `a` for all)
2. Press `b` to open bulk operations
3. Use `↑↓/k/j` to select operation
4. Press `Enter` to confirm
5. Confirm destructive operations
6. Monitor progress

**Safety:**
- Confirmation required for destructive operations
- Progress shown per VM
- Can cancel in-progress operations
- Failed VMs reported separately

---

### 9. VM Comparison View (Enhancement #16)
**Shortcut:** `C`

**Description:** Side-by-side comparison of 2-4 VMs

**Features:**
- Compare resources (CPU, memory, storage)
- Compare network configuration
- Compare storage layout
- Highlight differences
- Multiple comparison modes

**Comparison Modes:**
1. **Overview** - High-level comparison
2. **Resources** - CPU, memory, storage details
3. **Storage** - Disk configuration
4. **Network** - Network adapters and settings

**Usage:**
1. Select 2-4 VMs (use `Space`)
2. Press `C` to open comparison
3. Use `1/2/3/4` to switch modes
4. Use `↑↓/k/j` to scroll
5. Press `Esc` to exit

**Requirements:**
- Minimum 2 VMs selected
- Maximum 4 VMs (for readability)
- Works best with similar VMs

**Difference Highlighting:**
- Green: Value higher than others
- Red: Value lower than others
- Yellow: Different configuration

---

### 10. Saved Filters and Bookmarks (Enhancement #17)
**Shortcut:** `m`

**Description:** Save and reuse VM selections and filters

**Features:**
- Bookmark VM selections
- Save complex filters
- Quick access to saved sets
- Share bookmarks

**Bookmarks:**
- Save current selection as bookmark
- Name and describe bookmarks
- Restore selections instantly
- Delete old bookmarks

**Saved Filters:**
- Save filter configurations
- Reuse complex filter criteria
- Share filter definitions
- Import/export filters

**Usage:**

**Creating Bookmarks:**
1. Select desired VMs
2. Press `m` to open bookmarks
3. Press `s` to save bookmark
4. Enter name and description
5. Press `Enter` to confirm

**Applying Bookmarks:**
1. Press `m` to open bookmarks
2. Use `↑↓/k/j` to select bookmark
3. Press `Enter` to apply
4. VMs are automatically selected

**Managing Filters:**
1. Configure filters (search, power state, etc.)
2. Press `m` and switch to Filters tab
3. Press `s` to save current filters
4. Name the filter
5. Apply anytime with `Enter`

**Sample Bookmarks:**
- "Production Web Servers"
- "Development Environment"
- "VMs for Migration"
- "Backup Targets"

---

### 11. Performance Metrics Dashboard (Enhancement #18)
**Shortcut:** `M`

**Description:** Real-time infrastructure performance metrics

**Features:**
- Cluster-wide statistics
- Per-resource breakdowns
- Visual graphs and charts
- Distribution analysis

**Dashboard Modes:**
1. **Overview** - Cluster summary
2. **CPU** - CPU allocation and usage
3. **Memory** - Memory distribution
4. **Storage** - Storage capacity

**Metrics Displayed:**

**Overview Mode:**
- Total infrastructure resources
- Resource allocation percentages
- VM count and distribution
- OS distribution

**CPU Mode:**
- Total vCPUs allocated
- CPU distribution (1-2, 2-4, 4-8, 8+ cores)
- Average CPU per VM
- CPU allocation graph

**Memory Mode:**
- Total memory allocated
- Memory distribution (<4GB, 4-8GB, 8-16GB, 16GB+)
- Average memory per VM
- Memory allocation graph

**Storage Mode:**
- Total storage allocated
- Storage distribution (<100GB, 100-500GB, 500GB-1TB, 1TB+)
- Average storage per VM
- Storage capacity graph

**Usage:**
1. Press `M` to open metrics dashboard
2. Press `1/2/3/4` to switch modes
3. Press `r` to refresh metrics
4. Press `Esc` to exit

**Refresh:**
- Auto-refresh every 30 seconds
- Manual refresh with `r`
- Timestamp shown on screen

---

### 12. Advanced Filter Builder (Enhancement #19)
**Shortcut:** `G`

**Description:** Build complex filters with multiple criteria

**Features:**
- 13 configurable filter criteria
- AND/OR logic modes
- Wildcard and regex support
- Real-time preview
- Save filter configurations

**Filter Criteria:**
1. **Name Pattern** - VM name matching (wildcards: `web-*`, `*prod*`)
2. **Power State** - On/Off/Any
3. **OS Pattern** - Guest OS matching
4. **Min CPU** - Minimum vCPU count
5. **Max CPU** - Maximum vCPU count
6. **Min Memory** - Minimum RAM (GB)
7. **Max Memory** - Maximum RAM (GB)
8. **Min Storage** - Minimum disk space (GB)
9. **Max Storage** - Maximum disk space (GB)
10. **Folder Pattern** - Path matching
11. **Match Mode** - ALL (AND) or ANY (OR)
12. **Case Sensitive** - Case-sensitive matching
13. **Use Regex** - Enable regex patterns

**Usage:**

**Building Filters:**
1. Press `G` to open filter builder
2. Use `↑↓/k/j` to navigate fields
3. Enter values for desired criteria
4. Toggle boolean fields with `t`
5. Toggle match mode with `m`
6. See live match count at bottom

**Preview Results:**
1. Press `p` to preview matches
2. Review list of matching VMs
3. Press `a` to apply filter
4. Press `b` to go back to builder

**Save Filter:**
1. Press `s` to save filter
2. Enter filter name
3. Enter description
4. Press `Enter` to save
5. Filter saved to bookmarks

**Match Modes:**
- **ALL (AND)** - VM must match all criteria
- **ANY (OR)** - VM matches if any criterion matches

**Pattern Examples:**
- `web-*` - Matches "web-01", "web-prod", etc.
- `*prod*` - Matches any VM with "prod" in name
- `db-?` - Matches "db-1", "db-a", etc.

---

### 13. Snapshot Management Interface (Enhancement #20)
**Shortcut:** `S`

**Description:** Comprehensive snapshot management for VMs

**Features:**
- List all VM snapshots
- Create new snapshots
- Delete snapshots
- Revert to snapshots
- Tree view of snapshot hierarchy

**Snapshot Modes:**

**1. List View** (default)
- Shows all snapshots for selected VM
- Displays name, size, date, power state
- Current snapshot indicator
- Quick actions available

**2. Create Mode**
- Snapshot name (auto-generated or custom)
- Description field
- Include memory option
- Quiesce filesystem option

**3. Details Mode**
- Complete snapshot metadata
- Parent-child relationships
- Snapshot ID and timestamps
- Available actions (revert/delete)

**4. Tree View**
- Hierarchical snapshot visualization
- Parent-child relationships
- ASCII tree with branches
- Current snapshot highlighted

**Usage:**

**View Snapshots:**
1. Navigate to desired VM
2. Press `S` to open snapshot manager
3. Use `↑↓/k/j` to browse snapshots
4. Press `Enter` for details

**Create Snapshot:**
1. Press `c` in list view
2. Enter snapshot name (optional)
3. Enter description
4. Toggle memory/quiesce with `t`
5. Press `Enter` to create

**Revert to Snapshot:**
1. Select snapshot in list
2. Press `r` to revert
3. Confirm revert operation
4. VM restored to snapshot state

**Delete Snapshot:**
1. Select snapshot in list
2. Press `d` to delete
3. Confirm deletion
4. Snapshot removed

**Tree View:**
1. Press `t` in list view
2. See hierarchical structure
3. Navigate with `↑↓/k/j`
4. Press `l` to return to list

**Snapshot Options:**
- **Include Memory** - Captures RAM state (slower, larger)
- **Quiesce Filesystem** - Ensures filesystem consistency

**Best Practices:**
- Take snapshot before major changes
- Don't keep snapshots too long (disk space)
- Use descriptive names
- Clean up old snapshots regularly

---

### 14. Resource Allocation Planner (Enhancement #21)
**Shortcut:** `R`

**Description:** Plan and optimize resource allocation

**Features:**
- Cluster-wide resource overview
- Host-by-host breakdown
- VM placement planning
- Optimization recommendations
- Multiple optimization goals

**Planner Modes:**

**1. Overview Mode** (default)
- Cluster resource summary
- Visual capacity bars
- CPU, memory, storage totals
- Utilization percentages
- VM distribution stats

**2. Hosts Mode**
- Per-host resource details
- VM count per host
- Status indicators (healthy/overcommitted/underutilized)
- Resource utilization per host
- Sortable host list

**3. Plan Mode**
- Analyze selected VMs requirements
- Find suitable hosts
- Check capacity constraints
- Distribution recommendations

**4. Optimize Mode**
- Goal-based recommendations
- Optimization strategies
- Specific action items
- Resource balancing tips

**5. Recommendations Mode**
- General best practices
- Resource management tips
- DRS suggestions
- Capacity planning guidance

**Usage:**

**View Cluster Overview:**
1. Press `R` to open planner
2. See cluster-wide statistics
3. Resource bars show utilization
4. Color-coded (green/yellow/red)

**Check Host Resources:**
1. Press `2` to switch to hosts mode
2. Use `↑↓/k/j` to browse hosts
3. View per-host details
4. Identify overcommitted hosts

**Plan VM Allocation:**
1. Select VMs in main view
2. Press `R` then `3` for plan mode
3. See total requirements
4. View suitable hosts
5. Check if any host can accommodate

**Get Optimization Tips:**
1. Press `4` for optimize mode
2. Select optimization goal with `g`:
   - Balanced - Even distribution
   - CPU - CPU optimization
   - Memory - Memory optimization
   - Storage - Storage optimization
   - Cost - Cost reduction
3. Review recommendations
4. Apply suggestions

**Status Indicators:**
- **Green (Healthy)** - Resources well-balanced
- **Yellow (Underutilized)** - Resources available
- **Red (Overcommitted)** - Resources constrained

**Optimization Goals:**
- **Balanced** - Even resource distribution
- **CPU** - Optimize CPU allocation
- **Memory** - Optimize memory usage
- **Storage** - Optimize storage placement
- **Cost** - Minimize operational costs

**Navigation:**
- `1` - Overview mode
- `2` - Hosts mode
- `3` - Plan mode
- `4` - Optimize mode
- `5` - Recommendations mode
- `g` - Change optimization goal (in optimize mode)

---

### 15. Migration Wizard (Enhancement #22)
**Shortcut:** `W`

**Description:** Guided workflow for VM migrations

**Features:**
- 7-step guided wizard
- Platform compatibility checking
- Pre-migration validation
- Migration scheduling
- Time and cost estimates

**Wizard Steps:**

**Step 1: Select VMs**
- Review selected VMs
- See resource totals
- Validate selection

**Step 2: Source Configuration**
- Choose source platform:
  1. vSphere
  2. AWS
  3. Azure
  4. Hyper-V
- Confirm selection

**Step 3: Target Configuration**
- Choose target platform:
  1. vSphere
  2. AWS
  3. Azure
  4. KVM/Local
  5. OVF Export
- Confirm selection

**Step 4: Migration Mode**
- Select migration strategy:
  1. **Cold Migration** - Power off, migrate, power on (safest)
  2. **Hot Migration** - Minimal downtime (complex)
  3. **Snapshot Export** - Backup/clone
- Each mode has description

**Step 5: Validation**
- Automatic pre-migration checks:
  - ✓ VM selection
  - ✓ Platform compatibility
  - ✓ Migration mode
  - ✓ Storage capacity
  - ✓ Network connectivity
  - ✓ Permissions
- Pass/warning/failure summary
- Blocks if critical checks fail

**Step 6: Schedule**
- Choose timing:
  1. Migrate Now
  2. Schedule for Later
- See estimates:
  - Estimated duration
  - Total data size
  - Network bandwidth

**Step 7: Review**
- Final summary of all selections
- Review totals and estimates
- Ready/not ready indicator
- Start migration or go back

**Usage:**

**Start Wizard:**
1. Select VMs for migration
2. Press `W` to launch wizard
3. Follow step-by-step prompts

**Navigate Wizard:**
- `n` - Next step
- `b` - Back to previous step
- `1-5` - Select options (context-dependent)
- `Enter` - Start migration (step 7 only)
- `Esc` - Cancel and exit wizard

**Progress Indicator:**
- Shows all 7 steps
- Completed steps in green (●)
- Current step in blue (●)
- Pending steps in gray (○)
- Example: `● Step 1 → ● Step 2 → ○ Step 3 → ...`

**Validation Checks:**
All checks must pass to proceed:
- **Critical** - Must pass (blocks migration)
- **Warning** - Can proceed with caution
- **Info** - Informational only

**Estimates:**
- Duration: Based on 100 MB/s bandwidth
- Overhead: 5 minutes per VM
- Size: Sum of all VM storage
- Adjustable in settings

**Best Practices:**
- Run validation before scheduling
- Review all settings carefully
- Test with one VM first
- Schedule during low-traffic hours
- Backup before migration

---

### 16. Concurrent Multi-VM Exports with Live Progress (Enhancement #23)
**Automatic Feature** (activates after export confirmation)

**Description:** Export multiple VMs simultaneously with real-time progress tracking in split-screen layout

**Features:**
- Concurrent export operations (multiple VMs export in parallel)
- Split-screen layout (60% VM list / 40% export progress pane)
- Live progress tracking for all exports
- Real-time download speeds and ETAs
- Individual progress bars per VM
- Continue selecting more VMs while exports run
- Cancel exports with Ctrl+C or `q`

**How It Works:**

**Automatic Activation:**
1. Select multiple VMs for export
2. Press `Enter` to confirm
3. Validation screen appears
4. Press `y` or `Enter` to start
5. **Automatically** returns to VM selection with split-screen enabled
6. All exports start concurrently in the background

**Split-Screen Layout:**
```
┌─────────────────────────────────┬──────────────────────────┐
│ VM List (60%)                   │ Export Progress (40%)     │
│                                 │                          │
│ ▶ VM-web-01      [SELECTED]    │ Active Exports: 3        │
│   VM-db-01       [SELECTED]    │ Completed: 1             │
│   VM-app-01                    │ Running: 2               │
│                                 │ Failed: 0                │
│                                 │                          │
│                                 │ ⬇ VM-web-01             │
│                                 │   ████████░░░░ 65%      │
│                                 │   450MB / 680MB         │
│                                 │   23.5 MB/s • ETA: 10s  │
│                                 │                          │
│                                 │ ⬇ VM-db-01              │
│                                 │   ███░░░░░░░░ 30%       │
│                                 │   File 2/5              │
│                                 │   18.2 MB/s • ETA: 45s  │
└─────────────────────────────────┴──────────────────────────┘
```

**Export Status Indicators:**

**Status Icons:**
- `⬇` - Downloading (blue)
- `✓` - Completed (green)
- `✗` - Failed (red)
- `⊘` - Cancelled (yellow)
- `⋯` - Starting (gray)

**Progress Information Per Export:**
- VM name and current status
- Progress bar (40 characters width)
- Percentage complete (0-100%)
- Current file being downloaded
- File count progress (e.g., "File 3/7")
- Data transferred / Total size
- Real-time download speed (MB/s)
- Estimated time remaining (ETA)

**Usage While Exports Run:**

**Continue Working:**
- Navigate VM list normally with `↑↓/k/j`
- Select additional VMs with `Space`
- Apply filters with `/`, `s`, `p`
- View VM details with `i`
- Start more exports by selecting and pressing `Enter`

**Monitor Progress:**
- Right pane shows all active exports automatically
- Progress updates in real-time (every 500ms)
- Speed calculated from last 2 seconds of data
- ETA based on current speed

**Cancel Exports:**
- Press `Ctrl+C` to cancel all running exports
- Press `q` to quit (exports continue in background)
- Individual export cancellation: coming soon

**Export States:**

1. **Starting** - Export initialization
   - Setting up export directory
   - Creating snapshot (if enabled)
   - Preparing file list

2. **Downloading** - Active download
   - Shows progress bar
   - Displays current file name
   - Updates speed and ETA
   - File X/Y indicator

3. **Completed** - Successfully finished
   - Shows completion checkmark
   - Displays total size
   - Shows elapsed time
   - Files remain in export directory

4. **Failed** - Error occurred
   - Shows error icon
   - Displays error message
   - Check logs with `L` for details
   - Can retry export

5. **Cancelled** - User aborted
   - Shows cancellation icon
   - Partial files may remain
   - Can restart export

**Performance:**

**Parallel Downloads:**
- Each VM export uses 3 parallel downloads (default)
- Multiple VMs export concurrently (limited by system resources)
- Total concurrent downloads = VMs × 3
- Throttled to prevent system overload

**Resource Usage:**
- Memory: ~50MB per concurrent export
- Network: Limited by vSphere server and network bandwidth
- CPU: Minimal (I/O bound operation)
- Disk: Requires sufficient free space for all exports

**Progress Update Throttling:**
- UI updates every 500ms maximum
- Prevents excessive screen redraws
- Keeps terminal responsive
- Progress tracked continuously in background

**Best Practices:**

**Disk Space:**
- Ensure sufficient disk space before starting multiple exports
- Use export preview (`p`) to check size requirements
- Monitor available space during exports

**Network Considerations:**
- Start fewer concurrent exports on slow networks
- Monitor total bandwidth usage
- vSphere server may throttle connections

**System Resources:**
- Don't export too many VMs simultaneously (recommended: 3-5)
- Monitor system memory and CPU
- Close other applications if needed

**Workflow Tips:**
- Select and export smaller VMs first
- Group VMs by size for better management
- Use filters to organize export batches
- Monitor logs (`L`) for issues

**Example Workflow:**

**Export Multiple VMs:**
```
1. Select 3 VMs with Space
2. Press Enter to confirm
3. Validation runs automatically
4. Press y to start exports
5. Screen splits automatically:
   - Left: VM list (continue selecting)
   - Right: Export progress (all 3 VMs)
6. Select 2 more VMs while exports run
7. Press Enter again to export new VMs
8. All 5 exports now visible in progress pane
9. Monitor progress in real-time
10. Exports complete automatically
```

**Troubleshooting:**

**Progress Not Updating:**
- Check network connection
- Verify vSphere connectivity
- Look for errors in logs (`L`)

**Slow Export Speed:**
- Network bandwidth limitation
- vSphere server throttling
- Too many concurrent exports
- Reduce parallel export count

**Export Fails:**
- Check disk space
- Verify permissions
- Review error in logs (`L`)
- Check export history (`H`)

**Split-Screen Not Showing:**
- Ensure terminal is wide enough (minimum 100 columns)
- Resize terminal window
- Feature activates automatically after confirmation

---

## Keyboard Shortcuts Reference

### Global Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` | Confirm / Proceed |
| `Esc` | Go back / Cancel |
| `q` | Quit application |
| `?` / `h` | Toggle help |
| `Ctrl+C` | Force quit |

### VM Selection
| Key | Action |
|-----|--------|
| `Space` | Select/deselect VM |
| `a` | Select all visible VMs |
| `n` | Deselect all VMs |
| `/` | Search VMs |
| `c` | Clear filters |
| `s` | Cycle sort mode |
| `p` | Filter by power state |
| `1-4` | Apply quick filters |

### Views and Modes
| Key | Action |
|-----|--------|
| `i` / `I` | VM details |
| `v` / `V` | Toggle split-screen |
| `Tab` | Switch pane (split-screen) |
| `]` | Toggle tree view |
| `/` | Search |

### Enhanced Features
| Key | Action | Feature |
|-----|--------|---------|
| `e` / `E` | Export queue | Queue Management |
| `H` | Export history | History View |
| `L` | Live logs | Logs Viewer |
| `p` / `P` | Export preview | Preview |
| `x` / `X` | Quick actions | Actions Menu |
| `b` / `B` | Bulk operations | Bulk Ops |
| `C` | Compare VMs | Comparison |
| `m` | Bookmarks/Filters | Bookmarks |
| `M` | Metrics dashboard | Metrics |
| `G` | Filter builder | Filter Builder |
| `S` | Snapshot manager | Snapshots |
| `R` | Resource planner | Resources |
| `W` | Migration wizard | Migration |

### Feature-Specific

#### Queue Management (`e`)
| Key | Action |
|-----|--------|
| `^` | Move up in queue |
| `v` | Move down in queue |
| `1` | Set high priority |
| `2` | Set normal priority |
| `3` | Set low priority |
| `d` | Remove from queue |

#### History View (`H`)
| Key | Action |
|-----|--------|
| `f` | Filter by status |
| `d` | Filter by date |
| `/` | Search history |

#### Logs Viewer (`L`)
| Key | Action |
|-----|--------|
| `f` | Filter log level |
| `/` | Search logs |
| `a` | Toggle auto-scroll |
| `c` | Clear logs |

#### Snapshot Manager (`S`)
| Key | Action |
|-----|--------|
| `c` | Create snapshot |
| `d` | Delete snapshot |
| `r` | Revert to snapshot |
| `t` | Toggle tree view |
| `l` | Switch to list |

#### Resource Planner (`R`)
| Key | Action |
|-----|--------|
| `1` | Overview mode |
| `2` | Hosts mode |
| `3` | Plan mode |
| `4` | Optimize mode |
| `5` | Recommendations |
| `g` | Change goal |

#### Migration Wizard (`W`)
| Key | Action |
|-----|--------|
| `n` | Next step |
| `b` | Back step |
| `1-5` | Select option |

#### Filter Builder (`G`)
| Key | Action |
|-----|--------|
| `p` | Preview results |
| `s` | Save filter |
| `a` | Apply filter |
| `c` | Clear all |
| `t` | Toggle boolean |
| `m` | Toggle match mode |

---

## Workflow Examples

### Example 1: Export VMs to OVF Format

1. **Launch TUI**
   ```bash
   hyperexport -tui
   ```

2. **Search for VMs**
   - Press `/` to search
   - Type `web-prod`
   - Press Enter

3. **Select VMs**
   - Press `a` to select all matching
   - Or use `Space` to select individually

4. **Preview Export**
   - Press `Enter` to proceed
   - Press `p` for preview
   - Review file sizes and estimates

5. **Configure Export**
   - Press `Esc` to go back
   - Press `f` for advanced features
   - Configure snapshot/compression options

6. **Start Export**
   - Press `y` or `Enter` to confirm
   - Monitor progress in export view

7. **Check History**
   - Press `H` to view export history
   - Verify successful completion

### Example 2: Migrate VMs to AWS

1. **Select VMs for Migration**
   - Use filters to find VMs
   - Select with `Space` or `a`

2. **Open Migration Wizard**
   - Press `W`

3. **Step 1: Verify Selection**
   - Review VM list and totals
   - Press `n` for next

4. **Step 2: Source Platform**
   - Press `1` for vSphere
   - Press `n`

5. **Step 3: Target Platform**
   - Press `2` for AWS
   - Press `n`

6. **Step 4: Migration Mode**
   - Press `1` for Cold Migration
   - Press `n`

7. **Step 5: Validation**
   - Wait for checks to complete
   - Ensure all passed
   - Press `n`

8. **Step 6: Schedule**
   - Press `1` for "Migrate Now"
   - Or `2` for scheduled
   - Review estimates
   - Press `n`

9. **Step 7: Final Review**
   - Review all settings
   - Press `Enter` to start

10. **Monitor Progress**
    - Press `e` to view queue
    - Check export history with `H`

### Example 3: Find and Compare VMs

1. **Build Complex Filter**
   - Press `G` for filter builder
   - Set "Min Memory" to 8 GB
   - Set "Power State" to On
   - Set "OS Pattern" to `Windows*`
   - Press `p` to preview
   - Press `a` to apply

2. **Review Results**
   - Browse filtered VMs
   - Note VMs of interest

3. **Select VMs to Compare**
   - Select 2-4 VMs with `Space`
   - Press `C` for comparison

4. **Compare Resources**
   - Press `2` for resources mode
   - Review CPU/Memory differences
   - Press `3` for storage mode
   - Review disk configurations

5. **Save for Later**
   - Press `Esc` to exit comparison
   - Press `m` for bookmarks
   - Press `s` to save selection
   - Name: "Windows VMs 8GB+"

### Example 4: Snapshot and Export

1. **Select VM**
   - Navigate to desired VM

2. **Create Snapshot**
   - Press `S` for snapshot manager
   - Press `c` to create
   - Name: "Pre-migration backup"
   - Toggle "Quiesce" with `t`
   - Press `Enter`

3. **Verify Snapshot**
   - See snapshot in list
   - Press `Enter` for details
   - Press `Esc` to return

4. **Export VM**
   - Press `Esc` to exit snapshots
   - Select VM with `Space`
   - Press `Enter` to confirm
   - Choose export options
   - Start export

5. **Check Resource Usage**
   - Press `R` for resource planner
   - View cluster capacity
   - Check if export fits

### Example 5: Bulk Operations

1. **Filter Development VMs**
   - Press `2` for dev filter
   - Or use `/` and search "dev"

2. **Select All**
   - Press `a` to select all

3. **Open Bulk Operations**
   - Press `b` or `B`

4. **Snapshot All**
   - Navigate to "Snapshot All"
   - Press `Enter`
   - Confirm operation

5. **Monitor Progress**
   - See progress per VM
   - Wait for completion

6. **Export All**
   - Return to bulk ops
   - Select "Bulk Export"
   - Configure settings
   - Start export

7. **Check History**
   - Press `H`
   - Filter by status
   - Verify all succeeded

---

## Tips and Best Practices

### Performance Tips

1. **Use Filters Effectively**
   - Filter before selecting to reduce clutter
   - Save frequently-used filters
   - Use quick filters for common tasks

2. **Leverage Keyboard Shortcuts**
   - Learn vim-style navigation (`k/j`)
   - Use quick filters (`1-4`)
   - Master mode switching (`M`, `G`, `S`, etc.)

3. **Batch Operations**
   - Group similar VMs with bookmarks
   - Use bulk operations for multiple VMs
   - Queue exports to run overnight

4. **Monitor Resources**
   - Check resource planner before large exports
   - Ensure sufficient disk space
   - Monitor network bandwidth

### Organization Tips

1. **Use Bookmarks**
   - Create bookmarks for VM groups (prod, dev, test)
   - Name descriptively
   - Review and clean up periodically

2. **Save Complex Filters**
   - Build once, reuse many times
   - Share with team members
   - Document filter purposes

3. **Leverage Tree View**
   - Understand datacenter structure
   - Navigate by folder hierarchy
   - Select all VMs in a folder

### Safety Tips

1. **Preview Before Export**
   - Always check export preview
   - Verify disk space
   - Review file sizes

2. **Take Snapshots**
   - Snapshot before major operations
   - Name snapshots descriptively
   - Clean up old snapshots

3. **Validate Migrations**
   - Run migration wizard validation
   - Test with one VM first
   - Schedule during maintenance windows

4. **Check History**
   - Review export history regularly
   - Investigate failures promptly
   - Learn from patterns

### Efficiency Tips

1. **Split-Screen Mode**
   - Use for quick reference
   - Compare while browsing
   - Faster than entering/exiting details

2. **Quick Actions**
   - Press `x` for common operations
   - Avoid navigating through menus
   - Context-aware actions

3. **Live Logs**
   - Keep logs open during operations
   - Filter to ERROR for troubleshooting
   - Search logs for specific issues

4. **Queue Management**
   - Prioritize urgent exports
   - Reorder for optimization
   - Remove failed items

---

## Troubleshooting

### Common Issues

#### VMs Not Showing Up

**Problem:** VM list is empty or incomplete

**Solutions:**
1. Check vSphere connection:
   - Verify credentials
   - Check network connectivity
   - Ensure vCenter is accessible

2. Clear filters:
   - Press `c` to clear all filters
   - Check power state filter with `p`
   - Verify search is cleared (`/` then clear)

3. Refresh VM list:
   - Exit and re-launch TUI
   - Check vCenter permissions

#### Export Fails

**Problem:** Export operation fails

**Solutions:**
1. Check export history (`H`) for error details
2. Common causes:
   - Insufficient disk space
   - Network connectivity issues
   - VM locked by another operation
   - Permission denied

3. Retry with validation:
   - Use export preview (`p`)
   - Check disk space warnings
   - Verify VM is not running

#### Slow Performance

**Problem:** TUI is slow or unresponsive

**Solutions:**
1. Reduce VM count:
   - Use filters to limit visible VMs
   - Close tree view if open
   - Exit split-screen mode

2. Check system resources:
   - Monitor CPU/memory usage
   - Close other applications
   - Check network latency

3. Optimize display:
   - Disable auto-scroll in logs
   - Limit log retention
   - Close unused views

#### Keyboard Shortcuts Not Working

**Problem:** Key presses don't trigger actions

**Solutions:**
1. Check terminal compatibility:
   - Ensure terminal supports required key bindings
   - Try alternate shortcuts (e.g., `i` instead of `I`)

2. Modal context:
   - Some keys only work in specific modes
   - Press `Esc` to return to main view
   - Check help (`?`) for available keys

3. Terminal settings:
   - Disable key remapping
   - Check for conflicting shortcuts
   - Try different terminal emulator

#### Snapshot Operations Fail

**Problem:** Cannot create/delete/revert snapshots

**Solutions:**
1. Check VM state:
   - Some operations require VM to be powered off
   - Ensure no ongoing operations

2. Permissions:
   - Verify snapshot permissions in vCenter
   - Check quota limits

3. Disk space:
   - Ensure sufficient datastore space
   - Clean up old snapshots first

#### Migration Wizard Validation Fails

**Problem:** Validation checks fail in migration wizard

**Solutions:**
1. Review failed checks:
   - Read error messages carefully
   - Address each failed check

2. Common failures:
   - Platform not configured: Select source/target
   - No VMs selected: Go back and select VMs
   - Insufficient capacity: Check resource planner

3. Skip non-critical warnings:
   - Warnings don't block migration
   - Review and accept if acceptable

### Getting Help

**In-App Help:**
- Press `?` or `h` for quick help
- Each view has context-specific help footer

**Logs:**
- Press `L` to view live logs
- Check ERROR level for issues
- Search logs with `/`

**History:**
- Press `H` for export history
- Review failure details
- Look for patterns

**Documentation:**
- This guide: Comprehensive reference
- Comments in source code
- Git commit messages for feature details

**Support:**
- GitHub issues: Report bugs
- Feature requests: Suggest enhancements
- Community: Ask questions

---

## Advanced Topics

### Customization

Currently, the TUI uses a fixed color scheme and layout. Future enhancements will include:
- Settings manager for customization
- Configurable keyboard shortcuts
- Theme selection
- Default preferences

### Integration

The TUI integrates with:
- vSphere APIs for VM management
- Export history database
- Configuration files
- External tools (planned)

### Scripting

While the TUI is interactive, you can:
- Use command-line flags for automation
- Parse export history for reporting
- Integrate with CI/CD pipelines
- Script VM selection with filters

### Best Practices for Teams

1. **Standardize Naming**
   - Use consistent VM naming conventions
   - Organize VMs by folders
   - Use tags for categorization

2. **Share Bookmarks**
   - Export bookmark configurations
   - Share filter definitions
   - Document team standards

3. **Schedule Strategically**
   - Coordinate export windows
   - Use queue priorities
   - Monitor shared resources

4. **Monitor and Review**
   - Regular history reviews
   - Track export patterns
   - Optimize based on metrics

---

## Conclusion

The HyperExport Interactive TUI provides a powerful, keyboard-driven interface for managing VM exports and migrations. By mastering the features and shortcuts in this guide, you can:

- Efficiently browse and select VMs
- Preview and validate exports
- Manage snapshots and migrations
- Monitor operations in real-time
- Optimize resource allocation
- Organize and reuse configurations

**Next Steps:**
1. Practice basic navigation and selection
2. Explore each enhancement feature
3. Create bookmarks for common tasks
4. Master keyboard shortcuts
5. Review history and logs regularly

**Remember:**
- Use `?` for help anytime
- Preview before exporting
- Save configurations for reuse
- Monitor resources and logs

Happy exporting! 🚀

---

*For questions, issues, or feature requests, please visit: https://github.com/anthropics/hypersdk*
