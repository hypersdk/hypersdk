# HyperExport - New Huh-Based TUI

## Overview

The interactive TUI has been completely rewritten using `huh` (Charm's form library) instead of the complex Bubbletea implementation. This provides a simpler, more reliable, and more maintainable interface.

## What Changed

### Old Implementation (Bubbletea)
- **File**: `interactive_tui.go` (now renamed to `interactive_tui.go.old`)
- **Size**: 9000+ lines of code
- **Complexity**: 20+ phases/modes
- **Issues**: Complex state management, difficult to maintain, prone to panics
- **Dependencies**: Multiple bubbletea components with complex initialization

### New Implementation (Huh)
- **File**: `interactive_huh.go`
- **Size**: ~470 lines of code (95% reduction!)
- **Simplicity**: 4 clear steps
- **Benefits**: Simple, reliable, easy to maintain
- **Dependencies**: Just `huh` for forms, minimal setup

## Features

### 1. VM Selection
- Multi-select interface with filterable list
- Shows VM details: name, power state, CPU, RAM, storage
- Search/filter capability built-in
- Orange theme highlighting

### 2. Export Configuration
- Template selection (Quick Export, Production Backup, Development, Archive)
- Custom configuration options:
  - Output directory
  - Parallel downloads (1-8)
  - Export format (OVF/OVA)
  - Compression
  - Verification
- Conditional forms (advanced options only shown if requested)

### 3. Confirmation & Summary
- Clear summary of:
  - Selected VMs count
  - Total resources (CPU, RAM, storage)
  - Export settings
- Final confirmation before execution

### 4. Export Execution
- Progress indication with spinners
- Error handling with user choice to continue
- Detailed logging
- Success/failure reporting

## Orange Theme

The TUI uses a consistent orange color scheme:
- **Primary Orange**: `#FF9E64` - Vibrant peach/orange for highlights
- **Secondary Orange**: `#E0AF68` - Golden amber for accents
- **Deep Orange**: `#D35400` - Dark orange for contrast

Theme is applied to:
- Form borders
- Titles
- Selectors
- Selected options
- Summary boxes

## Usage

```bash
build/hyperexport --interactive
```

Or with the `--tui` flag:

```bash
build/hyperexport --tui
```

## Workflow

1. **Connect** - Automatically connects to vSphere
2. **Load VMs** - Fetches list of available VMs
3. **Select** - Multi-select interface to choose VMs
4. **Configure** - Choose template or customize export settings
5. **Confirm** - Review summary and confirm
6. **Export** - Executes exports with progress tracking

## Code Structure

```
interactive_huh.go
├── runInteractiveHuh()      # Main entry point
├── selectVMs()               # VM selection interface
├── configureExport()         # Export configuration
├── confirmAndExecute()       # Summary and execution
└── Helper functions
    ├── printBanner()
    ├── truncate()
    ├── sanitizeFilename()
```

## Error Handling

- **Connection errors**: Caught before entering TUI
- **No VMs**: Gracefully exits with message
- **Export failures**: Offers to continue with remaining VMs
- **User cancellation**: Clean exit at any step

## Benefits Over Old Implementation

1. **95% less code** - Easier to understand and maintain
2. **No crashes** - Simpler state management, no complex message passing
3. **Better UX** - Clear, focused workflow instead of confusing modes
4. **Faster** - Less overhead, no complex rendering logic
5. **Maintainable** - Simple forms instead of custom UI components
6. **Modern** - Uses latest Charm libraries (2026)

## Migration Notes

### Old Files Backed Up
- `interactive_tui.go` → `interactive_tui.go.old`
- `tui_cloud.go` → `tui_cloud.go.old`
- `tui_cloud_test.go` → `tui_cloud_test.go.old`
- `tui_cloud_integration_test.go` → `tui_cloud_integration_test.go.old`

### Old Function
- `runInteractiveTUI()` → `runInteractiveTUI_OLD()` (commented out in main.go)

### New Function
- `runInteractiveHuh()` - Active implementation

## Dependencies

```go
github.com/charmbracelet/huh v0.8.0        // Form library
github.com/charmbracelet/lipgloss          // Styling
github.com/pterm/pterm                      // Spinners & banners
```

## Future Enhancements

Possible additions (all optional, keep it simple):
- Cloud upload configuration step
- Snapshot management (if needed)
- Batch export from file
- Export history viewer

## Testing

Build and test:
```bash
go build -o build/hyperexport ./cmd/hyperexport
build/hyperexport --interactive
```

The TUI should:
1. Show HyperExport banner
2. Connect to vSphere
3. Load and display VMs
4. Allow multi-selection
5. Show configuration options
6. Execute exports with progress
7. Report success/failure

## Troubleshooting

**Q: Forms don't show orange theme**
A: Theme is applied via `WithTheme(theme)` - make sure it's passed to all forms

**Q: Can't select multiple VMs**
A: Use spacebar to select, enter to confirm

**Q: How to search VMs?**
A: Forms are filterable - just start typing

**Q: Want to cancel?**
A: Press ESC at any step, or select "No/Cancel" in confirmations
