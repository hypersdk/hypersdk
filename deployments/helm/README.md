# HyperSDK Helm Charts

Complete Helm chart infrastructure for deploying HyperSDK to Kubernetes.

## Overview

This directory contains the official Helm chart for HyperSDK, along with comprehensive tooling for packaging, publishing, deploying, and upgrading.

## Quick Start

### Install from Helm Repository

```bash
# Add repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm repo update

# Install HyperSDK
helm install my-hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --create-namespace
```

### Quick Deployment Script

```bash
# Deploy to k3d local cluster
./deployments/scripts/deploy-helm.sh k3d --create-namespace --wait

# Deploy to GKE
./deployments/scripts/deploy-helm.sh gke --from-repo --create-namespace

# Deploy to minikube
./deployments/scripts/deploy-helm.sh minikube --create-namespace
```

## Directory Structure

```
deployments/helm/
├── hypersdk/                          # Main Helm chart
│   ├── Chart.yaml                     # Chart metadata
│   ├── values.yaml                    # Default values
│   ├── templates/                     # Kubernetes templates
│   │   ├── deployment.yaml           # Main deployment
│   │   ├── service.yaml              # Services
│   │   ├── configmap.yaml            # Configuration
│   │   ├── secrets.yaml              # Credentials
│   │   ├── pvc.yaml                  # Storage
│   │   ├── hpa.yaml                  # Auto-scaling
│   │   ├── ingress.yaml              # Ingress
│   │   ├── networkpolicy.yaml        # Network security
│   │   ├── servicemonitor.yaml       # Prometheus monitoring
│   │   ├── pdb.yaml                  # Pod disruption budget
│   │   ├── rbac.yaml                 # RBAC resources
│   │   ├── serviceaccount.yaml       # Service account
│   │   ├── NOTES.txt                 # Post-install notes
│   │   └── _helpers.tpl              # Template helpers
│   ├── examples/                      # Example configurations
│   │   ├── k3d-values.yaml           # k3d local dev
│   │   ├── kind-values.yaml          # KIND testing
│   │   ├── minikube-values.yaml      # Minikube local
│   │   ├── gke-values.yaml           # Google Kubernetes Engine
│   │   ├── eks-values.yaml           # Amazon EKS
│   │   └── aks-values.yaml           # Azure AKS
│   ├── .helmignore                    # Helm ignore file
│   └── README.md                      # Chart documentation
├── packages/                          # Packaged charts
│   ├── hypersdk-0.2.0.tgz            # Chart package
│   └── index.yaml                     # Repository index
├── PUBLISHING.md                      # Publishing guide
├── UPGRADE.md                         # Upgrade guide
├── HELM-REPOSITORY-SETUP.md           # Repository setup
├── TEST-RESULTS.md                    # Test results
└── README.md                          # This file
```

## Documentation

### For Users

- **[Chart README](hypersdk/README.md)** - Complete chart documentation
  - Installation instructions
  - Configuration parameters
  - Usage examples
  - Cloud provider integrations

- **[Upgrade Guide](UPGRADE.md)** - Upgrading existing deployments
  - Upgrade strategies
  - Pre-upgrade checklist
  - Rollback procedures
  - Version-specific notes

### For Maintainers

- **[Publishing Guide](PUBLISHING.md)** - Chart publishing process
  - Manual publishing
  - Automated releases
  - Versioning guidelines
  - Security (chart signing)

- **[Repository Setup](HELM-REPOSITORY-SETUP.md)** - GitHub Pages setup
  - Enabling GitHub Pages
  - Repository structure
  - Troubleshooting

- **[Test Results](TEST-RESULTS.md)** - Comprehensive test results
  - 14 automated tests
  - Cloud provider validation
  - Performance metrics

## Deployment Methods

### 1. Quick Deployment Script (Recommended)

Use the deployment script for one-command installation:

```bash
# Local development (k3d)
../scripts/deploy-helm.sh k3d --create-namespace --wait

# Local testing (KIND)
../scripts/deploy-helm.sh kind --create-namespace

# Minikube
../scripts/deploy-helm.sh minikube --create-namespace

# Google Kubernetes Engine
../scripts/deploy-helm.sh gke --from-repo --create-namespace

# Amazon EKS
../scripts/deploy-helm.sh eks --from-repo --create-namespace

# Azure AKS
../scripts/deploy-helm.sh aks --from-repo --create-namespace

# Production (custom values)
../scripts/deploy-helm.sh custom --values prod-values.yaml
```

### 2. Helm Repository

Install from the public Helm repository:

```bash
# Add repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts

# Install
helm install hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --create-namespace

# Install specific version
helm install hypersdk hypersdk/hypersdk \
  --version 0.2.0 \
  --namespace hypersdk \
  --create-namespace
```

### 3. Local Chart

Install from local chart directory:

```bash
# Install with default values
helm install hypersdk ./hypersdk \
  --namespace hypersdk \
  --create-namespace

# Install with environment-specific values
helm install hypersdk ./hypersdk \
  --values ./hypersdk/examples/gke-values.yaml \
  --namespace hypersdk \
  --create-namespace
```

## Supported Environments

### Local Development

- **k3d** - k3s in Docker
  - Built-in LoadBalancer
  - local-path storage
  - Fast startup
  - [Example values](hypersdk/examples/k3d-values.yaml)

- **KIND** - Kubernetes in Docker
  - Upstream Kubernetes
  - CI/CD friendly
  - NodePort access
  - [Example values](hypersdk/examples/kind-values.yaml)

- **Minikube** - Local Kubernetes
  - Single-node cluster
  - Easy setup
  - Good for development
  - [Example values](hypersdk/examples/minikube-values.yaml)

### Cloud Providers

- **GKE** - Google Kubernetes Engine
  - Workload Identity
  - standard-rwo storage
  - Internal LoadBalancer
  - [Example values](hypersdk/examples/gke-values.yaml)

- **EKS** - Amazon Elastic Kubernetes Service
  - IRSA (IAM Roles for Service Accounts)
  - gp3 storage
  - Network Load Balancer
  - [Example values](hypersdk/examples/eks-values.yaml)

- **AKS** - Azure Kubernetes Service
  - Pod Identity
  - managed-premium storage
  - Azure Load Balancer
  - [Example values](hypersdk/examples/aks-values.yaml)

## Configuration

### Basic Configuration

```yaml
# values.yaml
replicaCount: 1

image:
  repository: ghcr.io/ssahani/hypersdk-hypervisord
  tag: latest
  pullPolicy: IfNotPresent

resources:
  requests:
    memory: "512Mi"
    cpu: "250m"
  limits:
    memory: "2Gi"
    cpu: "1000m"

persistence:
  data:
    size: 10Gi
  exports:
    size: 500Gi
```

### Cloud Provider Credentials

```yaml
# vSphere
credentials:
  vsphere:
    enabled: true
    url: "https://vcenter.example.com/sdk"
    username: "admin"
    password: "change-me"

# AWS
credentials:
  aws:
    enabled: true
    accessKeyId: "AKIA..."
    secretAccessKey: "..."
    region: "us-west-2"

# Azure
credentials:
  azure:
    enabled: true
    subscriptionId: "..."
    tenantId: "..."
    clientId: "..."
    clientSecret: "..."
```

### Production Features

```yaml
# Auto-scaling
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

# Monitoring
monitoring:
  serviceMonitor:
    enabled: true

# Network security
networkPolicy:
  enabled: true

# Ingress
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: hypersdk.example.com
      paths:
        - path: /
          pathType: Prefix
```

## Testing

### Running Tests

```bash
# Run all chart tests
../scripts/test-helm-chart.sh

# Run with deployment test
../scripts/test-helm-chart.sh --deploy

# Verbose output
../scripts/test-helm-chart.sh --verbose
```

### Test Coverage

- ✅ Chart structure validation
- ✅ Chart.yaml and values.yaml syntax
- ✅ Helm lint (0 errors/warnings)
- ✅ Template rendering (5 configurations)
- ✅ Required templates verification
- ✅ All example values validation
- ✅ Semantic version format
- ✅ YAML validity
- ✅ NOTES.txt validation

Results: **14/14 tests passed (100%)**

## Publishing

### For Maintainers

Package and publish new chart versions:

```bash
# Package chart
../scripts/package-helm-chart.sh

# Package and publish to GitHub Pages
../scripts/package-helm-chart.sh --publish

# Package with version override
../scripts/package-helm-chart.sh --version 0.3.0 --publish

# Sign chart with GPG
../scripts/package-helm-chart.sh --sign --publish
```

### Automated Release

Create and push a git tag for automated release:

```bash
# Tag release
git tag v0.3.0

# Push tag (triggers GitHub Actions)
git push origin v0.3.0
```

GitHub Actions will automatically:
1. Test the chart
2. Package the chart
3. Update repository index
4. Publish to GitHub Pages
5. Create GitHub Release
6. Test deployment

## Upgrading

### Quick Upgrade

```bash
# Upgrade to latest
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values

# Upgrade to specific version
helm upgrade hypersdk hypersdk/hypersdk \
  --version 0.3.0 \
  --namespace hypersdk \
  --reuse-values
```

See [UPGRADE.md](UPGRADE.md) for complete upgrade documentation.

## Monitoring

### Prometheus Metrics

HyperSDK exposes Prometheus metrics on port 8081:

```bash
# Port forward to metrics
kubectl port-forward -n hypersdk svc/hypersdk 8081:8081

# Scrape metrics
curl http://localhost:8081/metrics
```

### Grafana Dashboard

Import the Kubernetes-specific dashboard:

1. File: `../kubernetes/monitoring/grafana-dashboard-k8s.json`
2. Grafana → Dashboards → Import
3. Upload JSON file
4. Select Prometheus datasource

Dashboard includes:
- Running pods count
- Service health
- CPU/memory usage per pod
- Active/total jobs
- HTTP request rate
- Pod restart count
- PVC usage

## Troubleshooting

### Common Issues

**Chart not found in repository:**
```bash
helm repo update
helm search repo hypersdk
```

**Pod not starting:**
```bash
kubectl get pods -n hypersdk
kubectl describe pod -n hypersdk <pod-name>
kubectl logs -n hypersdk <pod-name>
```

**PVC not binding:**
```bash
kubectl get pvc -n hypersdk
kubectl describe pvc -n hypersdk hypersdk-data
kubectl get storageclass
```

**Service not accessible:**
```bash
kubectl get svc -n hypersdk
kubectl port-forward -n hypersdk svc/hypersdk 8080:8080
```

See [Troubleshooting Guide](../../docs/reference/troubleshooting-guide.md) for more.

## Scripts

### Deployment
- `../scripts/deploy-helm.sh` - Quick deployment to any environment

### Testing
- `../scripts/test-helm-chart.sh` - Comprehensive chart testing

### Publishing
- `../scripts/package-helm-chart.sh` - Package and publish charts

## CI/CD

### GitHub Actions Workflows

- `.github/workflows/helm-test.yml` - Chart testing on PR/push
- `.github/workflows/helm-release.yml` - Automated chart releases

### Integration

```yaml
# Example CI/CD
- name: Deploy HyperSDK
  run: |
    helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
    helm upgrade --install hypersdk hypersdk/hypersdk \
      --namespace hypersdk \
      --create-namespace \
      --wait
```

## Support

- **Documentation**: [docs/](../../docs/)
- **Issues**: https://github.com/ssahani/hypersdk/issues
- **Discussions**: https://github.com/ssahani/hypersdk/discussions

## License

The Helm chart is licensed under LGPL-3.0-or-later, same as HyperSDK.

## Summary

The HyperSDK Helm chart provides:

✅ **Production-ready deployment** with HA support
✅ **Multi-cloud configurations** (GKE, EKS, AKS)
✅ **Local development** (k3d, KIND, minikube)
✅ **Comprehensive testing** (14 automated tests)
✅ **Easy deployment** (one-command script)
✅ **Upgrade support** (with rollback)
✅ **Monitoring integration** (Prometheus, Grafana)
✅ **Security hardening** (NetworkPolicy, RBAC, non-root)
✅ **Auto-scaling** (HPA)
✅ **Complete documentation** (guides, examples, troubleshooting)

Get started with:
```bash
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm install hypersdk hypersdk/hypersdk --create-namespace -n hypersdk
```
