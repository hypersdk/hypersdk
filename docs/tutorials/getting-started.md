# Getting Started with HyperSDK

This tutorial guides you through deploying HyperSDK and running your first VM export job.

## Prerequisites

- **Container Runtime**: Docker 20.10+ or Podman 4.0+ (for container deployment)
- **Kubernetes**: K8s 1.24+ with kubectl (for Kubernetes deployment)
- **Cloud Access**: Credentials for at least one supported provider (vSphere, AWS, Azure, GCP, etc.)
- **Resources**: 4GB RAM, 2 CPU cores, 50GB+ disk space

## Option 1: Docker Quick Start (Recommended for First-Time Users)

### Step 1: Pull Pre-built Image

```bash
docker pull ghcr.io/ssahani/hypersdk-hypervisord:latest
```

### Step 2: Create Volumes

```bash
docker volume create hypersdk-data
docker volume create hypersdk-exports
```

### Step 3: Configure vSphere Credentials

Create an environment file:

```bash
cat > hypersdk.env <<EOF
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=administrator@vsphere.local
GOVC_PASSWORD=your-secure-password
GOVC_INSECURE=1
LOG_LEVEL=info
EOF
```

### Step 4: Run the Container

```bash
docker run -d \
  --name hypersdk \
  --env-file hypersdk.env \
  -p 8080:8080 \
  -p 8081:8081 \
  -v hypersdk-data:/data \
  -v hypersdk-exports:/exports \
  ghcr.io/ssahani/hypersdk-hypervisord:latest
```

### Step 5: Verify Deployment

```bash
# Check container status
docker ps | grep hypersdk

# Check health endpoint
curl http://localhost:8080/health

# View logs
docker logs -f hypersdk
```

Expected output from health endpoint:
```json
{
  "status": "healthy",
  "version": "v0.2.0",
  "uptime": "5m30s"
}
```

### Step 6: Access the Dashboard

Open your browser and navigate to:
```
http://localhost:8080/web/dashboard/
```

The dashboard provides:
- VM discovery and browsing
- Export job submission and monitoring
- Job history and status
- System metrics

## Option 2: Docker Compose (Full Stack with Monitoring)

### Step 1: Clone Repository

```bash
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk/deployments/docker
```

### Step 2: Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit with your credentials
vim .env
```

Minimal .env configuration:
```bash
# vSphere credentials
GOVC_URL=https://vcenter.example.com/sdk
GOVC_USERNAME=administrator@vsphere.local
GOVC_PASSWORD=your-password

# Daemon configuration
LOG_LEVEL=info
DAEMON_PORT=8080
METRICS_PORT=8081

# Optional: Redis cache
REDIS_ENABLED=true
REDIS_MAX_MEMORY=256mb
```

### Step 3: Start the Stack

```bash
docker compose up -d
```

This starts:
- **hypervisord** - Main API daemon
- **redis** - Caching layer
- **prometheus** - Metrics collection
- **grafana** - Visualization dashboards

### Step 4: Verify All Services

```bash
docker compose ps
```

Expected output:
```
NAME                      STATUS    PORTS
hypersdk-hypervisord      Up        0.0.0.0:8080->8080/tcp, 0.0.0.0:8081->8081/tcp
hypersdk-redis            Up        0.0.0.0:6379->6379/tcp
hypersdk-prometheus       Up        0.0.0.0:9090->9090/tcp
hypersdk-grafana          Up        0.0.0.0:3000->3000/tcp
```

### Step 5: Access Services

- **API Dashboard**: http://localhost:8080/web/dashboard/
- **Grafana Dashboards**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Metrics Endpoint**: http://localhost:8081/metrics

## Option 3: Kubernetes Deployment

### Step 1: Configure Secrets

```bash
cd hypersdk/deployments/kubernetes

# Copy secrets template
cp base/secrets.yaml.example overlays/development/secrets.yaml

# Edit with your credentials
vim overlays/development/secrets.yaml
```

Example secrets configuration:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vsphere-credentials
  namespace: hypersdk
type: Opaque
stringData:
  url: "https://vcenter.example.com/sdk"
  username: "administrator@vsphere.local"
  password: "your-password"
```

### Step 2: Deploy to Kubernetes

```bash
# Using deployment script
./deployments/scripts/deploy-k8s.sh development

# Or manually with kubectl
kubectl create namespace hypersdk
kubectl apply -f overlays/development/secrets.yaml
kubectl apply -k overlays/development
```

### Step 3: Wait for Deployment

```bash
kubectl rollout status deployment/hypervisord -n hypersdk
```

### Step 4: Port Forward to Access API

```bash
kubectl port-forward -n hypersdk svc/hypervisord 8080:8080
```

In another terminal:
```bash
curl http://localhost:8080/health
```

### Step 5: Access Dashboard

Open browser to: http://localhost:8080/web/dashboard/

## Running Your First VM Export

### Via Web Dashboard

1. Navigate to http://localhost:8080/web/dashboard/
2. Click "Browse VMs" to discover available virtual machines
3. Select a VM from the list
4. Click "Export VM"
5. Configure export options:
   - **Output Path**: `/exports/my-first-export`
   - **Format**: OVA (recommended) or VMDK
   - **Compression**: Enable for smaller file size
6. Click "Submit Job"
7. Monitor progress on the "Jobs" tab

### Via REST API

Submit an export job using curl:

```bash
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm": "/Datacenter/vm/my-test-vm",
    "output": "/exports/my-first-export",
    "format": "ova",
    "compress": true
  }'
```

Expected response:
```json
{
  "job_id": "job-20260130-123456",
  "status": "queued",
  "vm": "/Datacenter/vm/my-test-vm",
  "created_at": "2026-01-30T12:34:56Z"
}
```

### Check Job Status

```bash
# Query specific job
curl http://localhost:8080/jobs/query?id=job-20260130-123456

# List all jobs
curl http://localhost:8080/jobs/query
```

Job status response:
```json
{
  "job_id": "job-20260130-123456",
  "status": "running",
  "progress": 45,
  "vm": "/Datacenter/vm/my-test-vm",
  "output": "/exports/my-first-export",
  "started_at": "2026-01-30T12:35:10Z",
  "estimated_completion": "2026-01-30T12:40:00Z"
}
```

### Monitor Export Progress

Watch the job in real-time:

```bash
# Using watch command
watch -n 2 'curl -s http://localhost:8080/jobs/query?id=job-20260130-123456 | jq'

# Or view logs
docker logs -f hypersdk  # Docker
kubectl logs -f -n hypersdk deployment/hypervisord  # Kubernetes
```

### Access Exported Files

**Docker/Podman:**
```bash
# List exported files
docker exec hypersdk ls -lh /exports/my-first-export

# Copy to host
docker cp hypersdk:/exports/my-first-export ./my-first-export
```

**Kubernetes:**
```bash
# List files in PVC
kubectl exec -n hypersdk deployment/hypervisord -- ls -lh /exports/my-first-export

# Copy to local machine
kubectl cp hypersdk/hypervisord-pod:/exports/my-first-export ./my-first-export
```

## Monitoring and Metrics

### View Prometheus Metrics

```bash
curl http://localhost:8081/metrics
```

Key metrics to monitor:
- `hypersdk_export_jobs_total` - Total export jobs
- `hypersdk_export_jobs_running` - Currently running jobs
- `hypersdk_export_jobs_failed` - Failed jobs
- `hypersdk_export_bytes_transferred` - Bytes transferred
- `hypersdk_http_requests_total` - API request count

### Grafana Dashboards

Access Grafana at http://localhost:3000 (admin/admin)

Pre-configured dashboards:
1. **HyperSDK Overview** - System health and job statistics
2. **Export Performance** - Job duration, throughput, success rate
3. **Resource Usage** - CPU, memory, disk, network

## Troubleshooting

### Container Won't Start

```bash
# Check logs for errors
docker logs hypersdk

# Common issues:
# - Missing or invalid cloud credentials
# - Port 8080/8081 already in use
# - Volume permission issues
```

### Cannot Connect to vSphere

```bash
# Test connectivity from container
docker exec hypersdk curl -k https://vcenter.example.com/sdk

# Verify credentials
docker exec hypersdk env | grep GOVC
```

### Export Job Fails

```bash
# Check job details
curl http://localhost:8080/jobs/query?id=<job-id>

# Common issues:
# - Invalid VM path (case-sensitive)
# - Insufficient disk space in /exports
# - Network connectivity to vSphere
# - Permission issues on datastore
```

### Health Check Fails

```bash
# Check health endpoint
curl -v http://localhost:8080/health

# If unhealthy, check:
# - Database connectivity (SQLite file permissions)
# - Disk space in /data volume
# - Container resource limits (memory/CPU)
```

## Next Steps

- [Configuration Guide](configuration.md) - Advanced configuration options
- [Multi-Cloud Setup](multi-cloud-setup.md) - Configure multiple cloud providers
- [Production Deployment](production-deployment.md) - Deploy to production with HA
- [API Reference](../api/README.md) - Complete REST API documentation
- [Monitoring Setup](monitoring.md) - Set up alerts and dashboards

## Quick Reference

### Useful Commands

```bash
# Docker
docker ps | grep hypersdk                    # Check status
docker logs -f hypersdk                      # View logs
docker exec hypersdk hyperctl status         # Check daemon status
docker compose down && docker compose up -d  # Restart stack

# Kubernetes
kubectl get pods -n hypersdk                 # Check pods
kubectl logs -f -n hypersdk deployment/hypervisord  # View logs
kubectl describe pod -n hypersdk <pod-name>  # Pod details
kubectl delete pod -n hypersdk <pod-name>    # Restart pod

# API
curl http://localhost:8080/health            # Health check
curl http://localhost:8080/status            # Daemon status
curl http://localhost:8080/vms/list          # List VMs
curl http://localhost:8081/metrics           # Prometheus metrics
```

### Default Ports

- **8080** - REST API and Web Dashboard
- **8081** - Prometheus Metrics
- **3000** - Grafana (Docker Compose only)
- **9090** - Prometheus (Docker Compose only)
- **6379** - Redis (Docker Compose only)

### Volume Paths

- `/data` - Database and configuration
- `/exports` - VM export output
- `/config` - Optional configuration files

## Getting Help

- **Documentation**: https://github.com/ssahani/hypersdk/tree/main/docs
- **Issues**: https://github.com/ssahani/hypersdk/issues
- **Examples**: https://github.com/ssahani/hypersdk/tree/main/examples
