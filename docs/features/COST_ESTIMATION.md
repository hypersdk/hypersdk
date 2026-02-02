# Cost Estimation

HyperSDK provides comprehensive cloud storage cost estimation capabilities to help you make informed decisions before exporting VMs to cloud storage. Get accurate cost projections for AWS S3, Azure Blob Storage, and Google Cloud Storage with detailed breakdowns of storage, transfer, and request costs.

## Table of Contents

- [Overview](#overview)
- [Supported Providers](#supported-providers)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [SDK Examples](#sdk-examples)
- [Use Cases](#use-cases)
- [Cost Breakdown](#cost-breakdown)
- [Provider Comparison](#provider-comparison)
- [Best Practices](#best-practices)

## Overview

The cost estimation feature helps you:

- **Budget Planning**: Estimate costs before committing to cloud exports
- **Provider Selection**: Compare costs across AWS, Azure, and GCP
- **Cost Optimization**: Choose the most cost-effective storage class
- **Long-term Forecasting**: Project yearly costs with monthly breakdowns
- **Export Size Estimation**: Calculate compressed export sizes for different formats

## Supported Providers

### Amazon S3 (AWS)

**Storage Classes:**
- **Standard**: General-purpose storage ($0.023/GB/month)
- **Infrequent Access (IA)**: Infrequently accessed data ($0.0125/GB/month)
- **One Zone IA**: Single availability zone ($0.01/GB/month)
- **Glacier**: Long-term archive ($0.004/GB/month)
- **Glacier Deep Archive**: Lowest-cost archive ($0.00099/GB/month)

**Features:**
- Tiered data transfer pricing
- Early deletion charges for IA/Glacier (30-180 days minimum)
- Retrieval fees for archived data

### Azure Blob Storage

**Storage Classes:**
- **Hot**: Frequently accessed data ($0.0184/GB/month)
- **Cool**: Infrequently accessed data ($0.01/GB/month)
- **Archive**: Long-term archive ($0.002/GB/month)

**Features:**
- First 100GB of egress free per month
- Tiered data transfer pricing
- Early deletion charges (30-180 days minimum)

### Google Cloud Storage (GCS)

**Storage Classes:**
- **Standard**: General-purpose storage ($0.02/GB/month)
- **Nearline**: Access < once/month ($0.01/GB/month)
- **Coldline**: Access < once/quarter ($0.004/GB/month)
- **Archive**: Access < once/year ($0.0012/GB/month)

**Features:**
- Tiered data transfer pricing
- Early deletion charges (30-365 days minimum)
- Retrieval fees for Nearline/Coldline/Archive

## Quick Start

### REST API

Estimate costs for a single provider:

```bash
curl -X POST http://localhost:8080/cost/estimate \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "s3",
    "region": "us-east-1",
    "storage_class": "s3-standard",
    "storage_gb": 500,
    "transfer_gb": 100,
    "requests": 10000,
    "duration_days": 30
  }'
```

Compare costs across all providers:

```bash
curl -X POST http://localhost:8080/cost/compare \
  -H "Content-Type: application/json" \
  -d '{
    "storage_gb": 500,
    "transfer_gb": 100,
    "requests": 10000,
    "duration_days": 30
  }'
```

### Python SDK

```python
from hypersdk import HyperSDK

# Initialize client
client = HyperSDK("http://localhost:8080")

# Estimate cost for S3
estimate = client.estimate_cost(
    provider="s3",
    region="us-east-1",
    storage_class="s3-standard",
    storage_gb=500,
    transfer_gb=100,
    requests=10000,
    duration_days=30
)

print(f"Total Cost: ${estimate['total_cost']:.2f}")
print(f"Storage: ${estimate['breakdown']['storage_cost']:.2f}")
print(f"Transfer: ${estimate['breakdown']['transfer_cost']:.2f}")
```

### TypeScript SDK

```typescript
import { HyperSDK } from 'hypersdk';

// Initialize client
const client = new HyperSDK('http://localhost:8080');

// Estimate cost for Azure
const estimate = await client.estimateCost({
  provider: 'azure_blob',
  region: 'eastus',
  storageClass: 'azure-hot',
  storageGB: 500,
  transferGB: 100,
  requests: 10000,
  durationDays: 30
});

console.log(`Total Cost: $${estimate.total_cost.toFixed(2)}`);
```

## API Reference

### POST /cost/estimate

Estimate costs for a specific cloud provider and storage class.

**Request:**
```json
{
  "provider": "s3|azure_blob|gcs",
  "region": "us-east-1|eastus|us",
  "storage_class": "s3-standard|azure-hot|gcs-standard",
  "storage_gb": 500,
  "transfer_gb": 100,
  "requests": 10000,
  "duration_days": 30
}
```

**Response:**
```json
{
  "provider": "s3",
  "region": "us-east-1",
  "storage_class": "s3-standard",
  "breakdown": {
    "storage_cost": 11.50,
    "transfer_cost": 9.00,
    "request_cost": 0.05,
    "retrieval_cost": 0.00,
    "early_delete_cost": 0.00
  },
  "total_cost": 20.55,
  "currency": "USD",
  "estimated_at": "2026-01-15T10:30:00Z",
  "pricing_version": "2026-01"
}
```

### POST /cost/compare

Compare costs across all supported cloud providers.

**Request:**
```json
{
  "storage_gb": 500,
  "transfer_gb": 100,
  "requests": 10000,
  "duration_days": 30
}
```

**Response:**
```json
{
  "cheapest": "azure_blob",
  "recommended": "azure_blob",
  "savings_vs_expensive": 5.50,
  "estimates": [
    {
      "provider": "s3",
      "total_cost": 20.55
    },
    {
      "provider": "azure_blob",
      "total_cost": 15.05
    },
    {
      "provider": "gcs",
      "total_cost": 22.00
    }
  ]
}
```

### POST /cost/project

Project yearly costs with monthly breakdown.

**Request:**
```json
{
  "provider": "s3",
  "storage_class": "s3-standard",
  "storage_gb": 500,
  "transfer_gb": 100,
  "requests": 10000
}
```

**Response:**
```json
{
  "year": 2026,
  "total_cost": 246.60,
  "monthly_average": 20.55,
  "monthly_breakdown": [
    {
      "month": 1,
      "total_cost": 20.55,
      "breakdown": { ... }
    },
    ...
  ]
}
```

### POST /cost/estimate-size

Estimate the compressed size of a VM export.

**Request:**
```json
{
  "disk_size_gb": 500,
  "format": "ova|qcow2|raw",
  "include_snapshots": false
}
```

**Response:**
```json
{
  "total_disk_size_gb": 500,
  "estimated_export_gb": 350,
  "compression_ratio": 0.7,
  "format": "ova",
  "include_snapshots": false
}
```

## SDK Examples

### Python: Complete Workflow

```python
from hypersdk import HyperSDK

client = HyperSDK("http://localhost:8080")

# 1. Estimate export size
size_estimate = client.estimate_export_size(
    disk_size_gb=500,
    format="ova",
    include_snapshots=False
)
export_size = size_estimate['estimated_export_gb']
print(f"Estimated export size: {export_size:.2f} GB")

# 2. Compare providers
comparison = client.compare_providers(
    storage_gb=export_size,
    transfer_gb=export_size,  # One-time download
    requests=100,
    duration_days=90
)

print(f"\nCheapest provider: {comparison['cheapest']}")
print(f"Savings: ${comparison['savings_vs_expensive']:.2f}")

for estimate in comparison['estimates']:
    print(f"{estimate['provider']}: ${estimate['total_cost']:.2f}")

# 3. Get detailed estimate for cheapest provider
cheapest = comparison['cheapest']
storage_class_map = {
    's3': 's3-glacier',
    'azure_blob': 'azure-archive',
    'gcs': 'gcs-archive'
}

detailed = client.estimate_cost(
    provider=cheapest,
    region='us-east-1' if cheapest == 's3' else 'eastus',
    storage_class=storage_class_map[cheapest],
    storage_gb=export_size,
    transfer_gb=export_size,
    requests=100,
    duration_days=90
)

print(f"\nDetailed cost breakdown for {cheapest}:")
print(f"  Storage:  ${detailed['breakdown']['storage_cost']:.2f}")
print(f"  Transfer: ${detailed['breakdown']['transfer_cost']:.2f}")
print(f"  Requests: ${detailed['breakdown']['request_cost']:.2f}")
print(f"  Total:    ${detailed['total_cost']:.2f}")

# 4. Project yearly costs
projection = client.project_yearly_cost(
    provider=cheapest,
    storage_class=storage_class_map[cheapest],
    storage_gb=export_size,
    transfer_gb=0,  # Keep in archive
    requests=10
)

print(f"\nYearly projection:")
print(f"  Total: ${projection['total_cost']:.2f}")
print(f"  Monthly average: ${projection['monthly_average']:.2f}")
```

### TypeScript: Archive Storage Selection

```typescript
import { HyperSDK } from 'hypersdk';

const client = new HyperSDK('http://localhost:8080');

async function selectArchiveStorage(diskSizeGB: number) {
  // Estimate export size
  const sizeEstimate = await client.estimateExportSize({
    diskSizeGB: diskSizeGB,
    format: 'ova',
    includeSnapshots: false
  });

  console.log(`Export size: ${sizeEstimate.estimated_export_gb} GB`);

  // Compare archive storage classes
  const archiveClasses = [
    {
      provider: 's3',
      storageClass: 's3-glacier-deep-archive',
      region: 'us-east-1'
    },
    {
      provider: 'azure_blob',
      storageClass: 'azure-archive',
      region: 'eastus'
    },
    {
      provider: 'gcs',
      storageClass: 'gcs-archive',
      region: 'us'
    }
  ];

  const estimates = await Promise.all(
    archiveClasses.map(async (config) => {
      const estimate = await client.estimateCost({
        provider: config.provider,
        region: config.region,
        storageClass: config.storageClass,
        storageGB: sizeEstimate.estimated_export_gb,
        transferGB: 0,
        requests: 10,
        durationDays: 365
      });

      return {
        provider: config.provider,
        storageClass: config.storageClass,
        totalCost: estimate.total_cost
      };
    })
  );

  // Sort by cost
  estimates.sort((a, b) => a.totalCost - b.totalCost);

  console.log('\nArchive storage costs (1 year):');
  estimates.forEach((e) => {
    console.log(`  ${e.provider} (${e.storageClass}): $${e.totalCost.toFixed(2)}`);
  });

  return estimates[0]; // Return cheapest
}

// Example usage
selectArchiveStorage(1000).then((cheapest) => {
  console.log(`\nRecommendation: Use ${cheapest.provider} for $${cheapest.totalCost.toFixed(2)}/year`);
});
```

## Use Cases

### 1. Pre-Export Cost Analysis

Before exporting VMs to cloud storage, estimate the total cost:

```python
# Estimate export size
size = client.estimate_export_size(
    disk_size_gb=2000,
    format="ova",
    include_snapshots=True
)

# Compare providers for 1-year storage
comparison = client.compare_providers(
    storage_gb=size['estimated_export_gb'],
    transfer_gb=size['estimated_export_gb'],
    requests=1000,
    duration_days=365
)
```

### 2. Budget Planning

Project costs for the upcoming year:

```python
projection = client.project_yearly_cost(
    provider="s3",
    storage_class="s3-glacier",
    storage_gb=5000,
    transfer_gb=500,
    requests=10000
)

monthly_budget = projection['monthly_average']
print(f"Required monthly budget: ${monthly_budget:.2f}")
```

### 3. Archive vs. Frequent Access

Compare costs for different access patterns:

```python
# Archive scenario: store for 1 year, retrieve once
archive = client.estimate_cost(
    provider="s3",
    region="us-east-1",
    storage_class="s3-glacier-deep-archive",
    storage_gb=1000,
    transfer_gb=1000,  # One-time retrieval
    requests=100,
    duration_days=365
)

# Frequent access: monthly downloads
frequent = client.estimate_cost(
    provider="s3",
    region="us-east-1",
    storage_class="s3-standard",
    storage_gb=1000,
    transfer_gb=12000,  # Monthly downloads
    requests=1200,
    duration_days=365
)

print(f"Archive cost: ${archive['total_cost']:.2f}")
print(f"Frequent access cost: ${frequent['total_cost']:.2f}")
```

### 4. Multi-Region Comparison

Compare costs across different regions (future enhancement):

```python
# Compare US East vs. US West
us_east = client.estimate_cost(
    provider="s3",
    region="us-east-1",
    storage_class="s3-standard",
    storage_gb=1000,
    transfer_gb=100,
    requests=1000,
    duration_days=30
)

# Note: Currently all providers use default regions
# Regional pricing variations coming in future updates
```

## Cost Breakdown

### Storage Costs

Calculated as: `storage_gb × price_per_gb × (duration_days / 30)`

Example:
- 500 GB stored in S3 Standard for 30 days
- 500 × $0.023 × 1 = **$11.50**

### Transfer Costs

Tiered pricing based on total transfer:

**AWS S3:**
- First 10 TB: $0.09/GB
- Next 40 TB: $0.085/GB
- Next 100 TB: $0.07/GB
- Over 150 TB: $0.05/GB

**Azure Blob:**
- First 100 GB: Free
- Next 10 TB: $0.087/GB
- Next 40 TB: $0.083/GB
- Over 50 TB: $0.07/GB

**Google Cloud:**
- First 1 TB: $0.12/GB
- Next 9 TB: $0.11/GB
- Over 10 TB: $0.08/GB

### Request Costs

Estimated based on request type distribution:
- 60% GET requests
- 30% PUT requests
- 10% LIST requests

Example (10,000 requests to S3):
- 6,000 GET: (6000/1000) × $0.0004 = $0.0024
- 3,000 PUT: (3000/1000) × $0.005 = $0.015
- 1,000 LIST: (1000/1000) × $0.005 = $0.005
- **Total: $0.0224**

### Retrieval Costs

Applied to archive storage classes when data is retrieved:

- S3 Glacier: $0.01/GB
- S3 Deep Archive: $0.02/GB
- Azure Archive: $0.02/GB
- GCS Archive: $0.05/GB

### Early Deletion Costs

Minimum storage duration charges:

| Provider | Storage Class | Minimum Duration |
|----------|---------------|------------------|
| S3 | Infrequent Access | 30 days |
| S3 | Glacier | 90 days |
| S3 | Deep Archive | 180 days |
| Azure | Cool | 30 days |
| Azure | Archive | 180 days |
| GCS | Nearline | 30 days |
| GCS | Coldline | 90 days |
| GCS | Archive | 365 days |

Example: Deleting 100GB from S3 Glacier after 30 days:
- Charged for remaining 60 days
- 100 × $0.004 × (60/30) = **$0.80 early deletion fee**

## Provider Comparison

### Cost Comparison Table (500 GB, 30 days)

| Provider | Storage Class | Storage | Transfer (100GB) | Total |
|----------|---------------|---------|------------------|-------|
| AWS S3 | Standard | $11.50 | $9.00 | **$20.50** |
| AWS S3 | Glacier | $2.00 | $10.00* | **$12.00** |
| AWS S3 | Deep Archive | $0.50 | $11.00* | **$11.50** |
| Azure | Hot | $9.20 | $0.00** | **$9.20** |
| Azure | Cool | $5.00 | $1.00 | **$6.00** |
| Azure | Archive | $1.00 | $3.00* | **$4.00** |
| GCS | Standard | $10.00 | $12.00 | **$22.00** |
| GCS | Nearline | $5.00 | $13.00* | **$18.00** |
| GCS | Archive | $0.60 | $17.00* | **$17.60** |

\* Includes retrieval fees
\*\* First 100GB free

### Recommendation Matrix

| Use Case | Recommended Provider | Storage Class | Why |
|----------|---------------------|---------------|-----|
| Short-term (<30 days) | Azure | Hot | First 100GB egress free |
| Frequent access | Azure | Hot | Lowest combined storage + transfer |
| Quarterly access | Azure | Cool | Good balance, minimal retrieval fees |
| Long-term archive | AWS S3 | Deep Archive | Lowest storage cost ($0.00099/GB) |
| Compliance archive | GCS | Archive | Longest minimum retention |
| Budget-conscious | Azure | Archive | Best price for infrequent access |

## Best Practices

### 1. Estimate Before Export

Always estimate costs before starting an export:

```python
# Get export size first
size = client.estimate_export_size(disk_size_gb=vm_size, format="ova")

# Then estimate costs
cost = client.estimate_cost(
    provider="s3",
    storage_class="s3-glacier",
    storage_gb=size['estimated_export_gb'],
    transfer_gb=size['estimated_export_gb'],
    duration_days=90
)

if cost['total_cost'] > budget:
    print("Warning: Estimated cost exceeds budget!")
```

### 2. Consider Access Patterns

Choose storage class based on how often you'll access the data:

- **Daily access**: Standard/Hot storage
- **Monthly access**: Nearline/Cool storage
- **Quarterly access**: Coldline storage
- **Yearly or less**: Glacier/Archive storage

### 3. Factor in Retrieval Costs

Archive storage has lower storage costs but higher retrieval fees:

```python
# Calculate break-even point
archive_storage = 1000 * 0.00099 * 12  # $11.88/year
archive_retrieval = 1000 * 0.02  # $20 per retrieval

standard_storage = 1000 * 0.023 * 12  # $276/year
standard_retrieval = 0  # No retrieval fees

# Archive is cheaper if you retrieve < 13 times/year
# (276 - 11.88) / 20 = 13.2 retrievals
```

### 4. Use Comparison for Decision Making

Always compare providers before committing:

```python
comparison = client.compare_providers(
    storage_gb=1000,
    transfer_gb=100,
    duration_days=365
)

print(f"Cheapest: {comparison['cheapest']}")
print(f"Savings: ${comparison['savings_vs_expensive']:.2f}")
```

### 5. Monitor Long-term Costs

Project yearly costs for budgeting:

```python
projection = client.project_yearly_cost(
    provider="azure_blob",
    storage_class="azure-cool",
    storage_gb=1000,
    transfer_gb=100,
    requests=1000
)

# Set up budget alerts
monthly_budget = 100
if projection['monthly_average'] > monthly_budget:
    print("Warning: Projected costs exceed monthly budget!")
```

### 6. Account for Snapshots

Include snapshots in size estimates if applicable:

```python
size_with_snapshots = client.estimate_export_size(
    disk_size_gb=500,
    format="ova",
    include_snapshots=True  # Adds ~20% to size
)
```

### 7. Consider Minimum Storage Duration

Avoid early deletion fees by planning retention periods:

```python
# For S3 Glacier Deep Archive (180-day minimum)
if retention_days < 180:
    print("Consider using S3 Glacier (90-day minimum) instead")
    storage_class = "s3-glacier"
else:
    storage_class = "s3-glacier-deep-archive"
```

## Pricing Version

All pricing data is based on **January 2026** rates for:
- AWS S3 (US East 1)
- Azure Blob Storage (East US)
- Google Cloud Storage (US multi-region)

Prices are approximate and may vary based on:
- Actual cloud provider pricing updates
- Regional variations
- Volume discounts
- Enterprise agreements

For the most current pricing, consult the official cloud provider documentation:
- [AWS S3 Pricing](https://aws.amazon.com/s3/pricing/)
- [Azure Blob Pricing](https://azure.microsoft.com/pricing/details/storage/blobs/)
- [Google Cloud Storage Pricing](https://cloud.google.com/storage/pricing)

## Future Enhancements

Planned features for future releases:

- **Regional Pricing**: Support for region-specific pricing
- **Reserved Capacity**: Discount calculations for reserved storage
- **Lifecycle Policies**: Cost optimization with automated tiering
- **Budget Alerts**: Real-time cost monitoring and notifications
- **Cost Analytics**: Historical cost tracking and trend analysis
- **Custom Pricing**: Support for enterprise pricing agreements
- **Additional Providers**: Backblaze B2, Oracle Cloud, Wasabi

## Related Features

- [Multi-Language SDKs](./MULTI_LANGUAGE_SDKS.md) - Python and TypeScript client libraries
- [Advanced Scheduling](./ADVANCED_SCHEDULING.md) - Automated export workflows
- [Incremental Export](./INCREMENTAL_EXPORT.md) - Reduce export sizes with CBT

## Support

For questions or issues with cost estimation:

1. Check the [API Reference](#api-reference) for endpoint details
2. Review [Use Cases](#use-cases) for common scenarios
3. Consult [Best Practices](#best-practices) for optimization tips
4. Open an issue on GitHub for bugs or feature requests

---

**Note**: Cost estimates are approximations based on current pricing data. Actual costs may vary based on your specific usage patterns, cloud provider pricing changes, and regional variations. Always verify estimates against official cloud provider pricing calculators before making decisions.
