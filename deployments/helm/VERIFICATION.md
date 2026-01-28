# HyperSDK Helm Repository Verification Guide

This guide helps verify that the HyperSDK Helm repository is working correctly.

## Quick Verification

### 1. Check Repository Index

```bash
# Verify index.yaml is accessible
curl -I https://ssahani.github.io/hypersdk/helm-charts/index.yaml

# Expected: HTTP/2 200
```

### 2. Check Chart Package

```bash
# Verify chart package is accessible
curl -I https://ssahani.github.io/hypersdk/helm-charts/hypersdk-0.2.0.tgz

# Expected: HTTP/2 200
```

### 3. Add Helm Repository

```bash
# Add repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts

# Update repositories
helm repo update

# Expected: "Successfully got an update from the hypersdk chart repository"
```

### 4. Search for Charts

```bash
# Search for HyperSDK charts
helm search repo hypersdk

# Expected output:
# NAME                CHART VERSION  APP VERSION  DESCRIPTION
# hypersdk/hypersdk   0.2.0          0.2.0        Multi-cloud VM migration platform...
```

### 5. View Chart Information

```bash
# Show chart metadata
helm show chart hypersdk/hypersdk

# Show chart README
helm show readme hypersdk/hypersdk

# Show default values
helm show values hypersdk/hypersdk
```

## Complete Verification

### Test Installation (Dry Run)

```bash
# Test installation without actually deploying
helm install test-release hypersdk/hypersdk \
  --dry-run \
  --debug \
  --namespace hypersdk \
  --create-namespace

# Should render all Kubernetes manifests without errors
```

### Test with Example Values

```bash
# Test with minikube values
helm install test-release hypersdk/hypersdk \
  --dry-run \
  --values https://raw.githubusercontent.com/ssahani/hypersdk/main/deployments/helm/hypersdk/examples/minikube-values.yaml \
  --namespace hypersdk

# Should succeed without errors
```

## Troubleshooting

### Chart Not Found

**Problem**: `helm search repo hypersdk` returns no results

**Solutions**:

1. **Clear Helm cache**:
```bash
rm -rf ~/.cache/helm
helm repo remove hypersdk
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm repo update
```

2. **Verify repository URL**:
```bash
helm repo list | grep hypersdk
# Should show: https://ssahani.github.io/hypersdk/helm-charts
```

3. **Check GitHub Pages status**:
```bash
curl -s https://ssahani.github.io/hypersdk/helm-charts/index.yaml | head -20
```

### 404 Not Found on Chart Package

**Problem**: `Error: failed to fetch https://...hypersdk-0.2.0.tgz : 404 Not Found`

**Cause**: GitHub Pages caching or index.yaml not updated

**Solutions**:

1. **Wait for GitHub Pages to rebuild** (1-2 minutes after push)

2. **Clear Helm cache**:
```bash
rm -rf ~/.cache/helm
helm repo update
```

3. **Verify chart package URL in index.yaml**:
```bash
curl -s https://ssahani.github.io/hypersdk/helm-charts/index.yaml | grep -A 2 "urls:"
# Should show: https://ssahani.github.io/hypersdk/helm-charts/hypersdk-0.2.0.tgz
```

4. **Test direct download**:
```bash
curl -I https://ssahani.github.io/hypersdk/helm-charts/hypersdk-0.2.0.tgz
# Should return: HTTP/2 200
```

5. **Bypass cache with query parameter**:
```bash
helm repo remove hypersdk
helm repo add hypersdk "https://ssahani.github.io/hypersdk/helm-charts?t=$(date +%s)"
helm repo update
```

### GitHub Pages Not Enabled

**Problem**: Repository returns 404 for all URLs

**Solution**:

1. Go to: https://github.com/ssahani/hypersdk/settings/pages
2. Verify settings:
   - Source: Deploy from a branch
   - Branch: `main`
   - Folder: `/docs`
3. Click Save
4. Wait 1-2 minutes for deployment

### Index.yaml Has Wrong URLs

**Problem**: Chart URLs in index.yaml point to wrong location

**Solution**:

1. Check local index:
```bash
cat docs/helm-charts/index.yaml | grep -A 2 "urls:"
```

2. If incorrect, regenerate:
```bash
./deployments/scripts/package-helm-chart.sh --publish
git add docs/helm-charts
git commit -m "fix: Update Helm repository index"
git push origin main
```

3. Wait for GitHub Pages to rebuild

## Verification Checklist

Use this checklist to verify the repository is fully functional:

- [ ] Repository index accessible (200 OK)
  ```bash
  curl -I https://ssahani.github.io/hypersdk/helm-charts/index.yaml
  ```

- [ ] Chart package accessible (200 OK)
  ```bash
  curl -I https://ssahani.github.io/hypersdk/helm-charts/hypersdk-0.2.0.tgz
  ```

- [ ] Web interface accessible
  ```bash
  curl -I https://ssahani.github.io/hypersdk/helm-charts/
  ```

- [ ] Helm repository can be added
  ```bash
  helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
  ```

- [ ] Repository can be updated
  ```bash
  helm repo update hypersdk
  ```

- [ ] Charts can be found
  ```bash
  helm search repo hypersdk
  ```

- [ ] Chart metadata can be retrieved
  ```bash
  helm show chart hypersdk/hypersdk
  ```

- [ ] Chart values can be retrieved
  ```bash
  helm show values hypersdk/hypersdk
  ```

- [ ] Dry-run installation succeeds
  ```bash
  helm install test hypersdk/hypersdk --dry-run --namespace test
  ```

- [ ] Chart can be downloaded
  ```bash
  helm pull hypersdk/hypersdk
  ```

## GitHub Pages Cache

GitHub Pages uses CDN caching with a 10-minute TTL. After pushing changes:

1. **Wait 1-2 minutes** for GitHub Actions to rebuild pages
2. **Wait up to 10 minutes** for CDN cache to expire
3. **Clear Helm cache** to force redownload:
   ```bash
   rm -rf ~/.cache/helm
   helm repo update
   ```

### Force Cache Bypass

Use query parameter to bypass cache:

```bash
# Check latest index bypassing cache
curl "https://ssahani.github.io/hypersdk/helm-charts/index.yaml?t=$(date +%s)"

# Add repo with cache bypass
helm repo remove hypersdk
helm repo add hypersdk "https://ssahani.github.io/hypersdk/helm-charts?t=$(date +%s)"
```

## Automated Verification Script

```bash
#!/bin/bash
# verify-helm-repo.sh

set -e

REPO_URL="https://ssahani.github.io/hypersdk/helm-charts"
CHART_NAME="hypersdk"

echo "Verifying HyperSDK Helm Repository..."
echo ""

# Test 1: Index accessibility
echo "1. Checking index.yaml..."
if curl -f -s "${REPO_URL}/index.yaml" > /dev/null; then
    echo "   ✓ Index accessible"
else
    echo "   ✗ Index not accessible"
    exit 1
fi

# Test 2: Chart package
echo "2. Checking chart package..."
CHART_URL=$(curl -s "${REPO_URL}/index.yaml" | grep -A 2 "urls:" | tail -1 | awk '{print $2}')
if curl -f -I -s "${CHART_URL}" > /dev/null; then
    echo "   ✓ Chart package accessible"
else
    echo "   ✗ Chart package not accessible"
    exit 1
fi

# Test 3: Helm repo add
echo "3. Adding Helm repository..."
helm repo remove ${CHART_NAME} 2>/dev/null || true
if helm repo add ${CHART_NAME} "${REPO_URL}" > /dev/null 2>&1; then
    echo "   ✓ Repository added"
else
    echo "   ✗ Failed to add repository"
    exit 1
fi

# Test 4: Helm repo update
echo "4. Updating repository..."
if helm repo update ${CHART_NAME} > /dev/null 2>&1; then
    echo "   ✓ Repository updated"
else
    echo "   ✗ Failed to update repository"
    exit 1
fi

# Test 5: Search charts
echo "5. Searching for charts..."
if helm search repo ${CHART_NAME}/${CHART_NAME} | grep -q ${CHART_NAME}; then
    echo "   ✓ Chart found"
else
    echo "   ✗ Chart not found"
    exit 1
fi

# Test 6: Show chart
echo "6. Retrieving chart metadata..."
if helm show chart ${CHART_NAME}/${CHART_NAME} > /dev/null 2>&1; then
    echo "   ✓ Chart metadata retrieved"
else
    echo "   ✗ Failed to retrieve chart metadata"
    exit 1
fi

# Test 7: Dry run
echo "7. Testing dry-run installation..."
if helm install test ${CHART_NAME}/${CHART_NAME} --dry-run --namespace test > /dev/null 2>&1; then
    echo "   ✓ Dry-run succeeded"
else
    echo "   ✗ Dry-run failed"
    exit 1
fi

echo ""
echo "All tests passed! ✓"
echo ""
echo "Repository is fully functional:"
echo "  helm repo add ${CHART_NAME} ${REPO_URL}"
echo "  helm install my-${CHART_NAME} ${CHART_NAME}/${CHART_NAME}"
```

Save and run:
```bash
chmod +x verify-helm-repo.sh
./verify-helm-repo.sh
```

## Production Deployment Verification

After adding to production clusters:

```bash
# Add repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts

# Install to test namespace
helm install hypersdk-test hypersdk/hypersdk \
  --namespace hypersdk-test \
  --create-namespace \
  --wait \
  --timeout 5m

# Verify deployment
kubectl get all -n hypersdk-test

# Test health endpoint
kubectl port-forward -n hypersdk-test svc/hypersdk-test 8080:8080 &
curl http://localhost:8080/health

# Cleanup
helm uninstall hypersdk-test -n hypersdk-test
kubectl delete namespace hypersdk-test
```

## Support

If verification fails after following all troubleshooting steps:

1. Check [GitHub Actions](https://github.com/ssahani/hypersdk/actions) for pages-build-deployment
2. Review [GitHub Pages settings](https://github.com/ssahani/hypersdk/settings/pages)
3. Open an issue: https://github.com/ssahani/hypersdk/issues

## Summary

Repository should be fully functional when:

✅ All verification checks pass
✅ GitHub Pages shows "Active" in repository settings
✅ Helm can search and download charts
✅ Dry-run installations succeed

Current repository status can always be checked at:
https://ssahani.github.io/hypersdk/helm-charts/
