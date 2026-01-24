# Testing Guide

## Overview

HyperSDK includes comprehensive test suites for all major components. Tests are organized by layer (provider, API, CLI) and use standard Go testing tools.

## Test Structure

```
hypersdk/
├── providers/
│   ├── vsphere/
│   │   ├── infrastructure_test.go    # Host/cluster/vCenter tests
│   │   ├── performance_test.go       # Metrics collection tests
│   │   ├── clone_test.go             # VM cloning tests
│   │   ├── resource_pools_test.go    # Resource pool management tests
│   │   ├── events_test.go            # Event monitoring tests
│   │   └── export_test.go            # VM export tests (existing)
│   └── common/
│       └── pipeline_test.go          # Pipeline integration tests
├── daemon/
│   └── api/
│       ├── vsphere_handlers_test.go  # API handler tests (planned)
│       └── websocket_test.go         # WebSocket streaming tests (existing)
└── cmd/
    └── hyperctl/
        └── vsphere_commands_test.go  # CLI command tests (planned)
```

## Running Tests

### All Tests

```bash
go test ./... -v
```

### Provider Tests

```bash
# All vSphere provider tests
go test ./providers/vsphere/... -v

# Specific test file
go test ./providers/vsphere/infrastructure_test.go -v

# Specific test function
go test ./providers/vsphere -run TestListHosts -v
```

### Pipeline Tests

```bash
go test ./providers/common/pipeline_test.go -v
```

### Skip Integration Tests

Many tests are marked with `t.Skip()` as they require a running vCenter or specific setup:

```bash
# Run all non-skipped tests
go test ./providers/vsphere/... -v -short
```

## Test Categories

### 1. Unit Tests

Test individual functions and methods:

```go
func TestHostInfoValidation(t *testing.T) {
    hostInfo := HostInfo{
        Name:       "test-host",
        CPUCores:   16,
        MemoryMB:   65536,
    }

    assert.NotEmpty(t, hostInfo.Name)
    assert.Greater(t, hostInfo.CPUCores, int32(0))
}
```

### 2. Integration Tests

Test with vCenter simulator or real vCenter:

```go
func TestListHosts(t *testing.T) {
    client, cleanup := setupTestClient(t)
    defer cleanup()

    hosts, err := client.ListHosts(ctx, "*")
    require.NoError(t, err)
    assert.NotEmpty(t, hosts)
}
```

**Note:** Integration tests require:
- govmomi simulator (for mock testing)
- Or real vCenter connection (configured via environment variables)

### 3. End-to-End Tests

Test complete workflows:

```bash
# Test export + pipeline + libvirt
./test_e2e_migration.sh
```

## Test Helpers

### Mock Logger

```go
type mockLogger struct {
    infoCalls  []string
    warnCalls  []string
    errorCalls []string
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
    m.infoCalls = append(m.infoCalls, msg)
}
```

### Test Client Setup

```go
func setupTestClient(t *testing.T) (*VSphereClient, func()) {
    // Uses vcsim (vCenter simulator) for testing
    model := simulator.VPX()
    err := model.Create()
    // ...

    cleanup := func() {
        model.Remove()
    }

    return client, cleanup
}
```

## Test Coverage

### Current Coverage

Run coverage analysis:

```bash
# Generate coverage report
go test ./providers/vsphere/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View in browser
firefox coverage.html
```

### Coverage Goals

- **Unit tests**: >80% coverage
- **Integration tests**: Core functionality covered
- **Critical paths**: 100% coverage (export, pipeline, API)

## Testing Pipeline Integration

### Pipeline Unit Tests

```go
func TestPipelineExecutor(t *testing.T) {
    config := &Hyper2KVMConfig{
        Enabled:       true,
        Hyper2KVMPath: "/usr/bin/hyper2kvm",
        ManifestPath:  "/tmp/manifest.json",
    }

    executor := NewPipelineExecutor(config, logger)
    assert.NotNil(t, executor)
}
```

### Pipeline Integration Tests

```bash
# Requires hyper2kvm and libvirt installed
go test ./providers/common/pipeline_test.go \
  -run TestExecute -v
```

## Testing Web Dashboard

### Frontend Tests

```bash
cd web/dashboard-react
npm test
```

### API Integration Tests

```bash
# Start daemon in test mode
hyperd --test-mode &

# Run API tests
go test ./daemon/api/... -v

# Stop daemon
pkill hyperd
```

## Continuous Integration

### GitHub Actions

`.github/workflows/test.yml`:

```yaml
name: Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v3
        with:
          go-version: '1.24'

      - name: Run tests
        run: go test ./... -v -short

      - name: Coverage
        run: |
          go test ./... -coverprofile=coverage.out
          go tool cover -func=coverage.out
```

## Manual Testing

### Test Scenarios

**1. Host Information**
```bash
hyperctl host -op list
hyperctl host -op info -name esxi-host-01
```

**2. Performance Metrics**
```bash
hyperctl metrics -entity test-vm -type vm -realtime
hyperctl metrics -entity test-vm -type vm -watch
```

**3. VM Cloning**
```bash
hyperctl clone -source template -target test-clone
```

**4. Pipeline Integration**
```bash
hyperexport --vm test-vm --output /tmp/test \
  --manifest --pipeline --libvirt
```

**5. Resource Pools**
```bash
hyperctl pool -op create -name test-pool \
  -cpu-reserve 1000 -mem-reserve 2048
```

**6. Event Monitoring**
```bash
hyperctl events -since 1h
hyperctl events -follow
```

## Debugging Tests

### Verbose Output

```bash
go test ./providers/vsphere/... -v -run TestListHosts
```

### Test Logging

```bash
# Enable detailed logging
export HYPERSDK_LOG_LEVEL=debug
go test ./providers/vsphere/... -v
```

### Race Detection

```bash
go test ./... -race
```

### Memory Profiling

```bash
go test ./providers/vsphere -memprofile=mem.prof
go tool pprof mem.prof
```

## Known Issues

### vcsim Limitations

The vCenter Simulator (vcsim) has limitations:
- Limited performance metrics support
- Some advanced features not implemented
- May not match real vCenter behavior exactly

**Solution:** Use real vCenter for critical integration tests

### Test Timeouts

Some tests may timeout on slow systems:

```go
// Increase timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
```

### Mock vs Real vCenter

Tests marked with `t.Skip()` require real vCenter:

```go
func TestRealVCenter(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    // Test with real vCenter
}
```

Run integration tests:
```bash
# Export vCenter credentials
export VCENTER_URL=vcenter.example.com
export VCENTER_USERNAME=administrator@vsphere.local
export VCENTER_PASSWORD=password

# Run integration tests
go test ./providers/vsphere/... -v
```

## Contributing Tests

When adding new features, include tests:

1. **Unit tests** - Test functions in isolation
2. **Integration tests** - Test with vcsim or real vCenter
3. **Documentation** - Update this guide
4. **Examples** - Add usage examples

### Test Template

```go
func TestNewFeature(t *testing.T) {
    // Setup
    client, cleanup := setupTestClient(t)
    defer cleanup()

    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:    "valid input",
            input:   validInput,
            want:    expectedOutput,
            wantErr: false,
        },
        {
            name:    "invalid input",
            input:   invalidInput,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := client.NewFeature(ctx, tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## See Also

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [testify Assertion Library](https://github.com/stretchr/testify)
- [govmomi Simulator](https://github.com/vmware/govmomi/tree/master/simulator)
