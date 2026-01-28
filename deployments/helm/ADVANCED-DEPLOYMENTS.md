# Advanced Deployment Strategies for HyperSDK

This guide covers advanced deployment patterns for production environments including canary deployments, blue-green deployments, and progressive delivery.

## Overview

Advanced deployment strategies minimize risk during updates by:
- **Gradual rollout** - Expose changes to subset of users
- **Automated testing** - Validate in production with real traffic
- **Quick rollback** - Revert instantly if issues detected
- **Zero downtime** - Maintain service availability

## Canary Deployments

Gradually roll out changes to a small subset of users before full deployment.

### Using Flagger (with Istio/Linkerd)

#### Prerequisites

```bash
# Install Flagger
kubectl apply -k github.com/fluxcd/flagger//kustomize/istio

# Verify installation
kubectl -n istio-system rollout status deployment/flagger
```

#### Canary Configuration

```yaml
# canary/hypersdk-canary.yaml
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  # Target deployment
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hypersdk

  # Service
  service:
    port: 8080
    targetPort: 8080
    gateways:
      - istio-system/public-gateway
    hosts:
      - hypersdk.example.com

  # Analysis
  analysis:
    # Schedule interval (default 60s)
    interval: 1m
    # Max number of failed metric checks before rollback
    threshold: 5
    # Max traffic percentage routed to canary
    maxWeight: 50
    # Canary increment step
    stepWeight: 10

    # Metrics for canary analysis
    metrics:
      - name: request-success-rate
        # Minimum req success rate (non 5xx responses)
        thresholdRange:
          min: 99
        interval: 1m

      - name: request-duration
        # Maximum req duration P99
        thresholdRange:
          max: 500
        interval: 1m

    # Webhooks for additional validation
    webhooks:
      - name: load-test
        url: http://flagger-loadtester/
        timeout: 5s
        metadata:
          type: cmd
          cmd: "hey -z 1m -q 10 -c 2 http://hypersdk-canary.hypersdk:8080/health"

      - name: acceptance-test
        type: pre-rollout
        url: http://flagger-loadtester/
        timeout: 10s
        metadata:
          type: bash
          cmd: "curl -f http://hypersdk-canary.hypersdk:8080/health | grep -q ok"

  # Progressive traffic shifting
  canaryAnalysis:
    # Start after this many successful checks
    iterations: 10
    # Match conditions
    match:
      - headers:
          x-canary:
            exact: "insider"
    # Mirror traffic to canary
    mirror: false
    # Weight increment on each iteration
    stepWeight: 10
    # Max weight for canary
    maxWeight: 50
```

#### Deploy with Canary

```bash
# Install HyperSDK with Flagger
helm install hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --create-namespace

# Apply canary configuration
kubectl apply -f canary/hypersdk-canary.yaml

# Trigger canary deployment by updating image
kubectl -n hypersdk set image deployment/hypersdk \
  hypersdk=ghcr.io/ssahani/hypersdk-hypervisord:0.3.0

# Watch canary progress
watch kubectl -n hypersdk get canary hypersdk

# Check events
kubectl -n hypersdk describe canary hypersdk
```

#### Canary Promotion Flow

```
1. New version deployed as canary (0% traffic)
2. Run pre-rollout tests
3. Gradually increase traffic: 10% → 20% → 30% → 40% → 50%
4. At each step: analyze metrics, run load tests
5. If all checks pass: promote canary to primary
6. If any check fails: automatic rollback
```

### Using Argo Rollouts

#### Install Argo Rollouts

```bash
kubectl create namespace argo-rollouts
kubectl apply -n argo-rollouts -f https://github.com/argoproj/argo-rollouts/releases/latest/download/install.yaml

# Install kubectl plugin
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
chmod +x kubectl-argo-rollouts-linux-amd64
sudo mv kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts
```

#### Argo Rollout Configuration

```yaml
# rollouts/hypersdk-rollout.yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  replicas: 5

  # Strategy: Canary
  strategy:
    canary:
      # Max surge during update
      maxSurge: 1
      # Max unavailable during update
      maxUnavailable: 0

      # Canary steps
      steps:
        - setWeight: 20
        - pause: {duration: 5m}
        - setWeight: 40
        - pause: {duration: 5m}
        - setWeight: 60
        - pause: {duration: 5m}
        - setWeight: 80
        - pause: {duration: 5m}

      # Traffic routing (using Istio)
      trafficRouting:
        istio:
          virtualService:
            name: hypersdk
            routes:
              - primary

      # Analysis during canary
      analysis:
        templates:
          - templateName: success-rate
        startingStep: 2
        args:
          - name: service-name
            value: hypersdk

  # Deployment template
  selector:
    matchLabels:
      app: hypersdk

  template:
    metadata:
      labels:
        app: hypersdk
    spec:
      containers:
        - name: hypersdk
          image: ghcr.io/ssahani/hypersdk-hypervisord:0.2.0
          ports:
            - containerPort: 8080
          # ... rest of container spec

---
# Analysis template
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
  namespace: hypersdk
spec:
  args:
    - name: service-name
  metrics:
    - name: success-rate
      interval: 1m
      count: 5
      successCondition: result[0] >= 0.95
      provider:
        prometheus:
          address: http://prometheus.monitoring:9090
          query: |
            sum(rate(
              http_requests_total{
                service="{{args.service-name}}",
                status!~"5.."
              }[1m]
            )) /
            sum(rate(
              http_requests_total{
                service="{{args.service-name}}"
              }[1m]
            ))
```

#### Deploy with Argo Rollouts

```bash
# Apply rollout
kubectl apply -f rollouts/hypersdk-rollout.yaml

# Update image (triggers canary)
kubectl argo rollouts set image hypersdk \
  hypersdk=ghcr.io/ssahani/hypersdk-hypervisord:0.3.0 \
  -n hypersdk

# Watch rollout
kubectl argo rollouts get rollout hypersdk -n hypersdk --watch

# Promote canary manually
kubectl argo rollouts promote hypersdk -n hypersdk

# Abort rollout
kubectl argo rollouts abort hypersdk -n hypersdk
```

## Blue-Green Deployments

Run two identical production environments, switching traffic between them.

### Using Argo Rollouts

```yaml
# rollouts/hypersdk-bluegreen.yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  replicas: 5

  # Strategy: Blue-Green
  strategy:
    blueGreen:
      # Service for active (blue)
      activeService: hypersdk-active
      # Service for preview (green)
      previewService: hypersdk-preview

      # Auto promotion after analysis
      autoPromotionEnabled: false
      # Auto promotion time
      autoPromotionSeconds: 300

      # Pre-promotion analysis
      prePromotionAnalysis:
        templates:
          - templateName: smoke-tests
        args:
          - name: service-name
            value: hypersdk-preview

      # Post-promotion analysis
      postPromotionAnalysis:
        templates:
          - templateName: success-rate
        args:
          - name: service-name
            value: hypersdk-active

      # Scale down delay after promotion
      scaleDownDelaySeconds: 30
      # Scale down delay after abort
      scaleDownDelayRevisionLimit: 2

  selector:
    matchLabels:
      app: hypersdk

  template:
    metadata:
      labels:
        app: hypersdk
    spec:
      containers:
        - name: hypersdk
          image: ghcr.io/ssahani/hypersdk-hypervisord:0.2.0
          # ... container spec

---
# Active service (blue)
apiVersion: v1
kind: Service
metadata:
  name: hypersdk-active
  namespace: hypersdk
spec:
  selector:
    app: hypersdk
  ports:
    - port: 8080
      targetPort: 8080

---
# Preview service (green)
apiVersion: v1
kind: Service
metadata:
  name: hypersdk-preview
  namespace: hypersdk
spec:
  selector:
    app: hypersdk
  ports:
    - port: 8080
      targetPort: 8080

---
# Smoke tests analysis
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: smoke-tests
  namespace: hypersdk
spec:
  metrics:
    - name: smoke-tests
      count: 1
      provider:
        job:
          spec:
            template:
              spec:
                containers:
                  - name: smoke
                    image: curlimages/curl:latest
                    command:
                      - sh
                      - -c
                      - |
                        set -e
                        # Health check
                        curl -f http://hypersdk-preview.hypersdk:8080/health
                        # Status check
                        curl -f http://hypersdk-preview.hypersdk:8080/api/v1/status
                        # Capabilities check
                        curl -f http://hypersdk-preview.hypersdk:8080/api/v1/capabilities
                        echo "All smoke tests passed"
                restartPolicy: Never
            backoffLimit: 1
```

### Deploy Blue-Green

```bash
# Apply blue-green rollout
kubectl apply -f rollouts/hypersdk-bluegreen.yaml

# Update image (deploys to green)
kubectl argo rollouts set image hypersdk \
  hypersdk=ghcr.io/ssahani/hypersdk-hypervisord:0.3.0 \
  -n hypersdk

# Check green environment
kubectl argo rollouts get rollout hypersdk -n hypersdk

# Access preview service for testing
kubectl port-forward svc/hypersdk-preview -n hypersdk 8081:8080

# Run manual tests against http://localhost:8081

# Promote green to blue
kubectl argo rollouts promote hypersdk -n hypersdk

# Or abort and rollback
kubectl argo rollouts abort hypersdk -n hypersdk
```

## Progressive Delivery with GitOps

### Flagger + Flux

```yaml
# flux/hypersdk-canary.yaml
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  provider: istio

  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hypersdk

  progressDeadlineSeconds: 600

  service:
    port: 8080
    portDiscovery: true

  analysis:
    interval: 30s
    threshold: 5
    maxWeight: 50
    stepWeight: 5

    metrics:
      - name: request-success-rate
        thresholdRange:
          min: 99
        interval: 1m

      - name: request-duration
        thresholdRange:
          max: 500
        interval: 1m

    webhooks:
      - name: load-test
        url: http://flagger-loadtester.test/
        timeout: 5s
        metadata:
          cmd: "hey -z 1m -q 10 -c 2 http://hypersdk-canary.hypersdk:8080/"

---
# HelmRelease with canary
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  interval: 5m
  chart:
    spec:
      chart: hypersdk
      version: '>=0.2.0'
      sourceRef:
        kind: HelmRepository
        name: hypersdk

  # Flagger takes over deployment
  install:
    createNamespace: true
  upgrade:
    remediation:
      remediateLastFailure: true
```

### Argo Rollouts + ArgoCD

```yaml
# argocd/hypersdk-app-rollouts.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk
  namespace: argocd
spec:
  project: default

  source:
    repoURL: https://github.com/your-org/hypersdk-config
    targetRevision: main
    path: kubernetes/hypersdk

  destination:
    server: https://kubernetes.default.svc
    namespace: hypersdk

  syncPolicy:
    automated:
      prune: true
      selfHeal: true

    syncOptions:
      - CreateNamespace=true
      # Important: Let Argo Rollouts manage replicas
      - RespectIgnoreDifferences=true

  # Ignore differences in rollout status
  ignoreDifferences:
    - group: argoproj.io
      kind: Rollout
      jsonPointers:
        - /spec/replicas
```

## Feature Flags with HyperSDK

### Using LaunchDarkly

```yaml
# Add feature flag sidecar
spec:
  template:
    spec:
      containers:
        - name: hypersdk
          # ... main container

        - name: launchdarkly-relay
          image: launchdarkly/ld-relay:latest
          env:
            - name: LD_ENV_<environment>
              valueFrom:
                secretKeyRef:
                  name: launchdarkly
                  key: sdk-key
          ports:
            - containerPort: 8030
```

### Using Flagsmith

```yaml
# ConfigMap with feature flags
apiVersion: v1
kind: ConfigMap
metadata:
  name: hypersdk-features
data:
  features.yaml: |
    features:
      new-export-engine:
        enabled: false
        rollout: 10  # 10% of users

      enhanced-monitoring:
        enabled: true
        rollout: 100
```

## Traffic Shadowing (Mirroring)

Test new versions with production traffic without affecting users.

### Using Istio

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  hosts:
    - hypersdk.example.com
  http:
    - match:
        - uri:
            prefix: /
      route:
        - destination:
            host: hypersdk-stable
            port:
              number: 8080
          weight: 100
      mirror:
        host: hypersdk-canary
        port:
          number: 8080
      mirrorPercentage:
        value: 10.0  # Mirror 10% of traffic
```

## A/B Testing

Route traffic based on user attributes.

```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: hypersdk-ab
  namespace: hypersdk
spec:
  hosts:
    - hypersdk.example.com
  http:
    # Version B for beta users
    - match:
        - headers:
            x-user-group:
              exact: beta
      route:
        - destination:
            host: hypersdk
            subset: v2
          weight: 100

    # Version A for everyone else
    - route:
        - destination:
            host: hypersdk
            subset: v1
          weight: 100

---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  host: hypersdk
  subsets:
    - name: v1
      labels:
        version: v1
    - name: v2
      labels:
        version: v2
```

## Deployment Automation

### GitHub Actions with Progressive Delivery

```yaml
# .github/workflows/progressive-deploy.yml
name: Progressive Deployment

on:
  push:
    tags:
      - 'v*.*.*'

jobs:
  deploy-canary:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to canary
        run: |
          kubectl argo rollouts set image hypersdk \
            hypersdk=${IMAGE}:${TAG} \
            -n hypersdk

      - name: Wait for analysis
        run: |
          kubectl argo rollouts status hypersdk -n hypersdk

  promote-or-rollback:
    needs: deploy-canary
    runs-on: ubuntu-latest
    steps:
      - name: Check metrics
        id: metrics
        run: |
          # Query Prometheus for error rate
          ERROR_RATE=$(curl -s "http://prometheus/api/v1/query?query=error_rate" | jq '.data.result[0].value[1]')

          if [ "${ERROR_RATE}" -lt "0.01" ]; then
            echo "promote=true" >> $GITHUB_OUTPUT
          else
            echo "promote=false" >> $GITHUB_OUTPUT
          fi

      - name: Promote or abort
        run: |
          if [ "${{ steps.metrics.outputs.promote }}" = "true" ]; then
            kubectl argo rollouts promote hypersdk -n hypersdk
          else
            kubectl argo rollouts abort hypersdk -n hypersdk
          fi
```

## Best Practices

### 1. Start Small

```yaml
# Conservative canary steps
steps:
  - setWeight: 5    # Start with 5%
  - pause: {duration: 10m}
  - setWeight: 10
  - pause: {duration: 10m}
  - setWeight: 25
  - pause: {duration: 10m}
```

### 2. Monitor Everything

```yaml
metrics:
  - name: request-success-rate
  - name: request-duration
  - name: cpu-usage
  - name: memory-usage
  - name: error-rate
  - name: custom-business-metric
```

### 3. Automated Rollback

```yaml
analysis:
  threshold: 3  # Fail after 3 bad checks
  interval: 1m
```

### 4. Manual Gates for Critical Changes

```yaml
steps:
  - setWeight: 50
  - pause: {}  # Manual approval required
```

### 5. Test in Preview

Always test preview/canary before promotion.

## Summary

Advanced deployment strategies provide:

✅ **Risk Reduction** - Gradual rollout minimizes blast radius
✅ **Automated Testing** - Validation with real production traffic
✅ **Quick Rollback** - Instant revert on issues
✅ **Zero Downtime** - Maintain availability during updates
✅ **Confidence** - Deploy frequently with safety

Choose the right strategy:
- **Canary**: Gradual percentage-based rollout
- **Blue-Green**: Complete environment switch
- **Shadow**: Test without user impact
- **A/B**: Test different versions simultaneously
