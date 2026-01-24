# HyperSDK Multi-Cloud Migration Guide

**Complete Guide to Multi-Cloud VM Exports**
Version 0.2.0 | Phase 4 Implementation

---

## 🌐 Overview

HyperSDK provides unified VM export capabilities across 5 major platforms:

| Provider | Format | Storage | Status |
|----------|--------|---------|--------|
| **VMware vSphere** | OVF, OVA, VMDK | Local, NFS, SMB | ✅ Production |
| **AWS EC2** | VMDK, VHD, RAW | S3 | ✅ Production |
| **Azure VMs** | VHD, Image | Blob Storage | ✅ Production |
| **GCP Compute** | VMDK, Image | Cloud Storage | ✅ Production |
| **Microsoft Hyper-V** | VHDX, VHD | Local, SMB | ✅ Production |

---

## 🎯 Use Cases

### 1. Cloud Migration
Migrate on-premises VMs to public cloud:
```
vSphere → AWS EC2
vSphere → Azure VMs
vSphere → GCP Compute Engine
Hyper-V → Azure
```

### 2. Cloud-to-Cloud Migration
Move workloads between cloud providers:
```
AWS → Azure
Azure → GCP
GCP → AWS
```

### 3. Disaster Recovery
Create off-site backups:
```
vSphere → S3 (AWS)
Hyper-V → Blob Storage (Azure)
AWS → Cloud Storage (GCP)
```

### 4. Multi-Cloud Strategy
Distribute workloads across providers for redundancy and cost optimization.

---

## 📋 Prerequisites

### General Requirements
- HyperSDK v0.2.0 or later
- Network access to source and destination platforms
- Sufficient storage space (VM size + 20% overhead)
- Administrative credentials for all platforms

### Platform-Specific Requirements

#### vSphere
- vCenter Server 6.5+
- User with export permissions
- Datastore read access

#### AWS
- AWS account with active subscription
- IAM user with VM Import/Export permissions
- S3 bucket in same region as target EC2

#### Azure
- Azure subscription
- Service principal with Contributor role
- Storage account with blob container
- Managed disk access

#### GCP
- GCP project with Compute Engine API enabled
- Service account with required roles
- Cloud Storage bucket

#### Hyper-V
- Windows Server 2016+ with Hyper-V role
- Administrator privileges
- PowerShell 5.0+
- WinRM enabled (for remote)

---

## 🔧 Configuration

### Complete Multi-Cloud Config

```yaml
# config.yaml

# ===== vSphere (Source Platform) =====
vcenter_url: "https://vcenter.example.com"
username: "admin@vsphere.local"
password: "vsphere-password"
insecure: false

# ===== AWS EC2 Export =====
aws:
  enabled: true
  region: "us-east-1"
  access_key: "AKIAIOSFODNN7EXAMPLE"
  secret_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  s3_bucket: "vm-exports"
  export_format: "vmdk"  # vmdk, vhd, raw

# ===== Azure VHD Export =====
azure:
  enabled: true
  subscription_id: "00000000-0000-0000-0000-000000000000"
  tenant_id: "00000000-0000-0000-0000-000000000000"
  client_id: "00000000-0000-0000-0000-000000000000"
  client_secret: "azure-secret-key"
  resource_group: "vm-exports-rg"
  location: "eastus"
  storage_account: "vmexportsstorage"
  container: "vhd-exports"
  container_url: "https://vmexportsstorage.blob.core.windows.net/vhd-exports"
  export_format: "vhd"

# ===== GCP Export =====
gcp:
  enabled: true
  project_id: "my-gcp-project-123"
  zone: "us-central1-a"
  region: "us-central1"
  credentials_json: "/path/to/gcp-service-account.json"
  gcs_bucket: "vm-exports"
  export_format: "vmdk"

# ===== Hyper-V Export =====
hyperv:
  enabled: true
  host: "hyperv-server.example.com"
  username: "Administrator"
  password: "hyperv-password"
  use_winrm: true
  winrm_port: 5985
  use_https: false
  export_format: "vhdx"

# ===== General Settings =====
daemon_addr: "0.0.0.0:8080"
log_level: "info"
download_workers: 3
```

---

## 🚀 Provider Setup Guides

### 1. AWS EC2 Setup

#### Step 1: Create IAM User

```bash
# Create IAM user
aws iam create-user --user-name vm-export-user

# Attach VM Import/Export policy
aws iam attach-user-policy \
  --user-name vm-export-user \
  --policy-arn arn:aws:iam::aws:policy/service-role/VMImportExportRoleForAWSConnector

# Create access keys
aws iam create-access-key --user-name vm-export-user
```

#### Step 2: Create S3 Bucket

```bash
# Create bucket
aws s3 mb s3://vm-exports --region us-east-1

# Enable versioning (optional)
aws s3api put-bucket-versioning \
  --bucket vm-exports \
  --versioning-configuration Status=Enabled
```

#### Step 3: Create Service Role

```bash
# Create role trust policy
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "Service": "vmie.amazonaws.com" },
    "Action": "sts:AssumeRole",
    "Condition": {
      "StringEquals": {
        "sts:Externalid": "vmimport"
      }
    }
  }]
}
EOF

# Create role
aws iam create-role \
  --role-name vmimport \
  --assume-role-policy-document file://trust-policy.json

# Attach policy
cat > role-policy.json <<EOF
{
  "Version":"2012-10-17",
  "Statement":[{
    "Effect":"Allow",
    "Action":[
      "s3:GetBucketLocation",
      "s3:GetObject",
      "s3:ListBucket",
      "s3:PutObject",
      "s3:GetBucketAcl"
    ],
    "Resource":[
      "arn:aws:s3:::vm-exports",
      "arn:aws:s3:::vm-exports/*"
    ]
  },{
    "Effect":"Allow",
    "Action":[
      "ec2:ModifySnapshotAttribute",
      "ec2:CopySnapshot",
      "ec2:RegisterImage",
      "ec2:Describe*"
    ],
    "Resource":"*"
  }]
}
EOF

aws iam put-role-policy \
  --role-name vmimport \
  --policy-name vmimport \
  --policy-document file://role-policy.json
```

#### Step 4: Test Connection

```bash
# Using HyperSDK API
curl -X POST http://localhost:8080/aws/test-connection \
  -H "Content-Type: application/json" \
  -d '{
    "region": "us-east-1",
    "access_key": "AKIAIOSFODNN7EXAMPLE",
    "secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  }'
```

---

### 2. Azure Setup

#### Step 1: Create Service Principal

```bash
# Login to Azure
az login

# Create service principal
az ad sp create-for-rbac \
  --name "hypersdk-export" \
  --role Contributor \
  --scopes /subscriptions/YOUR-SUBSCRIPTION-ID

# Output will contain:
# - appId (client_id)
# - password (client_secret)
# - tenant
```

#### Step 2: Create Resource Group

```bash
# Create resource group
az group create \
  --name vm-exports-rg \
  --location eastus
```

#### Step 3: Create Storage Account

```bash
# Create storage account
az storage account create \
  --name vmexportsstorage \
  --resource-group vm-exports-rg \
  --location eastus \
  --sku Standard_LRS

# Create blob container
az storage container create \
  --name vhd-exports \
  --account-name vmexportsstorage

# Get container URL
az storage container show-connection-string \
  --name vhd-exports \
  --account-name vmexportsstorage
```

#### Step 4: Assign Permissions

```bash
# Get service principal object ID
SP_ID=$(az ad sp list --display-name "hypersdk-export" --query "[0].id" -o tsv)

# Assign roles
az role assignment create \
  --assignee $SP_ID \
  --role "Disk Snapshot Contributor" \
  --scope /subscriptions/YOUR-SUBSCRIPTION-ID

az role assignment create \
  --assignee $SP_ID \
  --role "Storage Blob Data Contributor" \
  --scope /subscriptions/YOUR-SUBSCRIPTION-ID
```

#### Step 5: Test Connection

```bash
curl -X POST http://localhost:8080/azure/test-connection \
  -H "Content-Type: application/json" \
  -d '{
    "subscription_id": "YOUR-SUBSCRIPTION-ID",
    "tenant_id": "YOUR-TENANT-ID",
    "client_id": "YOUR-CLIENT-ID",
    "client_secret": "YOUR-CLIENT-SECRET"
  }'
```

---

### 3. GCP Setup

#### Step 1: Create Service Account

```bash
# Set project
gcloud config set project my-gcp-project-123

# Create service account
gcloud iam service-accounts create hypersdk-export \
  --display-name "HyperSDK Export Service Account"

# Get service account email
SA_EMAIL=$(gcloud iam service-accounts list \
  --filter="displayName:HyperSDK Export" \
  --format="value(email)")
```

#### Step 2: Assign IAM Roles

```bash
# Compute Image User
gcloud projects add-iam-policy-binding my-gcp-project-123 \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/compute.imageUser"

# Storage Object Admin
gcloud projects add-iam-policy-binding my-gcp-project-123 \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/storage.objectAdmin"

# Compute Instance Admin
gcloud projects add-iam-policy-binding my-gcp-project-123 \
  --member="serviceAccount:$SA_EMAIL" \
  --role="roles/compute.instanceAdmin.v1"
```

#### Step 3: Create Service Account Key

```bash
# Create JSON key
gcloud iam service-accounts keys create ~/gcp-service-account.json \
  --iam-account=$SA_EMAIL

# Secure the key file
chmod 600 ~/gcp-service-account.json
```

#### Step 4: Create Cloud Storage Bucket

```bash
# Create bucket
gsutil mb -l us-central1 gs://vm-exports

# Enable versioning
gsutil versioning set on gs://vm-exports

# Set lifecycle (optional - delete after 30 days)
cat > lifecycle.json <<EOF
{
  "lifecycle": {
    "rule": [{
      "action": {"type": "Delete"},
      "condition": {"age": 30}
    }]
  }
}
EOF

gsutil lifecycle set lifecycle.json gs://vm-exports
```

#### Step 5: Enable Required APIs

```bash
# Enable Compute Engine API
gcloud services enable compute.googleapis.com

# Enable Cloud Storage API
gcloud services enable storage-api.googleapis.com

# Enable Cloud Resource Manager API
gcloud services enable cloudresourcemanager.googleapis.com
```

#### Step 6: Test Connection

```bash
curl -X POST http://localhost:8080/gcp/test-connection \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "my-gcp-project-123",
    "credentials_json": "/path/to/gcp-service-account.json"
  }'
```

---

### 4. Hyper-V Setup

#### Local Hyper-V (Same Machine)

```yaml
hyperv:
  enabled: true
  host: ""  # Empty for local
  use_winrm: false
  export_format: "vhdx"
```

#### Remote Hyper-V via WinRM

**Step 1: Enable WinRM on Hyper-V Host**

```powershell
# On Hyper-V server
# Enable WinRM
Enable-PSRemoting -Force

# Configure firewall
New-NetFirewallRule -DisplayName "WinRM HTTP" `
  -Direction Inbound -LocalPort 5985 -Protocol TCP -Action Allow

# For HTTPS (recommended)
New-NetFirewallRule -DisplayName "WinRM HTTPS" `
  -Direction Inbound -LocalPort 5986 -Protocol TCP -Action Allow

# Create self-signed certificate (production: use real cert)
$cert = New-SelfSignedCertificate -DnsName "hyperv-server" `
  -CertStoreLocation Cert:\LocalMachine\My

# Create HTTPS listener
New-WSManInstance -ResourceURI winrm/config/Listener `
  -SelectorSet @{Address="*";Transport="HTTPS"} `
  -ValueSet @{Hostname="hyperv-server";CertificateThumbprint=$cert.Thumbprint}
```

**Step 2: Configure TrustedHosts (if not domain-joined)**

```powershell
# On HyperSDK host (Linux)
# Install winrm client
sudo apt-get install ruby-dev
sudo gem install -r winrm

# Test connection
winrm -u Administrator -p 'password' `
  -e http://hyperv-server:5985/wsman `
  "Get-VM | Select-Object Name,State"
```

**Step 3: HyperSDK Configuration**

```yaml
hyperv:
  enabled: true
  host: "hyperv-server.example.com"
  username: "Administrator"
  password: "secure-password"
  use_winrm: true
  winrm_port: 5985  # or 5986 for HTTPS
  use_https: false  # Set true for HTTPS
  export_format: "vhdx"
```

---

## 📖 Export Examples

### Example 1: vSphere to AWS

**Export VM from vSphere and upload to S3**

```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "name": "migrate-web-server-to-aws",
    "provider": "vsphere",
    "vm_path": "/Datacenter/vm/web-server-01",
    "output_dir": "/tmp/exports",
    "format": "vmdk",
    "metadata": {
      "destination": "aws",
      "aws_region": "us-east-1",
      "s3_bucket": "vm-exports"
    }
  }'
```

**Then import to AWS EC2:**

```bash
aws ec2 import-image \
  --description "Web Server 01" \
  --disk-containers Format=VMDK,UserBucket="{S3Bucket=vm-exports,S3Key=web-server-01.vmdk}"
```

---

### Example 2: vSphere to Azure

**Export and convert to VHD format**

```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "name": "migrate-app-server-to-azure",
    "provider": "vsphere",
    "vm_path": "/Datacenter/vm/app-server-02",
    "output_dir": "/exports",
    "format": "vhd",
    "metadata": {
      "destination": "azure",
      "container_url": "https://vmexportsstorage.blob.core.windows.net/vhd-exports"
    }
  }'
```

**Then create Azure VM:**

```bash
# Create managed disk from VHD
az disk create \
  --resource-group vm-exports-rg \
  --name app-server-02-disk \
  --source "https://vmexportsstorage.blob.core.windows.net/vhd-exports/app-server-02.vhd"

# Create VM from disk
az vm create \
  --resource-group vm-exports-rg \
  --name app-server-02 \
  --attach-os-disk app-server-02-disk \
  --os-type Linux
```

---

### Example 3: AWS to GCP

**Export AWS EC2 instance to GCP**

```bash
# Step 1: Export EC2 instance to S3
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "aws",
    "vm_path": "i-1234567890abcdef0",
    "output_dir": "/exports",
    "format": "vmdk",
    "metadata": {
      "s3_bucket": "vm-exports",
      "region": "us-east-1"
    }
  }'

# Step 2: Download and re-upload to GCS
# (HyperSDK can automate this)

# Step 3: Import to GCP
gcloud compute images create ec2-imported-image \
  --source-uri gs://vm-exports/i-1234567890abcdef0.vmdk \
  --family ubuntu-2004-lts
```

---

### Example 4: Hyper-V to Azure

**Native migration path**

```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "hyperv",
    "vm_path": "Production-Web-01",
    "output_dir": "/exports",
    "format": "vhd",
    "metadata": {
      "host": "hyperv-server.example.com",
      "use_winrm": true,
      "destination": "azure",
      "container_url": "https://storage.blob.core.windows.net/vhd-exports"
    }
  }'
```

---

### Example 5: Bulk Migration (Scheduled)

**Schedule nightly backups to multiple clouds**

```bash
curl -X POST http://localhost:8080/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "multi-cloud-backup",
    "name": "Multi-Cloud Nightly Backup",
    "schedule": "0 2 * * *",
    "enabled": true,
    "job_template": {
      "vm_path": "/Datacenter/vm/production-*",
      "output_dir": "/backups",
      "format": "ova",
      "compress": true,
      "metadata": {
        "destinations": ["aws", "azure", "gcp"],
        "retention_days": 30
      }
    },
    "tags": ["backup", "multi-cloud", "production"]
  }'
```

---

## 🔄 Migration Workflows

### Workflow 1: Complete vSphere → AWS Migration

```mermaid
graph LR
    A[vSphere VM] --> B[Export to OVA]
    B --> C[Convert to VMDK]
    C --> D[Upload to S3]
    D --> E[AWS Import Task]
    E --> F[EC2 Instance]
```

**Steps:**

1. **Export from vSphere**
```bash
curl -X POST http://localhost:8080/jobs/submit -d '{
  "vm_path": "/Datacenter/vm/my-vm",
  "output_dir": "/exports",
  "format": "vmdk"
}'
```

2. **Upload to S3** (automatic with metadata)
```json
{
  "metadata": {
    "aws_s3_bucket": "vm-exports",
    "aws_region": "us-east-1"
  }
}
```

3. **Import to EC2**
```bash
aws ec2 import-image \
  --description "My VM" \
  --disk-containers file://containers.json
```

4. **Launch Instance**
```bash
aws ec2 run-instances \
  --image-id ami-xxxxx \
  --instance-type t3.medium
```

---

### Workflow 2: Hyper-V → Azure Native

```mermaid
graph LR
    A[Hyper-V VM] --> B[Export VHDX]
    B --> C[Upload to Blob]
    C --> D[Create Managed Disk]
    D --> E[Azure VM]
```

**Single Command:**

```bash
curl -X POST http://localhost:8080/hyperv/migrate-to-azure \
  -d '{
    "vm_name": "Production-App",
    "azure_resource_group": "vm-exports-rg",
    "azure_location": "eastus",
    "vm_size": "Standard_D2s_v3"
  }'
```

---

## 🔍 Format Compatibility Matrix

| Source Format | → | AWS | Azure | GCP | Hyper-V | vSphere |
|--------------|---|-----|-------|-----|---------|---------|
| **VMDK** | | ✅ | ⚠️ | ✅ | ⚠️ | ✅ |
| **VHD** | | ✅ | ✅ | ⚠️ | ✅ | ⚠️ |
| **VHDX** | | ❌ | ⚠️ | ❌ | ✅ | ❌ |
| **OVA/OVF** | | ⚠️ | ❌ | ⚠️ | ❌ | ✅ |
| **RAW** | | ✅ | ⚠️ | ✅ | ❌ | ⚠️ |

Legend:
- ✅ Native support
- ⚠️ Conversion required
- ❌ Not supported

---

## 🛠️ Troubleshooting

### AWS Issues

**Error: "VM Import is not available in this region"**
```bash
# Check supported regions
aws ec2 describe-regions --output table

# Use supported region (e.g., us-east-1)
```

**Error: "Role not found: vmimport"**
```bash
# Create vmimport role (see AWS Setup above)
aws iam create-role --role-name vmimport --assume-role-policy-document file://trust-policy.json
```

---

### Azure Issues

**Error: "Service principal does not have permission"**
```bash
# Grant required roles
az role assignment create \
  --assignee YOUR-SP-ID \
  --role "Disk Snapshot Contributor"
```

**Error: "Storage account not found"**
```bash
# Verify storage account
az storage account show --name vmexportsstorage
```

---

### GCP Issues

**Error: "API not enabled"**
```bash
# Enable required APIs
gcloud services enable compute.googleapis.com
gcloud services enable storage-api.googleapis.com
```

**Error: "Insufficient permissions"**
```bash
# Check service account roles
gcloud projects get-iam-policy YOUR-PROJECT-ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:YOUR-SA-EMAIL"
```

---

### Hyper-V Issues

**Error: "WinRM connection failed"**
```powershell
# On Hyper-V server, check WinRM status
Get-WSManInstance -ResourceURI winrm/config/listener

# Test from Linux
curl http://hyperv-server:5985/wsman
```

**Error: "Export-VM failed: Access denied"**
```powershell
# Grant permissions
icacls "C:\ClusterStorage\Volume1" /grant "HyperSDK:F" /T
```

---

## 📊 Performance Optimization

### Network Transfer Optimization

```yaml
# Increase chunk size for faster transfers
chunk_size: 67108864  # 64MB

# More download workers
download_workers: 10

# Enable compression
compress: true
compression_level: 6
```

### Cloud-Specific Tuning

**AWS**
```yaml
aws:
  # Use accelerated transfer
  s3_transfer_acceleration: true
  # Multipart upload
  multipart_threshold: 8388608  # 8MB
  multipart_chunksize: 8388608
```

**Azure**
```yaml
azure:
  # Use premium storage for faster transfers
  storage_tier: "Premium_LRS"
  # Larger blocks
  block_size: 4194304  # 4MB
```

**GCP**
```yaml
gcp:
  # Use multi-regional buckets
  storage_class: "MULTI_REGIONAL"
  # Parallel composite uploads
  parallel_composite_upload_threshold: 157286400  # 150MB
```

---

## 💰 Cost Optimization

### Storage Costs

| Provider | Storage Type | Cost/GB/Month | Transfer Out |
|----------|-------------|---------------|--------------|
| AWS S3 | Standard | $0.023 | $0.09/GB |
| Azure Blob | Hot | $0.0184 | $0.087/GB |
| GCP Cloud Storage | Standard | $0.020 | $0.12/GB |

**Recommendations:**
- Use lifecycle policies to delete old exports
- Enable compression to reduce storage
- Use regional storage for lower costs

### Transfer Costs

**Free Tier Usage:**
- AWS: 100GB/month free outbound
- Azure: 5GB/month free outbound
- GCP: 200GB/month free (Compute to Storage)

**Cost Reduction Tips:**
1. Export during off-peak hours
2. Use same-region transfers when possible
3. Enable compression (reduces transfer by 30-50%)
4. Batch exports to minimize API calls

---

## 🔐 Security Best Practices

### 1. Credential Management

**Never hardcode credentials:**
```yaml
# ❌ Bad
aws:
  access_key: "AKIAIOSFODNN7EXAMPLE"
  secret_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# ✅ Good - use environment variables
aws:
  access_key: "${AWS_ACCESS_KEY}"
  secret_key: "${AWS_SECRET_KEY}"
```

### 2. Network Security

```yaml
# Enable TLS for all cloud connections
tls:
  enabled: true
  min_version: "1.2"

# Use VPC endpoints (AWS)
aws:
  use_vpc_endpoint: true

# Use private endpoints (Azure)
azure:
  use_private_endpoint: true
```

### 3. Encryption

```yaml
# Enable encryption at rest
aws:
  s3_encryption: "AES256"
  kms_key_id: "arn:aws:kms:us-east-1:123456789012:key/xxxxx"

azure:
  encryption_enabled: true

gcp:
  encryption_key: "projects/PROJECT/locations/LOCATION/keyRings/KEYRING/cryptoKeys/KEY"
```

---

## 📈 Monitoring & Alerting

### Prometheus Metrics

```yaml
# Example PromQL queries

# Export success rate by provider
rate(hypersdk_jobs_completed_total{provider="aws"}[5m]) /
rate(hypersdk_jobs_total{provider="aws"}[5m])

# Average export duration by provider
avg(hypersdk_job_duration_seconds) by (provider)

# Failed exports
increase(hypersdk_jobs_failed_total[1h])
```

### Grafana Alerts

```yaml
# Alert on high failure rate
- alert: HighExportFailureRate
  expr: rate(hypersdk_jobs_failed_total[5m]) > 0.1
  annotations:
    summary: "High export failure rate detected"

- alert: SlowExports
  expr: hypersdk_job_duration_seconds > 3600
  annotations:
    summary: "Export taking longer than 1 hour"
```

---

## 🎓 Advanced Scenarios

### Scenario 1: Multi-Region Replication

Replicate VMs across 3 cloud providers for maximum redundancy:

```bash
curl -X POST http://localhost:8080/jobs/submit \
  -d '{
    "name": "multi-region-replication",
    "vm_path": "/Datacenter/vm/critical-app",
    "destinations": [
      {
        "provider": "aws",
        "region": "us-east-1",
        "s3_bucket": "backups-us-east"
      },
      {
        "provider": "azure",
        "region": "eastus",
        "container": "backups-eastus"
      },
      {
        "provider": "gcp",
        "region": "us-central1",
        "bucket": "backups-us-central"
      }
    ],
    "format": "vmdk",
    "compress": true
  }'
```

### Scenario 2: Automated DR Failover

```yaml
# Schedule hourly sync to DR cloud
schedules:
  - id: dr-sync
    schedule: "0 * * * *"
    job_template:
      vm_path: "/Datacenter/vm/production-*"
      destinations: ["aws-dr-region"]
      sync_only_changes: true
```

---

## ✅ Migration Checklist

### Pre-Migration
- [ ] Source platform credentials configured
- [ ] Destination platform credentials configured
- [ ] Network connectivity verified
- [ ] Storage space available
- [ ] Backup of source VMs created
- [ ] Migration window scheduled

### Migration
- [ ] Test migration with non-critical VM
- [ ] Verify exported format compatibility
- [ ] Monitor export progress
- [ ] Validate checksum/integrity
- [ ] Test imported VM boots correctly

### Post-Migration
- [ ] Application testing on new platform
- [ ] Performance validation
- [ ] Update DNS/load balancers
- [ ] Cleanup temporary storage
- [ ] Document migration
- [ ] Update disaster recovery plan

---

## 📚 Additional Resources

### Documentation
- [AWS VM Import/Export](https://docs.aws.amazon.com/vm-import/)
- [Azure Migrate](https://docs.microsoft.com/azure/migrate/)
- [GCP Migrate for Compute Engine](https://cloud.google.com/migrate/compute-engine/docs)
- [Hyper-V Export](https://docs.microsoft.com/virtualization/hyper-v-on-windows/)

### Tools
- AWS CLI: https://aws.amazon.com/cli/
- Azure CLI: https://docs.microsoft.com/cli/azure/
- GCloud SDK: https://cloud.google.com/sdk
- PowerShell: https://docs.microsoft.com/powershell/

### Support
- HyperSDK Issues: https://github.com/your-org/hypersdk/issues
- Community Forum: https://forum.hypersdk.io
- Email: support@hypersdk.io

---

**Ready to migrate? Start with a test VM and scale to production!** 🚀
