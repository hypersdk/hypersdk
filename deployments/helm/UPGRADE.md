# HyperSDK Helm Chart Upgrade Guide

This guide covers upgrading HyperSDK Helm chart deployments.

## Quick Upgrade

### From Helm Repository

```bash
# Update Helm repositories
helm repo update

# Upgrade to latest version
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values

# Upgrade to specific version
helm upgrade hypersdk hypersdk/hypersdk \
  --version 0.3.0 \
  --namespace hypersdk \
  --reuse-values
```

### From Local Chart

```bash
# Pull latest code
git pull origin main

# Upgrade with local chart
helm upgrade hypersdk ./deployments/helm/hypersdk \
  --namespace hypersdk \
  --reuse-values
```

## Upgrade Strategies

### 1. Reuse Values (Recommended)

Keeps your existing configuration:

```bash
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values
```

### 2. Reset Values

Uses new chart defaults (caution: may override your settings):

```bash
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reset-values
```

### 3. Merge Values

Combines existing values with new ones:

```bash
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --values new-values.yaml
```

### 4. Update Specific Values

Change only specific values:

```bash
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --set replicaCount=3 \
  --set resources.limits.memory=4Gi
```

## Pre-Upgrade Checklist

Before upgrading, complete these steps:

### 1. Backup Current Configuration

```bash
# Export current values
helm get values hypersdk -n hypersdk > current-values.yaml

# Export current manifests
helm get manifest hypersdk -n hypersdk > current-manifest.yaml

# Backup PVC data (optional, for critical deployments)
kubectl exec -n hypersdk deployment/hypersdk -- tar czf /tmp/backup.tar.gz /data /exports
kubectl cp hypersdk/$(kubectl get pod -n hypersdk -l app.kubernetes.io/name=hypersdk -o jsonpath='{.items[0].metadata.name}'):/tmp/backup.tar.gz ./backup-$(date +%Y%m%d).tar.gz
```

### 2. Check Current Version

```bash
# Check installed chart version
helm list -n hypersdk

# Check available versions
helm search repo hypersdk/hypersdk --versions
```

### 3. Review Changelog

Check [CHANGELOG.md](../../CHANGELOG.md) for breaking changes.

### 4. Test in Non-Production

Test the upgrade in a development or staging environment first.

## Upgrade Procedures

### Standard Upgrade (No Breaking Changes)

For patch and minor version upgrades (0.2.0 → 0.2.1 or 0.2.0 → 0.3.0):

```bash
# 1. Update repository
helm repo update

# 2. Check what will change
helm diff upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values

# 3. Perform upgrade
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --wait \
  --timeout 10m

# 4. Verify deployment
kubectl rollout status deployment/hypersdk -n hypersdk
kubectl get pods -n hypersdk
```

### Major Version Upgrade (With Breaking Changes)

For major version upgrades (0.x.x → 1.0.0):

```bash
# 1. Read migration guide
# Check CHANGELOG.md for version-specific instructions

# 2. Backup everything
helm get values hypersdk -n hypersdk > backup-values.yaml
kubectl get all -n hypersdk -o yaml > backup-resources.yaml

# 3. Update values file for breaking changes
vim backup-values.yaml  # Make necessary changes

# 4. Test with dry-run
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --values backup-values.yaml \
  --dry-run --debug

# 5. Perform upgrade
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --values backup-values.yaml \
  --wait \
  --timeout 10m

# 6. Verify and test
kubectl get all -n hypersdk
curl http://<service-url>:8080/health
```

### Rolling Update (Zero Downtime)

For production deployments with high availability:

```bash
# Ensure replicas > 1
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --set replicaCount=3 \
  --wait

# Upgrade with rolling update strategy
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --wait \
  --timeout 15m

# Monitor rollout
kubectl rollout status deployment/hypersdk -n hypersdk
```

## Post-Upgrade Verification

### 1. Check Deployment Status

```bash
# Verify pods are running
kubectl get pods -n hypersdk

# Check deployment status
kubectl rollout status deployment/hypersdk -n hypersdk

# View recent events
kubectl get events -n hypersdk --sort-by='.lastTimestamp' | tail -20
```

### 2. Test Endpoints

```bash
# Health check
curl http://<service-url>:8080/health

# API status
curl http://<service-url>:8080/api/v1/status

# Metrics
curl http://<service-url>:8081/metrics
```

### 3. Check Logs

```bash
# View application logs
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk --tail=100

# Follow logs
kubectl logs -n hypersdk -l app.kubernetes.io/name=hypersdk -f
```

### 4. Verify Configuration

```bash
# Check current Helm values
helm get values hypersdk -n hypersdk

# Compare with backup
diff current-values.yaml <(helm get values hypersdk -n hypersdk)
```

## Rollback

If the upgrade fails or causes issues:

### Quick Rollback

```bash
# Rollback to previous revision
helm rollback hypersdk -n hypersdk

# Rollback to specific revision
helm rollback hypersdk 2 -n hypersdk
```

### Check Revision History

```bash
# List all revisions
helm history hypersdk -n hypersdk

# View specific revision
helm get values hypersdk -n hypersdk --revision 2
```

### Complete Rollback Procedure

```bash
# 1. Check revision history
helm history hypersdk -n hypersdk

# 2. Identify good revision
# REVISION  UPDATED                   STATUS      CHART            DESCRIPTION
# 1         Mon Jan 29 10:00:00 2026  superseded  hypersdk-0.2.0   Install complete
# 2         Thu Jan 30 15:00:00 2026  deployed    hypersdk-0.3.0   Upgrade complete

# 3. Rollback to revision 1
helm rollback hypersdk 1 -n hypersdk --wait

# 4. Verify rollback
kubectl rollout status deployment/hypersdk -n hypersdk
kubectl get pods -n hypersdk
```

## Troubleshooting

### Upgrade Stuck

If upgrade hangs:

```bash
# Check pod status
kubectl get pods -n hypersdk

# Check pod events
kubectl describe pod -n hypersdk <pod-name>

# Check deployment status
kubectl describe deployment hypersdk -n hypersdk

# Force delete stuck pods
kubectl delete pod -n hypersdk <pod-name> --force --grace-period=0
```

### Image Pull Errors

```bash
# Check image pull status
kubectl describe pod -n hypersdk <pod-name> | grep -A 10 "Events:"

# Verify image exists
helm get values hypersdk -n hypersdk | grep image

# Update image pull policy
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --set image.pullPolicy=Always
```

### Configuration Errors

```bash
# Validate values
helm lint ./deployments/helm/hypersdk --values my-values.yaml

# Check ConfigMap
kubectl get configmap hypersdk -n hypersdk -o yaml

# Check Secrets
kubectl get secret -n hypersdk
```

### PVC Issues

```bash
# Check PVC status
kubectl get pvc -n hypersdk

# Describe PVC
kubectl describe pvc -n hypersdk hypersdk-data

# Check storage class
kubectl get storageclass
```

## Version-Specific Upgrade Notes

### 0.1.x → 0.2.0

**Changes**:
- Added NOTES.txt template
- Updated ConfigMap webhook configuration (map → array)
- Added cloud provider example values

**Action Required**:
```bash
# No breaking changes, standard upgrade
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values
```

### 0.2.x → 0.3.0 (Future)

When 0.3.0 is released, check CHANGELOG.md for specific instructions.

## Best Practices

### 1. Always Test First

```bash
# Dry-run before actual upgrade
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --dry-run --debug
```

### 2. Use Version Pinning

```bash
# Pin to specific version
helm upgrade hypersdk hypersdk/hypersdk \
  --version 0.2.0 \
  --namespace hypersdk \
  --reuse-values
```

### 3. Enable Wait and Timeout

```bash
# Wait for deployment to be ready
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --wait \
  --timeout 10m
```

### 4. Monitor During Upgrade

```bash
# In one terminal: upgrade
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk \
  --reuse-values \
  --wait

# In another terminal: monitor
watch kubectl get pods -n hypersdk
```

### 5. Keep Backup

Always keep a backup of:
- Current Helm values
- Current manifests
- Database (if critical)
- Configuration files

## Automation

### CI/CD Pipeline Upgrade

```yaml
# Example GitLab CI/CD
upgrade:
  stage: deploy
  script:
    - helm repo update
    - helm upgrade hypersdk hypersdk/hypersdk
        --namespace hypersdk
        --reuse-values
        --wait
        --timeout 10m
  only:
    - main
```

### Automated Upgrade Script

```bash
#!/bin/bash
# upgrade-hypersdk.sh

set -e

NAMESPACE="${NAMESPACE:-hypersdk}"
RELEASE="${RELEASE:-hypersdk}"

echo "Backing up current configuration..."
helm get values "${RELEASE}" -n "${NAMESPACE}" > backup-values-$(date +%Y%m%d).yaml

echo "Updating Helm repositories..."
helm repo update

echo "Upgrading ${RELEASE}..."
helm upgrade "${RELEASE}" hypersdk/hypersdk \
  --namespace "${NAMESPACE}" \
  --reuse-values \
  --wait \
  --timeout 10m

echo "Verifying deployment..."
kubectl rollout status deployment/"${RELEASE}" -n "${NAMESPACE}"

echo "Upgrade complete!"
```

## Support

For upgrade issues:

1. Check [Troubleshooting Guide](../docs/reference/troubleshooting-guide.md)
2. Search [GitHub Issues](https://github.com/ssahani/hypersdk/issues)
3. Open a new issue with:
   - Current chart version
   - Target chart version
   - Error messages
   - Helm values (sanitized)
   - Kubernetes logs

## Summary

**Quick Upgrade Steps**:
1. ✅ Backup current values
2. ✅ Check changelog
3. ✅ Update Helm repo
4. ✅ Test with dry-run
5. ✅ Perform upgrade
6. ✅ Verify deployment
7. ✅ Rollback if needed

**Remember**:
- Always backup before upgrading
- Test in non-production first
- Use `--wait` for safer upgrades
- Keep revision history
- Monitor during upgrade
- Know how to rollback
