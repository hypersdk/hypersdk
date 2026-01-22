// Host and Cluster Information

## Overview

HyperSDK provides comprehensive commands to query ESXi host information, vCenter details, cluster data, and datacenter inventory.

## Commands

### List ESXi Hosts

Get information about all ESXi hosts in your infrastructure:

```bash
hyperctl host -op list
```

**Output:**
```
┌─────────────────┬─────────────┬───────────┬─────────┬────────────┬──────────┬─────────┐
│ Host Name       │ Datacenter  │ Cluster   │ State   │ Power      │ CPU      │ Memory  │
├─────────────────┼─────────────┼───────────┼─────────┼────────────┼──────────┼─────────┤
│ esxi-host-01    │ DC1         │ Cluster1  │ connected│ poweredOn  │ 24 cores │ 256 GB  │
│ esxi-host-02    │ DC1         │ Cluster1  │ connected│ poweredOn  │ 24 cores │ 256 GB  │
│ esxi-host-03    │ DC2         │ Cluster2  │ connected│ poweredOn  │ 32 cores │ 512 GB  │
└─────────────────┴─────────────┴───────────┴─────────┴────────────┴──────────┴─────────┘
```

**Filter by pattern:**
```bash
hyperctl host -op list -pattern "esxi-prod-*"
```

**JSON output:**
```bash
hyperctl host -op list -json
```

**Output structure:**
```json
{
  "hosts": [
    {
      "name": "esxi-host-01",
      "datacenter": "DC1",
      "cluster": "Cluster1",
      "connection_state": "connected",
      "power_state": "poweredOn",
      "cpu_model": "Intel(R) Xeon(R) Gold 6248R CPU @ 3.00GHz",
      "cpu_cores": 24,
      "cpu_threads": 48,
      "cpu_mhz": 3000,
      "memory_mb": 262144,
      "num_vms": 45,
      "version": "7.0.3",
      "build": "19193900"
    }
  ]
}
```

### Get Specific Host Details

```bash
hyperctl host -op info -name esxi-host-01
```

### List Clusters

```bash
hyperctl cluster -op list
```

**Output:**
```
┌──────────────┬─────────┬────────────┬────────────┬──────────┬─────┬─────┐
│ Cluster Name │ Hosts   │ Total CPU  │ Total RAM  │ VMs      │ DRS │ HA  │
├──────────────┼─────────┼────────────┼────────────┼──────────┼─────┼─────┤
│ Cluster1     │ 4       │ 96 GHz     │ 1024 GB    │ 180      │ ✓   │ ✓   │
│ Cluster2     │ 6       │ 192 GHz    │ 3072 GB    │ 320      │ ✓   │ ✓   │
└──────────────┴─────────┴────────────┴────────────┴──────────┴─────┴─────┘
```

### Get vCenter Information

```bash
hyperctl vcenter -op info
```

**Output:**
```json
{
  "version": "7.0.3",
  "build": "19234570",
  "os_type": "linux-x64",
  "product_line_id": "vpx",
  "api_version": "7.0",
  "instance_uuid": "12345678-1234-1234-1234-123456789012"
}
```

### List Datacenters

```bash
hyperctl datacenter -op list
```

**Output:**
```
┌──────────────┬────────┬──────┬────────────┬──────────────┐
│ Datacenter   │ Hosts  │ VMs  │ Clusters   │ Datastores   │
├──────────────┼────────┼──────┼────────────┼──────────────┤
│ DC1          │ 10     │ 500  │ 2          │ 15           │
│ DC2          │ 15     │ 800  │ 3          │ 25           │
└──────────────┴────────┴──────┴────────────┴──────────────┘
```

## API Endpoints

### GET /vsphere/hosts

**Query Parameters:**
- `pattern` (optional): Filter hosts by name pattern (glob)
- `datacenter` (optional): Filter by datacenter name

**Response:**
```json
{
  "hosts": [
    {
      "name": "esxi-host-01",
      "path": "/DC1/host/Cluster1/esxi-host-01",
      "datacenter": "DC1",
      "cluster": "Cluster1",
      "connection_state": "connected",
      "power_state": "poweredOn",
      "cpu_cores": 24,
      "memory_mb": 262144,
      "num_vms": 45
    }
  ]
}
```

### GET /vsphere/hosts/:name

Get details for a specific host.

### GET /vsphere/clusters

**Query Parameters:**
- `pattern` (optional): Filter clusters by name pattern

**Response:**
```json
{
  "clusters": [
    {
      "name": "Cluster1",
      "total_cpu": 96000,
      "total_memory": 1048576,
      "num_hosts": 4,
      "num_cpu_cores": 96,
      "drs_enabled": true,
      "ha_enabled": true
    }
  ]
}
```

### GET /vsphere/vcenter/info

Get vCenter server information.

### GET /vsphere/datacenters

List all datacenters.

## Use Cases

### 1. Capacity Planning

Check available resources across your infrastructure:

```bash
# List all hosts with resource information
hyperctl host -op list -json > hosts.json

# Analyze CPU and memory utilization
# (Use jq or custom scripts to process JSON)
```

### 2. Health Monitoring

Monitor host connection states:

```bash
# Check for disconnected hosts
hyperctl host -op list | grep -i disconnected

# Get full status in JSON for monitoring tools
hyperctl host -op list -json | \
  jq '.hosts[] | select(.connection_state != "connected")'
```

### 3. Inventory Reporting

Generate infrastructure reports:

```bash
# Get cluster overview
hyperctl cluster -op list

# Get detailed datacenter inventory
hyperctl datacenter -op list -json | \
  jq '.datacenters[] | {name, hosts: .num_hosts, vms: .num_vms}'
```

### 4. Pre-Migration Checks

Before migrating VMs, verify target host capacity:

```bash
# Check host with most free resources
hyperctl host -op list -json | \
  jq '[.hosts[] | {name, memory_mb, num_vms}] | sort_by(.num_vms)'
```

## Error Handling

### Common Errors

**Error: Connection timeout**
```
Error: failed to connect to vCenter: context deadline exceeded
```
**Solution:** Check network connectivity and vCenter availability.

**Error: Host not found**
```
Error: host 'esxi-host-99' not found
```
**Solution:** Verify host name and check if host exists in vCenter.

**Error: Insufficient permissions**
```
Error: permission denied: user lacks read access
```
**Solution:** Ensure vCenter user has appropriate permissions.

## Performance Considerations

- Listing large numbers of hosts (>100) may take several seconds
- Use `-pattern` to filter results and improve performance
- API calls use property collector for efficient bulk queries
- Results are cached for 30 seconds in the daemon

## See Also

- [Performance Metrics](PERFORMANCE_METRICS.md)
- [Resource Pools](RESOURCE_POOLS.md)
- [VM Cloning](VM_CLONING.md)
