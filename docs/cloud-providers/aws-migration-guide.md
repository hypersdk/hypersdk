# AWS EC2 to KVM Migration Guide

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Export from AWS](#export-from-aws)
4. [Convert to KVM Format](#convert-to-kvm-format)
5. [Import to KVM](#import-to-kvm)
6. [Post-Migration Configuration](#post-migration-configuration)
7. [Troubleshooting](#troubleshooting)

## Overview

This guide covers migrating EC2 instances from AWS to KVM using HyperSDK.

### Challenges

- **Xen/Nitro drivers** need replacement with virtio
- **Instance metadata service** (169.254.169.254) not available
- **EBS volumes** require format conversion
- **Elastic Network Adapter (ENA)** driver changes
- **Cloud-init** configuration updates

## Prerequisites

### AWS Setup

```bash
# Install AWS CLI
pip install awscli

# Configure credentials
aws configure
# Or export credentials
export AWS_ACCESS_KEY_ID="AKIA..."
export AWS_SECRET_ACCESS_KEY="..."
export AWS_DEFAULT_REGION="us-east-1"

# Verify access
aws sts get-caller-identity
aws ec2 describe-instances --max-results 1
```

### Required Permissions

IAM policy for EC2 export:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateInstanceExportTask",
        "ec2:DescribeInstanceExportTasks",
        "ec2:CancelExportTask",
        "ec2:DescribeInstances",
        "ec2:CreateSnapshot",
        "ec2:DescribeSnapshots",
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:GetBucketLocation"
      ],
      "Resource": "*"
    }
  ]
}
```

### S3 Bucket

```bash
# Create S3 bucket for exports
aws s3 mb s3://my-vm-exports --region us-east-1

# Configure lifecycle policy (optional)
cat > lifecycle.json <<EOF
{
  "Rules": [
    {
      "Id": "DeleteOldExports",
      "Status": "Enabled",
      "Prefix": "exports/",
      "Expiration": {
        "Days": 7
      }
    }
  ]
}
EOF

aws s3api put-bucket-lifecycle-configuration \
  --bucket my-vm-exports \
  --lifecycle-configuration file://lifecycle.json
```

## Export from AWS

### Method 1: Using HyperSDK (Future Feature)

```bash
# Configure AWS provider
cat > /etc/hypervisord/config.yaml <<EOF
aws:
  region: "us-east-1"
  access_key_id: "${AWS_ACCESS_KEY_ID}"
  secret_access_key: "${AWS_SECRET_ACCESS_KEY}"
  s3_bucket: "my-vm-exports"
  export_format: "vmdk"
EOF

# Export EC2 instance (future feature)
./hyperexport -provider aws \
  -vm "i-1234567890abcdef0" \
  -output /exports/aws-migration
```

### Method 2: Manual Export via AWS CLI

#### Stop Instance (Recommended)

```bash
# Stop instance for consistent export
aws ec2 stop-instances --instance-ids i-1234567890abcdef0

# Wait for stopped state
aws ec2 wait instance-stopped --instance-ids i-1234567890abcdef0
```

#### Create AMI

```bash
# Create AMI from instance
aws ec2 create-image \
  --instance-id i-1234567890abcdef0 \
  --name "migration-$(date +%Y%m%d-%H%M%S)" \
  --description "Migration to KVM" \
  --no-reboot

# Get AMI ID from output
AMI_ID="ami-0abcdef1234567890"

# Wait for AMI to be available
aws ec2 wait image-available --image-ids $AMI_ID
```

#### Export to S3

```bash
# Create export task
EXPORT_TASK=$(aws ec2 create-instance-export-task \
  --instance-id i-1234567890abcdef0 \
  --target-environment vmware \
  --export-to-s3-task file://export-task.json \
  --query 'ExportTask.ExportTaskId' \
  --output text)

# export-task.json
cat > export-task.json <<EOF
{
  "DiskImageFormat": "VMDK",
  "S3Bucket": "my-vm-exports",
  "S3Prefix": "exports/"
}
EOF

# Monitor export progress
watch -n 30 "aws ec2 describe-export-tasks \
  --export-task-ids $EXPORT_TASK \
  --query 'ExportTasks[0].State'"

# Or script to wait
while true; do
  STATUS=$(aws ec2 describe-export-tasks \
    --export-task-ids $EXPORT_TASK \
    --query 'ExportTasks[0].State' \
    --output text)

  echo "Export status: $STATUS"

  if [ "$STATUS" = "completed" ]; then
    echo "Export completed!"
    break
  elif [ "$STATUS" = "cancelled" ] || [ "$STATUS" = "cancelling" ]; then
    echo "Export cancelled!"
    exit 1
  fi

  sleep 60
done
```

#### Download from S3

```bash
# Find exported file
aws s3 ls s3://my-vm-exports/exports/

# Download VMDK
aws s3 cp s3://my-vm-exports/exports/export-i-1234567890abcdef0.vmdk \
  /exports/aws-instance.vmdk

# Verify download
ls -lh /exports/aws-instance.vmdk
md5sum /exports/aws-instance.vmdk
```

### Method 3: EBS Snapshot Export

```bash
# Get root volume ID
VOLUME_ID=$(aws ec2 describe-instances \
  --instance-ids i-1234567890abcdef0 \
  --query 'Reservations[0].Instances[0].BlockDeviceMappings[0].Ebs.VolumeId' \
  --output text)

# Create snapshot
SNAPSHOT_ID=$(aws ec2 create-snapshot \
  --volume-id $VOLUME_ID \
  --description "Migration snapshot" \
  --query 'SnapshotId' \
  --output text)

# Wait for snapshot completion
aws ec2 wait snapshot-completed --snapshot-ids $SNAPSHOT_ID

# Export snapshot (requires additional setup)
# See: https://docs.aws.amazon.com/vm-import/latest/userguide/vmexport.html
```

## Convert to KVM Format

### VMDK to QCOW2 Conversion

```bash
# Convert to QCOW2
qemu-img convert -f vmdk -O qcow2 \
  -o cluster_size=2M \
  /exports/aws-instance.vmdk \
  /var/lib/libvirt/images/aws-instance.qcow2

# Verify conversion
qemu-img info /var/lib/libvirt/images/aws-instance.qcow2

# Check for errors
qemu-img check /var/lib/libvirt/images/aws-instance.qcow2

# Optional: Compress
qemu-img convert -f qcow2 -O qcow2 -c \
  /var/lib/libvirt/images/aws-instance.qcow2 \
  /var/lib/libvirt/images/aws-instance-compressed.qcow2
```

### RAW to QCOW2 (Alternative)

```bash
# If exported as RAW
qemu-img convert -f raw -O qcow2 \
  /exports/aws-instance.raw \
  /var/lib/libvirt/images/aws-instance.qcow2
```

## Import to KVM

### Create Libvirt Domain

```xml
<!-- aws-instance.xml -->
<domain type='kvm'>
  <name>aws-instance</name>
  <memory unit='GiB'>8</memory>
  <vcpu placement='static'>4</vcpu>

  <os>
    <type arch='x86_64' machine='pc-q35-6.2'>hvm</type>
    <boot dev='hd'/>
  </os>

  <features>
    <acpi/>
    <apic/>
  </features>

  <cpu mode='host-passthrough'/>

  <devices>
    <!-- Disk -->
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='none' io='native'/>
      <source file='/var/lib/libvirt/images/aws-instance.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>

    <!-- Network -->
    <interface type='bridge'>
      <source bridge='br0'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x00' slot='0x03' function='0x0'/>
    </interface>

    <!-- Serial console -->
    <serial type='pty'>
      <target type='isa-serial' port='0'/>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>

    <!-- Graphics -->
    <graphics type='vnc' port='-1' autoport='yes'/>
  </devices>
</domain>
```

### Define and Start VM

```bash
# Define VM
virsh define aws-instance.xml

# Start VM
virsh start aws-instance

# Connect to console
virsh console aws-instance
```

## Post-Migration Configuration

### Boot into Rescue Mode

If VM doesn't boot normally:

```bash
# Boot to emergency target
# Edit GRUB at boot, add to kernel line:
systemd.unit=emergency.target
```

### Remove AWS-Specific Drivers

```bash
# Remove ENA driver
modprobe -r ena

# Remove Xen drivers
modprobe -r xen_netfront
modprobe -r xen_blkfront

# Blacklist AWS drivers
cat > /etc/modprobe.d/blacklist-aws.conf <<EOF
blacklist ena
blacklist xen_netfront
blacklist xen_blkfront
EOF
```

### Install Virtio Drivers

```bash
# Amazon Linux 2
sudo yum install -y kernel-modules-extra

# Ubuntu/Debian
sudo apt-get install -y linux-image-extra-virtual

# Rebuild initramfs with virtio
sudo dracut --force --add-drivers "virtio_blk virtio_net virtio_pci virtio_scsi" \
  /boot/initramfs-$(uname -r).img $(uname -r)

# Or on Debian/Ubuntu
sudo update-initramfs -u
```

### Network Configuration

#### Amazon Linux 2

```bash
# Update network interface
cat > /etc/sysconfig/network-scripts/ifcfg-eth0 <<EOF
DEVICE=eth0
BOOTPROTO=dhcp
ONBOOT=yes
TYPE=Ethernet
USERCTL=no
EOF

# Restart network
sudo systemctl restart network
```

#### Ubuntu

```bash
# Netplan configuration
cat > /etc/netplan/01-netcfg.yaml <<EOF
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: true
EOF

sudo netplan apply
```

### Disable Cloud-Init (Optional)

```bash
# Disable cloud-init
sudo touch /etc/cloud/cloud-init.disabled

# Or configure for non-AWS
cat > /etc/cloud/cloud.cfg.d/99-local.cfg <<EOF
datasource_list: [ NoCloud, None ]
EOF

# Disable metadata service lookups
sudo systemctl disable cloud-init
sudo systemctl disable cloud-config
sudo systemctl disable cloud-final
```

### Install QEMU Guest Agent

```bash
# Amazon Linux 2
sudo yum install -y qemu-guest-agent
sudo systemctl enable qemu-guest-agent
sudo systemctl start qemu-guest-agent

# Ubuntu/Debian
sudo apt-get install -y qemu-guest-agent
sudo systemctl enable qemu-guest-agent
sudo systemctl start qemu-guest-agent
```

### Update GRUB

```bash
# Rebuild GRUB configuration
sudo grub2-mkconfig -o /boot/grub2/grub.cfg

# Reinstall bootloader if needed
sudo grub2-install /dev/vda
```

## Troubleshooting

### Boot Failure: Kernel Panic

```bash
# Boot from rescue image
# Mount root partition
mount /dev/vda1 /mnt
chroot /mnt

# Rebuild initramfs
dracut --force --add-drivers "virtio_blk virtio_scsi" \
  /boot/initramfs-$(uname -r).img $(uname -r)

exit
reboot
```

### Network Not Working

```bash
# Check interface name
ip link show

# Update configuration
sudo vi /etc/sysconfig/network-scripts/ifcfg-eth0
# Change DEVICE= to match actual interface

sudo systemctl restart network
```

### Metadata Service Dependency

Some applications may fail looking for metadata service:

```bash
# Block metadata service requests
sudo iptables -A OUTPUT -d 169.254.169.254 -j REJECT

# Or redirect to local service (advanced)
# Setup local metadata service mock
```

### Slow Boot

```bash
# Disable cloud-init to speed up boot
sudo systemctl disable cloud-init-local.service
sudo systemctl disable cloud-init.service
sudo systemctl disable cloud-config.service
sudo systemctl disable cloud-final.service
```

## Performance Optimization

### Disk I/O

```xml
<!-- Use virtio-scsi with direct I/O -->
<disk type='file' device='disk'>
  <driver name='qemu' type='qcow2' cache='none' io='native' discard='unmap'/>
  <target dev='sda' bus='scsi'/>
</disk>

<controller type='scsi' index='0' model='virtio-scsi'>
  <driver queues='4'/>
</controller>
```

### Network Performance

```xml
<!-- Enable multiqueue -->
<interface type='bridge'>
  <model type='virtio'/>
  <driver name='vhost' queues='4'/>
</interface>
```

```bash
# Inside VM
sudo ethtool -L eth0 combined 4
```

## See Also

- [Migration Workflows](migration-workflows.md)
- [Troubleshooting Guide](troubleshooting-guide.md)
- [Performance Tuning](performance-tuning.md)
- [AWS VM Import/Export Documentation](https://docs.aws.amazon.com/vm-import/latest/userguide/)
