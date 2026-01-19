# Migration Workflows Guide

## Table of Contents

1. [Overview](#overview)
2. [Single VM Migration](#single-vm-migration)
3. [Batch Migration](#batch-migration)
4. [Kubernetes Cluster Migration](#kubernetes-cluster-migration)
5. [Docker Host Migration](#docker-host-migration)
6. [Database Server Migration](#database-server-migration)
7. [Web Application Stack Migration](#web-application-stack-migration)
8. [Multi-Cloud Migration](#multi-cloud-migration)
9. [Zero-Downtime Migration](#zero-downtime-migration)
10. [Disaster Recovery Migration](#disaster-recovery-migration)

## Overview

This guide provides step-by-step workflows for common migration scenarios using HyperSDK.

## Single VM Migration

### Basic Single VM Migration

```bash
# Step 1: Export from vSphere
./hyperexport -vm "/datacenter/vm/web-server-01" \
  -output /exports/web-server-01 \
  -format ova \
  -compress \
  -verify

# Step 2: Convert VMDK to QCOW2
qemu-img convert -f vmdk -O qcow2 \
  /exports/web-server-01/web-server-01-disk1.vmdk \
  /var/lib/libvirt/images/web-server-01.qcow2

# Step 3: Create KVM domain
virt-install \
  --name web-server-01 \
  --memory 4096 \
  --vcpus 2 \
  --disk path=/var/lib/libvirt/images/web-server-01.qcow2,bus=scsi \
  --network bridge=br0,model=virtio \
  --graphics vnc \
  --import

# Step 4: Verify and configure
virsh start web-server-01
virsh console web-server-01
```

### With Pre-shutdown

```bash
# Step 1: Graceful shutdown VM in vSphere
./hyperexport -vm "/datacenter/vm/web-server-01" \
  -power-off \
  -output /exports/web-server-01

# Step 2-4: Same as above
```

## Batch Migration

### Migrate Multiple VMs

```bash
# Create VM list
cat > production-vms.txt <<EOF
/datacenter/vm/web-01
/datacenter/vm/web-02
/datacenter/vm/db-primary
/datacenter/vm/db-replica
/datacenter/vm/cache-redis
EOF

# Export all VMs
./hyperexport -batch production-vms.txt \
  -format ova \
  -compress \
  -output /exports/batch-migration

# Convert all to QCOW2
for vm in web-01 web-02 db-primary db-replica cache-redis; do
  qemu-img convert -f vmdk -O qcow2 \
    /exports/batch-migration/$vm/${vm}-disk1.vmdk \
    /var/lib/libvirt/images/${vm}.qcow2
done

# Create all KVM domains
for vm in web-01 web-02 db-primary db-replica cache-redis; do
  virt-install \
    --name $vm \
    --memory 8192 \
    --vcpus 4 \
    --disk path=/var/lib/libvirt/images/${vm}.qcow2,bus=scsi \
    --network bridge=br0,model=virtio \
    --graphics vnc \
    --import \
    --noautoconsole
done
```

### Automated Batch Migration Script

```bash
#!/bin/bash
# batch-migrate.sh

VMS_FILE="$1"
EXPORT_DIR="/exports/batch-$(date +%Y%m%d)"
KVM_IMAGE_DIR="/var/lib/libvirt/images"

# Export phase
echo "Starting batch export..."
./hyperexport -batch "$VMS_FILE" \
  -format ova \
  -compress \
  -verify \
  -output "$EXPORT_DIR"

# Conversion phase
echo "Converting VMDKs to QCOW2..."
while read -r vm_path; do
  vm_name=$(basename "$vm_path")

  vmdk_file=$(find "$EXPORT_DIR/$vm_name" -name "*.vmdk" | head -1)
  qcow2_file="$KVM_IMAGE_DIR/${vm_name}.qcow2"

  qemu-img convert -f vmdk -O qcow2 -p "$vmdk_file" "$qcow2_file"

done < "$VMS_FILE"

# Import phase
echo "Creating KVM domains..."
while read -r vm_path; do
  vm_name=$(basename "$vm_path")

  virt-install \
    --name "$vm_name" \
    --memory 4096 \
    --vcpus 2 \
    --disk path="$KVM_IMAGE_DIR/${vm_name}.qcow2",bus=scsi \
    --network bridge=br0,model=virtio \
    --graphics vnc \
    --import \
    --noautoconsole

  echo "Created: $vm_name"

done < "$VMS_FILE"

echo "Batch migration complete!"
```

## Kubernetes Cluster Migration

### Migrate Complete K8s Cluster

```bash
# Step 1: Identify cluster nodes
cat > k8s-cluster.txt <<EOF
/datacenter/vm/k8s-master-01
/datacenter/vm/k8s-worker-01
/datacenter/vm/k8s-worker-02
/datacenter/vm/k8s-worker-03
EOF

# Step 2: Drain and cordon nodes (on master)
kubectl drain k8s-worker-01 --ignore-daemonsets --delete-emptydir-data
kubectl drain k8s-worker-02 --ignore-daemonsets --delete-emptydir-data
kubectl drain k8s-worker-03 --ignore-daemonsets --delete-emptydir-data

# Step 3: Export VMs
./hyperexport -batch k8s-cluster.txt \
  -power-off \
  -format ova \
  -compress \
  -output /exports/k8s-cluster

# Step 4: Convert to QCOW2
for node in k8s-master-01 k8s-worker-01 k8s-worker-02 k8s-worker-03; do
  qemu-img convert -f vmdk -O qcow2 \
    /exports/k8s-cluster/$node/${node}-disk1.vmdk \
    /var/lib/libvirt/images/${node}.qcow2
done

# Step 5: Import to KVM
for node in k8s-master-01 k8s-worker-01 k8s-worker-02 k8s-worker-03; do
  virt-install \
    --name $node \
    --memory 8192 \
    --vcpus 4 \
    --disk path=/var/lib/libvirt/images/${node}.qcow2,bus=scsi \
    --network bridge=br0,model=virtio \
    --graphics vnc \
    --import \
    --noautoconsole
done

# Step 6: Start nodes in order
virsh start k8s-master-01
sleep 60  # Wait for master
virsh start k8s-worker-01
virsh start k8s-worker-02
virsh start k8s-worker-03

# Step 7: Uncordon nodes (on master)
kubectl uncordon k8s-worker-01
kubectl uncordon k8s-worker-02
kubectl uncordon k8s-worker-03

# Step 8: Verify cluster
kubectl get nodes
kubectl get pods --all-namespaces
```

## Docker Host Migration

### Migrate Docker Host with Containers

```bash
# Step 1: Export Docker volumes and configs (on source)
ssh docker-host "docker stop \$(docker ps -q)"
ssh docker-host "tar -czf /tmp/docker-volumes.tar.gz /var/lib/docker/volumes"
ssh docker-host "tar -czf /tmp/docker-data.tar.gz /var/lib/docker"
scp docker-host:/tmp/docker-*.tar.gz /exports/docker-host/

# Step 2: Export VM
./hyperexport -vm "/datacenter/vm/docker-host" \
  -power-off \
  -format ova \
  -compress \
  -output /exports/docker-host

# Step 3: Convert and import
qemu-img convert -f vmdk -O qcow2 \
  /exports/docker-host/docker-host-disk1.vmdk \
  /var/lib/libvirt/images/docker-host.qcow2

virt-install \
  --name docker-host \
  --memory 16384 \
  --vcpus 8 \
  --disk path=/var/lib/libvirt/images/docker-host.qcow2,bus=scsi \
  --network bridge=br0,model=virtio \
  --graphics vnc \
  --import

# Step 4: Restore Docker data (on migrated host)
virsh start docker-host
ssh docker-host-new "systemctl stop docker"
scp /exports/docker-host/docker-volumes.tar.gz docker-host-new:/tmp/
ssh docker-host-new "tar -xzf /tmp/docker-volumes.tar.gz -C /"
ssh docker-host-new "systemctl start docker"
ssh docker-host-new "docker ps -a"
```

## Database Server Migration

### PostgreSQL Migration

```bash
# Step 1: Backup database
ssh pg-server "sudo -u postgres pg_dumpall > /tmp/pg-backup.sql"
scp pg-server:/tmp/pg-backup.sql /exports/pg-server/

# Step 2: Export VM
./hyperexport -vm "/datacenter/vm/pg-server" \
  -power-off \
  -format ova \
  -verify \
  -output /exports/pg-server

# Step 3: Convert and import
qemu-img convert -f vmdk -O qcow2 \
  /exports/pg-server/pg-server-disk1.vmdk \
  /var/lib/libvirt/images/pg-server.qcow2

virt-install \
  --name pg-server \
  --memory 32768 \
  --vcpus 8 \
  --disk path=/var/lib/libvirt/images/pg-server.qcow2,bus=scsi,cache=none,io=native \
  --network bridge=br0,model=virtio \
  --graphics vnc \
  --import

# Step 4: Start and verify
virsh start pg-server
ssh pg-server-new "systemctl status postgresql"
ssh pg-server-new "sudo -u postgres psql -c '\l'"

# Step 5: Verify data integrity
ssh pg-server-new "sudo -u postgres pg_dumpall > /tmp/pg-verify.sql"
diff /exports/pg-server/pg-backup.sql <(ssh pg-server-new "cat /tmp/pg-verify.sql")
```

### MySQL/MariaDB Migration

```bash
# Step 1: Backup database
ssh mysql-server "mysqldump --all-databases > /tmp/mysql-backup.sql"
scp mysql-server:/tmp/mysql-backup.sql /exports/mysql-server/

# Step 2: Stop replication (if applicable)
ssh mysql-server "mysql -e 'STOP SLAVE;'"
ssh mysql-server "mysql -e 'SHOW MASTER STATUS;'" > /exports/mysql-server/master-status.txt

# Step 3: Export VM
./hyperexport -vm "/datacenter/vm/mysql-server" \
  -power-off \
  -format ova \
  -output /exports/mysql-server

# Step 4-5: Convert, import, verify (similar to PostgreSQL)
```

## Web Application Stack Migration

### LAMP Stack Migration

```bash
# Step 1: Export all stack components
cat > lamp-stack.txt <<EOF
/datacenter/vm/web-01
/datacenter/vm/web-02
/datacenter/vm/db-master
/datacenter/vm/memcached
EOF

# Step 2: Backup application data
for host in web-01 web-02; do
  ssh $host "tar -czf /tmp/www-backup.tar.gz /var/www"
  scp $host:/tmp/www-backup.tar.gz /exports/lamp-stack/$host/
done

# Step 3: Export VMs
./hyperexport -batch lamp-stack.txt \
  -power-off \
  -format ova \
  -compress \
  -output /exports/lamp-stack

# Step 4: Convert all
for vm in web-01 web-02 db-master memcached; do
  qemu-img convert -f vmdk -O qcow2 \
    /exports/lamp-stack/$vm/${vm}-disk1.vmdk \
    /var/lib/libvirt/images/${vm}.qcow2
done

# Step 5: Import in correct order (DB first, then cache, then web)
virsh define db-master.xml && virsh start db-master
sleep 30
virsh define memcached.xml && virsh start memcached
sleep 15
virsh define web-01.xml && virsh start web-01
virsh define web-02.xml && virsh start web-02

# Step 6: Verify stack
curl http://web-01-new/health
curl http://web-02-new/health
```

## Multi-Cloud Migration

### vSphere to AWS

```bash
# Step 1: Export from vSphere
./hyperexport -vm "/datacenter/vm/app-server" \
  -format ova \
  -compress \
  -output /exports/aws-migration

# Step 2: Convert to RAW format (for AWS import)
qemu-img convert -f vmdk -O raw \
  /exports/aws-migration/app-server/app-server-disk1.vmdk \
  /exports/aws-migration/app-server.raw

# Step 3: Upload to S3
aws s3 cp /exports/aws-migration/app-server.raw \
  s3://my-vm-imports/app-server.raw

# Step 4: Import to EC2 as AMI
aws ec2 import-image \
  --description "Migrated app-server" \
  --disk-containers Format=raw,UserBucket="{S3Bucket=my-vm-imports,S3Key=app-server.raw}"

# Step 5: Monitor import
aws ec2 describe-import-image-tasks --import-task-ids import-ami-xxxxxxxxx

# Step 6: Launch instance from AMI
aws ec2 run-instances \
  --image-id ami-xxxxxxxxx \
  --instance-type t3.large \
  --key-name my-key
```

### vSphere to Azure

```bash
# Step 1: Export from vSphere
./hyperexport -vm "/datacenter/vm/app-server" \
  -format ova \
  -compress \
  -output /exports/azure-migration

# Step 2: Convert to VHD
qemu-img convert -f vmdk -O vpc \
  /exports/azure-migration/app-server/app-server-disk1.vmdk \
  /exports/azure-migration/app-server.vhd

# Step 3: Upload to Azure Blob Storage
az storage blob upload \
  --account-name mystorageaccount \
  --container-name vhds \
  --name app-server.vhd \
  --file /exports/azure-migration/app-server.vhd

# Step 4: Create managed disk from VHD
az disk create \
  --resource-group my-rg \
  --name app-server-disk \
  --source https://mystorageaccount.blob.core.windows.net/vhds/app-server.vhd

# Step 5: Create VM from disk
az vm create \
  --resource-group my-rg \
  --name app-server \
  --attach-os-disk app-server-disk \
  --os-type Linux
```

## Zero-Downtime Migration

### Using Replication

```bash
# Step 1: Setup replication (for databases)
# Configure master-slave replication from source to target

# Step 2: Export and migrate replica
./hyperexport -vm "/datacenter/vm/db-replica" \
  -format ova \
  -output /exports/zdt-migration

# Convert and import
qemu-img convert -f vmdk -O qcow2 \
  /exports/zdt-migration/db-replica/db-replica-disk1.vmdk \
  /var/lib/libvirt/images/db-replica.qcow2

virt-install \
  --name db-replica \
  --memory 16384 \
  --vcpus 4 \
  --disk path=/var/lib/libvirt/images/db-replica.qcow2 \
  --network bridge=br0,model=virtio \
  --import

# Step 3: Verify replication lag
ssh db-replica-new "mysql -e 'SHOW SLAVE STATUS\G' | grep Seconds_Behind_Master"

# Step 4: Cutover during maintenance window
# - Stop writes to old master
# - Promote replica to master
# - Update application connection strings
# - Decommission old master
```

### Using Storage Replication

```bash
# Step 1: Setup storage-level replication (DRBD, GlusterFS, etc.)

# Step 2: Sync data to target
rsync -avz --progress /var/lib/libvirt/images/ \
  target-host:/var/lib/libvirt/images/

# Step 3: Final sync during cutover
# - Pause source VM
# - Final rsync
# - Start target VM
```

## Disaster Recovery Migration

### Emergency Migration

```bash
# Quick migration with minimal downtime

# Step 1: Fast export (no compression for speed)
./hyperexport -vm "/datacenter/vm/critical-app" \
  -format ovf \
  -parallel 8 \
  -output /exports/dr

# Step 2: Parallel conversion
qemu-img convert -f vmdk -O qcow2 -m 4 \
  /exports/dr/critical-app/critical-app-disk1.vmdk \
  /var/lib/libvirt/images/critical-app.qcow2

# Step 3: Immediate import and start
virt-install \
  --name critical-app \
  --memory 8192 \
  --vcpus 4 \
  --disk path=/var/lib/libvirt/images/critical-app.qcow2 \
  --network bridge=br0,model=virtio \
  --import

# Step 4: Update DNS/load balancer
# Point traffic to new instance
```

## Best Practices

### Pre-Migration Checklist

- [ ] Document current configuration
- [ ] Backup all data
- [ ] Test migration process on non-production VM
- [ ] Plan network configuration
- [ ] Schedule maintenance window
- [ ] Notify stakeholders
- [ ] Prepare rollback plan

### During Migration

- [ ] Monitor export progress
- [ ] Verify checksums
- [ ] Test converted images
- [ ] Keep original VMs until verification
- [ ] Document any issues

### Post-Migration

- [ ] Verify all services running
- [ ] Test application functionality
- [ ] Update documentation
- [ ] Update monitoring systems
- [ ] Update backup systems
- [ ] Decommission source VMs after verification period

## Troubleshooting

See [Troubleshooting Guide](troubleshooting-guide.md) for common issues and solutions.

## See Also

- [Installation Guide](installation-guide.md)
- [Configuration Reference](configuration-reference.md)
- [Photon OS Migration Guide](photon-os-migration-technical-guide.md)
- [API Reference](API_ENDPOINTS.md)
