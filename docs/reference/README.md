# Reference Documentation

This directory contains technical reference documentation for HyperSDK.

## Contents

- **[CLI Reference](cli-reference.md)** - Command-line interface reference for all HyperSDK tools
- **[Configuration Reference](configuration-reference.md)** - Complete configuration file reference
- **[Performance Tuning](performance-tuning.md)** - Performance optimization guide
- **[Troubleshooting Guide](troubleshooting-guide.md)** - Common issues and solutions

## Quick Reference

### Command Line Tools

HyperSDK provides three main command-line tools:
- **hyperctl** - Interactive TUI for VM migration
- **hyperexport** - Batch VM export tool
- **hypervisord** - Daemon service for automated migrations

See [CLI Reference](cli-reference.md) for complete command syntax and options.

### Configuration

Configuration can be provided via:
- YAML configuration files
- JSON configuration files
- Command-line flags
- Environment variables

See [Configuration Reference](configuration-reference.md) for all available options.

### Performance

For optimal performance:
- Configure appropriate bandwidth limits
- Use concurrent export workers
- Enable compression for network transfers
- Monitor system resources

See [Performance Tuning](performance-tuning.md) for detailed optimization strategies.

### Getting Help

If you encounter issues:
1. Check the [Troubleshooting Guide](troubleshooting-guide.md)
2. Review logs in `/var/log/hypervisord/`
3. Consult the [API Documentation](../api/)
4. Report issues on GitHub

## Related Documentation

- [Getting Started](../getting-started.md) - Initial setup and first migration
- [User Guides](../user-guides/) - Step-by-step usage guides
- [API Documentation](../api/) - REST API reference
- [Examples](../../examples/) - Configuration examples
