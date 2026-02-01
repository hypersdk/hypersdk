## HyperSDK Virtual Machine Management

Complete guide to managing virtual machines with HyperSDK on Kubernetes.

**Version**: v2.2.0
**Status**: Phase 1 Complete - VM Lifecycle Management

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Virtual Machine CRD](#virtualmachine-crd)
4. [VM Operations](#vm-operations)
5. [VM Templates](#vm-templates)
6. [VM Snapshots](#vm-snapshots)
7. [CLI Commands](#cli-commands)
8. [Dashboard](#dashboard)
9. [Advanced Features](#advanced-features)
10. [Troubleshooting](#troubleshooting)

---

## Overview

HyperSDK provides Kubernetes-native virtual machine management with:

✅ **Full VM Lifecycle**: Create, start, stop, restart, clone, migrate, delete
✅ **VM Templates**: Quick provisioning from pre-configured images
✅ **Snapshots**: Point-in-time VM snapshots with memory state
✅ **Carbon-Aware**: Optimize VM placement for minimal carbon footprint
✅ **High Availability**: Auto-restart and live migration on node failure
✅ **Web Dashboard**: Manage VMs through intuitive web interface
✅ **CLI Integration**: kubectl-style CLI commands

---

## Quick Start

### 1. Install CRDs

```bash
kubectl apply -f deploy/crds/hypersdk.io_virtualmachines.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmoperations.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmtemplates.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmsnapshots.yaml
```

### 2. Create a VM Template

```bash
kubectl apply -f deploy/examples/vmtemplate-ubuntu.yaml
```

### 3. Create a VM from Template

```bash
cat <<EOF | kubectl apply -f -
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
  cloudInit:
    userData: |
      #cloud-config
      hostname: my-vm
      users:
        - name: admin
          sudo: ['ALL=(ALL) NOPASSWD:ALL']
          ssh-authorized-keys:
            - ssh-rsa AAAA... user@host
EOF
```

### 4. Check VM Status

```bash
kubectl get vm
kubectl get vm my-vm -o yaml
kubectl describe vm my-vm
```

---

## VirtualMachine CRD

### Basic VM Configuration

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: ubuntu-vm
  namespace: default
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
    - name: data
      size: "100Gi"
      storageClass: standard

  # Networks
  networks:
    - name: default
      type: pod-network

  # Image
  image:
    source: "s3://my-bucket/ubuntu-22.04.qcow2"

  # Cloud-init
  cloudInit:
    userData: |
      #cloud-config
      hostname: ubuntu-vm
```

### Advanced Features

#### Carbon-Aware Scheduling

```yaml
spec:
  carbonAware:
    enabled: true
    maxIntensity: 200  # gCO2/kWh
    preferGreenEnergy: true
    zone: "US-CAL-CISO"
```

#### High Availability

```yaml
spec:
  highAvailability:
    enabled: true
    restartPolicy: Always
    restartDelay: "30s"
    maxRestarts: 3
    evictionStrategy: LiveMigrate
```

#### Node Placement

```yaml
spec:
  nodeSelector:
    storage: ssd
    zone: us-west-1

  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
              - key: kubernetes.io/hostname
                operator: In
                values:
                  - node-1
                  - node-2
```

### VM Status

```yaml
status:
  phase: Running
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-02-04T10:00:00Z"
  guestAgent:
    connected: true
    version: "1.0.0"
    hostname: ubuntu-vm
  ipAddresses:
    - 10.244.1.5
    - 192.168.1.100
  nodeName: node-1
  resources:
    cpu:
      usage: "45%"
      requests: 4
    memory:
      usage: "6.2Gi"
      requests: "8Gi"
  qemuPid: 12345
  vnc:
    port: 5900
    nodePort: 30590
    enabled: true
```

---

## VM Operations

### Start VM

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: start-my-vm
spec:
  vmRef:
    name: my-vm
  operation: start
```

Or use CLI:
```bash
hyperctl k8s vm start my-vm
```

### Stop VM

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: stop-my-vm
spec:
  vmRef:
    name: my-vm
  operation: stop
  force: false  # Set to true for forced shutdown
```

### Clone VM

```yaml
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
    linkedClone: true  # Faster, uses less space
    startAfterClone: true
```

CLI:
```bash
hyperctl k8s vm clone my-vm -target my-vm-clone --linked
```

### Migrate VM

```yaml
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
    live: true  # No downtime
    bandwidth: "100Mbps"
    autoConverge: true
```

CLI:
```bash
hyperctl k8s vm migrate my-vm -node node-2 --live
```

### Resize VM

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: resize-my-vm
spec:
  vmRef:
    name: my-vm
  operation: resize
  resizeSpec:
    cpus: 8
    memory: "16Gi"
    hotplug: true  # Hot-add without reboot
```

CLI:
```bash
hyperctl k8s vm resize my-vm -cpus 8 -memory 16Gi --hotplug
```

---

## VM Templates

### Create Template

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VMTemplate
metadata:
  name: ubuntu-22-04
  namespace: default
spec:
  displayName: "Ubuntu 22.04 LTS"
  description: "Ubuntu 22.04 with Docker pre-installed"
  version: "1.0.0"
  tags:
    - ubuntu
    - linux
    - docker

  osInfo:
    type: linux
    distribution: ubuntu
    version: "22.04"

  defaultSpec:
    cpus: 2
    memory: "4Gi"
    disks:
      - name: root
        size: "30Gi"

  image:
    source: "s3://templates/ubuntu-22.04-v1.0.0.qcow2"
    format: qcow2
    checksum:
      type: sha256
      value: "abc123..."
```

### List Templates

```bash
kubectl get vmtemplates
hyperctl k8s template list
```

### Create VM from Template

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: web-vm
spec:
  cpus: 4  # Override template defaults
  memory: "8Gi"
  running: true
  image:
    templateRef:
      name: ubuntu-22-04
```

CLI:
```bash
hyperctl k8s vm create -from-template ubuntu-22-04 -name web-vm
```

---

## VM Snapshots

### Create Snapshot

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VMSnapshot
metadata:
  name: my-vm-snapshot-1
spec:
  vmRef:
    name: my-vm
  includeMemory: true  # Include memory for live restore
  quiesce: true  # Quiesce filesystem
  description: "Before OS upgrade"
  retention:
    keepDays: 30
    autoDelete: true
```

CLI:
```bash
hyperctl k8s vm snapshot create my-vm -name pre-upgrade --memory
```

### List Snapshots

```bash
kubectl get vmsnapshots
hyperctl k8s vm snapshot list my-vm
```

### Restore from Snapshot

```bash
hyperctl k8s vm snapshot restore my-vm -snapshot my-vm-snapshot-1
```

### Delete Snapshot

```bash
kubectl delete vmsnapshot my-vm-snapshot-1
hyperctl k8s vm snapshot delete my-vm -snapshot my-vm-snapshot-1
```

---

## CLI Commands

### VM Management

```bash
# Create VM
hyperctl k8s vm create -name my-vm -cpus 4 -memory 8Gi -image ubuntu:22.04

# List VMs
hyperctl k8s vm list [-n namespace] [-o json|yaml]

# Get VM details
hyperctl k8s vm get my-vm [-n namespace]

# Delete VM
hyperctl k8s vm delete my-vm
```

### VM Operations

```bash
# Start/Stop/Restart
hyperctl k8s vm start my-vm
hyperctl k8s vm stop my-vm [--force]
hyperctl k8s vm restart my-vm

# Clone
hyperctl k8s vm clone my-vm -target my-vm-2 [--linked]

# Migrate
hyperctl k8s vm migrate my-vm -node node-2 [--live]

# Resize
hyperctl k8s vm resize my-vm -cpus 8 -memory 16Gi [--hotplug]
```

### VM Console

```bash
# VNC console
hyperctl k8s vm console my-vm --vnc

# Serial console
hyperctl k8s vm console my-vm --serial

# SSH (if guest agent running)
hyperctl k8s vm ssh my-vm -user admin -key ~/.ssh/id_rsa
```

### Snapshots

```bash
# Create snapshot
hyperctl k8s vm snapshot create my-vm -name snap1 [--memory]

# List snapshots
hyperctl k8s vm snapshot list my-vm

# Restore snapshot
hyperctl k8s vm snapshot restore my-vm -snapshot snap1

# Delete snapshot
hyperctl k8s vm snapshot delete my-vm -snapshot snap1
```

### Templates

```bash
# List templates
hyperctl k8s template list

# Get template details
hyperctl k8s template get ubuntu-22-04

# Create template from VM
hyperctl k8s template create -name my-template -from my-vm

# Create VM from template
hyperctl k8s vm create -from-template ubuntu-22-04 -name new-vm
```

---

## Dashboard

Access the VM dashboard:

```bash
# Port-forward to operator
kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080

# Open in browser
open http://localhost:8080/k8s/vms
```

### Dashboard Features

**VM List Page** (`/k8s/vms`):
- View all VMs with status
- Filter by namespace/node/status
- Quick actions (Start/Stop/Console/Delete)
- Resource usage indicators
- Bulk operations

**VM Detail Page** (`/k8s/vms/{namespace}/{name}`):
- Overview with status and resources
- Real-time metrics (CPU/Memory/Disk/Network)
- Embedded VNC console
- Serial console
- Disk management
- Network management
- Snapshot management
- Event log and operation history

---

## Advanced Features

### Multi-Network VMs

```yaml
spec:
  networks:
    - name: default
      type: pod-network
    - name: external
      type: multus
      multusNetworkName: external-net
    - name: storage
      type: multus
      multusNetworkName: storage-net
```

### GPU Passthrough

```yaml
spec:
  devices:
    gpus:
      - name: nvidia-tesla-t4
        count: 1
```

### Custom QEMU Args

```yaml
spec:
  machineType: q35
  firmware:
    bootloader: uefi
    secureBoot: true
```

### Automated Backups

Integrate with BackupSchedule CRD:

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: BackupSchedule
metadata:
  name: vm-daily-backup
spec:
  schedule: "0 2 * * *"
  vmSelector:
    matchLabels:
      backup: enabled
  destination:
    type: s3
    bucket: vm-backups
```

---

## Troubleshooting

### VM Won't Start

```bash
# Check VM status
kubectl describe vm my-vm

# Check events
kubectl get events --field-selector involvedObject.name=my-vm

# Check logs
kubectl logs -n hypersdk-system deployment/hypersdk-operator
```

### VM Stuck in Pending

Check node resources:
```bash
kubectl describe node <node-name>
```

Check storage:
```bash
kubectl get pvc
```

### Cannot Connect to VNC

Check VNC status:
```bash
kubectl get vm my-vm -o jsonpath='{.status.vnc}'
```

Port-forward to VNC:
```bash
kubectl port-forward pod/my-vm-pod 5900:5900
```

### Migration Fails

Check target node:
```bash
kubectl get node <target-node>
```

Check migration logs:
```bash
kubectl logs -n hypersdk-system deployment/hypersdk-operator | grep migrate
```

---

## API Reference

### VirtualMachine Phases

- `Pending`: VM resource created, waiting for scheduling
- `Creating`: VM is being created (disk provisioning, etc.)
- `Running`: VM is running
- `Stopped`: VM is stopped
- `Migrating`: VM is being migrated
- `Paused`: VM is paused
- `Failed`: VM failed to start or crashed
- `Unknown`: VM state unknown

### VMOperation Phases

- `Pending`: Operation queued
- `Running`: Operation in progress
- `Succeeded`: Operation completed successfully
- `Failed`: Operation failed
- `Cancelled`: Operation was cancelled

### Condition Types

- `Ready`: VM is ready and healthy
- `AgentConnected`: Guest agent is connected
- `LivenessProbe`: VM responds to liveness probe
- `DiskReady`: All disks are ready
- `NetworkReady`: All networks are configured

---

## Next Steps

1. **Phase 2: VM Monitoring** - Real-time metrics and dashboards
2. **Phase 3: VM Console** - Web-based VNC and serial console
3. **Phase 4: Networking & Storage** - Advanced network and storage features
4. **Phase 5: HA & Auto-Scaling** - High availability and auto-scaling
5. **Phase 6: Advanced Features** - GPU, migration policies, etc.

---

**HyperSDK VM Management**
Version: v2.2.0-alpha
Date: 2026-02-04
