# Testing Documentation

This directory contains all testing-related documentation for HyperSDK.

## 📊 Current Test Status

**Total Tests:** 340+ tests
**Daemon API Coverage:** 40.8%
**Status:** ✅ All tests passing
**Last Updated:** 2026-01-26

## Contents

### Overview & Guides
- **[00-testing-overview.md](00-testing-overview.md)** - Comprehensive test overview and structure
- **[testing-guide.md](testing-guide.md)** - Practical testing guide with examples
- **[bug-fixes-and-tests.md](bug-fixes-and-tests.md)** - Bug fixes and test cases documentation

### Component-Specific Testing
- **[dashboard-testing.md](dashboard-testing.md)** - Web dashboard testing
- **[hyperexport-testing.md](hyperexport-testing.md)** - Hyperexport testing guide
- **[hyperexport-quicktest.md](hyperexport-quicktest.md)** - Quick test procedures

## Quick Start

### Run All Tests
```bash
go test ./...
```

### Run with Coverage
```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./daemon/api

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

### Run Specific Package Tests
```bash
# API handlers
go test -v ./daemon/api

# Libvirt handlers only
go test -v ./daemon/api -run TestHandle.*Libvirt

# Jobs package
go test -v ./daemon/jobs
```

### Run with Verbose Output
```bash
go test -v ./daemon/api
```

## Test Organization

### daemon/api Package (340+ tests)
- **auth_handlers_test.go** (8 tests) - Authentication endpoints
- **backup_handlers_test.go** (25 tests) - Backup operations
- **batch_handlers_test.go** (15 tests) - Batch operations
- **cloud_handlers_test.go** (14 tests) - Cloud provider integrations
- **clone_handlers_test.go** (6 tests) - VM cloning
- **config_generator_test.go** (8 tests) - Configuration generation
- **console_handlers_test.go** (18 tests) - Console access (VNC/Serial)
- **cost_handlers_test.go** (11 tests) - Cost tracking
- **hyper2kvm_integration_test.go** (15 tests) - VM conversion
- **iso_handlers_test.go** (13 tests) - ISO management
- **libvirt_handlers_test.go** (32 tests) - Libvirt operations
- **progress_handlers_test.go** (15 tests) - Job progress tracking
- **server_handlers_test.go** (33 tests) - Server endpoints

### daemon/jobs Package
- **jobs_test.go** (27 tests) - Job management and deep copy

## Coverage by Component

### High Coverage (80-100%)
✅ Authentication (100%)
✅ Cloud integrations (100%)
✅ Cost tracking (100%)
✅ Helper functions (92-100%)
✅ Libvirt snapshots (85-93%)
✅ Console operations (92-100%)

### Medium Coverage (40-79%)
⚠️ Console info (54.5%)
⚠️ VNC proxy (56.2%)
⚠️ Serial device (46.2%)
⚠️ Clone/template deployment (44.8%)

### Low Coverage (<40%)
❌ Backup operations (17-29%)
❌ Batch operations (15%)
❌ Clone operations (0-10%)
❌ Workflow handlers (0%)
❌ Network management (0%)

## Adding New Tests

### 1. Create Test File
Create `*_test.go` file in the same package:
```go
package api

import (
    "testing"
    "net/http/httptest"
)

func TestHandleNewFeature(t *testing.T) {
    server := setupTestBasicServer(t)
    // test implementation
}
```

### 2. Follow Test Patterns
- **Method validation**: Test for MethodNotAllowed
- **Invalid JSON**: Test with malformed request body
- **Missing parameters**: Test with incomplete requests
- **Valid requests**: Test successful operations
- **Error cases**: Test edge cases and error conditions

### 3. Run New Tests
```bash
go test -v ./daemon/api -run TestHandleNewFeature
```

### 4. Check Coverage
```bash
go test -coverprofile=coverage.out ./daemon/api
go tool cover -func=coverage.out | grep handleNewFeature
```

## CI/CD Integration

### GitHub Actions
Tests run automatically on:
- Every push to main branch
- Every pull request
- Manual workflow dispatch

### Pre-commit Hooks
```bash
# Install pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
go test ./...
if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi
EOF
chmod +x .git/hooks/pre-commit
```

## Troubleshooting

### Tests Timeout
Increase timeout for slow tests:
```bash
go test -timeout 30s ./daemon/api
```

### Race Conditions
Run with race detector:
```bash
go test -race ./daemon/api
```

### Verbose Debugging
Enable verbose output:
```bash
go test -v ./daemon/api 2>&1 | tee test-output.log
```

## Related Documentation

- **[Test Results](../test-results.md)** - Current test coverage and detailed results
- **[Development Notes](../development/)** - Implementation notes and summaries
- **[API Documentation](../api/)** - API endpoint reference

## Contributing

When contributing tests:
1. Follow existing test patterns
2. Test both success and error paths
3. Use descriptive test names
4. Add comments for complex test cases
5. Ensure tests are deterministic
6. Update coverage metrics in test-results.md

## License

Test code is licensed under LGPL-3.0-or-later.
