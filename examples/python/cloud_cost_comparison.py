#!/usr/bin/env python3
"""
Cloud Cost Comparison Example

This script helps you find the most cost-effective cloud storage option
for your VM exports.

Usage:
    python cloud_cost_comparison.py --disk-size 500 --duration 365
"""

import argparse
import sys
from hypersdk import HyperSDK


def compare_cloud_costs(api_url: str, disk_size_gb: float, duration_days: int,
                        format: str = "ova", include_snapshots: bool = False):
    """Compare cloud storage costs across providers"""

    client = HyperSDK(api_url)

    print(f"☁️  Cloud Storage Cost Comparison\n")
    print(f"Parameters:")
    print(f"  Disk Size: {disk_size_gb} GB")
    print(f"  Duration: {duration_days} days")
    print(f"  Format: {format}")
    print(f"  Snapshots: {'Yes' if include_snapshots else 'No'}\n")

    # Step 1: Estimate export size
    print("📦 Estimating export size...")
    try:
        size_estimate = client.estimate_export_size(
            disk_size_gb=disk_size_gb,
            format=format,
            include_snapshots=include_snapshots
        )

        export_size = size_estimate['estimated_export_gb']
        compression = (1 - size_estimate['compression_ratio']) * 100

        print(f"✅ Estimated export size: {export_size:.2f} GB")
        print(f"   Compression ratio: {compression:.1f}%\n")

    except Exception as e:
        print(f"❌ Failed to estimate size: {e}")
        return 1

    # Step 2: Compare providers
    print("💰 Comparing cloud providers...")
    try:
        comparison = client.compare_providers(
            storage_gb=export_size,
            transfer_gb=0,  # No downloads planned
            requests=1000,  # Estimated API requests
            duration_days=duration_days
        )

        print(f"\nCost Comparison ({duration_days} days):\n")
        print(f"{'Provider':<15} {'Storage Class':<20} {'Total Cost':<15} {'Monthly':<15}")
        print("-" * 65)

        for estimate in sorted(comparison['estimates'],
                               key=lambda x: x['total_cost']):
            provider = estimate['provider']
            storage_class = estimate['storage_class']
            total = estimate['total_cost']
            monthly = total / (duration_days / 30)

            marker = "⭐" if provider == comparison['cheapest'] else "  "

            print(f"{marker} {provider:<13} {storage_class:<20} ${total:>12.2f} ${monthly:>12.2f}")

        print("-" * 65)
        print(f"\n🏆 Recommended: {comparison['recommended']}")
        print(f"💵 Savings vs most expensive: ${comparison['savings_vs_expensive']:.2f}")

        # Step 3: Detailed breakdown for cheapest option
        cheapest = next(e for e in comparison['estimates']
                        if e['provider'] == comparison['cheapest'])

        print(f"\n📊 Cost Breakdown for {comparison['cheapest']}:\n")
        breakdown = cheapest['breakdown']
        print(f"  Storage:        ${breakdown['storage_cost']:>10.2f}")
        print(f"  Transfer:       ${breakdown['transfer_cost']:>10.2f}")
        print(f"  Requests:       ${breakdown['request_cost']:>10.2f}")
        print(f"  Retrieval:      ${breakdown['retrieval_cost']:>10.2f}")
        print(f"  Early Delete:   ${breakdown['early_delete_cost']:>10.2f}")
        print(f"  {'-' * 27}")
        print(f"  Total:          ${cheapest['total_cost']:>10.2f}\n")

        # Step 4: Yearly projection for recommended provider
        if duration_days < 365:
            print(f"📅 Yearly Cost Projection for {comparison['recommended']}...")
            try:
                projection = client.project_yearly_cost(
                    provider=comparison['recommended'],
                    storage_class=cheapest['storage_class'],
                    storage_gb=export_size,
                    transfer_gb=0,
                    requests=1000
                )

                print(f"\nYearly Cost: ${projection['total_cost']:.2f}")
                print(f"Monthly Average: ${projection['monthly_average']:.2f}\n")

                # Show quarterly breakdown
                print(f"Quarterly Breakdown:")
                quarters = {
                    'Q1': sum(m['total_cost'] for m in projection['monthly_breakdown'][0:3]),
                    'Q2': sum(m['total_cost'] for m in projection['monthly_breakdown'][3:6]),
                    'Q3': sum(m['total_cost'] for m in projection['monthly_breakdown'][6:9]),
                    'Q4': sum(m['total_cost'] for m in projection['monthly_breakdown'][9:12]),
                }

                for quarter, cost in quarters.items():
                    print(f"  {quarter}: ${cost:>10.2f}")

            except Exception as e:
                print(f"⚠️  Could not project yearly costs: {e}")

        # Step 5: Recommendations
        print(f"\n💡 Recommendations:\n")

        if cheapest['provider'] == 's3':
            if 'glacier' in cheapest['storage_class'].lower():
                print(f"  ✓ S3 Glacier is excellent for long-term archival")
                print(f"  ✓ Consider Glacier Deep Archive for even lower costs")
                print(f"  ⚠  Note: Retrieval takes 3-5 hours (Glacier) or 12 hours (Deep Archive)")
            else:
                print(f"  ✓ S3 Standard provides fast access")
                print(f"  ⚠  For infrequent access, consider S3 IA or Glacier")

        elif cheapest['provider'] == 'azure_blob':
            if 'archive' in cheapest['storage_class'].lower():
                print(f"  ✓ Azure Archive provides lowest cost")
                print(f"  ✓ First 100GB egress per month is free")
                print(f"  ⚠  Note: Retrieval takes up to 15 hours")
            else:
                print(f"  ✓ Azure Hot tier provides fast access")

        elif cheapest['provider'] == 'gcs':
            print(f"  ✓ Google Cloud Storage provides consistent pricing")
            print(f"  ⚠  Transfer costs are higher than S3/Azure")

        print(f"\n  📖 For more details, see: docs/features/COST_ESTIMATION.md")

        return 0

    except Exception as e:
        print(f"❌ Failed to compare providers: {e}")
        return 1


def main():
    parser = argparse.ArgumentParser(
        description='Compare cloud storage costs for VM exports'
    )
    parser.add_argument('--api-url', default='http://localhost:8080',
                        help='HyperSDK API URL')
    parser.add_argument('--disk-size', type=float, required=True,
                        help='VM disk size in GB')
    parser.add_argument('--duration', type=int, default=365,
                        help='Storage duration in days (default: 365)')
    parser.add_argument('--format', default='ova',
                        choices=['ova', 'qcow2', 'raw'],
                        help='Export format (default: ova)')
    parser.add_argument('--snapshots', action='store_true',
                        help='Include snapshots in export')

    args = parser.parse_args()

    return compare_cloud_costs(
        args.api_url,
        args.disk_size,
        args.duration,
        args.format,
        args.snapshots
    )


if __name__ == '__main__':
    sys.exit(main())
