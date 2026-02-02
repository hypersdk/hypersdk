# Kubernetes Integration - Implementation Plan

**Feature**: Cloud-native VM management and backup for Kubernetes
**Target**: Kubernetes v1.27+, KubeVirt integration, Operator pattern
**Estimated Effort**: 2-3 weeks
**Status**: Planning Phase

---

## 🎯 Overview

Add comprehensive Kubernetes integration to HyperSDK, enabling:
- **KubeVirt VM management** - Backup and restore VMs running on Kubernetes
- **Kubernetes Operator** - Cloud-native deployment and management
- **Custom Resources** - CRDs for BackupJob, BackupSchedule, RestoreJob
- **Helm Charts** - Easy deployment to any Kubernetes cluster
- **Dashboard Integration** - Web UI for Kubernetes VM monitoring

---

## 📦 Integration Scope

### Option 1: KubeVirt Provider (VMs on Kubernetes)

**What**: Treat Kubernetes with KubeVirt as another VM provider, like vSphere or AWS.

**Features**:
- Discover VirtualMachine resources in Kubernetes clusters
- Export/backup KubeVirt VMs to object storage (S3, GCS, Azure Blob)
- Start/stop VirtualMachines via Kubernetes API
- Snapshot VirtualMachines using VolumeSnapshots
- Restore VMs from backups
- Live migration support

**Use Cases**:
- Backup KubeVirt VMs to cloud storage
- Migrate VMs between Kubernetes clusters
- Disaster recovery for Kubernetes-hosted VMs
- Clone VMs for testing/development

---

### Option 2: Kubernetes Operator (HyperSDK as K8s Native)

**What**: Deploy HyperSDK as a Kubernetes Operator with CRDs for backup management.

**Features**:
- Kubernetes Operator managing backup lifecycle
- CRDs: BackupJob, BackupSchedule, RestoreJob, BackupPolicy
- Automated discovery of VMs across all providers
- Scheduled backups using CronJobs
- Helm chart for installation
- Prometheus metrics and alerts
- Webhook admission controllers

**Use Cases**:
- GitOps-based backup management
- Multi-cluster backup orchestration
- Policy-driven backup automation
- Integration with ArgoCD/Flux

---

### Option 3: Hybrid Approach (Recommended)

**What**: Combine both approaches for maximum flexibility.

**Components**:
1. **KubeVirt Provider** - Manage VMs running on Kubernetes
2. **Kubernetes Operator** - Deploy and manage HyperSDK in Kubernetes
3. **CRDs** - Declarative backup management
4. **Helm Charts** - Easy installation
5. **Dashboard** - Enhanced web UI for Kubernetes

---

## 🏗️ Architecture Design

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                          │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              HyperSDK Operator                           │  │
│  │  ┌────────────────┐  ┌────────────────┐                │  │
│  │  │   Controller   │  │  Webhook       │                │  │
│  │  │   Manager      │  │  Server        │                │  │
│  │  └────────────────┘  └────────────────┘                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Custom Resources (CRDs)                     │  │
│  │  - BackupJob          - BackupSchedule                  │  │
│  │  - RestoreJob         - BackupPolicy                    │  │
│  │  - VMBackup           - BackupRepository                │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              HyperSDK Daemon (StatefulSet)               │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐              │  │
│  │  │ Pod 0    │  │ Pod 1    │  │ Pod 2    │              │  │
│  │  │ Worker   │  │ Worker   │  │ Worker   │              │  │
│  │  └──────────┘  └──────────┘  └──────────┘              │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Dashboard (Deployment)                      │  │
│  │  - React UI with Kubernetes integration                 │  │
│  │  - Real-time VM monitoring                              │  │
│  │  - Job management                                       │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
           │                      │                      │
           │                      │                      │
           ▼                      ▼                      ▼
    ┌───────────┐          ┌───────────┐        ┌───────────┐
    │ KubeVirt  │          │ vSphere   │        │   AWS     │
    │   VMs     │          │   VMs     │        │   VMs     │
    └───────────┘          └───────────┘        └───────────┘
```

---

## 📋 Phase 1: KubeVirt Provider (~1 week)

### 1.1 Provider Implementation

**File**: `providers/kubevirt/kubevirt.go`

```go
type KubeVirtProvider struct {
    clientset    kubernetes.Interface
    virtClient   kubevirt.KubevirtClient
    namespace    string
    storageClass string
}

// Core Provider interface
func (p *KubeVirtProvider) ListVMs() ([]VM, error)
func (p *KubeVirtProvider) GetVM(name string) (*VM, error)
func (p *KubeVirtProvider) StartVM(name string) error
func (p *KubeVirtProvider) StopVM(name string) error
func (p *KubeVirtProvider) DeleteVM(name string) error
func (p *KubeVirtProvider) ExportVM(name string, dest string) error

// KubeVirt-specific features
func (p *KubeVirtProvider) CreateSnapshot(vmName string) (*Snapshot, error)
func (p *KubeVirtProvider) RestoreSnapshot(snapshotName string) error
func (p *KubeVirtProvider) MigrateVM(vmName string, targetNode string) error
func (p *KubeVirtProvider) CloneVM(vmName string, newName string) error
```

**Features**:
- Kubernetes client-go integration
- KubeVirt API (VirtualMachine, VirtualMachineInstance)
- Volume snapshot support (CSI snapshots)
- PVC backup to object storage
- Network configuration preservation
- ConfigMap/Secret handling

**Dependencies**:
```go
import (
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    kubevirtv1 "kubevirt.io/api/core/v1"
    "kubevirt.io/client-go/kubecli"
)
```

**Estimated Lines**: ~800 lines

---

### 1.2 VM Discovery and Listing

**Endpoint**: `GET /vms/kubevirt`

**Request**:
```json
{
  "kubeconfig": "/path/to/kubeconfig",
  "namespace": "default",
  "labels": {
    "app": "production"
  }
}
```

**Response**:
```json
{
  "vms": [
    {
      "name": "ubuntu-vm-1",
      "namespace": "default",
      "status": "Running",
      "node": "worker-1",
      "cpu": 4,
      "memory": "8Gi",
      "disks": [
        {
          "name": "disk0",
          "size": "50Gi",
          "storageClass": "fast-ssd"
        }
      ],
      "networks": [
        {
          "name": "default",
          "type": "pod"
        }
      ],
      "created": "2026-02-04T10:00:00Z"
    }
  ]
}
```

**Estimated Lines**: ~200 lines

---

### 1.3 VM Operations

**Start VM**: `POST /vms/kubevirt/start`
```bash
curl -X POST http://localhost:8080/vms/kubevirt/start \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ubuntu-vm-1",
    "namespace": "default"
  }'
```

**Stop VM**: `POST /vms/kubevirt/stop`
**Delete VM**: `DELETE /vms/kubevirt/{namespace}/{name}`
**Migrate VM**: `POST /vms/kubevirt/migrate`

**Estimated Lines**: ~300 lines

---

### 1.4 VM Export/Backup

**Export VM to S3**:
```bash
hyperctl export kubevirt \
  --vm ubuntu-vm-1 \
  --namespace default \
  --output s3://my-bucket/backups/ubuntu-vm-1 \
  --format qcow2
```

**Process**:
1. Create VirtualMachine snapshot (VolumeSnapshot)
2. Create temporary PVC from snapshot
3. Start export pod with PVC mounted
4. Stream disk data to object storage
5. Export VM manifest (YAML) with metadata
6. Clean up temporary resources

**Estimated Lines**: ~400 lines

---

## 📋 Phase 2: Kubernetes Operator (~1 week)

### 2.1 Custom Resource Definitions (CRDs)

#### BackupJob CRD

**File**: `deploy/crds/hypersdk.io_backupjobs.yaml`

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: BackupJob
metadata:
  name: daily-vm-backup
  namespace: default
spec:
  # Source VM configuration
  source:
    provider: kubevirt
    namespace: default
    vmName: ubuntu-vm-1
    # OR for other providers
    # provider: vsphere
    # vmPath: /datacenter/vm/prod/ubuntu-vm-1

  # Destination configuration
  destination:
    type: s3
    bucket: my-backups
    prefix: kubevirt/daily
    region: us-west-2
    storageClass: STANDARD_IA

  # Backup options
  options:
    format: qcow2
    compression: true
    encryption:
      enabled: true
      kmsKeyId: arn:aws:kms:...

  # Retention policy
  retention:
    keepLast: 7
    keepDaily: 30
    keepWeekly: 12
    keepMonthly: 12

  # Carbon-aware scheduling
  carbonAware:
    enabled: true
    zone: US-CAL-CISO
    maxIntensity: 200
    maxDelayHours: 4

status:
  phase: Completed
  startTime: "2026-02-04T10:00:00Z"
  completionTime: "2026-02-04T12:00:00Z"
  backupSize: 45Gi
  carbonEmissions: 0.033 kg CO2
  backupLocation: s3://my-backups/kubevirt/daily/ubuntu-vm-1-20260204.qcow2
```

**Estimated Lines**: ~150 lines YAML + ~400 lines Go controller

---

#### BackupSchedule CRD

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: BackupSchedule
metadata:
  name: nightly-backup
  namespace: default
spec:
  schedule: "0 2 * * *"  # 2 AM daily

  # Job template
  jobTemplate:
    spec:
      source:
        provider: kubevirt
        namespace: default
        vmSelector:
          matchLabels:
            backup: "true"
            tier: "production"
      destination:
        type: s3
        bucket: prod-backups
      carbonAware:
        enabled: true

  # Concurrency policy
  concurrencyPolicy: Forbid  # or Allow, Replace

  # Success/failure history
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1

  # Suspend scheduling
  suspend: false

status:
  lastScheduleTime: "2026-02-04T02:00:00Z"
  lastSuccessfulTime: "2026-02-04T04:00:00Z"
  active:
    - name: nightly-backup-20260204
      namespace: default
```

**Estimated Lines**: ~120 lines YAML + ~350 lines Go controller

---

#### RestoreJob CRD

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: RestoreJob
metadata:
  name: restore-ubuntu-vm
  namespace: default
spec:
  # Source backup
  source:
    type: s3
    bucket: my-backups
    path: kubevirt/daily/ubuntu-vm-1-20260204.qcow2

  # Destination
  destination:
    provider: kubevirt
    namespace: default
    vmName: ubuntu-vm-1-restored
    storageClass: fast-ssd

  # Restore options
  options:
    startAfterRestore: false
    overwriteExisting: false

status:
  phase: Completed
  startTime: "2026-02-04T15:00:00Z"
  completionTime: "2026-02-04T15:30:00Z"
  restoredVM: default/ubuntu-vm-1-restored
```

**Estimated Lines**: ~100 lines YAML + ~300 lines Go controller

---

### 2.2 Operator Implementation

**Framework**: Operator SDK (Kubebuilder)

**File Structure**:
```
operators/hypersdk-operator/
├── cmd/
│   └── main.go
├── controllers/
│   ├── backupjob_controller.go      (400 lines)
│   ├── backupschedule_controller.go (350 lines)
│   └── restorejob_controller.go     (300 lines)
├── api/
│   └── v1alpha1/
│       ├── backupjob_types.go       (150 lines)
│       ├── backupschedule_types.go  (120 lines)
│       └── restorejob_types.go      (100 lines)
├── pkg/
│   ├── backup/
│   │   └── executor.go              (500 lines)
│   └── restore/
│       └── executor.go              (400 lines)
└── config/
    ├── crd/
    ├── rbac/
    ├── manager/
    └── samples/
```

**Key Components**:

1. **BackupJob Controller**:
   - Watch BackupJob resources
   - Create backup pods
   - Monitor backup progress
   - Update status
   - Clean up completed jobs

2. **BackupSchedule Controller**:
   - Watch BackupSchedule resources
   - Create BackupJob based on schedule
   - Manage job history
   - Handle concurrency policy

3. **RestoreJob Controller**:
   - Watch RestoreJob resources
   - Create restore pods
   - Recreate VirtualMachine resources
   - Update status

**Estimated Lines**: ~2,200 lines Go

---

### 2.3 Webhook Admission Controllers

**Validating Webhook**: Validate BackupJob/RestoreJob specs
**Mutating Webhook**: Set defaults, inject labels

```go
// ValidateBackupJob validates a BackupJob
func (v *BackupJobValidator) ValidateCreate(obj runtime.Object) error {
    job := obj.(*hypersdk.BackupJob)

    // Validate provider exists
    if !isValidProvider(job.Spec.Source.Provider) {
        return fmt.Errorf("invalid provider: %s", job.Spec.Source.Provider)
    }

    // Validate destination
    if err := validateDestination(job.Spec.Destination); err != nil {
        return err
    }

    // Validate carbon-aware settings
    if job.Spec.CarbonAware.Enabled {
        if !isValidCarbonZone(job.Spec.CarbonAware.Zone) {
            return fmt.Errorf("invalid carbon zone: %s", job.Spec.CarbonAware.Zone)
        }
    }

    return nil
}
```

**Estimated Lines**: ~400 lines

---

## 📋 Phase 3: Helm Charts (~2-3 days)

### 3.1 Helm Chart Structure

```
charts/hypersdk/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── statefulset.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── serviceaccount.yaml
│   ├── rbac.yaml
│   ├── pvc.yaml
│   └── hpa.yaml
└── crds/
    ├── backupjob.yaml
    ├── backupschedule.yaml
    └── restorejob.yaml
```

### 3.2 Values Configuration

**File**: `charts/hypersdk/values.yaml`

```yaml
# HyperSDK Operator configuration
operator:
  image:
    repository: ghcr.io/ssahani/hypersdk-operator
    tag: "2.1.0"
    pullPolicy: IfNotPresent

  replicas: 1

  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 100m
      memory: 128Mi

  # Carbon-aware integration
  carbonAware:
    enabled: true
    electricityMapApiKey: ""
    defaultZone: US-CAL-CISO

# HyperSDK Daemon configuration
daemon:
  image:
    repository: ghcr.io/ssahani/hypersdk
    tag: "2.1.0"

  # StatefulSet for state persistence
  replicas: 3

  resources:
    limits:
      cpu: 2000m
      memory: 4Gi
    requests:
      cpu: 500m
      memory: 1Gi

  # Storage for job state
  persistence:
    enabled: true
    storageClass: "standard"
    size: 10Gi

  # Configuration
  config:
    logLevel: info
    maxConcurrentJobs: 10
    jobTimeout: 24h

# Dashboard configuration
dashboard:
  enabled: true

  image:
    repository: ghcr.io/ssahani/hypersdk-dashboard
    tag: "2.1.0"

  replicas: 2

  service:
    type: ClusterIP
    port: 3000

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

# Service account
serviceAccount:
  create: true
  name: hypersdk
  annotations: {}

# RBAC
rbac:
  create: true

# Monitoring
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s

  prometheusRule:
    enabled: true
    rules:
      - alert: BackupJobFailed
        expr: hypersdk_backup_job_failed_total > 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Backup job failed"

      - alert: HighCarbonIntensity
        expr: hypersdk_carbon_intensity_gco2_kwh > 400
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "High grid carbon intensity"

# Backup repository defaults
backupRepository:
  # S3-compatible storage
  s3:
    endpoint: s3.amazonaws.com
    region: us-west-2
    bucket: ""
    accessKeySecret: ""

  # GCS
  gcs:
    bucket: ""
    credentialsSecret: ""

  # Azure Blob
  azure:
    storageAccount: ""
    container: ""
    credentialsSecret: ""

# Provider configurations
providers:
  kubevirt:
    enabled: true

  vsphere:
    enabled: false
    vcenter: ""
    credentialsSecret: ""

  aws:
    enabled: false
    region: us-west-2
    credentialsSecret: ""
```

**Estimated Lines**: ~400 lines YAML

---

### 3.3 Installation

```bash
# Add Helm repository
helm repo add hypersdk https://ssahani.github.io/hypersdk
helm repo update

# Install HyperSDK
helm install hypersdk hypersdk/hypersdk \
  --namespace hypersdk-system \
  --create-namespace \
  --set carbonAware.electricityMapApiKey=YOUR_API_KEY \
  --set dashboard.ingress.hosts[0].host=hypersdk.example.com

# Upgrade
helm upgrade hypersdk hypersdk/hypersdk \
  --namespace hypersdk-system \
  --values custom-values.yaml

# Uninstall
helm uninstall hypersdk --namespace hypersdk-system
```

**Estimated Lines**: ~50 lines docs

---

## 📋 Phase 4: CLI Enhancements (~2-3 days)

### 4.1 Kubernetes Commands

**File**: `cmd/hyperctl/kubernetes_commands.go`

```bash
# List VMs in Kubernetes
hyperctl k8s vms list --namespace default

# Get VM details
hyperctl k8s vms get ubuntu-vm-1 --namespace default

# Start/Stop VMs
hyperctl k8s vms start ubuntu-vm-1 --namespace default
hyperctl k8s vms stop ubuntu-vm-1 --namespace default

# Create backup
hyperctl k8s backup create ubuntu-vm-1 \
  --namespace default \
  --destination s3://my-bucket/backups \
  --carbon-aware

# List backups
hyperctl k8s backup list --namespace default

# Restore VM
hyperctl k8s restore ubuntu-vm-1-20260204 \
  --namespace default \
  --name ubuntu-vm-1-restored

# Manage backup schedules
hyperctl k8s schedule create nightly-backup \
  --cron "0 2 * * *" \
  --namespace default \
  --vm-selector backup=true

# Get operator status
hyperctl k8s operator status

# View metrics
hyperctl k8s metrics --namespace hypersdk-system
```

**Estimated Lines**: ~600 lines

---

### 4.2 Kubeconfig Integration

```bash
# Use default kubeconfig
hyperctl k8s --kubeconfig ~/.kube/config

# Use specific context
hyperctl k8s --context production-cluster

# List contexts
hyperctl k8s contexts list
```

**Estimated Lines**: ~200 lines

---

## 📋 Phase 5: Dashboard Enhancements (~3-4 days)

### 5.1 Kubernetes Integration Tab

**New React Components**:

```typescript
// components/Kubernetes/VMList.tsx
interface KubernetesVM {
  name: string;
  namespace: string;
  status: 'Running' | 'Stopped' | 'Paused' | 'Migrating';
  node: string;
  cpu: number;
  memory: string;
  disks: Disk[];
  networks: Network[];
  created: Date;
}

export function VMList() {
  const [vms, setVMs] = useState<KubernetesVM[]>([]);
  const [selectedNamespace, setSelectedNamespace] = useState('all');

  // WebSocket for real-time updates
  useEffect(() => {
    const ws = new WebSocket('ws://localhost:8080/k8s/vms/watch');
    ws.onmessage = (event) => {
      const vm = JSON.parse(event.data);
      setVMs(prev => updateVM(prev, vm));
    };
    return () => ws.close();
  }, []);

  return (
    <div className="vm-list">
      <NamespaceFilter value={selectedNamespace} onChange={setSelectedNamespace} />
      <VMTable vms={vms} onStart={handleStart} onStop={handleStop} />
    </div>
  );
}
```

**Features**:
- Real-time VM status updates via WebSocket
- Start/Stop/Migrate/Delete VMs
- Resource usage graphs (CPU, memory, disk)
- Network topology visualization
- Backup job management
- Restore interface
- Carbon-aware backup scheduling

**Estimated Lines**: ~1,200 lines TypeScript/React

---

### 5.2 Backup Management UI

```typescript
// components/Kubernetes/BackupManager.tsx
export function BackupManager() {
  return (
    <Tabs>
      <TabPanel title="Backups">
        <BackupJobList />
      </TabPanel>

      <TabPanel title="Schedules">
        <BackupScheduleList />
        <CreateScheduleButton />
      </TabPanel>

      <TabPanel title="Restores">
        <RestoreJobList />
      </TabPanel>

      <TabPanel title="Policies">
        <BackupPolicyEditor />
      </TabPanel>
    </Tabs>
  );
}
```

**Estimated Lines**: ~800 lines TypeScript/React

---

## 📊 Implementation Timeline

### Week 1: KubeVirt Provider
- **Days 1-2**: Provider implementation (kubevirt.go)
- **Days 3-4**: API endpoints and handlers
- **Day 5**: CLI commands and testing

### Week 2: Kubernetes Operator
- **Days 1-2**: CRD definitions and types
- **Days 3-4**: Controller implementation
- **Day 5**: Webhook and validation

### Week 3: Helm, CLI, Dashboard
- **Days 1-2**: Helm charts and installation
- **Day 3**: CLI enhancements
- **Days 4-5**: Dashboard integration

### Total: ~15-18 working days

---

## 📈 Success Metrics

### Technical Metrics
- ✅ Support KubeVirt VirtualMachines
- ✅ CRDs for declarative backup management
- ✅ Operator with reconciliation loops
- ✅ Helm chart with 50+ configuration options
- ✅ Dashboard with real-time updates
- ✅ CLI with full Kubernetes support

### Performance Metrics
- VM list performance: < 1s for 100 VMs
- Backup throughput: > 100 MB/s
- Restore time: < 30 min for 50GB VM
- Operator reconciliation: < 5s

### Business Metrics
- Cloud-native deployment model
- GitOps-ready
- Multi-cluster support
- Integration with Kubernetes ecosystem

---

## 🎯 Key Features Summary

1. **KubeVirt Provider**
   - Full VM lifecycle management
   - Snapshot and backup support
   - Live migration integration

2. **Kubernetes Operator**
   - Declarative backup management
   - Automated scheduling
   - Policy-driven retention

3. **Custom Resources**
   - BackupJob - One-time backups
   - BackupSchedule - Scheduled backups
   - RestoreJob - VM restoration
   - BackupPolicy - Retention policies

4. **Helm Charts**
   - Easy installation
   - Highly configurable
   - Production-ready defaults

5. **Enhanced Dashboard**
   - Real-time VM monitoring
   - Backup management UI
   - Carbon-aware scheduling

6. **CLI Integration**
   - Native kubectl-style commands
   - Kubeconfig support
   - Context switching

---

## 🔧 Dependencies

### Go Modules
```go
require (
    k8s.io/api v0.28.0
    k8s.io/apimachinery v0.28.0
    k8s.io/client-go v0.28.0
    kubevirt.io/api v1.0.0
    kubevirt.io/client-go v1.0.0
    sigs.k8s.io/controller-runtime v0.16.0
    sigs.k8s.io/controller-tools v0.13.0
)
```

### Tools
- Operator SDK v1.32+
- Kubebuilder v3.12+
- Helm v3.12+
- kubectl v1.27+

---

## 📚 Documentation Needed

1. **KUBERNETES_INTEGRATION.md** - Complete guide
2. **KUBEVIRT_PROVIDER.md** - Provider documentation
3. **OPERATOR_GUIDE.md** - Operator deployment
4. **HELM_CHART.md** - Helm chart reference
5. **CRD_REFERENCE.md** - Custom resource specs
6. **EXAMPLES.md** - Usage examples

**Estimated**: ~5,000 lines of documentation

---

## 🚀 Next Steps

1. **Approve Plan** - Review and approve this implementation plan
2. **Phase 1** - Start with KubeVirt provider (Week 1)
3. **Phase 2** - Implement Kubernetes Operator (Week 2)
4. **Phase 3-5** - Helm, CLI, Dashboard (Week 3)
5. **Testing** - Integration and E2E testing
6. **Documentation** - Complete all docs
7. **Release** - v2.1.0 with Kubernetes support

---

*Kubernetes Integration Plan - HyperSDK v2.1.0*
*Created: February 4, 2026*
*Estimated Effort: 2-3 weeks*
*Status: Planning Phase*
