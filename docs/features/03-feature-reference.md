# Feature Reference

This document provides a comprehensive overview of all features in the hypersdk interactive VM migration tool.

## Table of Contents

- [Core Features](#core-features)
- [Advanced Features](#advanced-features)
- [Performance Optimizations](#performance-optimizations)
- [Security Features](#security-features)

## Core Features

### Interactive TUI

Full-featured Terminal User Interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

**Capabilities:**
- Browse all VMs in vCenter
- Real-time VM discovery
- Multi-VM selection
- Progress monitoring
- Error handling and recovery

**Supported terminals:**
- Linux terminals (xterm, konsole, gnome-terminal, etc.)
- macOS Terminal and iTerm2
- tmux and screen sessions
- SSH sessions

### VM Selection

Flexible VM selection with visual feedback.

**Selection modes:**
- Individual (Space key)
- Bulk select all (a key)
- Bulk deselect all (n key)
- Checkbox indicators [✓] and [ ]

**Selection persistence:**
- Selections maintained across filters
- Selections maintained across sorts
- Selection count in status bar

### Migration Modes

Two execution modes for different use cases:

#### Terminal Mode (Interactive)
- Real-time progress display
- Immediate error feedback
- Best for short migrations
- Requires active terminal

#### Systemd Service Mode (Background)
- Runs as transient systemd unit
- Survives SSH disconnections
- Ideal for long-running migrations
- Monitor via journalctl

### Multi-Provider Support

Connect to vCenter via multiple methods:

**Daemon API:**
```bash
hyperctl migrate --daemon http://localhost:8080
```

**Direct Connection (Environment Variables):**
```bash
export GOVC_URL='https://vcenter/sdk'
export GOVC_USERNAME='user@vsphere.local'
export GOVC_PASSWORD='password'
export GOVC_INSECURE=1
hyperctl migrate
```

## Advanced Features

### Search

Case-insensitive full-text search across VM properties.

**Search targets:**
- VM name
- vCenter path
- Guest OS type

**Features:**
- Live preview with match count
- Shows first 10 results
- Instant filtering
- Backspace to edit
- Esc to cancel

**Performance:** O(n) linear scan, instant for 1000+ VMs

**Example searches:**
```
ubuntu           → Finds all Ubuntu VMs
web-server       → Finds VMs with "web-server" in name
/production/     → Finds VMs in production folder
windows server   → Finds Windows Server VMs
```

### Filtering

Power state filtering with easy toggling.

**Filter options:**
1. All VMs (no filter)
2. Powered ON only
3. Powered OFF only

**Use cases:**
- Migrate only offline VMs first
- Target only running production systems
- Verify powered-off dev environments

**Implementation:** O(n) filter pass, optimized with early exit

### Sorting

Multi-criteria sorting with efficient algorithms.

**Sort modes:**
1. **Name** - Alphabetical (A→Z)
2. **CPU** - CPU count (descending)
3. **Memory** - Memory size (descending)
4. **Storage** - Disk size (descending)
5. **Power** - Powered ON first, then OFF

**Features:**
- Stable sort (preserves relative order)
- Secondary sort by name for deterministic results
- Case-insensitive name sorting

**Performance:** O(n log n) using Go's optimized sort.Slice

**Algorithm:** Introsort (combination of quicksort, heapsort, insertion sort)

### Detail View

Comprehensive VM information display.

**Information shown:**
- Full VM name (no truncation)
- vCenter path
- Power state
- Guest operating system
- CPU count
- Memory (GB and MB)
- Storage (formatted and bytes)
- Current selection status

**Actions:**
- Toggle selection from detail view
- Quick access via `d` or `i` keys

### Dry-Run Mode

Preview migrations without execution.

**Features:**
- Toggle on/off with `r` key
- Visual indicators throughout UI
- Blocks actual migration execution
- Blocks systemd service creation
- Perfect for testing and training

**Use cases:**
- Test complex filter combinations
- Verify bulk selections
- Preview resource requirements
- Training new operators
- Demonstrate migration workflow

## Performance Optimizations

### Algorithmic Improvements

All operations optimized for large-scale deployments:

| Operation | Old | New | Improvement |
|-----------|-----|-----|-------------|
| Sorting | O(n²) bubble sort | O(n log n) introsort | 15-100x faster |
| Bulk select | O(n×m) nested loops | O(n) map lookup | 33x faster |
| Filter+sort | Multiple passes | Single pass | 2x faster |

**Benchmarks:**

100 VMs:
- Sort: 10,000 ops → 664 ops (15x faster)
- Bulk select: 5,000 ops → 150 ops (33x faster)

1,000 VMs:
- Sort: 1,000,000 ops → 9,966 ops (100x faster)
- Bulk select: 500,000 ops → 1,500 ops (333x faster)

### Memory Optimization

- Zero-allocation sorting (in-place)
- Map pre-allocation for known sizes
- Slice capacity hints
- Lazy filter application

### Rendering Optimization

- Windowed display (shows 20 VMs)
- Scroll without re-render
- Cached style calculations
- Efficient string building

## Security Features

### Path Traversal Prevention

All filenames and paths sanitized before file operations.

**Sanitization:**
- Remove `..` sequences
- Strip null bytes `\x00`
- Replace directory separators
- Remove shell metacharacters
- Limit to 255 characters

**Protected operations:**
- VM export directory creation
- OVF file writing
- Disk image storage

**Test coverage:** 14 comprehensive security tests

### Input Validation

All user inputs validated before processing.

**Validations:**
- Search query length limits
- Cursor bounds checking
- Array index validation
- Null/empty handling

### Credential Protection

Sensitive information never exposed.

**Protections:**
- Passwords excluded from JSON (`json:"-"`)
- Redacted() method for logging
- No password echoing in terminal
- Secure environment variable handling

**Example:**
```go
// Password never appears in logs or API responses
jd.Redacted()  // Returns copy with password = "***REDACTED***"
```

### Concurrency Safety

All shared state protected from race conditions.

**Synchronization:**
- Mutex protection for job state
- Deep copy on read operations
- WaitGroup for goroutine tracking
- Context-based cancellation

## Feature Matrix

| Feature | Status | Performance | Test Coverage |
|---------|--------|-------------|---------------|
| Search | ✅ | O(n) | 7 tests |
| Filter (power) | ✅ | O(n) | 4 tests |
| Sort (all modes) | ✅ | O(n log n) | 8 tests |
| Bulk select/deselect | ✅ | O(n) | 4 tests |
| Detail view | ✅ | O(1) | 3 tests |
| Dry-run mode | ✅ | N/A | 2 tests |
| Path sanitization | ✅ | O(n) | 14 tests |
| Credential protection | ✅ | O(1) | 2 tests |
| Concurrent job management | ✅ | O(1) | 6 tests |

**Total test coverage:** 50+ tests across all features

## Feature Interactions

### Combined Filters

Multiple filters work together:

```
Search: "ubuntu" + Power: ON + Sort: Memory
  ↓
Shows: All powered-on Ubuntu VMs sorted by memory (largest first)
```

### Filter Persistence

Selections persist across operations:

```
1. Search "web" → Select all (3 VMs selected)
2. Clear search → Still shows 3 selected
3. Filter "power: ON" → Still shows 3 selected (may be visible or hidden)
4. Select all visible → Adds to existing selection
```

### Sort Stability

When values are equal, secondary sort by name:

```
Sort by CPU:
  VM-A (8 CPU) ← First
  VM-C (4 CPU) ← Before VM-D (both 4 CPU, but A < D alphabetically)
  VM-D (4 CPU)
  VM-B (2 CPU) ← Last
```

## Planned Features

Future enhancements under consideration:

- [ ] Save/load filter presets
- [ ] Export selection to file
- [ ] Custom sort by multiple criteria
- [ ] Tag-based filtering
- [ ] Cluster/host filtering
- [ ] Regex search support
- [ ] Migration scheduling
- [ ] Batch size limiting
- [ ] Progress persistence across restarts
- [ ] Migration history tracking

## See Also

- [User Guide: Interactive Mode](user-guides/interactive-mode.md)
- [Architecture Documentation](architecture.md)
- [API Reference](api-reference.md)
- [Performance Benchmarks](benchmarks.md)
