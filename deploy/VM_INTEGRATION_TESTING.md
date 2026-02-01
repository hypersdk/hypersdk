# VM Management Integration Testing

Complete guide for testing HyperSDK VM management features on Kubernetes.

**Version**: v2.2.0
**Last Updated**: 2026-02-04

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Testing VM Lifecycle](#testing-vm-lifecycle)
4. [Testing VM Operations](#testing-vm-operations)
5. [Testing Templates](#testing-templates)
6. [Testing Snapshots](#testing-snapshots)
7. [Dashboard Testing](#dashboard-testing)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Components

1. **Kubernetes cluster** (v1.24+)
   ```bash
   kubectl version --client
   kubectl cluster-info
   ```

2. **HyperSDK Operator** deployed
   ```bash
   # Using Helm
   helm install hypersdk-operator ./deploy/helm/hypersdk-operator \
     --namespace hypersdk-system \
     --create-namespace

   # Or using kubectl
   kubectl apply -f deploy/operator/
   ```

3. **VM CRDs** installed
   ```bash
   kubectl apply -f deploy/crds/hypersdk.io_virtualmachines.yaml
   kubectl apply -f deploy/crds/hypersdk.io_vmoperations.yaml
   kubectl apply -f deploy/crds/hypersdk.io_vmtemplates.yaml
   kubectl apply -f deploy/crds/hypersdk.io_vmsnapshots.yaml
   ```

4. **Storage Class** available
   ```bash
   kubectl get storageclass
   ```

### Optional Components

- **hyperctl CLI** for simplified management
- **Dashboard** for web UI testing

---

## Quick Start

### 1. Install CRDs

```bash
kubectl apply -f deploy/crds/hypersdk.io_virtualmachines.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmoperations.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmtemplates.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmsnapshots.yaml
```

### 2. Create a Template

```bash
kubectl apply -f deploy/examples/vmtemplate-ubuntu.yaml
```

### 3. Create a VM

```bash
kubectl apply -f deploy/examples/vm-ubuntu.yaml
```

### 4. Check VM Status

```bash
kubectl get vm
kubectl get vm ubuntu-vm-1 -o yaml
kubectl describe vm ubuntu-vm-1
```

---

## Testing VM Lifecycle

### Test 1: Create VM from Template

**Purpose**: Verify VM creation from template works correctly

```bash
# Create template first
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMTemplate
metadata:
  name: test-template
  namespace: default
spec:
  displayName: "Test Ubuntu Template"
  version: "1.0.0"
  osInfo:
    type: linux
    distribution: ubuntu
    version: "22.04"
  defaultSpec:
    cpus: 2
    memory: "4Gi"
  image:
    source: "ubuntu:22.04"
    format: "qcow2"
EOF

# Wait for template to be ready
kubectl wait --for=condition=Ready vmtemplate/test-template --timeout=60s

# Create VM from template
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: test-vm-1
  namespace: default
spec:
  cpus: 4
  memory: "8Gi"
  running: true
  image:
    templateRef:
      name: test-template
  disks:
    - name: root
      size: "50Gi"
      storageClass: standard
      bootOrder: 1
  networks:
    - name: default
      type: pod-network
EOF

# Verify VM creation
kubectl wait --for=jsonpath='{.status.phase}'=Running vm/test-vm-1 --timeout=300s
kubectl get vm test-vm-1 -o yaml
```

**Expected Result**:
- VM status.phase should be "Running"
- VM should have IP address assigned
- PVCs should be created and bound

### Test 2: Create VM with Raw Image

**Purpose**: Verify VM creation from direct image source

```bash
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: test-vm-2
  namespace: default
spec:
  cpus: 2
  memory: "4Gi"
  running: true
  image:
    source: "s3://mybucket/ubuntu-22.04.qcow2"
  disks:
    - name: root
      size: "30Gi"
      storageClass: standard
  networks:
    - name: default
      type: pod-network
EOF

# Check status
kubectl get vm test-vm-2 -w
```

### Test 3: Stop VM

**Purpose**: Verify VM can be stopped

```bash
# Update VM to stop
kubectl patch vm test-vm-1 --type=merge -p '{"spec":{"running":false}}'

# Or use VMOperation
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: stop-test-vm-1
spec:
  vmRef:
    name: test-vm-1
  operation: stop
EOF

# Wait for stopped state
kubectl wait --for=jsonpath='{.status.phase}'=Stopped vm/test-vm-1 --timeout=120s
```

**Expected Result**:
- VM status.phase should be "Stopped"
- VM pod should be deleted
- PVCs should remain

### Test 4: Restart VM

**Purpose**: Verify VM can be restarted

```bash
# Update VM to start
kubectl patch vm test-vm-1 --type=merge -p '{"spec":{"running":true}}'

# Wait for running state
kubectl wait --for=jsonpath='{.status.phase}'=Running vm/test-vm-1 --timeout=300s
```

### Test 5: Delete VM

**Purpose**: Verify VM deletion cleans up resources

```bash
# Delete VM
kubectl delete vm test-vm-1

# Verify cleanup
kubectl get pvc -l vm=test-vm-1
kubectl get pod -l vm=test-vm-1
```

**Expected Result**:
- VM should be deleted
- All PVCs should be deleted
- VM pod should be deleted

---

## Testing VM Operations

### Test 6: Clone VM

**Purpose**: Verify VM cloning works

```bash
# Create source VM first
kubectl apply -f deploy/examples/vm-ubuntu.yaml
kubectl wait --for=jsonpath='{.status.phase}'=Running vm/ubuntu-vm-1 --timeout=300s

# Clone VM
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: clone-ubuntu
spec:
  vmRef:
    name: ubuntu-vm-1
  operation: clone
  cloneSpec:
    targetName: ubuntu-vm-clone
    linkedClone: false
    startAfterClone: true
EOF

# Wait for operation
kubectl wait --for=condition=Succeeded vmoperation/clone-ubuntu --timeout=600s

# Verify clone
kubectl get vm ubuntu-vm-clone
```

**Expected Result**:
- VMOperation status should be "Succeeded"
- Clone VM should exist
- Clone should have independent disks

### Test 7: Migrate VM

**Purpose**: Verify live migration

```bash
# Get available nodes
kubectl get nodes

# Migrate VM
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: migrate-ubuntu
spec:
  vmRef:
    name: ubuntu-vm-1
  operation: migrate
  migrateSpec:
    targetNode: <node-name>
    live: true
    bandwidth: "100Mbps"
EOF

# Watch migration
kubectl get vmoperation migrate-ubuntu -w
```

**Expected Result**:
- VM should migrate to target node
- No downtime (live migration)
- VM IP should remain the same

### Test 8: Resize VM

**Purpose**: Verify VM can be resized

```bash
# Resize VM (hotplug)
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMOperation
metadata:
  name: resize-ubuntu
spec:
  vmRef:
    name: ubuntu-vm-1
  operation: resize
  resizeSpec:
    cpus: 8
    memory: "16Gi"
    hotplug: true
EOF

# Wait for completion
kubectl wait --for=condition=Succeeded vmoperation/resize-ubuntu --timeout=120s

# Verify resize
kubectl get vm ubuntu-vm-1 -o jsonpath='{.spec.cpus}'
kubectl get vm ubuntu-vm-1 -o jsonpath='{.spec.memory}'
```

---

## Testing Templates

### Test 9: Template CRUD Operations

```bash
# Create template
kubectl apply -f deploy/examples/vmtemplate-ubuntu.yaml

# List templates
kubectl get vmtemplates

# Get template details
kubectl get vmtemplate ubuntu-22-04-template -o yaml

# Use template to create VM
hyperctl k8s -op vm-create \
  -vm test-from-template \
  -template ubuntu-22-04-template \
  -cpus 4 \
  -memory 8Gi | kubectl apply -f -

# Delete template (should fail if in use)
kubectl delete vmtemplate ubuntu-22-04-template
```

### Test 10: Template Versioning

```bash
# Create template v1
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMTemplate
metadata:
  name: ubuntu-template
spec:
  displayName: "Ubuntu 22.04"
  version: "1.0.0"
  image:
    source: "ubuntu:22.04-v1"
EOF

# Create template v2
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMTemplate
metadata:
  name: ubuntu-template-v2
spec:
  displayName: "Ubuntu 22.04"
  version: "2.0.0"
  image:
    source: "ubuntu:22.04-v2"
EOF

# Create VMs from different versions
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: vm-v1
spec:
  cpus: 2
  memory: "4Gi"
  running: true
  image:
    templateRef:
      name: ubuntu-template
EOF
```

---

## Testing Snapshots

### Test 11: Create Snapshot

```bash
# Ensure VM is running
kubectl get vm ubuntu-vm-1

# Create snapshot
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VMSnapshot
metadata:
  name: ubuntu-snapshot-1
spec:
  vmRef:
    name: ubuntu-vm-1
  includeMemory: true
  quiesce: true
  description: "Test snapshot before upgrade"
  retention:
    keepDays: 30
    autoDelete: true
EOF

# Wait for snapshot to be ready
kubectl wait --for=condition=Ready vmsnapshot/ubuntu-snapshot-1 --timeout=300s

# Check snapshot
kubectl get vmsnapshot ubuntu-snapshot-1 -o yaml
```

### Test 12: Restore from Snapshot

```bash
# Create restore operation
kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: RestoreJob
metadata:
  name: restore-from-snapshot
spec:
  source:
    snapshotRef:
      name: ubuntu-snapshot-1
      namespace: default
  destination:
    provider: kubevirt
    namespace: default
    vmName: ubuntu-vm-restored
  options:
    powerOnAfterRestore: true
EOF

# Wait for restore
kubectl wait --for=condition=Succeeded restorejob/restore-from-snapshot --timeout=600s

# Verify restored VM
kubectl get vm ubuntu-vm-restored
```

---

## Dashboard Testing

### Test 13: Access VM Dashboard

```bash
# Port-forward dashboard
kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080

# Open in browser
# http://localhost:8080/k8s/vms
```

**Manual Verification**:
1. ✅ VM list displays correctly
2. ✅ Running/Stopped tabs work
3. ✅ VM status badges show correct colors
4. ✅ Resource stats are accurate
5. ✅ Action buttons are functional
6. ✅ Real-time updates work (5s interval)
7. ✅ Templates tab shows all templates
8. ✅ Snapshots tab shows all snapshots

### Test 14: Dashboard API Endpoints

```bash
# Test VM metrics endpoint
curl http://localhost:8080/api/k8s/vm-metrics | jq

# Test VMs list endpoint
curl http://localhost:8080/api/k8s/vms | jq

# Test templates endpoint
curl http://localhost:8080/api/k8s/templates | jq

# Test snapshots endpoint
curl http://localhost:8080/api/k8s/snapshots | jq

# Test specific VM details
curl http://localhost:8080/api/k8s/vms/default/ubuntu-vm-1 | jq
```

---

## Troubleshooting

### VM Stuck in Pending

**Symptoms**: VM stays in "Pending" phase

**Diagnosis**:
```bash
kubectl describe vm <vm-name>
kubectl get events --field-selector involvedObject.name=<vm-name>
kubectl logs -n hypersdk-system deployment/hypersdk-operator
```

**Common Causes**:
- No nodes with sufficient resources
- Storage class not available
- Carbon-aware scheduling waiting for low intensity
- Node selector/affinity not matching any nodes

**Solutions**:
```bash
# Check node resources
kubectl describe nodes

# Check storage classes
kubectl get storageclass

# Disable carbon-aware if testing
kubectl patch vm <vm-name> --type=json -p='[{"op": "remove", "path": "/spec/carbonAware"}]'
```

### PVC Not Binding

**Symptoms**: VM stuck in "Creating" phase, PVCs pending

**Diagnosis**:
```bash
kubectl get pvc
kubectl describe pvc <pvc-name>
```

**Solutions**:
```bash
# Check if storage provisioner is running
kubectl get pods -n kube-system | grep provisioner

# Check storage class
kubectl get storageclass

# Manually provision PV if using local storage
```

### VM Pod Failing

**Symptoms**: VM shows "Failed" phase

**Diagnosis**:
```bash
kubectl get pod -l vm=<vm-name>
kubectl logs <vm-pod-name>
kubectl describe pod <vm-pod-name>
```

**Common Causes**:
- Image pull failure
- Insufficient node resources
- Security context issues
- Volume mount failures

### Dashboard Not Showing VMs

**Symptoms**: Dashboard loads but shows empty state

**Diagnosis**:
```bash
# Check operator logs
kubectl logs -n hypersdk-system deployment/hypersdk-operator

# Test API endpoint directly
curl http://localhost:8080/api/k8s/vms

# Check if CRDs are installed
kubectl get crd virtualmachines.hypersdk.io
```

**Solutions**:
```bash
# Restart operator
kubectl rollout restart -n hypersdk-system deployment/hypersdk-operator

# Check RBAC permissions
kubectl get clusterrole hypersdk-operator -o yaml
```

---

## Performance Testing

### Load Test: Multiple VMs

```bash
# Create 10 VMs
for i in {1..10}; do
  kubectl apply -f - <<EOF
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: load-test-vm-$i
spec:
  cpus: 2
  memory: "4Gi"
  running: true
  image:
    templateRef:
      name: ubuntu-22-04-template
  disks:
    - name: root
      size: "20Gi"
  networks:
    - name: default
      type: pod-network
EOF
done

# Monitor creation
watch kubectl get vm
```

### Stress Test: VM Operations

```bash
# Rapid start/stop cycles
for i in {1..5}; do
  kubectl patch vm load-test-vm-1 --type=merge -p '{"spec":{"running":false}}'
  sleep 30
  kubectl patch vm load-test-vm-1 --type=merge -p '{"spec":{"running":true}}'
  sleep 30
done
```

---

## Cleanup

```bash
# Delete all test VMs
kubectl delete vm --all

# Delete all snapshots
kubectl delete vmsnapshot --all

# Delete all templates
kubectl delete vmtemplate --all

# Delete operator
helm uninstall hypersdk-operator -n hypersdk-system

# Delete CRDs
kubectl delete crd virtualmachines.hypersdk.io
kubectl delete crd vmoperations.hypersdk.io
kubectl delete crd vmsnapshots.hypersdk.io
kubectl delete crd vmtemplates.hypersdk.io
```

---

## Success Criteria

✅ All VMs create successfully
✅ VMs can be started and stopped
✅ VM operations (clone, migrate, resize) work
✅ Snapshots can be created and restored
✅ Templates work for VM creation
✅ Dashboard shows real-time VM data
✅ No resource leaks (PVCs cleaned up)
✅ Operator logs show no errors

---

**HyperSDK VM Integration Testing**
Version: v2.2.0
Date: 2026-02-04
