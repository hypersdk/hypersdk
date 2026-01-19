# HyperExport TUI - Cloud Storage Integration Guide

## Overview

The HyperExport TUI now includes comprehensive cloud storage support, allowing you to export VMs and automatically upload them to Amazon S3, Azure Blob Storage, Google Cloud Storage, or SFTP servers directly from the interactive interface.

## Features

### Supported Cloud Providers

- **Amazon S3** ☁️
  - AWS S3 and S3-compatible storage (MinIO, Wasabi, DigitalOcean Spaces)
  - Support for custom endpoints
  - Automatic region detection

- **Azure Blob Storage** 🔷
  - Microsoft Azure Blob Storage
  - Container-based organization
  - SAS token support

- **Google Cloud Storage** 🌩️
  - GCS buckets
  - Service account authentication
  - Multi-region support

- **SFTP** 🔐
  - Secure File Transfer Protocol
  - Password or key-based authentication
  - Custom port support

## Quick Start

### Launch Interactive Mode with Cloud Upload

```bash
# Start interactive TUI
hyperexport --interactive

# Or use the alias
hyperexport --tui
```

### Basic Workflow

1. **Select VMs** - Browse and select VMs to export
2. **Press 'u'** - Configure cloud upload
3. **Choose Provider** - Select S3, Azure, GCS, or SFTP
4. **Enter Credentials** - Provide bucket name, access keys, etc.
5. **Confirm** - Review settings and start export
6. **Monitor Progress** - Real-time upload progress

## Using Cloud Upload in TUI

### Step 1: VM Selection

Navigate the VM list using keyboard shortcuts:

```
↑/k       Move up
↓/j       Move down
Space     Select/deselect VM
Enter     Continue to confirmation
u         Configure cloud upload (shortcut)
```

**Quick Filters:**
```
1         Powered ON VMs only
2         Powered OFF VMs only
3         Linux VMs
4         Windows VMs
5         High CPU (8+ cores)
6         High Memory (16GB+)
7         Large Storage (500GB+)
```

### Step 2: Cloud Provider Selection

Press **'u'** to open the cloud provider selection screen.

```
☁️  Cloud Storage Provider

Select a cloud storage provider for backup:

▶ 💾 Skip Cloud Upload
    Export to local storage only

  ☁️ Amazon S3
    AWS S3 or S3-compatible storage

  🔷 Azure Blob Storage
    Microsoft Azure Blob Storage

  🌩️ Google Cloud Storage
    Google Cloud Platform Storage

  🔐 SFTP Server
    Secure File Transfer Protocol

⚙️  Upload Options
  s: Stream upload (no local copy): ❌
  l: Keep local copy: ✅

↑/↓: Navigate | Enter: Select | Esc: Back | q: Quit
```

**Options:**
- **Stream Upload**: Upload directly without saving locally (saves disk space)
- **Keep Local**: Keep a local copy after uploading (recommended)

### Step 3: Enter Cloud Credentials

The TUI will guide you through entering credentials step-by-step.

#### Amazon S3 Configuration

```
🔧 Configure Amazon S3

S3 Bucket Name:
Enter the S3 bucket name (without s3:// prefix)

my-vm-backups█
Example: my-backup-bucket

Step 1 of 5
```

**Required Information:**
1. **Bucket Name** - S3 bucket (e.g., `my-vm-backups`)
2. **Region** - AWS region (e.g., `us-east-1`, `eu-west-1`)
3. **Access Key ID** - AWS access key (e.g., `AKIAIOSFODNN7EXAMPLE`)
4. **Secret Access Key** - AWS secret key (hidden with •••)
5. **Path Prefix** - Optional prefix (e.g., `prod/vms`)

**Environment Variables:**
```bash
# Alternatively, set credentials via environment:
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
```

#### Azure Blob Storage Configuration

```
🔧 Configure Azure Blob Storage

Container Name:
Enter the Azure container name

vm-backups█
Example: vm-backups

Step 1 of 4
```

**Required Information:**
1. **Container Name** - Azure container (e.g., `vm-backups`)
2. **Storage Account Name** - Azure account name
3. **Storage Account Key** - Azure account key (hidden with •••)
4. **Path Prefix** - Optional prefix (e.g., `exports/prod`)

**Environment Variables:**
```bash
export AZURE_STORAGE_ACCOUNT="mystorageaccount"
export AZURE_STORAGE_KEY="your-account-key"
```

#### Google Cloud Storage Configuration

```
🔧 Configure Google Cloud Storage

GCS Bucket Name:
Enter the Google Cloud Storage bucket name

my-gcs-bucket█
Example: my-gcs-bucket

Step 1 of 2
```

**Required Information:**
1. **Bucket Name** - GCS bucket (e.g., `my-gcs-bucket`)
2. **Path Prefix** - Optional prefix (e.g., `vm-exports`)

**Service Account Authentication:**
```bash
# Set service account credentials:
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"
```

#### SFTP Configuration

```
🔧 Configure SFTP Server

SFTP Host:
Enter the SFTP server hostname or IP

sftp.example.com█
Example: sftp.example.com

Step 1 of 5
```

**Required Information:**
1. **Host** - SFTP server hostname or IP
2. **Port** - SFTP port (default: 22)
3. **Username** - SFTP username
4. **Password** - SFTP password (or leave empty for key-based auth)
5. **Path Prefix** - Remote directory path

**Key-Based Authentication:**
- Leave password empty to use SSH key authentication
- Default key location: `~/.ssh/id_rsa`
- Custom key: Set via `--keyfile` flag

### Step 4: Confirmation Screen

Review your selection and cloud configuration before proceeding.

```
📋 Confirm Export

📦 web-server-01 | 4 CPU | 8.0 GB | 100.0G
📦 db-server-01 | 8 CPU | 16.0 GB | 500.0G

📊 Summary
VMs: 2 | CPUs: 12 | Memory: 24.0 GB | Storage: 600.0G

☁️  Cloud Upload
✓ Provider: s3 | Bucket: my-vm-backups | Prefix: prod/vms

✓ Disk space OK: 2.5T available

y/Y/Enter: Start export | u: Cloud upload | n/Esc: Go back | q: Quit
```

If cloud upload is not configured, you'll see:
```
☁️  Cloud upload: Not configured (press 'u' to configure)
```

### Step 5: Export and Upload Progress

Monitor real-time progress during export and upload.

**Local Export Phase:**
```
📦 Exporting VMs

web-server-01

[████████████████████░░░░░░░░░░░░░░░░░░░░] 65.3%

85.3 GB / 130.6 GB
Speed: 125.4 MB/s
Files: 12 / 18
Elapsed: 11m 23s

Export in progress... Press q to cancel
```

**Cloud Upload Phase:**
```
☁️  Uploading to Cloud

Uploading: web-server-01

[██████████████████████████████░░░░░░░░░░] 75.0%

98.0 GB / 130.6 GB
Speed: 45.2 MB/s
Files: 14 / 18

Upload in progress... Press q to cancel
```

### Step 6: Completion

```
✅ Export complete!

Local: /exports/web-server-01
Cloud: s3://my-vm-backups/prod/vms/web-server-01

Press q to quit
```

## Advanced Features

### Cloud Storage Browser

Browse and download previously uploaded exports.

```bash
# Launch cloud browser (future feature)
hyperexport --browse-cloud s3://my-bucket/exports
```

**Browser Interface:**
```
☁️ Cloud Storage Browser - Amazon S3

Found 15 files:

▶ 📄 web-server-01/web-server-01.ovf           2.5 GB  2026-01-20 14:30
  📄 web-server-01/web-server-01-disk1.vmdk  125.0 GB  2026-01-20 14:30
  📄 db-server-01/db-server-01.ovf             3.2 GB  2026-01-19 09:15
  📄 db-server-01/db-server-01-disk1.vmdk    500.0 GB  2026-01-19 09:15

↑/↓: Navigate | Enter/d: Download | x: Delete | r: Refresh | Esc: Back | q: Quit
```

### Export Profiles with Cloud

Save export configurations including cloud settings.

```bash
# Create profile with cloud upload
hyperexport --save-profile prod-backup \
  --provider vsphere \
  --format ova \
  --compress \
  --upload s3://my-bucket/prod \
  --stream-upload

# Use saved profile
hyperexport --interactive --profile prod-backup
```

### Batch Export with Cloud Upload

Export multiple VMs and upload to cloud in one operation.

```bash
# Using batch file
cat vms.txt
web-server-01
web-server-02
db-server-01

hyperexport --batch vms.txt \
  --upload s3://my-bucket/weekly-backup \
  --parallel 4
```

### Stream Upload Mode

Export directly to cloud without local storage:

```bash
# Stream mode (no local copy)
hyperexport --interactive \
  --upload s3://my-bucket/backups \
  --stream-upload
```

**Benefits:**
- No local disk space required
- Faster overall process
- Direct upload during export

**Considerations:**
- Cannot retry failed uploads
- No local backup
- Requires stable network

## Cloud Provider Setup

### Amazon S3

#### Create Bucket
```bash
aws s3 mb s3://my-vm-backups --region us-east-1
```

#### Create IAM Policy
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::my-vm-backups",
        "arn:aws:s3:::my-vm-backups/*"
      ]
    }
  ]
}
```

#### Create Access Keys
```bash
# Create IAM user
aws iam create-user --user-name hyperexport

# Attach policy
aws iam put-user-policy --user-name hyperexport \
  --policy-name S3Access --policy-document file://policy.json

# Generate access keys
aws iam create-access-key --user-name hyperexport
```

### Azure Blob Storage

#### Create Storage Account
```bash
az storage account create \
  --name mystorageaccount \
  --resource-group myresourcegroup \
  --location eastus \
  --sku Standard_LRS
```

#### Create Container
```bash
az storage container create \
  --name vm-backups \
  --account-name mystorageaccount
```

#### Get Access Keys
```bash
az storage account keys list \
  --account-name mystorageaccount \
  --resource-group myresourcegroup
```

### Google Cloud Storage

#### Create Bucket
```bash
gsutil mb -l us-east1 gs://my-gcs-bucket/
```

#### Create Service Account
```bash
# Create service account
gcloud iam service-accounts create hyperexport \
  --display-name="HyperExport Service Account"

# Grant permissions
gsutil iam ch serviceAccount:hyperexport@PROJECT_ID.iam.gserviceaccount.com:objectAdmin \
  gs://my-gcs-bucket

# Generate key file
gcloud iam service-accounts keys create key.json \
  --iam-account=hyperexport@PROJECT_ID.iam.gserviceaccount.com
```

### SFTP Server

#### Setup SSH Key Authentication
```bash
# Generate SSH key pair
ssh-keygen -t rsa -b 4096 -f ~/.ssh/hyperexport_key

# Copy public key to SFTP server
ssh-copy-id -i ~/.ssh/hyperexport_key.pub user@sftp.example.com

# Use with hyperexport
hyperexport --interactive --keyfile ~/.ssh/hyperexport_key
```

## Keyboard Shortcuts Reference

### Main Selection Screen
```
↑/k       Move cursor up
↓/j       Move cursor down
Space     Select/deselect VM
Enter     Continue to confirmation
u/U       Configure cloud upload

a         Select all visible VMs
n         Deselect all
A         Regex pattern selection
1-7       Quick filters

t/T       Export templates
s         Cycle sort mode
c         Clear all filters
h/?       Toggle help
q         Quit
Esc       Go back
```

### Cloud Provider Selection
```
↑/k       Navigate up
↓/j       Navigate down
Enter     Select provider
s         Toggle stream upload
l         Toggle keep local copy
Esc       Back to VM selection
q         Quit
```

### Cloud Credentials Input
```
Type      Enter characters
Backspace Delete last character
Enter     Continue to next field
Esc       Back to provider selection
q         Quit
```

### Confirmation Screen
```
y/Y/Enter Start export
u/U       Configure cloud upload
n/Esc     Go back to VM selection
q         Quit
```

### Cloud Browser (Future)
```
↑/k       Navigate up
↓/j       Navigate down
Enter/d   Download selected file
x/Del     Delete selected file
r         Refresh file list
Esc       Exit browser
q         Quit
```

## Troubleshooting

### Authentication Errors

**S3: "InvalidAccessKeyId"**
```
Check:
- Access key ID is correct
- Secret access key matches
- IAM user has necessary permissions
- Region is correct
```

**Azure: "AuthenticationFailed"**
```
Check:
- Storage account name is correct
- Account key is valid
- Container exists
- Network connectivity to Azure
```

**GCS: "PermissionDenied"**
```
Check:
- Service account JSON file path
- GOOGLE_APPLICATION_CREDENTIALS environment variable
- Service account has storage.objects.create permission
- Bucket exists and is accessible
```

**SFTP: "Permission denied"**
```
Check:
- Username is correct
- Password/key is correct
- SSH key permissions (chmod 600)
- Server allows password/key authentication
- Network connectivity on port 22 (or custom port)
```

### Upload Failures

**"Connection timeout"**
```
Solutions:
- Check network connectivity
- Verify firewall rules
- Try different region/endpoint
- Increase timeout settings
```

**"Insufficient storage space"**
```
Solutions:
- Check cloud storage quota
- Verify billing is active
- Check bucket/container limits
- Contact cloud provider support
```

**"File too large"**
```
Solutions:
- Enable multipart upload (automatic for >5GB)
- Use stream upload mode
- Split large disks (manual)
- Check provider limits (S3: 5TB, Azure: 190.7TB, GCS: 5TB)
```

### Performance Issues

**Slow upload speeds**
```
Optimization:
- Use nearest region
- Enable parallel uploads (--parallel)
- Check network bandwidth
- Use stream upload mode
- Enable compression (trade CPU for bandwidth)
```

**High memory usage**
```
Solutions:
- Use stream upload (no local buffering)
- Reduce parallel uploads
- Export fewer VMs at once
- Enable compression
```

## Security Best Practices

### Credentials Management

**Never hardcode credentials:**
```bash
# ❌ BAD - credentials in script
hyperexport --upload s3://bucket \
  --access-key AKIAIOSFODNN7EXAMPLE \
  --secret-key wJalrXUtnFEMI

# ✅ GOOD - use environment variables
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
hyperexport --upload s3://bucket
```

**Use credential files:**
```bash
# AWS credentials file
~/.aws/credentials

[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI

# Azure connection string
~/.azure/storage_connection_string
```

**Use IAM roles (AWS):**
```bash
# When running on EC2 with IAM role, no credentials needed
hyperexport --upload s3://bucket
```

### Network Security

**Use encryption in transit:**
- All cloud providers use HTTPS/TLS by default
- SFTP uses SSH encryption
- No configuration needed

**Restrict network access:**
```bash
# S3 bucket policy - IP restriction
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:*",
    "Resource": "arn:aws:s3:::my-bucket/*",
    "Condition": {
      "IpAddress": {
        "aws:SourceIp": "203.0.113.0/24"
      }
    }
  }]
}
```

**Use VPN/Private Links:**
- AWS PrivateLink for S3
- Azure Private Endpoints
- GCS VPC Service Controls
- SFTP over VPN

### Data Protection

**Enable versioning:**
```bash
# S3 versioning
aws s3api put-bucket-versioning \
  --bucket my-bucket \
  --versioning-configuration Status=Enabled

# Azure blob versioning
az storage account blob-service-properties update \
  --account-name mystorageaccount \
  --enable-versioning true
```

**Enable encryption at rest:**
- S3: SSE-S3, SSE-KMS, or SSE-C
- Azure: Microsoft-managed or customer-managed keys
- GCS: Google-managed or customer-managed keys

**Use lifecycle policies:**
```bash
# S3 lifecycle - delete after 30 days
aws s3api put-bucket-lifecycle-configuration \
  --bucket my-bucket \
  --lifecycle-configuration file://lifecycle.json
```

## Tips and Best Practices

### Organizing Cloud Exports

**Use meaningful prefixes:**
```
s3://my-bucket/
  ├── prod/
  │   ├── daily/2026-01-20/web-server-01/
  │   ├── daily/2026-01-19/web-server-01/
  │   └── weekly/2026-01-15/web-server-01/
  ├── dev/
  │   └── snapshots/web-server-dev/
  └── test/
      └── backups/test-vm-01/
```

**Include metadata:**
```bash
# Add tags for searchability
--prefix "backups/$(date +%Y-%m-%d)/$(hostname)"
```

### Cost Optimization

**Use appropriate storage classes:**
```
S3:
- Standard: Frequent access
- Infrequent Access: Monthly access
- Glacier: Long-term archive

Azure:
- Hot: Frequent access
- Cool: Infrequent access
- Archive: Long-term storage

GCS:
- Standard: Frequent access
- Nearline: Monthly access
- Coldline: Quarterly access
- Archive: Yearly access
```

**Enable compression:**
```bash
# Reduce storage costs by 30-70%
hyperexport --interactive --compress
```

**Cleanup old exports:**
```bash
# Delete exports older than 30 days
aws s3 ls s3://my-bucket/backups/ --recursive | \
  awk '$1 < "'$(date -d '30 days ago' +%Y-%m-%d)'" {print $4}' | \
  xargs -I {} aws s3 rm s3://my-bucket/{}
```

### Monitoring and Alerts

**Track upload costs:**
```bash
# AWS Cost Explorer API
aws ce get-cost-and-usage \
  --time-period Start=2026-01-01,End=2026-01-31 \
  --granularity MONTHLY \
  --metrics BlendedCost \
  --filter file://s3-filter.json
```

**Set up budget alerts:**
- AWS Budgets: Alert on S3 spend > $100/month
- Azure Cost Management: Budget alerts
- GCS Cloud Billing: Budget notifications

**Monitor upload success:**
```bash
# Check for failed uploads
grep "upload failed" /var/log/hyperexport.log
```

## Examples

### Daily Automated Backup to S3
```bash
#!/bin/bash
# daily-backup.sh

export AWS_ACCESS_KEY_ID="$(cat ~/.aws/access_key)"
export AWS_SECRET_ACCESS_KEY="$(cat ~/.aws/secret_key)"
export AWS_REGION="us-east-1"

DATE=$(date +%Y-%m-%d)
BUCKET="s3://my-backups/daily/$DATE"

hyperexport \
  --batch /etc/hyperexport/production-vms.txt \
  --format ova \
  --compress \
  --upload "$BUCKET" \
  --stream-upload \
  --parallel 4 \
  --quiet

# Cleanup old backups (keep last 7 days)
aws s3 ls s3://my-backups/daily/ | \
  awk '$1 < "'$(date -d '7 days ago' +%Y-%m-%d)'" {print $2}' | \
  xargs -I {} aws s3 rm s3://my-backups/daily/{} --recursive
```

### Multi-Cloud Backup
```bash
#!/bin/bash
# multi-cloud-backup.sh

VM_LIST="web-server-01 db-server-01"

for vm in $VM_LIST; do
  # Primary backup to S3
  hyperexport --vm "$vm" \
    --upload s3://primary-backups/prod \
    --compress

  # Secondary backup to Azure
  hyperexport --vm "$vm" \
    --upload azure://secondary-backups/prod \
    --compress

  # Tertiary backup to GCS
  hyperexport --vm "$vm" \
    --upload gs://tertiary-backups/prod \
    --compress
done
```

### Disaster Recovery Workflow
```bash
#!/bin/bash
# dr-backup.sh

# Export critical VMs to multiple regions

CRITICAL_VMS="db-master web-lb auth-server"
REGIONS="us-east-1 us-west-2 eu-west-1"

for vm in $CRITICAL_VMS; do
  for region in $REGIONS; do
    export AWS_REGION="$region"

    hyperexport --vm "$vm" \
      --format ova \
      --compress \
      --verify \
      --upload "s3://dr-backups-$region/critical" \
      --stream-upload
  done
done

# Send completion notification
echo "DR backup completed for $CRITICAL_VMS" | \
  mail -s "DR Backup Complete" ops@example.com
```

## Support

For issues, questions, or feature requests related to cloud storage integration:

- GitHub Issues: https://github.com/hypersdk/hypersdk/issues
- Documentation: https://hypersdk.dev/docs/cloud-storage
- Cloud Provider Support:
  - AWS: https://aws.amazon.com/support
  - Azure: https://azure.microsoft.com/support
  - GCP: https://cloud.google.com/support
