# HyperSDK Examples Index

Complete collection of ready-to-use examples for HyperSDK.

## 📁 Directory Structure

```
examples/
├── python/                   # Python SDK examples
│   ├── simple_export.py     # Basic VM export
│   ├── incremental_backup.py # CBT-based backups
│   └── cloud_cost_comparison.py # Cost analysis
├── typescript/              # TypeScript SDK examples (coming soon)
├── bash/                    # Shell script examples
│   └── export_vm.sh        # Basic export using curl
└── README.md                # hyperctl configuration examples
```

## 🐍 Python Examples

Full-featured Python examples using the HyperSDK Python client library.

### 1. simple_export.py

**Purpose**: Basic VM export with progress monitoring

**Usage**:
```bash
python examples/python/simple_export.py \
    --vm /datacenter/vm/my-vm \
    --output /exports \
    --format ova
```

**Learn**: Job submission, progress monitoring, error handling

---

### 2. incremental_backup.py

**Purpose**: Set up and perform CBT-based incremental backups

**Usage**:
```bash
python examples/python/incremental_backup.py \
    --vm /datacenter/vm/production-db \
    --output /backups
```

**Features**:
- Automatic CBT enablement
- Savings analysis
- 95% faster than full backups
- Timestamp-based backup folders

---

### 3. cloud_cost_comparison.py

**Purpose**: Compare cloud storage costs across providers

**Usage**:
```bash
python examples/python/cloud_cost_comparison.py \
    --disk-size 500 \
    --duration 365 \
    --format ova
```

**Output**: Detailed cost comparison, recommendations, yearly projections

---

## 🔧 Bash Examples

Simple shell scripts using the REST API directly.

### 1. export_vm.sh

**Purpose**: Basic export using curl and jq

**Usage**:
```bash
chmod +x examples/bash/export_vm.sh
./examples/bash/export_vm.sh /datacenter/vm/my-vm /exports
```

**Dependencies**: curl, jq

---

## 📘 TypeScript Examples

Coming soon! Will include:
- Simple VM export
- Batch operations
- Cost analysis
- Advanced scheduling

---

## 🎓 Learning Path

### Beginner

1. Start with `bash/export_vm.sh` to understand the API
2. Try `python/simple_export.py` for structured code
3. Read the inline comments and error handling

### Intermediate

1. Explore `python/incremental_backup.py` for CBT
2. Run `python/cloud_cost_comparison.py` for cost analysis
3. Modify examples for your use cases

### Advanced

1. Combine examples (incremental + cost optimization)
2. Build automation workflows
3. Integrate with CI/CD pipelines
4. Create custom SDKs for other languages

---

## 🚀 Quick Start

### Prerequisites

```bash
# Python examples
pip install hypersdk

# Bash examples
sudo apt-get install curl jq  # or equivalent for your OS
```

### Start HyperSDK

```bash
# Docker
docker run -d -p 8080:8080 \
    -e GOVC_URL='https://vcenter/sdk' \
    -e GOVC_USERNAME='admin' \
    -e GOVC_PASSWORD='pass' \
    hypersdk/hypervisord

# Verify
curl http://localhost:8080/health
```

### Run Your First Example

```bash
# Python
python examples/python/simple_export.py \
    --vm /datacenter/vm/test-vm \
    --output /exports

# Bash
./examples/bash/export_vm.sh /datacenter/vm/test-vm /exports
```

---

## 📚 Full Documentation

- [Quick Start Guide](../docs/QUICK_START.md)
- [Features Overview](../docs/FEATURES_OVERVIEW.md)
- [Python SDK Docs](../sdk/python/README.md)
- [TypeScript SDK Docs](../sdk/typescript/README.md)
- [API Reference](../docs/API_ENDPOINTS.md)

---

## 🤝 Contributing Examples

Have a useful example? We'd love to include it!

1. Create your example following the template
2. Add documentation
3. Test with real HyperSDK
4. Submit a PR

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

---

*For detailed usage of each example, see the individual README files in each directory.*
