# HyperExport Cloud TUI - Quick Reference

## Overview

The HyperExport TUI now includes comprehensive cloud storage integration, allowing you to export VMs and upload them to S3, Azure, GCS, or SFTP directly from the interactive interface.

## Quick Start

```bash
# Launch interactive mode
hyperexport --interactive

# In TUI:
# 1. Select VMs (Space to select, Enter to continue)
# 2. Press 'u' to configure cloud upload
# 3. Choose provider (S3, Azure, GCS, or SFTP)
# 4. Enter credentials step-by-step
# 5. Confirm and export
```

## Supported Providers

| Provider | Icon | Setup Required |
|----------|------|----------------|
| Amazon S3 | ☁️ | Access key + Secret key |
| Azure Blob | 🔷 | Account name + Account key |
| Google Cloud Storage | 🌩️ | Service account JSON |
| SFTP | 🔐 | Username + Password/Key |

## Key Features

### Interactive Configuration
- **Step-by-step input** - Guided credential entry
- **Password masking** - Secure credential entry (••••)
- **Real-time validation** - Immediate feedback
- **Progress indicators** - "Step 2 of 5"
- **Visual confirmation** - See what you've configured

### Real-Time Progress
- **Upload progress bars** - Visual feedback
- **Transfer speed** - MB/s tracking
- **File counter** - Files uploaded/total
- **ETA calculation** - Time remaining estimate

### Smart Features
- **Stream upload mode** - Skip local copy
- **Keep local option** - Preserve local backup
- **Resume support** - Continue interrupted uploads
- **Multi-part uploads** - Efficient large files

## Keyboard Shortcuts

```
VM Selection Screen:
  u/U       Configure cloud upload
  Space     Select/deselect VM
  Enter     Continue to confirmation
  a         Select all
  n         Deselect all
  1-7       Quick filters
  q         Quit

Cloud Provider Selection:
  ↑/↓       Navigate providers
  Enter     Select provider
  s         Toggle stream upload
  l         Toggle keep local
  Esc       Back to VM selection

Credentials Input:
  Type      Enter text
  Backspace Delete character
  Enter     Next field/Continue
  Esc       Back to provider selection

Confirmation Screen:
  y/Enter   Start export
  u         Configure cloud upload
  n/Esc     Go back
  q         Quit
```

## Example Workflows

### S3 Backup Workflow
```
1. hyperexport --interactive
2. Select VMs (Space key)
3. Press 'u' for cloud upload
4. Select "Amazon S3"
5. Enter:
   - Bucket: my-backups
   - Region: us-east-1
   - Access Key: AKIAIOSFODNN7EXAMPLE
   - Secret Key: ••••••••
   - Prefix: prod/vms
6. Press 'y' to start
7. Monitor progress
```

### Azure Quick Upload
```
1. hyperexport --interactive
2. Select VMs
3. Press 'u'
4. Select "Azure Blob Storage"
5. Enter:
   - Container: vm-backups
   - Account: mystorageaccount
   - Key: ••••••••
   - Prefix: exports
6. Start export
```

### Multi-Cloud Strategy
```
# Primary: S3
hyperexport --interactive
# Configure S3, export

# Secondary: Azure (run again)
hyperexport --interactive
# Configure Azure, export same VMs

# Result: VMs backed up to both clouds
```

## Documentation

- **[TUI_CLOUD_GUIDE.md](TUI_CLOUD_GUIDE.md)** - Complete user guide (900+ lines)
  - Detailed workflows
  - Provider setup instructions
  - Security best practices
  - Troubleshooting
  - Cost optimization
  - Real-world examples

- **[TESTING.md](TESTING.md)** - Testing guide
  - Unit test examples
  - Integration test setup
  - Mock environments
  - CI/CD integration

## Testing

### Unit Tests
```bash
# Run all unit tests
go test -v ./cmd/hyperexport/

# Test cloud TUI specifically
go test -v -run TestCloud ./cmd/hyperexport/

# With coverage
go test -v -cover ./cmd/hyperexport/
```

### Integration Tests
```bash
# Setup environment
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export TEST_S3_BUCKET="test-bucket"

# Run integration tests
go test -tags=integration -v ./cmd/hyperexport/

# Test specific provider
go test -tags=integration -v -run TestS3Integration ./cmd/hyperexport/
```

See [TESTING.md](TESTING.md) for complete testing documentation.

## Code Structure

```
cmd/hyperexport/
├── tui_cloud.go                     # Cloud TUI implementation (600+ lines)
│   ├── Cloud provider selection
│   ├── Credentials input screens
│   ├── Upload progress visualization
│   └── Cloud storage browser
│
├── tui_cloud_test.go                # Unit tests (500+ lines)
│   ├── Configuration tests
│   ├── Phase transition tests
│   ├── Validation tests
│   └── Benchmarks
│
├── tui_cloud_integration_test.go    # Integration tests (400+ lines)
│   ├── S3 integration
│   ├── Azure integration
│   ├── GCS integration
│   ├── SFTP integration
│   └── Large file uploads
│
├── interactive_tui.go               # Main TUI (modified)
│   └── Integrated cloud support
│
├── cloud_storage.go                 # Cloud interface
├── cloud_s3.go                      # S3 implementation
├── cloud_azure.go                   # Azure implementation
├── cloud_gcs.go                     # GCS implementation
└── cloud_sftp.go                    # SFTP implementation
```

## Environment Setup

### AWS S3
```bash
export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
export AWS_REGION="us-east-1"
```

### Azure Blob Storage
```bash
export AZURE_STORAGE_ACCOUNT="mystorageaccount"
export AZURE_STORAGE_KEY="your-account-key"
```

### Google Cloud Storage
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
```

### SFTP
```bash
# Password-based (less recommended)
export SFTP_PASSWORD="your-password"

# Key-based (recommended)
ssh-keygen -t rsa -b 4096 -f ~/.ssh/hypersdk_key
ssh-copy-id -i ~/.ssh/hypersdk_key.pub user@sftp.example.com
```

## Security Notes

⚠️ **Never commit credentials to version control**

✅ **Best practices:**
- Use environment variables
- Use credential files in `~/.config/`
- Use IAM roles when possible (AWS)
- Use key-based auth for SFTP
- Rotate credentials regularly
- Use least-privilege permissions

## Performance Tips

### Optimize Upload Speed
1. **Use nearest region** - Reduce latency
2. **Enable parallel uploads** - `--parallel 4`
3. **Use compression** - `--compress` (if bandwidth-limited)
4. **Stream mode** - `--stream-upload` (skip local disk)

### Reduce Costs
1. **Use lifecycle policies** - Auto-delete old backups
2. **Choose appropriate storage class**:
   - S3: Standard → Infrequent Access → Glacier
   - Azure: Hot → Cool → Archive
   - GCS: Standard → Nearline → Coldline
3. **Enable compression** - Reduce storage by 30-70%

## Troubleshooting

### "Authentication failed"
```bash
# Verify credentials are set
echo $AWS_ACCESS_KEY_ID

# Test with cloud CLI
aws s3 ls
az storage container list
gsutil ls
sftp user@host
```

### "Upload timeout"
```bash
# Increase timeout
hyperexport --upload-timeout 30m

# Check network
ping s3.amazonaws.com
ping blob.core.windows.net
```

### "Insufficient permissions"
```bash
# S3 - check IAM policy
aws iam get-user-policy --user-name hypersdk

# Azure - check account permissions
az role assignment list --assignee user@domain.com

# GCS - check service account
gcloud projects get-iam-policy PROJECT_ID
```

## Common Scenarios

### Daily Automated Backup
```bash
#!/bin/bash
export AWS_ACCESS_KEY_ID="$(cat ~/.aws/access_key)"
export AWS_SECRET_ACCESS_KEY="$(cat ~/.aws/secret_key)"

hyperexport \
  --batch production-vms.txt \
  --upload s3://backups/$(date +%Y-%m-%d) \
  --compress \
  --stream-upload
```

### Disaster Recovery
```bash
# Backup to 3 regions
for region in us-east-1 us-west-2 eu-west-1; do
  export AWS_REGION=$region
  hyperexport --vm critical-db \
    --upload s3://dr-$region/$(date +%Y%m%d)
done
```

### Compliance Backup
```bash
# Encrypted backup to compliant storage
hyperexport --vm production-db \
  --encrypt \
  --encrypt-method aes256 \
  --upload s3://compliance-backups \
  --verify
```

## API Reference

### Cloud Configuration Structure
```go
type cloudConfig struct {
    provider  CloudProvider  // s3, azure, gcs, sftp
    bucket    string        // S3/Azure/GCS bucket/container
    region    string        // AWS region
    accessKey string        // Access credentials
    secretKey string        // Secret credentials
    host      string        // SFTP host
    port      string        // SFTP port
    prefix    string        // Path prefix
}
```

### Cloud Providers
```go
const (
    CloudProviderNone  CloudProvider = "none"
    CloudProviderS3    CloudProvider = "s3"
    CloudProviderAzure CloudProvider = "azure"
    CloudProviderGCS   CloudProvider = "gcs"
    CloudProviderSFTP  CloudProvider = "sftp"
)
```

## FAQ

**Q: Can I use S3-compatible storage (MinIO, Wasabi)?**
A: Yes, set custom endpoint in S3 configuration.

**Q: How do I resume an interrupted upload?**
A: Uploads automatically resume on retry. Use `--resume` flag.

**Q: Can I upload to multiple clouds simultaneously?**
A: Not in TUI, but possible via command line with separate runs.

**Q: Is my data encrypted during upload?**
A: Yes, all providers use TLS/HTTPS by default.

**Q: How do I delete old backups?**
A: Use cloud provider lifecycle policies or the cloud browser in TUI.

**Q: What's the maximum file size?**
A: S3: 5TB, Azure: 190TB, GCS: 5TB, SFTP: unlimited (filesystem-dependent)

## Support

- **Issues**: https://github.com/hypersdk/hypersdk/issues
- **Documentation**: [TUI_CLOUD_GUIDE.md](TUI_CLOUD_GUIDE.md)
- **Testing**: [TESTING.md](TESTING.md)

## License

Same as HyperSDK main project (LGPL-3.0-or-later)
