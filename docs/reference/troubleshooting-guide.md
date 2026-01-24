# HyperSDK Troubleshooting Guide

## Table of Contents

1. [Connection Issues](#connection-issues)
2. [Export Failures](#export-failures)
3. [Conversion Problems](#conversion-problems)
4. [Import Issues](#import-issues)
5. [Boot Failures](#boot-failures)
6. [Network Issues](#network-issues)
7. [Performance Problems](#performance-problems)
8. [Daemon Issues](#daemon-issues)
9. [API Errors](#api-errors)
10. [WebSocket Problems](#websocket-problems)

## Connection Issues

### vSphere Connection Failed

**Error**: `Failed to connect to vSphere`

**Possible Causes**:
1. Incorrect URL, username, or password
2. Network connectivity issues
3. SSL certificate verification failure
4. Firewall blocking connection

**Solutions**:

```bash
# Test network connectivity
ping vcenter.example.com
telnet vcenter.example.com 443

# Test with curl
curl -k https://vcenter.example.com/sdk

# Verify credentials manually
govc about -u "https://user:pass@vcenter.example.com/sdk" -k

# Use insecure mode for self-signed certs
export GOVC_INSECURE=1

# Or in config.yaml
vsphere:
  insecure: true
```

### AWS Connection Failed

**Error**: `Unable to locate credentials`

**Solutions**:

```bash
# Verify credentials
aws sts get-caller-identity

# Set credentials
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
export AWS_REGION="us-east-1"

# Or use AWS CLI configure
aws configure

# Test EC2 access
aws ec2 describe-instances --max-results 1
```

### Azure Connection Failed

**Error**: `Failed to authenticate with Azure`

**Solutions**:

```bash
# Login with Azure CLI
az login

# Verify subscription
az account show

# Set environment variables
export AZURE_SUBSCRIPTION_ID="..."
export AZURE_TENANT_ID="..."
export AZURE_CLIENT_ID="..."
export AZURE_CLIENT_SECRET="..."

# Test Azure access
az vm list --output table
```

## Export Failures

### VM Not Found

**Error**: `VM not found: /datacenter/vm/my-vm`

**Solutions**:

```bash
# List all VMs to find correct path
govc find / -type m

# Search for VM by name
govc find / -type m -name "my-vm"

# Use exact path from vCenter
# Format: /datacenter/vm/folder/vm-name
```

### Disk Space Insufficient

**Error**: `No space left on device`

**Solutions**:

```bash
# Check available space
df -h /exports

# Clean up old exports
rm -rf /exports/old-*

# Use different output directory
./hyperexport -vm myvm -output /mnt/large-disk/exports

# Enable compression to save space
./hyperexport -vm myvm -compress
```

### Export Timeout

**Error**: `Export operation timed out`

**Solutions**:

```bash
# Increase timeout in config
daemon:
  job_timeout: "48h"  # Default is 24h

# Or use environment variable
export JOB_TIMEOUT="48h"

# For large VMs, use more parallel downloads
./hyperexport -vm myvm -parallel 8
```

### Permission Denied

**Error**: `Permission denied: /exports/myvm`

**Solutions**:

```bash
# Check directory permissions
ls -ld /exports

# Create with correct permissions
sudo mkdir -p /exports
sudo chown $USER:$USER /exports
sudo chmod 755 /exports

# Or run as root (not recommended)
sudo ./hyperexport -vm myvm
```

## Conversion Problems

### VMDK Conversion Failed

**Error**: `qemu-img: Could not open 'vm.vmdk'`

**Solutions**:

```bash
# Check VMDK file integrity
file vm-disk1.vmdk
qemu-img info vm-disk1.vmdk

# Try different VMDK format
qemu-img convert -f vmdk -O qcow2 vm-disk1.vmdk vm.qcow2

# For split VMDK files, combine first
cat vm-disk1.vmdk vm-disk2.vmdk > vm-combined.vmdk
qemu-img convert -f vmdk -O qcow2 vm-combined.vmdk vm.qcow2

# Extract from OVA first
tar -xvf vm.ova
qemu-img convert -f vmdk -O qcow2 vm-disk1.vmdk vm.qcow2
```

### Corrupted Image

**Error**: `Image is corrupted`

**Solutions**:

```bash
# Check QCOW2 integrity
qemu-img check vm.qcow2

# Attempt repair
qemu-img check -r all vm.qcow2

# If repair fails, re-export from source
./hyperexport -vm myvm -verify

# Try different conversion options
qemu-img convert -f vmdk -O qcow2 -o cluster_size=2M vm.vmdk vm.qcow2
```

## Import Issues

### Libvirt Import Failed

**Error**: `error: Failed to create domain from vm.xml`

**Solutions**:

```bash
# Validate XML
virt-xml-validate vm.xml

# Check libvirt status
systemctl status libvirtd

# Verify disk path exists
ls -lh /var/lib/libvirt/images/vm.qcow2

# Check disk permissions
sudo chown libvirt-qemu:kvm /var/lib/libvirt/images/vm.qcow2
sudo chmod 644 /var/lib/libvirt/images/vm.qcow2

# Test with virt-install
virt-install \
  --name test-vm \
  --memory 2048 \
  --vcpus 1 \
  --disk path=/var/lib/libvirt/images/vm.qcow2 \
  --import \
  --debug
```

### Network Bridge Not Found

**Error**: `Requested operation is not valid: network 'br0' is not active`

**Solutions**:

```bash
# List available networks
virsh net-list --all

# Start default network
virsh net-start default
virsh net-autostart default

# Or create bridge network
cat > br0.xml <<EOF
<network>
  <name>br0</name>
  <forward mode='bridge'/>
  <bridge name='br0'/>
</network>
EOF

virsh net-define br0.xml
virsh net-start br0
virsh net-autostart br0

# Or use NAT network instead
virt-install --network network=default ...
```

## Boot Failures

### Kernel Panic

**Error**: `Kernel panic - not syncing: VFS: Unable to mount root fs`

**Cause**: Missing virtio drivers in initramfs

**Solutions**:

```bash
# Boot from rescue CD
# Mount root partition
mount /dev/sda2 /mnt
mount --bind /dev /mnt/dev
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys
chroot /mnt

# Rebuild initramfs with virtio
dracut --force --add-drivers "virtio_blk virtio_scsi virtio_net" \
  /boot/initramfs-$(uname -r).img $(uname -r)

# Update GRUB
grub2-mkconfig -o /boot/grub2/grub.cfg

# Exit and reboot
exit
umount /mnt/{sys,proc,dev}
umount /mnt
reboot
```

### GRUB Boot Error

**Error**: `GRUB error: unknown filesystem`

**Solutions**:

```bash
# Boot from rescue CD
# Reinstall GRUB
mount /dev/sda2 /mnt
mount --bind /dev /mnt/dev
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys
chroot /mnt

grub2-install /dev/sda
grub2-mkconfig -o /boot/grub2/grub.cfg

exit
reboot
```

### Stuck at Boot

**Symptoms**: VM hangs at boot, no login prompt

**Solutions**:

```bash
# Connect to serial console
virsh console vm-name

# Check systemd journal
journalctl -xe

# Boot into rescue mode
# Edit GRUB entry, add to kernel line:
systemd.unit=rescue.target

# Or emergency mode
systemd.unit=emergency.target

# Disable problematic services
systemctl disable vmware-tools
systemctl mask cloud-init
```

## Network Issues

### Network Interface Not Found

**Error**: `Device eth0 does not seem to be present`

**Cause**: Interface name changed (ens192 → eth0)

**Solutions**:

```bash
# Check actual interface names
ip link show

# Update network configuration
cd /etc/systemd/network/

# Rename configuration file
mv 10-ens192.network 10-eth0.network

# Update interface name in file
sed -i 's/ens192/eth0/g' 10-eth0.network

# Restart networking
systemctl restart systemd-networkd
```

### No Network Connectivity

**Solutions**:

```bash
# Check interface status
ip addr show
ip link show

# Check if interface is up
ip link set eth0 up

# Test DHCP
dhclient eth0

# Or configure static IP
cat > /etc/systemd/network/10-eth0.network <<EOF
[Match]
Name=eth0

[Network]
Address=192.168.1.100/24
Gateway=192.168.1.1
DNS=8.8.8.8
EOF

systemctl restart systemd-networkd

# Verify routing
ip route show
ping -c 4 8.8.8.8
```

### MAC Address Changed

**Solutions**:

```bash
# Update network config to not bind to MAC
cat > /etc/systemd/network/10-eth0.network <<EOF
[Match]
Name=eth0
# Don't match on MAC address

[Network]
DHCP=yes
EOF

# Or update to new MAC
virsh dumpxml vm-name | grep "mac address"

# Restart VM
virsh destroy vm-name
virsh start vm-name
```

## Performance Problems

### Slow Disk I/O

**Solutions**:

```bash
# Check current cache mode
virsh dumpxml vm-name | grep cache

# Change to none for better performance
virsh edit vm-name
# Change: cache='writeback' to cache='none'

# Use native I/O
<driver name='qemu' type='qcow2' cache='none' io='native'/>

# Enable discard/TRIM
<driver name='qemu' type='qcow2' cache='none' io='native' discard='unmap'/>

# Use virtio-scsi instead of IDE
<disk type='file' device='disk'>
  <target dev='sda' bus='scsi'/>
</disk>
```

### High CPU Usage

**Solutions**:

```bash
# Use host CPU passthrough
virsh edit vm-name

<cpu mode='host-passthrough'/>

# Or specific CPU model
<cpu mode='custom' match='exact'>
  <model>IvyBridge</model>
</cpu>

# Enable nested virtualization (if needed)
<cpu mode='host-passthrough'>
  <feature policy='require' name='vmx'/>
</cpu>

# Verify on host
cat /sys/module/kvm_intel/parameters/nested
```

### Network Performance Issues

**Solutions**:

```bash
# Ensure virtio is used
virsh edit vm-name

<interface type='bridge'>
  <model type='virtio'/>
</interface>

# Enable multiqueue
<interface type='bridge'>
  <model type='virtio'/>
  <driver name='vhost' queues='4'/>
</interface>

# Inside VM, enable multiqueue
ethtool -L eth0 combined 4
```

## Daemon Issues

### Daemon Won't Start

**Error**: `Failed to start hypervisord`

**Solutions**:

```bash
# Check logs
sudo journalctl -u hypervisord -xe

# Test configuration
hypervisord --config /etc/hypervisord/config.yaml --validate

# Check port availability
sudo netstat -tlnp | grep 8080
sudo lsof -i :8080

# Kill process using port
sudo kill $(sudo lsof -t -i:8080)

# Start in foreground for debugging
hypervisord --config /etc/hypervisord/config.yaml --log-level debug
```

### Database Locked

**Error**: `database is locked`

**Solutions**:

```bash
# Stop daemon
sudo systemctl stop hypervisord

# Check for stale locks
lsof /var/lib/hypersdk/hypersdk.db

# Remove lock
rm /var/lib/hypersdk/hypersdk.db-lock

# Restart daemon
sudo systemctl start hypervisord
```

## API Errors

### 401 Unauthorized

**Solutions**:

```bash
# Check if authentication is required
curl http://localhost:8080/health

# If using API key
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8080/jobs/query
```

### 500 Internal Server Error

**Solutions**:

```bash
# Check daemon logs
sudo journalctl -u hypervisord -n 100

# Test API endpoint
curl -v http://localhost:8080/status

# Verify request format
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{"name":"test","vm_path":"/dc/vm/test"}'
```

## WebSocket Problems

### WebSocket Connection Failed

**Error**: `WebSocket connection to 'ws://localhost:8080/ws' failed`

**Solutions**:

```bash
# Check if daemon is running
systemctl status hypervisord

# Test WebSocket endpoint
wscat -c ws://localhost:8080/ws

# Check firewall
sudo firewall-cmd --list-ports
sudo firewall-cmd --add-port=8080/tcp

# Check browser console for detailed errors
# Verify no proxy blocking WebSocket
```

### Real-time Updates Not Working

**Solutions**:

```bash
# Verify WebSocket connection in browser dev tools
# Check Network tab for ws:// connection

# Test with curl
curl http://localhost:8080/ws

# Restart daemon
sudo systemctl restart hypervisord

# Clear browser cache
# Hard refresh: Ctrl+Shift+R
```

## Getting Help

If issues persist:

1. **Check Logs**: `journalctl -u hypervisord -n 200`
2. **Enable Debug Logging**: Set `log_level: debug` in config
3. **GitHub Issues**: https://github.com/ssahani/hypersdk/issues
4. **Include Information**:
   - HyperSDK version
   - Operating system
   - Configuration (sanitized)
   - Complete error messages
   - Steps to reproduce

## See Also

- [Installation Guide](installation-guide.md)
- [Configuration Reference](configuration-reference.md)
- [Migration Workflows](migration-workflows.md)
- [API Reference](API_ENDPOINTS.md)
