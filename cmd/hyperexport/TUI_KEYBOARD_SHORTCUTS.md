# HyperExport TUI - Keyboard Shortcuts Quick Reference

**Print this page for quick reference at your desk!**

---

## Essential Navigation

```
↑ / k         Move cursor up
↓ / j         Move cursor down
Enter         Confirm / Proceed
Esc           Go back / Cancel
q             Quit application
Ctrl+C        Force quit
? / h         Toggle help
```

---

## VM Selection & Filtering

```
Space         Select/deselect VM
a             Select all visible VMs
n             Deselect all VMs
/             Search VMs by name
c             Clear all filters
s             Cycle sort mode (name/cpu/memory/storage/power)
p             Filter by power state (on/off/all)
i / I         View VM details
```

---

## Quick Filters

```
1             Production VMs
2             Development VMs
3             Large VMs (>16GB RAM)
4             Small VMs (<4GB RAM)
```

---

## Main Features (A-Z)

```
b / B         Bulk Operations Manager
C             Compare VMs (2-4 selected)
e / E         Export Queue Management
G             Advanced Filter Builder
H             Export History View
L             Live Logs Viewer
m             Bookmarks & Saved Filters
M             Performance Metrics Dashboard
p / P         Export Preview (in confirm screen)
R             Resource Allocation Planner
S             Snapshot Manager (for current VM)
v / V         Toggle Split-Screen Mode
W             Migration Wizard
x / X         Quick Actions Menu (for current VM)
]             Folder Tree View
```

---

## Split-Screen Mode (v)

```
v / V         Toggle split-screen on/off
Tab           Switch between left/right panes
↑↓ / k/j      Navigate in focused pane
```

---

## Export Queue (e)

```
↑↓ / k/j      Navigate queue
^             Move item up in queue
v             Move item down in queue
1             Set high priority
2             Set normal priority
3             Set low priority
d             Remove from queue
Enter         Start processing queue
```

---

## Export History (H)

```
↑↓ / k/j      Navigate history
f             Filter by status (all/success/failed)
d             Filter by date (all/today/week/month)
/             Search in history
Enter         View export details
```

---

## Live Logs (L)

```
↑↓ / k/j      Scroll through logs
f             Filter by level (all/info/warn/error/debug)
/             Search in logs
a             Toggle auto-scroll
c             Clear logs
```

---

## Folder Tree View (])

```
↑↓ / k/j      Navigate tree
Enter         Expand/collapse folder
Space         Select VM in tree
Esc / b       Return to list view
```

---

## Quick Actions Menu (x)

```
↑↓ / k/j      Navigate actions
Enter         Execute selected action
Esc           Cancel and close menu
```

**Available Actions:**
- Power On/Off/Restart
- Create Snapshot
- Quick Export
- VM Details

---

## Bulk Operations (b)

```
↑↓ / k/j      Navigate operations
Enter         Execute bulk operation
Esc           Cancel and close
```

**Available Operations:**
- Power On/Off/Restart All
- Snapshot All
- Bulk Export
- Update VMware Tools

---

## VM Comparison (C)

**Requires 2-4 VMs selected**

```
1             Overview mode
2             Resources mode (CPU/Memory)
3             Storage mode
4             Network mode
↑↓ / k/j      Scroll comparison
Esc           Exit comparison
```

---

## Bookmarks & Filters (m)

```
↑↓ / k/j      Navigate bookmarks/filters
Enter         Apply selected bookmark/filter
s             Save current selection as bookmark
d             Delete selected bookmark
Tab           Switch between Bookmarks/Filters tabs
Esc           Exit bookmarks view
```

---

## Performance Metrics (M)

```
1             Overview mode
2             CPU metrics mode
3             Memory metrics mode
4             Storage metrics mode
r / R         Refresh metrics
Esc           Exit metrics dashboard
```

---

## Filter Builder (G)

```
↑↓ / k/j      Navigate filter fields
t / T         Toggle boolean fields
m / M         Toggle match mode (AND/OR)
p / P         Preview matching results
s / S         Save filter
a / A         Apply filter to VM list
c / C         Clear all filter criteria
Esc           Exit filter builder
```

**Filter Fields:**
- Name Pattern (wildcards: `*`, `?`)
- Power State
- OS Pattern
- CPU Range (Min/Max)
- Memory Range (Min/Max)
- Storage Range (Min/Max)
- Folder Pattern
- Match Mode (ALL=AND, ANY=OR)
- Case Sensitive
- Use Regex

---

## Snapshot Manager (S)

**Modes: List / Create / Details / Tree**

```
↑↓ / k/j      Navigate snapshots
c / C         Create new snapshot
d / D         Delete selected snapshot
r / R         Revert to selected snapshot
t / T         Toggle tree/list view
l / L         Switch to list view (from tree)
Enter         View snapshot details
Esc           Exit snapshot manager
```

**Create Snapshot:**
```
t / T         Toggle memory/quiesce options
Enter         Confirm and create
```

---

## Resource Planner (R)

```
1             Overview mode (cluster summary)
2             Hosts mode (per-host details)
3             Plan mode (VM placement planning)
4             Optimize mode (recommendations)
5             Recommendations mode (best practices)
↑↓ / k/j      Navigate (in hosts/optimize modes)
g / G         Change optimization goal (in optimize mode)
Esc           Exit resource planner
```

**Optimization Goals:**
- Balanced
- CPU
- Memory
- Storage
- Cost

---

## Migration Wizard (W)

**7-Step Workflow**

```
n / N         Next step
b / B         Back to previous step
1-5           Select option (context-dependent)
Enter         Start migration (final step only)
Esc           Cancel wizard
```

**Steps:**
1. Select VMs (review)
2. Source Config (vSphere/AWS/Azure/Hyper-V)
3. Target Config (vSphere/AWS/Azure/KVM/OVF)
4. Migration Mode (Cold/Hot/Snapshot)
5. Validation (automatic checks)
6. Schedule (now or later)
7. Review (final confirmation)

---

## Common Workflows

### Export Selected VMs
```
1. Select VMs with Space or a
2. Press Enter to confirm
3. Press p to preview (optional)
4. Press y or Enter to start export
5. Press e to view queue
```

### Create and Export with Snapshot
```
1. Navigate to VM
2. Press S for snapshot manager
3. Press c to create snapshot
4. Press Esc to return
5. Select VM with Space
6. Press Enter and export
```

### Bulk Operation on Filtered VMs
```
1. Press / to search or use quick filters
2. Press a to select all matching
3. Press b for bulk operations
4. Select operation with ↑↓
5. Press Enter to confirm
```

### Compare VMs
```
1. Select 2-4 VMs with Space
2. Press C for comparison
3. Press 1/2/3/4 to switch views
4. Press Esc when done
```

### Plan VM Migration
```
1. Select VMs for migration
2. Press W for wizard
3. Follow steps with n
4. Press 1-5 to make selections
5. Review validation results
6. Press Enter to start
```

---

## Tips & Tricks

**Vim-Style Navigation:**
- Use `k` and `j` instead of arrow keys for faster navigation
- More efficient once you learn the muscle memory

**Quick Filter Combinations:**
- Use `/` search + `p` power filter together
- Apply quick filter then refine with search

**Bookmark Frequently Used Selections:**
- Select VMs → Press `m` → Press `s` → Save
- Restore anytime with `m` → `Enter`

**Save Complex Filters:**
- Build in filter builder (`G`)
- Save with `s` for reuse
- Share filter JSON with team

**Monitor Long Operations:**
- Keep logs open (`L`) while exporting
- Use split-screen (`v`) to watch progress
- Check queue status (`e`) periodically

**Resource Planning Before Migration:**
- Press `R` to check cluster capacity
- Use plan mode (`3`) for selected VMs
- Verify sufficient resources before proceeding

---

## Color Indicators

**VM List:**
- 🟢 Green `●` - Powered On
- ⚫ Gray `●` - Powered Off
- 🔵 Blue highlight - Selected VM
- 🟡 Yellow `▶` - Current cursor position

**Resource Bars:**
- 🟢 Green - <60% utilized (healthy)
- 🟡 Yellow - 60-80% utilized (caution)
- 🔴 Red - >80% utilized (warning)

**Validation Checks:**
- ✓ Green - Passed
- ⚠ Yellow - Warning
- ✗ Red - Failed

**Host Status:**
- 🟢 Healthy - Well-balanced
- 🟡 Underutilized - Resources available
- 🔴 Overcommitted - Resources constrained

---

## Quick Troubleshooting

**VMs not showing?**
- Press `c` to clear filters
- Press `p` to reset power filter
- Check vSphere connection

**Can't select VMs?**
- Check if filters are hiding them
- Ensure cursor is on VM (not folder)
- Try `a` to select all visible

**Export failing?**
- Press `p` for preview to check disk space
- Review logs with `L`
- Check history `H` for error details

**Keyboard shortcuts not working?**
- Press `Esc` to return to main view
- Check you're in correct mode
- Press `?` for context-specific help

---

## Memory Aids

**BCHM** - Bulk, Compare, History, Metrics (major features)

**SLRW** - Snapshots, Logs, Resources, Wizard (management features)

**GMX** - filter builder (G), bookMarks (m), quick actions (X)

**Numbers 1-5:**
- Quick filters in main view
- Mode switching in multi-mode features
- Option selection in wizards

**Power State:**
- `p` for power filter
- `x` → power actions
- `b` → bulk power operations

---

**Print this page and keep it handy! 📄**

*For detailed information, see TUI_USER_GUIDE.md*
