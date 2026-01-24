# API Documentation

This directory contains comprehensive API documentation for HyperSDK.

## API References

1. **[API Overview](00-overview.md)** - Introduction to HyperSDK APIs and architecture
2. **[Daemon API](01-daemon-api.md)** - REST API reference for the hypervisord daemon
3. **[API Endpoints](02-endpoints.md)** - Complete endpoint reference with examples
4. **[New Features](03-new-features.md)** - Recently added API features and capabilities

## API Categories

### Daemon REST API

The hypervisord daemon exposes a RESTful API for managing VM exports and migrations:
- VM export operations
- Job management and monitoring
- Status and health checks
- Configuration management

### Authentication

All API requests require authentication. See [Security Best Practices](../security-best-practices.md) for credential management.

### Rate Limiting

API endpoints implement rate limiting to prevent abuse. See individual endpoint documentation for limits.

## Quick Start

```bash
# Start the daemon
sudo systemctl start hypervisord

# Check daemon status
curl http://localhost:8080/api/v1/status

# List export jobs
curl http://localhost:8080/api/v1/jobs
```

## Related Documentation

- [Getting Started](../getting-started.md) - Setup and initial configuration
- [Configuration Reference](../configuration-reference.md) - Configuration options
- [Integration Guides](../integration/) - Integrating with external systems
- [Examples](../../examples/) - API usage examples

## API Versioning

HyperSDK uses semantic versioning for its APIs. Current API version: **v1**

Breaking changes will result in a new API version (v2, etc.) while maintaining backward compatibility.
