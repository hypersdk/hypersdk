# K3d Quick Start Guide

Deploy HyperSDK to a local k3d (k3s in Docker) cluster in under 10 minutes.

## What is k3d?

k3d is a lightweight wrapper to run k3s (Rancher Lab's minimal Kubernetes distribution) in Docker. It's perfect for:
- Local development and testing
- CI/CD pipelines
- Learning Kubernetes
- Quick proof-of-concepts

## Prerequisites

- **Docker**: 20.10+ or Podman 4.0+
- **k3d**: 5.0+ ([Installation Guide](https://k3d.io/#installation))
- **kubectl**: 1.24+ ([Installation Guide](https://kubernetes.io/docs/tasks/tools/))
- **Resources**: 4GB RAM, 2 CPU cores minimum

## Quick Start

### 1. Create k3d Cluster

```bash
# Create cluster with LoadBalancer and port mappings
k3d cluster create hypersdk \
  --agents 2 \
  --port "8080:80@loadbalancer" \
  --port "9090:9090@server:0" \
  --k3s-arg "--disable=traefik@server:0"

# Verify cluster is ready
kubectl cluster-info
kubectl get nodes
```

Expected output:
```
NAME                    STATUS   ROLES                  AGE
k3d-hypersdk-server-0   Ready    control-plane,master   30s
k3d-hypersdk-agent-0    Ready    <none>                 25s
k3d-hypersdk-agent-1    Ready    <none>                 25s
```

### 2. Deploy HyperSDK

```bash
# Clone repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Create namespace
kubectl create namespace hypersdk

# Create secrets (configure your credentials)
cat <<EOF | kubectl apply -f -
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
  insecure: "1"
EOF

# Deploy using Kustomize
kubectl apply -k deployments/kubernetes/overlays/development

# Wait for deployment
kubectl wait --for=condition=ready pod \
  -l app=hypervisord \
  -n hypersdk \
  --timeout=300s
```

### 3. Verify Deployment

```bash
# Check pod status
kubectl get pods -n hypersdk

# Check services
kubectl get svc -n hypersdk

# View logs
kubectl logs -n hypersdk -l app=hypervisord --tail=50
```

Expected pod status:
```
NAME                          READY   STATUS    RESTARTS   AGE
hypervisord-xxxxxxxxx-xxxxx   1/1     Running   0          2m
```

### 4. Access HyperSDK

#### Option A: Via LoadBalancer (Recommended)

```bash
# Get LoadBalancer IP
LB_IP=$(kubectl get svc hypervisord-external -n hypersdk \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test API
curl http://$LB_IP/health
curl http://$LB_IP/status | jq

# Open Web Dashboard
open http://$LB_IP/web/dashboard/
```

#### Option B: Via Port Forward

```bash
# Forward API port
kubectl port-forward -n hypersdk svc/hypervisord 8080:8080 &

# Test API
curl http://localhost:8080/health
curl http://localhost:8080/status | jq

# Open Web Dashboard
open http://localhost:8080/web/dashboard/
```

### 5. Test VM Export

Submit a test export job:

```bash
LB_IP=$(kubectl get svc hypervisord-external -n hypersdk \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Submit export job
curl -X POST http://$LB_IP/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm": "/Datacenter/vm/test-vm",
    "output": "/exports/test-vm",
    "format": "ova",
    "compress": true
  }'

# Monitor job
curl http://$LB_IP/jobs/query | jq
```

## Advanced Setup

### Create Cluster with Custom Resources

```bash
# Create cluster with more resources
k3d cluster create hypersdk-dev \
  --agents 3 \
  --servers 1 \
  --port "8080:80@loadbalancer" \
  --port "8443:443@loadbalancer" \
  --port "9090:9090@server:0" \
  --k3s-arg "--disable=traefik@server:0" \
  --volume "/tmp/hypersdk-exports:/exports@all" \
  --registry-create hypersdk-registry:5000

# Set kubeconfig
export KUBECONFIG="$(k3d kubeconfig write hypersdk-dev)"
```

### Deploy with Monitoring Stack

```bash
# Install Prometheus Operator
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml

# Wait for operator
kubectl wait --for=condition=ready pod \
  -l app.kubernetes.io/name=prometheus-operator \
  -n default \
  --timeout=300s

# Deploy HyperSDK with monitoring
kubectl apply -k deployments/kubernetes/overlays/development
kubectl apply -f deployments/kubernetes/monitoring/

# Port forward Prometheus
kubectl port-forward -n hypersdk svc/prometheus 9090:9090 &
open http://localhost:9090
```

### Deploy with Persistent Storage

By default, k3d uses `local-path` storage. For production-like testing:

```bash
# Create cluster with volume mount
k3d cluster create hypersdk-storage \
  --agents 2 \
  --volume "/data/k3d-storage:/var/lib/rancher/k3s/storage@all"

# Deploy with larger PVCs
kubectl apply -k deployments/kubernetes/overlays/staging
```

## Working with k3d

### Useful Commands

```bash
# List clusters
k3d cluster list

# Stop cluster (preserves data)
k3d cluster stop hypersdk

# Start cluster
k3d cluster start hypersdk

# Delete cluster
k3d cluster delete hypersdk

# Get kubeconfig
k3d kubeconfig get hypersdk

# Import image to cluster
k3d image import ghcr.io/ssahani/hypersdk-hypervisord:latest -c hypersdk

# Access cluster nodes
docker exec -it k3d-hypersdk-server-0 sh
```

### Access Cluster Resources

```bash
# Get cluster info
kubectl cluster-info

# View all resources
kubectl get all -n hypersdk

# Describe pod
kubectl describe pod -n hypersdk -l app=hypervisord

# Execute command in pod
kubectl exec -it -n hypersdk deployment/hypervisord -- sh

# View persistent volumes
kubectl get pv,pvc -n hypersdk
```

### Debug Issues

```bash
# Check pod logs
kubectl logs -n hypersdk -l app=hypervisord --tail=100 -f

# Check events
kubectl get events -n hypersdk --sort-by='.lastTimestamp'

# Check pod details
kubectl describe pod -n hypersdk -l app=hypervisord

# Check resource usage
kubectl top pods -n hypersdk
kubectl top nodes

# Shell into pod
kubectl exec -it -n hypersdk deployment/hypervisord -- /bin/sh
```

## Integration Testing

### Test with Local Docker Images

```bash
# Build image locally
cd hypersdk
docker build -f deployments/docker/dockerfiles/Dockerfile.hypervisord \
  -t hypersdk/hypervisord:dev .

# Import to k3d
k3d image import hypersdk/hypervisord:dev -c hypersdk

# Update deployment to use local image
kubectl set image deployment/hypervisord \
  hypervisord=hypersdk/hypervisord:dev \
  -n hypersdk

# Verify
kubectl rollout status deployment/hypervisord -n hypersdk
```

### Test NFS Storage

```bash
# Deploy NFS server in cluster
kubectl create namespace nfs-server

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nfs-server
  namespace: nfs-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nfs-server
  template:
    metadata:
      labels:
        app: nfs-server
    spec:
      containers:
      - name: nfs-server
        image: k8s.gcr.io/volume-nfs:0.8
        ports:
        - name: nfs
          containerPort: 2049
        securityContext:
          privileged: true
        volumeMounts:
        - name: storage
          mountPath: /exports
      volumes:
      - name: storage
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: nfs-server
  namespace: nfs-server
spec:
  ports:
  - port: 2049
  selector:
    app: nfs-server
EOF

# Get NFS server IP
NFS_IP=$(kubectl get svc nfs-server -n nfs-server -o jsonpath='{.spec.clusterIP}')

# Create NFS PV
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: hypersdk-nfs
spec:
  capacity:
    storage: 50Gi
  accessModes:
    - ReadWriteMany
  nfs:
    server: $NFS_IP
    path: "/"
  mountOptions:
    - nfsvers=4.1
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-exports-nfs
  namespace: hypersdk
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 50Gi
  volumeName: hypersdk-nfs
  storageClassName: ""
EOF
```

### Load Testing

```bash
# Install hey (HTTP load tester)
go install github.com/rakyll/hey@latest

# Get LoadBalancer IP
LB_IP=$(kubectl get svc hypervisord-external -n hypersdk \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Load test health endpoint
hey -z 30s -c 10 http://$LB_IP/health

# Load test status endpoint
hey -z 30s -c 10 http://$LB_IP/status

# Monitor pod during load test
kubectl top pods -n hypersdk --watch
```

## CI/CD Integration

### GitHub Actions Example

```yaml
name: K3d Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Create k3d cluster
        uses: AbsaOSS/k3d-action@v2
        with:
          cluster-name: test
          args: >-
            --agents 2
            --port "8080:80@loadbalancer"

      - name: Deploy HyperSDK
        run: |
          kubectl create namespace hypersdk
          kubectl apply -k deployments/kubernetes/overlays/development
          kubectl wait --for=condition=ready pod -l app=hypervisord -n hypersdk --timeout=300s

      - name: Test deployment
        run: |
          LB_IP=$(kubectl get svc hypervisord-external -n hypersdk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
          curl -f http://$LB_IP/health
          curl -f http://$LB_IP/status
```

### GitLab CI Example

```yaml
test-k3d:
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - apk add --no-cache curl
    - curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash
    - curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
    - chmod +x kubectl && mv kubectl /usr/local/bin/
  script:
    - k3d cluster create test --agents 2
    - export KUBECONFIG="$(k3d kubeconfig write test)"
    - kubectl create namespace hypersdk
    - kubectl apply -k deployments/kubernetes/overlays/development
    - kubectl wait --for=condition=ready pod -l app=hypervisord -n hypersdk --timeout=300s
    - LB_IP=$(kubectl get svc hypervisord-external -n hypersdk -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
    - curl -f http://$LB_IP/health
```

## Performance Tips

### Optimize k3d Cluster

```bash
# Create cluster with performance optimizations
k3d cluster create hypersdk-perf \
  --agents 3 \
  --k3s-arg "--disable=traefik@server:0" \
  --k3s-arg "--disable=servicelb@server:0" \
  --k3s-arg "--disable=metrics-server@server:0" \
  --volume "/dev/shm:/dev/shm@all" \
  --memory 8g \
  --cpus 4
```

### Resource Limits

Adjust resource limits for testing:

```bash
# Edit deployment
kubectl patch deployment hypervisord -n hypersdk --type='json' \
  -p='[{
    "op": "replace",
    "path": "/spec/template/spec/containers/0/resources",
    "value": {
      "requests": {"memory": "512Mi", "cpu": "500m"},
      "limits": {"memory": "2Gi", "cpu": "2000m"}
    }
  }]'
```

## Cleanup

```bash
# Delete HyperSDK deployment
kubectl delete namespace hypersdk

# Stop cluster (preserves for later)
k3d cluster stop hypersdk

# Delete cluster completely
k3d cluster delete hypersdk

# Remove all k3d clusters
k3d cluster delete --all
```

## Troubleshooting

### Cluster Won't Start

```bash
# Check Docker is running
docker ps

# Check port conflicts
lsof -i :8080
lsof -i :6443

# Delete and recreate
k3d cluster delete hypersdk
k3d cluster create hypersdk --agents 2
```

### Pods Stuck in Pending

```bash
# Check events
kubectl get events -n hypersdk --sort-by='.lastTimestamp'

# Check node resources
kubectl describe nodes

# Check storage
kubectl get pvc -n hypersdk
kubectl describe pvc -n hypersdk
```

### Cannot Access LoadBalancer

```bash
# Check service
kubectl get svc -n hypersdk hypervisord-external

# Get LoadBalancer IP (may take 30-60s)
kubectl get svc -n hypersdk hypervisord-external -o wide

# Use NodePort as fallback
NODE_PORT=$(kubectl get svc hypervisord-external -n hypersdk -o jsonpath='{.spec.ports[0].nodePort}')
curl http://localhost:$NODE_PORT/health
```

### Image Pull Errors

```bash
# Import image manually
docker pull ghcr.io/ssahani/hypersdk-hypervisord:latest
k3d image import ghcr.io/ssahani/hypersdk-hypervisord:latest -c hypersdk

# Check image availability
kubectl describe pod -n hypersdk -l app=hypervisord | grep -A 5 "Events:"
```

## Best Practices

1. **Use Namespaces**: Keep k3d deployments isolated in dedicated namespaces
2. **Port Mappings**: Map commonly used ports during cluster creation
3. **Volume Mounts**: Mount host directories for persistent data
4. **Resource Limits**: Set appropriate limits to avoid resource exhaustion
5. **Clean Up**: Delete clusters when done to free resources
6. **Kubeconfig**: Use `export KUBECONFIG=$(k3d kubeconfig write <name>)` for easy cluster switching

## Next Steps

- [Kubernetes Deployment Guide](../deployments/kubernetes/README.md) - Production deployment
- [NFS Shared Storage](../docs/tutorials/nfs-shared-storage.md) - Multi-environment storage
- [Configuration Guide](../docs/tutorials/configuration.md) - Advanced configuration
- [API Documentation](../docs/api/README.md) - REST API reference

## Resources

- [k3d Documentation](https://k3d.io)
- [k3s Documentation](https://docs.k3s.io)
- [kubectl Cheat Sheet](https://kubernetes.io/docs/reference/kubectl/cheatsheet/)
