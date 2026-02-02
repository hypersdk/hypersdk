#!/usr/bin/env python3
"""
Incremental Backup Example

This script demonstrates how to set up and perform incremental backups using
Changed Block Tracking (CBT) for 95% faster backups.

Usage:
    python incremental_backup.py --vm /datacenter/vm/my-vm --output /backups
"""

import argparse
import sys
import time
from datetime import datetime
from hypersdk import HyperSDK


def setup_and_run_incremental_backup(api_url: str, vm_path: str, output_path: str):
    """Set up CBT and perform incremental backup"""

    client = HyperSDK(api_url)

    print(f"🔧 Setting up incremental backup for {vm_path}\n")

    # Step 1: Check CBT status
    print("📊 Checking CBT status...")
    try:
        cbt_status = client.get_cbt_status(vm_path)

        if cbt_status['cbt_enabled']:
            print(f"✅ CBT is already enabled")
        else:
            print(f"⚠️  CBT is not enabled, enabling now...")
            result = client.enable_cbt(vm_path)
            if result['success']:
                print(f"✅ CBT enabled successfully")
            else:
                print(f"❌ Failed to enable CBT: {result.get('error')}")
                return 1

    except Exception as e:
        print(f"❌ Error checking CBT status: {e}")
        return 1

    # Step 2: Analyze potential savings
    print("\n💰 Analyzing potential savings...")
    try:
        analysis = client.analyze_incremental_export(vm_path)

        if analysis['can_incremental']:
            savings_gb = analysis['estimated_savings_bytes'] / 1e9
            duration = analysis.get('estimated_duration', 'N/A')

            print(f"✅ Incremental export is possible!")
            print(f"   Changed data: {savings_gb:.2f} GB")
            print(f"   Estimated duration: {duration}")

            if analysis.get('last_export'):
                last = analysis['last_export']
                print(f"   Last export: {last.get('timestamp', 'N/A')}")
        else:
            reason = analysis.get('reason', 'Unknown')
            print(f"⚠️  Full export required: {reason}")

    except Exception as e:
        print(f"⚠️  Could not analyze savings: {e}")

    # Step 3: Submit incremental backup job
    print(f"\n🚀 Starting incremental backup...")
    timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')
    output_dir = f"{output_path}/{timestamp}"

    try:
        job_id = client.submit_job({
            "vm_path": vm_path,
            "output_path": output_dir,
            "format": "qcow2",
            "incremental": True,
            "compression": True
        })
        print(f"✅ Backup job submitted: {job_id}\n")

    except Exception as e:
        print(f"❌ Failed to submit backup job: {e}")
        return 1

    # Step 4: Monitor backup progress
    start_time = time.time()
    last_percent = -1

    while True:
        try:
            job = client.get_job(job_id)
            status = job['status']

            if status == 'completed':
                duration = time.time() - start_time
                result = job.get('result', {})

                print(f"\n✅ Incremental backup completed!")
                print(f"   Output: {output_dir}")
                print(f"   Duration: {duration:.1f} seconds")
                print(f"   Size: {result.get('total_size_bytes', 0) / 1e9:.2f} GB")

                # Show savings if available
                if analysis.get('estimated_savings_bytes'):
                    original_size = result.get('total_size_bytes', 0)
                    saved = (1 - original_size / analysis['estimated_savings_bytes']) * 100
                    print(f"   Storage saved: {saved:.1f}%")

                return 0

            elif status == 'failed':
                error = job.get('error', 'Unknown error')
                print(f"\n❌ Backup failed: {error}")
                return 1

            elif status == 'running':
                progress = job.get('progress', {})
                percent = progress.get('percent_complete', 0)

                if percent != last_percent:
                    phase = progress.get('phase', 'backing up')
                    speed = progress.get('speed_mbps', 0)
                    eta = progress.get('eta_seconds', 0)

                    print(f"⏳ {phase}: {percent:.1f}% ({speed:.1f} MB/s, ETA: {eta:.0f}s)")
                    last_percent = percent

        except KeyboardInterrupt:
            print("\n⚠️  Interrupted by user")
            return 130

        except Exception as e:
            print(f"⚠️  Error: {e}")

        time.sleep(3)


def main():
    parser = argparse.ArgumentParser(
        description='Perform incremental backup with CBT'
    )
    parser.add_argument('--api-url', default='http://localhost:8080',
                        help='HyperSDK API URL')
    parser.add_argument('--vm', required=True,
                        help='VM path to backup')
    parser.add_argument('--output', required=True,
                        help='Output directory for backups')

    args = parser.parse_args()

    return setup_and_run_incremental_backup(args.api_url, args.vm, args.output)


if __name__ == '__main__':
    sys.exit(main())
