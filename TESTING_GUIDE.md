# HyperSDK Testing Guide

## Running All Tests

```bash
# Run all tests
go test ./... -v

# Run tests with coverage
go test ./... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run specific package tests
go test ./providers/vsphere -v
go test ./daemon/jobs -v
go test ./daemon/scheduler -v
```

## Test Organization

### Unit Tests

#### Connection Pool Tests (`providers/vsphere/pool_test.go`)
```bash
# Test connection pool functionality
go test ./providers/vsphere -run TestConnectionPool -v

# Tests cover:
# - Pool creation and initialization
# - Connection statistics
# - Graceful shutdown
# - Context cancellation
# - Default configuration
```

#### OVA/Compression Tests (`providers/vsphere/ova_test.go`)
```bash
# Test OVA creation and compression
go test ./providers/vsphere -run TestCreateOVA -v
go test ./providers/vsphere -run TestExtractOVA -v
go test ./providers/vsphere -run TestValidateOVA -v

# Tests cover:
# - Uncompressed OVA creation
# - Compressed OVA creation (gzip)
# - OVF file ordering (OVF must be first)
# - OVA extraction (compressed and uncompressed)
# - OVA validation
# - Compression levels (1, 6, 9)
```

#### Schedule Persistence Tests (`daemon/scheduler/scheduler_persistence_test.go`)
```bash
# Test schedule persistence
go test ./daemon/scheduler -run TestScheduler_Persistence -v
go test ./daemon/scheduler -run TestScheduler_LoadSchedules -v

# Tests cover:
# - Schedule persistence to SQLite
# - Schedule loading on restart
# - Schedule updates with persistence
# - Schedule deletion with persistence
# - Execution history tracking
# - Multiple schedules management
```

#### Webhook Integration Tests (`daemon/jobs/webhook_integration_test.go`)
```bash
# Test webhook integration
go test ./daemon/jobs -run TestWebhookIntegration -v

# Tests cover:
# - Job created webhook
# - Job cancelled webhook
# - Multiple job webhooks
# - Manager without webhooks (graceful degradation)
# - Real HTTP server integration
```

## Integration Tests

### End-to-End Export with All Features

```bash
# Create test configuration
cat > test-config.yaml <<EOF
vcenter_url: "https://your-vcenter.com"
username: "test@vsphere.local"
password: "password"
insecure: true
database_path: "./test.db"

connection_pool:
  enabled: true
  max_connections: 3

webhooks:
  - url: "http://localhost:9000/webhook"
    events: ["*"]
    enabled: true
EOF

# Start webhook receiver
python3 -m http.server 9000 &

# Start daemon
./hypervisord --config test-config.yaml

# Submit test job
curl -X POST http://localhost:8080/jobs/submit \
  -H "Content-Type: application/json" \
  -d '{
    "vm_path": "/Datacenter/vm/test-vm",
    "output_dir": "/tmp/exports",
    "format": "ova",
    "compress": true,
    "compression_level": 6,
    "cleanup_ovf": true
  }'

# Check webhook was called
# Check OVA was created: ls -lh /tmp/exports/
```

### Connection Pool Test

```bash
# Submit multiple concurrent jobs to test pool
for i in {1..10}; do
  curl -X POST http://localhost:8080/jobs/submit \
    -H "Content-Type: application/json" \
    -d "{\"vm_path\": \"/test/vm$i\", \"output_dir\": \"/tmp\"}" &
done

# Check pool statistics
curl http://localhost:8080/stats/pool
# Should show connection reuse
```

### Schedule Persistence Test

```bash
# Add a schedule
curl -X POST http://localhost:8080/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-schedule",
    "name": "Test Schedule",
    "schedule": "*/5 * * * *",
    "enabled": true,
    "job_template": {
      "vm_path": "/test/vm",
      "output_dir": "/tmp"
    }
  }'

# Restart daemon
pkill hypervisord
./hypervisord --config test-config.yaml

# Verify schedule was restored
curl http://localhost:8080/schedules
# Should include "test-schedule"

# Check database
sqlite3 ./test.db "SELECT * FROM scheduled_jobs;"
```

## Benchmarks

### Connection Pool Benchmark

```go
// Add to pool_test.go
func BenchmarkConnectionPool_Sequential(b *testing.B) {
    cfg := &config.Config{...}
    poolCfg := DefaultPoolConfig()
    log := logger.New("error")
    pool := NewConnectionPool(cfg, poolCfg, log)
    defer pool.Close()

    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        client, _ := pool.Get(ctx)
        pool.Put(client)
    }
}
```

Run benchmarks:
```bash
go test ./providers/vsphere -bench=BenchmarkConnectionPool -benchmem
```

### Compression Benchmark

```go
// Add to ova_test.go
func BenchmarkOVACompression(b *testing.B) {
    tmpDir := b.TempDir()
    // Create test files...

    levels := []int{1, 6, 9}
    for _, level := range levels {
        b.Run(fmt.Sprintf("Level%d", level), func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                CreateOVA(tmpDir, ovaPath, true, level, log)
            }
        })
    }
}
```

## Stress Tests

### High Concurrency Test

```bash
# Test with 50 concurrent jobs
for i in {1..50}; do
  curl -X POST http://localhost:8080/jobs/submit \
    -H "Content-Type: application/json" \
    -d "{\"vm_path\": \"/test/vm$i\"}" &
done

# Monitor:
# - Memory usage: watch -n 1 'ps aux | grep hypervisord'
# - Database locks: sqlite3 test.db ".databases"
# - Pool stats: watch -n 1 'curl -s localhost:8080/stats/pool'
```

### Long-Running Schedule Test

```bash
# Add schedule that runs every minute
curl -X POST http://localhost:8080/schedules \
  -H "Content-Type: application/json" \
  -d '{
    "id": "stress-test",
    "schedule": "* * * * *",
    "enabled": true,
    "job_template": {...}
  }'

# Let run for 1 hour
sleep 3600

# Check execution history
curl http://localhost:8080/schedules/stress-test/history
# Should show ~60 executions
```

## Test Data Cleanup

```bash
# Clean up test databases
rm -f ./test.db ./hypersdk.db

# Clean up test exports
rm -rf /tmp/exports/*

# Reset test environment
pkill hypervisord
pkill python3  # webhook receiver
```

## Continuous Integration

### GitHub Actions Workflow (`.github/workflows/test.yml`)

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install dependencies
        run: go mod download

      - name: Run tests
        run: go test ./... -v -race -coverprofile=coverage.out

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Test Coverage Goals

| Package | Target Coverage | Current |
|---------|----------------|---------|
| providers/vsphere | 80% | Run tests to check |
| daemon/jobs | 75% | Run tests to check |
| daemon/scheduler | 80% | Run tests to check |
| daemon/webhooks | 70% | Run tests to check |
| daemon/store | 85% | Run tests to check |

Check current coverage:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Troubleshooting Test Failures

### Connection Pool Tests Fail
- **Symptom**: Timeout errors
- **Solution**: Increase timeout in test context
- **Fix**: `ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)`

### OVA Tests Fail
- **Symptom**: "permission denied" errors
- **Solution**: Check temp directory permissions
- **Fix**: Ensure test has write access to temp directory

### Schedule Tests Fail
- **Symptom**: "database is locked"
- **Solution**: Close database connections properly
- **Fix**: Add `defer db.Close()` in all tests

### Webhook Tests Fail
- **Symptom**: Webhooks not received
- **Solution**: Increase wait time for async operations
- **Fix**: Add `time.Sleep(200 * time.Millisecond)` after triggering webhook

## Manual Testing Checklist

- [ ] Connection pool reuses connections across multiple jobs
- [ ] Webhooks fire for all job lifecycle events
- [ ] OVA files are created with OVF first
- [ ] Compressed OVAs are 30-50% smaller
- [ ] Schedules survive daemon restarts
- [ ] Execution history is recorded
- [ ] Pool stats show correct reuse ratio
- [ ] Database grows reasonably with job history
- [ ] Graceful shutdown cleans up all resources
- [ ] Multiple simultaneous exports work correctly

## Performance Metrics

Track these metrics during testing:

```bash
# Memory usage over time
watch -n 5 'ps aux | grep hypervisord | awk "{print \$6}"'

# Database size growth
watch -n 60 'ls -lh ./hypersdk.db'

# Connection pool efficiency
watch -n 10 'curl -s localhost:8080/stats/pool | jq .reuse_ratio'

# Job completion rate
watch -n 30 'curl -s localhost:8080/stats | jq .completed_jobs'
```

## Test Documentation

All tests include:
- Clear test names describing what is being tested
- Setup and teardown using `t.TempDir()` for isolation
- Descriptive error messages
- Edge case coverage
- Thread safety verification where applicable

Example:
```go
func TestFeature_SpecificScenario(t *testing.T) {
    // Arrange: Set up test data
    tmpDir := t.TempDir()
    testData := createTestData()

    // Act: Execute the code being tested
    result, err := FeatureUnderTest(testData)

    // Assert: Verify results
    if err != nil {
        t.Fatalf("Unexpected error: %v", err)
    }

    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```
