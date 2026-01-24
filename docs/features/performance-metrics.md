# Performance Metrics

## Overview

HyperSDK provides real-time and historical performance metrics for VMs, hosts, and clusters. Metrics include CPU usage, memory usage, disk I/O, and network throughput.

## Commands

### Real-time Metrics

Get current performance metrics for a VM:

```bash
hyperctl metrics -entity vm-name -type vm -realtime
```

**Output:**
```
┌──────────────────┬───────────┬──────────────┬───────────┐
│ Metric           │ Value     │ Unit         │ Timestamp │
├──────────────────┼───────────┼──────────────┼───────────┤
│ CPU Usage        │ 45.2%     │ percent      │ 10:30:15  │
│ CPU (MHz)        │ 2400      │ MHz          │ 10:30:15  │
│ Memory Usage     │ 68.5%     │ percent      │ 10:30:15  │
│ Memory (MB)      │ 5536      │ MB           │ 10:30:15  │
│ Disk Read        │ 12.5      │ MB/s         │ 10:30:15  │
│ Disk Write       │ 8.2       │ MB/s         │ 10:30:15  │
│ Network RX       │ 3.4       │ MB/s         │ 10:30:15  │
│ Network TX       │ 2.1       │ MB/s         │ 10:30:15  │
└──────────────────┴───────────┴──────────────┴───────────┘
```

**For ESXi hosts:**
```bash
hyperctl metrics -entity esxi-host-01 -type host -realtime
```

**For clusters:**
```bash
hyperctl metrics -entity Cluster1 -type cluster -realtime
```

### Live Streaming Metrics

Watch metrics update in real-time:

```bash
hyperctl metrics -entity vm-name -type vm -watch
```

**Output** (updates every 20 seconds):
```
🔄 Streaming metrics for 'vm-name' (press Ctrl+C to stop)

[10:30:15] CPU: 45.2% | Memory: 68.5% | Disk: ↓12.5MB/s ↑8.2MB/s | Net: ↓3.4MB/s ↑2.1MB/s
[10:30:35] CPU: 47.8% | Memory: 68.9% | Disk: ↓14.1MB/s ↑9.3MB/s | Net: ↓3.8MB/s ↑2.3MB/s
[10:30:55] CPU: 43.1% | Memory: 68.7% | Disk: ↓11.2MB/s ↑7.8MB/s | Net: ↓3.2MB/s ↑2.0MB/s
```

### Historical Metrics

Query metrics over a time range:

```bash
# Last hour with 5-minute intervals
hyperctl metrics -entity vm-name -type vm \
  -start "1h ago" -interval 5min

# Specific date range
hyperctl metrics -entity vm-name -type vm \
  -start "2024-01-20T10:00:00Z" \
  -end "2024-01-20T18:00:00Z" \
  -interval 15min
```

**Output:**
```json
{
  "entity_name": "vm-name",
  "entity_type": "vm",
  "interval": 300,
  "metrics": [
    {
      "timestamp": "2024-01-20T10:00:00Z",
      "cpu_percent": 42.5,
      "memory_percent": 65.2,
      "disk_read_mbps": 10.5,
      "disk_write_mbps": 7.2,
      "net_rx_mbps": 3.1,
      "net_tx_mbps": 1.9
    },
    {
      "timestamp": "2024-01-20T10:05:00Z",
      "cpu_percent": 45.8,
      "memory_percent": 66.1,
      "disk_read_mbps": 12.3,
      "disk_write_mbps": 8.4,
      "net_rx_mbps": 3.5,
      "net_tx_mbps": 2.2
    }
  ]
}
```

## API Endpoints

### GET /vsphere/metrics

Get real-time or historical metrics.

**Query Parameters:**
- `entity` (required): Entity name (VM, host, or cluster)
- `type` (required): Entity type (`vm`, `host`, `cluster`)
- `realtime` (optional): Get real-time metrics (default: true)
- `start` (optional): Start time for historical data (RFC3339)
- `end` (optional): End time for historical data (RFC3339)
- `interval` (optional): Sample interval in seconds (default: 300)

**Response:**
```json
{
  "entity_name": "vm-name",
  "entity_type": "vm",
  "timestamp": "2024-01-20T10:30:15Z",
  "cpu_percent": 45.2,
  "cpu_usage_mhz": 2400,
  "memory_percent": 68.5,
  "memory_usage_mb": 5536,
  "disk_read_mbps": 12.5,
  "disk_write_mbps": 8.2,
  "net_rx_mbps": 3.4,
  "net_tx_mbps": 2.1
}
```

### WebSocket /vsphere/metrics/stream

Stream real-time metrics via WebSocket.

**Connect:**
```javascript
const ws = new WebSocket('ws://localhost:8080/vsphere/metrics/stream?entity=vm-name&type=vm');

ws.onmessage = (event) => {
  const metrics = JSON.parse(event.data);
  console.log(`CPU: ${metrics.cpu_percent}%`);
};
```

**Message Format:**
```json
{
  "entity_name": "vm-name",
  "entity_type": "vm",
  "timestamp": "2024-01-20T10:30:15Z",
  "cpu_percent": 45.2,
  "memory_percent": 68.5,
  "disk_read_mbps": 12.5,
  "disk_write_mbps": 8.2,
  "net_rx_mbps": 3.4,
  "net_tx_mbps": 2.1
}
```

Messages sent every 20 seconds (vSphere realtime interval).

## Use Cases

### 1. Performance Monitoring

Monitor VM performance during load testing:

```bash
# Start streaming metrics
hyperctl metrics -entity app-server-01 -type vm -watch

# In another terminal, run load tests
# Watch CPU and memory usage in real-time
```

### 2. Capacity Planning

Analyze historical usage patterns:

```bash
# Get last 24 hours of metrics
hyperctl metrics -entity vm-name -type vm \
  -start "24h ago" -interval 1h -json > metrics.json

# Analyze with jq
cat metrics.json | jq '.metrics[] | {time: .timestamp, cpu: .cpu_percent, mem: .memory_percent}'
```

### 3. Troubleshooting

Identify resource bottlenecks:

```bash
# Check all hosts for high CPU usage
for host in $(hyperctl host -op list -json | jq -r '.hosts[].name'); do
  echo "=== $host ==="
  hyperctl metrics -entity $host -type host -realtime | grep "CPU Usage"
done
```

### 4. Dashboard Integration

Build custom dashboards using the WebSocket API:

```html
<!DOCTYPE html>
<html>
<head><title>VM Metrics Dashboard</title></head>
<body>
  <div id="metrics"></div>
  <script>
    const ws = new WebSocket('ws://localhost:8080/vsphere/metrics/stream?entity=vm-name&type=vm');

    ws.onmessage = (event) => {
      const m = JSON.parse(event.data);
      document.getElementById('metrics').innerHTML = `
        <h2>${m.entity_name}</h2>
        <p>CPU: ${m.cpu_percent}%</p>
        <p>Memory: ${m.memory_percent}%</p>
        <p>Disk I/O: ↓${m.disk_read_mbps} MB/s ↑${m.disk_write_mbps} MB/s</p>
        <p>Network: ↓${m.net_rx_mbps} MB/s ↑${m.net_tx_mbps} MB/s</p>
      `;
    };
  </script>
</body>
</html>
```

## Metrics Details

### CPU Metrics

- **cpu_percent**: CPU usage percentage (0-100%)
- **cpu_usage_mhz**: CPU usage in MHz
- Calculated from `cpu.usage.average` counter
- Updated every 20 seconds (realtime interval)

### Memory Metrics

- **memory_percent**: Memory usage percentage (0-100%)
- **memory_usage_mb**: Memory usage in MB
- Calculated from `mem.active.average` counter
- Includes active memory only (excludes ballooning, swapping)

### Disk I/O Metrics

- **disk_read_mbps**: Disk read throughput in MB/s
- **disk_write_mbps**: Disk write throughput in MB/s
- Aggregated across all virtual disks
- Calculated from `disk.read.average` and `disk.write.average` counters

### Network Metrics

- **net_rx_mbps**: Network receive throughput in MB/s
- **net_tx_mbps**: Network transmit throughput in MB/s
- Aggregated across all network adapters
- Calculated from `net.received.average` and `net.transmitted.average` counters

## Performance Considerations

### Real-time Metrics

- vSphere provides real-time metrics with 20-second granularity
- Metrics are averaged over the collection interval
- Low overhead on vCenter Server

### Historical Metrics

- Historical metrics stored with varying granularity:
  - **Level 1**: 20 seconds (last hour)
  - **Level 2**: 5 minutes (last day)
  - **Level 3**: 30 minutes (last week)
  - **Level 4**: 2 hours (last month)
- Querying large time ranges may take several seconds
- Use appropriate interval to limit data points returned

### WebSocket Streaming

- Each WebSocket connection polls vCenter every 20 seconds
- Limit concurrent connections to avoid overloading vCenter
- Connections automatically close when client disconnects

## Error Handling

### Common Errors

**Error: Entity not found**
```
Error: VM 'vm-name' not found
```
**Solution:** Verify entity name and type are correct.

**Error: Metrics unavailable**
```
Error: performance data not available for 'vm-name'
```
**Solution:**
- VM may be powered off
- Performance data collection may be disabled
- Wait a few minutes for metrics to become available

**Error: Invalid time range**
```
Error: start time must be before end time
```
**Solution:** Check time range parameters.

## See Also

- [Host and Cluster Info](HOST_CLUSTER_INFO.md)
- [Event Monitoring](EVENT_MONITORING.md)
- [WebSocket API Reference](../api/WEBSOCKET.md)
