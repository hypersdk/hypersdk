# NFS Shared Storage for Cross-Environment VM Exports

This guide explains how to deploy HyperSDK in Kubernetes with NFS-backed storage so that exported VMs can be consumed by hyper2kvm or other tools running in either native Linux or Kubernetes environments.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                       NFS Server                                 │
│                  /exports/hypersdk                              │
└────────────────────────┬────────────────────────────────────────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
          ▼              ▼              ▼
┌─────────────────┐ ┌─────────────┐ ┌──────────────────┐
│  Kubernetes     │ │  Kubernetes │ │  Native Linux    │
│  HyperSDK Pod   │ │  hyper2kvm  │ │  hyper2kvm       │
│                 │ │  Pod        │ │                  │
│  Exports to:    │ │  Reads:     │ │  Reads:          │
│  /exports       │ │  /imports   │ │  /mnt/nfs/...    │
│  (NFS PVC)      │ │  (NFS PVC)  │ │  (NFS mount)     │
└─────────────────┘ └─────────────┘ └──────────────────┘
```

## Workflow

1. **HyperSDK in Kubernetes** exports VMs to NFS-backed PersistentVolume
2. **NFS Server** provides shared storage accessible from multiple environments
3. **hyper2kvm in Kubernetes** reads exports from same NFS via PVC
4. **hyper2kvm on native Linux** reads exports from same NFS via direct mount

## Prerequisites

- Kubernetes cluster (1.24+)
- NFS server with sufficient storage (1TB+ recommended)
- kubectl with cluster access
- NFS client utilities on worker nodes

## Part 1: NFS Server Setup

### Option A: Existing NFS Server

If you have an existing NFS server, create an export for HyperSDK:

```bash
# On NFS server
sudo mkdir -p /exports/hypersdk
sudo chown -R 1000:1000 /exports/hypersdk
sudo chmod 755 /exports/hypersdk

# Add to /etc/exports
echo "/exports/hypersdk *(rw,sync,no_subtree_check,no_root_squash)" | sudo tee -a /etc/exports

# Apply changes
sudo exportfs -ra
sudo systemctl restart nfs-server
```

### Option B: Deploy NFS Server in Kubernetes

For testing/development, deploy an NFS server in Kubernetes:

```yaml
# nfs-server.yaml
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: nfs-storage
  namespace: nfs-server
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 2Ti
  storageClassName: standard  # Use your storage class

---
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
        - name: mountd
          containerPort: 20048
        - name: rpcbind
          containerPort: 111
        securityContext:
          privileged: true
        volumeMounts:
        - name: storage
          mountPath: /exports
      volumes:
      - name: storage
        persistentVolumeClaim:
          claimName: nfs-storage

---
apiVersion: v1
kind: Service
metadata:
  name: nfs-server
  namespace: nfs-server
spec:
  ports:
  - name: nfs
    port: 2049
  - name: mountd
    port: 20048
  - name: rpcbind
    port: 111
  selector:
    app: nfs-server
```

Deploy:
```bash
kubectl create namespace nfs-server
kubectl apply -f nfs-server.yaml
kubectl get svc -n nfs-server nfs-server
```

## Part 2: Configure Kubernetes to Use NFS Storage

### Create NFS PersistentVolume and PersistentVolumeClaim

```yaml
# nfs-pv-pvc.yaml
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: hypersdk-exports-nfs
spec:
  capacity:
    storage: 2Ti
  accessModes:
    - ReadWriteMany  # Multiple pods can read/write
  persistentVolumeReclaimPolicy: Retain
  nfs:
    server: 192.168.1.100  # Your NFS server IP or DNS
    path: /exports/hypersdk
  mountOptions:
    - nfsvers=4.1
    - hard
    - timeo=600
    - retrans=2
    - rsize=1048576
    - wsize=1048576

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-exports
  namespace: hypersdk
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 2Ti
  volumeName: hypersdk-exports-nfs  # Bind to specific PV
  storageClassName: ""  # Empty for static provisioning
```

Apply:
```bash
# Create namespace if not exists
kubectl create namespace hypersdk

# Apply PV and PVC
kubectl apply -f nfs-pv-pvc.yaml

# Verify
kubectl get pv hypersdk-exports-nfs
kubectl get pvc -n hypersdk hypersdk-exports
```

Expected output:
```
NAME                    CAPACITY   ACCESS MODES   STATUS   CLAIM
hypersdk-exports-nfs    2Ti        RWX            Bound    hypersdk/hypersdk-exports
```

## Part 3: Deploy HyperSDK with NFS Storage

### Update Kustomization for NFS

Edit `deployments/kubernetes/overlays/production/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: hypersdk

resources:
  - ../../base
  - ingress.yaml
  - hpa.yaml
  - networkpolicy.yaml

# Use external NFS PVC instead of creating one
patchesStrategicMerge:
  - |-
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: hypersdk-exports
    $patch: delete

# Update deployment to use NFS PVC
patches:
  - target:
      kind: Deployment
      name: hypervisord
    patch: |-
      - op: replace
        path: /spec/template/spec/volumes/1
        value:
          name: exports
          persistentVolumeClaim:
            claimName: hypersdk-exports  # Use pre-created NFS PVC

replicas:
  - name: hypervisord
    count: 1  # Can scale up with ReadWriteMany

images:
  - name: hypersdk/hypervisord
    newName: ghcr.io/ssahani/hypersdk-hypervisord
    newTag: latest
```

### Deploy HyperSDK

```bash
# Create secrets first
cp deployments/kubernetes/base/secrets.yaml.example \
   deployments/kubernetes/overlays/production/secrets.yaml

vim deployments/kubernetes/overlays/production/secrets.yaml
# Configure your vSphere/cloud credentials

# Deploy
kubectl apply -f deployments/kubernetes/overlays/production/secrets.yaml
kubectl apply -k deployments/kubernetes/overlays/production

# Wait for deployment
kubectl rollout status deployment/hypervisord -n hypersdk

# Verify NFS mount
kubectl exec -n hypersdk deployment/hypervisord -- df -h /exports
kubectl exec -n hypersdk deployment/hypervisord -- touch /exports/test-write
kubectl exec -n hypersdk deployment/hypervisord -- ls -l /exports/test-write
```

## Part 4: Export VMs to NFS Storage

### Submit Export Job

```bash
# Port forward to access API
kubectl port-forward -n hypersdk svc/hypervisord 8080:8080 &

# Submit export job
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm": "/Datacenter/vm/test-vm",
    "output": "/exports/test-vm",
    "format": "ova",
    "compress": true
  }'

# Monitor progress
curl http://localhost:8080/jobs/query | jq
```

### Verify Export on NFS

```bash
# From Kubernetes pod
kubectl exec -n hypersdk deployment/hypervisord -- ls -lh /exports/test-vm/

# Expected output:
# -rw-r--r-- 1 hypersdk hypersdk 5.2G Jan 30 12:34 test-vm.ova
```

## Part 5: Access Exports from Native Linux (hyper2kvm)

### Mount NFS on Native Linux Host

```bash
# Install NFS client
sudo apt-get install -y nfs-common  # Debian/Ubuntu
sudo yum install -y nfs-utils       # RHEL/CentOS

# Create mount point
sudo mkdir -p /mnt/hypersdk-exports

# Mount NFS share
sudo mount -t nfs4 -o vers=4.1,rsize=1048576,wsize=1048576 \
  192.168.1.100:/exports/hypersdk /mnt/hypersdk-exports

# Verify mount
df -h /mnt/hypersdk-exports
ls -l /mnt/hypersdk-exports/test-vm/
```

### Make Mount Persistent

Add to `/etc/fstab`:

```bash
echo "192.168.1.100:/exports/hypersdk /mnt/hypersdk-exports nfs4 vers=4.1,rsize=1048576,wsize=1048576,hard,timeo=600,retrans=2 0 0" | sudo tee -a /etc/fstab

# Test fstab
sudo mount -a
```

### Use hyper2kvm on Native Linux

Assuming hyper2kvm is a tool that converts/imports VMs:

```bash
# Example: Convert OVA to KVM-compatible format
hyper2kvm convert \
  --input /mnt/hypersdk-exports/test-vm/test-vm.ova \
  --output /var/lib/libvirt/images/test-vm.qcow2 \
  --format qcow2

# Example: Import to KVM
hyper2kvm import \
  --source /mnt/hypersdk-exports/test-vm/test-vm.ova \
  --name test-vm \
  --memory 4096 \
  --vcpus 2
```

## Part 6: Access Exports from Kubernetes (hyper2kvm Pod)

### Deploy hyper2kvm in Kubernetes

Create a deployment that shares the same NFS PVC:

```yaml
# hyper2kvm-deployment.yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hyper2kvm
  namespace: hypersdk
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hyper2kvm
  template:
    metadata:
      labels:
        app: hyper2kvm
    spec:
      containers:
      - name: hyper2kvm
        image: your-registry/hyper2kvm:latest  # Your hyper2kvm container image
        command: ["/bin/bash", "-c", "sleep infinity"]  # Keep running for manual jobs

        volumeMounts:
        - name: exports
          mountPath: /imports  # Mount as /imports in hyper2kvm pod
          readOnly: true       # Read-only since we're only importing

        - name: kvm-images
          mountPath: /var/lib/libvirt/images  # Output directory for converted images

        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "8Gi"
            cpu: "4000m"

      volumes:
      - name: exports
        persistentVolumeClaim:
          claimName: hypersdk-exports  # Same NFS PVC as HyperSDK

      - name: kvm-images
        persistentVolumeClaim:
          claimName: kvm-images  # Separate PVC for KVM images

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: kvm-images
  namespace: hypersdk
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Ti
  storageClassName: standard
```

Deploy:
```bash
kubectl apply -f hyper2kvm-deployment.yaml

# Wait for pod
kubectl wait --for=condition=ready pod -l app=hyper2kvm -n hypersdk --timeout=300s
```

### Run hyper2kvm in Kubernetes

```bash
# List available exports
kubectl exec -n hypersdk deployment/hyper2kvm -- ls -l /imports/

# Convert VM
kubectl exec -n hypersdk deployment/hyper2kvm -- \
  hyper2kvm convert \
    --input /imports/test-vm/test-vm.ova \
    --output /var/lib/libvirt/images/test-vm.qcow2 \
    --format qcow2

# Or run as a Job for automation
```

### Automated Processing with Kubernetes Job

Create a Job to automatically process exports:

```yaml
# hyper2kvm-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: hyper2kvm-convert-test-vm
  namespace: hypersdk
spec:
  template:
    spec:
      containers:
      - name: hyper2kvm
        image: your-registry/hyper2kvm:latest
        command:
        - hyper2kvm
        - convert
        - --input
        - /imports/test-vm/test-vm.ova
        - --output
        - /var/lib/libvirt/images/test-vm.qcow2
        - --format
        - qcow2

        volumeMounts:
        - name: exports
          mountPath: /imports
          readOnly: true
        - name: kvm-images
          mountPath: /var/lib/libvirt/images

      volumes:
      - name: exports
        persistentVolumeClaim:
          claimName: hypersdk-exports
      - name: kvm-images
        persistentVolumeClaim:
          claimName: kvm-images

      restartPolicy: OnFailure
  backoffLimit: 3
```

Run:
```bash
kubectl apply -f hyper2kvm-job.yaml
kubectl logs -n hypersdk job/hyper2kvm-convert-test-vm -f
```

## Part 7: Complete Workflow Example

### End-to-End VM Migration

```bash
# 1. Export VM from vSphere using HyperSDK (in Kubernetes)
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm": "/Datacenter/vm/production-app",
    "output": "/exports/production-app",
    "format": "ova"
  }'

# 2. Wait for export to complete
while true; do
  STATUS=$(curl -s http://localhost:8080/jobs/query | jq -r '.jobs[0].status')
  echo "Export status: $STATUS"
  [[ "$STATUS" == "completed" ]] && break
  sleep 10
done

# 3a. Convert on Native Linux
ssh linux-host "hyper2kvm convert \
  --input /mnt/hypersdk-exports/production-app/production-app.ova \
  --output /var/lib/libvirt/images/production-app.qcow2"

# 3b. Or convert in Kubernetes
kubectl exec -n hypersdk deployment/hyper2kvm -- \
  hyper2kvm convert \
    --input /imports/production-app/production-app.ova \
    --output /var/lib/libvirt/images/production-app.qcow2

# 4. Import to KVM
ssh linux-host "hyper2kvm import \
  --source /var/lib/libvirt/images/production-app.qcow2 \
  --name production-app \
  --memory 8192 \
  --vcpus 4"
```

## Part 8: Monitoring and Observability

### Monitor NFS Performance

```bash
# On NFS server
nfsstat -s  # Server statistics
nfsstat -c  # Client statistics

# In Kubernetes
kubectl exec -n hypersdk deployment/hypervisord -- nfsstat
```

### Monitor Disk Usage

```bash
# Check NFS storage usage
kubectl exec -n hypersdk deployment/hypervisord -- df -h /exports

# Alert on disk usage
kubectl exec -n hypersdk deployment/hypervisord -- \
  bash -c 'USAGE=$(df /exports | tail -1 | awk "{print \$5}" | sed "s/%//"); \
           if [ $USAGE -gt 80 ]; then echo "WARNING: NFS storage $USAGE% full"; fi'
```

### Verify NFS Connectivity

```bash
# Test NFS mount from pod
kubectl run -it --rm nfs-test \
  --image=busybox \
  --overrides='
{
  "spec": {
    "containers": [{
      "name": "nfs-test",
      "image": "busybox",
      "command": ["sh"],
      "volumeMounts": [{
        "name": "nfs",
        "mountPath": "/mnt/nfs"
      }]
    }],
    "volumes": [{
      "name": "nfs",
      "persistentVolumeClaim": {
        "claimName": "hypersdk-exports"
      }
    }]
  }
}' \
  --namespace hypersdk

# Inside the pod:
ls -la /mnt/nfs
touch /mnt/nfs/test-file
rm /mnt/nfs/test-file
exit
```

## Part 9: High Availability and Performance

### NFS Server HA (Production)

For production, use a highly available NFS solution:

**Option 1: Cloud Provider NFS**
- AWS: EFS (Elastic File System)
- Azure: Azure Files
- GCP: Filestore

**Option 2: On-Premises HA NFS**
- GlusterFS
- CephFS
- NetApp/Dell EMC storage

### NFS Performance Tuning

Optimize NFS mount options:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: hypersdk-exports-nfs
spec:
  capacity:
    storage: 2Ti
  accessModes:
    - ReadWriteMany
  nfs:
    server: 192.168.1.100
    path: /exports/hypersdk
  mountOptions:
    - nfsvers=4.1
    - hard              # Retry indefinitely on errors
    - timeo=600         # 60 second timeout
    - retrans=2         # Retry 2 times
    - rsize=1048576     # 1MB read size
    - wsize=1048576     # 1MB write size
    - ac                # Attribute caching enabled
    - actimeo=30        # Cache attributes for 30 seconds
    - noatime           # Don't update access times
    - nodiratime        # Don't update directory access times
```

### Scale HyperSDK with ReadWriteMany

With NFS (ReadWriteMany), you can scale HyperSDK horizontally:

```bash
# Scale to 3 replicas
kubectl scale deployment/hypervisord --replicas=3 -n hypersdk

# All replicas share the same NFS storage
kubectl get pods -n hypersdk
```

## Part 10: Security Considerations

### NFS Security

```bash
# On NFS server - restrict access by IP
sudo vim /etc/exports
# Change from:
/exports/hypersdk *(rw,sync,no_subtree_check,no_root_squash)
# To:
/exports/hypersdk 192.168.1.0/24(rw,sync,no_subtree_check,no_root_squash)

sudo exportfs -ra
```

### Kubernetes RBAC for NFS Access

```yaml
# Restrict PVC access
apiVersion: v1
kind: ResourceQuota
metadata:
  name: pvc-quota
  namespace: hypersdk
spec:
  hard:
    persistentvolumeclaims: "5"
    requests.storage: "5Ti"
```

### Encryption in Transit

Use Kerberos for NFS security:

```bash
# Configure NFSv4 with Kerberos
# On NFS server:
sudo apt-get install -y krb5-user nfs-kernel-server

# Configure /etc/exports with sec=krb5p
/exports/hypersdk *(rw,sync,no_subtree_check,sec=krb5p)
```

## Troubleshooting

### NFS Mount Issues

```bash
# Check NFS server is accessible
kubectl run -it --rm nfs-debug --image=busybox --restart=Never -- \
  nc -zv 192.168.1.100 2049

# Check mount from worker node
ssh worker-node
sudo mount -t nfs4 192.168.1.100:/exports/hypersdk /mnt/test
sudo umount /mnt/test

# Check Kubernetes events
kubectl get events -n hypersdk --sort-by='.lastTimestamp'
```

### Permission Issues

```bash
# Fix NFS permissions
# On NFS server:
sudo chown -R 1000:1000 /exports/hypersdk
sudo chmod -R 755 /exports/hypersdk

# In Kubernetes pod:
kubectl exec -n hypersdk deployment/hypervisord -- id
# Should show: uid=1000(hypersdk) gid=1000(hypersdk)
```

### Performance Issues

```bash
# Check NFS server load
ssh nfs-server "top -b -n 1 | grep nfsd"

# Monitor network throughput
kubectl exec -n hypersdk deployment/hypervisord -- \
  dd if=/dev/zero of=/exports/test bs=1M count=1024 oflag=direct

# Increase NFS threads on server
ssh nfs-server "sudo sed -i 's/RPCNFSDCOUNT=8/RPCNFSDCOUNT=32/' /etc/default/nfs-kernel-server"
ssh nfs-server "sudo systemctl restart nfs-kernel-server"
```

## Complete Example Scripts

### deploy-with-nfs.sh

```bash
#!/bin/bash
set -e

NFS_SERVER="192.168.1.100"
NFS_PATH="/exports/hypersdk"

echo "Deploying HyperSDK with NFS shared storage..."

# Create namespace
kubectl create namespace hypersdk --dry-run=client -o yaml | kubectl apply -f -

# Create NFS PV and PVC
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolume
metadata:
  name: hypersdk-exports-nfs
spec:
  capacity:
    storage: 2Ti
  accessModes:
    - ReadWriteMany
  nfs:
    server: ${NFS_SERVER}
    path: ${NFS_PATH}
  mountOptions:
    - nfsvers=4.1
    - hard
    - timeo=600
    - rsize=1048576
    - wsize=1048576
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: hypersdk-exports
  namespace: hypersdk
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 2Ti
  volumeName: hypersdk-exports-nfs
  storageClassName: ""
EOF

# Wait for PVC to bind
kubectl wait --for=jsonpath='{.status.phase}'=Bound \
  pvc/hypersdk-exports -n hypersdk --timeout=60s

# Deploy HyperSDK
kubectl apply -k deployments/kubernetes/overlays/production

echo "Deployment complete!"
echo "Run: kubectl get pods -n hypersdk"
```

## Summary

This setup provides:

✅ **Shared Storage**: NFS-backed PVC accessible from multiple environments
✅ **Kubernetes HyperSDK**: Exports VMs to shared NFS storage
✅ **Native Linux Access**: Direct NFS mount for hyper2kvm on bare metal
✅ **Kubernetes Access**: hyper2kvm pods can read from same NFS PVC
✅ **Scalability**: ReadWriteMany allows multiple readers/writers
✅ **High Availability**: Can use enterprise NFS or cloud file services
✅ **Cross-Platform**: Works across Kubernetes and native Linux seamlessly

## Next Steps

- [Configuration Guide](configuration.md) - Advanced NFS tuning
- [Monitoring Guide](monitoring.md) - Monitor NFS performance
- [Production Deployment](production-deployment.md) - HA NFS setup
- [Kubernetes Guide](../../deployments/kubernetes/README.md) - Full Kubernetes reference
