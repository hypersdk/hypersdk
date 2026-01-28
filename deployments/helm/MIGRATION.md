# Migration Guide - Moving to HyperSDK Helm Charts

Comprehensive guide for migrating from manual deployments, Docker Compose, or other Kubernetes deployments to HyperSDK Helm charts.

## Overview

This guide covers migration from:

- [Manual Kubernetes YAML](#from-manual-kubernetes-yaml)
- [Docker Compose](#from-docker-compose)
- [Kustomize](#from-kustomize)
- [Other Helm Charts](#from-other-helm-charts)
- [VM-based Deployment](#from-vm-based-deployment)
- [Legacy hyper2kvm](#from-legacy-hyper2kvm)

## Pre-Migration Checklist

### Data Backup

✅ **Critical**: Back up all data before migration

```bash
# 1. Backup SQLite database
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db ".backup /tmp/backup.db"
kubectl cp hypersdk/PODNAME:/tmp/backup.db ./hypersdk-backup-$(date +%Y%m%d).db

# 2. Backup exports directory
kubectl exec -n hypersdk deploy/hypersdk -- \
  tar -czf /tmp/exports-backup.tar.gz /exports
kubectl cp hypersdk/PODNAME:/tmp/exports-backup.tar.gz ./exports-backup-$(date +%Y%m%d).tar.gz

# 3. Backup configuration
kubectl get configmap,secret -n hypersdk -o yaml > config-backup.yaml

# 4. Export current deployment manifest
kubectl get all -n hypersdk -o yaml > current-deployment.yaml
```

### Environment Assessment

```bash
# Record current state
kubectl get all -n hypersdk > current-state.txt
kubectl top pods -n hypersdk > current-usage.txt
kubectl describe deployment -n hypersdk > current-config.txt

# Note important details:
# - Resource requests/limits
# - Environment variables
# - Volume mounts
# - Network configuration
# - Cloud provider credentials
```

### Prerequisites

- [ ] Helm 3.8+ installed
- [ ] kubectl access to cluster
- [ ] Backup completed and verified
- [ ] Maintenance window scheduled
- [ ] Rollback plan documented

## From Manual Kubernetes YAML

### Current State

You have YAML files like:
```
kubernetes/
├── deployment.yaml
├── service.yaml
├── configmap.yaml
├── secrets.yaml
└── pvc.yaml
```

### Migration Steps

#### Step 1: Extract Current Configuration

```bash
# Extract current values
kubectl get deployment -n hypersdk hypersdk -o yaml > current-deployment.yaml
kubectl get configmap -n hypersdk hypersdk -o yaml > current-configmap.yaml
kubectl get secret -n hypersdk -o yaml > current-secrets.yaml
```

#### Step 2: Map to Helm Values

Create `migration-values.yaml`:

```yaml
# From deployment.yaml
replicaCount: 3  # spec.replicas

image:
  repository: ghcr.io/ssahani/hypersdk-hypervisord  # from image
  tag: "0.2.0"  # from image tag
  pullPolicy: IfNotPresent

# From resources in deployment
resources:
  requests:
    memory: 512Mi
    cpu: 250m
  limits:
    memory: 2Gi
    cpu: 1000m

# From configmap
config:
  logLevel: info
  downloadWorkers: 3
  # ... other config values

# From environment variables
env:
  - name: CUSTOM_VAR
    value: "custom-value"

# From service.yaml
service:
  type: LoadBalancer
  port: 8080

# From PVC
persistence:
  data:
    enabled: true
    size: 10Gi
    storageClass: gp2
  exports:
    enabled: true
    size: 500Gi
    storageClass: standard

# From secrets (create separately)
credentials:
  vsphere:
    existingSecret: vsphere-credentials
```

#### Step 3: Create Secrets

```bash
# Re-create secrets from backup
kubectl apply -f current-secrets.yaml
```

#### Step 4: Test with Helm

```bash
# Dry-run to verify
helm template hypersdk hypersdk/hypersdk \
  -f migration-values.yaml \
  --namespace hypersdk > helm-rendered.yaml

# Compare with current deployment
diff current-deployment.yaml helm-rendered.yaml
```

#### Step 5: Perform Migration

```bash
# Option A: In-place migration (zero downtime)
# Scale down current deployment
kubectl scale deployment/hypersdk --replicas=0 -n hypersdk

# Install Helm chart
helm install hypersdk hypersdk/hypersdk \
  -f migration-values.yaml \
  --namespace hypersdk

# Verify
kubectl rollout status deployment/hypersdk -n hypersdk

# Delete old resources (after verification)
kubectl delete -f deployment.yaml
kubectl delete -f service.yaml
kubectl delete -f configmap.yaml

# Option B: Blue-green migration (recommended)
# Create new namespace
kubectl create namespace hypersdk-new

# Install to new namespace
helm install hypersdk hypersdk/hypersdk \
  -f migration-values.yaml \
  --namespace hypersdk-new

# Copy data from old PVC to new
kubectl exec -n hypersdk deploy/hypersdk -- \
  tar -czf /tmp/data-backup.tar.gz /data /exports

kubectl cp hypersdk/PODNAME:/tmp/data-backup.tar.gz data-backup.tar.gz

kubectl cp data-backup.tar.gz hypersdk-new/PODNAME:/tmp/

kubectl exec -n hypersdk-new deploy/hypersdk -- \
  tar -xzf /tmp/data-backup.tar.gz -C /

# Switch traffic (update ingress/service)
kubectl patch ingress hypersdk \
  -p '{"spec":{"rules":[{"host":"hypersdk.example.com","http":{"paths":[{"backend":{"service":{"name":"hypersdk","namespace":"hypersdk-new"}}}]}}]}}'

# Monitor new deployment
kubectl logs -n hypersdk-new -l app.kubernetes.io/name=hypersdk -f

# Delete old namespace (after verification)
kubectl delete namespace hypersdk
```

## From Docker Compose

### Current State

You have `docker-compose.yml`:

```yaml
version: '3.8'
services:
  hypervisord:
    image: ghcr.io/ssahani/hypersdk-hypervisord:0.2.0
    ports:
      - "8080:8080"
      - "8081:8081"
    environment:
      - GOVC_URL=https://vcenter.example.com/sdk
      - GOVC_USERNAME=admin
      - GOVC_PASSWORD=password
    volumes:
      - ./data:/data
      - ./exports:/exports
    restart: unless-stopped
```

### Migration Steps

#### Step 1: Create Values File

```yaml
# docker-compose-migration-values.yaml

# From image
image:
  repository: ghcr.io/ssahani/hypersdk-hypervisord
  tag: "0.2.0"

# From ports
service:
  type: LoadBalancer
  port: 8080
  metricsPort: 8081

# From restart policy
replicaCount: 1

# From volumes (create PVC)
persistence:
  data:
    enabled: true
    size: 10Gi
  exports:
    enabled: true
    size: 100Gi

# From environment (create secret)
credentials:
  vsphere:
    url: "https://vcenter.example.com/sdk"
    username: "admin"
    password: "password"  # Will create secret
```

#### Step 2: Backup Docker Volumes

```bash
# Backup Docker volumes
docker-compose down
sudo tar -czf data-backup.tar.gz /var/lib/docker/volumes/*hypersdk*

# Or copy data directly
docker cp hypervisord:/data ./data-backup
docker cp hypervisord:/exports ./exports-backup
```

#### Step 3: Deploy with Helm

```bash
# Install Helm chart
helm install hypersdk hypersdk/hypersdk \
  -f docker-compose-migration-values.yaml \
  --namespace hypersdk \
  --create-namespace

# Wait for PVCs to be created
kubectl wait --for=condition=Bound pvc --all -n hypersdk --timeout=300s

# Copy data to PVCs
POD=$(kubectl get pod -n hypersdk -l app.kubernetes.io/name=hypersdk -o jsonpath='{.items[0].metadata.name}')

kubectl cp data-backup "$hypersdk/$POD:/data/"
kubectl cp exports-backup "$hypersdk/$POD:/exports/"

# Restart to pick up data
kubectl rollout restart deployment/hypersdk -n hypersdk
```

#### Step 4: Migrate Network Configuration

```bash
# If using custom network in Docker Compose
# Configure Ingress in Kubernetes

helm upgrade hypersdk hypersdk/hypersdk \
  --reuse-values \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=hypersdk.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix
```

## From Kustomize

### Current State

You have Kustomize overlays:
```
kustomize/
├── base/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── kustomization.yaml
└── overlays/
    ├── dev/
    ├── staging/
    └── production/
```

### Migration Strategy

#### Step 1: Extract Per-Environment Configuration

```bash
# For each environment
for env in dev staging production; do
  kubectl kustomize kustomize/overlays/$env > rendered-$env.yaml

  # Extract values
  # (manually create values-$env.yaml based on rendered YAML)
done
```

#### Step 2: Create Helm Values per Environment

```yaml
# values-production.yaml
replicaCount: 3

image:
  tag: "0.2.0"

resources:
  requests:
    memory: 1Gi
    cpu: 500m
  limits:
    memory: 4Gi
    cpu: 2000m

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10

persistence:
  data:
    size: 10Gi
    storageClass: premium-ssd
  exports:
    size: 1Ti
    storageClass: standard
```

```yaml
# values-staging.yaml
replicaCount: 2

resources:
  requests:
    memory: 512Mi
    cpu: 250m
  limits:
    memory: 2Gi
    cpu: 1000m

persistence:
  data:
    size: 5Gi
  exports:
    size: 100Gi
```

#### Step 3: GitOps Integration (Recommended)

If using GitOps with Kustomize, migrate to GitOps with Helm:

**With ArgoCD:**
```yaml
# argocd/hypersdk-production.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk-production
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://ssahani.github.io/hypersdk/helm-charts
    chart: hypersdk
    targetRevision: 0.2.0
    helm:
      valueFiles:
        - values-production.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: hypersdk-production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

**With Flux:**
```yaml
# flux/hypersdk-production.yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: hypersdk
  namespace: hypersdk-production
spec:
  interval: 5m
  chart:
    spec:
      chart: hypersdk
      version: 0.2.0
      sourceRef:
        kind: HelmRepository
        name: hypersdk
  valuesFrom:
    - kind: ConfigMap
      name: hypersdk-values-production
```

## From Other Helm Charts

### Migration from Custom Chart

If you have a custom Helm chart:

#### Step 1: Compare Chart Structures

```bash
# Extract your current values
helm get values my-hypersdk -n hypersdk > old-values.yaml

# Compare chart structures
helm show values hypersdk/hypersdk > new-values-reference.yaml

# Review differences
diff old-values.yaml new-values-reference.yaml
```

#### Step 2: Map Values

Create mapping document:

```yaml
# Old Chart → New Chart mapping
# old-values.yaml          # new values.yaml

app:
  replicas: 3              → replicaCount: 3
  image: custom/image      → image.repository: ghcr.io/ssahani/hypersdk-hypervisord

database:
  path: /db                → persistence.data.mountPath: /data

network:
  serviceType: LB          → service.type: LoadBalancer
```

#### Step 3: Migrate with Zero Downtime

```bash
# Install new chart alongside old
helm install hypersdk-new hypersdk/hypersdk \
  -f migrated-values.yaml \
  --namespace hypersdk-new \
  --create-namespace

# Copy data
kubectl exec -n hypersdk deploy/my-hypersdk -- \
  sqlite3 /db/hypersdk.db ".backup /tmp/backup.db"

POD_NEW=$(kubectl get pod -n hypersdk-new -l app.kubernetes.io/name=hypersdk -o jsonpath='{.items[0].metadata.name}')

kubectl cp hypersdk/my-hypersdk-xxx:/tmp/backup.db - | \
  kubectl exec -i -n hypersdk-new "$POD_NEW" -- \
  sh -c "cat > /data/hypersdk.db"

# Switch traffic
kubectl patch service my-hypersdk -n hypersdk \
  -p '{"spec":{"selector":{"app.kubernetes.io/name":"hypersdk"}}}'

# Monitor and clean up old chart
kubectl delete namespace hypersdk-old
```

## From VM-based Deployment

### Current State

HyperSDK running on VM with systemd:

```
/opt/hypersdk/
├── hypervisord (binary)
├── config.yaml
├── data/
│   └── hypersdk.db
└── exports/
```

### Migration Steps

#### Step 1: Backup VM Data

```bash
# On VM
sudo systemctl stop hypervisord

# Backup database
sudo cp /opt/hypersdk/data/hypersdk.db /tmp/hypersdk-backup.db

# Backup exports
sudo tar -czf /tmp/exports-backup.tar.gz /opt/hypersdk/exports

# Copy to local machine
scp user@vm:/tmp/hypersdk-backup.db ./
scp user@vm:/tmp/exports-backup.tar.gz ./
```

#### Step 2: Extract Configuration

```bash
# Copy config from VM
scp user@vm:/opt/hypersdk/config.yaml ./vm-config.yaml

# Convert to Helm values
cat vm-config.yaml  # Manually create values file
```

#### Step 3: Deploy to Kubernetes

```bash
# Create namespace
kubectl create namespace hypersdk

# Create secrets from VM credentials
kubectl create secret generic vsphere-credentials \
  --from-literal=url=https://vcenter.example.com/sdk \
  --from-literal=username=admin \
  --from-literal=password=password \
  -n hypersdk

# Install Helm chart
helm install hypersdk hypersdk/hypersdk \
  -f vm-migration-values.yaml \
  --namespace hypersdk

# Wait for PVC creation
kubectl wait --for=condition=Bound pvc --all -n hypersdk

# Copy data to Kubernetes
POD=$(kubectl get pod -n hypersdk -l app.kubernetes.io/name=hypersdk -o jsonpath='{.items[0].metadata.name}')

kubectl cp hypersdk-backup.db "hypersdk/$POD:/data/hypersdk.db"
kubectl exec -n hypersdk "$POD" -- tar -xzf - -C / < exports-backup.tar.gz

# Restart to pick up data
kubectl rollout restart deployment/hypersdk -n hypersdk
```

#### Step 4: Update DNS/Load Balancer

```bash
# Get LoadBalancer IP
kubectl get svc -n hypersdk hypersdk

# Update DNS to point to new LoadBalancer IP
# Or use Ingress with existing domain
```

#### Step 5: Decommission VM

```bash
# After verification
# On VM:
sudo systemctl stop hypervisord
sudo systemctl disable hypervisord

# Keep VM for 30 days as backup, then terminate
```

## From Legacy hyper2kvm

### Background

The legacy `hyper2kvm` Dockerfile (located at `/Dockerfile` in repo) was for a specific conversion tool. The new Helm chart deploys the full HyperSDK platform.

### Key Differences

| Legacy hyper2kvm | New HyperSDK |
|------------------|--------------|
| Single container | Full microservices |
| Manual deployment | Helm managed |
| No persistence | PVC-backed storage |
| No monitoring | Full observability |
| Single binary | REST API + CLI tools |

### Migration Not Applicable

If you're using the legacy `hyper2kvm` container, this is a different tool. You should:

1. Deploy HyperSDK fresh using this Helm chart
2. Migrate your conversion workflows to use HyperSDK's REST API
3. Update automation to call the new API endpoints

## Post-Migration Checklist

### Verification Steps

```bash
# 1. Check all pods running
kubectl get pods -n hypersdk
# Expected: All Running

# 2. Verify data migrated
kubectl exec -n hypersdk deploy/hypersdk -- \
  sqlite3 /data/hypersdk.db "SELECT COUNT(*) FROM jobs;"

# 3. Test API
kubectl port-forward -n hypersdk svc/hypersdk 8080:8080 &
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/capabilities

# 4. Test export job (if applicable)
# Submit test job via API

# 5. Verify monitoring
kubectl get servicemonitor -n hypersdk
kubectl port-forward -n hypersdk svc/hypersdk 8081:8081 &
curl http://localhost:8081/metrics

# 6. Check resource usage
kubectl top pods -n hypersdk

# 7. Verify persistence
kubectl get pvc -n hypersdk
kubectl exec -n hypersdk deploy/hypersdk -- df -h

# 8. Test failover (if HA)
kubectl delete pod -n hypersdk -l app.kubernetes.io/name=hypersdk
kubectl get pods -n hypersdk -w
```

### Performance Comparison

```bash
# Before migration (record baseline)
# - API response time
# - Export job duration
# - Resource usage

# After migration (compare)
# Run same tests

# Example:
time curl http://localhost:8080/api/v1/status
```

### Rollback Plan

If migration fails:

```bash
# Option 1: Rollback Helm release
helm rollback hypersdk -n hypersdk

# Option 2: Restore from backup
kubectl delete namespace hypersdk
# Re-deploy old resources
kubectl apply -f current-deployment.yaml

# Restore data
# (reverse of backup process)
```

## Common Migration Issues

### Issue: Data Loss During Migration

**Prevention:**
- Always backup before migration
- Verify backups are restorable
- Test migration in non-production first

### Issue: Downtime During Migration

**Solution:**
- Use blue-green migration approach
- Deploy to new namespace
- Switch traffic only after verification

### Issue: Configuration Mismatch

**Solution:**
- Carefully map old config to new values
- Use `helm template` to preview changes
- Compare rendered manifests with old deployment

### Issue: Storage Class Not Compatible

**Solution:**
```bash
# Copy data between PVCs
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: data-migration
spec:
  template:
    spec:
      containers:
        - name: copy
          image: alpine:3.19
          command: ['sh', '-c', 'cp -r /old/* /new/']
          volumeMounts:
            - name: old-data
              mountPath: /old
            - name: new-data
              mountPath: /new
      volumes:
        - name: old-data
          persistentVolumeClaim:
            claimName: old-pvc
        - name: new-data
          persistentVolumeClaim:
            claimName: new-pvc
      restartPolicy: Never
EOF
```

## Migration Timeline

### Recommended Schedule

**Week 1: Planning**
- Review current deployment
- Create migration plan
- Backup all data
- Set up test environment

**Week 2: Testing**
- Deploy Helm chart to test environment
- Migrate test data
- Verify functionality
- Performance testing

**Week 3: Staging**
- Deploy to staging environment
- Migrate staging data
- User acceptance testing
- Create runbook

**Week 4: Production**
- Schedule maintenance window
- Execute migration during off-peak hours
- Monitor closely for 24 hours
- Document lessons learned

## Support

If you encounter issues during migration:

1. Check [TROUBLESHOOTING-FAQ.md](TROUBLESHOOTING-FAQ.md)
2. Review [OPERATIONS-RUNBOOK.md](OPERATIONS-RUNBOOK.md)
3. Create GitHub issue with:
   - Source deployment type
   - Error messages
   - Migration steps attempted
   - Diagnostic bundle

## Summary

### Migration Checklist

- [ ] Backup completed and verified
- [ ] Migration values file created
- [ ] Test deployment successful
- [ ] Data migration plan documented
- [ ] Rollback plan ready
- [ ] Maintenance window scheduled
- [ ] Stakeholders notified
- [ ] Monitoring configured
- [ ] Post-migration verification passed
- [ ] Old resources cleaned up
- [ ] Documentation updated

### Success Criteria

✅ All data migrated successfully
✅ Zero data loss
✅ Minimal downtime (< 15 minutes for production)
✅ API functionality verified
✅ Monitoring operational
✅ Performance meets or exceeds baseline
✅ Team trained on Helm chart operations

---

**Remember**: Migration is a one-time activity - take time to plan and execute carefully!
