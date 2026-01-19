# Network Monitoring and Retry Integration

## Overview

The network monitoring system provides real-time detection of network state changes and integrates with the retry mechanism to handle host network failures intelligently.

---

## Features

### 1. Real-time Network State Detection

**Netlink Monitoring** (Linux only):
- Monitors network interface state changes via netlink
- Detects interface up/down events instantly
- Tracks multiple interfaces simultaneously
- Zero polling overhead

**Connectivity Verification**:
- Verifies actual internet connectivity by reaching check hosts
- Configurable check hosts (default: 8.8.8.8, 1.1.1.1, 8.8.4.4)
- Periodic connectivity checks as fallback
- Smart timeout handling

### 2. Network States

- **StateUp**: Network is fully available
- **StateDown**: Network is completely unavailable
- **StateDegraded**: Partial connectivity (some check hosts reachable)
- **StateUnknown**: Initial state before first check

### 3. Integration with Retry Mechanism

When network monitoring is enabled:
- Retry operations **pause** when network goes down
- Automatically **resume** when network recovers
- **No wasted retry attempts** during network outages
- Smart backoff still applies for transient errors

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Network Monitor                       │
│                                                          │
│  ┌────────────────┐         ┌──────────────────────┐   │
│  │ Netlink Events │────────▶│  State Manager        │   │
│  │  (Interface)   │         │  - StateUp/Down       │   │
│  └────────────────┘         │  - Notify Listeners   │   │
│                             └──────────────────────┘   │
│  ┌────────────────┐                    │               │
│  │ Periodic Check │────────────────────┘               │
│  │ (Connectivity) │                                    │
│  └────────────────┘                                    │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                    Retry Mechanism                       │
│                                                          │
│  Operation Fails ──▶ Check Network State                │
│                              │                           │
│                    ┌─────────┴──────────┐               │
│                    │                    │               │
│                Network Up          Network Down         │
│                    │                    │               │
│            Apply Backoff      Wait for Network         │
│                    │                    │               │
│                Retry                    │               │
│                                  Network Recovers       │
│                                         │               │
│                                    Resume Retry         │
└─────────────────────────────────────────────────────────┘
```

---

## Usage

### 1. Basic Network Monitoring

```go
package main

import (
    "context"
    "hypersdk/logger"
    "hypersdk/network"
)

func main() {
    log := logger.New(logger.InfoLevel)

    // Create network monitor with defaults
    monitor := network.NewMonitor(nil, log)

    // Start monitoring
    ctx := context.Background()
    if err := monitor.Start(ctx); err != nil {
        log.Error("failed to start network monitor", "error", err)
        return
    }
    defer monitor.Stop()

    // Check current state
    if monitor.IsUp() {
        log.Info("network is available")
    } else {
        log.Info("network is unavailable")
    }

    // Subscribe to state changes
    stateCh := monitor.Subscribe()
    go func() {
        for state := range stateCh {
            log.Info("network state changed", "state", state)
        }
    }()

    // Wait for network if needed
    if err := monitor.WaitForNetwork(ctx); err != nil {
        log.Error("failed waiting for network", "error", err)
    }
}
```

### 2. Custom Configuration

```go
config := &network.MonitorConfig{
    CheckInterval:   15 * time.Second,      // Check every 15 seconds
    CheckTimeout:    3 * time.Second,       // 3 second timeout per check
    CheckHosts:      []string{"8.8.8.8", "1.1.1.1"},
    NotifyOnChange:  true,                  // Only notify on state change
    EnableNetlink:   true,                  // Enable netlink (Linux)
    PreferredIfaces: []string{"eth0", "wlan0"}, // Monitor specific interfaces
}

monitor := network.NewMonitor(config, log)
```

### 3. Integration with Retry Mechanism

```go
package main

import (
    "context"
    "time"
    "hypersdk/logger"
    "hypersdk/network"
    "hypersdk/retry"
)

func main() {
    log := logger.New(logger.InfoLevel)

    // Create network monitor
    monitor := network.NewMonitor(nil, log)
    ctx := context.Background()
    monitor.Start(ctx)
    defer monitor.Stop()

    // Create retry config with network awareness
    retryConfig := &retry.RetryConfig{
        MaxAttempts:    5,
        InitialDelay:   1 * time.Second,
        MaxDelay:       30 * time.Second,
        Multiplier:     2.0,
        Jitter:         true,
        WaitForNetwork: true, // Enable network-aware retry
    }

    // Create retryer and set network monitor
    retryer := retry.NewRetryer(retryConfig, log)
    retryer.SetNetworkMonitor(monitor)

    // Use retryer for operations
    err := retryer.Do(ctx, func(ctx context.Context, attempt int) error {
        // Your operation here
        return performNetworkOperation()
    }, "my operation")

    if err != nil {
        log.Error("operation failed", "error", err)
    }
}
```

---

## How It Works

### Network Down Scenario

```
1. Operation fails (e.g., connection timeout)
2. Retry mechanism checks: Is error retryable? → Yes
3. Retry mechanism checks: Is network up?
   └─ Network Monitor: No, network is DOWN
4. Retry pauses and waits for network recovery
   └─ No retry attempts consumed
5. Network Monitor detects interface comes back up
6. Network Monitor verifies connectivity to check hosts
7. Network Monitor notifies: Network is UP
8. Retry mechanism resumes operation immediately
   └─ No additional delay needed
9. Operation succeeds or continues normal retry logic
```

### Transient Error Scenario (Network is Up)

```
1. Operation fails (e.g., server timeout)
2. Retry mechanism checks: Is error retryable? → Yes
3. Retry mechanism checks: Is network up?
   └─ Network Monitor: Yes, network is UP
4. Apply exponential backoff delay (1s → 2s → 4s)
5. Retry operation
6. Operation succeeds or continues retry
```

---

## Benefits

### 1. No Wasted Retry Attempts
- When network is down, retries **pause** instead of failing
- Retry attempts preserved for actual transient errors
- Better success rate for long-running operations

### 2. Faster Recovery
- Netlink events detected **instantly** (< 100ms)
- No need to wait for timeout on every retry
- Operations resume as soon as network recovers

### 3. Better Resource Usage
- No CPU/bandwidth wasted on retry attempts during outage
- Operations can wait indefinitely for network recovery
- Context cancellation still works

### 4. Improved User Experience
- Clear logging: "network is down, waiting for recovery"
- Operations don't fail immediately when network drops
- Automatic recovery when network comes back

---

## Configuration Examples

### Conservative (Fast Checks, Shorter Waits)

```yaml
network_monitor:
  check_interval: 5s
  check_timeout: 2s
  check_hosts:
    - "8.8.8.8"
    - "1.1.1.1"
  enable_netlink: true

retry:
  max_attempts: 3
  initial_delay: 1s
  wait_for_network: true
```

**Use Case**: Development, testing, quick failover scenarios

### Balanced (Default)

```yaml
network_monitor:
  check_interval: 10s
  check_timeout: 5s
  check_hosts:
    - "8.8.8.8"
    - "1.1.1.1"
    - "8.8.4.4"
  enable_netlink: true

retry:
  max_attempts: 5
  initial_delay: 1s
  wait_for_network: true
```

**Use Case**: Production deployments, general purpose

### Patient (Slow Checks, Longer Waits)

```yaml
network_monitor:
  check_interval: 30s
  check_timeout: 10s
  check_hosts:
    - "8.8.8.8"
    - "1.1.1.1"
    - "8.8.4.4"
    - "1.0.0.1"
  enable_netlink: true

retry:
  max_attempts: 10
  initial_delay: 2s
  wait_for_network: true
```

**Use Case**: Unreliable networks, satellite links, remote locations

---

## Netlink Integration Details

### How Netlink Works

Netlink is a Linux kernel interface for communication between kernel and userspace:
- **Zero Polling**: Kernel pushes events to userspace
- **Real-time**: Interface state changes detected instantly
- **Low Overhead**: Event-driven, no periodic checks needed
- **Comprehensive**: Captures all network interface events

### Events Monitored

1. **Link Up/Down**: Interface becomes available/unavailable
2. **Link State**: Running, dormant, up, down
3. **Carrier Changes**: Physical link state changes
4. **Address Changes**: IP address added/removed

### Fallback Behavior

If netlink is not available (non-Linux systems):
- Falls back to periodic connectivity checks only
- Still functional, just no real-time event detection
- Slightly higher latency for detecting state changes

---

## Interface Statistics

Get detailed interface statistics:

```go
stats, err := monitor.GetInterfaceStats()
if err != nil {
    log.Error("failed to get interface stats", "error", err)
    return
}

for name, stat := range stats {
    log.Info("interface statistics",
        "name", name,
        "up", stat.IsUp,
        "rxBytes", stat.RxBytes,
        "txBytes", stat.TxBytes,
        "rxErrors", stat.RxErrors,
        "txErrors", stat.TxErrors,
        "mtu", stat.MTU,
        "mac", stat.MACAddress,
    )
}
```

---

## Best Practices

### 1. Always Use Context
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

// Context cancellation works even while waiting for network
err := retryer.Do(ctx, operation, "name")
```

### 2. Monitor Specific Interfaces
```go
config := &network.MonitorConfig{
    PreferredIfaces: []string{"eth0"}, // Only monitor primary interface
}
```

### 3. Choose Appropriate Check Hosts
```go
config := &network.MonitorConfig{
    CheckHosts: []string{
        "your-vcenter-server.com", // Check vCenter connectivity
        "8.8.8.8",                  // Fallback to public DNS
    },
}
```

### 4. Combine with Webhooks
```go
stateCh := monitor.Subscribe()
go func() {
    for state := range stateCh {
        // Send webhook notification
        webhookManager.SendNetworkStateChange(state)
    }
}()
```

---

## Performance Impact

### Overhead

- **Netlink monitoring**: < 0.1% CPU, ~1MB memory
- **Periodic checks**: Negligible (only DNS/TCP connection attempts)
- **State changes**: Atomic operations, no locking overhead

### Benefits

- **Saved retry attempts**: Preserve retries for actual errors
- **Faster recovery**: Detect network up in < 100ms vs seconds
- **Reduced network traffic**: No failed requests during outage

---

## Troubleshooting

### Issue: Network monitor shows "down" but network works

**Cause**: Check hosts unreachable (firewall, DNS, routing)

**Solution**:
```yaml
network_monitor:
  check_hosts:
    - "192.168.1.1"  # Use local gateway
    - "8.8.8.8"       # Keep public DNS as fallback
```

### Issue: Netlink events not detected

**Cause**: Not running on Linux or insufficient permissions

**Solution**:
- Run on Linux for netlink support
- Ensure process has network access
- Check `EnableNetlink` is true

### Issue: Operations hang waiting for network

**Cause**: Network never recovers and context has no timeout

**Solution**:
```go
// Always use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
```

---

## Summary

Network monitoring with netlink provides:
- ✅ **Instant network state detection** via netlink events
- ✅ **Connectivity verification** via check hosts
- ✅ **Smart retry integration** - pause during outages
- ✅ **No wasted retry attempts** during network downtime
- ✅ **Automatic recovery** when network comes back
- ✅ **Zero performance overhead** with event-driven design

**When to Use**:
- Long-running export operations
- Unstable network environments
- Operations that must eventually succeed
- Systems where network can be temporarily unavailable

**When NOT to Use**:
- Very short operations (< 5 seconds)
- Operations with strict time limits
- Environments where network is always stable
