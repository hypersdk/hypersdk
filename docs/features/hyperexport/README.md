# Hyperexport Features

This directory contains feature documentation specific to the `hyperexport` tool.

## Overview

Hyperexport is HyperSDK's batch VM export tool with advanced features for automated migrations.

## Feature Documentation

### Core Features
- **[Features Overview](features-overview.md)** - Complete list of hyperexport features
- **[Export Resumption](export-resumption.md)** - Resume interrupted exports
- **[Bandwidth Throttling](bandwidth-throttling.md)** - Network bandwidth control
- **[Multi-Cloud Concurrent Export](multi-cloud-concurrent.md)** - Parallel multi-cloud exports
- **[Shell Completion](shell-completion.md)** - Bash/Zsh command completion

### TUI (Text User Interface)
- **[User Guide](user-guide.md)** - Complete TUI user guide
- **[Modern TUI](modern-tui.md)** - Modern terminal interface features
- **[Interactive TUI](interactive-tui.md)** - Interactive mode documentation
- **[UI Features](ui-features.md)** - UI enhancements and capabilities
- **[Keyboard Shortcuts](keyboard-shortcuts.md)** - TUI keyboard reference
- **[Cloud TUI Guide](cloud-tui-guide.md)** - Cloud-specific TUI features
- **[Cloud TUI README](cloud-tui-readme.md)** - Cloud TUI overview

### Migration Features
- **[Retry Guide](retry-guide.md)** - Retry mechanisms and strategies

## Quick Start

```bash
# Interactive mode with TUI
hyperexport --interactive

# Batch export with config file
hyperexport --config export-config.yaml

# Resume interrupted export
hyperexport --resume --job-id abc123
```

## Related Documentation

- [CLI Reference](../../reference/cli-reference.md) - Command-line options
- [Configuration Reference](../../reference/configuration-reference.md) - Config file format
- [Cloud Providers](../../cloud-providers/) - Platform-specific guides
- [Testing](../../testing/hyperexport-testing.md) - Testing guide
