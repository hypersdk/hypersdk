# HyperSDK Helm - Quick Reference Card

Essential commands and configurations for daily operations.

## Installation

```bash
# From Helm repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm install hypersdk hypersdk/hypersdk -n hypersdk --create-namespace

# From OCI registry
helm install hypersdk oci://ghcr.io/ssahani/charts/hypersdk --version 0.2.0 -n hypersdk --create-namespace

# With custom values
helm install hypersdk hypersdk/hypersdk -f values.yaml -n hypersdk --create-namespace

# Dry run (test before installing)
helm install hypersdk hypersdk/hypersdk --dry-run --debug -n hypersdk
```

## Common Operations

```bash
# List releases
helm list -n hypersdk

# Get values
helm get values hypersdk -n hypersdk

# Get manifest
helm get manifest hypersdk -n hypersdk

# Upgrade
helm upgrade hypersdk hypersdk/hypersdk -f values.yaml -n hypersdk

# Rollback
helm rollback hypersdk -n hypersdk
helm rollback hypersdk 3 -n hypersdk  # to specific revision

# History
helm history hypersdk -n hypersdk

# Uninstall
helm uninstall hypersdk -n hypersdk
```

## Kubernetes Commands

```bash
# Check pods
kubectl get pods -n hypersdk
kubectl describe pod -n hypersdk POD_NAME

# Check all resources
kubectl get all -n hypersdk

# Logs
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk --previous  # previous container
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk -f  # follow

# Events
kubectl get events -n hypersdk --sort-by='.lastTimestamp'

# Resource usage
kubectl top pods -n hypersdk
kubectl top nodes

# Port forward
kubectl port-forward -n hypersdk svc/hypersdk 8080:8080
kubectl port-forward -n hypersdk svc/hypersdk 8081:8081  # metrics

# Execute in pod
kubectl exec -it -n hypersdk deploy/hypersdk -- /bin/sh

# Scale
kubectl scale deployment/hypersdk --replicas=5 -n hypersdk

# Restart
kubectl rollout restart deployment/hypersdk -n hypersdk
kubectl rollout status deployment/hypersdk -n hypersdk
```

## Health Checks

```bash
# API health
curl http://localhost:8080/health

# Metrics
curl http://localhost:8081/metrics

# Status
curl http://localhost:8080/api/v1/status

# Capabilities
curl http://localhost:8080/api/v1/capabilities
```

## Configuration Examples

### Minimal (Development)

```yaml
replicaCount: 1
resources:
  requests: {memory: 256Mi, cpu: 100m}
  limits: {memory: 512Mi, cpu: 500m}
persistence:
  data: {size: 1Gi}
  exports: {size: 10Gi}
```

### Production

```yaml
replicaCount: 3
resources:
  requests: {memory: 1Gi, cpu: 500m}
  limits: {memory: 4Gi, cpu: 2000m}
persistence:
  data: {size: 10Gi, storageClass: premium-ssd}
  exports: {size: 500Gi, storageClass: standard}
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
ingress:
  enabled: true
  hosts: [{host: hypersdk.example.com}]
```

## Troubleshooting

### Pod Pending

```bash
kubectl describe pod -n hypersdk POD_NAME
# Check: resources, PVC, node selector, taints
```

### CrashLoopBackOff

```bash
kubectl logs -n hypersdk POD_NAME --previous
kubectl describe pod -n hypersdk POD_NAME
# Common: database permissions, invalid config, OOMKilled
```

### Can't Access API

```bash
kubectl get svc,ingress -n hypersdk
kubectl get endpoints -n hypersdk
# Check service type and ingress configuration
```

### High CPU/Memory

```bash
kubectl top pods -n hypersdk
# Increase limits or scale out
helm upgrade hypersdk hypersdk/hypersdk --set resources.limits.memory=4Gi
```

### Storage Full

```bash
kubectl exec -n hypersdk deploy/hypersdk -- df -h
# Clean old data or expand PVC
kubectl exec -n hypersdk deploy/hypersdk -- find /exports -type f -mtime +7 -delete
```

## Monitoring

### Prometheus Queries

```promql
# Request rate
rate(hypersdk_http_requests_total[5m])

# Error rate
rate(hypersdk_http_requests_total{status=~"5.."}[5m])

# Active jobs
hypersdk_active_jobs

# Export duration (P95)
histogram_quantile(0.95, rate(hypersdk_export_duration_seconds_bucket[5m]))
```

### LogQL Queries (Loki)

```logql
# All logs
{app="hypersdk"}

# Errors only
{app="hypersdk"} |= "level=error"

# Specific job
{app="hypersdk"} | json | job_id="12345"
```

## Backup & Recovery

```bash
# Backup database
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db ".backup /tmp/backup.db"
kubectl cp hypersdk/POD:/tmp/backup.db ./backup-$(date +%Y%m%d).db

# Restore database
kubectl scale deployment/hypersdk --replicas=0 -n hypersdk
kubectl cp backup.db hypersdk/POD:/data/hypersdk.db
kubectl scale deployment/hypersdk --replicas=3 -n hypersdk

# Volume snapshot
kubectl apply -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: hypersdk-snapshot-$(date +%Y%m%d)
  namespace: hypersdk
spec:
  volumeSnapshotClassName: csi-snapclass
  source:
    persistentVolumeClaimName: hypersdk-data
EOF
```

## Security

### Create Secrets

```bash
# vSphere
kubectl create secret generic vsphere-credentials \
  --from-literal=url=https://vcenter.example.com/sdk \
  --from-literal=username=admin \
  --from-literal=password=changeme \
  -n hypersdk

# AWS
kubectl create secret generic aws-credentials \
  --from-literal=access_key=AKIAIOSFODNN7EXAMPLE \
  --from-literal=secret_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --from-literal=region=us-east-1 \
  -n hypersdk
```

### Check Security

```bash
# Pod security context
kubectl get pod -n hypersdk -o yaml | grep -A 10 securityContext

# RBAC
kubectl auth can-i --list --as=system:serviceaccount:hypersdk:hypersdk -n hypersdk

# Network policies
kubectl get networkpolicy -n hypersdk
```

## Cost Optimization

```bash
# Check resource usage
kubectl top pods -n hypersdk

# Right-size requests
helm upgrade hypersdk hypersdk/hypersdk \
  --set resources.requests.memory=512Mi \
  --set resources.requests.cpu=250m

# Enable autoscaling
helm upgrade hypersdk hypersdk/hypersdk \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=2 \
  --set autoscaling.maxReplicas=10

# Use cheaper storage
helm upgrade hypersdk hypersdk/hypersdk \
  --set persistence.exports.storageClass=standard
```

## GitOps

### ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk
spec:
  source:
    repoURL: oci://ghcr.io/ssahani/charts
    chart: hypersdk
    targetRevision: 0.2.0
  destination:
    namespace: hypersdk
  syncPolicy:
    automated: {prune: true, selfHeal: true}
```

```bash
# Sync manually
argocd app sync hypersdk

# Check status
argocd app get hypersdk
```

### Flux

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: hypersdk
spec:
  chart:
    spec:
      chart: hypersdk
      version: 0.2.0
      sourceRef:
        kind: HelmRepository
        name: hypersdk
```

```bash
# Force reconcile
flux reconcile helmrelease hypersdk -n flux-system

# Check status
flux get helmrelease hypersdk
```

## Testing

```bash
# Helm lint
helm lint deployments/helm/hypersdk

# Template and dry-run
helm template hypersdk deployments/helm/hypersdk | kubectl apply --dry-run=client -f -

# Install to test namespace
helm install test deployments/helm/hypersdk -n test --create-namespace

# Run tests
helm test hypersdk -n hypersdk

# Cleanup
helm uninstall test -n test
kubectl delete namespace test
```

## Diagnostic Bundle

```bash
# Collect diagnostics
kubectl get all -n hypersdk > diagnostics.txt
kubectl describe pod -n hypersdk >> diagnostics.txt
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk >> diagnostics.txt
kubectl get events -n hypersdk --sort-by='.lastTimestamp' >> diagnostics.txt
kubectl get configmap,secret,pvc -n hypersdk -o yaml >> diagnostics.txt
helm get values hypersdk -n hypersdk >> diagnostics.txt
```

## Environment Variables

```yaml
# Add custom environment variables
env:
  - name: CUSTOM_VAR
    value: "custom-value"
  - name: SECRET_VAR
    valueFrom:
      secretKeyRef:
        name: my-secret
        key: my-key
```

## Quick Links

| Task | Guide | Section |
|------|-------|---------|
| Install | [README.md](README.md) | Quick Start |
| Upgrade | [UPGRADE-GUIDE.md](UPGRADE-GUIDE.md) | Upgrade Procedures |
| Troubleshoot | [TROUBLESHOOTING-FAQ.md](TROUBLESHOOTING-FAQ.md) | Common Issues |
| Monitor | [OBSERVABILITY.md](OBSERVABILITY.md) | Metrics & Logs |
| Secure | [SECURITY.md](SECURITY.md) | Security Hardening |
| Optimize | [COST-OPTIMIZATION.md](COST-OPTIMIZATION.md) | Cost Savings |
| Recover | [DISASTER-RECOVERY.md](DISASTER-RECOVERY.md) | Backup & DR |
| Migrate | [MIGRATION.md](MIGRATION.md) | Migration Paths |

## Support

- **Issues**: https://github.com/ssahani/hypersdk/issues
- **Full Docs**: [OPERATIONAL-EXCELLENCE.md](OPERATIONAL-EXCELLENCE.md)
- **Troubleshooting**: [TROUBLESHOOTING-FAQ.md](TROUBLESHOOTING-FAQ.md)

---

**🔖 Bookmark this page for quick reference!**
