# Retry Mechanism with Exponential Backoff

**Version**: 1.0
**Last Updated**: 2026-01-21

---

## Overview

The retry mechanism provides intelligent retry logic with exponential backoff, smart error detection, and optional network-aware pausing for all HyperSDK operations.

---

## Features

### 1. Exponential Backoff

Delays increase exponentially to avoid overwhelming failing services:

```
Attempt 1: Fail → Wait 1s
Attempt 2: Fail → Wait 2s
Attempt 3: Fail → Wait 4s
Attempt 4: Fail → Wait 8s
Attempt 5: Fail → Wait 16s (capped at MaxDelay)
```

**Formula**: `delay = InitialDelay × (Multiplier ^ (attempt - 1))`

### 2. Jitter

Random variation (±25%) prevents thundering herd:

```
Without Jitter:
  Client 1: Retry at exactly 2.0s
  Client 2: Retry at exactly 2.0s
  Client 3: Retry at exactly 2.0s
  → All hit server simultaneously

With Jitter:
  Client 1: Retry at 1.8s
  Client 2: Retry at 2.2s
  Client 3: Retry at 1.9s
  → Spread out, less server load
```

### 3. Smart Error Detection

Automatically determines if errors are retryable:

**Retryable Errors**:
- Network errors (connection refused, timeout, unreachable)
- HTTP 5xx (500, 502, 503, 504)
- HTTP 429 (Too Many Requests)
- Cloud provider throttling (ThrottlingException, RequestTimeout)
- Transient I/O errors

**Non-Retryable Errors**:
- HTTP 4xx (except 429)
- File not found (404, ENOENT)
- Permission denied (403, EPERM)
- Invalid arguments
- Explicitly marked non-retryable

### 4. Network-Aware Retry (Optional)

Integrates with network monitor to pause during outages:

```go
retryConfig := &retry.RetryConfig{
    WaitForNetwork: true,  // Enable network awareness
}

retryer := retry.NewRetryer(retryConfig, log)
retryer.SetNetworkMonitor(monitor)

// Operations pause when network goes down
// Resume automatically when network recovers
```

---

## Configuration

### Basic Configuration

```go
config := &retry.RetryConfig{
    MaxAttempts:  3,                    // Max 3 attempts
    InitialDelay: 1 * time.Second,     // Start with 1s
    MaxDelay:     30 * time.Second,    // Cap at 30s
    Multiplier:   2.0,                 // Double each time
    Jitter:       true,                // Add randomness
}
```

### Conservative (Few Retries, Short Delays)

```go
config := &retry.RetryConfig{
    MaxAttempts:  2,
    InitialDelay: 500 * time.Millisecond,
    MaxDelay:     5 * time.Second,
    Multiplier:   2.0,
    Jitter:       true,
}
```

**Use Case**: Quick operations, fail fast

### Balanced (Default)

```go
config := &retry.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: 1 * time.Second,
    MaxDelay:     30 * time.Second,
    Multiplier:   2.0,
    Jitter:       true,
}
```

**Use Case**: General purpose, production environments

### Aggressive (Many Retries, Patient)

```go
config := &retry.RetryConfig{
    MaxAttempts:  10,
    InitialDelay: 1 * time.Second,
    MaxDelay:     60 * time.Second,
    Multiplier:   2.0,
    Jitter:       true,
}
```

**Use Case**: Operations that must succeed, unreliable networks

### With Network Awareness

```go
config := &retry.RetryConfig{
    MaxAttempts:    5,
    InitialDelay:   1 * time.Second,
    MaxDelay:       30 * time.Second,
    Multiplier:     2.0,
    Jitter:         true,
    WaitForNetwork: true,  // Pause during network outages
}

retryer := retry.NewRetryer(config, log)
retryer.SetNetworkMonitor(monitor)
```

**Use Case**: Long-running operations, network-dependent tasks

---

## API Reference

### Types

```go
type RetryConfig struct {
    MaxAttempts     int           // Maximum retry attempts (default: 3)
    InitialDelay    time.Duration // Initial delay (default: 1s)
    MaxDelay        time.Duration // Maximum delay cap (default: 30s)
    Multiplier      float64       // Exponential multiplier (default: 2.0)
    Jitter          bool          // Enable jitter (default: true)
    RetryableErrors []error       // Custom retryable errors
    WaitForNetwork  bool          // Wait for network recovery (default: false)
}

type RetryOperation func(ctx context.Context, attempt int) error
```

### Functions

```go
// Create retryer with config
func NewRetryer(config *RetryConfig, log logger.Logger) *Retryer

// Attach network monitor (optional)
func (r *Retryer) SetNetworkMonitor(monitor NetworkMonitor)

// Execute operation with retry
func (r *Retryer) Do(ctx context.Context, operation RetryOperation, name string) error

// Execute and return result
func (r *Retryer) DoWithResult(ctx context.Context,
    operation func(ctx context.Context, attempt int) (interface{}, error),
    name string) (interface{}, error)

// Mark error as non-retryable
func IsNonRetryable(err error) error
```

---

## Usage Examples

### Basic Usage

```go
retryer := retry.NewRetryer(nil, log) // Use defaults

err := retryer.Do(ctx, func(ctx context.Context, attempt int) error {
    if attempt > 1 {
        log.Info("retrying operation", "attempt", attempt)
    }

    return performOperation()
}, "operation name")
```

### With Result

```go
result, err := retryer.DoWithResult(ctx, func(ctx context.Context, attempt int) (interface{}, error) {
    data, err := fetchData()
    if err != nil {
        return nil, err
    }

    return data, nil
}, "fetch data")

if err != nil {
    log.Error("failed to fetch data", "error", err)
    return
}

data := result.(MyDataType)
```

### Non-Retryable Errors

```go
err := retryer.Do(ctx, func(ctx context.Context, attempt int) error {
    data, err := os.Open(filename)
    if err != nil {
        if os.IsNotExist(err) {
            // File not found - don't retry
            return retry.IsNonRetryable(err)
        }
        // Other errors - retry
        return err
    }

    return processData(data)
}, "process file")
```

### Custom Retryable Errors

```go
config := &retry.RetryConfig{
    RetryableErrors: []error{
        MyCustomTransientError,
        MyTemporaryError,
    },
}

retryer := retry.NewRetryer(config, log)
```

### Context with Timeout

```go
// Operation will retry for up to 5 minutes total
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

err := retryer.Do(ctx, operation, "name")
// Returns context.DeadlineExceeded if timeout reached
```

---

## Integration with Providers

### vSphere Provider

All vSphere operations use retry automatically:

```go
client, _ := vsphere.NewVSphereClient(ctx, config, log)

// VM lookup with retry
result, err := client.ExportOVF(ctx, vmPath, opts)

// Automatically retries:
// - VM lookup
// - Export lease creation
// - Lease wait
// - Lease completion
// - File downloads (with resume)
```

### Cloud Storage (S3, Azure, GCS, SFTP)

All cloud operations use retry automatically:

```go
storage, _ := NewS3Storage(config, log)

// Upload with retry
err := storage.Upload(ctx, localPath, remotePath, progress)

// Automatically retries:
// - Upload
// - Download
// - List
// - Delete
// - Exists check
```

---

## Logging

### Success After Retry

```
[INFO] operation succeeded after retry | operation=S3 upload, attempt=3, total_attempts=5
```

### Non-Retryable Error

```
[WARN] operation failed with non-retryable error | operation=download, attempt=1, error=file not found
```

### Max Attempts Exceeded

```
[ERROR] operation failed after max attempts | operation=upload, attempts=5, error=connection timeout
```

### Retry Attempt

```
[WARN] operation failed, retrying | operation=export, attempt=2, max_attempts=5, delay=2s, error=connection refused
```

### Network Wait

```
[WARN] network is down, waiting for network to recover | operation=upload, attempt=2
[INFO] network recovered, retrying operation | operation=upload, attempt=2
```

---

## Best Practices

### 1. Always Use Context

```go
// Good: Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
err := retryer.Do(ctx, operation, "name")

// Bad: No timeout
ctx := context.Background()
err := retryer.Do(ctx, operation, "name")
// Could wait forever if network never recovers
```

### 2. Mark Non-Retryable Errors

```go
// Good: Mark validation errors as non-retryable
if err := validateInput(data); err != nil {
    return retry.IsNonRetryable(err)
}

// Bad: Let validation errors retry
if err := validateInput(data); err != nil {
    return err  // Will retry validation failures
}
```

### 3. Log Retry Attempts

```go
err := retryer.Do(ctx, func(ctx context.Context, attempt int) error {
    if attempt > 1 {
        log.Info("retrying", "attempt", attempt)
    }

    return operation()
}, "operation")
```

### 4. Choose Appropriate Config

```go
// For user-facing operations (fail fast)
config := &retry.RetryConfig{
    MaxAttempts:  2,
    InitialDelay: 500 * time.Millisecond,
}

// For background jobs (be patient)
config := &retry.RetryConfig{
    MaxAttempts:  10,
    InitialDelay: 2 * time.Second,
}
```

### 5. Enable Network Awareness for Long Operations

```go
// For long-running operations like VM exports
config := &retry.RetryConfig{
    MaxAttempts:    5,
    WaitForNetwork: true,
}
retryer.SetNetworkMonitor(monitor)
```

---

## Performance

### Overhead

| Operation | Latency | Notes |
|-----------|---------|-------|
| Success (no retry) | < 1 µs | Minimal wrapper overhead |
| Retry decision | < 1 µs | Error pattern matching |
| Network check | 31 ns | Atomic state read |
| Backoff calculation | < 100 ns | Math operations |

### Memory

- Retryer struct: ~200 bytes
- Per-operation: 0 allocations in success path
- Error wrapping: 1 allocation per failure

---

## Testing

### Testing with Retry

```go
func TestWithRetry(t *testing.T) {
    log := logger.NewTestLogger(t)
    retryer := retry.NewRetryer(nil, log)

    attempts := 0
    err := retryer.Do(context.Background(), func(ctx context.Context, attempt int) error {
        attempts++
        if attempts < 3 {
            return fmt.Errorf("transient error")
        }
        return nil
    }, "test operation")

    if err != nil {
        t.Fatalf("expected success, got: %v", err)
    }

    if attempts != 3 {
        t.Errorf("expected 3 attempts, got %d", attempts)
    }
}
```

### Testing Non-Retryable Errors

```go
func TestNonRetryable(t *testing.T) {
    log := logger.NewTestLogger(t)
    retryer := retry.NewRetryer(nil, log)

    attempts := 0
    err := retryer.Do(context.Background(), func(ctx context.Context, attempt int) error {
        attempts++
        return retry.IsNonRetryable(fmt.Errorf("validation error"))
    }, "test operation")

    if err == nil {
        t.Fatal("expected error")
    }

    if attempts != 1 {
        t.Errorf("expected 1 attempt (no retry), got %d", attempts)
    }
}
```

---

## Troubleshooting

### Operations retry forever

**Cause**: No context timeout

**Solution**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()
```

### Validation errors being retried

**Cause**: Not marking errors as non-retryable

**Solution**:
```go
return retry.IsNonRetryable(validationErr)
```

### Too many retry attempts

**Cause**: MaxAttempts too high

**Solution**:
```go
config := &retry.RetryConfig{
    MaxAttempts: 3,  // Reduce from 10
}
```

### Network-aware retry not working

**Cause**: Network monitor not attached or WaitForNetwork disabled

**Solution**:
```go
config.WaitForNetwork = true
retryer.SetNetworkMonitor(monitor)
```

---

## Summary

The retry mechanism provides:

✅ **Exponential backoff** - Intelligent delay increases
✅ **Jitter** - Prevents thundering herd
✅ **Smart error detection** - Automatic retry/no-retry decisions
✅ **Network awareness** - Optional pause during outages
✅ **Context support** - Proper cancellation and timeouts
✅ **Zero overhead** - Fast path when successful

Use retry for all network operations to improve reliability and user experience!
