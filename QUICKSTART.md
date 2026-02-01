# HyperSDK Kubernetes Integration - Quick Start Guide

**Get HyperSDK VM management running on your Kubernetes cluster in 10 minutes!**

Version: v2.2.0
Status: Production Ready

---

## Prerequisites

Before you begin, ensure you have:

### Required Tools
- **kubectl** (v1.25+) - [Install Guide](https://kubernetes.io/docs/tasks/tools/)
- **Helm** (v3.8+) - [Install Guide](https://helm.sh/docs/intro/install/)
- **Kubernetes cluster** (v1.25+) with:
  - At least 2 worker nodes (for VM migration)
  - 8+ GB RAM available per node
  - Storage class configured (for VM disks)

### Verify Prerequisites

```bash
# Check kubectl
kubectl version --client

# Check Helm
helm version

# Check cluster access
kubectl cluster-info

# Check nodes
kubectl get nodes

# Check storage classes
kubectl get storageclass
```

---

## Step 1: Clone Repository

```bash
git clone https://github.com/your-org/hypersdk.git
cd hypersdk
```

---

## Step 2: Install Custom Resource Definitions (CRDs)

Install all HyperSDK CRDs:

```bash
# Install VM CRDs
kubectl apply -f deploy/crds/hypersdk.io_virtualmachines.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmoperations.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmtemplates.yaml
kubectl apply -f deploy/crds/hypersdk.io_vmsnapshots.yaml

# Install Backup CRDs
kubectl apply -f deploy/crds/hypersdk.io_backupjobs.yaml
kubectl apply -f deploy/crds/hypersdk.io_backupschedules.yaml
kubectl apply -f deploy/crds/hypersdk.io_restorejobs.yaml
```

### Verify CRD Installation

```bash
# Check CRDs are installed
kubectl get crd | grep hypersdk.io

# Expected output:
# backupjobs.hypersdk.io
# backupschedules.hypersdk.io
# restorejobs.hypersdk.io
# virtualmachines.hypersdk.io
# vmoperations.hypersdk.io
# vmsnapshots.hypersdk.io
# vmtemplates.hypersdk.io
```

---

## Step 3: Deploy HyperSDK Operator

Deploy the operator using Helm:

```bash
# Create namespace
kubectl create namespace hypersdk-system

# Install operator
helm install hypersdk-operator ./deploy/helm/hypersdk-operator \
  --namespace hypersdk-system \
  --create-namespace
```

### Verify Operator Deployment

```bash
# Check operator pod
kubectl get pods -n hypersdk-system

# Expected output:
# NAME                                READY   STATUS    RESTARTS   AGE
# hypersdk-operator-xxxxxxxxx-xxxxx   1/1     Running   0          30s

# Check operator logs
kubectl logs -n hypersdk-system -l app=hypersdk-operator --tail=20
```

---

## Step 4: Access the Dashboard

The HyperSDK dashboard provides real-time monitoring and management:

```bash
# Port-forward dashboard to local machine
kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080
```

**Open in your browser**: http://localhost:8080

### Dashboard Features
- **Main Dashboard** (/) - System overview
- **K8s Resources** (/k8s) - Kubernetes resources
- **VM Management** (/k8s/vms) - Virtual machines
- **Charts & Analytics** (/k8s/charts) - Metrics and visualizations

---

## Step 5: Create Your First VM

### Option A: Using Example Templates (Recommended)

```bash
# 1. Create a VM template
kubectl apply -f deploy/examples/vmtemplate-ubuntu.yaml

# 2. Verify template
kubectl get vmtemplate

# 3. Create VM from template
kubectl apply -f deploy/examples/vm-ubuntu.yaml

# 4. Check VM status
kubectl get vm
```

### Option B: Using CLI

```bash
# Generate VM manifest
hyperctl k8s -op vm-create \
  -vm my-first-vm \
  -cpus 2 \
  -memory 4Gi \
  -disk 20Gi > my-vm.yaml

# Review the manifest
cat my-vm.yaml

# Apply to cluster
kubectl apply -f my-vm.yaml
```

### Watch VM Creation

```bash
# Watch VM status
kubectl get vm -w

# Or check detailed status
kubectl describe vm ubuntu-vm
```

---

## Step 6: Manage Your VM

### Start the VM

```bash
# Using CLI
hyperctl k8s -op vm-start -vm ubuntu-vm | kubectl apply -f -

# Or using kubectl
kubectl patch vm ubuntu-vm --type=merge -p '{"spec":{"powerState":"Running"}}'
```

### Check VM Status

```bash
# Get VM info
kubectl get vm ubuntu-vm -o wide

# View VM details
kubectl describe vm ubuntu-vm

# Check VM logs
kubectl logs -n default $(kubectl get pod -l hypersdk.io/vm=ubuntu-vm -o name)
```

### Stop the VM

```bash
# Using CLI
hyperctl k8s -op vm-stop -vm ubuntu-vm | kubectl apply -f -
```

### Access VM Console (if configured)

```bash
# Port-forward to VM pod
kubectl port-forward pod/ubuntu-vm-xxxxx 2222:22

# SSH to VM (if SSH is configured)
ssh -p 2222 user@localhost
```

---

## Step 7: Advanced Operations

### Create a Snapshot

```bash
# Using CLI
hyperctl k8s -op vm-snapshot-create \
  -vm ubuntu-vm \
  -snapshot ubuntu-snapshot-1 | kubectl apply -f -

# Check snapshot
kubectl get vmsnapshot
```

### Clone a VM

```bash
# Using CLI
hyperctl k8s -op vm-clone \
  -vm ubuntu-vm \
  -target ubuntu-vm-clone | kubectl apply -f -

# Watch clone operation
kubectl get vmoperation -w
```

### Migrate VM to Another Node

```bash
# Using CLI
hyperctl k8s -op vm-migrate \
  -vm ubuntu-vm \
  -target-node worker-node-2 | kubectl apply -f -

# Watch migration
kubectl get vmoperation -w
```

### Resize VM

```bash
# Using CLI
hyperctl k8s -op vm-resize \
  -vm ubuntu-vm \
  -cpus 4 \
  -memory 8Gi | kubectl apply -f -
```

---

## Step 8: Monitor with Dashboard

### View VMs in Dashboard

1. Open http://localhost:8080/k8s/vms
2. Navigate between tabs:
   - **Running VMs** - Active virtual machines
   - **Stopped VMs** - Stopped virtual machines
   - **Templates** - Available VM templates
   - **Snapshots** - VM snapshots

### View Charts

1. Open http://localhost:8080/k8s/charts
2. Explore 12 interactive charts:
   - VM trend over time
   - VM status distribution
   - VMs by node
   - Resource allocation
   - Carbon intensity
   - Storage distribution
   - And more...

---

## Common Tasks

### List All VMs

```bash
kubectl get vm
```

### Get VM Details

```bash
kubectl get vm ubuntu-vm -o yaml
```

### Delete a VM

```bash
kubectl delete vm ubuntu-vm

# Or using CLI
hyperctl k8s -op vm-delete -vm ubuntu-vm | kubectl apply -f -
```

### List VM Templates

```bash
kubectl get vmtemplate
```

### List VM Operations

```bash
kubectl get vmoperation
```

### List Snapshots

```bash
kubectl get vmsnapshot
```

---

## Troubleshooting

### VM Stuck in Pending

```bash
# Check VM events
kubectl describe vm <vm-name>

# Check operator logs
kubectl logs -n hypersdk-system -l app=hypersdk-operator --tail=50

# Check if template exists (if using template)
kubectl get vmtemplate <template-name>

# Check storage class
kubectl get storageclass
```

### VM Pod Not Starting

```bash
# Check pod events
kubectl describe pod <vm-pod-name>

# Check pod logs
kubectl logs <vm-pod-name>

# Check resource availability
kubectl describe node <node-name>
```

### Dashboard Not Accessible

```bash
# Check operator pod status
kubectl get pods -n hypersdk-system

# Check service
kubectl get svc -n hypersdk-system

# Check operator logs
kubectl logs -n hypersdk-system -l app=hypersdk-operator

# Verify port-forward is running
ps aux | grep port-forward
```

### Operation Stuck

```bash
# Check operation status
kubectl describe vmoperation <operation-name>

# Check operator logs
kubectl logs -n hypersdk-system -l app=hypersdk-operator --tail=100

# Delete stuck operation (if needed)
kubectl delete vmoperation <operation-name>
```

### CRD Installation Issues

```bash
# Verify CRDs are installed
kubectl get crd | grep hypersdk

# Re-install specific CRD
kubectl apply -f deploy/crds/hypersdk.io_virtualmachines.yaml

# Check CRD details
kubectl describe crd virtualmachines.hypersdk.io
```

---

## Configuration Options

### Customize Operator Deployment

Edit `deploy/helm/hypersdk-operator/values.yaml` or use `--set`:

```bash
# Custom replica count
helm install hypersdk-operator ./deploy/helm/hypersdk-operator \
  --namespace hypersdk-system \
  --set replicaCount=2

# Custom resource limits
helm install hypersdk-operator ./deploy/helm/hypersdk-operator \
  --namespace hypersdk-system \
  --set resources.limits.memory=1Gi

# Enable carbon-aware scheduling
helm install hypersdk-operator ./deploy/helm/hypersdk-operator \
  --namespace hypersdk-system \
  --set carbonAware.enabled=true
```

### Configure Storage

Specify storage class for VMs:

```yaml
apiVersion: hypersdk.io/v1alpha1
kind: VirtualMachine
metadata:
  name: my-vm
spec:
  storage:
    storageClassName: fast-ssd  # Your storage class
    size: 50Gi
```

---

## Cleanup

### Uninstall Everything

```bash
# Delete all VMs
kubectl delete vm --all

# Delete all VM operations
kubectl delete vmoperation --all

# Delete all templates
kubectl delete vmtemplate --all

# Delete all snapshots
kubectl delete vmsnapshot --all

# Uninstall operator
helm uninstall hypersdk-operator -n hypersdk-system

# Delete namespace
kubectl delete namespace hypersdk-system

# Delete CRDs (optional - removes all custom resources)
kubectl delete crd virtualmachines.hypersdk.io
kubectl delete crd vmoperations.hypersdk.io
kubectl delete crd vmsnapshots.hypersdk.io
kubectl delete crd vmtemplates.hypersdk.io
kubectl delete crd backupjobs.hypersdk.io
kubectl delete crd backupschedules.hypersdk.io
kubectl delete crd restorejobs.hypersdk.io
```

---

## Next Steps

### Learn More

- [VM Management Guide](docs/VM_MANAGEMENT.md) - Comprehensive VM guide
- [Integration Testing](deploy/VM_INTEGRATION_TESTING.md) - Testing procedures
- [CLI Enhancements](docs/CLI_ENHANCEMENTS_GUIDE.md) - Advanced CLI features
- [Final Report](FINAL_COMPLETION_REPORT.md) - Complete feature list

### Production Deployment

For production use, consider:

1. **Storage**: Use fast storage class (SSD) for VM disks
2. **Resources**: Ensure adequate node resources (CPU, RAM)
3. **Networking**: Configure network policies for VM isolation
4. **Backup**: Set up regular VM snapshots
5. **Monitoring**: Enable operator metrics and alerts
6. **High Availability**: Run operator with multiple replicas
7. **Security**: Configure RBAC and pod security policies

### Run Integration Tests

```bash
# Automated test suite
./deploy/test-vm-lifecycle.sh

# Manual testing guide
See deploy/VM_INTEGRATION_TESTING.md
```

### Contributing

- Report issues on GitHub
- Submit pull requests
- Improve documentation
- Share your use cases

---

## Quick Reference

### Essential Commands

```bash
# VMs
kubectl get vm                           # List VMs
kubectl describe vm <name>               # VM details
kubectl delete vm <name>                 # Delete VM

# Templates
kubectl get vmtemplate                   # List templates
kubectl describe vmtemplate <name>       # Template details

# Operations
kubectl get vmoperation                  # List operations
kubectl describe vmoperation <name>      # Operation details

# Snapshots
kubectl get vmsnapshot                   # List snapshots
kubectl describe vmsnapshot <name>       # Snapshot details

# Dashboard
kubectl port-forward -n hypersdk-system deployment/hypersdk-operator 8080:8080

# Logs
kubectl logs -n hypersdk-system -l app=hypersdk-operator -f
```

### Useful Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc
alias k='kubectl'
alias kgvm='kubectl get vm'
alias kdvm='kubectl describe vm'
alias kgvmo='kubectl get vmoperation'
alias klogs='kubectl logs -n hypersdk-system -l app=hypersdk-operator'
```

---

## Support

- **Documentation**: `/docs` directory
- **Examples**: `/deploy/examples` directory
- **Testing**: `/deploy/test-vm-lifecycle.sh`
- **Issues**: GitHub Issues

---

## License

LGPL-3.0-or-later

---

**HyperSDK - Enterprise VM Management on Kubernetes**

Version: v2.2.0
Status: Production Ready
Built with Claude Code
