# Alibaba Cloud (Aliyun) Integration Guide

## Overview

HyperSDK provides comprehensive Alibaba Cloud integration supporting **ECS** (Elastic Compute Service) for instance management and **OSS** (Object Storage Service) for backup storage. The implementation uses official Alibaba Cloud Go SDKs with intelligent retry mechanisms and network monitoring.

## Features

### Alibaba Cloud ECS (Elastic Compute Service)
- ✅ List all ECS instances
- ✅ Get detailed instance information
- ✅ Stop and start instances
- ✅ Create custom images from instances
- ✅ Create disk snapshots
- ✅ Export images to OSS
- ✅ Wait for image creation (polling with timeout)
- ✅ Delete custom images with cleanup

### Alibaba Cloud OSS (Object Storage Service)
- ✅ Upload files and streams with progress tracking
- ✅ Download objects with retry support
- ✅ List objects with marker-based pagination
- ✅ Delete objects and cleanup
- ✅ Check object existence
- ✅ Native `oss://` URL support
- ✅ Automatic endpoint detection from region

### Reliability Features
- **Retry with Exponential Backoff** - 5 attempts, 2s→4s→8s→16s→30s delays
- **Network-Aware Retry** - Pauses during network outages, resumes automatically
- **Smart Error Detection** - Distinguishes retryable (5xx, timeouts) from non-retryable (404, 403) errors
- **Progress Tracking** - Real-time progress callbacks during uploads/downloads
- **Automatic Endpoint Configuration** - Auto-detects OSS endpoint from region

## Configuration

### Method 1: Configuration File

Create `/etc/hypervisord/config.yaml`:

```yaml
alibaba_cloud:
  # Authentication (RAM User AccessKey)
  access_key_id: "LTAI4G..."
  access_key_secret: "your-access-key-secret"

  # Region
  region_id: "cn-hangzhou"      # Available: cn-beijing, cn-shanghai, etc.

  # OSS Configuration
  bucket: "vm-backups"           # OSS bucket name

  # Export Settings
  export_format: "qcow2"         # Image format: qcow2, raw

  enabled: true
```

### Method 2: Environment Variables

```bash
# Alibaba Cloud Authentication
export ALIBABA_CLOUD_ACCESS_KEY_ID="LTAI4G..."
export ALIBABA_CLOUD_ACCESS_KEY_SECRET="your-access-key-secret"
export ALIBABA_CLOUD_REGION_ID="cn-hangzhou"

# Cloud Storage URL
export CLOUD_STORAGE_URL="oss://vm-backups/exports/"
```

## Usage Examples

### Export Instance

```bash
# Interactive export
./hyperexport -provider alibabacloud -vm i-abc123...

# Non-interactive with options
./hyperexport \
  -provider alibabacloud \
  -vm i-abc123... \
  -output /backup/exports \
  -format qcow2 \
  -compress
```

### Upload to OSS

```bash
# Export instance and upload to OSS
./hyperexport \
  -vm i-abc123... \
  -upload oss://vm-backups/2024-01-21/
```

## License

LGPL-3.0-or-later - See LICENSE file for details
