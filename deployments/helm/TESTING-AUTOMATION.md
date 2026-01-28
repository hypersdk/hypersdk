# Helm Chart Testing & Automation

Comprehensive testing and automation infrastructure for the HyperSDK Helm chart.

## 🎯 Overview

This document covers the automated testing, CI/CD, security scanning, cost reporting, and disaster recovery automation for the HyperSDK Helm chart.

## 📋 Table of Contents

- [Unit Tests](#unit-tests)
- [GitHub Actions Workflows](#github-actions-workflows)
- [Grafana Dashboards](#grafana-dashboards)
- [Local Testing](#local-testing)
- [CI/CD Integration](#cicd-integration)

## Unit Tests

### Helm Unittest Plugin

We use the [helm-unittest](https://github.com/helm-unittest/helm-unittest) plugin for chart unit testing.

**Installation**:
```bash
helm plugin install https://github.com/helm-unittest/helm-unittest
```

**Test Structure**:
```
deployments/helm/hypersdk/tests/
├── deployment_test.yaml      # Deployment resource tests
├── service_test.yaml          # Service resource tests
├── ingress_test.yaml          # Ingress resource tests
├── pvc_test.yaml             # PersistentVolumeClaim tests
└── servicemonitor_test.yaml   # ServiceMonitor tests
```

**Running Tests**:
```bash
# Run all tests
helm unittest deployments/helm/hypersdk

# Run tests with verbose output
helm unittest deployments/helm/hypersdk -3

# Run specific test file
helm unittest deployments/helm/hypersdk -f tests/deployment_test.yaml

# Update snapshots
helm unittest deployments/helm/hypersdk -u
```

### Test Coverage

#### Deployment Tests (16 tests)
- ✅ Default values validation
- ✅ Replica count configuration
- ✅ Image repository and tag handling
- ✅ Resource requests and limits
- ✅ Security context (non-root, fsGroup)
- ✅ Volume mounts (data, exports)
- ✅ Liveness and readiness probes
- ✅ Prometheus annotations
- ✅ Node selector and tolerations
- ✅ Affinity rules
- ✅ Environment variables
- ✅ Secret references

#### Service Tests (10 tests)
- ✅ ClusterIP service (default)
- ✅ Port configuration (8080, 8081)
- ✅ LoadBalancer service type
- ✅ NodePort with custom nodePort
- ✅ LoadBalancer IP and source ranges
- ✅ Custom annotations
- ✅ Selector labels
- ✅ Session affinity

#### Ingress Tests (8 tests)
- ✅ Ingress disabled by default
- ✅ Ingress enabled with hosts
- ✅ API version detection
- ✅ IngressClassName configuration
- ✅ TLS configuration
- ✅ Custom annotations (cert-manager, etc.)
- ✅ Multiple hosts
- ✅ Backend service configuration

#### PVC Tests (7 tests)
- ✅ Data PVC creation
- ✅ Exports PVC creation
- ✅ Custom storage sizes
- ✅ StorageClass configuration
- ✅ Disable data PVC
- ✅ Disable exports PVC
- ✅ Existing claim usage

#### ServiceMonitor Tests (7 tests)
- ✅ ServiceMonitor disabled by default
- ✅ ServiceMonitor creation
- ✅ Endpoint configuration
- ✅ Custom interval and scrapeTimeout
- ✅ Selector labels
- ✅ Custom labels
- ✅ Namespace configuration

**Total: 48 automated tests**

## GitHub Actions Workflows

### 1. Helm Chart Tests (`helm-test.yml`)

**Triggers**:
- Pull requests modifying `deployments/helm/**`
- Push to `main` branch

**Jobs**:
1. **lint-and-test**: Helm lint, unittest, chart-testing
2. **kubeconform**: Kubernetes manifest schema validation
3. **install-test**: End-to-end installation on Kind cluster
4. **security-scan**: Trivy vulnerability scanning
5. **documentation**: helm-docs verification
6. **test-summary**: Aggregate results

**Usage**:
```bash
# Automatically runs on PR
git push origin feature-branch

# Manually trigger
gh workflow run helm-test.yml
```

### 2. Helm Chart Publishing (`helm-publish.yml`)

**Triggers**:
- Release published
- Manual workflow dispatch

**Jobs**:
1. **publish-ghcr**: Push to GitHub Container Registry
2. **publish-github-pages**: Update Helm repository
3. **publish-dockerhub**: Push to Docker Hub (optional)
4. **create-release-notes**: Generate installation instructions
5. **verify-publication**: Validate successful publication

**Usage**:
```bash
# Automatically runs on release
gh release create v0.3.0 --title "Release v0.3.0"

# Manual publish
gh workflow run helm-publish.yml -f version=0.3.0
```

**Publishes to**:
- ✅ GHCR: `oci://ghcr.io/ssahani/charts/hypersdk`
- ✅ GitHub Pages: `https://ssahani.github.io/hypersdk/helm-charts`
- ⭕ Docker Hub: `oci://registry-1.docker.io/<username>/hypersdk` (optional)

### 3. Security Scanning (`security-scan.yml`)

**Triggers**:
- Daily at midnight UTC
- Manual workflow dispatch
- Push to `main` modifying deployments

**Jobs**:
1. **trivy-scan**: Vulnerability scanning for Helm and Dockerfiles
2. **secret-scanning**: Gitleaks and TruffleHog
3. **helm-security-check**: Best practices validation
4. **kubernetes-security**: kubesec and kube-score
5. **policy-check**: OPA/Conftest policy validation
6. **dependency-check**: Chart dependency scanning
7. **security-summary**: Aggregate and create issue on failure

**Security Checks**:
- ✅ No hardcoded secrets in values
- ✅ `runAsNonRoot: true`
- ✅ No privileged containers
- ✅ `allowPrivilegeEscalation: false`
- ✅ Resource limits defined
- ✅ No `latest` tag usage
- ✅ SARIF upload to GitHub Security

**Usage**:
```bash
# Manual scan
gh workflow run security-scan.yml

# View results
# GitHub → Security → Code scanning alerts
```

### 4. Weekly Cost Analysis (`cost-report.yml`)

**Triggers**:
- Weekly on Monday at 9 AM UTC
- Manual workflow dispatch

**Jobs**:
1. **analyze-resources**: Extract CPU/memory/storage specs
2. **compare-configurations**: Dev vs Staging vs Production costs
3. **check-optimization-opportunities**: HPA, PDB, resource utilization
4. **create-cost-issue**: Weekly report as GitHub issue

**Cost Estimates**:
- CPU: cores × 730 hours × $0.04/core-hour
- Memory: GB × 730 hours × $0.004/GB-hour
- Storage: GB × $0.10/GB-month

**Provides**:
- ✅ Estimated monthly costs (AWS/GCP/Azure)
- ✅ Configuration comparison table
- ✅ Optimization checklist
- ✅ Resource waste identification

**Usage**:
```bash
# Manual run
gh workflow run cost-report.yml

# View reports
gh issue list --label cost-report
```

### 5. Monthly DR Drill (`dr-drill.yml`)

**Triggers**:
- Monthly on 1st day at 10 AM UTC
- Manual workflow dispatch

**Jobs**:
1. **backup-test**: Volume snapshot and database backup
2. **restore-test**: Full restore in new cluster
3. **failover-test**: Simulate failure and measure RTO
4. **rpo-test**: Measure Recovery Point Objective
5. **generate-report**: Comprehensive DR report and GitHub issue

**Tests**:
- ✅ Automated backup creation
- ✅ Backup integrity verification
- ✅ Full restore procedures
- ✅ Self-healing validation
- ✅ RTO measurement (<15 minutes)
- ✅ RPO measurement (<1 hour)

**Compliance**:
- ✅ SOC 2: DR testing requirement
- ✅ HIPAA: Backup/recovery validation
- ✅ PCI-DSS: Business continuity

**Usage**:
```bash
# Manual DR drill
gh workflow run dr-drill.yml -f environment=staging

# View DR reports
gh issue list --label disaster-recovery
```

## Grafana Dashboards

### 1. HyperSDK Overview Dashboard

**File**: `deployments/helm/dashboards/hypersdk-overview.json`

**Panels**:
- Running instances gauge
- Request rate time series
- HTTP status code distribution
- Request duration percentiles (p50, p95, p99)
- Active export jobs
- Job completion rate
- Export duration percentiles
- Memory and CPU usage

**Metrics Used**:
```promql
# Request rate
rate(hypersdk_http_requests_total[5m])

# Error rate
rate(hypersdk_http_requests_total{status=~"5.."}[5m])

# Latency
histogram_quantile(0.95, rate(hypersdk_http_request_duration_seconds_bucket[5m]))

# Active jobs
hypersdk_active_jobs

# Job success rate
rate(hypersdk_jobs_completed_total{status="success"}[5m])
```

**Use Cases**:
- Real-time operations monitoring
- Performance troubleshooting
- SLA tracking (99.9% uptime, <200ms p95 latency)
- Capacity planning

### 2. Cost Tracking Dashboard

**File**: `deployments/helm/dashboards/hypersdk-cost-tracking.json`

**Panels**:
- Estimated monthly CPU cost
- Estimated monthly memory cost
- Estimated monthly storage cost
- Total estimated monthly cost
- Resource utilization vs requests
- CPU requests vs usage
- Resource waste by pod (table)

**Metrics Used**:
```promql
# CPU cost
sum(kube_pod_container_resource_requests{resource="cpu"}) * 730 * 0.04

# Memory cost
sum(kube_pod_container_resource_requests{resource="memory"}) / 1024^3 * 730 * 0.004

# Storage cost
sum(kube_persistentvolumeclaim_resource_requests_storage_bytes) / 1024^3 * 0.10

# Utilization
avg(rate(container_cpu_usage_seconds_total[5m])) / avg(kube_pod_container_resource_requests{resource="cpu"}) * 100
```

**Use Cases**:
- FinOps cost tracking
- Resource right-sizing
- Budget forecasting
- Waste identification (over-provisioned resources)

**Installation**:
```bash
# Import via Grafana UI
# Or deploy via ConfigMap
kubectl create configmap hypersdk-dashboards \
  --from-file=deployments/helm/dashboards/ \
  -n monitoring

kubectl label configmap hypersdk-dashboards \
  grafana_dashboard=1 \
  -n monitoring
```

## Local Testing

### Prerequisites

Install required tools:

```bash
# Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# helm-unittest
helm plugin install https://github.com/helm-unittest/helm-unittest

# kubeconform
wget https://github.com/yannh/kubeconform/releases/latest/download/kubeconform-linux-amd64.tar.gz
tar xf kubeconform-linux-amd64.tar.gz
sudo mv kubeconform /usr/local/bin/

# kubectl (for Kind tests)
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Kind (for integration tests)
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

### Test Script

Run the comprehensive test suite:

```bash
# Full test suite
./deployments/scripts/test-helm-chart.sh

# Lint only
./deployments/scripts/test-helm-chart.sh --skip-template --skip-values

# With deployment to Kind
./deployments/scripts/test-helm-chart.sh --deploy

# Verbose mode
./deployments/scripts/test-helm-chart.sh -v
```

### Manual Testing

```bash
# 1. Lint
helm lint deployments/helm/hypersdk

# 2. Unit tests
helm unittest deployments/helm/hypersdk

# 3. Template rendering
helm template test deployments/helm/hypersdk

# 4. Dry-run install
helm install --dry-run --debug test deployments/helm/hypersdk

# 5. Schema validation
helm template test deployments/helm/hypersdk | kubeconform -strict -summary

# 6. Install to Kind
kind create cluster --name hypersdk-test
helm install hypersdk deployments/helm/hypersdk -n hypersdk --create-namespace
kubectl wait --for=condition=available --timeout=300s deployment/hypersdk -n hypersdk
kubectl port-forward -n hypersdk svc/hypersdk 8080:8080
curl http://localhost:8080/health
```

## CI/CD Integration

### Pre-commit Hooks

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash
set -e

echo "Running Helm chart tests..."

# Lint
helm lint deployments/helm/hypersdk

# Unit tests
helm unittest deployments/helm/hypersdk

# Template test
helm template test deployments/helm/hypersdk > /dev/null

echo "✓ All pre-commit tests passed"
```

Make executable:
```bash
chmod +x .git/hooks/pre-commit
```

### Pull Request Checklist

Before merging:

- [ ] All unit tests pass (`helm unittest`)
- [ ] Helm lint passes (`helm lint`)
- [ ] Schema validation passes (`kubeconform`)
- [ ] Security scan passes (no HIGH/CRITICAL)
- [ ] Documentation updated (if applicable)
- [ ] Chart version bumped (if applicable)
- [ ] CHANGELOG updated (if applicable)

### Release Process

1. **Update version**:
   ```bash
   # Update Chart.yaml
   sed -i 's/^version:.*/version: 0.3.0/' deployments/helm/hypersdk/Chart.yaml
   sed -i 's/^appVersion:.*/appVersion: 0.3.0/' deployments/helm/hypersdk/Chart.yaml
   ```

2. **Update CHANGELOG**:
   ```bash
   # Add release notes to deployments/helm/hypersdk/CHANGELOG.md
   ```

3. **Commit and tag**:
   ```bash
   git add deployments/helm/hypersdk/Chart.yaml
   git commit -m "chore(helm): Bump version to 0.3.0"
   git tag -a v0.3.0 -m "Release v0.3.0"
   git push origin main --tags
   ```

4. **Create GitHub release**:
   ```bash
   gh release create v0.3.0 \
     --title "HyperSDK Helm Chart v0.3.0" \
     --notes-file deployments/helm/hypersdk/CHANGELOG.md
   ```

5. **Automated publishing**:
   - GitHub Actions automatically publishes to GHCR and GitHub Pages
   - Verify publication after ~5 minutes

6. **Test published chart**:
   ```bash
   helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
   helm repo update
   helm search repo hypersdk --version 0.3.0
   helm pull oci://ghcr.io/ssahani/charts/hypersdk --version 0.3.0
   ```

## Metrics and Monitoring

### Test Metrics

Track test quality over time:

- **Test count**: 48 unit tests
- **Coverage**: 100% of core templates
- **Pass rate**: Target 100%
- **Execution time**: <2 minutes

### CI/CD Metrics

Monitor pipeline health:

- **Build success rate**: Target >95%
- **Average build time**: ~5-10 minutes
- **Time to production**: <30 minutes
- **Deployment frequency**: Multiple times per day

### Security Metrics

Track security posture:

- **Critical vulnerabilities**: Target 0
- **High vulnerabilities**: Target <5
- **Secret leaks**: Target 0
- **Policy violations**: Target 0

## Troubleshooting

### Test Failures

**Unit tests fail**:
```bash
# Run with verbose output
helm unittest deployments/helm/hypersdk -3

# Debug specific test
helm unittest deployments/helm/hypersdk -f tests/deployment_test.yaml -3
```

**Schema validation fails**:
```bash
# Check manifest syntax
helm template test deployments/helm/hypersdk | kubectl apply --dry-run=client -f -

# Validate against specific K8s version
helm template test deployments/helm/hypersdk | \
  kubeconform -strict -kubernetes-version 1.29.0
```

**Security scan fails**:
```bash
# Local Trivy scan
trivy config deployments/helm/hypersdk

# Check for secrets
git secrets --scan deployments/helm/
```

### CI/CD Issues

**Workflow fails**:
```bash
# View workflow runs
gh run list --workflow helm-test.yml

# View specific run
gh run view <run-id>

# Download logs
gh run download <run-id>
```

**Publication fails**:
```bash
# Check GITHUB_TOKEN permissions
# Settings → Actions → General → Workflow permissions

# Verify gh-pages branch exists
git fetch origin gh-pages

# Manual publish
./deployments/scripts/package-helm-chart.sh
./deployments/scripts/publish-oci.sh
```

## Best Practices

1. **Always run tests locally** before pushing
2. **Keep tests fast** (<2 minutes total)
3. **Update tests** when changing templates
4. **Monitor test coverage** (aim for 100% of critical paths)
5. **Review security scans** daily
6. **Investigate cost anomalies** weekly
7. **Validate DR drills** monthly
8. **Update dashboards** as metrics evolve

## Related Documentation

- [HELM-TESTING-GUIDE.md](HELM-TESTING-GUIDE.md) - Detailed testing guide
- [OBSERVABILITY.md](OBSERVABILITY.md) - Metrics and monitoring setup
- [COST-OPTIMIZATION.md](COST-OPTIMIZATION.md) - Cost reduction strategies
- [DISASTER-RECOVERY.md](DISASTER-RECOVERY.md) - DR procedures
- [SECURITY.md](SECURITY.md) - Security hardening

## Support

- **Issues**: https://github.com/ssahani/hypersdk/issues
- **Discussions**: https://github.com/ssahani/hypersdk/discussions
- **Helm Docs**: https://helm.sh/docs/

---

**🚀 With comprehensive testing and automation, achieve 99.9% uptime and rapid deployment velocity!**
