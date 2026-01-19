# Migrating VMware Photon OS to KVM: A Technical Deep Dive

## Executive Summary

This guide provides a comprehensive technical walkthrough for migrating VMware Photon OS virtual machines to KVM/libvirt environments using the HyperSDK toolkit. Photon OS, VMware's purpose-built container host optimized for vSphere, presents unique challenges when migrating to KVM due to its tight integration with VMware technologies.

## Table of Contents

1. [Introduction](#introduction)
2. [Understanding Photon OS](#understanding-photon-os)
3. [Pre-Migration Assessment](#pre-migration-assessment)
4. [Migration Architecture](#migration-architecture)
5. [Technical Implementation](#technical-implementation)
6. [Post-Migration Optimization](#post-migration-optimization)
7. [Troubleshooting](#troubleshooting)
8. [Best Practices](#best-practices)

## Introduction

Photon OS is a minimal Linux container host optimized for VMware vSphere. When migrating Photon OS VMs from VMware to KVM, several technical challenges arise:

- **VMware-specific kernel modules** (vmxnet3, pvscsi, vmw_balloon)
- **VMware Tools dependencies** affecting system initialization
- **VMDK disk format** requiring conversion to QCOW2
- **Boot loader configuration** tied to VMware hardware
- **Network interface naming** changes from ens to eth naming

This guide addresses all these challenges with practical solutions.

## Understanding Photon OS

### What is Photon OS?

Photon OS is VMware's open-source, minimal Linux container host designed for:
- Cloud-native applications
- Containerized workloads (Docker, Kubernetes)
- vSphere integration
- Minimal attack surface (< 50MB base installation)
- Fast boot times
- Optimized for VMware ESXi

### Key Characteristics

**Package Manager**: tdnf (Tiny DNF)
**Init System**: systemd
**Default Kernel**: Linux 5.10+ (LTS)
**Container Runtime**: Docker, containerd
**Optimizations**: Tuned for vSphere, includes VMware tools

## Pre-Migration Assessment

### 1. Inventory Current State

```bash
# Check Photon OS version
cat /etc/photon-release

# List installed packages
tdnf list installed

# Check running containers
docker ps -a

# Review systemd services
systemctl list-units --type=service --state=running

# Check disk layout
lsblk
df -h

# Network configuration
ip addr show
networkctl status
```

### 2. Identify VMware Dependencies

```bash
# Check for VMware kernel modules
lsmod | grep -E '(vmw|vm_)'

# VMware Tools status
systemctl status vmware-tools
systemctl status vmtoolsd

# Check for VMware-specific packages
tdnf list installed | grep -i vmware
```

### 3. Document Critical Services

Create an inventory of:
- Running containers and their configurations
- Persistent volumes and mount points
- Network configurations (static IPs, VLANs)
- Custom systemd services
- Cron jobs and timers
- Application dependencies

## Migration Architecture

### HyperSDK Migration Flow

```
┌──────────────────────────────────────────────────────────────┐
│ Phase 1: Discovery & Export                                  │
│  ├─ Connect to vCenter                                       │
│  ├─ Identify Photon OS VMs                                   │
│  ├─ Export as OVF/OVA                                        │
│  └─ Download VMDK files                                      │
└──────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ Phase 2: Conversion                                          │
│  ├─ VMDK → QCOW2 conversion                                  │
│  ├─ Driver remapping (vmxnet3 → virtio-net)                  │
│  ├─ Storage controller change (pvscsi → virtio-scsi)         │
│  └─ Boot configuration adjustment                            │
└──────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ Phase 3: KVM Import & Configuration                          │
│  ├─ Create libvirt domain                                    │
│  ├─ Attach converted disks                                   │
│  ├─ Configure virtio devices                                 │
│  └─ Network setup (bridged/NAT)                              │
└──────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ Phase 4: First Boot & Remediation                            │
│  ├─ Boot into emergency/rescue mode                          │
│  ├─ Remove VMware modules from initramfs                     │
│  ├─ Update network interface names                           │
│  ├─ Reinstall bootloader                                     │
│  └─ Clean up VMware packages                                 │
└──────────────────────────────────────────────────────────────┘
                         │
                         ▼
┌──────────────────────────────────────────────────────────────┐
│ Phase 5: Validation & Optimization                           │
│  ├─ Verify all services running                              │
│  ├─ Container runtime functional                             │
│  ├─ Network connectivity confirmed                           │
│  ├─ Install KVM guest tools                                  │
│  └─ Performance tuning                                       │
└──────────────────────────────────────────────────────────────┘
```

## Technical Implementation

### Phase 1: VM Export with HyperSDK

#### Using HyperExport

```bash
# Export Photon OS VM from vSphere
./hyperexport \
  -vm "/datacenter/vm/photon-k8s-master" \
  -output /exports/photon-vms \
  -format ova \
  -compress \
  -verify

# Batch export multiple Photon OS VMs
cat > photon-vms.txt <<EOF
/datacenter/vm/photon-k8s-master
/datacenter/vm/photon-k8s-worker-01
/datacenter/vm/photon-k8s-worker-02
/datacenter/vm/photon-docker-host
EOF

./hyperexport -batch photon-vms.txt -format ova -compress
```

#### Using HyperCTL (Daemon Mode)

```bash
# Start daemon
./hypervisord --config /etc/hypervisord/config.yaml

# Submit export job
./hyperctl submit \
  -vm "/datacenter/vm/photon-k8s-master" \
  -output /exports/photon-vms

# Monitor progress
./hyperctl query -all
```

### Phase 2: Disk Conversion

#### VMDK to QCOW2 Conversion

```bash
# Extract OVA if needed
tar -xvf photon-k8s-master.ova

# Convert VMDK to QCOW2
qemu-img convert \
  -f vmdk \
  -O qcow2 \
  -o cluster_size=2M \
  photon-k8s-master-disk1.vmdk \
  photon-k8s-master.qcow2

# Verify conversion
qemu-img info photon-k8s-master.qcow2
qemu-img check photon-k8s-master.qcow2

# Optional: Compress QCOW2
qemu-img convert \
  -f qcow2 \
  -O qcow2 \
  -c \
  photon-k8s-master.qcow2 \
  photon-k8s-master-compressed.qcow2
```

#### Driver Injection (Advanced)

```bash
# Mount QCOW2 to inject virtio drivers
sudo modprobe nbd max_part=8
sudo qemu-nbd --connect=/dev/nbd0 photon-k8s-master.qcow2

# Mount root partition
sudo mkdir -p /mnt/photon
sudo mount /dev/nbd0p2 /mnt/photon

# Check current kernel modules
ls /mnt/photon/lib/modules/$(uname -r)/kernel/drivers/

# Ensure virtio modules present
ls /mnt/photon/lib/modules/*/kernel/drivers/virtio/
ls /mnt/photon/lib/modules/*/kernel/drivers/net/virtio_net.ko
ls /mnt/photon/lib/modules/*/kernel/drivers/block/virtio_blk.ko

# Unmount and disconnect
sudo umount /mnt/photon
sudo qemu-nbd --disconnect /dev/nbd0
```

### Phase 3: KVM Domain Creation

#### Libvirt XML Definition

```xml
<domain type='kvm'>
  <name>photon-k8s-master</name>
  <memory unit='GiB'>4</memory>
  <vcpu placement='static'>2</vcpu>

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
    <!-- Disk with virtio-scsi -->
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='writeback' io='threads'/>
      <source file='/var/lib/libvirt/images/photon-k8s-master.qcow2'/>
      <target dev='sda' bus='scsi'/>
      <address type='drive' controller='0' bus='0' target='0' unit='0'/>
    </disk>

    <!-- Virtio SCSI controller -->
    <controller type='scsi' index='0' model='virtio-scsi'>
      <address type='pci' domain='0x0000' bus='0x00' slot='0x04' function='0x0'/>
    </controller>

    <!-- Network with virtio -->
    <interface type='bridge'>
      <source bridge='br0'/>
      <model type='virtio'/>
      <address type='pci' domain='0x0000' bus='0x00' slot='0x03' function='0x0'/>
    </interface>

    <!-- Serial console for troubleshooting -->
    <serial type='pty'>
      <target type='isa-serial' port='0'>
        <model name='isa-serial'/>
      </target>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>

    <!-- VNC/Spice graphics -->
    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'>
      <listen type='address' address='0.0.0.0'/>
    </graphics>
  </devices>
</domain>
```

#### Create and Start Domain

```bash
# Define VM from XML
virsh define photon-k8s-master.xml

# Start VM
virsh start photon-k8s-master

# Connect to console (for troubleshooting boot issues)
virsh console photon-k8s-master
```

### Phase 4: First Boot Remediation

#### Boot into Rescue Mode

If VM fails to boot normally:

```bash
# Boot with emergency target
# Edit GRUB entry at boot, add to kernel line:
systemd.unit=emergency.target

# Or rescue mode
systemd.unit=rescue.target
```

#### Remove VMware Dependencies

```bash
# Once in rescue/emergency shell

# Remove VMware kernel modules from initramfs
dracut --force --omit-drivers "vmxnet3 vmw_pvscsi vmw_balloon vmwgfx" \
  /boot/initramfs-$(uname -r).img $(uname -r)

# Rebuild initramfs with virtio modules
dracut --force --add-drivers "virtio_blk virtio_scsi virtio_net virtio_pci" \
  /boot/initramfs-$(uname -r).img $(uname -r)

# Update GRUB configuration
grub2-mkconfig -o /boot/grub2/grub.cfg

# Reinstall bootloader (if needed)
grub2-install /dev/sda

# Remove VMware packages
tdnf remove -y open-vm-tools vmware-tools

# Clean up VMware systemd services
systemctl disable vmware-tools vmtoolsd
rm -f /etc/systemd/system/multi-user.target.wants/vmware-tools.service
systemctl daemon-reload
```

#### Network Interface Remediation

Photon OS network interfaces may change from `ens*` to `eth*`:

```bash
# Check current interface names
ip link show

# Update systemd network files
cd /etc/systemd/network/

# Rename interface files
mv 10-ens192.network 10-eth0.network

# Edit and update interface name
sed -i 's/ens192/eth0/g' 10-eth0.network

# Or create new configuration
cat > 10-eth0.network <<EOF
[Match]
Name=eth0

[Network]
DHCP=yes

[DHCP]
UseDNS=true
EOF

# Restart networking
systemctl restart systemd-networkd
networkctl reload
```

#### Static IP Configuration

```bash
cat > /etc/systemd/network/10-eth0.network <<EOF
[Match]
Name=eth0

[Network]
Address=192.168.1.100/24
Gateway=192.168.1.1
DNS=8.8.8.8
DNS=8.8.4.4

[Route]
Gateway=192.168.1.1
EOF

systemctl restart systemd-networkd
```

### Phase 5: Post-Migration Configuration

#### Install QEMU Guest Agent

```bash
# Install QEMU guest agent for better KVM integration
tdnf install -y qemu-guest-agent

# Enable and start service
systemctl enable qemu-guest-agent
systemctl start qemu-guest-agent

# Verify
systemctl status qemu-guest-agent
```

#### Optimize for KVM

```bash
# Install virtio drivers (if not already present)
tdnf install -y linux-drivers-virtio

# Update kernel parameters for KVM
cat >> /etc/default/grub <<EOF
GRUB_CMDLINE_LINUX="console=tty0 console=ttyS0,115200n8"
EOF

grub2-mkconfig -o /boot/grub2/grub.cfg
```

#### Container Runtime Verification

```bash
# Verify Docker is running
systemctl status docker

# Test container functionality
docker run --rm hello-world

# For Kubernetes nodes
systemctl status kubelet
kubectl get nodes
```

## Post-Migration Optimization

### Performance Tuning

```bash
# Enable virtio-balloon for memory management
# (Already configured in libvirt XML)

# Optimize disk I/O
# Set elevator scheduler for virtio devices
echo "none" > /sys/block/sda/queue/scheduler

# Make persistent
cat > /etc/udev/rules.d/60-scheduler.rules <<EOF
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/scheduler}="none"
EOF
```

### Monitoring and Logging

```bash
# Check system logs for errors
journalctl -xe
journalctl -u systemd-networkd
journalctl -u docker

# Monitor performance
vmstat 1
iostat -x 1
```

### Backup Configuration

```bash
# Create VM snapshot
virsh snapshot-create-as photon-k8s-master \
  "post-migration-snapshot" \
  "Snapshot after successful migration"

# List snapshots
virsh snapshot-list photon-k8s-master
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Boot Failure: "Kernel Panic - not syncing"

**Cause**: Missing virtio drivers in initramfs

**Solution**:
```bash
# Boot from live CD/rescue image
# Mount root partition
mount /dev/sda2 /mnt
mount --bind /dev /mnt/dev
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys
chroot /mnt

# Rebuild initramfs with virtio
dracut --force --add-drivers "virtio_blk virtio_scsi" \
  /boot/initramfs-$(uname -r).img $(uname -r)

# Exit and reboot
exit
reboot
```

#### 2. Network Not Working

**Cause**: Interface name mismatch

**Solution**:
```bash
# Check actual interface names
ip link show

# Update networkd configuration
cd /etc/systemd/network/
# Edit files to match actual interface names (eth0, eth1, etc.)

systemctl restart systemd-networkd
```

#### 3. Docker Fails to Start

**Cause**: Storage driver issues or missing dependencies

**Solution**:
```bash
# Check Docker logs
journalctl -u docker -n 50

# Reset Docker
systemctl stop docker
rm -rf /var/lib/docker
systemctl start docker

# Or reinstall
tdnf remove -y docker
tdnf install -y docker
systemctl enable --now docker
```

#### 4. Slow Disk Performance

**Cause**: Incorrect cache settings or SCSI controller

**Solution**:
```bash
# Update VM XML
virsh edit photon-k8s-master

# Change disk driver cache mode
<driver name='qemu' type='qcow2' cache='none' io='native'/>

# Or use writeback with discard
<driver name='qemu' type='qcow2' cache='writeback' io='threads' discard='unmap'/>

# Restart VM
virsh destroy photon-k8s-master
virsh start photon-k8s-master
```

## Best Practices

### 1. Pre-Migration Checklist

- [ ] Document current VM configuration
- [ ] Export container configurations
- [ ] Backup persistent data
- [ ] Note custom configurations
- [ ] Test migration on non-production VM first
- [ ] Plan maintenance window

### 2. During Migration

- [ ] Use OVA format with compression
- [ ] Verify checksums after export
- [ ] Keep original VMDK until migration confirmed
- [ ] Use virtio drivers for best performance
- [ ] Configure serial console for troubleshooting

### 3. Post-Migration

- [ ] Verify all services running
- [ ] Test container functionality
- [ ] Confirm network connectivity
- [ ] Update monitoring systems
- [ ] Document any configuration changes
- [ ] Create post-migration snapshot

### 4. Automation

For migrating multiple Photon OS VMs:

```bash
#!/bin/bash
# batch-photon-migration.sh

VMS_FILE="photon-vms.txt"
OUTPUT_DIR="/exports/photon-vms"
KVM_IMAGE_DIR="/var/lib/libvirt/images"

# Export all VMs
./hyperexport -batch "$VMS_FILE" -format ova -compress -output "$OUTPUT_DIR"

# Convert each VM
while read -r vm_path; do
  vm_name=$(basename "$vm_path")

  # Extract OVA
  cd "$OUTPUT_DIR/$vm_name" || continue
  tar -xvf "${vm_name}.ova"

  # Convert VMDK to QCOW2
  qemu-img convert -f vmdk -O qcow2 -o cluster_size=2M \
    "${vm_name}-disk1.vmdk" \
    "$KVM_IMAGE_DIR/${vm_name}.qcow2"

  # Create libvirt domain
  virt-install \
    --name "$vm_name" \
    --memory 4096 \
    --vcpus 2 \
    --disk path="$KVM_IMAGE_DIR/${vm_name}.qcow2",bus=scsi \
    --controller type=scsi,model=virtio-scsi \
    --network bridge=br0,model=virtio \
    --graphics vnc \
    --import \
    --noautoconsole

done < "$VMS_FILE"

echo "Migration complete. Review each VM and perform post-migration steps."
```

## Conclusion

Migrating Photon OS from VMware to KVM requires attention to driver compatibility, network configuration, and boot processes. The HyperSDK toolkit streamlines the export and conversion phases, while proper post-migration remediation ensures reliable operation on KVM.

Key takeaways:
- Always rebuild initramfs with virtio drivers
- Update network interface configurations
- Remove VMware-specific packages and services
- Use virtio devices for optimal performance
- Test thoroughly before decommissioning source VMs

## Additional Resources

- [Photon OS Documentation](https://vmware.github.io/photon/)
- [HyperSDK GitHub Repository](https://github.com/ssahani/hypersdk)
- [KVM/QEMU Documentation](https://www.qemu.org/documentation/)
- [Libvirt Domain XML Format](https://libvirt.org/formatdomain.html)

---

**Author**: HyperSDK Team
**Last Updated**: January 2026
**Version**: 1.0
