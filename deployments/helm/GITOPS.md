# GitOps Integration Guide for HyperSDK

This guide covers deploying HyperSDK using GitOps tools like ArgoCD and Flux.

## Overview

GitOps provides:
- **Declarative deployments** - Infrastructure as code
- **Automated synchronization** - Git as source of truth
- **Audit trail** - All changes tracked in Git
- **Rollback capability** - Revert to any Git commit
- **Multi-cluster support** - Deploy to multiple clusters

## ArgoCD Integration

### Prerequisites

- ArgoCD installed in cluster
- Git repository for manifests
- Helm repository accessible

### Method 1: Using Helm Repository

Create an ArgoCD Application manifest:

```yaml
# argocd/hypersdk-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk
  namespace: argocd
spec:
  project: default

  source:
    # Use public Helm repository
    repoURL: https://ssahani.github.io/hypersdk/helm-charts
    chart: hypersdk
    targetRevision: 0.2.0

    helm:
      # Custom values
      values: |
        replicaCount: 1

        credentials:
          vsphere:
            enabled: true
            existingSecret: vsphere-credentials

        persistence:
          data:
            size: 20Gi
          exports:
            size: 1Ti

        monitoring:
          serviceMonitor:
            enabled: true

  destination:
    server: https://kubernetes.default.svc
    namespace: hypersdk

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
      allowEmpty: false
    syncOptions:
      - CreateNamespace=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m
```

Apply the application:

```bash
kubectl apply -f argocd/hypersdk-app.yaml
```

### Method 2: Using Git Repository

Store values in Git and reference the chart:

```yaml
# argocd/hypersdk-git-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk
  namespace: argocd
spec:
  project: default

  source:
    repoURL: https://github.com/your-org/your-infra-repo
    targetRevision: main
    path: kubernetes/hypersdk

    helm:
      # Reference the Helm chart
      releaseName: hypersdk

      # Use values from Git
      valueFiles:
        - values.yaml
        - values-production.yaml

  destination:
    server: https://kubernetes.default.svc
    namespace: hypersdk

  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

Git repository structure:

```
your-infra-repo/
├── kubernetes/
│   └── hypersdk/
│       ├── Chart.yaml
│       ├── values.yaml              # Base values
│       ├── values-production.yaml   # Production overrides
│       └── requirements.yaml        # Chart dependencies
```

**Chart.yaml**:
```yaml
apiVersion: v2
name: hypersdk
version: 1.0.0
dependencies:
  - name: hypersdk
    version: 0.2.0
    repository: https://ssahani.github.io/hypersdk/helm-charts
```

**requirements.yaml** (alternative):
```yaml
dependencies:
  - name: hypersdk
    version: 0.2.0
    repository: https://ssahani.github.io/hypersdk/helm-charts
```

### Method 3: ArgoCD ApplicationSet

Deploy to multiple clusters/environments:

```yaml
# argocd/hypersdk-appset.yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: hypersdk
  namespace: argocd
spec:
  generators:
    # Generate applications for each cluster
    - list:
        elements:
          - cluster: dev
            url: https://dev.k8s.example.com
            values: values-dev.yaml
          - cluster: staging
            url: https://staging.k8s.example.com
            values: values-staging.yaml
          - cluster: production
            url: https://production.k8s.example.com
            values: values-production.yaml

  template:
    metadata:
      name: 'hypersdk-{{cluster}}'

    spec:
      project: default

      source:
        repoURL: https://ssahani.github.io/hypersdk/helm-charts
        chart: hypersdk
        targetRevision: 0.2.0

        helm:
          valueFiles:
            - '{{values}}'

      destination:
        server: '{{url}}'
        namespace: hypersdk

      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

### ArgoCD Best Practices

#### 1. Separate Secrets

Don't store secrets in Git. Use external secret management:

```yaml
# Use Sealed Secrets
apiVersion: bitnami.com/v1alpha1
kind: SealedSecret
metadata:
  name: vsphere-credentials
  namespace: hypersdk
spec:
  encryptedData:
    url: AgBx7Qw...
    username: AgBy8Rv...
    password: AgCz9Sx...
```

Or use External Secrets Operator:

```yaml
# Use External Secrets
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: vsphere-credentials
  namespace: hypersdk
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: vsphere-credentials
  data:
    - secretKey: url
      remoteRef:
        key: hypersdk/vsphere
        property: url
    - secretKey: username
      remoteRef:
        key: hypersdk/vsphere
        property: username
    - secretKey: password
      remoteRef:
        key: hypersdk/vsphere
        property: password
```

#### 2. Health Assessment

Configure custom health checks:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk
spec:
  # ... other fields ...

  # Custom health assessment
  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas  # Ignore if HPA manages replicas

  # Wait for these resources
  sync:
    hooks:
      - name: pre-sync-hook
        hookType: PreSync
```

#### 3. Progressive Rollout

Use ArgoCD sync waves for ordered deployment:

```yaml
# In Helm chart values
podAnnotations:
  argocd.argoproj.io/sync-wave: "1"  # Deploy pods after dependencies

# In dependent resources
annotations:
  argocd.argoproj.io/sync-wave: "0"  # Deploy first
```

## Flux Integration

### Prerequisites

- Flux installed in cluster
- Git repository for manifests
- GitHub/GitLab token configured

### Method 1: HelmRelease with Helm Repository

```yaml
# flux/hypersdk-source.yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  interval: 10m
  url: https://ssahani.github.io/hypersdk/helm-charts
---
# flux/hypersdk-release.yaml
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
      interval: 5m

  values:
    replicaCount: 1

    credentials:
      vsphere:
        enabled: true
        existingSecret: vsphere-credentials

    persistence:
      data:
        size: 20Gi
      exports:
        size: 1Ti

    monitoring:
      serviceMonitor:
        enabled: true

  install:
    createNamespace: true
    remediation:
      retries: 3

  upgrade:
    remediation:
      retries: 3
      remediateLastFailure: true
    cleanupOnFail: true

  rollback:
    recreate: true
    cleanupOnFail: true
```

Apply with Flux:

```bash
kubectl apply -f flux/hypersdk-source.yaml
kubectl apply -f flux/hypersdk-release.yaml
```

### Method 2: Kustomization with Git Source

```yaml
# flux/hypersdk-git-source.yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: hypersdk-config
  namespace: flux-system
spec:
  interval: 1m
  url: https://github.com/your-org/hypersdk-config
  ref:
    branch: main
---
# flux/hypersdk-kustomization.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  interval: 10m
  path: ./kubernetes/hypersdk
  prune: true
  sourceRef:
    kind: GitRepository
    name: hypersdk-config

  # Health checks
  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: hypersdk
      namespace: hypersdk

  # Wait for dependencies
  dependsOn:
    - name: sealed-secrets
```

### Method 3: Multi-Environment with Flux

```yaml
# flux/environments/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - hypersdk-source.yaml
  - hypersdk-release.yaml

# flux/environments/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
patchesStrategicMerge:
  - hypersdk-values.yaml

# flux/environments/production/hypersdk-values.yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  values:
    replicaCount: 3

    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10

    resources:
      requests:
        memory: 1Gi
        cpu: 500m
      limits:
        memory: 4Gi
        cpu: 2000m
```

### Flux Best Practices

#### 1. Automated Image Updates

```yaml
# flux/image-policy.yaml
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageRepository
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  image: ghcr.io/ssahani/hypersdk-hypervisord
  interval: 1m
---
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImagePolicy
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  imageRepositoryRef:
    name: hypersdk
  policy:
    semver:
      range: 0.2.x
---
apiVersion: image.toolkit.fluxcd.io/v1beta1
kind: ImageUpdateAutomation
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  interval: 1m
  sourceRef:
    kind: GitRepository
    name: flux-system
  git:
    checkout:
      ref:
        branch: main
    commit:
      author:
        email: fluxcdbot@users.noreply.github.com
        name: fluxcdbot
      messageTemplate: 'Update HyperSDK to {{range .Updated.Images}}{{println .}}{{end}}'
    push:
      branch: main
  update:
    path: ./kubernetes/hypersdk
    strategy: Setters
```

#### 2. Notifications

```yaml
# flux/notification.yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Provider
metadata:
  name: slack
  namespace: flux-system
spec:
  type: slack
  channel: deployments
  secretRef:
    name: slack-url
---
apiVersion: notification.toolkit.fluxcd.io/v1beta1
kind: Alert
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  providerRef:
    name: slack
  eventSeverity: info
  eventSources:
    - kind: HelmRelease
      name: '*'
      namespace: hypersdk
  summary: "HyperSDK deployment status"
```

## Complete GitOps Example

### Directory Structure

```
gitops-repo/
├── apps/
│   └── hypersdk/
│       ├── base/
│       │   ├── kustomization.yaml
│       │   ├── namespace.yaml
│       │   ├── helm-repository.yaml
│       │   └── helm-release.yaml
│       ├── overlays/
│       │   ├── development/
│       │   │   ├── kustomization.yaml
│       │   │   └── values.yaml
│       │   ├── staging/
│       │   │   ├── kustomization.yaml
│       │   │   └── values.yaml
│       │   └── production/
│       │       ├── kustomization.yaml
│       │       ├── values.yaml
│       │       └── sealed-secrets.yaml
│       └── README.md
├── infrastructure/
│   ├── sealed-secrets/
│   └── prometheus-operator/
└── clusters/
    ├── development/
    │   └── apps.yaml
    ├── staging/
    │   └── apps.yaml
    └── production/
        └── apps.yaml
```

### Base Configuration

```yaml
# apps/hypersdk/base/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: hypersdk
  labels:
    app: hypersdk

---
# apps/hypersdk/base/helm-repository.yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: HelmRepository
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  interval: 10m
  url: https://ssahani.github.io/hypersdk/helm-charts

---
# apps/hypersdk/base/helm-release.yaml
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
      version: '>=0.2.0 <1.0.0'
      sourceRef:
        kind: HelmRepository
        name: hypersdk
        namespace: hypersdk

  # Will be overridden by overlays
  values: {}

---
# apps/hypersdk/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - namespace.yaml
  - helm-repository.yaml
  - helm-release.yaml
```

### Production Overlay

```yaml
# apps/hypersdk/overlays/production/values.yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  values:
    replicaCount: 3

    image:
      tag: "0.2.0"

    credentials:
      vsphere:
        enabled: true
        existingSecret: vsphere-credentials-sealed

    persistence:
      data:
        storageClass: premium-rwo
        size: 50Gi
      exports:
        storageClass: premium-rwo
        size: 2Ti

    autoscaling:
      enabled: true
      minReplicas: 3
      maxReplicas: 10
      targetCPUUtilizationPercentage: 70

    resources:
      requests:
        memory: 1Gi
        cpu: 500m
      limits:
        memory: 4Gi
        cpu: 2000m

    monitoring:
      serviceMonitor:
        enabled: true

    networkPolicy:
      enabled: true

    ingress:
      enabled: true
      className: nginx
      hosts:
        - host: hypersdk.production.example.com
          paths:
            - path: /
              pathType: Prefix
      tls:
        - secretName: hypersdk-tls
          hosts:
            - hypersdk.production.example.com

---
# apps/hypersdk/overlays/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base
patchesStrategicMerge:
  - values.yaml
```

### Cluster Configuration

```yaml
# clusters/production/apps.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: hypersdk
  namespace: flux-system
spec:
  interval: 10m
  path: ./apps/hypersdk/overlays/production
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system

  healthChecks:
    - apiVersion: apps/v1
      kind: Deployment
      name: hypersdk
      namespace: hypersdk

  timeout: 5m

  # Wait for infrastructure
  dependsOn:
    - name: sealed-secrets
    - name: prometheus-operator
```

## Monitoring GitOps Deployments

### ArgoCD UI

Access ArgoCD UI:
```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

Navigate to: https://localhost:8080

### Flux CLI

Monitor Flux deployments:
```bash
# Check all HelmReleases
flux get helmreleases --all-namespaces

# Check specific release
flux get helmrelease hypersdk -n hypersdk

# Reconcile manually
flux reconcile helmrelease hypersdk -n hypersdk

# Suspend/resume
flux suspend helmrelease hypersdk -n hypersdk
flux resume helmrelease hypersdk -n hypersdk
```

## Troubleshooting

### ArgoCD Sync Issues

```bash
# Check application status
kubectl get application hypersdk -n argocd -o yaml

# Force sync
argocd app sync hypersdk

# View sync history
argocd app history hypersdk

# Rollback
argocd app rollback hypersdk <revision>
```

### Flux Reconciliation Issues

```bash
# Check HelmRelease status
kubectl describe helmrelease hypersdk -n hypersdk

# Check events
kubectl get events -n hypersdk --sort-by='.lastTimestamp'

# Force reconciliation
flux reconcile source helm hypersdk -n flux-system
flux reconcile helmrelease hypersdk -n hypersdk

# Debug
flux logs --level=debug
```

## Best Practices

1. **Version Pinning**: Pin chart versions in production
2. **Secret Management**: Use Sealed Secrets or External Secrets
3. **Progressive Delivery**: Deploy to dev → staging → production
4. **Health Checks**: Configure proper health assessments
5. **Notifications**: Set up alerts for deployment events
6. **Backup**: Regular backups of GitOps repositories
7. **RBAC**: Proper access controls for Git repositories
8. **Testing**: Validate manifests before committing

## Summary

GitOps with HyperSDK provides:
- ✅ Automated deployments from Git
- ✅ Audit trail of all changes
- ✅ Easy rollbacks
- ✅ Multi-cluster support
- ✅ Progressive delivery
- ✅ Self-healing deployments
- ✅ Secret management integration

Choose ArgoCD for:
- Rich UI experience
- Application-centric view
- Multi-tenancy

Choose Flux for:
- GitOps-native approach
- Automated image updates
- Tight Git integration
