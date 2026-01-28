# Cost Optimization Guide for HyperSDK

Comprehensive strategies for optimizing Kubernetes infrastructure costs while maintaining performance and reliability.

## Overview

This guide covers cost optimization across:

- **Resource Right-Sizing** - Optimize CPU/memory requests and limits
- **Storage Optimization** - Reduce storage costs
- **Auto-Scaling** - Scale resources based on demand
- **Cluster Optimization** - Optimize node pools and instance types
- **Cost Monitoring** - Track and analyze spending
- **Cloud Provider Optimizations** - Platform-specific savings

## Cost Analysis Framework

### Current Cost Breakdown

Typical HyperSDK deployment costs:

| Component | Monthly Cost | Percentage | Optimization Potential |
|-----------|-------------|------------|----------------------|
| Compute (Nodes) | $500-2000 | 50-60% | High |
| Storage (PVC) | $200-800 | 20-30% | Medium |
| Network (Egress) | $100-300 | 10-15% | Medium |
| Load Balancers | $50-100 | 5-10% | Low |
| **Total** | **$850-3200** | **100%** | - |

### Cost Drivers

1. **Over-provisioned resources** (40% waste)
2. **Always-on development environments** (30% waste)
3. **Inefficient storage** (20% waste)
4. **Network egress** (10% waste)

## Resource Right-Sizing

### Analyze Current Usage

```bash
# Check actual resource usage
kubectl top pods -n hypersdk

# Get resource requests vs limits
kubectl describe pod -n hypersdk -l app.kubernetes.io/name=hypersdk | \
  grep -A 5 "Requests\|Limits"

# Analyze over 7 days (requires metrics-server)
kubectl get --raw "/apis/metrics.k8s.io/v1beta1/namespaces/hypersdk/pods" | \
  jq '.items[] | {name:.metadata.name, cpu:.containers[].usage.cpu, memory:.containers[].usage.memory}'
```

### Right-Sizing Recommendations

#### Development Environment

```yaml
# values-dev.yaml - Minimal resources
resources:
  requests:
    memory: 256Mi
    cpu: 100m
  limits:
    memory: 512Mi
    cpu: 500m

replicaCount: 1

persistence:
  data:
    size: 1Gi
  exports:
    size: 10Gi

autoscaling:
  enabled: false
```

**Estimated Monthly Cost**: $50-100

#### Staging Environment

```yaml
# values-staging.yaml - Medium resources
resources:
  requests:
    memory: 512Mi
    cpu: 250m
  limits:
    memory: 2Gi
    cpu: 1000m

replicaCount: 2

persistence:
  data:
    size: 5Gi
  exports:
    size: 100Gi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
```

**Estimated Monthly Cost**: $200-400

#### Production Environment

```yaml
# values-production.yaml - Optimized resources
resources:
  requests:
    memory: 1Gi
    cpu: 500m
  limits:
    memory: 4Gi
    cpu: 2000m

replicaCount: 3

persistence:
  data:
    size: 10Gi
    storageClass: standard  # Use cheaper tier
  exports:
    size: 500Gi
    storageClass: standard  # Not performance-critical

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
```

**Estimated Monthly Cost**: $500-1000

### Vertical Pod Autoscaler (VPA)

```yaml
# Install VPA
kubectl apply -f https://github.com/kubernetes/autoscaler/releases/download/vertical-pod-autoscaler-0.13.0/vpa-v0.13.0.yaml

---
# VPA for HyperSDK
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: hypersdk-vpa
  namespace: hypersdk
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hypersdk
  updatePolicy:
    updateMode: "Auto"  # or "Recreate", "Initial", "Off"
  resourcePolicy:
    containerPolicies:
      - containerName: hypersdk
        minAllowed:
          cpu: 100m
          memory: 256Mi
        maxAllowed:
          cpu: 2000m
          memory: 8Gi
        controlledResources: ["cpu", "memory"]
```

## Storage Optimization

### Storage Class Selection

```yaml
# Use appropriate storage classes
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-data
  namespace: hypersdk
spec:
  # Fast SSD for database (20% of storage cost)
  storageClassName: premium-ssd
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-exports
  namespace: hypersdk
spec:
  # Cheaper HDD for exports (80% of storage cost)
  storageClassName: standard
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 500Gi
```

### Storage Lifecycle Management

```bash
# Cleanup script for old exports
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup-old-exports
  namespace: hypersdk
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: cleanup
              image: alpine:3.19
              command:
                - sh
                - -c
                - |
                  # Delete exports older than 7 days
                  find /exports -type f -mtime +7 -delete
                  # Delete empty directories
                  find /exports -type d -empty -delete
              volumeMounts:
                - name: exports
                  mountPath: /exports
          volumes:
            - name: exports
              persistentVolumeClaim:
                claimName: hypersdk-exports
          restartPolicy: OnFailure
EOF
```

### Storage Compression

```yaml
# Enable compression in HyperSDK config
apiVersion: v1
kind: ConfigMap
metadata:
  name: hypersdk
  namespace: hypersdk
data:
  config.yaml: |
    export:
      compress: true
      compression_level: 6  # Balance between size and CPU
      format: gzip

    cleanup:
      enabled: true
      retention_days: 7
      compress_old_exports: true  # Compress after 2 days
      compress_after_days: 2
```

### Cloud-Specific Storage Optimization

#### AWS EBS

```yaml
# Use gp3 instead of gp2 (20% cheaper)
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
allowVolumeExpansion: true

---
# Use st1 (throughput-optimized HDD) for exports (75% cheaper)
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: st1
provisioner: ebs.csi.aws.com
parameters:
  type: st1
allowVolumeExpansion: true
```

**Savings**: 40-50% on storage costs

#### Azure Disk

```yaml
# Use Standard SSD instead of Premium SSD for data
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: managed-standard-ssd
provisioner: disk.csi.azure.com
parameters:
  skuName: StandardSSD_LRS

---
# Use Standard HDD for exports
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: managed-standard-hdd
provisioner: disk.csi.azure.com
parameters:
  skuName: Standard_LRS
```

**Savings**: 50-60% on storage costs

#### GCP Persistent Disk

```yaml
# Use pd-standard instead of pd-ssd for exports
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard-rwo
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-standard
  replication-type: regional-pd
```

**Savings**: 70% on storage costs

## Auto-Scaling Optimization

### Horizontal Pod Autoscaling (HPA)

```yaml
# Optimized HPA configuration
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hypersdk
  namespace: hypersdk
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hypersdk
  minReplicas: 2  # Minimum for HA
  maxReplicas: 10
  metrics:
    # Scale on CPU (primary)
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70

    # Scale on memory (secondary)
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80

    # Scale on custom metric (active jobs)
    - type: Pods
      pods:
        metric:
          name: hypersdk_active_jobs
        target:
          type: AverageValue
          averageValue: "5"

  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300  # Wait 5 min before scaling down
      policies:
        - type: Percent
          value: 50  # Scale down max 50% at a time
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0  # Scale up immediately
      policies:
        - type: Percent
          value: 100  # Double capacity if needed
          periodSeconds: 15
        - type: Pods
          value: 2  # Add max 2 pods at a time
          periodSeconds: 15
      selectPolicy: Max
```

### Cluster Autoscaler

```yaml
# AWS EKS - Node group with autoscaling
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig
metadata:
  name: hypersdk-cluster
  region: us-east-1

managedNodeGroups:
  # On-demand nodes for baseline
  - name: baseline
    instanceType: t3.medium
    minSize: 2
    maxSize: 3
    desiredCapacity: 2
    labels:
      workload-type: baseline
    tags:
      k8s.io/cluster-autoscaler/enabled: "true"

  # Spot instances for burst capacity
  - name: burst
    instanceTypes: ["t3.medium", "t3a.medium", "t2.medium"]
    spot: true
    minSize: 0
    maxSize: 10
    desiredCapacity: 0
    labels:
      workload-type: burst
    taints:
      - key: spot
        value: "true"
        effect: NoSchedule
    tags:
      k8s.io/cluster-autoscaler/enabled: "true"

---
# Allow HyperSDK on spot instances (toleration)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hypersdk
spec:
  template:
    spec:
      tolerations:
        - key: spot
          operator: Equal
          value: "true"
          effect: NoSchedule
      affinity:
        nodeAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 50
              preference:
                matchExpressions:
                  - key: workload-type
                    operator: In
                    values:
                      - burst
```

**Savings**: 60-70% on compute (spot instances)

### KEDA (Event-Driven Autoscaling)

```yaml
# Install KEDA
helm repo add kedacore https://kedacore.github.io/charts
helm install keda kedacore/keda --namespace keda --create-namespace

---
# Scale based on job queue length
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: hypersdk-scaler
  namespace: hypersdk
spec:
  scaleTargetRef:
    name: hypersdk
  minReplicaCount: 1  # Scale to zero when idle (cost savings!)
  maxReplicaCount: 10
  pollingInterval: 30
  cooldownPeriod: 300
  triggers:
    # Scale based on Prometheus metric
    - type: prometheus
      metadata:
        serverAddress: http://prometheus.monitoring:9090
        metricName: hypersdk_pending_jobs
        threshold: '5'
        query: sum(hypersdk_pending_jobs)
```

**Savings**: Scale to 1 replica during off-hours (50% savings)

## Cluster Optimization

### Node Pool Strategy

#### Multi-Tier Node Pools

```yaml
# GKE - Cost-optimized node pools
resource "google_container_node_pool" "general_purpose" {
  name       = "general-purpose"
  cluster    = google_container_cluster.primary.name

  node_config {
    machine_type = "n1-standard-2"
    preemptible  = false  # On-demand for reliability
  }

  autoscaling {
    min_node_count = 2
    max_node_count = 5
  }
}

resource "google_container_node_pool" "spot_burst" {
  name       = "spot-burst"
  cluster    = google_container_cluster.primary.name

  node_config {
    machine_type = "n1-standard-4"
    spot         = true  # 60-90% cheaper
  }

  autoscaling {
    min_node_count = 0
    max_node_count = 10
  }
}

resource "google_container_node_pool" "high_memory" {
  name       = "high-memory"
  cluster    = google_container_cluster.primary.name

  node_config {
    machine_type = "n1-highmem-2"
    spot         = true
  }

  autoscaling {
    min_node_count = 0
    max_node_count = 3
  }

  # Only for large exports
  node_taints {
    key    = "workload"
    value  = "large-export"
    effect = "NO_SCHEDULE"
  }
}
```

### Node Affinity for Cost Optimization

```yaml
# Prefer spot instances
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hypersdk
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          # Prefer spot nodes (cheaper)
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              preference:
                matchExpressions:
                  - key: cloud.google.com/gke-spot
                    operator: In
                    values:
                      - "true"
          # Fallback to on-demand if no spot available
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: workload-type
                    operator: In
                    values:
                      - general-purpose
                      - spot-burst
```

### Bin Packing

```yaml
# Cluster autoscaler config for bin packing
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-autoscaler-priority-expander
  namespace: kube-system
data:
  priorities: |
    10:
      - .*-spot-.*   # Prefer spot instances
    5:
      - .*-standard-.* # Then standard
    1:
      - .*-premium-.* # Last resort
```

## Scheduled Scaling

### Scale Down Non-Production During Off-Hours

```yaml
# CronJob to scale down development at night
apiVersion: batch/v1
kind: CronJob
metadata:
  name: scale-down-dev
  namespace: hypersdk
spec:
  schedule: "0 19 * * 1-5"  # 7 PM weekdays
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: scaler
          containers:
            - name: kubectl
              image: bitnami/kubectl:latest
              command:
                - /bin/sh
                - -c
                - |
                  kubectl scale deployment/hypersdk --replicas=0 -n hypersdk-dev
          restartPolicy: OnFailure

---
# CronJob to scale up development in morning
apiVersion: batch/v1
kind: CronJob
metadata:
  name: scale-up-dev
  namespace: hypersdk
spec:
  schedule: "0 8 * * 1-5"  # 8 AM weekdays
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: scaler
          containers:
            - name: kubectl
              image: bitnami/kubectl:latest
              command:
                - /bin/sh
                - -c
                - |
                  kubectl scale deployment/hypersdk --replicas=2 -n hypersdk-dev
          restartPolicy: OnFailure

---
# ServiceAccount with permissions
apiVersion: v1
kind: ServiceAccount
metadata:
  name: scaler
  namespace: hypersdk

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: scaler
  namespace: hypersdk
rules:
  - apiGroups: ["apps"]
    resources: ["deployments", "deployments/scale"]
    verbs: ["get", "patch", "update"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: scaler
  namespace: hypersdk
subjects:
  - kind: ServiceAccount
    name: scaler
roleRef:
  kind: Role
  name: scaler
  apiGroup: rbac.authorization.k8s.io
```

**Savings**: 50% reduction in dev/staging costs (13 hours/day * 5 days = 65 hours/week saved)

## Network Cost Optimization

### Reduce Egress Traffic

```yaml
# Use VPC endpoints (AWS)
# Avoid internet egress charges for AWS services

# Configure HyperSDK to use VPC endpoints
apiVersion: v1
kind: ConfigMap
metadata:
  name: hypersdk
data:
  config.yaml: |
    aws:
      s3_endpoint: https://bucket.vpce-xxx.s3.us-east-1.vpce.amazonaws.com
      ec2_endpoint: https://vpce-xxx.ec2.us-east-1.vpce.amazonaws.com
```

### Regional Traffic

```yaml
# Keep traffic within region
apiVersion: v1
kind: Service
metadata:
  name: hypersdk
  annotations:
    # AWS - internal load balancer
    service.beta.kubernetes.io/aws-load-balancer-internal: "true"
    # GCP - internal load balancer
    cloud.google.com/load-balancer-type: "Internal"
    # Azure - internal load balancer
    service.beta.kubernetes.io/azure-load-balancer-internal: "true"
spec:
  type: LoadBalancer
```

### Content Delivery Network (CDN)

```yaml
# Use CloudFront/CloudFlare for static assets
# Cache API responses when possible

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hypersdk
  annotations:
    # Enable caching
    nginx.ingress.kubernetes.io/proxy-cache-valid: "200 302 10m"
    nginx.ingress.kubernetes.io/proxy-cache-valid: "404 1m"
spec:
  rules:
    - host: hypersdk.example.com
      http:
        paths:
          - path: /api/v1/status
            pathType: Prefix
            backend:
              service:
                name: hypersdk
                port:
                  number: 8080
```

## Cost Monitoring

### Kubernetes Cost Analysis Tools

#### Kubecost

```bash
# Install Kubecost
helm repo add kubecost https://kubecost.github.io/cost-analyzer/
helm install kubecost kubecost/cost-analyzer \
  --namespace kubecost \
  --create-namespace \
  --set kubecostToken="your-token"

# Access dashboard
kubectl port-forward -n kubecost svc/kubecost-cost-analyzer 9090:9090
# Open http://localhost:9090
```

#### OpenCost

```bash
# Install OpenCost (open-source)
kubectl apply -f https://raw.githubusercontent.com/opencost/opencost/develop/kubernetes/opencost.yaml

# Access UI
kubectl port-forward -n opencost svc/opencost 9003:9003
```

### Cost Allocation Labels

```yaml
# Add cost allocation labels
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hypersdk
  labels:
    app.kubernetes.io/name: hypersdk
    app.kubernetes.io/component: api
    cost-center: engineering
    environment: production
    team: platform
spec:
  template:
    metadata:
      labels:
        app.kubernetes.io/name: hypersdk
        cost-center: engineering
        environment: production
        team: platform
```

### Prometheus Cost Metrics

```yaml
# Track cost-related metrics
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: hypersdk-cost-metrics
spec:
  groups:
    - name: hypersdk.cost
      interval: 1h
      rules:
        - record: hypersdk:cost:cpu_hours
          expr: |
            sum(rate(container_cpu_usage_seconds_total{namespace="hypersdk"}[1h])) * 3600

        - record: hypersdk:cost:memory_gb_hours
          expr: |
            sum(container_memory_usage_bytes{namespace="hypersdk"}) / 1024 / 1024 / 1024

        - record: hypersdk:cost:storage_gb_hours
          expr: |
            sum(kubelet_volume_stats_used_bytes{namespace="hypersdk"}) / 1024 / 1024 / 1024
```

## Cloud Provider Specific Optimizations

### AWS Cost Optimization

```yaml
# 1. Use Savings Plans (72% discount vs on-demand)
# Purchase 1-year or 3-year compute savings plans

# 2. Reserved Instances for baseline capacity
# Reserve 50-70% of baseline capacity

# 3. Spot instances for burst (70-90% discount)
# Already covered in autoscaling section

# 4. EBS optimization
# - Use gp3 instead of gp2 (20% cheaper)
# - Use st1 for large sequential workloads (75% cheaper)
# - Delete unattached volumes

# 5. EKS Fargate for serverless (pay per pod)
apiVersion: v1
kind: Namespace
metadata:
  name: hypersdk-batch
  labels:
    fargate: "true"
```

**Potential Savings**: 60-75% on compute

### Azure Cost Optimization

```yaml
# 1. Azure Reserved VM Instances (72% discount)
# 2. Azure Spot VMs (90% discount)
# 3. Azure Hybrid Benefit (Windows licensing)
# 4. Scale down/up based on schedule

# AKS node pool with spot instances
resource "azurerm_kubernetes_cluster_node_pool" "spot" {
  name                  = "spot"
  kubernetes_cluster_id = azurerm_kubernetes_cluster.aks.id
  vm_size              = "Standard_D2s_v3"
  priority             = "Spot"
  eviction_policy      = "Delete"
  spot_max_price       = 0.05  # Max price per hour
  enable_auto_scaling  = true
  min_count            = 0
  max_count            = 10
}
```

**Potential Savings**: 60-70% on compute

### GCP Cost Optimization

```yaml
# 1. Committed use discounts (57% discount)
# 2. Preemptible VMs (80% discount)
# 3. Sustained use discounts (automatic 30% discount)
# 4. Custom machine types (right-size CPU/memory ratio)

# GKE autopilot (managed, optimized pricing)
resource "google_container_cluster" "autopilot" {
  name     = "hypersdk-autopilot"
  location = "us-central1"

  enable_autopilot = true  # Google manages optimization

  # Autopilot automatically:
  # - Right-sizes node pools
  # - Uses preemptible nodes where possible
  # - Bins packs efficiently
  # - Scales to zero
}
```

**Potential Savings**: 50-65% on compute

## Cost Optimization Checklist

### Quick Wins (Implement Today)

- [ ] Right-size resource requests (analyze with kubectl top)
- [ ] Use cheaper storage classes for non-critical data
- [ ] Enable HPA to scale down during low usage
- [ ] Delete unused PVCs and snapshots
- [ ] Scale down dev/staging environments at night
- [ ] Review and delete unattached resources

**Expected Savings**: 20-30%

### Medium-Term (This Month)

- [ ] Implement spot/preemptible instances for burst capacity
- [ ] Set up cluster autoscaler
- [ ] Implement storage lifecycle policies
- [ ] Add cost allocation labels
- [ ] Install Kubecost or OpenCost
- [ ] Review and optimize network egress

**Expected Savings**: 40-50%

### Long-Term (This Quarter)

- [ ] Purchase reserved instances/savings plans
- [ ] Implement KEDA for scale-to-zero
- [ ] Optimize multi-region architecture
- [ ] Implement automated cost anomaly detection
- [ ] Regular cost review meetings
- [ ] FinOps culture and training

**Expected Savings**: 60-70%

## Cost Optimization Results

### Before Optimization

```
Monthly Cost Breakdown:
- Compute: $2000 (60%)
- Storage: $800 (24%)
- Network: $300 (9%)
- Load Balancers: $100 (3%)
- Other: $100 (3%)
Total: $3,200/month
```

### After Optimization

```
Monthly Cost Breakdown:
- Compute: $700 (58%)  # Spot instances + right-sizing + HPA
- Storage: $300 (25%)  # Storage classes + lifecycle
- Network: $150 (12%)  # VPC endpoints + regional traffic
- Load Balancers: $50 (4%)   # Internal LB
- Other: $10 (1%)      # Cleanup
Total: $1,210/month

Savings: $1,990/month (62% reduction)
Annual Savings: $23,880
```

## Summary

### Cost Optimization Strategy

1. **Measure First** - Install Kubecost/OpenCost, analyze current spending
2. **Quick Wins** - Right-size resources, enable autoscaling
3. **Automate** - Schedule scaling, cleanup jobs, automated optimization
4. **Commit** - Reserved instances for baseline capacity
5. **Optimize Continuously** - Monthly cost reviews, anomaly detection

### Best Practices

✅ **Resource Efficiency**
- Right-size CPU/memory requests
- Use VPA for automatic right-sizing
- Enable HPA for demand-based scaling

✅ **Storage Optimization**
- Choose appropriate storage classes
- Implement lifecycle policies
- Enable compression

✅ **Cluster Efficiency**
- Use spot/preemptible instances
- Cluster autoscaler with bin packing
- Multi-tier node pools

✅ **Cost Visibility**
- Cost allocation labels
- Regular cost reports
- Anomaly detection alerts

✅ **FinOps Culture**
- Cost-aware engineering
- Budget alerts and limits
- Regular optimization reviews

### ROI Calculator

```
Initial Investment:
- Engineering time: 40 hours @ $100/hr = $4,000
- Tools (Kubecost): $50/month

Monthly Savings: $1,990
Payback Period: 2 months
Annual ROI: 597%
```

---

**Cost optimization is an ongoing process. Review monthly and adjust strategies based on usage patterns.**
