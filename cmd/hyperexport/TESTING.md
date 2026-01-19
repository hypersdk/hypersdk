# HyperExport Testing Guide

## Overview

This document covers testing for HyperExport, including unit tests, integration tests, and cloud storage testing.

## Test Structure

```
cmd/hyperexport/
├── tui_cloud_test.go                    # Unit tests for cloud TUI
├── tui_cloud_integration_test.go        # Integration tests with real cloud providers
├── testdata/
│   └── cloud_test_config.yaml          # Test configuration
└── TESTING.md                          # This file
```

## Running Tests

### Unit Tests

Unit tests can be run without any external dependencies:

```bash
# Run all unit tests
go test -v ./cmd/hyperexport/

# Run specific test
go test -v -run TestGetConfigSteps ./cmd/hyperexport/

# Run with coverage
go test -v -cover ./cmd/hyperexport/

# Generate coverage report
go test -coverprofile=coverage.out ./cmd/hyperexport/
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require real cloud provider credentials and use build tags:

```bash
# Run all integration tests
go test -tags=integration -v ./cmd/hyperexport/

# Run specific provider integration test
go test -tags=integration -v -run TestS3Integration ./cmd/hyperexport/

# Skip integration tests in CI
go test -v -short ./cmd/hyperexport/
```

### Benchmark Tests

```bash
# Run all benchmarks
go test -bench=. -benchmem ./cmd/hyperexport/

# Run specific benchmark
go test -bench=BenchmarkNewCloudSelectionModel -benchmem ./cmd/hyperexport/

# Run benchmarks with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./cmd/hyperexport/
go tool pprof cpu.prof
```

## Test Coverage

### Unit Tests (`tui_cloud_test.go`)

| Function/Component | Test Coverage | Notes |
|--------------------|---------------|-------|
| `getConfigSteps()` | ✅ 100% | All providers tested |
| `getConfigStep()` | ✅ 100% | All phases tested |
| `cloudProviders` | ✅ 100% | Validation checks |
| `newCloudSelectionModel()` | ✅ 100% | Initialization tests |
| `newCloudCredentialsModel()` | ✅ 100% | All providers |
| `newCloudBrowserModel()` | ✅ 100% | All providers |
| Phase transitions | ✅ 100% | S3, Azure, GCS, SFTP |
| Configuration validation | ✅ 100% | Valid/invalid configs |
| URL generation | ✅ 100% | All provider formats |
| Provider names/icons | ✅ 100% | Display formatting |
| Edge cases | ✅ 100% | Empty, nil, special chars |

**Total Coverage**: ~95% of cloud TUI code

### Integration Tests (`tui_cloud_integration_test.go`)

| Test | Provider | Requirements | Duration |
|------|----------|--------------|----------|
| `TestS3Integration` | AWS S3 | AWS credentials | ~10s |
| `TestAzureIntegration` | Azure | Azure credentials | ~10s |
| `TestGCSIntegration` | GCS | GCP credentials | ~10s |
| `TestSFTPIntegration` | SFTP | SFTP server | ~5s |
| `TestMultiFileUpload` | S3 | AWS credentials | ~15s |
| `TestLargeFileUpload` | S3 | AWS credentials | ~30s |

## Setting Up Test Environments

### AWS S3 Testing

#### Create Test Bucket

```bash
# Create bucket
aws s3 mb s3://hypersdk-test-bucket --region us-east-1

# Set lifecycle policy (auto-delete after 1 day)
cat > lifecycle.json <<EOF
{
  "Rules": [{
    "Id": "DeleteTestFiles",
    "Status": "Enabled",
    "Prefix": "test-",
    "Expiration": {"Days": 1}
  }]
}
EOF

aws s3api put-bucket-lifecycle-configuration \
  --bucket hypersdk-test-bucket \
  --lifecycle-configuration file://lifecycle.json
```

#### Set Environment Variables

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
export TEST_S3_BUCKET="hypersdk-test-bucket"
```

#### Run S3 Tests

```bash
go test -tags=integration -v -run TestS3 ./cmd/hyperexport/
```

### Azure Blob Storage Testing

#### Create Test Container

```bash
# Create resource group (if needed)
az group create --name hypersdk-test --location eastus

# Create storage account
az storage account create \
  --name hypersdktest \
  --resource-group hypersdk-test \
  --location eastus \
  --sku Standard_LRS

# Create container
az storage container create \
  --name vm-backups \
  --account-name hypersdktest

# Get connection string
az storage account show-connection-string \
  --name hypersdktest \
  --resource-group hypersdk-test
```

#### Set Environment Variables

```bash
export AZURE_STORAGE_ACCOUNT="hypersdktest"
export AZURE_STORAGE_KEY="your-account-key"
export TEST_AZURE_CONTAINER="vm-backups"
```

#### Run Azure Tests

```bash
go test -tags=integration -v -run TestAzure ./cmd/hyperexport/
```

### Google Cloud Storage Testing

#### Create Test Bucket

```bash
# Set project
gcloud config set project your-project-id

# Create bucket
gsutil mb -l us-east1 gs://hypersdk-gcs-test/

# Set lifecycle (auto-delete after 1 day)
cat > lifecycle.json <<EOF
{
  "lifecycle": {
    "rule": [{
      "action": {"type": "Delete"},
      "condition": {
        "age": 1,
        "matchesPrefix": ["test-"]
      }
    }]
  }
}
EOF

gsutil lifecycle set lifecycle.json gs://hypersdk-gcs-test/

# Create service account
gcloud iam service-accounts create hypersdk-test \
  --display-name "HyperSDK Test Account"

# Grant permissions
gsutil iam ch serviceAccount:hypersdk-test@PROJECT_ID.iam.gserviceaccount.com:objectAdmin \
  gs://hypersdk-gcs-test

# Generate key file
gcloud iam service-accounts keys create key.json \
  --iam-account hypersdk-test@PROJECT_ID.iam.gserviceaccount.com
```

#### Set Environment Variables

```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/key.json"
export TEST_GCS_BUCKET="hypersdk-gcs-test"
```

#### Run GCS Tests

```bash
go test -tags=integration -v -run TestGCS ./cmd/hyperexport/
```

### SFTP Testing

#### Setup Local SFTP Server (Docker)

```bash
# Start SFTP server
docker run -d \
  --name sftp-test \
  -p 2222:22 \
  -v $(pwd)/testdata/sftp:/home/testuser/upload \
  atmoz/sftp testuser:testpass:::upload

# Verify connection
sftp -P 2222 testuser@localhost
```

#### Set Environment Variables

```bash
export TEST_SFTP_HOST="localhost:2222"
export TEST_SFTP_USERNAME="testuser"
export TEST_SFTP_PASSWORD="testpass"
```

#### Run SFTP Tests

```bash
go test -tags=integration -v -run TestSFTP ./cmd/hyperexport/
```

#### Cleanup

```bash
docker stop sftp-test
docker rm sftp-test
```

## Mock Testing (No External Dependencies)

### Using LocalStack for AWS

```bash
# Start LocalStack
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=s3 \
  localstack/localstack

# Configure AWS CLI for LocalStack
export AWS_ENDPOINT_URL="http://localhost:4566"
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_REGION="us-east-1"

# Create test bucket
aws --endpoint-url=http://localhost:4566 s3 mb s3://test-bucket

# Run tests against LocalStack
TEST_S3_BUCKET="test-bucket" go test -tags=integration -v -run TestS3 ./cmd/hyperexport/

# Cleanup
docker stop localstack
docker rm localstack
```

### Using Azurite for Azure

```bash
# Start Azurite
docker run -d \
  --name azurite \
  -p 10000:10000 \
  mcr.microsoft.com/azure-storage/azurite \
  azurite-blob --blobHost 0.0.0.0

# Use default Azurite credentials
export AZURE_STORAGE_ACCOUNT="devstoreaccount1"
export AZURE_STORAGE_KEY="Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="
export TEST_AZURE_CONTAINER="test-container"

# Run tests
go test -tags=integration -v -run TestAzure ./cmd/hyperexport/

# Cleanup
docker stop azurite
docker rm azurite
```

### Using Fake GCS for Google Cloud Storage

```bash
# Start fake-gcs-server
docker run -d \
  --name fake-gcs \
  -p 4443:4443 \
  fsouza/fake-gcs-server \
  -scheme http

# Configure
export STORAGE_EMULATOR_HOST="http://localhost:4443"
export TEST_GCS_BUCKET="test-bucket"

# Run tests
go test -tags=integration -v -run TestGCS ./cmd/hyperexport/

# Cleanup
docker stop fake-gcs
docker rm fake-gcs
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run unit tests
        run: go test -v -cover ./cmd/hyperexport/

      - name: Upload coverage
        uses: codecov/codecov-action@v3

  integration-tests:
    runs-on: ubuntu-latest
    services:
      localstack:
        image: localstack/localstack
        ports:
          - 4566:4566
        env:
          SERVICES: s3

    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run integration tests
        env:
          AWS_ENDPOINT_URL: http://localhost:4566
          AWS_ACCESS_KEY_ID: test
          AWS_SECRET_ACCESS_KEY: test
          TEST_S3_BUCKET: test-bucket
        run: |
          aws --endpoint-url=http://localhost:4566 s3 mb s3://test-bucket
          go test -tags=integration -v -run TestS3 ./cmd/hyperexport/
```

## Test Data Management

### Generating Test Files

```bash
# Create test files of various sizes
# Small file (1KB)
dd if=/dev/urandom of=testdata/small.bin bs=1K count=1

# Medium file (10MB)
dd if=/dev/urandom of=testdata/medium.bin bs=1M count=10

# Large file (100MB)
dd if=/dev/urandom of=testdata/large.bin bs=1M count=100

# OVF/OVA test files
mkdir -p testdata/test-vm
echo "test OVF content" > testdata/test-vm/vm.ovf
dd if=/dev/urandom of=testdata/test-vm/vm-disk1.vmdk bs=1M count=50
```

### Cleanup Test Data

```bash
# Cleanup S3
aws s3 rm s3://hypersdk-test-bucket/ --recursive --include "test-*"

# Cleanup Azure
az storage blob delete-batch \
  --account-name hypersdktest \
  --source vm-backups \
  --pattern "test-*"

# Cleanup GCS
gsutil -m rm -r gs://hypersdk-gcs-test/test-*

# Cleanup local
rm -rf testdata/*.bin testdata/test-vm
```

## Troubleshooting Tests

### Common Issues

#### "AWS credentials not found"

```bash
# Check credentials are set
echo $AWS_ACCESS_KEY_ID
echo $AWS_SECRET_ACCESS_KEY

# Test AWS CLI access
aws s3 ls

# Export credentials
export AWS_ACCESS_KEY_ID="your-key"
export AWS_SECRET_ACCESS_KEY="your-secret"
```

#### "Azure authentication failed"

```bash
# Check account name and key
echo $AZURE_STORAGE_ACCOUNT
echo $AZURE_STORAGE_KEY

# Test Azure CLI access
az storage container list --account-name $AZURE_STORAGE_ACCOUNT

# Get new key if needed
az storage account keys list \
  --account-name $AZURE_STORAGE_ACCOUNT \
  --resource-group hypersdk-test
```

#### "GCS permission denied"

```bash
# Check service account key
echo $GOOGLE_APPLICATION_CREDENTIALS
cat $GOOGLE_APPLICATION_CREDENTIALS

# Test gcloud access
gsutil ls gs://$TEST_GCS_BUCKET

# Verify service account permissions
gcloud projects get-iam-policy PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:hypersdk-test@"
```

#### "SFTP connection refused"

```bash
# Check SFTP server is running
docker ps | grep sftp-test

# Test connection manually
sftp -P 2222 testuser@localhost

# Check port forwarding
netstat -an | grep 2222
```

#### "Tests timeout"

```bash
# Increase test timeout
go test -timeout 5m -tags=integration -v ./cmd/hyperexport/

# Run specific test with verbose output
go test -tags=integration -v -run TestS3Integration ./cmd/hyperexport/
```

## Performance Testing

### Upload Speed Test

```bash
# Test upload speed to S3
time go test -tags=integration -v -run TestLargeFileUpload ./cmd/hyperexport/

# Measure throughput
# File size: 100MB
# Expected time: ~30s (depends on network)
# Expected speed: ~3-5 MB/s
```

### Concurrent Upload Test

```bash
# Test concurrent uploads
go test -tags=integration -v -run TestMultiFileUpload ./cmd/hyperexport/

# Monitor with:
watch -n 1 'aws s3 ls s3://hypersdk-test-bucket/test-multi-upload/ --recursive | wc -l'
```

### Memory Usage Test

```bash
# Run with memory profiling
go test -tags=integration -memprofile=mem.prof -run TestLargeFileUpload ./cmd/hyperexport/

# Analyze memory usage
go tool pprof mem.prof
```

## Test Maintenance

### Updating Tests

When adding new cloud providers:

1. Add provider to `cloudProviders` in `tui_cloud.go`
2. Add test cases to `TestGetConfigSteps`
3. Add test cases to `TestCloudConfigPhaseTransitions`
4. Create integration test `TestXXXIntegration`
5. Update this documentation

### Test Coverage Goals

- Unit tests: > 90% coverage
- Integration tests: All cloud providers
- Edge cases: All error paths tested
- Performance: Benchmarks for critical paths

## Reporting Issues

When reporting test failures, include:

1. Test output: `go test -v output`
2. Environment: OS, Go version, cloud provider
3. Configuration: Sanitized credentials
4. Steps to reproduce
5. Expected vs actual behavior

Example:

```bash
go test -v -run TestS3Integration ./cmd/hyperexport/ 2>&1 | tee test-output.log
```

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [AWS SDK for Go Testing](https://aws.github.io/aws-sdk-go-v2/docs/testing/)
- [Azure SDK Testing](https://github.com/Azure/azure-sdk-for-go/blob/main/documentation/testing.md)
- [Google Cloud Go Client Testing](https://github.com/googleapis/google-cloud-go/blob/main/testing.md)
- [Testify Framework](https://github.com/stretchr/testify) (if using)
