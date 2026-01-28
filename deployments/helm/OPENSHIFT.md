# HyperSDK on OpenShift

Complete guide for deploying HyperSDK on Red Hat OpenShift Container Platform.

## 📋 Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [OpenShift-Specific Features](#openshift-specific-features)
- [Accessing the Web Server](#accessing-the-web-server)
- [Deployment Methods](#deployment-methods)
- [Security Configuration](#security-configuration)
- [Networking](#networking)
- [Storage](#storage)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [Production Checklist](#production-checklist)
- [Related Documentation](#related-documentation)
- [References](#references)
- [Support](#support)

## Overview

HyperSDK is fully compatible with Red Hat OpenShift Container Platform (OCP) and provides native support for OpenShift-specific features:

- ✅ **Routes** for ingress (OpenShift's native alternative to Kubernetes Ingress)
- ✅ **SecurityContextConstraints** (SCC) for fine-grained security control
- ✅ **OpenShift Monitoring** integration (cluster-level Prometheus)
- ✅ **Random UID assignment** support (OpenShift security model)
- ✅ **OpenShift Pipelines** (Tekton) compatibility
- ✅ **OpenShift GitOps** (Argo CD) support
- ✅ **OperatorHub** integration (future)

## Prerequisites

### OpenShift Cluster

- **OpenShift Version**: 4.10+ (tested on 4.10, 4.11, 4.12, 4.13, 4.14)
- **Cluster Access**: `cluster-admin` or namespace `admin` role
- **oc CLI**: Installed and configured

```bash
# Verify OpenShift version
oc version

# Login to cluster
oc login https://api.cluster.example.com:6443

# Create project
oc new-project hypersdk
```

### Tools

```bash
# Install Helm 3.8+
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify
helm version
oc version
```

### Resource Requirements

**Minimum** (development):
- CPU: 2 cores
- Memory: 4 GB
- Storage: 20 GB

**Recommended** (production):
- CPU: 4 cores
- Memory: 8 GB
- Storage: 500 GB+ (for VM exports)

## Quick Start

### 1. Install with OpenShift Defaults

```bash
# Add Helm repository
helm repo add hypersdk https://ssahani.github.io/hypersdk/helm-charts
helm repo update

# Install with OpenShift-specific values
helm install hypersdk hypersdk/hypersdk \
  -f https://raw.githubusercontent.com/ssahani/hypersdk/main/deployments/helm/hypersdk/examples/openshift-values.yaml \
  -n hypersdk --create-namespace

# Wait for deployment
oc wait --for=condition=available --timeout=300s \
  deployment/hypersdk -n hypersdk
```

### 2. Verify Installation

```bash
# Check pods
oc get pods -n hypersdk

# Check route
oc get route -n hypersdk

# Get route URL
ROUTE_URL=$(oc get route hypersdk -n hypersdk -o jsonpath='{.spec.host}')
echo "HyperSDK URL: https://${ROUTE_URL}"

# Test health endpoint
curl -k https://${ROUTE_URL}/health
```

### 3. Access Application

**Get the Route URL**:

```bash
# Get route URL
ROUTE_URL=$(oc get route hypersdk -n hypersdk -o jsonpath='{.spec.host}')
echo "HyperSDK Web Server: https://${ROUTE_URL}"
```

**Access via Browser** (Recommended):

```bash
# Open the web UI in your browser
firefox https://${ROUTE_URL}
# or
google-chrome https://${ROUTE_URL}
```

**Or use Port Forwarding** (Development):

```bash
# Forward the web server to localhost
oc port-forward svc/hypersdk 8080:8080 -n hypersdk

# Access at
http://localhost:8080
```

For more access methods, see [Accessing the Web Server](#accessing-the-web-server).

## OpenShift-Specific Features

### Routes (OpenShift Ingress)

OpenShift uses **Routes** instead of Kubernetes Ingress for external access.

**Enable Route**:

```yaml
# values.yaml
route:
  enabled: true
  host: hypersdk.apps.cluster.example.com
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect

# Disable standard Ingress
ingress:
  enabled: false
```

**Route Types**:

#### Edge Termination (Recommended)
TLS terminates at the router, HTTP to backend:

```yaml
route:
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
```

#### Passthrough Termination
TLS passthrough to backend (requires TLS in HyperSDK):

```yaml
route:
  tls:
    termination: passthrough
```

#### Re-encrypt Termination
TLS at router and backend:

```yaml
route:
  tls:
    termination: reencrypt
    destinationCACertificate: |
      -----BEGIN CERTIFICATE-----
      ...
      -----END CERTIFICATE-----
```

**Custom Route Configuration**:

```bash
helm install hypersdk hypersdk/hypersdk \
  --set route.enabled=true \
  --set route.host=hypersdk.apps.example.com \
  --set route.tls.termination=edge \
  --set ingress.enabled=false \
  -n hypersdk
```

### SecurityContextConstraints (SCC)

OpenShift uses SCCs for pod-level security policies.

**Enable Custom SCC**:

```yaml
openshift:
  enabled: true
  scc:
    enabled: true
```

**What the SCC Does**:
- Allows running as UID 1000 (or OpenShift-assigned random UID)
- Permits required volume types (PVC, ConfigMap, Secret)
- Drops all capabilities
- Prevents privilege escalation
- Non-root enforcement

**Verify SCC**:

```bash
# Check SCC is created
oc get scc | grep hypersdk

# View SCC details
oc describe scc hypersdk-scc

# Check pod's SCC assignment
oc get pod -n hypersdk -o yaml | grep openshift.io/scc
```

**Using Default SCCs**:

If not creating custom SCC, use `restricted-v2` (default):

```yaml
openshift:
  scc:
    enabled: false

# Pod will use restricted-v2 SCC automatically
```

### Random UID Assignment

OpenShift assigns random UIDs in allowed range for security.

**Configure for OpenShift UID Assignment**:

```yaml
podSecurityContext:
  runAsNonRoot: true
  # Don't set runAsUser - let OpenShift assign it
  fsGroup: 1000

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
```

**Handle Init Containers**:

The default `init-permissions` container needs modification for OpenShift:

```yaml
# Disable default init container
initContainers: []

# Or modify to work with arbitrary UIDs
podSecurityContext:
  fsGroup: 1000  # This is sufficient for volume permissions
```

### OpenShift Monitoring Integration

Integrate with OpenShift cluster monitoring (built-in Prometheus).

**Enable Cluster Monitoring**:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    namespace: openshift-monitoring
    labels:
      openshift.io/cluster-monitoring: "true"

  prometheusRule:
    enabled: true
    namespace: openshift-monitoring
    labels:
      openshift.io/cluster-monitoring: "true"
```

**Install with Monitoring**:

```bash
helm install hypersdk hypersdk/hypersdk \
  --set monitoring.serviceMonitor.enabled=true \
  --set monitoring.serviceMonitor.namespace=openshift-monitoring \
  --set monitoring.serviceMonitor.labels.openshift\\.io/cluster-monitoring=true \
  -n hypersdk
```

**View Metrics in OpenShift Console**:

1. Navigate to **Observe** → **Metrics**
2. Query: `hypersdk_http_requests_total`
3. View dashboards in **Observe** → **Dashboards**

**Access Prometheus UI**:

```bash
# Port forward to Prometheus
oc port-forward -n openshift-monitoring \
  svc/prometheus-k8s 9090:9090

# Open browser
http://localhost:9090
```

## Accessing the Web Server

HyperSDK (hypervisord) exposes a web server on port 8080 and metrics on port 8081. OpenShift provides multiple ways to access these services.

### Method 1: OpenShift Route (Recommended for Production)

The OpenShift Route provides external HTTPS access with automatic TLS termination.

**How It Works**:
```
User Browser → Route (TLS) → Service → Pod (hypervisord:8080)
  HTTPS           Edge        ClusterIP    Container
```

**Access via Browser**:

```bash
# Get the Route URL
ROUTE_URL=$(oc get route hypersdk -n hypersdk -o jsonpath='{.spec.host}')
echo "Web Server: https://${ROUTE_URL}"

# Open in browser
firefox https://${ROUTE_URL}
# or
google-chrome https://${ROUTE_URL}
```

**Access via Command Line**:

```bash
# Test the web server
curl -k https://${ROUTE_URL}

# Test specific endpoints (if available)
curl -k https://${ROUTE_URL}/health
curl -k https://${ROUTE_URL}/api/v1/status
```

**Route Configuration**:

The Route is automatically created with:
- **TLS Termination**: Edge (router handles SSL)
- **Redirect**: HTTP → HTTPS automatic
- **Port**: 8080 (hypervisord web server)

**Custom Domain**:

```bash
# Install with custom domain
helm install hypersdk hypersdk/hypersdk \
  -f examples/openshift-values.yaml \
  --set route.host=hypersdk.production.com \
  -n hypersdk
```

### Method 2: Port Forwarding (Recommended for Development)

Port forwarding provides direct access to the service from your local machine.

**Forward Web Server**:

```bash
# Forward port 8080 to localhost
oc port-forward svc/hypersdk 8080:8080 -n hypersdk

# Access at
curl http://localhost:8080
firefox http://localhost:8080
```

**Forward Both Web and Metrics**:

```bash
# Forward both ports simultaneously
oc port-forward svc/hypersdk 8080:8080 8081:8081 -n hypersdk

# Access web server
curl http://localhost:8080

# Access metrics
curl http://localhost:8081/metrics
```

**Forward to Different Local Port**:

```bash
# If port 8080 is already in use
oc port-forward svc/hypersdk 9090:8080 -n hypersdk

# Access at
curl http://localhost:9090
```

**Forward Directly to Pod** (for debugging):

```bash
# Get pod name
POD=$(oc get pods -n hypersdk -l app.kubernetes.io/name=hypersdk -o jsonpath='{.items[0].metadata.name}')

# Forward from specific pod
oc port-forward pod/$POD 8080:8080 -n hypersdk
```

### Method 3: Internal Service DNS

For pod-to-pod communication within the cluster.

**From Same Namespace**:

```bash
# Service DNS (short form)
curl http://hypersdk:8080

# Access metrics
curl http://hypersdk:8081/metrics
```

**From Different Namespace**:

```bash
# Fully qualified service DNS
curl http://hypersdk.hypersdk.svc.cluster.local:8080

# Format: <service>.<namespace>.svc.cluster.local:<port>
```

**Test with Debug Pod**:

```bash
# Start interactive debug pod
oc run curl-test --image=curlimages/curl --rm -it -n hypersdk -- sh

# From inside the pod
curl http://hypersdk:8080
curl http://hypersdk:8081/metrics
exit
```

### Method 4: OpenShift Web Console

Access the web server through the OpenShift console GUI.

**Via Routes**:

1. Login to OpenShift Console
2. Navigate to: **Networking → Routes**
3. Select project/namespace: **hypersdk**
4. Click on: **hypersdk** route
5. Click the URL under "Location" column
6. Opens in browser: `https://hypersdk.apps.cluster.example.com`

**Via Topology View** (Developer Perspective):

1. Switch to **Developer** perspective (dropdown top-left)
2. Select project: **hypersdk**
3. Click: **Topology**
4. Click the **hypersdk** deployment circle
5. Right sidebar shows route link
6. Click to open web interface

**Port Forward from Console**:

1. Navigate: **Workloads → Pods**
2. Select: hypersdk pod
3. Click: **Actions** → **Port forward**
4. Select port: **8080**
5. Click: **Start**
6. Access at: `http://localhost:8080`

### Access Method Comparison

| Method | URL Example | Use Case | Pros | Cons |
|--------|-------------|----------|------|------|
| **Route** | `https://hypersdk.apps.example.com` | Production | External, TLS, Production-ready | Requires DNS |
| **Port-forward** | `http://localhost:8080` | Development | Quick, no config | Local only, temporary |
| **Service DNS** | `http://hypersdk:8080` | Microservices | Fast, internal | Cluster-only |
| **Console** | Click in UI | Management | Visual, easy | Manual process |

### Accessing Metrics

The metrics endpoint (port 8081) can be accessed using the same methods:

**Via Route** (if additional route created):

```bash
# Create separate route for metrics
oc create route edge hypersdk-metrics \
  --service=hypersdk \
  --port=8081 \
  --hostname=hypersdk-metrics.apps.cluster.example.com \
  -n hypersdk

# Access
curl -k https://hypersdk-metrics.apps.cluster.example.com/metrics
```

**Via Port Forward**:

```bash
# Forward metrics port
oc port-forward svc/hypersdk 8081:8081 -n hypersdk

# Access metrics
curl http://localhost:8081/metrics
```

**Via Service DNS**:

```bash
# From another pod
curl http://hypersdk:8081/metrics
```

### Troubleshooting Access

**Route Not Accessible**:

```bash
# 1. Check Route exists
oc get route hypersdk -n hypersdk

# 2. Check Service has endpoints
oc get endpoints hypersdk -n hypersdk
# Should show pod IPs, not <none>

# 3. Check Pod is running
oc get pods -n hypersdk -l app.kubernetes.io/name=hypersdk
# Status should be "Running"

# 4. Test connectivity
ROUTE_URL=$(oc get route hypersdk -n hypersdk -o jsonpath='{.spec.host}')
curl -v -k https://$ROUTE_URL
```

**Port Forward Not Working**:

```bash
# 1. Verify service exists
oc get svc hypersdk -n hypersdk

# 2. Check pod status
oc get pods -n hypersdk -l app.kubernetes.io/name=hypersdk

# 3. Try direct pod access
POD=$(oc get pod -n hypersdk -l app.kubernetes.io/name=hypersdk -o name | head -1)
oc port-forward $POD 8080:8080 -n hypersdk
```

**Connection Refused**:

```bash
# Check if hypervisord is listening on correct port
oc exec -it deployment/hypersdk -n hypersdk -- netstat -tlnp | grep 8080

# Check pod logs
oc logs -l app.kubernetes.io/name=hypersdk -n hypersdk

# Check service endpoints
oc describe svc hypersdk -n hypersdk
```

### Production Examples

**Multiple Routes for Different Services**:

```bash
# Main web UI
oc get route hypersdk -n hypersdk

# Additional route for metrics
oc create route edge hypersdk-metrics \
  --service=hypersdk \
  --port=8081 \
  --hostname=metrics-hypersdk.production.com \
  -n hypersdk

# Access
https://hypersdk.production.com           # Web UI
https://metrics-hypersdk.production.com   # Metrics
```

**Passthrough TLS** (if hypervisord handles TLS):

```yaml
# values.yaml
route:
  enabled: true
  host: hypersdk.production.com
  tls:
    termination: passthrough
```

## Deployment Methods

### Method 1: Helm CLI

```bash
# Using OpenShift-specific values
helm install hypersdk hypersdk/hypersdk \
  -f deployments/helm/hypersdk/examples/openshift-values.yaml \
  -n hypersdk --create-namespace

# Or inline values
helm install hypersdk hypersdk/hypersdk \
  --set route.enabled=true \
  --set route.host=hypersdk.apps.$(oc get ingresses.config.openshift.io cluster -o jsonpath='{.spec.domain}') \
  --set openshift.enabled=true \
  --set openshift.scc.enabled=true \
  -n hypersdk --create-namespace
```

### Method 2: OpenShift GitOps (Argo CD)

OpenShift GitOps operator provides Argo CD for GitOps deployments.

**Install OpenShift GitOps Operator**:

```bash
# Install from OperatorHub
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: openshift-gitops-operator
  namespace: openshift-operators
spec:
  channel: latest
  name: openshift-gitops-operator
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF

# Wait for operator
oc wait --for=condition=ready pod \
  -l name=openshift-gitops-operator \
  -n openshift-operators --timeout=300s
```

**Create Argo CD Application**:

```yaml
# hypersdk-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hypersdk
  namespace: openshift-gitops
spec:
  project: default
  source:
    repoURL: https://ssahani.github.io/hypersdk/helm-charts
    chart: hypersdk
    targetRevision: 0.2.0
    helm:
      valueFiles:
        - https://raw.githubusercontent.com/ssahani/hypersdk/main/deployments/helm/hypersdk/examples/openshift-values.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: hypersdk
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

Apply:

```bash
oc apply -f hypersdk-app.yaml

# Check sync status
oc get application hypersdk -n openshift-gitops
```

### Method 3: OpenShift Pipelines (Tekton)

Deploy using OpenShift Pipelines for CI/CD.

**Install OpenShift Pipelines Operator**:

```bash
oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: openshift-pipelines-operator
  namespace: openshift-operators
spec:
  channel: latest
  name: openshift-pipelines-operator-rh
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF
```

**Create Pipeline**:

```yaml
apiVersion: tekton.dev/v1beta1
kind: Pipeline
metadata:
  name: deploy-hypersdk
  namespace: hypersdk
spec:
  params:
    - name: chart-version
      type: string
      default: "0.2.0"
  tasks:
    - name: helm-install
      taskRef:
        name: helm-upgrade-from-repo
        kind: ClusterTask
      params:
        - name: helm_repo
          value: https://ssahani.github.io/hypersdk/helm-charts
        - name: chart_name
          value: hypersdk
        - name: release_version
          value: $(params.chart-version)
        - name: release_name
          value: hypersdk
        - name: release_namespace
          value: hypersdk
        - name: overwrite_values
          value: route.enabled=true,openshift.enabled=true
```

### Method 4: OpenShift Template

Convert Helm chart to OpenShift Template.

**Generate Template**:

```bash
# Template Helm chart
helm template hypersdk hypersdk/hypersdk \
  -f deployments/helm/hypersdk/examples/openshift-values.yaml \
  > hypersdk-template.yaml

# Add Template wrapper
oc create -f hypersdk-template.yaml -n hypersdk
```

## Security Configuration

### Pod Security

OpenShift enforces strict pod security by default.

**Security Context (OpenShift Compatible)**:

```yaml
podSecurityContext:
  runAsNonRoot: true
  # Don't set runAsUser - OpenShift assigns random UID
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: false  # SQLite needs write access
  runAsNonRoot: true
```

### Service Accounts

**Create Service Account with Specific Permissions**:

```bash
# Service account is created by chart
oc get sa hypersdk -n hypersdk

# Add custom role
oc create role hypersdk-role \
  --verb=get,list,watch \
  --resource=pods,services \
  -n hypersdk

oc create rolebinding hypersdk-binding \
  --role=hypersdk-role \
  --serviceaccount=hypersdk:hypersdk \
  -n hypersdk
```

### Secrets Management

**Create Secrets for Cloud Providers**:

```bash
# vSphere credentials
oc create secret generic vsphere-credentials \
  --from-literal=url=https://vcenter.example.com/sdk \
  --from-literal=username=admin@vsphere.local \
  --from-literal=password='changeme' \
  --from-literal=insecure=1 \
  -n hypersdk

# AWS credentials
oc create secret generic aws-credentials \
  --from-literal=accessKeyId=AKIAIOSFODNN7EXAMPLE \
  --from-literal=secretAccessKey=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --from-literal=region=us-east-1 \
  -n hypersdk

# Azure credentials
oc create secret generic azure-credentials \
  --from-literal=subscriptionId=00000000-0000-0000-0000-000000000000 \
  --from-literal=tenantId=00000000-0000-0000-0000-000000000000 \
  --from-literal=clientId=00000000-0000-0000-0000-000000000000 \
  --from-literal=clientSecret=changeme \
  -n hypersdk
```

**Use Secrets in Deployment**:

```yaml
credentials:
  vsphere:
    enabled: true
    existingSecret: vsphere-credentials
  aws:
    enabled: true
    existingSecret: aws-credentials
  azure:
    enabled: true
    existingSecret: azure-credentials
```

### Network Policies

OpenShift SDN network policies.

**Enable Network Policy**:

```yaml
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Allow from OpenShift router
    - from:
        - namespaceSelector:
            matchLabels:
              network.openshift.io/policy-group: ingress
      ports:
        - protocol: TCP
          port: 8080
    # Allow from Prometheus
    - from:
        - namespaceSelector:
            matchLabels:
              network.openshift.io/policy-group: monitoring
      ports:
        - protocol: TCP
          port: 8081
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: openshift-dns
      ports:
        - protocol: UDP
          port: 53
    # Allow HTTPS
    - to:
        - podSelector: {}
      ports:
        - protocol: TCP
          port: 443
```

## Networking

### Routes

**Get Route Details**:

```bash
# List routes
oc get route -n hypersdk

# Describe route
oc describe route hypersdk -n hypersdk

# Get route URL
oc get route hypersdk -n hypersdk \
  -o jsonpath='{.spec.host}'
```

**Custom Domain**:

```bash
helm upgrade hypersdk hypersdk/hypersdk \
  --set route.host=api.example.com \
  -n hypersdk --reuse-values
```

**Multiple Routes**:

Create additional routes manually:

```yaml
apiVersion: route.openshift.io/v1
kind: Route
metadata:
  name: hypersdk-metrics
  namespace: hypersdk
spec:
  host: metrics-hypersdk.apps.cluster.example.com
  to:
    kind: Service
    name: hypersdk
  port:
    targetPort: metrics
  tls:
    termination: edge
    insecureEdgeTerminationPolicy: Redirect
```

### Service Mesh (Istio)

OpenShift Service Mesh integration (optional).

**Install Red Hat OpenShift Service Mesh**:

```bash
# Install operators (Elasticsearch, Jaeger, Kiali, Service Mesh)
# Then create ServiceMeshControlPlane

oc apply -f - <<EOF
apiVersion: maistra.io/v2
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
    - hypersdk
EOF
```

**Enable Sidecar Injection**:

```yaml
podAnnotations:
  sidecar.istio.io/inject: "true"
```

## Storage

### Persistent Volumes

OpenShift provides several storage options.

**Check Available StorageClasses**:

```bash
oc get storageclass

# Example output:
# NAME                          PROVISIONER
# gp2 (default)                 kubernetes.io/aws-ebs
# ocs-storagecluster-ceph-rbd   openshift-storage.rbd.csi.ceph.com
# ocs-storagecluster-cephfs     openshift-storage.cephfs.csi.ceph.com
```

**Use Specific StorageClass**:

```yaml
persistence:
  data:
    enabled: true
    size: 10Gi
    storageClass: ocs-storagecluster-ceph-rbd  # Fast SSD
  exports:
    enabled: true
    size: 500Gi
    storageClass: ocs-storagecluster-cephfs  # Shared filesystem
    accessMode: ReadWriteMany  # Allow multiple pods
```

**OpenShift Container Storage (OCS)**:

For production, use OpenShift Data Foundation (ODF/OCS):

```bash
# Install ODF operator from OperatorHub
# Then create StorageCluster

# Use ODF storage
helm install hypersdk hypersdk/hypersdk \
  --set persistence.data.storageClass=ocs-storagecluster-ceph-rbd \
  --set persistence.exports.storageClass=ocs-storagecluster-cephfs \
  --set persistence.exports.accessMode=ReadWriteMany \
  -n hypersdk
```

### Volume Snapshots

OpenShift supports CSI volume snapshots.

**Create VolumeSnapshotClass**:

```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshotClass
metadata:
  name: ocs-snapshot
driver: openshift-storage.rbd.csi.ceph.com
deletionPolicy: Delete
```

**Create Snapshot**:

```bash
oc apply -f - <<EOF
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: hypersdk-data-snapshot
  namespace: hypersdk
spec:
  volumeSnapshotClassName: ocs-snapshot
  source:
    persistentVolumeClaimName: hypersdk-data
EOF
```

## Monitoring

### OpenShift Monitoring Stack

Use built-in cluster monitoring.

**Enable User Workload Monitoring**:

```bash
oc apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
  namespace: openshift-monitoring
data:
  config.yaml: |
    enableUserWorkload: true
EOF
```

**Deploy with Monitoring**:

```bash
helm install hypersdk hypersdk/hypersdk \
  --set monitoring.serviceMonitor.enabled=true \
  --set monitoring.serviceMonitor.namespace=openshift-monitoring \
  --set monitoring.serviceMonitor.labels.openshift\\.io/cluster-monitoring=true \
  -n hypersdk
```

**Query Metrics**:

```bash
# Port forward to Thanos Querier
oc port-forward -n openshift-monitoring \
  svc/thanos-querier 9090:9091

# Query: http://localhost:9090
```

### Custom Dashboards

Import Grafana dashboards into OpenShift console.

**Create ConfigMap with Dashboard**:

```bash
oc create configmap hypersdk-dashboard \
  --from-file=deployments/helm/dashboards/hypersdk-overview.json \
  -n openshift-config-managed

oc label configmap hypersdk-dashboard \
  console.openshift.io/dashboard=true \
  -n openshift-config-managed
```

**View in Console**:

Navigate to **Observe** → **Dashboards** → **HyperSDK Overview**

### Alerts

**Create PrometheusRule**:

```yaml
monitoring:
  prometheusRule:
    enabled: true
    namespace: openshift-monitoring
    labels:
      openshift.io/cluster-monitoring: "true"
    rules:
      - alert: HyperSDKDown
        expr: up{job="hypersdk"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "HyperSDK is down"
          description: "HyperSDK has been down for more than 5 minutes"
```

## Troubleshooting

### Common Issues

#### Issue 1: Pod in CrashLoopBackOff

```bash
# Check pod logs
oc logs -n hypersdk -l app.kubernetes.io/name=hypersdk

# Check events
oc get events -n hypersdk --sort-by='.lastTimestamp'

# Check pod description
oc describe pod -n hypersdk -l app.kubernetes.io/name=hypersdk
```

**Common Causes**:
- Permission issues on volumes (check SCC)
- Database connection failures
- Invalid configuration

**Solution**:
```bash
# Check SCC assignment
oc get pod -n hypersdk -o yaml | grep openshift.io/scc

# Verify volumes
oc get pvc -n hypersdk

# Check configuration
oc get configmap hypersdk-config -n hypersdk -o yaml
```

#### Issue 2: Route Not Accessible

```bash
# Check route
oc get route hypersdk -n hypersdk

# Test from within cluster
oc run curl --image=curlimages/curl -it --rm -- \
  curl -k https://hypersdk.apps.cluster.example.com/health
```

**Common Causes**:
- Route not created
- TLS certificate issues
- Service not ready

**Solution**:
```bash
# Verify service endpoints
oc get endpoints hypersdk -n hypersdk

# Check router logs
oc logs -n openshift-ingress -l app=router
```

#### Issue 3: Permission Denied Errors

```bash
# Check SCC
oc get scc | grep hypersdk
oc describe scc hypersdk-scc

# Check pod security context
oc get pod -n hypersdk -o yaml | grep -A 10 securityContext
```

**Solution**:
```bash
# Grant SCC to service account
oc adm policy add-scc-to-user hypersdk-scc \
  system:serviceaccount:hypersdk:hypersdk
```

#### Issue 4: Storage Issues

```bash
# Check PVC status
oc get pvc -n hypersdk

# Describe PVC
oc describe pvc hypersdk-data -n hypersdk

# Check storage class
oc get storageclass
```

**Solution**:
```bash
# Delete and recreate PVC
oc delete pvc hypersdk-data -n hypersdk
helm upgrade hypersdk hypersdk/hypersdk -n hypersdk --reuse-values
```

### Diagnostic Commands

```bash
# Comprehensive diagnostics
oc get all -n hypersdk
oc get pvc,configmap,secret -n hypersdk
oc describe pod -n hypersdk
oc logs -n hypersdk -l app.kubernetes.io/name=hypersdk --tail=100
oc get events -n hypersdk --sort-by='.lastTimestamp'

# Check SCC
oc describe scc hypersdk-scc
oc get pod -n hypersdk -o yaml | grep scc

# Check route
oc get route hypersdk -n hypersdk -o yaml
oc describe route hypersdk -n hypersdk

# Check monitoring
oc get servicemonitor -n hypersdk
oc get prometheusrule -n hypersdk
```

### Debug Pod

Run debug pod for troubleshooting:

```bash
oc debug deployment/hypersdk -n hypersdk

# Or start interactive shell
oc run debug --image=registry.access.redhat.com/ubi8/ubi:latest \
  -it --rm -n hypersdk -- /bin/bash
```

## Best Practices

### 1. Use Projects (Namespaces)

Create dedicated project for each environment:

```bash
oc new-project hypersdk-dev
oc new-project hypersdk-staging
oc new-project hypersdk-prod
```

### 2. Resource Quotas

Set resource limits per project:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: hypersdk-quota
  namespace: hypersdk
spec:
  hard:
    requests.cpu: "4"
    requests.memory: 8Gi
    persistentvolumeclaims: "10"
```

### 3. Limit Ranges

Define default resource limits:

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: hypersdk-limits
  namespace: hypersdk
spec:
  limits:
    - type: Container
      default:
        cpu: 500m
        memory: 1Gi
      defaultRequest:
        cpu: 100m
        memory: 256Mi
```

### 4. Use OpenShift GitOps

Manage deployments declaratively:

```bash
# Install OpenShift GitOps operator
# Create Application CRD pointing to Git repo
# Changes auto-sync from Git
```

### 5. Enable Cluster Monitoring

Integrate with OpenShift monitoring:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    namespace: openshift-monitoring
    labels:
      openshift.io/cluster-monitoring: "true"
```

### 6. Backup Strategy

Regular backups using OADP (OpenShift API for Data Protection):

```bash
# Install OADP operator
# Create backup schedule
# Test restore procedures
```

### 7. Security Scanning

Use OpenShift security scanning:

```bash
# Enable container scanning
oc get imagestreamtag -n openshift

# View vulnerabilities
oc describe imagestreamtag <image>
```

### 8. Multi-tenancy

Isolate tenants using NetworkPolicies:

```yaml
networkPolicy:
  enabled: true
  # Define strict ingress/egress rules
```

## Production Checklist

Before going to production on OpenShift:

- [ ] **Security**:
  - [ ] Custom SCC configured
  - [ ] Secrets stored securely (not in values.yaml)
  - [ ] Network policies enabled
  - [ ] Pod security standards enforced

- [ ] **High Availability**:
  - [ ] Multiple replicas (replicaCount >= 2)
  - [ ] PodDisruptionBudget enabled
  - [ ] Anti-affinity rules configured
  - [ ] Liveness and readiness probes tuned

- [ ] **Storage**:
  - [ ] Production-grade storage class (ODF/OCS)
  - [ ] Volume snapshots configured
  - [ ] Backup strategy implemented

- [ ] **Networking**:
  - [ ] Route configured with custom domain
  - [ ] TLS enabled (edge termination minimum)
  - [ ] Network policies tested

- [ ] **Monitoring**:
  - [ ] ServiceMonitor enabled
  - [ ] PrometheusRules configured
  - [ ] Alerts tested
  - [ ] Dashboards imported

- [ ] **Performance**:
  - [ ] Resource requests/limits tuned
  - [ ] HPA enabled and tested
  - [ ] Load testing completed

- [ ] **Disaster Recovery**:
  - [ ] Backup schedule configured
  - [ ] Restore procedures tested
  - [ ] DR drills scheduled

## Related Documentation

- [DEPLOYMENT.md](DEPLOYMENT.md) - General deployment guide
- [SECURITY.md](SECURITY.md) - Security hardening
- [OBSERVABILITY.md](OBSERVABILITY.md) - Monitoring setup
- [DISASTER-RECOVERY.md](DISASTER-RECOVERY.md) - DR procedures

## References

- [OpenShift Documentation](https://docs.openshift.com/)
- [OpenShift Routes](https://docs.openshift.com/container-platform/latest/networking/routes/route-configuration.html)
- [Security Context Constraints](https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html)
- [OpenShift Monitoring](https://docs.openshift.com/container-platform/latest/monitoring/monitoring-overview.html)
- [OpenShift GitOps](https://docs.openshift.com/gitops/latest/)

## Support

- **Issues**: https://github.com/ssahani/hypersdk/issues
- **OpenShift Forum**: https://access.redhat.com/discussions
- **Red Hat Support**: For subscribed customers

---

**🚀 Deploy HyperSDK on OpenShift with confidence!**
