# HyperSDK API Reference - New Features

**Quick reference for the 34 newly added API endpoints**

---

## Network Management (8 endpoints)

### List Networks
```http
GET /libvirt/networks
```
Returns list of all libvirt networks with status.

### Get Network Details
```http
GET /libvirt/network?name=<network-name>
```
Returns detailed network information (UUID, active, bridge, etc.).

### Create Network
```http
POST /libvirt/network/create
Content-Type: application/json

{
  "name": "vm-network",
  "bridge": "virbr1",
  "forward": "nat",
  "subnet": "192.168.100.1",
  "ip_start": "192.168.100.10",
  "ip_end": "192.168.100.100"
}
```

### Delete Network
```http
POST /libvirt/network/delete
Content-Type: application/json

{
  "name": "vm-network"
}
```

### Start Network
```http
POST /libvirt/network/start
Content-Type: application/json

{
  "name": "vm-network"
}
```

### Stop Network
```http
POST /libvirt/network/stop
Content-Type: application/json

{
  "name": "vm-network"
}
```

### Attach Network Interface
```http
POST /libvirt/interface/attach
Content-Type: application/json

{
  "vm_name": "my-vm",
  "network": "vm-network",
  "model": "virtio"
}
```

### Detach Network Interface
```http
POST /libvirt/interface/detach
Content-Type: application/json

{
  "vm_name": "my-vm",
  "mac": "52:54:00:12:34:56"
}
```

---

## Volume Operations (7 endpoints)

### Get Volume Info
```http
GET /libvirt/volume/info?pool=<pool>&volume=<name>
```

### Create Volume
```http
POST /libvirt/volume/create
Content-Type: application/json

{
  "pool": "default",
  "name": "data-disk",
  "format": "qcow2",
  "capacity": 50,
  "prealloc": false
}
```

### Clone Volume
```http
POST /libvirt/volume/clone
Content-Type: application/json

{
  "pool": "default",
  "source_volume": "vm-disk",
  "target_volume": "vm-disk-backup",
  "target_pool": "default"
}
```

### Resize Volume
```http
POST /libvirt/volume/resize
Content-Type: application/json

{
  "pool": "default",
  "volume": "data-disk",
  "capacity": 100,
  "shrink": false
}
```

### Delete Volume
```http
POST /libvirt/volume/delete
Content-Type: application/json

{
  "pool": "default",
  "volume": "old-disk"
}
```

### Upload Volume
```http
POST /libvirt/volume/upload
Content-Type: application/json

{
  "pool": "default",
  "volume": "imported-vm",
  "source_path": "/data/vm-disk.vmdk",
  "format": "qcow2"
}
```

### Wipe Volume
```http
POST /libvirt/volume/wipe
Content-Type: application/json

{
  "pool": "default",
  "volume": "old-disk",
  "algorithm": "zero"
}
```

**Wipe Algorithms**: `zero`, `nnsa`, `dod`, `bsi`, `gutmann`, `schneier`, `pfitzner7`, `pfitzner33`, `random`

---

## Resource Monitoring (6 endpoints)

### Get All VM Stats
```http
GET /libvirt/stats?name=<vm-name>
```
Returns comprehensive stats: CPU, memory, disk I/O, network I/O.

### Get All Running VMs Stats
```http
GET /libvirt/stats/all
```
Returns stats for all running VMs.

### Get CPU Stats
```http
GET /libvirt/stats/cpu?name=<vm-name>
```
Returns: vCPUs, CPU time (user, system, total), usage %.

### Get Memory Stats
```http
GET /libvirt/stats/memory?name=<vm-name>
```
Returns: total, used, available, usage %, swap in/out.

### Get Disk I/O Stats
```http
GET /libvirt/stats/disk?name=<vm-name>
```
Returns per-disk: read/write bytes, read/write requests, errors.

### Get Network I/O Stats
```http
GET /libvirt/stats/network?name=<vm-name>
```
Returns per-interface: RX/TX bytes, packets, errors, dropped.

---

## Batch Operations (7 endpoints)

### Batch Start VMs
```http
POST /libvirt/batch/start
Content-Type: application/json

{
  "domains": ["vm1", "vm2", "vm3"],
  "paused": false
}
```

### Batch Stop VMs
```http
POST /libvirt/batch/stop
Content-Type: application/json

{
  "domains": ["vm1", "vm2", "vm3"],
  "force": false
}
```
**force**: `true` = destroy (immediate), `false` = shutdown (graceful)

### Batch Reboot VMs
```http
POST /libvirt/batch/reboot
Content-Type: application/json

{
  "domains": ["vm1", "vm2"],
  "force": false
}
```
**force**: `true` = reset (hard), `false` = reboot (graceful)

### Batch Snapshot VMs
```http
POST /libvirt/batch/snapshot
Content-Type: application/json

{
  "domains": ["vm1", "vm2"],
  "name_prefix": "backup",
  "description": "Pre-upgrade backup",
  "atomic": true,
  "disk_only": false
}
```

### Batch Delete VMs
```http
POST /libvirt/batch/delete
Content-Type: application/json

{
  "domains": ["old-vm1", "old-vm2"],
  "remove_storage": true,
  "snapshots_only": false
}
```

### Batch Pause VMs
```http
POST /libvirt/batch/pause
Content-Type: application/json

{
  "domains": ["vm1", "vm2"]
}
```

### Batch Resume VMs
```http
POST /libvirt/batch/resume
Content-Type: application/json

{
  "domains": ["vm1", "vm2"]
}
```

---

## VM Cloning & Templates (6 endpoints)

### Clone VM
```http
POST /libvirt/clone
Content-Type: application/json

{
  "source": "base-vm",
  "target": "cloned-vm",
  "files": [],
  "new_mac": true,
  "auto_clone": true,
  "preserve": true
}
```

### Clone Multiple VMs
```http
POST /libvirt/clone/multiple
Content-Type: application/json

{
  "source": "web-template",
  "name_prefix": "vm-web",
  "count": 5,
  "start_index": 1
}
```
Creates: `vm-web-1`, `vm-web-2`, `vm-web-3`, `vm-web-4`, `vm-web-5`

### Create Template from VM
```http
POST /libvirt/template/create
Content-Type: application/json

{
  "domain": "base-ubuntu",
  "name": "ubuntu-22-template",
  "description": "Ubuntu 22.04 base template",
  "metadata": {},
  "seal": true
}
```
**seal**: Uses `virt-sysprep` to remove machine-specific config

### Deploy VM from Template
```http
POST /libvirt/template/deploy
Content-Type: application/json

{
  "template": "ubuntu-22-template",
  "name": "new-webserver",
  "memory": 4096,
  "vcpus": 2,
  "network": "vm-network",
  "autostart": true,
  "customize": {}
}
```

### List Templates
```http
GET /libvirt/template/list
```
Returns all VMs with "template" in their name or shut-off VMs.

### Export Template
```http
POST /libvirt/template/export
Content-Type: application/json

{
  "template": "ubuntu-22-template",
  "export_path": "/backups/templates",
  "compress": true
}
```
Exports XML definition and disk images, optionally compressed.

---

## Common Response Format

All endpoints return JSON responses:

### Success Response
```json
{
  "status": "success",
  "message": "Operation completed successfully",
  "data": { ... },
  "timestamp": "2026-01-19T12:30:00Z"
}
```

### Error Response
```json
{
  "error": "Error message here",
  "timestamp": "2026-01-19T12:30:00Z"
}
```

### Batch Operation Response
```json
{
  "operation": "start",
  "total": 4,
  "successful": 3,
  "failed": 1,
  "results": [
    {"domain": "vm1", "success": true},
    {"domain": "vm2", "success": true},
    {"domain": "vm3", "success": true},
    {"domain": "vm4", "success": false, "error": "domain not found"}
  ],
  "start_time": "2026-01-19T12:00:00Z",
  "end_time": "2026-01-19T12:00:15Z",
  "duration": "15s"
}
```

---

## Testing Endpoints

### Using curl

```bash
# Test network listing
curl http://localhost:8080/libvirt/networks

# Test VM stats
curl http://localhost:8080/libvirt/stats?name=my-vm

# Test batch start
curl -X POST http://localhost:8080/libvirt/batch/start \
  -H "Content-Type: application/json" \
  -d '{"domains":["vm1","vm2"]}'
```

### Using httpie

```bash
# Test network creation
http POST http://localhost:8080/libvirt/network/create \
  name=vm-network \
  bridge=virbr1 \
  forward=nat \
  subnet=192.168.100.1 \
  ip_start=192.168.100.10 \
  ip_end=192.168.100.100

# Test volume creation
http POST http://localhost:8080/libvirt/volume/create \
  pool=default \
  name=data-disk \
  format=qcow2 \
  capacity:=50

# Test VM cloning
http POST http://localhost:8080/libvirt/clone \
  source=base-vm \
  target=cloned-vm \
  new_mac:=true
```

---

## Error Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created (for create operations) |
| 400 | Bad Request (invalid JSON, missing parameters) |
| 404 | Not Found (VM, network, or volume doesn't exist) |
| 405 | Method Not Allowed (wrong HTTP method) |
| 500 | Internal Server Error (virsh command failed) |

---

## Notes

- All POST endpoints require `Content-Type: application/json` header
- GET endpoints use URL query parameters
- Batch operations execute concurrently (in parallel)
- Network/volume names must be valid libvirt identifiers
- Resource monitoring requires qemu-guest-agent for some metrics
- VM cloning requires sufficient disk space
- Template sealing requires `virt-sysprep` package

---

## Complete Endpoint List

**Total: 119 endpoints** (85 original + 34 new)

### New in This Release (34 endpoints)
1. Network Management: 8 endpoints
2. Volume Operations: 7 endpoints
3. Resource Monitoring: 6 endpoints
4. Batch Operations: 7 endpoints
5. VM Cloning & Templates: 6 endpoints

### Original Features (85 endpoints)
- Core: 3 endpoints
- Jobs: 4 endpoints
- VMs: 5 endpoints
- Scheduler: 5 endpoints
- Users: 6 endpoints
- Notifications: 5 endpoints
- Hyper2KVM: 5 endpoints
- Cost: 4 endpoints
- Organization: 6 endpoints
- Cloud: 6 endpoints
- Security: 5 endpoints
- Migration: 3 endpoints
- Config: 2 endpoints
- Libvirt: 15 endpoints
- Workflow: 2 endpoints
- Console: 6 endpoints

---

## Quick Start Examples

### Scenario 1: Create Network and Attach to VM

```bash
# Step 1: Create network
curl -X POST http://localhost:8080/libvirt/network/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "vm-network",
    "bridge": "virbr1",
    "forward": "nat",
    "subnet": "192.168.100.1",
    "ip_start": "192.168.100.10",
    "ip_end": "192.168.100.100"
  }'

# Step 2: Start network
curl -X POST http://localhost:8080/libvirt/network/start \
  -H "Content-Type: application/json" \
  -d '{"name": "vm-network"}'

# Step 3: Attach interface to VM
curl -X POST http://localhost:8080/libvirt/interface/attach \
  -H "Content-Type: application/json" \
  -d '{
    "vm_name": "my-vm",
    "network": "vm-network",
    "model": "virtio"
  }'
```

### Scenario 2: Clone VM and Monitor

```bash
# Step 1: Clone VM
curl -X POST http://localhost:8080/libvirt/clone \
  -H "Content-Type: application/json" \
  -d '{
    "source": "template-vm",
    "target": "new-vm",
    "new_mac": true
  }'

# Step 2: Start cloned VM
curl -X POST http://localhost:8080/libvirt/domain/start \
  -H "Content-Type: application/json" \
  -d '{"name": "new-vm"}'

# Step 3: Monitor performance
curl http://localhost:8080/libvirt/stats?name=new-vm
```

### Scenario 3: Batch Deploy Infrastructure

```bash
# Step 1: Clone template 10 times
curl -X POST http://localhost:8080/libvirt/clone/multiple \
  -H "Content-Type: application/json" \
  -d '{
    "source": "web-template",
    "name_prefix": "web-server",
    "count": 10
  }'

# Step 2: Start all web servers
curl -X POST http://localhost:8080/libvirt/batch/start \
  -H "Content-Type: application/json" \
  -d '{
    "domains": [
      "web-server-1", "web-server-2", "web-server-3",
      "web-server-4", "web-server-5", "web-server-6",
      "web-server-7", "web-server-8", "web-server-9",
      "web-server-10"
    ]
  }'

# Step 3: Monitor all servers
curl http://localhost:8080/libvirt/stats/all
```

---

**For full documentation, see**: `CRITICAL_FEATURES_ADDED.md`
