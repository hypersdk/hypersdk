# HyperSDK Features Summary

## New Advanced Features

### 1. Infrastructure Information 🏢

Query ESXi hosts, vCenter, clusters, and datacenters:

```bash
# List all ESXi hosts
hyperctl host -op list

# Get host details
hyperctl host -op info -name esxi-host-01

# List clusters
hyperctl cluster -op list

# Get vCenter information
hyperctl vcenter -op info

# List datacenters
hyperctl datacenter -op list
```

**API Endpoints:**
- `GET /vsphere/hosts` - List ESXi hosts
- `GET /vsphere/hosts/:name` - Get host details
- `GET /vsphere/clusters` - List clusters
- `GET /vsphere/vcenter/info` - vCenter information
- `GET /vsphere/datacenters` - List datacenters

**Use Cases:**
- Capacity planning
- Infrastructure inventory
- Pre-migration checks
- Health monitoring

### 2. Performance Metrics 📊

Real-time and historical performance monitoring:

```bash
# Real-time VM metrics
hyperctl metrics -entity vm-name -type vm -realtime

# Live streaming metrics
hyperctl metrics -entity vm-name -type vm -watch

# Historical metrics
hyperctl metrics -entity vm-name -type vm \
  -start "24h ago" -interval 1h

# Host metrics
hyperctl metrics -entity esxi-host-01 -type host -realtime

# Cluster metrics
hyperctl metrics -entity Cluster1 -type cluster -realtime
```

**Metrics Collected:**
- CPU usage (percent and MHz)
- Memory usage (percent and MB)
- Disk I/O (read/write MB/s)
- Network throughput (RX/TX MB/s)

**API Endpoints:**
- `GET /vsphere/metrics` - Get real-time or historical metrics
- `WebSocket /vsphere/metrics/stream` - Stream real-time metrics

**WebSocket Example:**
```javascript
const ws = new WebSocket('ws://localhost:8080/vsphere/metrics/stream?entity=vm-name&type=vm');
ws.onmessage = (event) => {
  const metrics = JSON.parse(event.data);
  console.log(`CPU: ${metrics.cpu_percent}% | Memory: ${metrics.memory_percent}%`);
};
```

### 3. VM Cloning and Templates 🔄

Full clones, linked clones, template operations, and bulk cloning:

```bash
# Clone a VM
hyperctl clone -source template-vm -target new-vm -power-on

# Linked clone
hyperctl clone -source template-vm -target linked-vm \
  -linked -snapshot baseline-snapshot

# Convert VM to template
hyperctl template -op create -vm source-vm

# Deploy from template
hyperctl template -op deploy -template template-name -target new-vm

# Bulk clone from YAML
hyperctl clone -bulk clones.yaml -max-concurrent 5
```

**Bulk Clone Example** (`clones.yaml`):
```yaml
clones:
  - source_vm: "template-ubuntu"
    target_name: "web-server-01"
    power_on: true
    resource_pool: "production"

  - source_vm: "template-ubuntu"
    target_name: "web-server-02"
    power_on: true
    resource_pool: "production"
```

**API Endpoints:**
- `POST /vsphere/clone` - Clone single VM
- `POST /vsphere/clone/bulk` - Bulk clone VMs
- `POST /vsphere/template/create` - Convert VM to template
- `POST /vsphere/template/deploy` - Deploy from template

### 4. Resource Pool Management 🎯

Create, update, and manage vSphere resource pools:

```bash
# List resource pools
hyperctl pool -op list

# Create resource pool
hyperctl pool -op create -name dev-pool \
  -cpu-reserve 2000 -cpu-limit 8000 \
  -mem-reserve 4096 -mem-limit 16384

# Update resource pool
hyperctl pool -op update -name dev-pool -cpu-limit 12000

# Delete resource pool
hyperctl pool -op delete -name dev-pool
```

**API Endpoints:**
- `GET /vsphere/pools` - List resource pools
- `POST /vsphere/pools` - Create resource pool
- `PUT /vsphere/pools/:name` - Update resource pool
- `DELETE /vsphere/pools/:name` - Delete resource pool

**Use Cases:**
- Environment isolation (dev/test/prod)
- Resource allocation and limits
- Capacity management
- Multi-tenant environments

### 5. Event Monitoring 📡

Stream vCenter events and alerts in real-time:

```bash
# Get recent events
hyperctl events -since 1h

# Filter by event type
hyperctl events -since 24h -types VmPoweredOnEvent,VmCreatedEvent

# Stream events live
hyperctl events -follow
```

**Event Types:**
- VM events (power on/off, created, deleted, migrated)
- Host events (connected, disconnected, maintenance mode)
- Datastore events (discovered, removed)
- Alarm events (triggered, cleared)

**API Endpoints:**
- `GET /vsphere/events` - Get recent events
- `WebSocket /vsphere/events/stream` - Stream events

**WebSocket Example:**
```javascript
const ws = new WebSocket('ws://localhost:8080/vsphere/events/stream');
ws.onmessage = (event) => {
  const evt = JSON.parse(event.data);
  console.log(`[${evt.severity}] ${evt.event_type}: ${evt.message}`);
};
```

### 6. Pipeline Integration (hyper2kvm + libvirt) 🔧

Complete end-to-end VM migration from vSphere to KVM:

**Architecture Flow:**
```
vSphere VM → HyperSDK Export → Artifact Manifest → hyper2kvm → libvirt KVM
```

**CLI Usage:**
```bash
hyperexport \
  --vm "/DC1/vm/Ubuntu-Server" \
  --output /var/lib/libvirt/images/ubuntu \
  --format ova \
  --manifest \
  --pipeline \
  --hyper2kvm-path /home/tt/hyper2kvm/hyper2kvm \
  --libvirt \
  --libvirt-uri "qemu:///system" \
  --libvirt-autostart
```

**What Happens:**
1. ✅ Export VM from vSphere
2. ✅ Generate Artifact Manifest v1.0
3. ✅ Execute hyper2kvm pipeline:
   - **INSPECT**: Detect guest OS, analyze drivers
   - **FIX**: Fix fstab, GRUB, initramfs for KVM
   - **CONVERT**: Convert VMDK → qcow2 with compression
   - **VALIDATE**: Verify image integrity
4. ✅ Define VM in libvirt
5. ✅ Configure auto-start (optional)

**Web Dashboard Integration:**

The pipeline is fully integrated into the web dashboard with collapsible UI sections:

**Pipeline Configuration:**
- ✓ Enable hyper2kvm pipeline after export
- hyper2kvm Path: `/home/tt/hyper2kvm/hyper2kvm`
- Compression Level: `1-9` (default: 6)
- Pipeline Stages:
  - ✓ INSPECT (detect OS)
  - ✓ FIX (fstab, grub)
  - ✓ CONVERT (→ qcow2)
  - ✓ VALIDATE (integrity)

**Libvirt Integration:**
- ✓ Define VM in libvirt after conversion
- Libvirt URI: `qemu:///system`
- Network Bridge: `virbr0`
- Storage Pool: `default`
- ✓ Enable VM auto-start

**API Request Example:**
```json
{
  "name": "Ubuntu Server Migration",
  "vm_path": "/DC1/vm/ubuntu-server",
  "output_dir": "/var/lib/libvirt/images/ubuntu",
  "format": "ova",
  "options": {
    "enable_pipeline": true,
    "hyper2kvm_path": "/home/tt/hyper2kvm/hyper2kvm",
    "pipeline_inspect": true,
    "pipeline_fix": true,
    "pipeline_convert": true,
    "pipeline_validate": true,
    "pipeline_compress": true,
    "compress_level": 6,
    "libvirt_integration": true,
    "libvirt_uri": "qemu:///system",
    "libvirt_autostart": true,
    "libvirt_bridge": "virbr0",
    "libvirt_pool": "default"
  }
}
```

**Pipeline Features:**
- Automatic OS detection (Linux, Windows)
- BIOS and UEFI support
- Multi-disk VM support
- Non-fatal error handling (export succeeds even if pipeline fails)
- Dry-run mode for testing
- Configurable compression levels
- Custom libvirt network bridges and storage pools

**Generated Artifact Manifest:**
```json
{
  "manifest_version": "1.0",
  "source": {
    "provider": "vsphere",
    "vm_name": "Ubuntu-Server",
    "datacenter": "DC1"
  },
  "vm": {
    "cpu": 4,
    "mem_gb": 8,
    "firmware": "bios",
    "os_hint": "linux"
  },
  "disks": [
    {
      "id": "disk-0",
      "source_format": "vmdk",
      "bytes": 107374182400,
      "local_path": "/var/lib/libvirt/images/ubuntu/disk.vmdk",
      "boot_order_hint": 0,
      "disk_type": "boot"
    }
  ],
  "pipeline": {
    "inspect": {"enabled": true},
    "fix": {"enabled": true, "fstab_mode": "stabilize-all"},
    "convert": {"enabled": true, "compress": true},
    "validate": {"enabled": true}
  }
}
```

### 7. Bulk Operations in Interactive TUI 🎮

Enhanced interactive mode with bulk VM operations:

```bash
hyperctl migrate
```

**New Keyboard Shortcuts:**
- `b` - Bulk operations menu
- `m` - Show metrics for current VM
- `e` - Show events for current VM
- `c` - Clone current VM

**Bulk Operations Menu:**
1. Clone VMs - Clone selected VMs concurrently
2. Create Snapshots - Snapshot all selected VMs
3. Power Off VMs - Graceful shutdown of selected VMs
4. Delete VMs - Remove selected VMs
5. Modify Resource Pool - Change pool assignment
6. Change Network - Reconfigure network settings

**Multi-Selection:**
- Press `Space` to select/deselect VMs
- Press `b` to open bulk operations menu
- Choose operation and confirm
- Watch real-time progress for all operations

## Command Reference

### Infrastructure Commands

```bash
# Hosts
hyperctl host -op list [-pattern "esxi-*"] [-json]
hyperctl host -op info -name esxi-host-01

# Clusters
hyperctl cluster -op list [-pattern "Cluster*"] [-json]

# vCenter
hyperctl vcenter -op info

# Datacenters
hyperctl datacenter -op list
```

### Performance Commands

```bash
# Real-time metrics
hyperctl metrics -entity vm-name -type vm -realtime
hyperctl metrics -entity host-name -type host -realtime
hyperctl metrics -entity cluster-name -type cluster -realtime

# Watch metrics live
hyperctl metrics -entity vm-name -type vm -watch

# Historical metrics
hyperctl metrics -entity vm-name -type vm \
  -start "2024-01-20T10:00:00Z" \
  -end "2024-01-20T18:00:00Z" \
  -interval 5min
```

### Clone Commands

```bash
# Single clone
hyperctl clone -source template -target new-vm [-power-on]

# Linked clone
hyperctl clone -source template -target linked-vm \
  -linked -snapshot baseline

# Bulk clone
hyperctl clone -bulk clones.yaml -max-concurrent 5

# Template operations
hyperctl template -op create -vm source-vm
hyperctl template -op deploy -template template-name -target new-vm
```

### Resource Pool Commands

```bash
# List pools
hyperctl pool -op list

# Create pool
hyperctl pool -op create -name pool-name \
  -cpu-reserve 2000 -cpu-limit 8000 \
  -mem-reserve 4096 -mem-limit 16384

# Update pool
hyperctl pool -op update -name pool-name -cpu-limit 12000

# Delete pool
hyperctl pool -op delete -name pool-name
```

### Event Commands

```bash
# Recent events
hyperctl events -since 1h
hyperctl events -since 24h

# Filter by type
hyperctl events -since 1h -types VmPoweredOnEvent,VmCreatedEvent

# Stream events
hyperctl events -follow
```

### Pipeline Commands

```bash
# Export with pipeline
hyperexport --vm "VM-Name" --output /path/to/output \
  --format ova --manifest \
  --pipeline \
  --hyper2kvm-path /path/to/hyper2kvm \
  --libvirt --libvirt-uri "qemu:///system"

# Pipeline options
--pipeline-inspect        # Enable INSPECT stage
--pipeline-fix            # Enable FIX stage
--pipeline-convert        # Enable CONVERT stage
--pipeline-validate       # Enable VALIDATE stage
--pipeline-compress       # Enable compression
--compress-level 6        # Compression level 1-9

# Libvirt options
--libvirt                 # Define in libvirt
--libvirt-uri URI         # Libvirt URI
--libvirt-autostart       # Enable auto-start
--libvirt-bridge BRIDGE   # Network bridge
--libvirt-pool POOL       # Storage pool
```

## API Summary

### New Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/vsphere/hosts` | GET | List ESXi hosts |
| `/vsphere/hosts/:name` | GET | Get host details |
| `/vsphere/clusters` | GET | List clusters |
| `/vsphere/vcenter/info` | GET | vCenter information |
| `/vsphere/datacenters` | GET | List datacenters |
| `/vsphere/metrics` | GET | Get performance metrics |
| `/vsphere/metrics/stream` | WebSocket | Stream metrics |
| `/vsphere/clone` | POST | Clone VM |
| `/vsphere/clone/bulk` | POST | Bulk clone VMs |
| `/vsphere/template/create` | POST | Create template |
| `/vsphere/template/deploy` | POST | Deploy from template |
| `/vsphere/pools` | GET | List resource pools |
| `/vsphere/pools` | POST | Create resource pool |
| `/vsphere/pools/:name` | PUT | Update resource pool |
| `/vsphere/pools/:name` | DELETE | Delete resource pool |
| `/vsphere/events` | GET | Get events |
| `/vsphere/events/stream` | WebSocket | Stream events |

## Documentation

Comprehensive documentation available:

- [Host and Cluster Info](docs/features/HOST_CLUSTER_INFO.md)
- [Performance Metrics](docs/features/PERFORMANCE_METRICS.md)
- [VM Cloning](docs/features/VM_CLONING.md)
- [Resource Pools](docs/features/RESOURCE_POOLS.md)
- [Event Monitoring](docs/features/EVENT_MONITORING.md)
- [Pipeline Integration](PIPELINE_INTEGRATION.md)
- [Web Dashboard Pipeline](web/dashboard-react/PIPELINE_INTEGRATION.md)
- [Pipeline Tutorial](docs/TUTORIAL_PIPELINE.md)

## Test Coverage

Comprehensive test suite with >80% coverage:

- Unit tests for all provider functions
- Integration tests for API handlers
- End-to-end CLI tests
- WebSocket streaming tests
- Pipeline integration tests

Run tests:
```bash
go test ./providers/vsphere/... -v
go test ./daemon/api/... -v
go test ./cmd/hyperctl/... -v
go test ./providers/common/... -v
```

## Performance Characteristics

- **Metrics Collection**: 20-second realtime interval (vSphere limitation)
- **Event Polling**: 5-second interval for new events
- **Concurrent Cloning**: 5-10 concurrent operations recommended
- **WebSocket Updates**: Automatic 20-second push intervals
- **API Response Time**: <100ms for most queries (cached)
- **Pipeline Execution**: Depends on VM size (typical: 10-30 minutes)

## Future Enhancements

- [ ] Live VM migration (vMotion equivalent for KVM)
- [ ] VM snapshot management
- [ ] Storage vMotion support
- [ ] Network configuration migration
- [ ] Cloud-init integration
- [ ] Multi-cloud pipeline support (AWS, Azure, GCP)
- [ ] Advanced scheduling with cron expressions
- [ ] Email notifications for pipeline completion
- [ ] Metrics export to Prometheus
- [ ] Grafana dashboard templates
