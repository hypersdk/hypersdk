# HyperSDK Helm Repository Setup

This document describes the Helm repository infrastructure and how to enable it.

## Overview

HyperSDK now has a fully automated Helm chart distribution system with:

- 📦 **Automated Packaging** - Scripts and CI/CD for chart packaging
- 🌐 **GitHub Pages Repository** - Public Helm repository hosting
- 🚀 **Automated Releases** - Git tag-triggered chart publishing
- ✅ **Comprehensive Testing** - 14 automated tests + deployment validation
- 📚 **Complete Documentation** - Publishing guides and troubleshooting

## Repository URL

```
https://ssahani.github.io/hypersdk/helm-charts
```

## Quick Start (Users)

### Install from Helm Repository

```bash
# Add the repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm repo update

# Search for charts
helm search repo hypersdk

# Install HyperSDK
helm install my-hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --create-namespace
```

### Cloud Provider Installations

```bash
# Google Kubernetes Engine (GKE)
helm install hypersdk hypersdk/hypersdk \
  --values https://raw.githubusercontent.com/ssahani/hypersdk/main/deployments/helm/hypersdk/examples/gke-values.yaml

# Amazon EKS
helm install hypersdk hypersdk/hypersdk \
  --values https://raw.githubusercontent.com/ssahani/hypersdk/main/deployments/helm/hypersdk/examples/eks-values.yaml

# Azure AKS
helm install hypersdk hypersdk/hypersdk \
  --values https://raw.githubusercontent.com/ssahani/hypersdk/main/deployments/helm/hypersdk/examples/aks-values.yaml
```

## Enabling GitHub Pages (Repository Maintainers)

### Step 1: Navigate to Repository Settings

1. Go to: https://github.com/ssahani/hypersdk
2. Click **Settings** (top navigation)
3. Click **Pages** (left sidebar under "Code and automation")

### Step 2: Configure GitHub Pages

1. **Source**: Select "Deploy from a branch"
2. **Branch**: Select `main`
3. **Folder**: Select `/docs`
4. Click **Save**

### Step 3: Wait for Deployment

GitHub Pages will deploy automatically:
- Deployment time: ~1-2 minutes
- Status: Check the "Actions" tab for "pages-build-deployment" workflow
- Once complete, repository will be accessible at:
  ```
  https://ssahani.github.io/hypersdk/helm-charts
  ```

### Step 4: Verify Repository

```bash
# Test repository access
helm repo add hypersdk-test https://ssahani.github.io/hypersdk/helm-charts

# Search for charts
helm search repo hypersdk-test

# Expected output:
# NAME                  CHART VERSION  APP VERSION  DESCRIPTION
# hypersdk-test/hypersdk 0.2.0          0.2.0        Multi-cloud VM export and migration toolkit
```

### Step 5: Update DNS (Optional)

For custom domain:
1. Add CNAME record: `charts.hypersdk.io` → `ssahani.github.io`
2. In GitHub Pages settings, add custom domain: `charts.hypersdk.io`
3. Enable "Enforce HTTPS"

## Repository Structure

```
docs/helm-charts/                    # GitHub Pages directory
├── index.yaml                       # Helm repository index
├── index.html                       # Web interface (auto-generated)
├── README.md                        # Repository documentation
├── hypersdk-0.2.0.tgz              # Chart package (v0.2.0)
├── hypersdk-0.3.0.tgz              # Chart package (v0.3.0)
└── ...                              # Future versions

deployments/helm/packages/           # Build artifacts
├── index.yaml                       # Local repository index
├── hypersdk-0.2.0.tgz              # Packaged chart
└── ...

deployments/helm/hypersdk/           # Chart source
├── Chart.yaml                       # Chart metadata
├── values.yaml                      # Default values
├── templates/                       # Kubernetes templates
└── examples/                        # Example configurations
```

## Publishing New Chart Versions

### Method 1: Automated (Recommended)

Create and push a git tag:

```bash
# Tag the release
git tag v0.3.0

# Push the tag
git push origin v0.3.0
```

GitHub Actions will automatically:
1. ✅ Run 14 chart tests
2. 📦 Package the chart
3. 📝 Update repository index
4. 🌐 Publish to GitHub Pages
5. 🎉 Create GitHub Release
6. 🧪 Test deployment on minikube/kind

### Method 2: Manual

```bash
# Package and publish
./deployments/scripts/package-helm-chart.sh --publish

# Commit and push
git add docs/helm-charts deployments/helm/packages
git commit -m "helm: Publish chart version 0.3.0"
git push origin main
```

### Method 3: GitHub Actions Manual Dispatch

1. Go to: https://github.com/ssahani/hypersdk/actions/workflows/helm-release.yml
2. Click "Run workflow"
3. Enter chart version: `0.3.0`
4. Click "Run workflow"

## Scripts and Automation

### Packaging Script

**File**: `deployments/scripts/package-helm-chart.sh`

**Purpose**: Package and publish Helm charts

**Usage**:
```bash
# Package only
./deployments/scripts/package-helm-chart.sh

# Package and publish to GitHub Pages
./deployments/scripts/package-helm-chart.sh --publish

# Override version
./deployments/scripts/package-helm-chart.sh --version 0.3.0 --publish

# Sign chart with GPG
./deployments/scripts/package-helm-chart.sh --sign --publish
```

**Features**:
- Helm lint validation
- Chart test execution
- Package creation (.tgz)
- Repository index update
- GitHub Pages file generation
- Optional GPG signing

### CI/CD Workflows

#### Chart Testing (`.github/workflows/helm-test.yml`)

**Triggers**:
- Push to `main` or `develop` (helm changes)
- Pull requests (helm changes)

**Jobs**:
1. **lint-and-template**: Lint and render all configurations
2. **minikube-deployment**: Full deployment test
3. **kind-deployment**: Multi-node deployment test
4. **cloud-values-validation**: GKE/EKS/AKS validation

#### Chart Release (`.github/workflows/helm-release.yml`)

**Triggers**:
- Git tags: `v*.*.*` or `helm-*`
- Manual workflow dispatch

**Jobs**:
1. **lint-and-test**: Chart validation
2. **package-and-publish**: Package and publish to GitHub Pages
3. **verify-repository**: Repository accessibility check
4. **deploy-test**: Deployment testing on minikube/kind

## Monitoring and Verification

### Check GitHub Actions

```bash
# View workflow runs
open https://github.com/ssahani/hypersdk/actions
```

### Verify Repository Index

```bash
# Check index.yaml exists
curl -I https://ssahani.github.io/hypersdk/helm-charts/index.yaml

# View repository web interface
open https://ssahani.github.io/hypersdk/helm-charts
```

### Test Chart Installation

```bash
# Add repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts

# Update repositories
helm repo update

# Search for charts
helm search repo hypersdk

# Show chart info
helm show chart hypersdk/hypersdk
helm show values hypersdk/hypersdk
```

## Troubleshooting

### Repository Returns 404

**Problem**: `https://ssahani.github.io/hypersdk/helm-charts` returns 404

**Solutions**:
1. Verify GitHub Pages is enabled:
   - Settings → Pages → Source: `main` branch, `/docs` folder
2. Wait 1-2 minutes for GitHub Pages deployment
3. Check Actions tab for "pages-build-deployment" workflow
4. Verify `docs/helm-charts/` directory exists in main branch

### Chart Not Found

**Problem**: `helm search repo hypersdk` returns no results

**Solutions**:
1. Update Helm cache: `helm repo update`
2. Verify repository URL: `helm repo list`
3. Check index.yaml: `curl https://ssahani.github.io/hypersdk/helm-charts/index.yaml`
4. Remove and re-add repository:
   ```bash
   helm repo remove hypersdk
   helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
   helm repo update
   ```

### GitHub Actions Fails

**Problem**: Chart release workflow fails

**Solutions**:
1. Check workflow logs in Actions tab
2. Verify chart tests pass: `./deployments/scripts/test-helm-chart.sh`
3. Verify lint passes: `helm lint deployments/helm/hypersdk`
4. Check Chart.yaml version format (must be semver)

### Index.yaml Out of Sync

**Problem**: Old chart versions missing from repository

**Solutions**:
1. Repository index uses `--merge`, preserving old versions
2. If corrupted, regenerate:
   ```bash
   rm deployments/helm/packages/index.yaml
   ./deployments/scripts/package-helm-chart.sh --update-index
   ```
3. Commit and push updated index

## Best Practices

### Version Management

1. **Follow Semantic Versioning**: `MAJOR.MINOR.PATCH`
   - MAJOR: Breaking changes
   - MINOR: New features (backward compatible)
   - PATCH: Bug fixes (backward compatible)

2. **Update Chart Version**: Edit `Chart.yaml` before tagging
   ```yaml
   version: 0.3.0
   appVersion: 0.3.0
   ```

3. **Create Git Tags**: Always tag releases
   ```bash
   git tag -a v0.3.0 -m "Release version 0.3.0"
   git push origin v0.3.0
   ```

### Release Checklist

Before releasing a new chart version:

- [ ] Update `version` in `Chart.yaml`
- [ ] Update `appVersion` in `Chart.yaml`
- [ ] Update `CHANGELOG.md`
- [ ] Run tests: `./deployments/scripts/test-helm-chart.sh`
- [ ] Lint chart: `helm lint deployments/helm/hypersdk`
- [ ] Test all examples (GKE, EKS, AKS, minikube)
- [ ] Update documentation if needed
- [ ] Create git tag: `git tag vX.Y.Z`
- [ ] Push tag: `git push origin vX.Y.Z`
- [ ] Verify GitHub Actions succeeds
- [ ] Test installation from repository
- [ ] Announce release

### Security

1. **Chart Signing** (recommended for production):
   ```bash
   ./deployments/scripts/package-helm-chart.sh --sign
   ```

2. **Verify Signed Charts**:
   ```bash
   helm verify hypersdk-0.3.0.tgz
   ```

3. **HTTPS Only**: GitHub Pages enforces HTTPS automatically

## Documentation

- **Chart README**: [deployments/helm/hypersdk/README.md](hypersdk/README.md)
- **Publishing Guide**: [PUBLISHING.md](PUBLISHING.md)
- **Test Results**: [TEST-RESULTS.md](TEST-RESULTS.md)
- **Main README**: [../../README.md](../../README.md)

## Support

- **Issues**: https://github.com/ssahani/hypersdk/issues
- **Discussions**: https://github.com/ssahani/hypersdk/discussions
- **Documentation**: https://github.com/ssahani/hypersdk/tree/main/docs

## Summary

The HyperSDK Helm repository is now fully configured and ready for use:

✅ **Packaging**: Automated scripts for chart packaging
✅ **Testing**: 14 comprehensive tests + deployment validation
✅ **Publishing**: GitHub Actions workflows for automated releases
✅ **Hosting**: GitHub Pages ready (needs enablement)
✅ **Documentation**: Complete guides and troubleshooting
✅ **Security**: Optional GPG signing support

**Next Action**: Enable GitHub Pages in repository settings to make the repository publicly accessible.

Once GitHub Pages is enabled, users can install HyperSDK with a simple:
```bash
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm install my-hypersdk hypersdk/hypersdk
```
