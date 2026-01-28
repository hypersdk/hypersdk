# HyperSDK Helm Chart

Official Helm chart for deploying HyperSDK to Kubernetes.

## TL;DR

```bash
helm repo add hypersdk https://ssahani.github.io/hypersdk/charts
helm install my-hypersdk hypersdk/hypersdk
```

Or install from local chart:

```bash
helm install my-hypersdk ./deployments/helm/hypersdk \
  --set credentials.vsphere.enabled=true \
  --set credentials.vsphere.url="https://vcenter.example.com/sdk" \
  --set credentials.vsphere.username="admin" \
  --set credentials.vsphere.password="password"
```

## Introduction

This chart bootstraps a HyperSDK deployment on a Kubernetes cluster using the Helm package manager.

## Prerequisites

- Kubernetes 1.24+
- Helm 3.8+
- PV provisioner support in the underlying infrastructure
- Cloud provider credentials (vSphere, AWS, Azure, or GCP)

## Installing the Chart

### Quick Start

```bash
# Add Helm repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/charts
helm repo update

# Install with vSphere credentials
helm install my-hypersdk hypersdk/hypersdk \
  --create-namespace \
  --namespace hypersdk \
  --set credentials.vsphere.enabled=true \
  --set credentials.vsphere.url="https://vcenter.example.com/sdk" \
  --set credentials.vsphere.username="administrator@vsphere.local" \
  --set credentials.vsphere.password="your-password"
```

### Install with Custom Values

```bash
# Create custom values file
cat > my-values.yaml <<EOF
credentials:
  vsphere:
    enabled: true
    url: "https://vcenter.example.com/sdk"
    username: "administrator@vsphere.local"
    password: "change-me"

replicaCount: 1

resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "4Gi"
    cpu: "2000m"

persistence:
  exports:
    size: 1Ti

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: hypersdk.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: hypersdk-tls
      hosts:
        - hypersdk.example.com
EOF

# Install with custom values
helm install my-hypersdk hypersdk/hypersdk \
  -f my-values.yaml \
  --namespace hypersdk \
  --create-namespace
```

### Install from Local Chart

```bash
cd /path/to/hypersdk

helm install my-hypersdk ./deployments/helm/hypersdk \
  -f my-values.yaml \
  --namespace hypersdk \
  --create-namespace
```

## Uninstalling the Chart

```bash
helm uninstall my-hypersdk --namespace hypersdk
```

This removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration

### Common Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas (use >1 only with PostgreSQL) | `1` |
| `image.repository` | Image repository | `ghcr.io/ssahani/hypersdk-hypervisord` |
| `image.tag` | Image tag | `latest` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### Service Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `service.type` | Service type | `ClusterIP` |
| `service.port` | Service port | `8080` |
| `service.metricsPort` | Metrics port | `8081` |
| `externalService.enabled` | Enable LoadBalancer service | `true` |
| `externalService.type` | External service type | `LoadBalancer` |

### Ingress Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `nginx` |
| `ingress.annotations` | Ingress annotations | `{}` |
| `ingress.hosts` | Ingress hosts | `[]` |
| `ingress.tls` | Ingress TLS configuration | `[]` |

### Storage Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.data.enabled` | Enable data persistence | `true` |
| `persistence.data.storageClass` | Storage class | `""` (cluster default) |
| `persistence.data.size` | Data volume size | `10Gi` |
| `persistence.exports.enabled` | Enable exports persistence | `true` |
| `persistence.exports.size` | Exports volume size | `500Gi` |
| `persistence.exports.accessMode` | Access mode | `ReadWriteOnce` |

### Credentials Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `credentials.vsphere.enabled` | Enable vSphere credentials | `false` |
| `credentials.vsphere.url` | vCenter URL | `""` |
| `credentials.vsphere.username` | vSphere username | `""` |
| `credentials.vsphere.password` | vSphere password | `""` |
| `credentials.vsphere.existingSecret` | Use existing secret | `""` |
| `credentials.aws.enabled` | Enable AWS credentials | `false` |
| `credentials.aws.accessKeyId` | AWS access key | `""` |
| `credentials.aws.secretAccessKey` | AWS secret key | `""` |
| `credentials.aws.region` | AWS region | `us-east-1` |
| `credentials.azure.enabled` | Enable Azure credentials | `false` |
| `credentials.gcp.enabled` | Enable GCP credentials | `false` |

### Autoscaling Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `false` |
| `autoscaling.minReplicas` | Minimum replicas | `2` |
| `autoscaling.maxReplicas` | Maximum replicas | `10` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU | `70` |
| `autoscaling.targetMemoryUtilizationPercentage` | Target memory | `80` |

### Monitoring Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `monitoring.serviceMonitor.enabled` | Enable ServiceMonitor | `false` |
| `monitoring.serviceMonitor.interval` | Scrape interval | `30s` |
| `monitoring.prometheusRule.enabled` | Enable PrometheusRule | `false` |

## Examples

### Example 1: Development Environment

```yaml
# dev-values.yaml
replicaCount: 1

credentials:
  vsphere:
    enabled: true
    url: "https://vcenter.dev.example.com/sdk"
    username: "admin"
    password: "dev-password"

resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "1Gi"
    cpu: "500m"

persistence:
  data:
    size: 5Gi
  exports:
    size: 50Gi

config:
  logLevel: debug
```

Install:
```bash
helm install dev hypersdk/hypersdk \
  -f dev-values.yaml \
  --namespace hypersdk-dev \
  --create-namespace
```

### Example 2: Production with HA (requires PostgreSQL)

```yaml
# prod-values.yaml
replicaCount: 3

credentials:
  vsphere:
    enabled: true
    existingSecret: "vsphere-prod-credentials"

resources:
  requests:
    memory: "1Gi"
    cpu: "500m"
  limits:
    memory: "4Gi"
    cpu: "2000m"

persistence:
  data:
    storageClass: "ssd"
    size: 20Gi
  exports:
    storageClass: "standard"
    size: 2Ti
    # For RWX storage:
    # accessMode: ReadWriteMany

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

podDisruptionBudget:
  enabled: true
  minAvailable: 2

monitoring:
  serviceMonitor:
    enabled: true
    labels:
      release: prometheus

networkPolicy:
  enabled: true

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: hypersdk.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: hypersdk-tls
      hosts:
        - hypersdk.example.com

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/name
                operator: In
                values:
                  - hypersdk
          topologyKey: topology.kubernetes.io/zone
```

Install:
```bash
# Create secret first
kubectl create secret generic vsphere-prod-credentials \
  --from-literal=url="https://vcenter.example.com/sdk" \
  --from-literal=username="prod-user" \
  --from-literal=password="prod-password" \
  --from-literal=insecure="0" \
  -n hypersdk

# Install chart
helm install prod hypersdk/hypersdk \
  -f prod-values.yaml \
  --namespace hypersdk \
  --create-namespace
```

### Example 3: Multi-Cloud Setup

```yaml
# multi-cloud-values.yaml
credentials:
  vsphere:
    enabled: true
    url: "https://vcenter.example.com/sdk"
    username: "admin"
    password: "vsphere-pass"

  aws:
    enabled: true
    accessKeyId: "AKIA..."
    secretAccessKey: "secret..."
    region: "us-east-1"

  azure:
    enabled: true
    subscriptionId: "00000000-0000-0000-0000-000000000000"
    tenantId: "00000000-0000-0000-0000-000000000000"
    clientId: "00000000-0000-0000-0000-000000000000"
    clientSecret: "azure-secret"

  gcp:
    enabled: true
    projectId: "my-project"
    serviceAccountJSON: |
      {
        "type": "service_account",
        ...
      }

persistence:
  exports:
    size: 2Ti
```

### Example 4: GKE with Workload Identity

```yaml
# gke-values.yaml
serviceAccount:
  create: true
  annotations:
    iam.gke.io/gcp-service-account: hypersdk@PROJECT_ID.iam.gserviceaccount.com

persistence:
  data:
    storageClass: "standard-rwo"
  exports:
    storageClass: "standard-rwo"
    size: 1Ti

externalService:
  type: LoadBalancer
  annotations:
    cloud.google.com/load-balancer-type: "Internal"

credentials:
  vsphere:
    enabled: true
    existingSecret: "vsphere-credentials"
```

### Example 5: EKS with IRSA

```yaml
# eks-values.yaml
serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/hypersdk-role

persistence:
  data:
    storageClass: "gp3"
  exports:
    storageClass: "gp3"
    size: 1Ti

credentials:
  vsphere:
    enabled: true
    existingSecret: "vsphere-credentials"
```

## Upgrading

### To 0.2.0

Version 0.2.0 introduces breaking changes:

- `config.webhooks` changed from map to array
- New required init container for permissions

Upgrade:
```bash
helm upgrade my-hypersdk hypersdk/hypersdk \
  -f my-values.yaml \
  --namespace hypersdk
```

## Troubleshooting

### Pods stuck in Pending

```bash
# Check PVC status
kubectl get pvc -n hypersdk

# Check events
kubectl describe pod -n hypersdk -l app=hypervisord
```

### Cannot access LoadBalancer

```bash
# Check service
kubectl get svc -n hypersdk

# Check LoadBalancer status
kubectl describe svc my-hypersdk-external -n hypersdk
```

### Credentials not working

```bash
# Check secret
kubectl get secret -n hypersdk

# Verify secret data
kubectl get secret my-hypersdk-vsphere -n hypersdk -o yaml
```

## Further Information

- [Kubernetes Deployment Guide](../../docs/guides/upstream-kubernetes.md)
- [Configuration Reference](../../docs/tutorials/configuration.md)
- [API Documentation](../../docs/api/README.md)

## License

LGPL-3.0-or-later
