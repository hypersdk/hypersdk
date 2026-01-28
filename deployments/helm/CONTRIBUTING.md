# Contributing to HyperSDK Helm Charts

Thank you for your interest in contributing to the HyperSDK Helm charts! This guide will help you get started.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)
- [Community](#community)

## Code of Conduct

### Our Standards

- **Be respectful** - Treat everyone with respect
- **Be constructive** - Provide helpful feedback
- **Be collaborative** - Work together toward common goals
- **Be patient** - Help newcomers learn

## Getting Started

### Prerequisites

Before contributing, ensure you have:

```bash
# Required tools
helm >= 3.8.0
kubectl >= 1.24.0
git
yamllint
helm-docs (optional, for documentation generation)

# Recommended tools
k3d or kind (for local testing)
```

### Fork and Clone

```bash
# Fork the repository on GitHub
# Then clone your fork

git clone https://github.com/YOUR_USERNAME/hypersdk.git
cd hypersdk
git remote add upstream https://github.com/ssahani/hypersdk.git
```

## Development Setup

### Local Kubernetes Cluster

```bash
# Create local cluster with k3d
k3d cluster create hypersdk-dev

# Or with KIND
kind create cluster --name hypersdk-dev

# Verify
kubectl cluster-info
```

### Install Development Tools

```bash
# Install helm-docs (for generating README)
GO111MODULE=on go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest

# Install yamllint
pip install yamllint

# Install ct (chart-testing)
pip install ct
```

## Making Changes

### Chart Structure

```
deployments/helm/hypersdk/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default values
├── values.schema.json      # JSON schema for values (optional)
├── README.md              # Auto-generated from templates/
├── templates/             # Kubernetes manifests
│   ├── _helpers.tpl       # Template helpers
│   ├── deployment.yaml    # Main deployment
│   ├── service.yaml       # Service
│   ├── configmap.yaml     # Configuration
│   ├── secrets.yaml       # Secrets (optional)
│   ├── ingress.yaml       # Ingress (optional)
│   ├── hpa.yaml           # HPA (optional)
│   ├── pvc.yaml           # PersistentVolumeClaims
│   ├── servicemonitor.yaml # Prometheus monitoring (optional)
│   ├── networkpolicy.yaml # NetworkPolicy (optional)
│   ├── pdb.yaml           # PodDisruptionBudget (optional)
│   ├── rbac.yaml          # RBAC (optional)
│   ├── serviceaccount.yaml # ServiceAccount
│   └── NOTES.txt          # Install notes
├── examples/              # Example values files
│   ├── k3d-values.yaml
│   ├── kind-values.yaml
│   ├── production-values.yaml
│   └── ...
└── tests/                 # Chart tests (optional)
    └── test-connection.yaml
```

### Types of Contributions

#### 1. Bug Fixes

**Example: Fix incorrect label**

```yaml
# templates/deployment.yaml
# BEFORE
metadata:
  labels:
    app: hypersdk  # ❌ Should use helpers

# AFTER
metadata:
  labels:
    {{- include "hypersdk.labels" . | nindent 4 }}  # ✅ Use helper
```

#### 2. Feature Additions

**Example: Add support for podAntiAffinity**

```yaml
# values.yaml
affinity: {}
  # Add example in comments
  # podAntiAffinity:
  #   requiredDuringSchedulingIgnoredDuringExecution:
  #     - labelSelector:
  #         matchExpressions:
  #           - key: app.kubernetes.io/name
  #             operator: In
  #             values:
  #               - hypersdk
  #       topologyKey: kubernetes.io/hostname

# templates/deployment.yaml
spec:
  template:
    spec:
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

#### 3. Documentation Improvements

**Example: Improve values.yaml comments**

```yaml
# BEFORE
replicaCount: 3

# AFTER
# Number of HyperSDK replicas
# Note: SQLite limitation requires single replica for write operations
# Use 1 for development, 3+ for production with read replicas
replicaCount: 3
```

#### 4. Template Optimization

**Example: Add conditional resource**

```yaml
# templates/ingress.yaml
{{- if .Values.ingress.enabled -}}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "hypersdk.fullname" . }}
  labels:
    {{- include "hypersdk.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  {{- if .Values.ingress.className }}
  ingressClassName: {{ .Values.ingress.className }}
  {{- end }}
  # ... rest of spec
{{- end }}
```

### Best Practices

#### Template Guidelines

**1. Use Helpers for Common Patterns**

```yaml
# templates/_helpers.tpl
{{/*
Common labels
*/}}
{{- define "hypersdk.labels" -}}
helm.sh/chart: {{ include "hypersdk.chart" . }}
{{ include "hypersdk.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

# Usage in templates
metadata:
  labels:
    {{- include "hypersdk.labels" . | nindent 4 }}
```

**2. Make Resources Optional**

```yaml
# Good: Allow disabling features
{{- if .Values.monitoring.serviceMonitor.enabled -}}
# ServiceMonitor definition
{{- end }}

# Bad: Always create resources
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
# ...
```

**3. Validate Required Values**

```yaml
# templates/_helpers.tpl
{{- define "hypersdk.validateValues" -}}
{{- if and .Values.credentials.vsphere.enabled (not .Values.credentials.vsphere.existingSecret) -}}
{{- if or (not .Values.credentials.vsphere.url) (not .Values.credentials.vsphere.username) (not .Values.credentials.vsphere.password) -}}
{{- fail "When vsphere credentials are enabled, you must provide url, username, and password OR set existingSecret" -}}
{{- end -}}
{{- end -}}
{{- end -}}

# templates/deployment.yaml
{{- include "hypersdk.validateValues" . -}}
```

**4. Use Proper Indentation**

```yaml
# Good
{{- with .Values.nodeSelector }}
nodeSelector:
  {{- toYaml . | nindent 2 }}
{{- end }}

# Bad (incorrect indentation)
{{- with .Values.nodeSelector }}
  nodeSelector:
    {{- toYaml . | nindent 2 }}
{{- end }}
```

**5. Handle Empty Values**

```yaml
# Good
{{- if .Values.annotations }}
annotations:
  {{- toYaml .Values.annotations | nindent 4 }}
{{- end }}

# Also good (with default)
annotations:
  {{- if .Values.annotations }}
  {{- toYaml .Values.annotations | nindent 2 }}
  {{- else }}
  {}
  {{- end }}
```

## Testing

### Lint Chart

```bash
# Lint the chart
helm lint deployments/helm/hypersdk

# Check YAML syntax
yamllint deployments/helm/hypersdk

# Template and validate
helm template hypersdk deployments/helm/hypersdk | kubectl apply --dry-run=client -f -
```

### Test Locally

```bash
# Create test cluster
k3d cluster create test

# Install chart
helm install hypersdk deployments/helm/hypersdk \
  --namespace hypersdk \
  --create-namespace \
  --values deployments/helm/hypersdk/examples/k3d-values.yaml

# Verify installation
kubectl get all -n hypersdk
helm test hypersdk -n hypersdk

# Check rendered templates
helm get manifest hypersdk -n hypersdk

# Cleanup
k3d cluster delete test
```

### Run Test Suite

```bash
# Run the automated test suite
cd deployments/scripts
./test-helm-chart.sh

# Expected output:
# ✅ Test 1: Chart structure validation - PASSED
# ✅ Test 2: Chart metadata verification - PASSED
# ... (14 tests)
# All tests passed!
```

### Test Upgrades

```bash
# Install old version
helm install hypersdk hypersdk/hypersdk --version 0.1.0 -n hypersdk --create-namespace

# Upgrade to local version
helm upgrade hypersdk deployments/helm/hypersdk -n hypersdk

# Verify upgrade succeeded
helm history hypersdk -n hypersdk
kubectl rollout status deployment/hypersdk -n hypersdk
```

### Test Different Configurations

```bash
# Test with minimal values
helm install test1 deployments/helm/hypersdk \
  --set persistence.data.enabled=false \
  --set persistence.exports.enabled=false \
  -n test1 --create-namespace

# Test with full features
helm install test2 deployments/helm/hypersdk \
  -f deployments/helm/hypersdk/examples/production-values.yaml \
  -n test2 --create-namespace

# Test with monitoring enabled
helm install test3 deployments/helm/hypersdk \
  --set monitoring.serviceMonitor.enabled=true \
  --set monitoring.prometheusRule.enabled=true \
  -n test3 --create-namespace
```

## Documentation

### Generate README

The chart README is auto-generated from values.yaml comments:

```bash
# Install helm-docs
GO111MODULE=on go install github.com/norwoodj/helm-docs/cmd/helm-docs@latest

# Generate README
cd deployments/helm/hypersdk
helm-docs

# Verify changes
git diff README.md
```

### values.yaml Documentation Format

```yaml
# -- Number of replicas
# @default -- 1 for development, 3 for production
replicaCount: 3

# Image configuration
image:
  # -- Container image repository
  repository: ghcr.io/ssahani/hypersdk-hypervisord

  # -- Image pull policy
  # @default -- IfNotPresent
  pullPolicy: IfNotPresent

  # -- Overrides the image tag (default is chart appVersion)
  tag: ""

# @ignore (excludes from generated docs)
internal_field: value
```

### Update NOTES.txt

```bash
# templates/NOTES.txt - Shown after helm install
# Keep concise and actionable

1. Get the application URL:
{{- if .Values.ingress.enabled }}
  http{{ if .Values.ingress.tls }}s{{ end }}://{{ .Values.ingress.hosts[0].host }}
{{- else if contains "NodePort" .Values.service.type }}
  export NODE_PORT=$(kubectl get --namespace {{ .Release.Namespace }} -o jsonpath="{.spec.ports[0].nodePort}" services {{ include "hypersdk.fullname" . }})
  export NODE_IP=$(kubectl get nodes --namespace {{ .Release.Namespace }} -o jsonpath="{.items[0].status.addresses[0].address}")
  echo http://$NODE_IP:$NODE_PORT
{{- else if contains "LoadBalancer" .Values.service.type }}
  NOTE: It may take a few minutes for the LoadBalancer IP to be available.
  export SERVICE_IP=$(kubectl get svc --namespace {{ .Release.Namespace }} {{ include "hypersdk.fullname" . }} --template "{{"{{ range (index .status.loadBalancer.ingress 0) }}{{.}}{{ end }}"}}")
  echo http://$SERVICE_IP:{{ .Values.service.port }}
{{- end }}

2. Check the deployment status:
  kubectl rollout status deployment/{{ include "hypersdk.fullname" . }} -n {{ .Release.Namespace }}
```

### Example Values Files

When adding new features, update example values:

```bash
# deployments/helm/hypersdk/examples/production-values.yaml
# Add new feature
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  # NEW: Add memory-based scaling
  targetMemoryUtilizationPercentage: 80
```

## Submitting Changes

### Commit Message Format

Follow conventional commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting changes
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance

**Examples:**

```
feat(helm): Add support for pod anti-affinity

Added configurable pod anti-affinity rules to improve
high availability by spreading pods across nodes.

Closes #123
```

```
fix(helm): Correct service port mapping

Fixed incorrect port mapping in service template that
prevented external access to metrics endpoint.

Fixes #456
```

```
docs(helm): Improve values.yaml documentation

Enhanced documentation for persistence configuration
with examples for different cloud providers.
```

### Pull Request Process

1. **Create Feature Branch**

```bash
git checkout -b feat/add-pod-anti-affinity
```

2. **Make Changes**

```bash
# Edit files
vim deployments/helm/hypersdk/values.yaml
vim deployments/helm/hypersdk/templates/deployment.yaml

# Generate documentation
cd deployments/helm/hypersdk
helm-docs
```

3. **Test Changes**

```bash
# Lint
helm lint deployments/helm/hypersdk

# Test locally
k3d cluster create test
helm install test deployments/helm/hypersdk -n test --create-namespace
helm test test -n test

# Run full test suite
./deployments/scripts/test-helm-chart.sh
```

4. **Commit Changes**

```bash
git add deployments/helm/hypersdk/
git commit -m "feat(helm): Add support for pod anti-affinity"
```

5. **Push and Create PR**

```bash
git push origin feat/add-pod-anti-affinity

# Create PR on GitHub
# Fill out PR template with:
# - Description of changes
# - Testing performed
# - Screenshots (if UI changes)
# - Breaking changes (if any)
```

6. **Address Review Comments**

```bash
# Make requested changes
git add .
git commit -m "address review comments"
git push origin feat/add-pod-anti-affinity
```

### PR Checklist

Before submitting, ensure:

- [ ] Code follows chart best practices
- [ ] All tests pass (`./test-helm-chart.sh`)
- [ ] Documentation updated (README, values.yaml comments)
- [ ] Example values updated (if new feature)
- [ ] NOTES.txt updated (if user-facing changes)
- [ ] Commit messages follow conventional format
- [ ] No breaking changes (or clearly documented)
- [ ] Chart version bumped (see versioning below)

## Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: New functionality (backward compatible)
- **PATCH** version: Bug fixes (backward compatible)

### Update Chart Version

```yaml
# Chart.yaml

# For bug fixes
version: 0.2.1  # was 0.2.0

# For new features
version: 0.3.0  # was 0.2.0

# For breaking changes
version: 1.0.0  # was 0.2.0
```

### Update App Version

```yaml
# Chart.yaml
# When HyperSDK application version changes

appVersion: "0.3.0"  # was "0.2.0"
```

### Release Checklist

**For Maintainers:**

1. Update `Chart.yaml` version
2. Update `CHANGELOG.md`
3. Run full test suite
4. Create release tag: `git tag chart-v0.3.0`
5. Push tag: `git push origin chart-v0.3.0`
6. GitHub Actions will automatically:
   - Test chart
   - Package chart
   - Publish to GitHub Pages
   - Create GitHub Release

## Community

### Getting Help

- **Questions**: Open a [Discussion](https://github.com/ssahani/hypersdk/discussions)
- **Bugs**: Open an [Issue](https://github.com/ssahani/hypersdk/issues)
- **Chat**: Join our community (if available)

### Recognition

Contributors are recognized in:
- Git commit history
- Release notes
- `Co-Authored-By` in commit messages

## Advanced Topics

### Adding New Cloud Provider Support

Example: Adding DigitalOcean support

```yaml
# values.yaml
credentials:
  # ... existing providers
  digitalocean:
    enabled: false
    # Token for DigitalOcean API
    token: ""
    # Or use existing secret
    existingSecret: ""

# templates/secrets.yaml
{{- if and .Values.credentials.digitalocean.enabled (not .Values.credentials.digitalocean.existingSecret) }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "hypersdk.fullname" . }}-digitalocean
  labels:
    {{- include "hypersdk.labels" . | nindent 4 }}
type: Opaque
stringData:
  token: {{ .Values.credentials.digitalocean.token | quote }}
{{- end }}

# templates/deployment.yaml
{{- if .Values.credentials.digitalocean.enabled }}
- name: DIGITALOCEAN_TOKEN
  valueFrom:
    secretKeyRef:
      name: {{ .Values.credentials.digitalocean.existingSecret | default (printf "%s-digitalocean" (include "hypersdk.fullname" .)) }}
      key: token
{{- end }}
```

### Adding Chart Tests

```yaml
# templates/tests/test-connection.yaml
apiVersion: v1
kind: Pod
metadata:
  name: "{{ include "hypersdk.fullname" . }}-test-connection"
  labels:
    {{- include "hypersdk.labels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": test
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args: ['{{ include "hypersdk.fullname" . }}:{{ .Values.service.port }}/health']
  restartPolicy: Never
```

### Supporting Multiple Kubernetes Versions

```yaml
# Use API version checks
{{- if .Capabilities.APIVersions.Has "networking.k8s.io/v1/Ingress" }}
apiVersion: networking.k8s.io/v1
{{- else }}
apiVersion: networking.k8s.io/v1beta1
{{- end }}
kind: Ingress
```

## Thank You!

Thank you for contributing to HyperSDK! Your efforts help make this project better for everyone.

**Questions?** Feel free to open a discussion or issue.

---

**Happy Helm charting!** ⛵
