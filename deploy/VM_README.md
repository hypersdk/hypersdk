# HyperSDK Virtual Machine Management

Complete Kubernetes-native virtual machine management with HyperSDK.

**Version**: v2.2.0
**Status**: Production Ready ✅

---

## 🚀 Quick Start

### 1. Install CRDs

```bash
kubectl apply -f deploy/crds/hypersdk.io_virtualmachines.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmoperations.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmtemplates.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmsnapshots.yaml
```

### 2. Deploy Operator

Using Helm:
```bash
helm install hypersdk-operator ./deploy/helm/hypersdk-operator \
  --namespace hypersdk-system \
  --create-namespace
```

Or using kubectl:
```bash
kubectl apply -f deploy/operator/
```

### 3. Create Your First VM

```bash
# Create a template
kubectl apply -f deploy/examples/vmtemplate-ubuntu.yaml

# Create a VM from the template
kubectl apply -f deploy/examples/vm-ubuntu.yaml

# Check VM status
kubectl get vm
```

---

## ✨ Features

### VM Lifecycle Management
- ✅ Create VMs from templates or images
- ✅ Start, stop, restart VMs
- ✅ Full pod and PVC resource orchestration
- ✅ Carbon-aware VM scheduling
- ✅ High availability with auto-restart

### VM Operations
- ✅ Clone VMs (full and linked clones)
- ✅ Live migrate VMs between nodes
- ✅ Resize VMs (CPU/memory with hotplug)
- ✅ Create and restore snapshots
- ✅ Complete state machine and reconciliation

### VM Dashboard
- ✅ Real-time VM monitoring
- ✅ Resource usage tracking
- ✅ Template catalog
- ✅ Snapshot management
- ✅ Quick actions (start/stop/clone/delete)

### VM Templates
- ✅ Pre-configured VM images
- ✅ OS information tracking
- ✅ Default resource specifications
- ✅ Usage count tracking
- ✅ Version management

### VM Snapshots
- ✅ Point-in-time snapshots
- ✅ Memory state capture
- ✅ Quick restore capability
- ✅ Retention policies
- ✅ Snapshot chains

---

## 📚 Documentation

- **[VM Management Guide](../docs/VM_MANAGEMENT.md)** - Complete usage guide
- **[Integration Testing](VM_INTEGRATION_TESTING.md)** - Testing procedures
- **[API Reference](#api-reference)** - CRD specifications
- **[Troubleshooting](#troubleshooting)** - Common issues and solutions

---

## 🎯 Usage Examples

### Create a VM from Template

```bash
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: my-vm
  namespace: default
spec:
  cpus: 4
  memory: "8Gi"
  running: true
  image:
    templateRef:
      name: ubuntu-22-04-template
  disks:
    - name: root
      size: "50Gi"
      storageClass: fast-ssd
  networks:
    - name: default
      type: pod-network
  cloudInit:
    userData: |
      #cloud-config
      hostname: my-vm
      users:
        - name: admin
          sudo: ['ALL=(ALL) NOPASSWD:ALL']
EOF
```

### Clone a VM

```bash
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: clone-my-vm
spec:
  vmRef:
    name: my-vm
  operation: clone
  cloneSpec:
    targetName: my-vm-clone
    linkedClone: true
    startAfterClone: true
EOF
```

### Create a Snapshot

```bash
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMSnapshot
metadata:
  name: my-vm-snapshot
spec:
  vmRef:
    name: my-vm
  includeMemory: true
  quiesce: true
  description: "Before OS upgrade"
  retention:
    keepDays: 30
    autoDelete: true
EOF
```

### Migrate a VM

```bash
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: migrate-my-vm
spec:
  vmRef:
    name: my-vm
  operation: migrate
  migrateSpec:
    targetNode: node-2
    live: true
    bandwidth: "100Mbps"
    autoConverge: true
EOF
```

---

## 🛠️ CLI Commands

HyperSDK provides a powerful CLI for VM management:

### List VMs

```bash
hyperctl k8s -op vm-list -namespace default
# Or use kubectl
kubectl get vm
```

### Create VM

```bash
hyperctl k8s -op vm-create \
  -vm my-vm \
  -cpus 4 \
  -memory 8Gi \
  -template ubuntu-22-04 | kubectl apply -f -
```

### Start/Stop VM

```bash
# Start
hyperctl k8s -op vm-start -vm my-vm | kubectl apply -f -

# Stop
hyperctl k8s -op vm-stop -vm my-vm | kubectl apply -f -

# Restart
hyperctl k8s -op vm-restart -vm my-vm | kubectl apply -f -
```

### Clone VM

```bash
hyperctl k8s -op vm-clone \
  -vm my-vm \
  -target my-vm-clone | kubectl apply -f -
```

### Create Snapshot

```bash
hyperctl k8s -op vm-snapshot-create \
  -vm my-vm \
  -snapshot my-snapshot \
  -include-memory | kubectl apply -f -
```

---

## 📊 Dashboard

Access the VM management dashboard:

```bash
# Port-forward to operator
kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080

# Open in browser
open http://localhost:8080/k8s/vms
```

### Dashboard Features

- **VM Overview**: Real-time VM list with status indicators
- **Running VMs Tab**: All running VMs with resource usage
- **Stopped VMs Tab**: All stopped VMs
- **Templates Tab**: Available VM templates
- **Snapshots Tab**: All VM snapshots
- **Quick Actions**: Start, stop, clone, snapshot, delete
- **Auto-refresh**: Real-time updates every 5 seconds
- **Resource Stats**: Total vCPUs, memory, storage

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                              │
│  ┌──────────────┐      ┌──────────────┐                    │
│  │   VirtualMachine    │   VMTemplate  │                    │
│  │   CRD               │   CRD         │                    │
│  └──────────────┘      └──────────────┘                    │
│         │                      │                             │
│         │                      │                             │
│  ┌──────▼──────────────────────▼──────┐                    │
│  │                                      │                    │
│  │     HyperSDK Operator                │                    │
│  │   (VM Controller + Reconciler)       │                    │
│  │                                      │                    │
│  └──────┬──────────────────────┬───────┘                    │
│         │                      │                             │
│         ▼                      ▼                             │
│  ┌──────────────┐      ┌──────────────┐                    │
│  │   VM Pod     │      │   PVCs        │                    │
│  │   (QEMU)     │      │   (Disks)     │                    │
│  └──────────────┘      └──────────────┘                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘

         │                      │
         ▼                      ▼
  ┌────────────┐        ┌────────────┐
  │  hyperctl  │        │ Dashboard  │
  │    CLI     │        │   Web UI   │
  └────────────┘        └────────────┘
```

### Components

1. **Custom Resource Definitions (CRDs)**
   - VirtualMachine: VM instance definition
   - VMOperation: Async VM operations
   - VMTemplate: Reusable VM configurations
   - VMSnapshot: Point-in-time VM snapshots

2. **Operator Controllers**
   - VirtualMachine Controller: Manages VM lifecycle
   - VMOperation Controller: Handles VM operations
   - VMTemplate Controller: Manages templates
   - VMSnapshot Controller: Manages snapshots

3. **Runtime Components**
   - VM Pod: Runs QEMU/KVM process
   - PVCs: Persistent storage for VM disks
   - Services: Networking for VM access

4. **User Interfaces**
   - hyperctl: CLI for VM management
   - Dashboard: Web UI for monitoring

---

## 🔧 Configuration

### VM Spec

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
spec:
  # Resources
  cpus: 4
  memory: "8Gi"

  # Power state
  running: true

  # Disks
  disks:
    - name: root
      size: "50Gi"
      storageClass: fast-ssd
      bootOrder: 1

  # Networks
  networks:
    - name: default
      type: pod-network

  # Image
  image:
    templateRef:
      name: ubuntu-22-04

  # Cloud-init
  cloudInit:
    userData: |
      #cloud-config
      hostname: my-vm

  # Carbon-aware scheduling
  carbonAware:
    enabled: true
    maxIntensity: 200

  # High Availability
  highAvailability:
    enabled: true
    restartPolicy: Always
    evictionStrategy: LiveMigrate
```

---

## 🧪 Testing

Run the integration test suite:

```bash
# Quick lifecycle test
./deploy/test-vm-lifecycle.sh

# Full test suite
kubectl apply -f deploy/examples/
```

See [VM_INTEGRATION_TESTING.md](VM_INTEGRATION_TESTING.md) for detailed testing procedures.

---

## 🐛 Troubleshooting

### VM Stuck in Pending

**Check node resources:**
```bash
kubectl describe nodes
```

**Check storage:**
```bash
kubectl get storageclass
kubectl get pvc
```

**Check events:**
```bash
kubectl get events --field-selector involvedObject.name=<vm-name>
```

### VM Failed to Start

**Check operator logs:**
```bash
kubectl logs -n hypersdk-system deployment/hypersdk-operator
```

**Check VM pod:**
```bash
kubectl get pod -l vm=<vm-name>
kubectl logs <vm-pod>
kubectl describe pod <vm-pod>
```

### Common Issues

1. **PVC not binding**: Check storage provisioner
2. **Image pull failure**: Verify image source
3. **Insufficient resources**: Scale cluster
4. **Carbon-aware delays**: Disable for testing

See [Troubleshooting Guide](VM_INTEGRATION_TESTING.md#troubleshooting) for more details.

---

## 📋 API Reference

### VirtualMachine

**Phases:**
- `Pending`: VM created, waiting for scheduling
- `Creating`: VM resources being provisioned
- `Running`: VM is running
- `Stopped`: VM is stopped
- `Migrating`: VM is being migrated
- `Failed`: VM failed to start

**Conditions:**
- `Ready`: VM is ready and healthy
- `AgentConnected`: Guest agent is connected
- `DiskReady`: All disks are ready
- `NetworkReady`: All networks configured

### VMOperation

**Operations:**
- `start`: Start a stopped VM
- `stop`: Stop a running VM
- `restart`: Restart a VM
- `clone`: Clone a VM
- `migrate`: Migrate VM to another node
- `resize`: Resize VM resources
- `snapshot`: Create VM snapshot

**Phases:**
- `Pending`: Operation queued
- `Running`: Operation in progress
- `Succeeded`: Operation completed
- `Failed`: Operation failed
- `Cancelled`: Operation cancelled

### VMTemplate

**Fields:**
- `displayName`: Human-readable name
- `version`: Template version
- `osInfo`: OS information
- `defaultSpec`: Default VM configuration
- `image`: Image source and checksum

### VMSnapshot

**Phases:**
- `Pending`: Snapshot initiated
- `Creating`: Snapshot being created
- `Ready`: Snapshot ready for restore
- `Failed`: Snapshot creation failed
- `Expired`: Snapshot past retention

---

## 🤝 Contributing

Contributions welcome! Please see:
- Feature requests: Open an issue
- Bug reports: Include logs and manifests
- Pull requests: Follow code style

---

## 📄 License

LGPL-3.0-or-later

---

## 🔗 Related Documentation

- [Kubernetes Integration Progress](../docs/KUBERNETES_INTEGRATION_PROGRESS.md)
- [Carbon-Aware Quick Start](../docs/CARBON_AWARE_QUICK_START.md)
- [HyperSDK Main Documentation](../README.md)

---

## 📞 Support

- **Issues**: https://github.com/hypersdk/hypersdk/issues
- **Documentation**: https://docs.hypersdk.io
- **Community**: Join our Slack/Discord

---

**HyperSDK VM Management**
Making Kubernetes-native VM management simple and powerful.

🖥️ Create • ▶️ Run • 📋 Clone • 📸 Snapshot • 🔀 Migrate
