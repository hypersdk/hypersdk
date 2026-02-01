# Kubernetes Dashboard Integration Testing Guide

This guide explains how to test the Kubernetes Dashboard with real CRDs and live data.

## Prerequisites

1. **Kubernetes cluster** (minikube, kind, or production)
2. **kubectl** configured
3. **HyperSDK CRDs installed**
4. **HyperSDK operator running** (optional)

## Quick Start

### 1. Install CRDs

```bash
# Install HyperSDK CRDs
kubectl apply -f deploy/crds/hypersdk.io_backupjobs.yaml
kubectl apply -f deploy/crds/hypersdk.io_backupschedules.yaml
kubectl apply -f deploy/crds/hypersdk.io_restorejobs.yaml

# Verify CRDs are installed
kubectl get crds | grep hypersdk
```

Expected output:
```
backupjobs.hypersdk.io                2026-02-04T...
backupschedules.hypersdk.io           2026-02-04T...
restorejobs.hypersdk.io               2026-02-04T...
```

### 2. Create Test Resources

Create a test BackupJob:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: hypersdk.io/v1alpha1
kind: BackupJob
metadata:
  name: test-backup-1
  namespace: default
spec:
  source:
    provider: kubevirt
    namespace: default
    vmName: test-vm-1
  destination:
    type: s3
    bucket: test-backups
    region: us-west-2
  carbonAware:
    enabled: true
    zone: US-CAL-CISO
    maxIntensity: 200.0
  format:
    type: qcow2
    compression: gzip
status:
  phase: Running
  progress:
    percentComplete: 45
    currentStep: "Copying disk data"
  backupSize: 5368709120
  carbonIntensity: 125.5
EOF
```

Create a test BackupSchedule:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: hypersdk.io/v1alpha1
kind: BackupSchedule
metadata:
  name: nightly-backup
  namespace: default
spec:
  schedule: "0 2 * * *"
  timezone: America/Los_Angeles
  suspend: false
  jobTemplate:
    spec:
      source:
        provider: kubevirt
        namespace: default
        vmName: prod-vm
      destination:
        type: s3
        bucket: prod-backups
status:
  active: 0
  successfulJobs: 15
  failedJobs: 1
  lastScheduleTime: "2026-02-04T02:00:00Z"
  nextScheduleTime: "2026-02-05T02:00:00Z"
EOF
```

Create a test RestoreJob:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: hypersdk.io/v1alpha1
kind: RestoreJob
metadata:
  name: test-restore-1
  namespace: default
spec:
  source:
    backupJobRef:
      name: test-backup-1
      namespace: default
  destination:
    provider: kubevirt
    namespace: default
    vmName: restored-vm-1
  options:
    powerOnAfterRestore: true
status:
  phase: Running
  progress:
    percentComplete: 67
    currentStep: "Restoring disk image"
EOF
```

### 3. Start the Dashboard

**Option A: Local Development**

```bash
# Build and run
cd /home/ssahani/go/github/hypersdk
go run ./cmd/hypervisord --dashboard-enabled --dashboard-port 8080

# Or build first
go build -o hypervisord ./cmd/hypervisord
./hypervisord --dashboard-enabled --dashboard-port 8080
```

**Option B: Kubernetes Deployment**

```bash
# Deploy operator with dashboard
helm install hypersdk-operator ./deploy/helm/hypersdk-operator \
  --namespace hypersdk-system \
  --create-namespace

# Port-forward to access
kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080
```

### 4. Access the Dashboard

Open your browser and navigate to:

```
http://localhost:8080/k8s
```

You should see:
- **Cluster Info**: Connected status, version, node count
- **Overview**: Stats showing your test resources
- **BackupJobs Tab**: test-backup-1 with 45% progress
- **Schedules Tab**: nightly-backup with next run time
- **Restores Tab**: test-restore-1 with 67% progress
- **Carbon Stats**: Carbon-aware backup count and intensity

## Testing Dynamic Updates

### Test 1: Update Progress

Update a BackupJob to simulate progress:

```bash
kubectl patch backupjob test-backup-1 -p '{"status":{"progress":{"percentComplete":75}}}' --type=merge
```

The dashboard should update within 5 seconds showing 75% progress.

### Test 2: Complete a Job

Mark a backup as completed:

```bash
kubectl patch backupjob test-backup-1 -p '{
  "status": {
    "phase": "Completed",
    "progress": {"percentComplete": 100},
    "completionTime": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
  }
}' --type=merge
```

The dashboard should show the backup in "Completed" state with green badge.

### Test 3: Add More Resources

Create multiple backups to test pagination and stats:

```bash
for i in {2..10}; do
  kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: BackupJob
metadata:
  name: test-backup-$i
  namespace: default
spec:
  source:
    provider: kubevirt
    vmName: test-vm-$i
  destination:
    type: s3
    bucket: test-backups
status:
  phase: Completed
  progress:
    percentComplete: 100
  backupSize: $((RANDOM * 1000000))
EOF
done
```

The dashboard should now show 10 total backups in the Overview.

### Test 4: Test WebSocket

Open browser console (F12) and run:

```javascript
// Should see WebSocket connection
console.log('WebSocket state:', window.ws ? window.ws.readyState : 'Not connected');

// Watch for updates
// You'll see JSON messages every 5 seconds with metrics
```

## Testing API Endpoints

Test all API endpoints:

```bash
# All metrics
curl -s http://localhost:8080/api/k8s/metrics | jq .

# Cluster info
curl -s http://localhost:8080/api/k8s/cluster | jq .

# BackupJobs
curl -s http://localhost:8080/api/k8s/backupjobs | jq .

# Specific backup
curl -s http://localhost:8080/api/k8s/backupjobs/test-backup-1 | jq .

# Schedules
curl -s http://localhost:8080/api/k8s/backupschedules | jq .

# Restore jobs
curl -s http://localhost:8080/api/k8s/restorejobs | jq .

# Carbon stats
curl -s http://localhost:8080/api/k8s/carbon | jq .

# Storage stats
curl -s http://localhost:8080/api/k8s/storage | jq .
```

## Testing Carbon-Aware Features

Create a carbon-aware backup:

```bash
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: BackupJob
metadata:
  name: carbon-backup-1
spec:
  source:
    provider: kubevirt
    vmName: eco-vm
  destination:
    type: s3
    bucket: eco-backups
  carbonAware:
    enabled: true
    zone: SE
    maxIntensity: 150.0
    maxDelayHours: 6.0
status:
  phase: Pending
  carbonIntensity: 95.2
  carbonAware:
    delayed: true
    delayReason: "Waiting for lower carbon intensity"
    estimatedStartTime: "2026-02-04T14:00:00Z"
EOF
```

In the Carbon Stats tab, you should see:
- Carbon-aware backups: 2
- Average intensity: ~110 gCO2/kWh
- Delayed backups: 1

## Testing Error States

Create a failed backup:

```bash
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: BackupJob
metadata:
  name: failed-backup-1
spec:
  source:
    provider: kubevirt
    vmName: broken-vm
  destination:
    type: s3
    bucket: test-backups
status:
  phase: Failed
  conditions:
  - type: Failed
    status: "True"
    reason: "VMNotFound"
    message: "Virtual machine 'broken-vm' not found in namespace 'default'"
    lastTransitionTime: "2026-02-04T10:00:00Z"
EOF
```

The dashboard should show:
- Red "Failed" badge
- Error message in tooltip or details

## Performance Testing

### Load Test with Many Resources

Create 100 BackupJobs:

```bash
for i in {1..100}; do
  kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: BackupJob
metadata:
  name: load-test-$i
spec:
  source:
    provider: kubevirt
    vmName: vm-$i
  destination:
    type: s3
    bucket: load-test
status:
  phase: $( [ $((RANDOM % 4)) -eq 0 ] && echo "Pending" || [ $((RANDOM % 4)) -eq 1 ] && echo "Running" || [ $((RANDOM % 4)) -eq 2 ] && echo "Completed" || echo "Failed" )
  progress:
    percentComplete: $((RANDOM % 100))
EOF
done
```

Verify dashboard performance:
- Page load time < 2 seconds
- Update cycle < 1 second
- Memory usage stable
- No UI freezing

### WebSocket Load Test

Check WebSocket client limit:

```bash
# Open 100 browser tabs to http://localhost:8080/k8s
# Or use a WebSocket testing tool

# Check client count via API
curl http://localhost:8080/api/k8s/metrics | jq '.active_connections'
```

## Troubleshooting

### Dashboard Not Showing Data

1. **Check CRDs are installed**:
   ```bash
   kubectl get crds | grep hypersdk
   ```

2. **Check resources exist**:
   ```bash
   kubectl get backupjobs --all-namespaces
   kubectl get backupschedules --all-namespaces
   kubectl get restorejobs --all-namespaces
   ```

3. **Check dashboard logs**:
   ```bash
   # If running locally
   # Check terminal output for errors

   # If in Kubernetes
   kubectl logs -n hypersdk-system deployment/hypersdk-operator
   ```

4. **Check Kubernetes connection**:
   ```bash
   # Test kubectl works
   kubectl cluster-info
   kubectl get nodes
   ```

5. **Check browser console**:
   - Open F12 Developer Tools
   - Look for JavaScript errors
   - Check Network tab for failed requests

### WebSocket Not Connecting

1. **Check WebSocket endpoint**:
   ```bash
   curl -i -N -H "Connection: Upgrade" -H "Upgrade: websocket" \
     http://localhost:8080/ws/k8s
   ```

2. **Check browser console**:
   ```javascript
   // Should see:
   WebSocket connection to 'ws://localhost:8080/ws/k8s' established
   ```

3. **Verify WebSocket hub started**:
   - Check dashboard startup logs
   - Look for "WebSocket hub started" message

### Slow Updates

1. **Increase update interval**:
   Edit `k8s-dashboard.js`:
   ```javascript
   // Change from 5000 to 10000 for 10-second updates
   setInterval(fetchAndUpdate, 10000);
   ```

2. **Check API response time**:
   ```bash
   time curl -s http://localhost:8080/api/k8s/metrics > /dev/null
   ```

3. **Reduce resource count**:
   - Dashboard fetches all resources
   - Consider namespace filtering
   - Implement pagination (future)

## Cleanup

Remove all test resources:

```bash
# Delete all BackupJobs
kubectl delete backupjobs --all --all-namespaces

# Delete all BackupSchedules
kubectl delete backupschedules --all --all-namespaces

# Delete all RestoreJobs
kubectl delete restorejobs --all --all-namespaces

# Or delete entire namespace
kubectl delete namespace test-hypersdk
```

## Next Steps

1. **Add authentication**: Implement OAuth2/OIDC
2. **Add RBAC**: Restrict access by role
3. **Add filtering**: Filter by namespace, provider, status
4. **Add search**: Full-text search across resources
5. **Add charts**: Visualize trends over time
6. **Add export**: Export data to CSV/JSON
7. **Add alerts**: Configure alert rules
8. **Multi-cluster**: Support multiple Kubernetes clusters

## Integration with Operator

When the operator is running, it will automatically create and update BackupJobs,
BackupSchedules, and RestoreJobs based on their specs. The dashboard will show
live updates as the operator reconciles resources.

Test with operator:

```bash
# Deploy operator
helm install hypersdk-operator ./deploy/helm/hypersdk-operator

# Create a BackupSchedule (operator will create BackupJobs)
kubectl apply -f deploy/examples/backupschedule-nightly.yaml

# Watch the dashboard for:
- New BackupJob created by schedule
- Job progressing through phases
- Status updates in real-time
```

## Monitoring

Monitor dashboard health:

```bash
# Check metrics endpoint
curl http://localhost:8080/api/k8s/metrics | jq '{
  cluster: .cluster_info.connected,
  backups: .backup_jobs.total,
  schedules: .backup_schedules.total,
  restores: .restore_jobs.total
}'

# Set up Prometheus scraping (future)
# Add /metrics endpoint for Prometheus

# Create Grafana dashboard
# Import dashboard JSON template
```

## Security Testing

Test authentication and authorization:

```bash
# TODO: Add OAuth2 provider
# TODO: Test RBAC rules
# TODO: Test TLS/HTTPS
# TODO: Test rate limiting
# TODO: Test input validation
```

## Documentation

For more information:
- **Dashboard README**: `K8S_DASHBOARD_README.md`
- **Integration Progress**: `docs/KUBERNETES_INTEGRATION_PROGRESS.md`
- **CRD Spec**: `pkg/apis/hypersdk/v1alpha1/types.go`
- **CLI Usage**: `hyperctl k8s -op help`
