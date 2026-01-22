# GitHub CI/CD for HyperSDK Packaging

This document describes the continuous integration and deployment workflows for HyperSDK, including RPM packaging automation.

## Overview

HyperSDK uses GitHub Actions for automated testing, building, and releasing across multiple platforms and distributions.

## Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- Push to `main` or `develop` branches
- Pull requests to `main` or `develop` branches

**Jobs:**

#### Test
- Runs on Go versions: 1.21, 1.22, 1.23, 1.24
- Executes unit tests with race detection
- Generates code coverage reports
- Uploads coverage to Codecov

#### Lint
- Runs golangci-lint with 5-minute timeout
- Checks code quality and style

#### Build
- Builds all three binaries:
  - `hyperexport` - Interactive VM export CLI
  - `hypervisord` - REST API daemon
  - `hyperctl` - Control CLI
- Verifies binary versions
- Uploads binaries as artifacts

#### Build RPM
- Builds main `hypersdk` RPM package
- Tests on Fedora latest
- Uploads RPM artifacts

#### Build Daemon RPM
- Builds `hyper2kvm-daemon` RPM package
- Verifies systemd unit installation
- Tests RPM installation
- Uploads daemon RPM artifacts

#### Integration Test
- Tests daemon startup
- Validates hyperctl connectivity
- Verifies basic functionality

#### Security Scan
- Runs Gosec security scanner
- Runs Trivy vulnerability scanner
- Uploads results to GitHub Security tab

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers:**
- Push of version tags (e.g., `v1.0.0`, `v2.1.3`)

**Jobs:**

#### Build Binaries
- Builds cross-platform binaries for:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)
- Embeds version in binaries via ldflags
- Creates platform-specific archives
- Uploads binaries as release assets

#### Build RPM
- Builds main `hypersdk` RPM package
- Extracts version from git tag
- Uploads to GitHub release

#### Build Daemon RPM
- Builds `hyper2kvm-daemon` for multiple distributions:
  - Fedora 39 (`fc39`)
  - Fedora 40 (`fc40`)
  - Rocky Linux 8 (`el8`)
  - Rocky Linux 9 (`el9`)
- Uses `packaging/rpm/build.sh` automated script
- Uploads all distribution RPMs to release

#### Create Release
- Generates changelog from commits
- Creates GitHub release with notes
- Includes installation instructions
- Links to documentation
- Attaches all artifacts (binaries + RPMs)

#### Publish Verification
- Downloads Linux AMD64 binary
- Verifies binary integrity
- Tests version output

### 3. RPM Packaging Workflow (`.github/workflows/rpm-packaging.yml`)

**Triggers:**
- Push to `main` affecting:
  - `packaging/**`
  - `systemd/**`
  - Workflow file itself
- Pull requests to `main` (same paths)
- Release creation events
- Manual workflow dispatch with version/release parameters

**Jobs:**

#### Build RPM (Matrix)
Tests packaging across distributions:
- Fedora 39
- Fedora 40
- Rocky Linux 8
- Rocky Linux 9
- AlmaLinux 9

**Steps per distribution:**
1. Install RPM build tools
2. Build package using `build.sh`
3. Verify package metadata (`rpm -qpi`)
4. List package contents (`rpm -qpl`)
5. Test installation
6. Verify systemd units exist
7. Upload artifacts with 30-day retention

#### Lint Spec
- Runs `rpmlint` on spec file
- Checks for packaging best practices
- Continues on error (advisory only)

#### Test Build Script
- Tests `build.sh` help output
- Tests clean build
- Tests custom version/release
- Verifies version in built package

#### Security Scan
- Runs ShellCheck on build script
- Checks systemd units for security:
  - User directive
  - NoNewPrivileges
  - ProtectSystem
  - PrivateTmp

#### Documentation
- Verifies all docs exist:
  - `packaging/README.md`
  - `packaging/rpm/README.md`
  - `systemd/README.md`
  - `SYSTEMD_DAEMON_INTEGRATION.md`
- Runs markdownlint (advisory)

#### Summary
- Aggregates all job results
- Generates workflow summary
- Reports build status

## Artifacts

### CI Artifacts (Retained 30 days)

**Binaries:**
- `binaries` - Main binaries (hyperexport, hypervisord, hyperctl)
- `rpm-packages` - Main hypersdk RPM
- `daemon-rpm-packages` - hyper2kvm-daemon RPM

### Release Artifacts (Permanent)

**Binary Archives:**
- `hyperexport-{os}-{arch}[.exe]`
- `hypervisord-{os}-{arch}[.exe]`
- `hyperctl-{os}-{arch}[.exe]`

**RPM Packages:**
- `hypersdk-{version}-1.{dist}.{arch}.rpm`
- `hyper2kvm-daemon-{version}-1.{dist}.noarch.rpm`

**Source RPMs:**
- `hypersdk-{version}-1.{dist}.src.rpm`
- `hyper2kvm-daemon-{version}-1.{dist}.src.rpm`

### RPM Packaging Artifacts (Retained 30 days)

**Per Distribution:**
- `rpm-{dist}-{version}-{release}` - Binary and source RPMs
- `daemon-rpm-{dist}` - Daemon RPMs for each distro

## Manual Workflow Dispatch

### RPM Packaging Workflow

Manually trigger with custom version:

```bash
# Via GitHub CLI
gh workflow run rpm-packaging.yml \
  -f version=1.2.3 \
  -f release=2

# Via GitHub web UI
# Actions → RPM Packaging → Run workflow
# Set version: 1.2.3
# Set release: 2
```

## Environment Variables

### Release Workflow

- `GITHUB_REF` - Git reference (e.g., `refs/tags/v1.0.0`)
- `GITHUB_TOKEN` - Auto-generated GitHub token for releases

### Build Environment

- `GOOS` - Target operating system
- `GOARCH` - Target architecture
- `CGO_ENABLED` - Set to 0 for static binaries

## Version Management

### Git Tags

Release versions are determined by git tags:

```bash
# Create release
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# This triggers:
# 1. Release workflow
# 2. RPM packaging workflow (if enabled)
```

### Spec File Versions

RPM spec files are updated automatically:

```bash
# release.yml extracts version from tag
VERSION="${GITHUB_REF#refs/tags/v}"

# Updates spec file
sed -i "s/^Version:.*/Version: $VERSION/" hyper2kvm-daemon.spec
```

## Distribution Support

### Primary Distributions

- **Fedora**: 39, 40
- **Rocky Linux**: 8, 9
- **AlmaLinux**: 9
- **RHEL**: 8, 9 (compatible with Rocky/Alma)
- **CentOS Stream**: 8, 9 (compatible)

### Binary Platforms

- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64

## Testing Matrix

### Go Versions (CI)
- 1.21
- 1.22
- 1.23
- 1.24

### RPM Distributions
- Fedora 39, 40
- Rocky Linux 8, 9
- AlmaLinux 9

### Binary Platforms
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64
- windows/amd64

## Security

### Code Scanning

**Gosec:**
- Scans for security vulnerabilities
- Checks for insecure code patterns
- Runs on every CI build

**Trivy:**
- Scans filesystem for vulnerabilities
- Checks dependencies
- Uploads results to GitHub Security

**ShellCheck:**
- Lints shell scripts
- Identifies common mistakes
- Runs on packaging workflow

### Systemd Security

Automated checks verify:
- Non-root user execution
- Security hardening directives
- Resource limits
- Filesystem restrictions

## Release Process

### Creating a Release

1. **Prepare release:**
   ```bash
   # Update version in code if needed
   # Update CHANGELOG
   git add .
   git commit -m "Prepare release v1.0.0"
   ```

2. **Create and push tag:**
   ```bash
   git tag -a v1.0.0 -m "Release version 1.0.0"
   git push origin v1.0.0
   ```

3. **Monitor workflows:**
   - Check GitHub Actions tab
   - Verify all jobs complete successfully
   - Review build logs if needed

4. **Verify release:**
   - Check GitHub Releases page
   - Download and test artifacts
   - Verify release notes

### Release Artifacts Checklist

- [ ] Binary: `hyperexport-linux-amd64`
- [ ] Binary: `hyperexport-linux-arm64`
- [ ] Binary: `hyperexport-darwin-amd64`
- [ ] Binary: `hyperexport-darwin-arm64`
- [ ] Binary: `hyperexport-windows-amd64.exe`
- [ ] Binary: `hypervisord-linux-amd64`
- [ ] Binary: `hypervisord-linux-arm64`
- [ ] Binary: `hypervisord-darwin-amd64`
- [ ] Binary: `hypervisord-darwin-arm64`
- [ ] Binary: `hypervisord-windows-amd64.exe`
- [ ] Binary: `hyperctl-linux-amd64`
- [ ] Binary: `hyperctl-linux-arm64`
- [ ] Binary: `hyperctl-darwin-amd64`
- [ ] Binary: `hyperctl-darwin-arm64`
- [ ] Binary: `hyperctl-windows-amd64.exe`
- [ ] RPM: `hypersdk-{version}-1.fc39.rpm`
- [ ] RPM: `hypersdk-{version}-1.fc40.rpm`
- [ ] RPM: `hypersdk-{version}-1.el8.rpm`
- [ ] RPM: `hypersdk-{version}-1.el9.rpm`
- [ ] RPM: `hyper2kvm-daemon-{version}-1.fc39.noarch.rpm`
- [ ] RPM: `hyper2kvm-daemon-{version}-1.fc40.noarch.rpm`
- [ ] RPM: `hyper2kvm-daemon-{version}-1.el8.noarch.rpm`
- [ ] RPM: `hyper2kvm-daemon-{version}-1.el9.noarch.rpm`

## Troubleshooting

### Workflow Failures

**Build fails on specific distribution:**
```bash
# Test locally using container
docker run -it fedora:39 bash
dnf install -y rpm-build rpmdevtools git
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk/packaging/rpm
./build.sh
```

**RPM installation test fails:**
```bash
# Check systemd files exist in spec
grep -A5 "%files" packaging/rpm/hyper2kvm-daemon.spec

# Verify tarball contents
tar tzf ~/rpmbuild/SOURCES/hyper2kvm-daemon-*.tar.gz
```

**Version not updating:**
```bash
# Verify tag format
git tag -l "v*"

# Tag should be: v1.0.0 (not 1.0.0 or V1.0.0)
```

### Manual Build Testing

Test workflows locally before pushing:

```bash
# Install act (https://github.com/nektos/act)
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Test CI workflow
act push -W .github/workflows/ci.yml

# Test RPM packaging
act workflow_dispatch -W .github/workflows/rpm-packaging.yml \
  -j build-rpm
```

## Metrics

### Build Times (Approximate)

- **CI Workflow**: 15-20 minutes
- **Release Workflow**: 30-40 minutes
- **RPM Packaging**: 20-30 minutes

### Resource Usage

- **Disk**: ~2GB per workflow run
- **Artifacts**: ~500MB per release
- **Retention**: 30 days for artifacts, permanent for releases

## Best Practices

### Committing Changes

1. **Run tests locally first:**
   ```bash
   go test ./...
   go build ./cmd/...
   ```

2. **Test RPM build:**
   ```bash
   cd packaging/rpm
   ./build.sh --clean
   ```

3. **Verify workflows pass:**
   - Push to feature branch
   - Check CI passes
   - Create PR to main

### Creating Releases

1. **Version bumps only on main:**
   - Merge all changes to main first
   - Tag from main branch

2. **Test before tagging:**
   - Verify CI passes on main
   - Test manual builds

3. **Use semantic versioning:**
   - Major.Minor.Patch (e.g., 1.2.3)
   - Prefix with 'v' (v1.2.3)

## Future Enhancements

Planned improvements:

- [ ] Add DEB package workflow
- [ ] Container image builds and pushes
- [ ] Automated changelog generation
- [ ] Performance benchmarks
- [ ] Integration test suite
- [ ] Nightly builds
- [ ] Dependency caching optimization
- [ ] Multi-arch Docker images

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [RPM Packaging Guide](https://rpm-packaging-guide.github.io/)
- [Go Release Process](https://go.dev/doc/devel/release)
- [Semantic Versioning](https://semver.org/)

## Support

For workflow issues:
- Check workflow logs in GitHub Actions tab
- Review this documentation
- Open issue: https://github.com/ssahani/hypersdk/issues

For packaging issues:
- See [packaging/README.md](README.md)
- See [packaging/rpm/README.md](rpm/README.md)
