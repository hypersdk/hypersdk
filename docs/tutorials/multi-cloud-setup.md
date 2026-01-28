# Multi-Cloud Setup Guide

This guide shows how to configure HyperSDK to work with multiple cloud providers simultaneously.

## Overview

HyperSDK supports 9 cloud providers:
1. **VMware vSphere** - On-premises and vCloud
2. **AWS** - Amazon Web Services
3. **Azure** - Microsoft Azure
4. **GCP** - Google Cloud Platform
5. **Hyper-V** - Microsoft Hyper-V
6. **OCI** - Oracle Cloud Infrastructure
7. **OpenStack** - Private cloud platform
8. **Alibaba Cloud** - Alibaba Cloud
9. **Proxmox** - Proxmox Virtual Environment

You can configure one or more providers based on your needs.

## Configuration Approaches

### Approach 1: Environment Variables (Docker/Podman)

Configure multiple providers in your `.env` file:

```bash
# vSphere (Primary)
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=administrator@vsphere.local
GOVC_PASSWORD=vsphere-password
GOVC_INSECURE=1

# AWS
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
AWS_REGION=us-east-1

# Azure
AZURE_SUBSCRIPTION_ID=00000000-0000-0000-0000-000000000000
AZURE_TENANT_ID=00000000-0000-0000-0000-000000000000
AZURE_CLIENT_ID=00000000-0000-0000-0000-000000000000
AZURE_CLIENT_SECRET=azure-client-secret

# GCP
GOOGLE_APPLICATION_CREDENTIALS=/config/gcp-service-account.json
GCP_PROJECT_ID=my-gcp-project

# OpenStack
OS_AUTH_URL=https://openstack.example.com:5000/v3
OS_USERNAME=admin
OS_PASSWORD=openstack-password
OS_PROJECT_NAME=admin
OS_USER_DOMAIN_NAME=Default
OS_PROJECT_DOMAIN_NAME=Default
```

Start with Docker Compose:

```bash
cd deployments/docker
docker compose up -d
```

### Approach 2: Kubernetes Secrets

Create separate secrets for each provider:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: vsphere-credentials
  namespace: hypersdk
type: Opaque
stringData:
  url: "https://vcenter.example.com/sdk"
  username: "administrator@vsphere.local"
  password: "vsphere-password"
  insecure: "1"

---
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: hypersdk
type: Opaque
stringData:
  access-key-id: "AKIAIOSFODNN7EXAMPLE"
  secret-access-key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  region: "us-east-1"

---
apiVersion: v1
kind: Secret
metadata:
  name: azure-credentials
  namespace: hypersdk
type: Opaque
stringData:
  subscription-id: "00000000-0000-0000-0000-000000000000"
  tenant-id: "00000000-0000-0000-0000-000000000000"
  client-id: "00000000-0000-0000-0000-000000000000"
  client-secret: "azure-client-secret"

---
apiVersion: v1
kind: Secret
metadata:
  name: gcp-credentials
  namespace: hypersdk
type: Opaque
stringData:
  service-account.json: |
    {
      "type": "service_account",
      "project_id": "my-gcp-project",
      "private_key_id": "key-id",
      "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
      "client_email": "service-account@my-gcp-project.iam.gserviceaccount.com",
      "client_id": "123456789",
      "auth_uri": "https://accounts.google.com/o/oauth2/auth",
      "token_uri": "https://oauth2.googleapis.com/token"
    }
  project-id: "my-gcp-project"
```

Reference in deployment:

```yaml
env:
  # vSphere
  - name: GOVC_URL
    valueFrom:
      secretKeyRef:
        name: vsphere-credentials
        key: url
  - name: GOVC_USERNAME
    valueFrom:
      secretKeyRef:
        name: vsphere-credentials
        key: username
  - name: GOVC_PASSWORD
    valueFrom:
      secretKeyRef:
        name: vsphere-credentials
        key: password

  # AWS
  - name: AWS_ACCESS_KEY_ID
    valueFrom:
      secretKeyRef:
        name: aws-credentials
        key: access-key-id
  - name: AWS_SECRET_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        name: aws-credentials
        key: secret-access-key
  - name: AWS_REGION
    valueFrom:
      secretKeyRef:
        name: aws-credentials
        key: region

  # Azure
  - name: AZURE_SUBSCRIPTION_ID
    valueFrom:
      secretKeyRef:
        name: azure-credentials
        key: subscription-id
  # ... (similar for other Azure credentials)

  # GCP
  - name: GOOGLE_APPLICATION_CREDENTIALS
    value: /config/gcp/service-account.json
  - name: GCP_PROJECT_ID
    valueFrom:
      secretKeyRef:
        name: gcp-credentials
        key: project-id

volumeMounts:
  - name: gcp-credentials
    mountPath: /config/gcp
    readOnly: true

volumes:
  - name: gcp-credentials
    secret:
      secretName: gcp-credentials
      items:
        - key: service-account.json
          path: service-account.json
```

## Provider-Specific Setup

### 1. VMware vSphere

**Requirements:**
- vCenter Server 6.7+ or ESXi 6.7+
- User account with VM export permissions

**Configuration:**

```bash
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=administrator@vsphere.local
GOVC_PASSWORD=your-password
GOVC_INSECURE=1                    # For self-signed certificates
GOVC_DATACENTER=Datacenter1        # Optional: default datacenter
GOVC_DATASTORE=datastore1          # Optional: default datastore
```

**Verification:**

```bash
# Test connection
docker exec hypersdk govc about

# List VMs
docker exec hypersdk govc ls /*/vm
```

### 2. Amazon Web Services (AWS)

**Requirements:**
- AWS account with EC2 permissions
- IAM user with programmatic access

**Create IAM User:**

```bash
aws iam create-user --user-name hypersdk-exporter

aws iam attach-user-policy --user-name hypersdk-exporter \
  --policy-arn arn:aws:iam::aws:policy/AmazonEC2ReadOnlyAccess

aws iam create-access-key --user-name hypersdk-exporter
```

**Configuration:**

```bash
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
AWS_REGION=us-east-1
AWS_DEFAULT_REGION=us-east-1      # Alternative to AWS_REGION
```

**Verification:**

```bash
# Test AWS connectivity
docker exec hypersdk aws ec2 describe-instances --region us-east-1
```

### 3. Microsoft Azure

**Requirements:**
- Azure subscription
- Service Principal with contributor access

**Create Service Principal:**

```bash
az login

az ad sp create-for-rbac --name hypersdk-exporter \
  --role Contributor \
  --scopes /subscriptions/<subscription-id>
```

Output:
```json
{
  "appId": "00000000-0000-0000-0000-000000000000",
  "displayName": "hypersdk-exporter",
  "password": "your-client-secret",
  "tenant": "00000000-0000-0000-0000-000000000000"
}
```

**Configuration:**

```bash
AZURE_SUBSCRIPTION_ID=<subscription-id>
AZURE_TENANT_ID=<tenant-from-output>
AZURE_CLIENT_ID=<appId-from-output>
AZURE_CLIENT_SECRET=<password-from-output>
AZURE_RESOURCE_GROUP=my-resource-group
```

**Verification:**

```bash
# Test Azure connectivity
docker exec hypersdk az vm list --resource-group my-resource-group
```

### 4. Google Cloud Platform (GCP)

**Requirements:**
- GCP project with Compute Engine API enabled
- Service account with Compute Viewer role

**Create Service Account:**

```bash
gcloud iam service-accounts create hypersdk-exporter \
  --display-name "HyperSDK Exporter"

gcloud projects add-iam-policy-binding my-project-id \
  --member serviceAccount:hypersdk-exporter@my-project-id.iam.gserviceaccount.com \
  --role roles/compute.viewer

gcloud iam service-accounts keys create gcp-service-account.json \
  --iam-account hypersdk-exporter@my-project-id.iam.gserviceaccount.com
```

**Configuration (Docker):**

```bash
# Copy service account JSON to config directory
cp gcp-service-account.json deployments/docker/config/

# Update .env
GOOGLE_APPLICATION_CREDENTIALS=/config/gcp-service-account.json
GOOGLE_CREDENTIALS_FILE=./config/gcp-service-account.json
GCP_PROJECT_ID=my-project-id
```

**Update docker-compose.yml:**

```yaml
volumes:
  - ./config/gcp-service-account.json:/config/gcp-service-account.json:ro
```

**Verification:**

```bash
# Test GCP connectivity
docker exec hypersdk gcloud compute instances list --project my-project-id
```

### 5. Microsoft Hyper-V

**Requirements:**
- Hyper-V Server or Windows Server with Hyper-V role
- WinRM enabled on the Hyper-V host
- User with Hyper-V administrator permissions

**Enable WinRM (on Hyper-V host):**

```powershell
Enable-PSRemoting -Force
Set-Item WSMan:\localhost\Client\TrustedHosts * -Force
winrm set winrm/config/service/auth '@{Basic="true"}'
```

**Configuration:**

```bash
HYPERV_HOST=hyperv-host.example.com
HYPERV_USERNAME=Administrator
HYPERV_PASSWORD=your-password
HYPERV_PORT=5985                   # 5985 for HTTP, 5986 for HTTPS
HYPERV_USE_SSL=false
```

### 6. Oracle Cloud Infrastructure (OCI)

**Requirements:**
- OCI tenancy
- API signing key

**Create API Key:**

```bash
mkdir ~/.oci
openssl genrsa -out ~/.oci/oci-api-key.pem 2048
openssl rsa -pubout -in ~/.oci/oci-api-key.pem -out ~/.oci/oci-api-key-public.pem
```

Upload public key to OCI Console: User Settings → API Keys

**Configuration:**

```bash
OCI_TENANCY_OCID=ocid1.tenancy.oc1..exampleuniqueid
OCI_USER_OCID=ocid1.user.oc1..exampleuniqueid
OCI_FINGERPRINT=aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99
OCI_KEY_FILE=/config/oci-api-key.pem
OCI_REGION=us-ashburn-1
```

**Mount key in Docker:**

```yaml
volumes:
  - ~/.oci/oci-api-key.pem:/config/oci-api-key.pem:ro
```

### 7. OpenStack

**Requirements:**
- OpenStack cloud (public or private)
- User credentials with instance read access

**Configuration:**

```bash
OS_AUTH_URL=https://openstack.example.com:5000/v3
OS_USERNAME=admin
OS_PASSWORD=your-password
OS_PROJECT_NAME=admin
OS_USER_DOMAIN_NAME=Default
OS_PROJECT_DOMAIN_NAME=Default
OS_REGION_NAME=RegionOne
OS_IDENTITY_API_VERSION=3
```

**Verification:**

```bash
docker exec hypersdk openstack server list
```

### 8. Alibaba Cloud

**Requirements:**
- Alibaba Cloud account
- RAM user with ECS read permissions

**Create RAM User (in Alibaba Console):**
1. Go to RAM Console
2. Create user with programmatic access
3. Attach `AliyunECSReadOnlyAccess` policy
4. Save AccessKey ID and Secret

**Configuration:**

```bash
ALIBABA_CLOUD_ACCESS_KEY_ID=LTAI4G...
ALIBABA_CLOUD_ACCESS_KEY_SECRET=your-secret
ALIBABA_CLOUD_REGION_ID=cn-hangzhou
```

### 9. Proxmox VE

**Requirements:**
- Proxmox VE 6.0+
- User with VM.Audit privileges

**Create API Token (in Proxmox UI):**
1. Datacenter → Permissions → API Tokens
2. Add token for user
3. Save Token ID and Secret

**Configuration:**

```bash
PROXMOX_URL=https://proxmox.example.com:8006/api2/json
PROXMOX_USERNAME=root@pam
PROXMOX_PASSWORD=your-password
# Or use API token
PROXMOX_TOKEN_ID=root@pam!token-id
PROXMOX_TOKEN_SECRET=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
PROXMOX_NODE=pve
PROXMOX_INSECURE=1
```

## Complete Multi-Cloud Example

### Docker Compose with All Providers

Create `deployments/docker/.env`:

```bash
# vSphere
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=administrator@vsphere.local
GOVC_PASSWORD=vsphere-pass
GOVC_INSECURE=1

# AWS
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=secret...
AWS_REGION=us-east-1

# Azure
AZURE_SUBSCRIPTION_ID=00000000-0000-0000-0000-000000000000
AZURE_TENANT_ID=00000000-0000-0000-0000-000000000000
AZURE_CLIENT_ID=00000000-0000-0000-0000-000000000000
AZURE_CLIENT_SECRET=azure-secret

# GCP
GOOGLE_APPLICATION_CREDENTIALS=/config/gcp-sa.json
GOOGLE_CREDENTIALS_FILE=./config/gcp-sa.json
GCP_PROJECT_ID=my-gcp-project

# Hyper-V
HYPERV_HOST=hyperv.example.com
HYPERV_USERNAME=Administrator
HYPERV_PASSWORD=hyperv-pass

# OCI
OCI_TENANCY_OCID=ocid1.tenancy.oc1..example
OCI_USER_OCID=ocid1.user.oc1..example
OCI_FINGERPRINT=aa:bb:cc:...
OCI_KEY_FILE=/config/oci-key.pem
OCI_REGION=us-ashburn-1

# OpenStack
OS_AUTH_URL=https://openstack.example.com:5000/v3
OS_USERNAME=admin
OS_PASSWORD=openstack-pass
OS_PROJECT_NAME=admin

# Alibaba
ALIBABA_CLOUD_ACCESS_KEY_ID=LTAI...
ALIBABA_CLOUD_ACCESS_KEY_SECRET=ali-secret
ALIBABA_CLOUD_REGION_ID=cn-hangzhou

# Proxmox
PROXMOX_URL=https://proxmox.example.com:8006/api2/json
PROXMOX_USERNAME=root@pam
PROXMOX_PASSWORD=proxmox-pass
```

Start the stack:

```bash
cd deployments/docker
docker compose up -d
```

### Kubernetes with All Providers

Create `deployments/kubernetes/overlays/production/secrets.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: multi-cloud-credentials
  namespace: hypersdk
type: Opaque
stringData:
  # vSphere
  vsphere-url: "https://vcenter.example.com/sdk"
  vsphere-username: "administrator@vsphere.local"
  vsphere-password: "vsphere-pass"

  # AWS
  aws-access-key-id: "AKIA..."
  aws-secret-access-key: "secret..."
  aws-region: "us-east-1"

  # Azure
  azure-subscription-id: "00000000-0000-0000-0000-000000000000"
  azure-tenant-id: "00000000-0000-0000-0000-000000000000"
  azure-client-id: "00000000-0000-0000-0000-000000000000"
  azure-client-secret: "azure-secret"

  # GCP
  gcp-project-id: "my-gcp-project"
  gcp-service-account.json: |
    {
      "type": "service_account",
      ...
    }

  # OpenStack
  os-auth-url: "https://openstack.example.com:5000/v3"
  os-username: "admin"
  os-password: "openstack-pass"
  os-project-name: "admin"
```

Deploy:

```bash
./deployments/scripts/deploy-k8s.sh production
```

## Testing Multi-Cloud Setup

### Verify Provider Connectivity

```bash
# Check daemon status
curl http://localhost:8080/status

# Should show all configured providers
{
  "status": "healthy",
  "providers": {
    "vsphere": "connected",
    "aws": "connected",
    "azure": "connected",
    "gcp": "connected",
    "hyperv": "connected",
    "oci": "connected",
    "openstack": "connected",
    "alibaba": "connected",
    "proxmox": "connected"
  }
}
```

### List VMs from All Providers

```bash
# List VMs from all providers
curl http://localhost:8080/vms/list

# List from specific provider
curl http://localhost:8080/vms/list?provider=vsphere
curl http://localhost:8080/vms/list?provider=aws
curl http://localhost:8080/vms/list?provider=azure
```

### Export from Different Providers

```bash
# Export from vSphere
curl -X POST http://localhost:8080/jobs/submit \
  -d '{"provider":"vsphere","vm":"/Datacenter/vm/test","output":"/exports/vsphere-vm"}'

# Export from AWS
curl -X POST http://localhost:8080/jobs/submit \
  -d '{"provider":"aws","instance_id":"i-1234567890abcdef0","output":"/exports/aws-instance"}'

# Export from Azure
curl -X POST http://localhost:8080/jobs/submit \
  -d '{"provider":"azure","vm_name":"my-azure-vm","resource_group":"my-rg","output":"/exports/azure-vm"}'
```

## Security Best Practices

### 1. Use Secret Management

**Kubernetes**: Use External Secrets Operator or Sealed Secrets

```bash
# Install External Secrets Operator
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace

# Create SecretStore pointing to AWS Secrets Manager, Azure Key Vault, or GCP Secret Manager
```

**Docker**: Use Docker secrets (Swarm mode)

```bash
echo "vsphere-password" | docker secret create vsphere_password -
```

### 2. Rotate Credentials Regularly

```bash
# Rotate AWS keys
aws iam create-access-key --user-name hypersdk-exporter
# Update secret, delete old key
aws iam delete-access-key --user-name hypersdk-exporter --access-key-id OLD_KEY

# Rotate Azure service principal secret
az ad sp credential reset --id <client-id>
```

### 3. Use Least Privilege

Each provider should have minimum required permissions:
- vSphere: VM read-only permissions
- AWS: EC2 DescribeInstances, DescribeVolumes
- Azure: Reader role on resource group
- GCP: Compute Viewer role

### 4. Encrypt Secrets at Rest

Kubernetes: Enable encryption at rest

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    - aescbc:
        keys:
        - name: key1
          secret: <base64-encoded-secret>
```

## Troubleshooting

### Provider Connection Failures

```bash
# Check logs
docker logs -f hypersdk

# Common issues:
# - Incorrect credentials
# - Network connectivity
# - Firewall blocking access
# - Certificate validation failures
```

### Test Individual Providers

```bash
# vSphere
docker exec hypersdk govc about

# AWS
docker exec hypersdk aws sts get-caller-identity

# Azure
docker exec hypersdk az account show

# GCP
docker exec hypersdk gcloud auth list
```

## Next Steps

- [Monitoring Guide](monitoring.md) - Monitor multi-cloud exports
- [Production Deployment](production-deployment.md) - Deploy at scale
- [API Reference](../api/README.md) - API endpoints for each provider
