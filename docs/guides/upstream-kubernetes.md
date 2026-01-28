# Upstream Kubernetes Deployment Guide

Complete guide for deploying HyperSDK to production Kubernetes clusters (GKE, EKS, AKS, vanilla Kubernetes).

## Overview

HyperSDK supports deployment to any Kubernetes 1.24+ cluster:

- **Local Development**: k3d, kind, minikube, Docker Desktop
- **Managed Kubernetes**: GKE, EKS, AKS, DigitalOcean Kubernetes
- **On-Premises**: Vanilla Kubernetes, OpenShift, Rancher RKE2

## Differences: k3d vs Upstream Kubernetes

### k3d (k3s in Docker)

**Pros:**
- ✅ Fast cluster creation (~30 seconds)
- ✅ Minimal resource usage (512MB RAM)
- ✅ Built-in LoadBalancer (via Docker)
- ✅ Built-in local-path storage provisioner
- ✅ Perfect for local development and CI/CD

**Cons:**
- ⚠️ Not production-grade
- ⚠️ Limited to single-host
- ⚠️ SQLite etcd (not full etcd cluster)
- ⚠️ Some features disabled by default
- ⚠️ No HA capabilities

### Upstream Kubernetes

**Pros:**
- ✅ Production-ready with HA support
- ✅ Full Kubernetes API compatibility
- ✅ Multi-node clusters across zones
- ✅ Full etcd cluster with backup/restore
- ✅ Complete ecosystem support
- ✅ Enterprise-grade security

**Cons:**
- ⚠️ More complex setup
- ⚠️ Higher resource requirements (2GB+ RAM per node)
- ⚠️ May need external LoadBalancer/Ingress
- ⚠️ May need storage provisioner configuration
- ⚠️ Higher operational overhead

### Feature Comparison

| Feature | k3d | Upstream K8s |
|---------|-----|--------------|
| Setup Time | 30 seconds | 5-15 minutes |
| Minimum RAM | 512MB | 2GB per node |
| etcd | SQLite | Full etcd cluster |
| LoadBalancer | Built-in | Cloud/MetalLB |
| Storage | local-path | Configurable |
| HA Support | No | Yes |
| Multi-Zone | No | Yes |
| Production Use | No | Yes |
| Cost | Free | Cloud costs apply |

## Cloud Provider Deployments

### Google Kubernetes Engine (GKE)

#### Create Cluster

```bash
# Standard cluster
gcloud container clusters create hypersdk \
  --zone us-central1-a \
  --num-nodes 3 \
  --machine-type n1-standard-2 \
  --disk-size 50 \
  --enable-autoscaling \
  --min-nodes 2 \
  --max-nodes 5 \
  --enable-stackdriver-kubernetes

# Autopilot (hands-off management)
gcloud container clusters create-auto hypersdk \
  --region us-central1

# Get credentials
gcloud container clusters get-credentials hypersdk --zone us-central1-a
```

#### Deploy HyperSDK

```bash
# Create namespace
kubectl create namespace hypersdk

# Create secrets
kubectl create secret generic vsphere-credentials \
  --from-literal=url="https://vcenter.example.com/sdk" \
  --from-literal=username="administrator@vsphere.local" \
  --from-literal=password="your-password" \
  --from-literal=insecure="1" \
  -n hypersdk

# Deploy
kubectl apply -k deployments/kubernetes/overlays/production

# Wait for LoadBalancer
kubectl get svc hypervisord-external -n hypersdk -w
```

#### GKE-Specific Configuration

Create `deployments/kubernetes/overlays/gke/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../base

namespace: hypersdk

# Use GKE storage class
patches:
- patch: |-
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: hypervisord-data-pvc
    spec:
      storageClassName: standard-rwo  # GKE standard persistent disk
  target:
    kind: PersistentVolumeClaim
    name: hypervisord-data-pvc

- patch: |-
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: hypervisord-exports-pvc
    spec:
      storageClassName: standard-rwo
      resources:
        requests:
          storage: 1Ti  # Large exports volume
  target:
    kind: PersistentVolumeClaim
    name: hypervisord-exports-pvc

# Use GKE Workload Identity
- patch: |-
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: hypervisord
      annotations:
        iam.gke.io/gcp-service-account: hypervisord@PROJECT_ID.iam.gserviceaccount.com
  target:
    kind: ServiceAccount
    name: hypervisord

# Production resources
- patch: |-
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: hypervisord
    spec:
      replicas: 3
      template:
        spec:
          containers:
          - name: hypervisord
            resources:
              requests:
                memory: "1Gi"
                cpu: "500m"
              limits:
                memory: "4Gi"
                cpu: "2000m"
  target:
    kind: Deployment
    name: hypervisord

images:
- name: hypersdk/hypervisord
  newName: ghcr.io/ssahani/hypersdk-hypervisord
  newTag: latest

commonAnnotations:
  environment: gke-production
```

Deploy with:
```bash
kubectl apply -k deployments/kubernetes/overlays/gke
```

### Amazon EKS

#### Create Cluster

```bash
# Using eksctl
eksctl create cluster \
  --name hypersdk \
  --region us-east-1 \
  --nodegroup-name standard-workers \
  --node-type t3.medium \
  --nodes 3 \
  --nodes-min 2 \
  --nodes-max 5 \
  --managed \
  --with-oidc \
  --ssh-access \
  --ssh-public-key ~/.ssh/id_rsa.pub

# Update kubeconfig
aws eks update-kubeconfig --region us-east-1 --name hypersdk

# Install EBS CSI driver (for PVCs)
eksctl create addon --name aws-ebs-csi-driver --cluster hypersdk --region us-east-1
```

#### Deploy HyperSDK

```bash
# Create namespace
kubectl create namespace hypersdk

# Deploy
kubectl apply -k deployments/kubernetes/overlays/production
```

#### EKS-Specific Configuration

Create `deployments/kubernetes/overlays/eks/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../base

namespace: hypersdk

# Use EBS storage class
patches:
- patch: |-
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: hypervisord-data-pvc
    spec:
      storageClassName: gp3  # General Purpose SSD
  target:
    kind: PersistentVolumeClaim
    name: hypervisord-data-pvc

- patch: |-
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: hypervisord-exports-pvc
    spec:
      storageClassName: gp3
      resources:
        requests:
          storage: 1Ti
  target:
    kind: PersistentVolumeClaim
    name: hypervisord-exports-pvc

# Use IAM roles for service accounts (IRSA)
- patch: |-
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: hypervisord
      annotations:
        eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/hypersdk-role
  target:
    kind: ServiceAccount
    name: hypervisord

images:
- name: hypersdk/hypervisord
  newName: ghcr.io/ssahani/hypersdk-hypervisord
  newTag: latest

commonAnnotations:
  environment: eks-production
```

### Azure Kubernetes Service (AKS)

#### Create Cluster

```bash
# Create resource group
az group create --name hypersdk-rg --location eastus

# Create AKS cluster
az aks create \
  --resource-group hypersdk-rg \
  --name hypersdk \
  --node-count 3 \
  --node-vm-size Standard_D2s_v3 \
  --enable-managed-identity \
  --enable-cluster-autoscaler \
  --min-count 2 \
  --max-count 5 \
  --generate-ssh-keys

# Get credentials
az aks get-credentials --resource-group hypersdk-rg --name hypersdk
```

#### Deploy HyperSDK

```bash
kubectl create namespace hypersdk
kubectl apply -k deployments/kubernetes/overlays/production
```

#### AKS-Specific Configuration

Create `deployments/kubernetes/overlays/aks/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../base

namespace: hypersdk

# Use Azure Disk
patches:
- patch: |-
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: hypervisord-data-pvc
    spec:
      storageClassName: managed-premium  # Premium SSD
  target:
    kind: PersistentVolumeClaim

# Use Azure Pod Identity
- patch: |-
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: hypervisord
      labels:
        aadpodidbinding: hypersdk-identity
  target:
    kind: ServiceAccount
    name: hypervisord

images:
- name: hypersdk/hypervisord
  newName: ghcr.io/ssahani/hypersdk-hypervisord
  newTag: latest
```

## kind (Kubernetes in Docker)

kind provides a more upstream-like experience than k3d:

### Create Cluster with LoadBalancer Support

```bash
# Install kind
# On Linux:
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# On macOS:
brew install kind

# Create cluster with ingress support
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: hypersdk
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
- role: worker
- role: worker
- role: worker
EOF
```

### Install MetalLB for LoadBalancer

```bash
# Install MetalLB
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.12/config/manifests/metallb-native.yaml

# Wait for pods
kubectl wait --namespace metallb-system \
  --for=condition=ready pod \
  --selector=app=metallb \
  --timeout=90s

# Get kind network subnet
docker network inspect kind | jq '.[0].IPAM.Config[0].Subnet' -r

# Configure IP pool (adjust based on your network)
kubectl apply -f - <<EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: default
  namespace: metallb-system
spec:
  addresses:
  - 172.18.255.200-172.18.255.250  # Adjust for your kind network
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: default
  namespace: metallb-system
EOF
```

### Deploy HyperSDK

```bash
kubectl create namespace hypersdk
kubectl apply -k deployments/kubernetes/overlays/development

# Get LoadBalancer IP
LB_IP=$(kubectl get svc hypervisord-external -n hypersdk \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test
curl http://$LB_IP/health
```

## On-Premises Kubernetes

### Prerequisites

- Vanilla Kubernetes 1.24+ cluster
- MetalLB or hardware LoadBalancer
- Storage provisioner (Rook-Ceph, NFS, local-path)
- Ingress controller (nginx, traefik)

### Install MetalLB

```bash
# Install MetalLB
kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.13.12/config/manifests/metallb-native.yaml

# Configure IP pool for your network
kubectl apply -f - <<EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: default
  namespace: metallb-system
spec:
  addresses:
  - 192.168.1.240-192.168.1.250  # Adjust for your network
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: default
  namespace: metallb-system
spec:
  ipAddressPools:
  - default
EOF
```

### Install Storage Provisioner

#### Option 1: local-path (testing only)

```bash
kubectl apply -f https://raw.githubusercontent.com/rancher/local-path-provisioner/v0.0.24/deploy/local-path-storage.yaml
```

#### Option 2: Rook-Ceph (production)

```bash
git clone --single-branch --branch v1.12.0 https://github.com/rook/rook.git
cd rook/deploy/examples
kubectl create -f crds.yaml -f common.yaml -f operator.yaml
kubectl create -f cluster.yaml
```

#### Option 3: NFS

See [NFS Shared Storage Guide](../tutorials/nfs-shared-storage.md)

### Deploy HyperSDK

```bash
kubectl create namespace hypersdk
kubectl apply -k deployments/kubernetes/overlays/production
```

### Alternative: NodePort (without LoadBalancer)

If you can't use LoadBalancer, use NodePort:

```bash
# Patch service to NodePort
kubectl patch svc hypervisord-external -n hypersdk -p '{"spec":{"type":"NodePort","ports":[{"port":80,"targetPort":8080,"nodePort":30080}]}}'

# Access via any node IP
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
curl http://$NODE_IP:30080/health
```

## Production Best Practices

### High Availability

For true HA, migrate from SQLite to PostgreSQL:

```yaml
# Add PostgreSQL dependency
- name: postgresql
  image: postgres:15-alpine
  env:
  - name: POSTGRES_DB
    value: hypersdk
  - name: POSTGRES_USER
    valueFrom:
      secretKeyRef:
        name: postgres-credentials
        key: username
  - name: POSTGRES_PASSWORD
    valueFrom:
      secretKeyRef:
        name: postgres-credentials
        key: password

# Update hypervisord to use PostgreSQL
env:
- name: DATABASE_TYPE
  value: "postgres"
- name: DATABASE_URL
  value: "postgresql://postgres:5432/hypersdk"
```

Then scale replicas:
```bash
kubectl scale deployment hypervisord --replicas=3 -n hypersdk
```

### Ingress with TLS

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# Create Ingress
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hypervisord
  namespace: hypersdk
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - hypersdk.example.com
    secretName: hypersdk-tls
  rules:
  - host: hypersdk.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hypervisord
            port:
              number: 8080
EOF
```

### HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hypervisord
  namespace: hypersdk
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hypervisord
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### NetworkPolicies

See production overlay: `deployments/kubernetes/overlays/production/networkpolicy.yaml`

### Resource Quotas

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: hypersdk-quota
  namespace: hypersdk
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    persistentvolumeclaims: "5"
    requests.storage: 2Ti
```

## Monitoring and Observability

### Prometheus Operator

```bash
# Install Prometheus Operator
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml

# Apply HyperSDK monitoring
kubectl apply -f deployments/kubernetes/monitoring/servicemonitor.yaml
kubectl apply -f deployments/kubernetes/monitoring/prometheusrule.yaml
```

### Grafana

```bash
# Install Grafana
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: grafana
  namespace: hypersdk
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grafana
  namespace: hypersdk
spec:
  selector:
    matchLabels:
      app: grafana
  template:
    metadata:
      labels:
        app: grafana
    spec:
      containers:
      - name: grafana
        image: grafana/grafana:10.2.3
        ports:
        - containerPort: 3000
        volumeMounts:
        - name: storage
          mountPath: /var/lib/grafana
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: grafana
---
apiVersion: v1
kind: Service
metadata:
  name: grafana
  namespace: hypersdk
spec:
  ports:
  - port: 3000
  selector:
    app: grafana
EOF
```

## Migration from k3d

1. **Test with kind first** (closer to upstream):
   ```bash
   # Create kind cluster with MetalLB
   kind create cluster --name test
   # Deploy HyperSDK
   kubectl apply -k deployments/kubernetes/overlays/production
   ```

2. **Update storage classes**:
   - Replace `local-path` with cloud provider storage
   - Adjust PVC sizes for production

3. **Configure LoadBalancer/Ingress**:
   - For cloud: Use cloud LoadBalancer
   - For on-prem: Install MetalLB or use Ingress

4. **Enable monitoring**:
   ```bash
   kubectl apply -f deployments/kubernetes/monitoring/
   ```

5. **Apply production settings**:
   - Resource limits
   - Replicas (if using PostgreSQL)
   - HPA
   - NetworkPolicies

6. **Test thoroughly**:
   ```bash
   ./deployments/scripts/health-check.sh kubernetes --namespace hypersdk
   ```

## Comparison Matrix

| Feature | k3d | kind | GKE | EKS | AKS | On-Prem |
|---------|-----|------|-----|-----|-----|---------|
| Setup Complexity | ⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| Cost | Free | Free | $$$ | $$$ | $$$ | Hardware |
| HA Support | ❌ | ❌ | ✅ | ✅ | ✅ | ✅ |
| LoadBalancer | ✅ | Manual | ✅ | ✅ | ✅ | Manual |
| Storage | local-path | local | GCE PD | EBS | Azure Disk | Manual |
| Upgrades | Manual | Manual | Managed | Managed | Managed | Manual |
| Best For | Dev/CI | Testing | Production | Production | Production | Production |

## Next Steps

- [k3d Quick Start](../../examples/k3d-quickstart.md) - Local testing
- [NFS Architecture](../architecture/nfs-deployment-architecture.md) - Shared storage
- [Configuration Guide](../tutorials/configuration.md) - Advanced config
- [Monitoring](../tutorials/monitoring.md) - Observability

## Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)
- [EKS Documentation](https://docs.aws.amazon.com/eks/)
- [AKS Documentation](https://docs.microsoft.com/azure/aks/)
- [MetalLB](https://metallb.universe.tf/)
- [cert-manager](https://cert-manager.io/)
