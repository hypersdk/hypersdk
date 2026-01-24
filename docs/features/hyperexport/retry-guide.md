# Connection Retry Guide

## Overview

The HyperSDK connection retry system provides automatic retry with exponential backoff for cloud storage operations and network requests. This helps handle transient network failures, rate limiting, and temporary service unavailability.

---

## Features

### Automatic Retry
- **Exponential Backoff**: Delays increase exponentially between retries (1s → 2s → 4s → 8s...)
- **Jitter**: Random variation in delays to prevent thundering herd
- **Configurable**: Maximum attempts, delays, and multiplier
- **Smart Detection**: Automatically identifies retryable vs non-retryable errors
- **Context-Aware**: Respects context cancellation and timeouts

### Supported Error Types
Automatically retries on:
- **Network Errors**: Connection refused, timeout, unreachable, broken pipe
- **HTTP 5xx Errors**: 500, 502, 503, 504
- **Rate Limiting**: 429 Too Many Requests
- **Cloud Provider Errors**: ThrottlingException, RequestTimeout, SlowDown
- **Transient Failures**: Temporary unavailability

Does **NOT** retry on:
- **Not Found**: 404, NoSuchKey, file not found
- **Permission Denied**: 403, access denied
- **Invalid Input**: 400, validation errors
- **File I/O Errors**: Local file system errors

---

## Configuration

### Default Configuration
```go
config := &RetryConfig{
    MaxAttempts:  3,                  // Maximum retry attempts
    InitialDelay: 1 * time.Second,    // Initial delay between retries
    MaxDelay:     30 * time.Second,   // Maximum delay cap
    Multiplier:   2.0,                // Exponential backoff multiplier
    Jitter:       true,               // Add random jitter
}
```

### Custom Configuration
```go
// Aggressive retry for critical operations
config := &RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 500 * time.Millisecond,
    MaxDelay:     60 * time.Second,
    Multiplier:   2.5,
    Jitter:       true,
}

// Conservative retry for less critical operations
config := &RetryConfig{
    MaxAttempts:  2,
    InitialDelay: 2 * time.Second,
    MaxDelay:     10 * time.Second,
    Multiplier:   2.0,
    Jitter:       false,
}
```

### Cloud Storage Configuration
```go
storageConfig := &CloudStorageConfig{
    Provider:  "s3",
    Bucket:    "my-bucket",
    Region:    "us-east-1",
    AccessKey: "AKIA...",
    SecretKey: "...",

    // Custom retry configuration
    RetryConfig: &RetryConfig{
        MaxAttempts:  5,
        InitialDelay: 1 * time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   2.0,
        Jitter:       true,
    },
}

storage, err := NewS3Storage(storageConfig, log)
```

---

## Usage Examples

### Basic Retry
```go
// Using default retry configuration
err := WithRetry(ctx, func(ctx context.Context, attempt int) error {
    // Your operation here
    return client.Upload(ctx, file)
}, "upload file", log)
```

### Custom Retry
```go
config := &RetryConfig{
    MaxAttempts:  5,
    InitialDelay: 500 * time.Millisecond,
    MaxDelay:     60 * time.Second,
}

retryer := NewRetryer(config, log)

err := retryer.Do(ctx, func(ctx context.Context, attempt int) error {
    log.Info("attempting operation", "attempt", attempt)
    return performOperation()
}, "operation name")
```

### Retry with Result
```go
result, err := retryer.DoWithResult(ctx,
    func(ctx context.Context, attempt int) (interface{}, error) {
        data, err := fetchData()
        if err != nil {
            return nil, err
        }
        return data, nil
    }, "fetch data")

if err != nil {
    log.Error("operation failed", "error", err)
    return err
}

data := result.(MyDataType)
```

### Marking Errors as Non-Retryable
```go
operation := func(ctx context.Context, attempt int) error {
    file, err := os.Open(path)
    if err != nil {
        // File errors are not retryable
        return IsNonRetryable(fmt.Errorf("open file: %w", err))
    }
    defer file.Close()

    // Network operation - let retry logic decide
    return uploadToCloud(file)
}

err := retryer.Do(ctx, operation, "upload")
```

### Custom Retryable Errors
```go
customError := errors.New("custom retryable error")

config := &RetryConfig{
    MaxAttempts:     3,
    InitialDelay:    1 * time.Second,
    RetryableErrors: []error{customError},
}

retryer := NewRetryer(config, log)
```

---

## Cloud Storage Integration

### S3 Upload with Retry
```go
storage, err := NewS3Storage(&CloudStorageConfig{
    Provider:  "s3",
    Bucket:    "backups",
    Region:    "us-east-1",
    AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
    SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),

    // Retry configuration (optional, uses defaults if nil)
    RetryConfig: &RetryConfig{
        MaxAttempts:  3,
        InitialDelay: 1 * time.Second,
        MaxDelay:     30 * time.Second,
    },
}, log)

// Upload will automatically retry on network errors
err = storage.Upload(ctx, "/path/to/export.ovf", "exports/vm1.ovf", progressCallback)
```

### Azure Upload with Retry
```go
storage, err := NewAzureStorage(&CloudStorageConfig{
    Provider:  "azure",
    Bucket:    "container",
    AccessKey: os.Getenv("AZURE_STORAGE_ACCOUNT"),
    SecretKey: os.Getenv("AZURE_STORAGE_KEY"),

    RetryConfig: &RetryConfig{
        MaxAttempts:  5,  // More attempts for Azure
        InitialDelay: 2 * time.Second,
    },
}, log)
```

---

## Retry Behavior

### Exponential Backoff Example

With default configuration (InitialDelay=1s, Multiplier=2.0):

| Attempt | Delay Before | Notes |
|---------|--------------|-------|
| 1       | 0s           | Initial attempt |
| 2       | 1s           | 1s * 2^0 = 1s |
| 3       | 2s           | 1s * 2^1 = 2s |
| 4       | 4s           | 1s * 2^2 = 4s |
| 5       | 8s           | 1s * 2^3 = 8s |

With jitter enabled, each delay has ±25% random variation.

### Jitter Calculation
```
actual_delay = base_delay + random(0, base_delay * 0.25)
```

Example with 4s base delay:
- Minimum: 4s
- Maximum: 5s (4s + 1s jitter)

---

## Best Practices

### 1. Choose Appropriate MaxAttempts
```go
// Critical operations (backups, important uploads)
MaxAttempts: 5

// Standard operations (regular uploads)
MaxAttempts: 3

// Quick operations (metadata updates)
MaxAttempts: 2
```

### 2. Set Reasonable Delays
```go
// Fast-failing operations
InitialDelay: 500 * time.Millisecond
MaxDelay:     5 * time.Second

// Standard operations
InitialDelay: 1 * time.Second
MaxDelay:     30 * time.Second

// Long-running operations
InitialDelay: 2 * time.Second
MaxDelay:     60 * time.Second
```

### 3. Use Context Timeouts
```go
// Set overall timeout for operation including retries
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

err := retryer.Do(ctx, operation, "upload")
```

### 4. Mark Non-Retryable Errors Explicitly
```go
if err := validateInput(data); err != nil {
    return IsNonRetryable(err)  // Don't retry validation errors
}

if err := uploadToCloud(data); err != nil {
    return err  // Let retry logic decide
}
```

### 5. Log Retry Attempts
The retry system automatically logs:
- Warnings on retry with error details
- Info on successful retry
- Errors when max attempts exceeded

```
[2026-01-21 14:30:00] WARN: operation failed, retrying | operation=S3 upload, attempt=1, max_attempts=3, delay=1s, error=connection timeout
[2026-01-21 14:30:01] WARN: operation failed, retrying | operation=S3 upload, attempt=2, max_attempts=3, delay=2s, error=connection timeout
[2026-01-21 14:30:03] INFO: operation succeeded after retry | operation=S3 upload, attempt=3, total_attempts=3
```

---

## Error Detection

### Automatic Retryable Error Patterns

**Network Errors:**
```
connection refused
connection reset
connection timeout
network unreachable
no such host
timeout
TLS handshake timeout
i/o timeout
broken pipe
EOF
```

**HTTP/Cloud Service Errors:**
```
500 Internal Server Error
502 Bad Gateway
503 Service Unavailable
504 Gateway Timeout
429 Too Many Requests
RequestTimeout
ServiceUnavailable
InternalError
SlowDown
ThrottlingException
```

**Cloud Provider Errors:**
```
RequestLimitExceeded
ProvisionedThroughputExceededException
TransactionInProgressException
TooManyRequests
```

### Custom Error Handling
```go
type MyRetryableError struct {
    Err error
}

func (e *MyRetryableError) Error() string {
    return e.Err.Error()
}

// Include in retry config
config := &RetryConfig{
    RetryableErrors: []error{&MyRetryableError{}},
}
```

---

## Performance Considerations

### Memory Usage
- Each retry opens a new file handle (for uploads)
- Progress callbacks reset on each attempt
- Total memory: ~O(file_size) per concurrent operation

### Total Time
Calculate maximum total time:
```
max_time = sum of all delays + (max_attempts * operation_time)

Example (3 attempts, 2s operation):
max_time = (1s + 2s) + (3 * 2s) = 9s
```

### Concurrency
Retry operations can run concurrently:
```go
// Upload multiple files with retry
for _, file := range files {
    go func(f string) {
        storage.Upload(ctx, f, remotePath, nil)
    }(file)
}
```

---

## Monitoring

### Metrics to Track
- **Retry Rate**: `retries / total_operations`
- **Success After Retry**: `successful_retries / total_retries`
- **Max Attempts Exceeded**: `failed_operations / total_operations`
- **Average Retry Count**: `total_retries / operations_with_retries`

### Log Analysis
```bash
# Count retry attempts
grep "retrying" logs.txt | wc -l

# Find operations that failed after max attempts
grep "failed after.*attempts" logs.txt

# Identify most retried operations
grep "retrying" logs.txt | awk '{print $5}' | sort | uniq -c | sort -rn
```

---

## Testing

### Test Transient Failures
```go
func TestRetryOnTransientError(t *testing.T) {
    attempts := 0
    operation := func(ctx context.Context, attempt int) error {
        attempts++
        if attempt < 3 {
            return fmt.Errorf("connection timeout")
        }
        return nil
    }

    err := WithRetry(ctx, operation, "test", log)

    if err != nil {
        t.Errorf("Expected success after retry, got: %v", err)
    }

    if attempts != 3 {
        t.Errorf("Expected 3 attempts, got %d", attempts)
    }
}
```

### Test Non-Retryable Errors
```go
func TestNoRetryOnNotFound(t *testing.T) {
    attempts := 0
    operation := func(ctx context.Context, attempt int) error {
        attempts++
        return IsNonRetryable(fmt.Errorf("file not found"))
    }

    err := WithRetry(ctx, operation, "test", log)

    if attempts != 1 {
        t.Errorf("Expected 1 attempt (no retry), got %d", attempts)
    }
}
```

---

## Troubleshooting

### Issue: Too Many Retries
**Symptom**: Operations take very long to fail

**Solution**:
```go
// Reduce max attempts
RetryConfig{
    MaxAttempts: 2,  // Instead of 5
}

// Add context timeout
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

### Issue: No Retries Happening
**Symptom**: Operations fail immediately

**Possible Causes**:
1. Error marked as non-retryable
2. Error pattern not recognized
3. Context already cancelled

**Debug**:
```go
// Check if error is retryable
retryer := NewRetryer(config, log)
if retryer.isRetryable(err) {
    log.Info("error is retryable")
} else {
    log.Warn("error is NOT retryable", "error", err)
}
```

### Issue: Excessive Delays
**Symptom**: Retries take too long

**Solution**:
```go
// Reduce delays
RetryConfig{
    InitialDelay: 500 * time.Millisecond,  // Instead of 1s
    MaxDelay:     10 * time.Second,        // Instead of 30s
}
```

---

## API Reference

### Types

#### RetryConfig
```go
type RetryConfig struct {
    MaxAttempts     int           // Maximum retry attempts (default: 3)
    InitialDelay    time.Duration // Initial delay (default: 1s)
    MaxDelay        time.Duration // Maximum delay (default: 30s)
    Multiplier      float64       // Backoff multiplier (default: 2.0)
    Jitter          bool          // Add jitter (default: true)
    RetryableErrors []error       // Custom retryable errors
}
```

#### Retryer
```go
type Retryer struct {
    config *RetryConfig
    log    logger.Logger
}
```

### Functions

#### NewRetryer
```go
func NewRetryer(config *RetryConfig, log logger.Logger) *Retryer
```
Creates a new retryer with configuration.

#### Do
```go
func (r *Retryer) Do(ctx context.Context, operation RetryOperation, operationName string) error
```
Executes operation with retry logic.

#### DoWithResult
```go
func (r *Retryer) DoWithResult(ctx context.Context, operation func(ctx context.Context, attempt int) (interface{}, error), operationName string) (interface{}, error)
```
Executes operation with retry and returns result.

#### WithRetry (Helper)
```go
func WithRetry(ctx context.Context, operation RetryOperation, operationName string, log logger.Logger) error
```
Quick helper with default configuration.

#### IsNonRetryable
```go
func IsNonRetryable(err error) error
```
Marks an error as non-retryable.

#### IsRetryable
```go
func IsRetryable(err error) error
```
Marks an error as retryable.

---

## Examples

### Complete Example: VM Export with Retry
```go
package main

import (
    "context"
    "time"
    "hypersdk/logger"
)

func exportVMWithRetry(vmName, outputPath string) error {
    log := logger.New("info")
    ctx := context.Background()

    // Configure cloud storage with retry
    storage, err := NewS3Storage(&CloudStorageConfig{
        Provider:  "s3",
        Bucket:    "vm-backups",
        Region:    "us-east-1",
        AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
        SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),

        RetryConfig: &RetryConfig{
            MaxAttempts:  5,
            InitialDelay: 1 * time.Second,
            MaxDelay:     60 * time.Second,
            Multiplier:   2.0,
            Jitter:       true,
        },
    }, log)

    if err != nil {
        return err
    }
    defer storage.Close()

    // Upload with automatic retry
    remotePath := fmt.Sprintf("exports/%s/%s.ovf",
        time.Now().Format("2006-01-02"), vmName)

    log.Info("starting VM export", "vm", vmName, "remote", remotePath)

    err = storage.Upload(ctx, outputPath, remotePath, func(transferred, total int64) {
        pct := float64(transferred) / float64(total) * 100
        log.Debug("upload progress", "percent", fmt.Sprintf("%.1f%%", pct))
    })

    if err != nil {
        log.Error("export failed", "vm", vmName, "error", err)
        return err
    }

    log.Info("export completed", "vm", vmName, "remote", remotePath)
    return nil
}
```

---

## Summary

The connection retry system provides:
- ✅ **Automatic retry** with exponential backoff
- ✅ **Smart error detection** (retryable vs non-retryable)
- ✅ **Configurable** delays, attempts, and behavior
- ✅ **Context-aware** (respects cancellation)
- ✅ **Well-tested** (15+ unit tests)
- ✅ **Production-ready** integration with cloud storage

Use retry to make your cloud operations resilient against transient failures!
