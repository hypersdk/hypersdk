# VM Cloning and Templates

## Overview

HyperSDK provides powerful VM cloning capabilities including full clones, linked clones, template creation, and template deployment. Supports bulk cloning with configurable concurrency.

## Commands

### Clone a Single VM

Basic VM clone:

```bash
hyperctl clone -source source-vm -target cloned-vm
```

**Clone and power on:**
```bash
hyperctl clone -source source-vm -target cloned-vm -power-on
```

**Clone to specific folder:**
```bash
hyperctl clone -source source-vm -target cloned-vm \
  -folder "/DC1/vm/Development"
```

**Clone to specific resource pool:**
```bash
hyperctl clone -source source-vm -target cloned-vm \
  -pool "dev-pool"
```

**Clone to specific datastore:**
```bash
hyperctl clone -source source-vm -target cloned-vm \
  -datastore "datastore1"
```

### Linked Clones

Create a linked clone from a snapshot:

```bash
hyperctl clone -source template-vm -target linked-clone \
  -linked -snapshot "baseline-snapshot"
```

**Benefits of linked clones:**
- Fast creation (seconds vs minutes)
- Minimal disk space (only stores differences)
- Ideal for testing and development

**Requirements:**
- Source VM must have at least one snapshot
- Specify snapshot name with `-snapshot` flag

### Template Operations

**Convert VM to template:**
```bash
hyperctl template -op create -vm source-vm
```

**Deploy VM from template:**
```bash
hyperctl template -op deploy -template template-name \
  -target new-vm -power-on
```

**Convert template back to VM:**
```bash
hyperctl template -op convert -template template-name
```

### Bulk Cloning

Clone multiple VMs from a YAML file:

**clones.yaml:**
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

  - source_vm: "template-ubuntu"
    target_name: "web-server-03"
    power_on: true
    resource_pool: "production"

  - source_vm: "template-centos"
    target_name: "db-server-01"
    power_on: false
    resource_pool: "database"
```

**Execute bulk clone:**
```bash
hyperctl clone -bulk clones.yaml -max-concurrent 5
```

**Output:**
```
Bulk cloning 4 VMs (max 5 concurrent)...

✓ web-server-01: Success (45s)
✓ web-server-02: Success (47s)
✓ web-server-03: Success (46s)
✓ db-server-01: Success (52s)

Summary: 4 succeeded, 0 failed (Total: 3m10s)
```

## API Endpoints

### POST /vsphere/clone

Clone a single VM.

**Request:**
```json
{
  "source_vm": "template-ubuntu",
  "target_name": "web-server-01",
  "target_folder": "/DC1/vm/Production",
  "resource_pool": "prod-pool",
  "datastore": "datastore1",
  "power_on": true,
  "linked_clone": false,
  "template": false
}
```

**Response:**
```json
{
  "vm_name": "web-server-01",
  "vm_path": "/DC1/vm/Production/web-server-01",
  "success": true,
  "duration": "45s"
}
```

### POST /vsphere/clone/bulk

Clone multiple VMs concurrently.

**Request:**
```json
{
  "clones": [
    {
      "source_vm": "template-ubuntu",
      "target_name": "web-server-01",
      "power_on": true
    },
    {
      "source_vm": "template-ubuntu",
      "target_name": "web-server-02",
      "power_on": true
    }
  ],
  "max_concurrent": 5
}
```

**Response:**
```json
{
  "results": [
    {
      "vm_name": "web-server-01",
      "vm_path": "/DC1/vm/web-server-01",
      "success": true,
      "duration": "45s"
    },
    {
      "vm_name": "web-server-02",
      "vm_path": "/DC1/vm/web-server-02",
      "success": true,
      "duration": "47s"
    }
  ],
  "succeeded": 2,
  "failed": 0,
  "total_duration": "47s"
}
```

### POST /vsphere/template/create

Convert VM to template.

**Request:**
```json
{
  "vm_name": "source-vm"
}
```

### POST /vsphere/template/deploy

Deploy VM from template.

**Request:**
```json
{
  "source_vm": "template-name",
  "target_name": "new-vm",
  "power_on": true
}
```

## Use Cases

### 1. Test Environment Provisioning

Quickly create multiple test VMs from a template:

```bash
# Create 10 test VMs
for i in {1..10}; do
  hyperctl clone -source test-template \
    -target "test-vm-$(printf %02d $i)" \
    -pool "test-pool"
done
```

### 2. Development Environment

Create linked clones for developers:

```bash
# Create snapshot of base environment
# Then create linked clones for each developer

hyperctl clone -source dev-baseline -target dev-alice \
  -linked -snapshot "baseline-2024-01" -power-on

hyperctl clone -source dev-baseline -target dev-bob \
  -linked -snapshot "baseline-2024-01" -power-on
```

### 3. Production Deployment

Deploy production servers from template:

```yaml
# prod-deployment.yaml
clones:
  - source_vm: "template-app-server"
    target_name: "app-server-01"
    resource_pool: "production"
    datastore: "ssd-storage"
    power_on: true

  - source_vm: "template-app-server"
    target_name: "app-server-02"
    resource_pool: "production"
    datastore: "ssd-storage"
    power_on: true
```

```bash
hyperctl clone -bulk prod-deployment.yaml -max-concurrent 2
```

### 4. Disaster Recovery

Clone VMs to a DR site:

```bash
# Clone critical VMs to DR datastore
hyperctl clone -source prod-db -target dr-db \
  -datastore "dr-datastore" -folder "/DR/vm"

hyperctl clone -source prod-web -target dr-web \
  -datastore "dr-datastore" -folder "/DR/vm"
```

## Interactive TUI

In interactive mode, select a VM and press `c` to clone:

```bash
hyperctl migrate
```

1. Select source VM from list
2. Press `c` for clone
3. Enter target VM name
4. Choose clone options (full/linked, power on)
5. Confirm and execute

**Bulk clone in TUI:**
1. Select multiple VMs (press `space` to select)
2. Press `b` for bulk operations
3. Choose "Clone VMs"
4. Confirm bulk clone

## Clone Specifications

### Full Clone

- Complete copy of source VM
- Independent of source VM
- Can be moved to different datastore
- Takes longer to create
- Uses more disk space

### Linked Clone

- References parent snapshot
- Fast creation
- Minimal disk space
- Dependent on parent snapshot
- Cannot be moved independently

### Template Clone

- Source remains a template
- New VM is independent
- Common for production deployments

## Performance Considerations

### Clone Speed

Clone time depends on:
- VM disk size
- Storage performance
- Network speed (if using NFS/iSCSI)
- Thin vs thick provisioning

**Typical times:**
- Small VM (20GB): 1-2 minutes
- Medium VM (100GB): 5-10 minutes
- Large VM (500GB): 20-40 minutes
- Linked clone: 10-30 seconds

### Concurrent Cloning

```bash
# Clone 50 VMs with max 10 concurrent operations
hyperctl clone -bulk clones.yaml -max-concurrent 10
```

**Recommendations:**
- Use 5-10 concurrent clones for best performance
- Higher concurrency may overload vCenter or storage
- Monitor datastore I/O during bulk operations

## Error Handling

### Common Errors

**Error: Source VM not found**
```
Error: VM 'template-vm' not found
```
**Solution:** Verify source VM name exists.

**Error: Target VM already exists**
```
Error: VM 'cloned-vm' already exists
```
**Solution:** Use a different target name or delete existing VM.

**Error: Insufficient disk space**
```
Error: insufficient disk space on datastore 'datastore1'
```
**Solution:** Free up space or choose different datastore.

**Error: Snapshot not found (linked clone)**
```
Error: snapshot 'baseline' not found on 'template-vm'
```
**Solution:** Verify snapshot exists or create one:
```bash
# Create snapshot first
govc snapshot.create -vm template-vm baseline
```

**Error: Concurrent clone limit**
```
Error: maximum concurrent clones exceeded (10)
```
**Solution:** Reduce `-max-concurrent` value or wait for clones to complete.

## Best Practices

### 1. Use Templates for Production

Convert base VMs to templates:
```bash
hyperctl template -op create -vm base-ubuntu-2204
```

### 2. Create Snapshots for Linked Clones

Before creating linked clones:
```bash
# Take snapshot of base VM
govc snapshot.create -vm dev-baseline baseline-2024-01
```

### 3. Organize with Folders and Resource Pools

```bash
hyperctl clone -source template -target vm-name \
  -folder "/DC1/vm/Development" \
  -pool "dev-pool"
```

### 4. Use Bulk Cloning for Scale

For >5 VMs, use bulk cloning with YAML:
```bash
hyperctl clone -bulk deployment.yaml -max-concurrent 10
```

### 5. Monitor Clone Progress

Use JSON output for automation:
```bash
hyperctl clone -source template -target vm-name -json | \
  jq '.duration'
```

## See Also

- [Resource Pools](RESOURCE_POOLS.md)
- [Bulk Operations](BULK_OPERATIONS.md)
- [Interactive TUI Guide](../user-guide/INTERACTIVE_MODE.md)
