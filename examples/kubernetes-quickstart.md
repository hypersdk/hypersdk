# Kubernetes Quick Start Guide

Deploy HyperSDK to Kubernetes with Kustomize in under 10 minutes.

## Prerequisites

- Kubernetes 1.24+ cluster
- kubectl configured
- Storage class for PVCs
- 2GB RAM, 1 CPU minimum

## Quick Start (Development)

```bash
# Clone repository
git clone https://github.com/ssahani/hypersdk.git
cd hypersdk

# Configure secrets
cp deployments/kubernetes/base/secrets.yaml.example \
   deployments/kubernetes/overlays/development/secrets.yaml

# Edit with your cloud provider credentials
vim deployments/kubernetes/overlays/development/secrets.yaml
```

**Minimal secrets configuration:**

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

**Deploy to Kubernetes:**

```bash
# Deploy using script
./deployments/scripts/deploy-k8s.sh development

# Or deploy manually
kubectl create namespace hypersdk
kubectl apply -f deployments/kubernetes/overlays/development/secrets.yaml
kubectl apply -k deployments/kubernetes/overlays/development

# Wait for deployment
kubectl rollout status deployment/hypervisord -n hypersdk

# Check pods
kubectl get pods -n hypersdk
```

## Access the Service

### Port Forward (Development)

```bash
# Forward API port
kubectl port-forward -n hypersdk svc/hypervisord 8080:8080

# In another terminal, access services
curl http://localhost:8080/health
open http://localhost:8080/web/dashboard/

# Forward metrics port
kubectl port-forward -n hypersdk svc/hypervisord 8081:8081
curl http://localhost:8081/metrics
```

### LoadBalancer (Cloud)

If using LoadBalancer service:

```bash
# Get external IP
kubectl get svc hypervisord-external -n hypersdk

# Wait for EXTERNAL-IP
NAME                   TYPE           EXTERNAL-IP      PORT(S)
hypervisord-external   LoadBalancer   34.123.45.67     80:30080/TCP

# Access via external IP
curl http://34.123.45.67/health
```

## Quick Test

```bash
# Check daemon status
kubectl exec -n hypersdk deployment/hypervisord -- \
  curl -s http://localhost:8080/status | jq

# Submit export job
kubectl exec -n hypersdk deployment/hypervisord -- \
  curl -X POST http://localhost:8080/jobs/submit \
    -H "Content-Type: application/json" \
    -d '{"vm":"/datacenter/vm/test","output":"/exports/test"}'

# Query jobs
kubectl exec -n hypersdk deployment/hypervisord -- \
  curl -s http://localhost:8080/jobs/query | jq
```

## Staging Deployment

Deploy to staging with Ingress and autoscaling:

```bash
# Configure secrets for staging
cp deployments/kubernetes/base/secrets.yaml.example \
   deployments/kubernetes/overlays/staging/secrets.yaml
vim deployments/kubernetes/overlays/staging/secrets.yaml

# Update Ingress hostname
vim deployments/kubernetes/overlays/staging/ingress.yaml
# Change: hypersdk-staging.example.com to your domain

# Deploy to staging
./deployments/scripts/deploy-k8s.sh staging

# Check ingress
kubectl get ingress -n hypersdk

# Access via ingress (after DNS configured)
curl https://hypersdk-staging.example.com/health
```

## Production Deployment

Production includes NetworkPolicy, resource limits, and HPA:

```bash
# Configure secrets
cp deployments/kubernetes/base/secrets.yaml.example \
   deployments/kubernetes/overlays/production/secrets.yaml
vim deployments/kubernetes/overlays/production/secrets.yaml

# Update Ingress hostname
vim deployments/kubernetes/overlays/production/ingress.yaml
# Change: hypersdk.example.com to your domain

# Deploy to production
./deployments/scripts/deploy-k8s.sh production

# Verify deployment
kubectl get all -n hypersdk
kubectl get pvc -n hypersdk
kubectl get networkpolicy -n hypersdk
kubectl get hpa -n hypersdk
```

## Monitoring with Prometheus Operator

If you have Prometheus Operator installed:

```bash
# Apply monitoring resources
kubectl apply -f deployments/kubernetes/monitoring/servicemonitor.yaml
kubectl apply -f deployments/kubernetes/monitoring/prometheusrule.yaml

# Verify
kubectl get servicemonitor -n hypersdk
kubectl get prometheusrule -n hypersdk

# Check metrics in Prometheus
# Query: hypersdk_http_requests_total
```

## Using with Different Environments

### Minikube

```bash
# Start minikube
minikube start --memory=4096 --cpus=2

# Enable ingress addon
minikube addons enable ingress

# Deploy
./deployments/scripts/deploy-k8s.sh development

# Access via minikube service
minikube service hypervisord-external -n hypersdk
```

### Kind (Kubernetes in Docker)

```bash
# Create cluster with ingress
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
  - containerPort: 443
    hostPort: 443
