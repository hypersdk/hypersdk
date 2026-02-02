#!/usr/bin/env python3
"""
Simple VM Export Example

This script demonstrates how to export a single VM using the HyperSDK Python client.

Usage:
    python simple_export.py --vm /datacenter/vm/my-vm --output /exports
"""

import argparse
import time
import sys
from hypersdk import HyperSDK


def export_vm(api_url: str, vm_path: str, output_path: str, format: str = "ova"):
    """Export a VM and monitor progress"""

    # Initialize client
    client = HyperSDK(api_url)

    print(f"🚀 Starting export of {vm_path}")
    print(f"   Output: {output_path}")
    print(f"   Format: {format}")

    # Submit export job
    try:
        job_id = client.submit_job({
            "vm_path": vm_path,
            "output_path": output_path,
            "format": format,
            "compression": True,
            "verify": True
        })
        print(f"✅ Job submitted: {job_id}\n")
    except Exception as e:
        print(f"❌ Failed to submit job: {e}")
        return 1

    # Monitor progress
    last_phase = None
    while True:
        try:
            job = client.get_job(job_id)
            status = job['status']

            if status == 'completed':
                result = job.get('result', {})
                print(f"\n✅ Export completed!")
                print(f"   OVF Path: {result.get('ovf_path', 'N/A')}")
                print(f"   Total Time: {job.get('duration', 'N/A')}")
                return 0

            elif status == 'failed':
                error = job.get('error', 'Unknown error')
                print(f"\n❌ Export failed: {error}")
                return 1

            elif status == 'running':
                progress = job.get('progress', {})
                phase = progress.get('phase', 'unknown')
                percent = progress.get('percent_complete', 0)
                speed = progress.get('speed_mbps', 0)

                # Only print if phase changed or every 5% progress
                if phase != last_phase or percent % 5 == 0:
                    print(f"⏳ {phase}: {percent:.1f}% complete ({speed:.1f} MB/s)")
                    last_phase = phase

        except KeyboardInterrupt:
            print("\n⚠️  Interrupted by user")
            print(f"   Job {job_id} is still running on the server")
            return 130

        except Exception as e:
            print(f"⚠️  Error checking status: {e}")

        time.sleep(5)


def main():
    parser = argparse.ArgumentParser(description='Export a VM using HyperSDK')
    parser.add_argument('--api-url', default='http://localhost:8080',
                        help='HyperSDK API URL (default: http://localhost:8080)')
    parser.add_argument('--vm', required=True,
                        help='VM path (e.g., /datacenter/vm/my-vm)')
    parser.add_argument('--output', required=True,
                        help='Output directory path')
    parser.add_argument('--format', default='ova', choices=['ova', 'ovf', 'vmdk'],
                        help='Export format (default: ova)')

    args = parser.parse_args()

    return export_vm(args.api_url, args.vm, args.output, args.format)


if __name__ == '__main__':
    sys.exit(main())
