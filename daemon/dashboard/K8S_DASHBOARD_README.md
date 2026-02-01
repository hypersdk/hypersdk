# Kubernetes Dashboard for HyperSDK

The Kubernetes Dashboard provides real-time monitoring and management of HyperSDK resources running on Kubernetes.

## Features

### 📊 Resource Monitoring

- **BackupJobs**: View all backup jobs with status, progress, and carbon intensity
- **BackupSchedules**: Monitor scheduled backups and their execution history
- **RestoreJobs**: Track VM restoration progress
- **Operator Status**: Real-time operator health and replica count

### 🌱 Carbon-Aware Statistics

- Track carbon-aware backups
- Monitor average carbon intensity
- View estimated CO₂ savings
- See delayed backups waiting for green energy

### ☸️ Cluster Information

- Kubernetes cluster version
- Node count
- Namespace count
- KubeVirt detection
- Operator status

## Access

### Local Development

```bash
# Start the dashboard (runs on port 8080 by default)
./hypervisord --dashboard-enabled

# Or specify a custom port
./hypervisord --dashboard-port 8888
```

Access the Kubernetes dashboard at: `http://localhost:8080/k8s`

### Kubernetes Deployment

When running the HyperSDK operator in Kubernetes, the dashboard is automatically available:

```bash
# Port-forward to access locally
kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080

# Access at
http://localhost:8080/k8s
```

### Helm Deployment

The dashboard is enabled by default in the Helm chart:

```bash
helm install hypersdk-operator ./deploy/helm/hypersdk-operator

# Access via port-forward
kubectl port-forward -n hypersdk-system svc/hypersdk-operator 8080:8080
```

## Dashboard Tabs

### Overview

Shows high-level statistics:
- Total backups
- Running backups
- Active schedules
- Pending restores

### Backup Jobs

Table view of all BackupJob resources with:
- Name and namespace
- VM name
- Provider (KubeVirt, vSphere, AWS, etc.)
- Phase (Pending, Running, Completed, Failed)
- Progress bar with percentage
- Backup size
- Carbon intensity (for carbon-aware backups)
- Duration

### Schedules

List of all BackupSchedule resources:
- Schedule name
- Cron expression and timezone
- Target VM
- Provider
- Status (Active/Suspended)
- Success/failure counts
- Next scheduled run time

### Restore Jobs

Track VM restorations:
- Restore job name
- Target VM name
- Provider
- Source backup reference
- Phase and progress
- Power-on status
- Duration

### Carbon Stats

Sustainability metrics:
- Number of carbon-aware backups
- Average carbon intensity (gCO₂/kWh)
- Estimated CO₂ savings (kg)
- Delayed backups count
- Average delay hours

## API Endpoints

The dashboard exposes several API endpoints for programmatic access:

### General Metrics

```bash
# Get all Kubernetes metrics
curl http://localhost:8080/api/k8s/metrics

# Get cluster information
curl http://localhost:8080/api/k8s/cluster
```

### BackupJobs

```bash
# List all BackupJobs
curl http://localhost:8080/api/k8s/backupjobs

# Get specific BackupJob
curl http://localhost:8080/api/k8s/backupjobs/<name>
```

### BackupSchedules

```bash
# List all BackupSchedules
curl http://localhost:8080/api/k8s/backupschedules
```

### RestoreJobs

```bash
# List all RestoreJobs
curl http://localhost:8080/api/k8s/restorejobs
```

### Carbon Statistics

```bash
# Get carbon-aware statistics
curl http://localhost:8080/api/k8s/carbon
```

### Storage Statistics

```bash
# Get storage statistics
curl http://localhost:8080/api/k8s/storage
```

## WebSocket Support

For real-time updates, the dashboard supports WebSocket connections:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/k8s');

ws.onmessage = (event) => {
    const metrics = JSON.parse(event.data);
    console.log('Received update:', metrics);
};
```

## Configuration

### Dashboard Settings

Configure the dashboard in your `config.yaml`:

```yaml
dashboard:
  enabled: true
  port: 8080
  update_interval: 5s
  max_clients: 100

kubernetes:
  kubeconfig: ~/.kube/config
  namespace: ""  # empty = all namespaces
```

### Kubernetes Client

The dashboard automatically detects Kubernetes configuration:

1. **In-cluster config**: When running inside Kubernetes
2. **KUBECONFIG env**: Falls back to `$KUBECONFIG`
3. **Default location**: Uses `~/.kube/config`

## Customization

### Theming

The dashboard uses CSS variables for theming. Override in `/static/css/k8s.css`:

```css
:root {
    --orange: #f97316;
    --blue: #3b82f6;
    --green: #10b981;
    --gray-900: #0f172a;
}
```

### Refresh Interval

Adjust the auto-refresh interval in `k8s-dashboard.js`:

```javascript
// Update every 5 seconds (default)
setInterval(fetchAndUpdate, 5000);

// Or change to 10 seconds
setInterval(fetchAndUpdate, 10000);
```

## Troubleshooting

### Dashboard Not Loading

1. **Check operator is running**:
   ```bash
   kubectl get pods -n hypersdk-system
   ```

2. **Verify port-forward**:
   ```bash
   kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080
   ```

3. **Check logs**:
   ```bash
   kubectl logs -n hypersdk-system deployment/hypersdk-operator
   ```

### No Data Showing

1. **Verify CRDs are installed**:
   ```bash
   kubectl get crds | grep hypersdk
   ```

2. **Check for resources**:
   ```bash
   kubectl get backupjobs --all-namespaces
   kubectl get backupschedules --all-namespaces
   kubectl get restorejobs --all-namespaces
   ```

3. **Create test resources**:
   ```bash
   hyperctl k8s -op backup-create -vm test-vm -bucket test | kubectl apply -f -
   ```

### Connection Issues

1. **Check kubeconfig**:
   ```bash
   kubectl config current-context
   kubectl cluster-info
   ```

2. **Verify RBAC permissions**:
   ```bash
   kubectl auth can-i get backupjobs --as=system:serviceaccount:hypersdk-system:hypersdk-operator
   ```

3. **Test API connectivity**:
   ```bash
   curl http://localhost:8080/api/k8s/cluster
   ```

## Security

### Authentication

The dashboard currently supports:
- In-cluster ServiceAccount authentication
- Kubeconfig-based authentication

For production deployments, consider adding:
- OAuth2 proxy
- RBAC-based access control
- TLS/HTTPS
- Network policies

### RBAC

The operator requires these permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hypersdk-operator
rules:
- apiGroups: ["hypersdk.io"]
  resources: ["backupjobs", "backupschedules", "restorejobs"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["pods", "nodes", "namespaces"]
  verbs: ["get", "list"]
```

## Performance

### Metrics Collection

The dashboard collects metrics every 5 seconds. For large clusters:

1. **Increase update interval**:
   ```yaml
   dashboard:
     update_interval: 10s
   ```

2. **Limit namespace scope**:
   ```yaml
   kubernetes:
     namespace: "production"  # Watch only one namespace
   ```

3. **Enable caching**:
   The dashboard caches cluster info for 1 minute by default.

### Resource Usage

Typical resource usage:
- Memory: ~50MB base + ~1MB per 100 resources
- CPU: ~0.1 cores (burst to 0.5 during updates)
- Network: ~10KB/s for metrics updates

## Integration

### Grafana

Export metrics for Grafana:

```bash
# Prometheus format endpoint (coming soon)
curl http://localhost:8080/metrics
```

### Alerts

Configure alerts in your monitoring system:

```yaml
# Example Prometheus alert
groups:
- name: hypersdk
  rules:
  - alert: BackupJobFailed
    expr: hypersdk_backupjob_failed_total > 0
    annotations:
      summary: "BackupJob {{ $labels.name }} failed"
```

### CLI Integration

The dashboard complements the CLI:

```bash
# CLI commands
hyperctl k8s -op backup-list
hyperctl k8s -op status

# Dashboard equivalent
curl http://localhost:8080/api/k8s/backupjobs
curl http://localhost:8080/api/k8s/cluster
```

## Development

### Adding New Metrics

1. **Update `k8s_dashboard.go`**:
   ```go
   type K8sMetrics struct {
       NewMetric int `json:"new_metric"`
   }
   ```

2. **Update collection logic**:
   ```go
   func (kd *K8sDashboard) collectK8sMetrics(ctx context.Context) {
       // Fetch and populate new metric
   }
   ```

3. **Update frontend** (`k8s-dashboard.js`):
   ```javascript
   function updateNewMetric(metrics) {
       document.getElementById('new-metric').textContent = metrics.new_metric;
   }
   ```

### Testing

```bash
# Run dashboard tests
go test ./daemon/dashboard/... -v

# Load test
ab -n 1000 -c 10 http://localhost:8080/api/k8s/metrics
```

## Roadmap

- [ ] Real-time WebSocket updates
- [ ] Advanced filtering and search
- [ ] Export to CSV/JSON
- [ ] Custom dashboard widgets
- [ ] Multi-cluster support
- [ ] Historical data visualization
- [ ] Alert configuration UI
- [ ] Backup restore wizard
- [ ] Schedule builder UI

## Support

For issues and questions:
- GitHub Issues: https://github.com/ssahani/hypersdk/issues
- Documentation: See `docs/KUBERNETES_INTEGRATION_PROGRESS.md`
- CLI Help: `hyperctl k8s -op help`

## License

SPDX-License-Identifier: LGPL-3.0-or-later
