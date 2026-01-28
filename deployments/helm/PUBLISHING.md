# Publishing HyperSDK Helm Charts

This guide covers how to package and publish HyperSDK Helm charts.

## Overview

HyperSDK Helm charts are published to a Helm repository hosted on GitHub Pages at:
```
https://ssahani.github.io/hypersdk/helm-charts
```

## Prerequisites

- Helm 3.x installed
- Git configured with appropriate credentials
- GitHub Pages enabled for the repository

## Manual Publishing

### 1. Package the Chart

Package the chart with all validations:

```bash
cd /path/to/hypersdk
./deployments/scripts/package-helm-chart.sh
```

This will:
- Run Helm lint
- Run chart tests
- Package the chart to `deployments/helm/packages/`

### 2. Update Repository Index

Package and update the Helm repository index:

```bash
./deployments/scripts/package-helm-chart.sh --update-index
```

This creates/updates `deployments/helm/packages/index.yaml`.

### 3. Publish to GitHub Pages

Package, update index, and prepare for GitHub Pages:

```bash
./deployments/scripts/package-helm-chart.sh --publish
```

This will:
- Package the chart
- Update the repository index
- Copy files to `docs/helm-charts/`
- Create `index.html` for web interface
- Create README.md

### 4. Commit and Push

```bash
git add docs/helm-charts deployments/helm/packages
git commit -m "helm: Publish chart version X.Y.Z"
git push origin main
```

### 5. Enable GitHub Pages

In repository settings:
- Go to Settings → Pages
- Source: Deploy from a branch
- Branch: `main`
- Folder: `/docs`
- Save

## Automated Publishing (Recommended)

### Via Git Tags

Create and push a version tag:

```bash
# For general releases
git tag v0.3.0
git push origin v0.3.0

# For Helm-specific releases
git tag helm-0.3.0
git push origin helm-0.3.0
```

The GitHub Actions workflow (`.github/workflows/helm-release.yml`) will:
1. Run chart tests
2. Package the chart
3. Update the repository index
4. Commit to GitHub Pages directory
5. Create a GitHub Release with chart package
6. Verify the repository
7. Test deployment on minikube and kind

### Via GitHub Actions Manual Dispatch

Trigger a release manually:

1. Go to Actions → Helm Chart Release
2. Click "Run workflow"
3. Enter chart version (e.g., `0.3.0`)
4. Click "Run workflow"

## Versioning

### Chart Version

The chart version follows [Semantic Versioning](https://semver.org/):

- `MAJOR.MINOR.PATCH` (e.g., `0.3.0`)
- Update `version` in `Chart.yaml`
- Update `appVersion` in `Chart.yaml`

### Version Bumping

Manual version update:

```bash
# Edit Chart.yaml
vim deployments/helm/hypersdk/Chart.yaml

# Update version and appVersion
version: 0.3.0
appVersion: 0.3.0
```

Automated version update:

```bash
./deployments/scripts/package-helm-chart.sh --version 0.3.0 --publish
```

## Signing Charts (Optional)

Sign charts with GPG for added security:

```bash
# Generate GPG key if needed
gpg --gen-key

# Package and sign
./deployments/scripts/package-helm-chart.sh --sign
```

This creates:
- `hypersdk-X.Y.Z.tgz` - Chart package
- `hypersdk-X.Y.Z.tgz.prov` - Provenance file

## Repository Structure

```
docs/helm-charts/
├── index.yaml              # Helm repository index
├── index.html              # Web interface
├── README.md               # Repository documentation
├── hypersdk-0.2.0.tgz     # Chart package (version 0.2.0)
├── hypersdk-0.3.0.tgz     # Chart package (version 0.3.0)
└── ...                     # Additional versions
```

## Using the Published Repository

### Add Repository

```bash
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm repo update
```

### Search Charts

```bash
helm search repo hypersdk
```

Example output:
```
NAME              CHART VERSION  APP VERSION  DESCRIPTION
hypersdk/hypersdk 0.3.0          0.3.0        Multi-cloud VM export and migration toolkit
```

### Install Chart

```bash
helm install my-hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --create-namespace
```

### Install Specific Version

```bash
helm install my-hypersdk hypersdk/hypersdk \
  --version 0.2.0 \
  --namespace hypersdk \
  --create-namespace
```

## Troubleshooting

### Chart Not Found After Publishing

**Problem**: Chart not available in repository after publishing.

**Solution**:
1. Verify GitHub Pages is enabled and deployed
2. Check `docs/helm-charts/index.yaml` exists
3. Wait 1-2 minutes for GitHub Pages to update
4. Clear Helm cache: `helm repo update`

### Version Conflict

**Problem**: Chart version already exists.

**Solution**:
1. Bump the version in `Chart.yaml`
2. Re-package with new version
3. Or delete old package and re-publish

### Index.yaml Corruption

**Problem**: Repository index is corrupted or invalid.

**Solution**:
1. Delete `deployments/helm/packages/index.yaml`
2. Re-run: `./deployments/scripts/package-helm-chart.sh --update-index`
3. Verify: `helm repo index --merge` succeeded

### GitHub Pages 404

**Problem**: Repository URL returns 404.

**Solution**:
1. Verify GitHub Pages settings:
   - Settings → Pages
   - Source: `main` branch, `/docs` folder
2. Check `docs/helm-charts/` directory exists
3. Wait for GitHub Pages deployment (1-2 minutes)
4. Check GitHub Actions for Pages deployment logs

## Best Practices

### Before Publishing

1. **Run Tests**: Always run chart tests before publishing
   ```bash
   ./deployments/scripts/test-helm-chart.sh
   ```

2. **Lint Chart**: Ensure no lint errors
   ```bash
   helm lint deployments/helm/hypersdk
   ```

3. **Test All Configurations**: Test with all example values
   ```bash
   for values in deployments/helm/hypersdk/examples/*.yaml; do
       helm template test deployments/helm/hypersdk --values "$values"
   done
   ```

4. **Verify Version**: Ensure version is incremented
   ```bash
   grep '^version:' deployments/helm/hypersdk/Chart.yaml
   ```

### Version Management

1. **Semantic Versioning**: Follow semver strictly
   - `MAJOR`: Breaking changes
   - `MINOR`: New features, backward compatible
   - `PATCH`: Bug fixes, backward compatible

2. **Change Documentation**: Update `CHANGELOG.md` for each release

3. **Tag Releases**: Always create git tags for releases
   ```bash
   git tag -a v0.3.0 -m "Release version 0.3.0"
   git push origin v0.3.0
   ```

### Repository Maintenance

1. **Keep Old Versions**: Don't delete old chart packages
2. **Update Index**: Always update index when adding new versions
3. **Clean Packages**: Periodically clean very old versions (>1 year)

## Release Checklist

- [ ] Update chart version in `Chart.yaml`
- [ ] Update `appVersion` in `Chart.yaml`
- [ ] Update `CHANGELOG.md`
- [ ] Run chart tests: `./deployments/scripts/test-helm-chart.sh`
- [ ] Lint chart: `helm lint deployments/helm/hypersdk`
- [ ] Test all example configurations
- [ ] Package and publish: `./deployments/scripts/package-helm-chart.sh --publish`
- [ ] Commit and push GitHub Pages updates
- [ ] Create and push git tag
- [ ] Verify GitHub Actions workflow succeeds
- [ ] Verify chart is available in repository
- [ ] Test installation from repository
- [ ] Update documentation if needed
- [ ] Announce release

## Continuous Integration

The GitHub Actions workflow (`.github/workflows/helm-release.yml`) provides automated:

1. **Testing**: Lint and test on every tag push
2. **Packaging**: Automatic chart packaging
3. **Publishing**: Automatic GitHub Pages update
4. **Verification**: Repository accessibility check
5. **Deployment Testing**: Install and verify on minikube/kind
6. **GitHub Release**: Create release with chart artifacts

## Security

### Chart Signing

For production releases, sign charts:

1. Generate GPG key:
   ```bash
   gpg --gen-key
   ```

2. Export public key:
   ```bash
   gpg --export-secret-keys > ~/.gnupg/secring.gpg
   gpg --export > ~/.gnupg/pubring.gpg
   ```

3. Sign chart:
   ```bash
   ./deployments/scripts/package-helm-chart.sh --sign
   ```

### Verifying Signed Charts

Users can verify signed charts:

```bash
# Import public key
gpg --import pubkey.asc

# Verify chart
helm verify hypersdk-0.3.0.tgz
```

## Support

For issues with chart publishing:

1. Check [GitHub Actions logs](https://github.com/ssahani/hypersdk/actions)
2. Review [troubleshooting guide](#troubleshooting)
3. Open an issue on [GitHub](https://github.com/ssahani/hypersdk/issues)

## References

- [Helm Chart Repository Guide](https://helm.sh/docs/topics/chart_repository/)
- [GitHub Pages Documentation](https://docs.github.com/en/pages)
- [Semantic Versioning](https://semver.org/)
- [Chart Signing](https://helm.sh/docs/topics/provenance/)
