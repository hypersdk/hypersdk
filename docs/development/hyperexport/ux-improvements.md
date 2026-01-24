# Default TUI UX Improvements

This document describes the enhancements made to the default interactive TUI (simple VM selector).

## Overview

The default TUI is shown when you run `hyperexport` without specifying a VM name. It has been completely redesigned with a modern, polished user experience.

## What Changed

### Before
```
Select a VM to export [type to search]:
  vm1
  vm2
  vm3
> vm4
```

### After
```
 ██   ██ ██    ██ ██████  ███████ ██████
 ██   ██  ██  ██  ██   ██ ██      ██   ██
 ███████   ████   ██████  █████   ██████
 ██   ██    ██    ██      ██      ██   ██
 ██   ██    ██    ██      ███████ ██   ██

 ███████ ██   ██ ██████   ██████  ██████  ████████
 ██       ██ ██  ██   ██ ██    ██ ██   ██    ██
 █████     ███   ██████  ██    ██ ██████     ██
 ██       ██ ██  ██      ██    ██ ██   ██    ██
 ███████ ██   ██ ██       ██████  ██   ██    ██

┌─ VM Selection ────────────────────────────────┐
│ Found 201 virtual machines                    │
│                                                │
│ 💡 Use ↑/↓ arrows to navigate                 │
│ 💡 Press / to search and filter               │
│ 💡 Press Enter to select                      │
│ 💡 Press Ctrl+C to cancel                     │
└────────────────────────────────────────────────┘

━━━━━━━━━━━━━━━━━ Select VM to Export ━━━━━━━━━━━━━━━━━

Select a VM to export [type to search]:
  esx8.0-rhel9.5-x86_64
  esx8.0-sles11sp4-x86_64
  esx8.0-ubuntu22.04.5
> esx8.0-win10-x86_64-efi
  esx8.0-win2022-x86_64

 ✔ SELECTED: esx8.0-win10-x86_64-efi
```

## Key Improvements

### 1. Screen Clearing
- **Before**: VM selector appeared below all previous output
- **After**: Screen is cleared for a clean, focused view

### 2. Branded Header
- **Before**: No branding
- **After**: Large "HYPEREXPORT" ASCII art in cyan/blue gradient
- Creates professional first impression
- Reinforces brand identity

### 3. Instructional Box
- **Before**: Minimal instructions
- **After**: Beautiful bordered box with:
  - VM count ("Found 201 virtual machines")
  - Navigation instructions with emojis
  - Search/filter help
  - Keyboard shortcuts

### 4. Sorted VM List
- **Before**: VMs in random order
- **After**: Alphabetically sorted for easy navigation
- Original indices preserved for correct selection

### 5. Enhanced Selection Prompt
- **Before**: Plain text prompt
- **After**: Full-width header with background color
- Clear visual separation from instructions

### 6. Better Search UX
- **Before**: Basic filter
- **After**:
  - Inline search with "[type to search]" reminder
  - Limited height (15 items) for better scrolling
  - Instant filtering as you type

### 7. Success Confirmation
- **Before**: `Success: Selected vm-name`
- **After**: Custom styled prefix box "SELECTED" with green background
- Clear visual confirmation

### 8. Enhanced VM Info Display

#### Before:
```
┌────────────────────────────────┐
│ Property    │ Value            │
├─────────────┼──────────────────┤
│ Name        │ my-vm            │
│ Power State │ poweredOn        │
│ Guest OS    │ Ubuntu Linux     │
└────────────────────────────────┘
```

#### After:
```
━━━━━━━━━━━━━ 📋 Virtual Machine Details ━━━━━━━━━━━━━

┌──────────────────┬─────────────────────────────────┐
│ Property         │ Value                           │
├──────────────────┼─────────────────────────────────┤
│ 🖥️  VM Name      │ my-vm                           │
│ ⚡ Power State   │ 🟢 poweredOn                    │
│ 💿 Guest OS      │ Ubuntu Linux (64-bit)           │
│ 🧠 Memory        │ 16384 MB (16.0 GB)              │
│ ⚙️  vCPUs        │ 8                               │
│ 💾 Storage       │ 512.0 GB                        │
│ 📁 Path          │ /datacenter/vm/my-vm            │
└──────────────────┴─────────────────────────────────┘
```

Features:
- Icon prefixes for each property
- Bold VM name
- Memory shown in both MB and GB
- Full VM path included
- Cyan-colored header
- Unicode box-drawing characters

### 9. Enhanced Export Summary

#### Before:
```
Export Summary
┌────────────┬───────────┐
│ Metric     │ Value     │
├────────────┼───────────┤
│ Duration   │ 5m30s     │
└────────────┴───────────┘
```

#### After:
```
 ███████ ██    ██  ██████  ██████ ███████ ███████ ███████
 ██      ██    ██ ██      ██      ██      ██      ██
 ███████ ██    ██ ██      ██      █████   ███████ ███████
      ██ ██    ██ ██      ██      ██           ██      ██
 ███████  ██████   ██████  ██████ ███████ ███████ ███████

━━━━━━━━━━━━━━━━━ ✅ Export Summary ━━━━━━━━━━━━━━━━━

┌──────────────────┬──────────────────────────────────┐
│ Metric           │ Value                            │
├──────────────────┼──────────────────────────────────┤
│ 🖥️  VM Name      │ my-vm                            │
│ ⏱️  Duration     │ 5m30s                            │
│ 💾 Total Size    │ 512.0 GB                         │
│ ⚡ Avg Speed     │ 156.4 MB/s                       │
│ 📦 Files Exported│ 4                                │
│ 📁 Output Dir    │ /exports/my-vm                   │
│ 📋 Manifest      │ /exports/my-vm/manifest.json     │
└──────────────────┴──────────────────────────────────┘
```

Features:
- Large "SUCCESS" ASCII art in green
- Transfer speed calculation (MB/s)
- Icons for each metric
- Green header bar
- Color-coded values (cyan for numbers, gray for paths)
- Manifest path highlighted in green

## User Experience Benefits

### 1. **Professional Appearance**
- Branded headers create trust
- Consistent visual language
- Modern, polished interface

### 2. **Better Discoverability**
- Clear instructions upfront
- Visual cues (emojis, colors, icons)
- Hints for keyboard shortcuts

### 3. **Easier Navigation**
- Sorted alphabetically
- Search with visual feedback
- Limited scroll height prevents overwhelm

### 4. **Reduced Cognitive Load**
- Screen clearing focuses attention
- Icons make scanning easier
- Color coding highlights important info

### 5. **Faster Workflow**
- Quick search/filter
- Keyboard-optimized
- Clear feedback at each step

### 6. **More Information**
- Speed calculations
- Both MB and GB for memory
- Full paths for reference

## Technical Details

### Colors Used
- **Cyan**: Primary brand color, headers
- **Green**: Success states, positive info
- **Gray**: Secondary info, paths
- **Black on Colored**: High contrast headers

### Icons
- 🖥️ VM/Computer
- ⚡ Power/Speed
- 💿 OS/Disk
- 🧠 Memory
- ⚙️ CPU
- 💾 Storage
- 📁 Directory/Path
- 📋 Manifest/Document
- 📦 Files/Package
- ⏱️ Time/Duration
- ✅ Success
- 💡 Tips/Info

### Layout Principles
1. Top-down flow
2. Generous whitespace
3. Visual hierarchy (big headers → content → details)
4. Consistent spacing
5. Box-drawing for structure

## Code Changes

### Modified Functions

1. **selectVMInteractive()**
   - Added screen clearing
   - Added big text branding
   - Added instruction box
   - Implemented alphabetical sorting
   - Enhanced selection confirmation

2. **displayVMInfo()**
   - Added header bar
   - Added icons for properties
   - Enhanced table formatting
   - Added full VM path

3. **showExportSummary()**
   - Added "SUCCESS" banner
   - Added speed calculation
   - Added icons for metrics
   - Color-coded values
   - Enhanced layout

## Usage

The enhanced TUI appears automatically when you run:

```bash
hyperexport
# or
hyperexport --provider vsphere
```

For the advanced multi-select TUI, use:

```bash
hyperexport -interactive
# or
hyperexport -tui
```

## Compatibility

- Works on all terminals that support ANSI colors
- Degrades gracefully on limited terminals
- No dependencies required
- Cross-platform (Linux, macOS, Windows)
