
# OCI Registry Support for HyperSDK Helm Charts

This guide covers using OCI (Open Container Initiative) registries for Helm chart distribution.

## Overview

OCI registries provide a modern alternative to traditional Helm repositories:

**Benefits:**
- ✅ Use existing container registries (no separate infrastructure)
- ✅ Standard authentication (same as container images)
- ✅ Better security and access control
- ✅ Integrated with existing CI/CD pipelines
- ✅ Support for multi-tenancy
- ✅ Efficient storage and deduplication

**Supported OCI Registries:**
- GitHub Container Registry (ghcr.io)
- Docker Hub (registry-1.docker.io)
- AWS Elastic Container Registry (ECR)
- Azure Container Registry (ACR)
- Google Artifact Registry (GAR)
- Harbor
- JFrog Artifactory

## Quick Start

### Install from OCI Registry

```bash
# GitHub Container Registry
helm install my-hypersdk oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0

# With custom values
helm install my-hypersdk oci://ghcr.io/ssahani/charts/hypersdk \
  --version 0.2.0 \
  --namespace hypersdk \
  --create-namespace \
  --set credentials.vsphere.enabled=true
```

### Pull Chart for Inspection

```bash
# Pull chart locally
helm pull oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0

# Extract and inspect
tar -xzf hypersdk-0.2.0.tgz
cat hypersdk/values.yaml
```

## Publishing to OCI Registries

### GitHub Container Registry (ghcr.io)

#### Prerequisites

```bash
# Install Helm 3.8+
helm version

# Login to GitHub Container Registry
echo $GITHUB_TOKEN | helm registry login ghcr.io --username $GITHUB_USER --password-stdin
```

#### Publish Chart

```bash
# Using publish script
./deployments/scripts/publish-oci.sh \
  --registry ghcr.io \
  --namespace myuser/charts

# Or manually
helm package deployments/helm/hypersdk
helm push hypersdk-0.2.0.tgz oci://ghcr.io/myuser/charts
```

#### Automated Publishing (GitHub Actions)

The repository includes automated OCI publishing:

```bash
# Create and push a tag
git tag chart-v0.3.0
git push origin chart-v0.3.0

# GitHub Actions will:
# 1. Test the chart
# 2. Package the chart
# 3. Push to ghcr.io/ssahani/charts/hypersdk
# 4. Create GitHub Release
```

### Docker Hub

```bash
# Login to Docker Hub
echo $DOCKER_PASSWORD | helm registry login registry-1.docker.io --username $DOCKER_USERNAME --password-stdin

# Publish chart
./deployments/scripts/publish-oci.sh \
  --registry registry-1.docker.io \
  --namespace myuser \
  --username $DOCKER_USERNAME \
  --password $DOCKER_PASSWORD

# Install from Docker Hub
helm install my-hypersdk oci://registry-1.docker.io/myuser/hypersdk --version 0.2.0
```

### AWS Elastic Container Registry (ECR)

```bash
# Login to ECR (using AWS CLI)
aws ecr get-login-password --region us-east-1 | \
  helm registry login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com

# Publish chart
./deployments/scripts/publish-oci.sh \
  --registry 123456789.dkr.ecr.us-east-1.amazonaws.com \
  --namespace charts \
  --skip-login  # Already logged in

# Install from ECR
helm install my-hypersdk oci://123456789.dkr.ecr.us-east-1.amazonaws.com/charts/hypersdk --version 0.2.0
```

### Azure Container Registry (ACR)

```bash
# Login to ACR (using Azure CLI)
az acr login --name myregistry

# Or use service principal
echo $SP_PASSWORD | helm registry login myregistry.azurecr.io --username $SP_APP_ID --password-stdin

# Publish chart
./deployments/scripts/publish-oci.sh \
  --registry myregistry.azurecr.io \
  --namespace charts \
  --username $SP_APP_ID \
  --password $SP_PASSWORD

# Install from ACR
helm install my-hypersdk oci://myregistry.azurecr.io/charts/hypersdk --version 0.2.0
```

### Google Artifact Registry (GAR)

```bash
# Login to GAR (using gcloud)
gcloud auth print-access-token | \
  helm registry login us-docker.pkg.dev --username oauth2accesstoken --password-stdin

# Publish chart
./deployments/scripts/publish-oci.sh \
  --registry us-docker.pkg.dev \
  --namespace myproject/charts \
  --skip-login

# Install from GAR
helm install my-hypersdk oci://us-docker.pkg.dev/myproject/charts/hypersdk --version 0.2.0
```

### Harbor

```bash
# Login to Harbor
echo $HARBOR_PASSWORD | helm registry login harbor.example.com --username $HARBOR_USERNAME --password-stdin

# Publish chart
./deployments/scripts/publish-oci.sh \
  --registry harbor.example.com \
  --namespace library \
  --username $HARBOR_USERNAME \
  --password $HARBOR_PASSWORD

# Install from Harbor
helm install my-hypersdk oci://harbor.example.com/library/hypersdk --version 0.2.0
```

## OCI vs Traditional Helm Repository

### Comparison

| Feature | OCI Registry | Traditional Helm Repo |
|---------|-------------|----------------------|
| **Infrastructure** | Use existing container registry | Requires web server (GitHub Pages, S3, etc.) |
| **Authentication** | Standard container auth | HTTP basic auth or none |
| **Access Control** | Fine-grained RBAC | Limited |
| **Storage** | Efficient deduplication | Full file storage |
| **Versioning** | Built-in tagging | index.yaml file |
| **Multi-tenancy** | Native support | Limited |
| **Integration** | Works with CI/CD for containers | Separate pipeline |
| **Maturity** | Newer (Helm 3.8+) | Established |

### When to Use OCI

✅ **Use OCI when:**
- You already have an OCI registry
- You need fine-grained access control
- You want unified authentication for charts and images
- You're building in cloud environments (AWS/Azure/GCP)
- You need multi-tenant chart distribution

✅ **Use Traditional Helm Repo when:**
- You want public chart distribution
- You need maximum compatibility
- You prefer simple HTTP hosting (GitHub Pages)
- You have many charts to index together

### Hybrid Approach

You can use **both** methods:

```bash
# Publish to both OCI and traditional repo
./deployments/scripts/package-helm-chart.sh --publish  # GitHub Pages
./deployments/scripts/publish-oci.sh                   # ghcr.io

# Users can choose:
helm install from-repo hypersdk/hypersdk  # Traditional
helm install from-oci oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0  # OCI
```

## GitOps with OCI

### ArgoCD with OCI

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk
  namespace: argocd
spec:
  project: default

  source:
    # Use OCI registry
    repoURL: ghcr.io/ssahani/charts
    chart: hypersdk
    targetRevision: 0.2.0

    helm:
      values: |
        replicaCount: 3
        # ... other values

  destination:
    server: https://kubernetes.default.svc
    namespace: hypersdk

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Flux with OCI

```yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  type: oci
  interval: 5m
  url: oci://ghcr.io/ssahani/charts

---
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  interval: 10m
  chart:
    spec:
      chart: hypersdk
      version: '0.2.0'
      sourceRef:
        kind: HelmRepository
        name: hypersdk
        namespace: flux-system

  values:
    replicaCount: 3
```

## Private OCI Registries

### Using Image Pull Secrets

For private OCI registries, create an image pull secret:

```bash
# Create secret for registry authentication
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=$GITHUB_USER \
  --docker-password=$GITHUB_TOKEN \
  --namespace hypersdk

# Helm will use this secret automatically if configured
```

### Using Service Accounts

For cloud provider registries:

**AWS ECR:**
```yaml
# Use IAM role
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789:role/ecr-reader
```

**Azure ACR:**
```yaml
# Use managed identity
serviceAccount:
  labels:
    aadpodidbinding: acr-reader
```

**GCP GAR:**
```yaml
# Use Workload Identity
serviceAccount:
  annotations:
    iam.gke.io/gcp-service-account: chart-reader@project.iam.gserviceaccount.com
```

## Best Practices

### 1. Version Tagging

```bash
# Use semantic versioning
helm push hypersdk-0.2.0.tgz oci://ghcr.io/myuser/charts

# Chart is available at:
# oci://ghcr.io/myuser/charts/hypersdk:0.2.0
```

### 2. Access Control

```yaml
# GitHub Container Registry - set package visibility
# Repository → Packages → hypersdk → Package settings → Visibility

# For private charts, configure access:
# Repository → Packages → hypersdk → Package settings → Manage access
```

### 3. Retention Policies

```bash
# Configure retention in your OCI registry
# Example for GitHub: Keep last 10 versions
# Repository → Packages → hypersdk → Package settings → Retention
```

### 4. Scanning and Security

```yaml
# Enable vulnerability scanning in registry
# Most OCI registries provide built-in scanning

# GitHub Container Registry:
# - Automatic vulnerability scanning
# - Dependency graph
# - Security alerts
```

### 5. Multi-Arch Charts

```bash
# Charts are platform-independent, but document requirements
# In Chart.yaml:
annotations:
  "artifacthub.io/platforms": "linux/amd64,linux/arm64"
```

## Troubleshooting

### Authentication Errors

```bash
# Error: failed to authorize: failed to fetch anonymous token
# Solution: Login to the registry

helm registry login ghcr.io --username $USER --password-stdin
```

### Chart Not Found

```bash
# Error: chart "hypersdk" version "0.2.0" not found
# Solution: Verify chart exists and you have access

# Pull to verify
helm pull oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0
```

### Version Issues

```bash
# Error: Helm version 3.7 doesn't support OCI
# Solution: Upgrade to Helm 3.8+

# Check version
helm version

# Upgrade Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

## Comparison: Installation Methods

### Method 1: Traditional Helm Repository

```bash
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm repo update
helm install my-hypersdk hypersdk/hypersdk
```

**Pros:**
- Simple public distribution
- No authentication needed
- Works with older Helm versions

### Method 2: OCI Registry

```bash
helm install my-hypersdk oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0
```

**Pros:**
- Integrated with container infrastructure
- Better security and access control
- Standard authentication

### Method 3: Local Chart

```bash
git clone https://github.com/ssahani/hypersdk.git
helm install my-hypersdk ./hypersdk/deployments/helm/hypersdk
```

**Pros:**
- Full control and customization
- Development and testing
- No network dependency

## Summary

OCI registries provide a modern, secure, and integrated way to distribute Helm charts:

✅ **Use OCI for:**
- Private chart distribution
- Enterprise environments
- Integrated CI/CD pipelines
- Cloud-native deployments

✅ **Use Traditional Helm Repo for:**
- Public chart distribution
- Simple hosting needs
- Maximum compatibility

HyperSDK supports both methods - choose what works best for your use case!

**Quick Reference:**

```bash
# OCI (GitHub Container Registry)
helm install hypersdk oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0

# Traditional (GitHub Pages)
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm install hypersdk hypersdk/hypersdk
```
